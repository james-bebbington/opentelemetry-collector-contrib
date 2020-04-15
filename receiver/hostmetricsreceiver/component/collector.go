// Copyright  OpenTelemetry Authors
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

package component

import (
	"context"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"go.uber.org/zap"
)

// Collector gathers metrics from the host machine and converts
// these into internal metrics format.
type Collector interface {
	// Initialize performs any timely initialization tasks such as
	// setting up performance counters for initial collection.
	Initialize() error
	// Close cleans up any unmanaged resources such as performance
	// counter handles.
	Close() error
	// CollectMetrics returns a list of collected metrics.
	CollectMetrics(ctx context.Context) ([]*metricspb.Metric, error)
}

// CollectorFactory can create a Collector.
type CollectorFactory interface {
	// CreateMetricsCollector creates a collector based on this config.
	// If the config is not valid, error will be returned instead.
	CreateMetricsCollector(logger *zap.Logger,
		cfg CollectorConfig) (Collector, error)
}

// CollectorConfig is the configuration of a collector.
type CollectorConfig interface {
}
