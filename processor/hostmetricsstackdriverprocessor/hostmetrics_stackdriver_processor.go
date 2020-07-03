// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hostmetricsstackdriverprocessor

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/translator/conventions"
)

type hostMetricsStackdriverProcessor struct {
	next consumer.MetricsConsumer
}

func newHostMetricsStackdriverProcessor(next consumer.MetricsConsumer) *hostMetricsStackdriverProcessor {
	return &hostMetricsStackdriverProcessor{next: next}
}

// GetCapabilities returns the Capabilities associated with the metrics transform processor.
func (mtp *hostMetricsStackdriverProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (*hostMetricsStackdriverProcessor) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*hostMetricsStackdriverProcessor) Shutdown(ctx context.Context) error {
	return nil
}

// ConsumeMetrics implements the MetricsProcessor interface.
func (mtp *hostMetricsStackdriverProcessor) ConsumeMetrics(ctx context.Context, metrics pdata.Metrics) error {
	md := pdatautil.MetricsToInternalMetrics(metrics)
	combineProcessMetrics(md.ResourceMetrics())
	splitReadWriteBytesMetrics(md.ResourceMetrics())
	metrics = pdatautil.MetricsFromInternalMetrics(md)
	return mtp.next.ConsumeMetrics(ctx, metrics)
}

// The following code performs a translation from metrics that store process
// information as resources to metrics that store process information as labels.
//
// Starting format:
//
//    ResourceMetrics         ResourceMetrics               ResourceMetrics
// +-------------------+ +-----------------------+     +-----------------------+
// |  Resource: Empty  | |  Resource: Process 1  |     |  Resource: Process X  |
// +-------+---+-------+ +---------+---+---------+ ... +---------+---+---------+
// |Metric1|...|MetricN| |MetricN+1|...|MetricN+M|     |MetricN+1|...|MetricN+M|
// +-------+---+-------+ +---------+---+---------+     +---------+---+---------+
//
// Converted format:
//
//                             ResourceMetrics
// +---------------------------------------------------------------------+
// |                           Resource: Empty                           |
// +-------+---+-------+----------------------+---+----------------------+
// |Metric1|...|MetricN|MetricN+1{P1, ..., PX}|...|MetricN+M{P1, ..., PX}|
// +-------+---+-------+----------------------+---+----------------------+
//
// Assumptions:
// * There is at most one resource metrics slice without process resource info (will raise error if not)
// * There is no other resource info supplied that needs to be retained (info may be silently lost if it exists)
// * All metrics with the same name have the same descriptor (process metrics will be merged if the names match)

func combineProcessMetrics(rms pdata.ResourceMetricsSlice) error {
	// combine all process metrics, disregarding any ResourceMetrics with no
	// associated process resource attributes
	processMetrics, otherMetrics, err := createProcessMetrics(rms)
	if err != nil {
		return err
	}

	// if no non-process metrics were supplied, initialize an empty
	// ResourceMetrics object
	if otherMetrics.IsNil() {
		otherMetrics.InitEmpty()
		ilms := otherMetrics.InstrumentationLibraryMetrics()
		ilms.Resize(1)
		ilms.At(0).InitEmpty()
	}

	// append the process metrics to the other ResourceMetrics object
	metrics := otherMetrics.InstrumentationLibraryMetrics().At(0).Metrics()
	// ideally, we would Resize & Set, but not available at this time
	for _, metric := range processMetrics {
		metrics.Append(&metric.Metric) // also, why does append take a pointer?
	}

	// TODO: This is super inefficient. Instead, we should just return a new
	// data.MetricData struct, but currently blocked as it is internal
	rms.Resize(1)
	otherMetrics.CopyTo(rms.At(0))
	return nil
}

func createProcessMetrics(rms pdata.ResourceMetricsSlice) (processMetrics convertedMetrics, otherMetrics pdata.ResourceMetrics, err error) {
	processMetrics = convertedMetrics{}
	otherMetrics = pdata.NewResourceMetrics()

	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		resource := rm.Resource()

		// if these ResourceMetrics do not contain process resource attributes,
		// these must be the "other" non-process metrics
		if !includesProcessAttributes(resource) {
			if !otherMetrics.IsNil() {
				err = errors.New("unexpectedly received multiple Resource Metrics  with err")
				return
			}

			otherMetrics = rm
			continue
		}

		// combine all metrics into the process metrics map by appending
		// the data points
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			metrics := ilms.At(j).Metrics()
			for k := 0; k < metrics.Len(); k++ {
				err = processMetrics.append(metrics.At(k), resource)
				if err != nil {
					return
				}
			}
		}
	}

	return
}

