// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package windowsperfcountersreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

// createMetricsReceiver creates a metrics receiver based on provided config.
func createMetricsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {
	oCfg := cfg.(*Config)
	scraper, err := newScraper(oCfg)
	if err != nil {
		return nil, err
	}

	return receiverhelper.NewScraperControllerReceiver(
		&oCfg.ScraperControllerSettings,
		consumer,
		receiverhelper.AddMetricsScraper(
			receiverhelper.NewMetricsScraper(
				cfg.Name(),
				scraper.scrape,
				receiverhelper.WithInitialize(scraper.initialize),
				receiverhelper.WithClose(scraper.close),
			),
		),
	)
}
