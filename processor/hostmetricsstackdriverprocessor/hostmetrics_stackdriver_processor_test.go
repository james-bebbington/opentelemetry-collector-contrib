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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver"
	"go.uber.org/zap"
)

type testCase struct {
	name     string
	input    pdata.Metrics
	expected pdata.Metrics
}

func TestHostMetricsStackdriverProcessor(t *testing.T) {
	tests := []testCase{
		{
			name:     "process-resources-case",
			input:    generateProcessResourceMetricsInput(),
			expected: generateProcessResourceMetricsExpected(),
		},
		{
			name:     "read-write-split-case",
			input:    generateReadWriteMetricsInput(),
			expected: generateReadWriteMetricsExpected(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &Factory{}
			tmn := &exportertest.SinkMetricsExporter{}
			rmp, err := factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, tmn, &Config{})
			require.NoError(t, err)

			assert.True(t, rmp.GetCapabilities().MutatesConsumedData)

			require.NoError(t, rmp.Start(context.Background(), componenttest.NewNopHost()))
			defer func() { assert.NoError(t, rmp.Shutdown(context.Background())) }()

			err = rmp.ConsumeMetrics(context.Background(), tt.input)
			require.NoError(t, err)

			assertEqual(t, tt.expected, tmn.AllMetrics()[0])
		})
	}
}

//    ResourceMetrics         ResourceMetrics               ResourceMetrics
// +-------------------+ +-----------------------+     +-----------------------+
// |  Resource: Empty  | |  Resource: Process 1  |     |  Resource: Process X  |
// +-------+---+-------+ +---------+---+---------+ ... +---------+---+---------+
// |Metric1|...|MetricN| |MetricN+1|...|MetricN+M|     |MetricN+1|...|MetricN+M|
// +-------+---+-------+ +---------+---+---------+     +---------+---+---------+
func generateProcessResourceMetricsInput() pdata.Metrics {
	input := pdatautil.MetricsToInternalMetrics(newInternalMetrics())
	input.ResourceMetrics().Resize(3)

	m1 := initializeResourceMetrics(input.ResourceMetrics().At(0), nil)
	m1.Resize(2)
	initializeInt64Metric(m1.At(0), "m1", map[string]string{"label1": "value1"})
	initializeDoubleMetric(m1.At(1), "m2", map[string]string{"label1": "value1"})

	// TODO add other process attributes
	m2 := initializeResourceMetrics(input.ResourceMetrics().At(1), map[string]pdata.AttributeValue{"process.id": pdata.NewAttributeValueInt(1)})
	m2.Resize(2)
	initializeInt64Metric(m2.At(0), "m3", map[string]string{"label1": "value1"})
	initializeDoubleMetric(m2.At(1), "m4", map[string]string{"label1": "value1"})

	m3 := initializeResourceMetrics(input.ResourceMetrics().At(2), map[string]pdata.AttributeValue{"process.id": pdata.NewAttributeValueInt(2)})
	m3.Resize(2)
	initializeInt64Metric(m3.At(0), "m3", map[string]string{"label1": "value2"})
	initializeDoubleMetric(m3.At(1), "m4", map[string]string{"label1": "value2"})

	return pdatautil.MetricsFromInternalMetrics(input)
}

//                             ResourceMetrics
// +---------------------------------------------------------------------+
// |                           Resource: Empty                           |
// +-------+---+-------+----------------------+---+----------------------+
// |Metric1|...|MetricN|MetricN+1{P1, ..., PX}|...|MetricN+M{P1, ..., PX}|
// +-------+---+-------+----------------------+---+----------------------+
func generateProcessResourceMetricsExpected() pdata.Metrics {
	expected := pdatautil.MetricsToInternalMetrics(newInternalMetrics())
	expected.ResourceMetrics().Resize(1)

	m := initializeResourceMetrics(expected.ResourceMetrics().At(0), nil)
	m.Resize(4)
	initializeInt64Metric(m.At(0), "m1", map[string]string{"label1": "value1"})
	initializeDoubleMetric(m.At(1), "m2", map[string]string{"label1": "value1"})
	initializeInt64Metric(m.At(2), "m3", map[string]string{"label1": "value1", "pid": "1"}, map[string]string{"label1": "value2", "pid": "2"})
	initializeDoubleMetric(m.At(3), "m4", map[string]string{"label1": "value1", "pid": "1"}, map[string]string{"label1": "value2", "pid": "2"})

	return pdatautil.MetricsFromInternalMetrics(expected)
}

