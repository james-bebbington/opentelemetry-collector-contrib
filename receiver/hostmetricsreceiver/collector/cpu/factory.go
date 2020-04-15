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
	"fmt"
	"runtime"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/component"
)

// This file implements Factory for CPU collector.

// Factory is the Factory for collector.
type Factory struct {
}

// CreateMetricsCollector creates a collector based on provided config.
func (f *Factory) CreateMetricsCollector(
	logger *zap.Logger,
	config component.CollectorConfig,
) (component.Collector, error) {

	cfg := config.(*Config)

	switch os := runtime.GOOS; os {
	case "windows":
		return NewWindowsCPUCollector(cfg), nil
	default:
		return nil, fmt.Errorf("cpu metrics collector is not currently supported on %s", os)
	}
}
