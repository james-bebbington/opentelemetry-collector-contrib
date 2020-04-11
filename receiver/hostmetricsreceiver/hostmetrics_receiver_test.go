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

	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/collector/cpu"
)

var logger = zap.NewNop()

var getSingleTickFn = func() <-chan time.Time {
	c := make(chan time.Time)
	go func() { c <- time.Now() }()
	return c
}

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporterOld{}

	config := &Config{
		ScrapeInterval: 0,
		CPUConfig: &cpu.Config{
			ReportPerCPU:     true,
			ReportPerProcess: true,
		},
	}

	receiver, err := NewHostMetricsReceiver(logger, config, sink, getSingleTickFn)
	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	// TODO consider adding mechanism to get notifed when metrics are processed so we
	// dont have to wait here
	<-time.After(10 * time.Millisecond)

	got := sink.AllMetrics()

	if runtime.GOOS == "windows" {
		assert.EqualValues(t, 1, len(got))
		assert.EqualValues(t, 0, len(got[0].Metrics))
		return
	}

	assert.EqualValues(t, 1, len(got))
	gotMetrics := got[0].Metrics
	assert.EqualValues(t, 3, len(gotMetrics))
	assert.EqualValues(t, cpu.MetricCPUSeconds, gotMetrics[0].MetricDescriptor)

	assert.EqualValues(t, 5, len(gotMetrics[0].Timeseries))
	assert.EqualValues(t, cpu.LabelValueCPUUser, gotMetrics[0].Timeseries[0].LabelValues[0])
	assert.EqualValues(t, cpu.LabelValueCPUSystem, gotMetrics[0].Timeseries[1].LabelValues[0])
	assert.EqualValues(t, cpu.LabelValueCPUIdle, gotMetrics[0].Timeseries[2].LabelValues[0])
	assert.EqualValues(t, cpu.LabelValueCPUNice, gotMetrics[0].Timeseries[3].LabelValues[0])
	assert.EqualValues(t, cpu.LabelValueCPUIOWait, gotMetrics[0].Timeseries[4].LabelValues[0])

	assert.EqualValues(t, cpu.MetricCPUUtilization, gotMetrics[1].MetricDescriptor)
	assert.EqualValues(t, 1, len(gotMetrics[1].Timeseries))

	assert.EqualValues(t, cpu.MetricProcessCPUSeconds, gotMetrics[2].MetricDescriptor)
	assert.EqualValues(t, 1, len(gotMetrics[2].Timeseries))
}