// +-----------------------------------------------------+
// |                     metric                          |
// +---------+---+---------+------------+---+------------+
// |dp1{read}|...|dpN{read}|dpN+1{write}|...|dpN+N{write}|
// +---------+---+---------+------------+---+------------+
func generateReadWriteMetricsInput() pdata.Metrics {
	input := pdatautil.MetricsToInternalMetrics(newInternalMetrics())
	input.ResourceMetrics().Resize(1)

	m := initializeResourceMetrics(input.ResourceMetrics().At(0), nil)
	m.Resize(2)
	initializeInt64Metric(m.At(0), "host/disk/bytes", map[string]string{"label1": "value1", "direction": "read"}, map[string]string{"label1": "value2", "direction": "write"})
	initializeDoubleMetric(m.At(1), "process/disk/bytes", map[string]string{"label1": "value1", "direction": "read"}, map[string]string{"label1": "value2", "direction": "write"})

	return pdatautil.MetricsFromInternalMetrics(input)
}

// +-----------+ +---------------+
// |read_metric| | write_metric  |
// +---+---+---+ +-----+---+-----+
// |dp1|...|dpN| |dpN+1|...|dpN+N|
// +---+---+---+ +-----+---+-----+
func generateReadWriteMetricsExpected() pdata.Metrics {
	expected := pdatautil.MetricsToInternalMetrics(newInternalMetrics())
	expected.ResourceMetrics().Resize(1)

	m := initializeResourceMetrics(expected.ResourceMetrics().At(0), nil)
	m.Resize(4)
	initializeInt64Metric(m.At(0), "host/disk/read_bytes", map[string]string{"label1": "value1"})
	initializeDoubleMetric(m.At(1), "process/disk/read_bytes", map[string]string{"label1": "value1"})
	initializeInt64Metric(m.At(2), "host/disk/write_bytes", map[string]string{"label1": "value2"})
	initializeDoubleMetric(m.At(3), "process/disk/write_bytes", map[string]string{"label1": "value2"})

	return pdatautil.MetricsFromInternalMetrics(expected)
}

func initializeResourceMetrics(rm pdata.ResourceMetrics, resourceAttributes map[string]pdata.AttributeValue) pdata.MetricSlice {
	if resourceAttributes != nil {
		rm.Resource().InitEmpty()
		rm.Resource().Attributes().InitFromMap(resourceAttributes)
	}

	rm.InstrumentationLibraryMetrics().Resize(1)
	ilm := rm.InstrumentationLibraryMetrics().At(0)
	ilm.InitEmpty()
	return ilm.Metrics()
}

func initializeInt64Metric(metric pdata.Metric, name string, labelsMaps ...map[string]string) {
	metric.MetricDescriptor().InitEmpty()
	metric.MetricDescriptor().SetName(name)
	metric.Int64DataPoints().Resize(len(labelsMaps))
	for i, labels := range labelsMaps {
		metric.Int64DataPoints().At(i).InitEmpty()
		metric.Int64DataPoints().At(i).LabelsMap().InitFromMap(labels)
	}
}

func initializeDoubleMetric(metric pdata.Metric, name string, labelsMaps ...map[string]string) {
	metric.MetricDescriptor().InitEmpty()
	metric.MetricDescriptor().SetName(name)
	metric.DoubleDataPoints().Resize(len(labelsMaps))
	for i, labels := range labelsMaps {
		metric.DoubleDataPoints().At(i).InitEmpty()
		metric.DoubleDataPoints().At(i).LabelsMap().InitFromMap(labels)
	}
}

