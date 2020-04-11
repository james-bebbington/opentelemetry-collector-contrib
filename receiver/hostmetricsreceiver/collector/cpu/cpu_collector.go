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
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
)

// Collector for CPU
type Collector struct {
	startTime time.Time

	config *Config
}

// NewCPUCollector creates a set of CPU related metrics
func NewCPUCollector(cfg *Config) *Collector {
	return &Collector{
		startTime: time.Now(),
		config:    cfg,
	}
}

// CollectMetrics related to CPU
func (c *Collector) CollectMetrics() []*metricspb.Metric {
	var metrics []*metricspb.Metric

	metrics = append(metrics, c.getPerCoreMetrics(c.config.ReportPerCPU)...)
	if c.config.ReportPerProcess {
		metrics = append(metrics, c.getPerProcessMetrics()...)
	}

	return metrics
}