const processAttributePrefix = "process."

// includesProcessAttributes returns true if the resource includes
// any attributes with a "process." prefix
func includesProcessAttributes(resource pdata.Resource) bool {
	if resource.IsNil() {
		return false
	}

	includesProcessAttributes := false
	resource.Attributes().ForEach(func(k string, _ pdata.AttributeValue) {
		if strings.HasPrefix(k, processAttributePrefix) {
			includesProcessAttributes = true
		}
	})
	return includesProcessAttributes
}

// convertedMetrics stores a map of metric names to converted metrics
// where convertedMetrics have process information stored as labels.
type convertedMetrics map[string]*convertedMetric

// append appends the data points associated with the provided metric to the
// associated converted metric (creating this metric if it doesn't exist yet),
// and appends the provided resource attributes as labels against these
// data points.
func (cms convertedMetrics) append(metric pdata.Metric, resource pdata.Resource) error {
	cm := cms.getOrCreate(metric)
	return cm.append(metric, resource)
}

// getOrCreate returns the converted metric associated with a given metric
// name (creating this metric if it doesn't exist yet).
func (cms convertedMetrics) getOrCreate(metric pdata.Metric) *convertedMetric {
	// if we have an existing converted metric, return this
	metricName := metric.MetricDescriptor().Name()
	if cm, ok := cms[metricName]; ok {
		return cm
	}

	// if there is no existing converted metric, create one using the
	// descriptor from the provided metric
	cm := newConvertedMetric(metric.MetricDescriptor())
	cms[metricName] = cm
	return cm
}

// convertedMetric is a pdata.Metric with process information stored as labels.
type convertedMetric struct {
	pdata.Metric
}

// newConvertedMetric creates a new convertedMetric with no data points
// using the provided descriptor
func newConvertedMetric(descriptor pdata.MetricDescriptor) *convertedMetric {
	cm := &convertedMetric{pdata.NewMetric()}
	cm.InitEmpty()
	descriptor.CopyTo(cm.MetricDescriptor())
	return cm
}

// append appends the data points associated with the provided metric to the
// converted metric and appends the provided resource attributes as labels
// against these data points.
func (cm convertedMetric) append(metric pdata.Metric, resource pdata.Resource) error {
	// int64 data points
	idps := metric.Int64DataPoints()
	for i := 0; i < idps.Len(); i++ {
		err := appendAttributesToLabels(idps.At(i).LabelsMap(), resource.Attributes())
		if err != nil {
			return err
		}
	}
	idps.MoveAndAppendTo(cm.Int64DataPoints())

	// double data points
	ddps := metric.DoubleDataPoints()
	for i := 0; i < ddps.Len(); i++ {
		err := appendAttributesToLabels(ddps.At(i).LabelsMap(), resource.Attributes())
		if err != nil {
			return err
		}
	}
	ddps.MoveAndAppendTo(cm.DoubleDataPoints())

	return nil
}

// appendAttributesToLabels appends the provided attributes to the provided labels map.
// This requires converting the attributes to string format.
func appendAttributesToLabels(labels pdata.StringMap, attributes pdata.AttributeMap) error {
	var err error
	attributes.ForEach(func(k string, v pdata.AttributeValue) {
		// break if error has occurred in previous iteration
		if err != nil {
			return
		}

		key := toCloudMonitoringLabel(k)
		// ignore attributes that do not map to a cloud ops label
		if key == "" {
			return
		}

		var value string
		value, err = stringValue(v)
		// break if error
		if err != nil {
			return
		}

		labels.Insert(key, value)
	})
	return err
}

func toCloudMonitoringLabel(resourceAttributeKey string) string {
	// see https://cloud.google.com/monitoring/api/metrics_agent#agent-processes
	switch resourceAttributeKey {
	case conventions.AttributeProcessID:
		return "pid"
	case conventions.AttributeProcessExecutablePath:
		return "process"
	case conventions.AttributeProcessCommand:
		return "command"
	case conventions.AttributeProcessCommandLine:
		return "command_line"
	case conventions.AttributeProcessUsername:
		return "owner"
	default:
		return ""
	}
}

func stringValue(attributeValue pdata.AttributeValue) (string, error) {
	switch t := attributeValue.Type(); t {
	case pdata.AttributeValueBOOL:
		return strconv.FormatBool(attributeValue.BoolVal()), nil
	case pdata.AttributeValueINT:
		return strconv.FormatInt(attributeValue.IntVal(), 10), nil
	case pdata.AttributeValueDOUBLE:
		return strconv.FormatFloat(attributeValue.DoubleVal(), 'f', -1, 64), nil
	case pdata.AttributeValueSTRING:
		return attributeValue.StringVal(), nil
	default:
		return "", fmt.Errorf("unexpected attribute type: %v", t)
	}
}

