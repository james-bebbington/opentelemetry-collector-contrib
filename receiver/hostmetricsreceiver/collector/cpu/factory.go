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
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/component"
)

// This file implements Factory for CPU collector.

const (
	// The value of "type" key in configuration.
	TypeStr = "cpu"
)

// Factory is the Factory for collector.
type Factory struct {
}

// Type gets the type of the collector config created by this Factory.
func (f *Factory) Type() string {
	return TypeStr
}

// CreateDefaultConfig creates the default configuration for the Collector.
func (f *Factory) CreateDefaultConfig() component.CollectorConfig {
	return &Config{
		ReportPerCPU: true,
	}
}

// CreateMetricsCollector creates a collector based on provided config.
func (f *Factory) CreateMetricsCollector(
	logger *zap.Logger,
	config component.CollectorConfig,
) (component.Collector, error) {
	cfg := config.(*Config)
	return NewCPUCollector(cfg)
}
