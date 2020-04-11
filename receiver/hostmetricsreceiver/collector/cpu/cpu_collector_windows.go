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
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal"
)

func (c *Collector) getPerCoreMetrics(reportPerCPU bool) []*metricspb.Metric {
	return []*metricspb.Metric{
		{
			MetricDescriptor: MetricCPUSeconds,
			Timeseries: []*metricspb.TimeSeries{
				internal.GetDoubleTimeSeries(c.startTime, 0, LabelValueCPUUser, &metricspb.LabelValue{HasValue: false}),
				internal.GetDoubleTimeSeries(c.startTime, 0, LabelValueCPUSystem, &metricspb.LabelValue{HasValue: false}),
				internal.GetDoubleTimeSeries(c.startTime, 0, LabelValueCPUIdle, &metricspb.LabelValue{HasValue: false}),
				internal.GetDoubleTimeSeries(c.startTime, 0, LabelValueCPUNice, &metricspb.LabelValue{HasValue: false}),
				internal.GetDoubleTimeSeries(c.startTime, 0, LabelValueCPUIOWait, &metricspb.LabelValue{HasValue: false}),
			},
		},
		{
			MetricDescriptor: MetricCPUUtilization,
			Timeseries:       []*metricspb.TimeSeries{internal.GetDoubleTimeSeries(c.startTime, 0, &metricspb.LabelValue{HasValue: false})},
		},
	}
}

func (c *Collector) getPerProcessMetrics() []*metricspb.Metric {
	return []*metricspb.Metric{
		{
			MetricDescriptor: MetricProcessCPUSeconds,
			Timeseries:       []*metricspb.TimeSeries{internal.GetDoubleTimeSeries(c.startTime, 0, &metricspb.LabelValue{HasValue: false}, &metricspb.LabelValue{HasValue: false})},
		},
	}
}
