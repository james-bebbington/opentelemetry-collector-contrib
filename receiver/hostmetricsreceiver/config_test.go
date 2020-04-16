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
	"path"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/collector/cpu"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/component"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	collectorFactories, err := component.MakeCollectorFactoryMap(
		&cpu.Factory{},
	)
	require.NoError(t, err)

	factory := &Factory{CollectorFactories: collectorFactories}
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 2)

	r0 := cfg.Receivers["hostmetrics"]
	defaultConfigAllCollectors := factory.CreateDefaultConfig()
	defaultConfigAllCollectors.(*Config).Collectors = map[string]component.CollectorConfig{
		cpu.TypeStr: collectorFactories[cpu.TypeStr].CreateDefaultConfig(),
	}
	assert.Equal(t, r0, defaultConfigAllCollectors)

	r1 := cfg.Receivers["hostmetrics/customname"].(*Config)
	assert.Equal(t, r1,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "hostmetrics/customname",
			},
			ScrapeInterval: 10 * time.Second,
			Collectors: map[string]component.CollectorConfig{
				cpu.TypeStr: &cpu.Config{
					ReportPerCPU:     true,
					ReportPerProcess: true,
				},
			},
		})
}
