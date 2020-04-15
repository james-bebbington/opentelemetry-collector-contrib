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

package cpu

import (
	"context"
	"runtime"
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrics_Uninitialized(t *testing.T) {
	config := &Config{}

	collector := NewWindowsCPUCollector(config)

	_, err := collector.CollectMetrics(context.Background())
	if assert.Error(t, err) {
		assert.Equal(t, "cpu collector has not been initialized", err.Error())
	}
}

func TestCollectMetrics_MinimalData(t *testing.T) {
	config := &Config{}

	gotMetrics := createCollectorAndCollectMetrics(t, config)

	// expect 2 metrics
	assert.Equal(t, 2, len(gotMetrics))

	// for cpu seconds metric, expect 5 timeseries with appropriate labels
	assert.Equal(t, MetricCPUSeconds, gotMetrics[0].MetricDescriptor)
	assert.Equal(t, 5, len(gotMetrics[0].Timeseries))
	assert.Equal(t, LabelValueCPUUser, gotMetrics[0].Timeseries[0].LabelValues[0])
	assert.Equal(t, LabelValueCPUSystem, gotMetrics[0].Timeseries[1].LabelValues[0])
	assert.Equal(t, LabelValueCPUIdle, gotMetrics[0].Timeseries[2].LabelValues[0])
	assert.Equal(t, LabelValueCPUInterrupt, gotMetrics[0].Timeseries[3].LabelValues[0])
	assert.Equal(t, LabelValueCPUIOWait, gotMetrics[0].Timeseries[4].LabelValues[0])

	// for cpu utilization metric, expect 1 timeseries with a value < 110
	// (value can go over 100% by a small margin)
	assert.Equal(t, MetricCPUUtilization, gotMetrics[1].MetricDescriptor)
	assert.Equal(t, 1, len(gotMetrics[1].Timeseries))
	assert.Equal(t, 1, len(gotMetrics[1].Timeseries[0].Points))
	assert.LessOrEqual(t, gotMetrics[1].Timeseries[0].Points[0].Value.(*metricspb.Point_DoubleValue).DoubleValue, float64(110))
}

func TestCollectMetrics_AllData(t *testing.T) {
	config := &Config{
		ReportPerCPU:     true,
		ReportPerProcess: true,
	}

	gotMetrics := createCollectorAndCollectMetrics(t, config)

	// expect 3 metrics
	assert.Equal(t, 3, len(gotMetrics))

	// for cpu seconds metric, expect 5*#cores timeseries with appropriate labels
	assert.Equal(t, MetricCPUSeconds, gotMetrics[0].MetricDescriptor)
	assert.Equal(t, 5*int(runtime.NumCPU()), len(gotMetrics[0].Timeseries))
	assert.Equal(t, LabelValueCPUUser, gotMetrics[0].Timeseries[0].LabelValues[0])
	assert.Equal(t, LabelValueCPUSystem, gotMetrics[0].Timeseries[1].LabelValues[0])
	assert.Equal(t, LabelValueCPUIdle, gotMetrics[0].Timeseries[2].LabelValues[0])
	assert.Equal(t, LabelValueCPUInterrupt, gotMetrics[0].Timeseries[3].LabelValues[0])
	assert.Equal(t, LabelValueCPUIOWait, gotMetrics[0].Timeseries[4].LabelValues[0])

	// for cpu utilization metric, expect #cores timeseries each with a value < 110
	// (values can go over 100% by a small margin)
	assert.Equal(t, MetricCPUUtilization, gotMetrics[1].MetricDescriptor)
	assert.Equal(t, int(runtime.NumCPU()), len(gotMetrics[1].Timeseries))
	for _, ts := range gotMetrics[1].Timeseries {
		assert.Equal(t, 1, len(ts.Points))
		assert.LessOrEqual(t, ts.Points[0].Value.(*metricspb.Point_DoubleValue).DoubleValue, float64(110))
	}

	// for cpu utilization per process metric, expect >1 timeseries
	assert.Equal(t, MetricProcessCPUUtilization, gotMetrics[2].MetricDescriptor)
	assert.GreaterOrEqual(t, len(gotMetrics[2].Timeseries), 1)
}

func createCollectorAndCollectMetrics(t *testing.T, config *Config) []*metricspb.Metric {
	collector := NewWindowsCPUCollector(config)

	// need to sleep briefly to ensure enough time has passed to generate
	// two valid processor time measurements
	<-time.After(10 * time.Millisecond)

	err := collector.Initialize()
	require.NoError(t, err, "Failed to initialize cpu collector: %v", err)
	defer func() { assert.NoError(t, collector.Close()) }()

	var metrics []*metricspb.Metric
	metrics, err = collector.CollectMetrics(context.Background())
	require.NoError(t, err, "Failed to collect cpu metrics: %v", err)

	return metrics
}