// The following code splits metrics with read/write direction labels into
// two separate metrics.
//
// Starting format:
//
// +-----------------------------------------------------+
// |                     metric                          |
// +---------+---+---------+------------+---+------------+
// |dp1{read}|...|dpN{read}|dpN+1{write}|...|dpN+N{write}|
// +---------+---+---------+------------+---+------------+
//
// Converted format:
//
// +-----------+ +---------------+
// |read_metric| | write_metric  |
// +---+---+---+ +-----+---+-----+
// |dp1|...|dpN| |dpN+1|...|dpN+N|
// +---+---+---+ +-----+---+-----+

const hostDiskBytes = "host/disk/bytes"
const processDiskBytes = "process/disk/bytes"

func splitReadWriteBytesMetrics(rms pdata.ResourceMetricsSlice) error {
	for i := 0; i < rms.Len(); i++ {
		ilms := rms.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			metrics := ilms.At(j).Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)

				// ignore all metrics except "disk/bytes" metrics
				metricName := metric.MetricDescriptor().Name()
				if metricName != hostDiskBytes && metricName != processDiskBytes {
					continue
				}

				// split into read and write metrics
				read, write, err := splitReadWriteBytesMetric(metric)
				if err != nil {
					return err
				}

				// append the new metrics to the collection, overwriting the old one
				read.CopyTo(metric)
				metrics.Append(&write)
			}
		}
	}

	return nil
}

const (
	directionLabel = "direction"
	readDirection  = "read"
	writeDirection = "write"
)

// matches the string after the last "/" (or the whole string if no "/")
var re = regexp.MustCompile(`([^\/]*$)`)

func splitReadWriteBytesMetric(metric pdata.Metric) (read pdata.Metric, write pdata.Metric, err error) {
	// create new read metric with descriptor & name including "read_" prefix
	read = pdata.NewMetric()
	read.InitEmpty()
	metric.MetricDescriptor().CopyTo(read.MetricDescriptor())
	read.MetricDescriptor().SetName(re.ReplaceAllString(metric.MetricDescriptor().Name(), "read_$1"))

	// create new write metric with descriptor & name including "read_" prefix
	write = pdata.NewMetric()
	write.InitEmpty()
	metric.MetricDescriptor().CopyTo(write.MetricDescriptor())
	write.MetricDescriptor().SetName(re.ReplaceAllString(metric.MetricDescriptor().Name(), "write_$1"))

	// append int64 data points to the read or write metric as appropriate
	if err = appendInt64DataPoints(metric, read, write); err != nil {
		return read, write, err
	}

	// append int64 data points to the read or write metric as appropriate
	if err = appendDoubleDataPoints(metric, read, write); err != nil {
		return read, write, err
	}

	return read, write, err
}

func appendInt64DataPoints(metric, read, write pdata.Metric) error {
	idps := metric.Int64DataPoints()
	for i := 0; i < idps.Len(); i++ {
		idp := idps.At(i)
		labels := idp.LabelsMap()

		dir, ok := labels.Get(directionLabel)
		if !ok {
			return fmt.Errorf("metric %v did not contain %v label as expected", metric.MetricDescriptor().Name(), directionLabel)
		}
		labels.Delete(directionLabel)

		switch d := dir.Value(); d {
		case readDirection:
			read.Int64DataPoints().Append(&idp)
		case writeDirection:
			write.Int64DataPoints().Append(&idp)
		default:
			return fmt.Errorf("metric %v label %v contained unexpected value %v", metric.MetricDescriptor().Name(), directionLabel, d)
		}
	}

	return nil
}

func appendDoubleDataPoints(metric, read, write pdata.Metric) error {
	idps := metric.DoubleDataPoints()
	for i := 0; i < idps.Len(); i++ {
		idp := idps.At(i)
		labels := idp.LabelsMap()

		dir, ok := labels.Get(directionLabel)
		if !ok {
			return fmt.Errorf("metric %v did not contain %v label as expected", metric.MetricDescriptor().Name(), directionLabel)
		}
		labels.Delete(directionLabel)

		switch d := dir.Value(); d {
		case readDirection:
			read.DoubleDataPoints().Append(&idp)
		case writeDirection:
			write.DoubleDataPoints().Append(&idp)
		default:
			return fmt.Errorf("metric %v label %v contained unexpected value %v", metric.MetricDescriptor().Name(), directionLabel, d)
		}
	}

	return nil
}
