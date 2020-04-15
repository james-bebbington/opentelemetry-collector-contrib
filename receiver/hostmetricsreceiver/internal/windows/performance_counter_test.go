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
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	perf "github.com/influxdata/telegraf/plugins/inputs/win_perf_counters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPerfCounter_InvalidPath(t *testing.T) {
	descriptor := &PerfCounterDescriptor{
		Name: "test",
		Path: "Invalid Counter Path",
	}

	_, err := NewPerfCounter(descriptor)
	if assert.Error(t, err) {
		assert.Regexp(t, "^Unable to parse the counter path", err.Error())
	}
}

func TestNewPerfCounter_Valid(t *testing.T) {
	descriptor := &PerfCounterDescriptor{
		Name: "test",
		Path: `\Memory\Committed Bytes`,
	}

	pc, err := NewPerfCounter(descriptor)
	require.NoError(t, err, "Failed to create performance counter: %v", err)

	assert.Equal(t, descriptor, pc.descriptor)
	assert.NotNil(t, pc.query)
	assert.NotNil(t, pc.handle)

	var vals []perf.CounterValue
	vals, err = pc.query.GetFormattedCounterArrayDouble(pc.handle)
	require.NoError(t, err)
	assert.Equal(t, []perf.CounterValue{{InstanceName: "", Value: 0}}, vals)

	err = pc.query.Close()
	require.NoError(t, err, "Failed to close initialized performance counter query: %v", err)
}

func TestNewPerfCounter_CollectOnStartup(t *testing.T) {
	descriptor := &PerfCounterDescriptor{
		Name:             "test",
		Path:             `\Memory\Committed Bytes`,
		CollectOnStartup: true,
	}

	pc, err := NewPerfCounter(descriptor)
	require.NoError(t, err, "Failed to create performance counter: %v", err)

	assert.Equal(t, descriptor, pc.descriptor)
	assert.NotNil(t, pc.query)
	assert.NotNil(t, pc.handle)

	var vals []perf.CounterValue
	vals, err = pc.query.GetFormattedCounterArrayDouble(pc.handle)
	require.NoError(t, err)
	assert.Greater(t, vals[0].Value, float64(0))

	err = pc.query.Close()
	require.NoError(t, err, "Failed to close initialized performance counter query: %v", err)
}

func TestPerfCounter_Close(t *testing.T) {
	pc := getTestPerfCounter(t, "")
	err := pc.Close()
	require.NoError(t, err, "Failed to close initialized performance counter query: %v", err)

	err = pc.Close()
	if assert.Error(t, err) {
		assert.Equal(t, "uninitialised query", err.Error())
	}
}

func TestPerfCounter_CollectDataAndConvertToMetric_NoLabels(t *testing.T) {
	pc := getTestPerfCounter(t, "")

	metricDescriptor := &metricspb.MetricDescriptor{Name: "test", Unit: "1", Type: metricspb.MetricDescriptor_GAUGE_DOUBLE}
	metric, err := pc.CollectDataAndConvertToMetric(time.Now(), metricDescriptor)
	require.NoError(t, err, "Failed to collect data: %v", err)

	assert.Equal(t, metricDescriptor, metric.MetricDescriptor)
	assert.Equal(t, 1, len(metric.Timeseries))
	assert.Equal(t, 0, len(metric.Timeseries[0].LabelValues))
	assert.Equal(t, 1, len(metric.Timeseries[0].Points))
	assert.Greater(t, metric.Timeseries[0].Points[0].Value.(*metricspb.Point_DoubleValue).DoubleValue, float64(0))
}

func TestPerfCounter_CollectDataAndConvertToMetric_Labels(t *testing.T) {
	pc := getTestPerfCounter(t, `\Processor Information(*)\Processor Frequency`)

	metricDescriptor := &metricspb.MetricDescriptor{
		Name:      "test",
		Unit:      "1",
		Type:      metricspb.MetricDescriptor_GAUGE_DOUBLE,
		LabelKeys: []*metricspb.LabelKey{{Key: "cpu"}},
	}
	metric, err := pc.CollectDataAndConvertToMetric(time.Now(), metricDescriptor)
	require.NoError(t, err, "Failed to collect data: %v", err)

	assert.Equal(t, metricDescriptor, metric.MetricDescriptor)
	assert.GreaterOrEqual(t, len(metric.Timeseries), 1)
	assert.Equal(t, 1, len(metric.Timeseries[0].LabelValues))
	assert.NotEmpty(t, metric.Timeseries[0].LabelValues[0].Value)
	assert.Equal(t, 1, len(metric.Timeseries[0].Points))
	assert.Greater(t, metric.Timeseries[0].Points[0].Value.(*metricspb.Point_DoubleValue).DoubleValue, float64(0))
}

func getTestPerfCounter(t *testing.T, path string) *PerfCounter {
	if path == "" {
		path = `\Memory\Committed Bytes`
	}

	pc, err := NewPerfCounter(&PerfCounterDescriptor{Name: "test", Path: path})
	require.NoError(t, err)
	return pc
}
