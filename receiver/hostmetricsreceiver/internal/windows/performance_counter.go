// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package windows

import (
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	perf "github.com/influxdata/telegraf/plugins/inputs/win_perf_counters"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal"
)

type ValueTransformationFn func(val float64) float64

type PerfCounterDescriptor struct {
	Name             string
	Path             string
	CollectOnStartup bool
	TransformFn      ValueTransformationFn
}

type PerfCounter struct {
	descriptor *PerfCounterDescriptor
	query      perf.PerformanceQuery
	handle     perf.PDH_HCOUNTER
}

// NewPerfCounter returns a new performance counter for the specified descriptor.
func NewPerfCounter(descriptor *PerfCounterDescriptor) (*PerfCounter, error) {
	query := &perf.PerformanceQueryImpl{}
	err := query.Open()
	if err != nil {
		return nil, err
	}

	var handle perf.PDH_HCOUNTER
	handle, err = query.AddEnglishCounterToQuery(descriptor.Path)
	if err != nil {
		return nil, err
	}

	// some perf counters (e.g. cpu) return the usage stats since the last measure
	// so we need to collect data on startup to avoid an invalid initial reading
	if descriptor.CollectOnStartup {
		err = query.CollectData()
		if err != nil {
			return nil, err
		}
	}

	counter := &PerfCounter{
		descriptor: descriptor,
		query:      query,
		handle:     handle,
	}

	return counter, nil
}

// Close all counters/handles related to the query and free all associated memory.
func (pc *PerfCounter) Close() error {
	return pc.query.Close()
}

// CollectDataAndConvertToMetric takes a measure of the performance counter and
// converts this value into an Otel metric.
func (c *PerfCounter) CollectDataAndConvertToMetric(
	startTime time.Time,
	descriptor *metricspb.MetricDescriptor,
) (*metricspb.Metric, error) {
	vals, err := c.CollectData()
	if err != nil {
		return nil, err
	}

	metric := c.convertToMetric(vals, startTime, descriptor)

	return metric, nil
}

func (pc *PerfCounter) CollectData() ([]perf.CounterValue, error) {
	var vals []perf.CounterValue

	err := pc.query.CollectData()
	if err != nil {
		return nil, err
	}

	vals, err = pc.query.GetFormattedCounterArrayDouble(pc.handle)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

func (pc *PerfCounter) convertToMetric(vals []perf.CounterValue, startTime time.Time, descriptor *metricspb.MetricDescriptor) *metricspb.Metric {
	var timeseries = []*metricspb.TimeSeries{}
	for _, val := range vals {
		// ignore the total value unless it is the only value measured
		if len(vals) > 1 && val.InstanceName == "_Total" {
			continue
		}

		value := val.Value
		if pc.descriptor.TransformFn != nil {
			value = pc.descriptor.TransformFn(val.Value)
		}

		labels := []*metricspb.LabelValue{}

		if len(vals) > 1 || (val.InstanceName != "" && val.InstanceName != "_Total") {
			labels = append(labels, &metricspb.LabelValue{Value: val.InstanceName, HasValue: true})
		}

		timeseries = append(timeseries, internal.GetDoubleTimeSeries(startTime, value, labels...))
	}

	metric := &metricspb.Metric{
		MetricDescriptor: descriptor,
		Timeseries:       timeseries,
	}

	return metric
}
