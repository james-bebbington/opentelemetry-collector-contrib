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

package cpu

import (
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
)

// Host and process metric constants.

var MetricCPUSeconds = &metricspb.MetricDescriptor{
	Name:        "host/cpu/time",
	Description: "Total CPU seconds broken down by different states",
	Unit:        "s",
	Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
	LabelKeys: []*metricspb.LabelKey{
		{Key: "state", Description: "State of CPU time, e.g user/system/idle"},
		{Key: "cpu", Description: "CPU Logical Number, e.g. 0/1/2"},
	},
}

var MetricCPUUtilization = &metricspb.MetricDescriptor{
	Name:        "host/cpu/utilization",
	Description: "Total User/System CPU seconds divided by total available CPU seconds",
	Unit:        "1",
	Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
	LabelKeys:   []*metricspb.LabelKey{{Key: "cpu", Description: "CPU Logical Number, e.g. 0/1/2"}},
}

var MetricProcessCPUSeconds = &metricspb.MetricDescriptor{
	Name:        "process/cpu/time",
	Description: "CPU seconds for this process",
	Unit:        "s",
	Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
	LabelKeys:   []*metricspb.LabelKey{{Key: "processname", Description: "Name of the process"}},
}

var (
	LabelValueCPUUser   = &metricspb.LabelValue{Value: "user", HasValue: true}
	LabelValueCPUSystem = &metricspb.LabelValue{Value: "system", HasValue: true}
	LabelValueCPUIdle   = &metricspb.LabelValue{Value: "idle", HasValue: true}
	LabelValueCPUNice   = &metricspb.LabelValue{Value: "nice", HasValue: true}
	LabelValueCPUIOWait = &metricspb.LabelValue{Value: "iowait", HasValue: true}
)
