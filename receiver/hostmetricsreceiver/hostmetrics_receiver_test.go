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

package hostmetricsreceiver

import (
	"context"
	"runtime"
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/collector/cpu"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/component"
)

var getSingleTickFn = func() <-chan time.Time {
	c := make(chan time.Time)
	go func() { c <- time.Now() }()
	return c
}

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporterOld{}

	config := &Config{
		ScrapeInterval: 0,
		Collectors: map[string]component.CollectorConfig{
			cpu.TypeStr: &cpu.Config{
				ReportPerCPU:     true,
				ReportPerProcess: true,
			},
		},
	}

	factories := map[string]component.CollectorFactory{
		cpu.TypeStr: &cpu.Factory{},
	}

	receiver, err := NewHostMetricsReceiver(zap.NewNop(), config, factories, sink, getSingleTickFn)

	if runtime.GOOS != "windows" {
		require.Error(t, err, "Expected error when creating a metrics receiver with cpu collector on a non-windows environment")
		return
	}

	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	// TODO consider adding mechanism to get notifed when metrics are processed so we
	// dont have to wait here
	<-time.After(10 * time.Millisecond)

	got := sink.AllMetrics()

	if runtime.GOOS != "windows" {
		assert.EqualValues(t, 1, len(got))
		assert.EqualValues(t, 0, len(got[0].Metrics))
		return
	}

	// expect 1 MetricsData object
	assert.Equal(t, 1, len(got))

	gotMetrics := got[0].Metrics

	// expect 3 metrics
	assert.Equal(t, 3, len(gotMetrics))

	// for cpu seconds metric, expect 5*#cores timeseries with appropriate labels
	assert.Equal(t, cpu.MetricCPUSeconds, gotMetrics[0].MetricDescriptor)
	assert.Equal(t, 5*int(runtime.NumCPU()), len(gotMetrics[0].Timeseries))
	assert.Equal(t, cpu.LabelValueCPUUser, gotMetrics[0].Timeseries[0].LabelValues[0])
	assert.Equal(t, cpu.LabelValueCPUSystem, gotMetrics[0].Timeseries[1].LabelValues[0])
	assert.Equal(t, cpu.LabelValueCPUIdle, gotMetrics[0].Timeseries[2].LabelValues[0])
	assert.Equal(t, cpu.LabelValueCPUInterrupt, gotMetrics[0].Timeseries[3].LabelValues[0])
	assert.Equal(t, cpu.LabelValueCPUIOWait, gotMetrics[0].Timeseries[4].LabelValues[0])

	// for cpu utilization metric, expect #cores timeseries each with a value < 110
	// (values can go over 100% by a small margin)
	assert.Equal(t, cpu.MetricCPUUtilization, gotMetrics[1].MetricDescriptor)
	assert.Equal(t, int(runtime.NumCPU()), len(gotMetrics[1].Timeseries))
	for _, ts := range gotMetrics[1].Timeseries {
		assert.Equal(t, 1, len(ts.Points))
		assert.LessOrEqual(t, ts.Points[0].Value.(*metricspb.Point_DoubleValue).DoubleValue, float64(110))
	}

	// for cpu utilization per process metric, expect >1 timeseries
	assert.Equal(t, cpu.MetricProcessCPUUtilization, gotMetrics[2].MetricDescriptor)
	assert.GreaterOrEqual(t, len(gotMetrics[2].Timeseries), 1)
}