func assertEqual(t *testing.T, expected, actual pdata.Metrics) {
	rmsAct := pdatautil.MetricsToInternalMetrics(actual).ResourceMetrics()
	rmsExp := pdatautil.MetricsToInternalMetrics(expected).ResourceMetrics()
	require.Equal(t, rmsExp.Len(), rmsAct.Len())
	for i := 0; i < rmsAct.Len(); i++ {
		rmAct := rmsAct.At(i)
		rmExp := rmsExp.At(i)

		// assert equality of resource attributes
		assert.Equal(t, rmExp.Resource().IsNil(), rmAct.Resource().IsNil())
		if !rmExp.Resource().IsNil() {
			assert.Equal(t, rmExp.Resource().Attributes().Sort(), rmAct.Resource().Attributes().Sort())
		}

		// assert equality of IL metrics
		ilmsAct := rmAct.InstrumentationLibraryMetrics()
		ilmsExp := rmExp.InstrumentationLibraryMetrics()
		require.Equal(t, ilmsExp.Len(), ilmsAct.Len())
		for j := 0; j < ilmsAct.Len(); j++ {
			ilmAct := ilmsAct.At(j)
			ilmExp := ilmsExp.At(j)

			// currently expect IL to always be nil
			assert.True(t, ilmAct.InstrumentationLibrary().IsNil())
			assert.True(t, ilmExp.InstrumentationLibrary().IsNil())

			// assert equality of metrics
			metricsAct := ilmAct.Metrics()
			metricsExp := ilmExp.Metrics()
			require.Equal(t, metricsExp.Len(), metricsAct.Len())
			for k := 0; k < metricsAct.Len(); k++ {
				metricAct := metricsAct.At(k)
				metricExp := metricsExp.At(k)

				// assert equality of descriptors
				assert.Equal(t, metricExp.MetricDescriptor(), metricAct.MetricDescriptor())

				// assert equality of int data points
				idpsAct := metricAct.Int64DataPoints()
				idpsExp := metricExp.Int64DataPoints()
				require.Equal(t, idpsExp.Len(), idpsAct.Len())
				for l := 0; l < idpsAct.Len(); l++ {
					idpAct := idpsAct.At(l)
					idpExp := idpsExp.At(l)

					assert.Equal(t, idpExp.LabelsMap().Sort(), idpAct.LabelsMap().Sort())
					assert.Equal(t, idpExp.StartTime(), idpAct.StartTime())
					assert.Equal(t, idpExp.Timestamp(), idpAct.Timestamp())
					assert.Equal(t, idpExp.Value(), idpAct.Value())
				}

				// assert equality of double data points
				ddpsAct := metricAct.DoubleDataPoints()
				ddpsExp := metricExp.DoubleDataPoints()
				require.Equal(t, ddpsExp.Len(), ddpsAct.Len())
				for l := 0; l < ddpsAct.Len(); l++ {
					ddpAct := ddpsAct.At(l)
					ddpExp := ddpsExp.At(l)

					assert.Equal(t, ddpExp.LabelsMap().Sort(), ddpAct.LabelsMap().Sort())
					assert.Equal(t, ddpExp.StartTime(), ddpAct.StartTime())
					assert.Equal(t, ddpExp.Timestamp(), ddpAct.Timestamp())
					assert.Equal(t, ddpExp.Value(), ddpAct.Value())
				}

				// currently expect other kinds of data points to always be empty
				assert.True(t, metricAct.HistogramDataPoints().Len() == 0)
				assert.True(t, metricExp.HistogramDataPoints().Len() == 0)
				assert.True(t, metricAct.SummaryDataPoints().Len() == 0)
				assert.True(t, metricExp.SummaryDataPoints().Len() == 0)
			}
		}
	}
}

var cached *pdata.Metrics

// massive hack to get an internal.Data object since we shouldn't be using it yet,
// but its needed to construct the resource tests
func newInternalMetrics() pdata.Metrics {
	if cached == nil {
		f := &hostmetricsreceiver.Factory{}
		c := &hostmetricsreceiver.Config{CollectionInterval: 3 * time.Millisecond}
		s := &exportertest.SinkMetricsExporter{}
		r, _ := f.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{Logger: zap.NewNop()}, c, s)
		r.Start(context.Background(), componenttest.NewNopHost())
		time.Sleep(10 * time.Millisecond)
		r.Shutdown(context.Background())
		md := s.AllMetrics()[0]
		cached = &md
	}

	return pdatautil.MetricsFromInternalMetrics(pdatautil.MetricsToInternalMetrics(*cached).Clone())
}
