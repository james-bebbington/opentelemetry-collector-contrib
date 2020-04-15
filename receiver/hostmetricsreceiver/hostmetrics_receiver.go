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
	"fmt"
	"reflect"
	"sync"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/collector/cpu"
	hmcomponent "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/component"
)

// Receiver is the type used to handle metrics from VM metrics.
type Receiver struct {
	consumer consumer.MetricsConsumerOld

	tickerFn getTickerC

	config     *Config
	collectors []hmcomponent.Collector

	startOnce sync.Once
	stopOnce  sync.Once

	done chan struct{}
}

type getTickerC func() <-chan time.Time

// NewHostMetricsReceiver creates a new set of VM and Process Metrics
func NewHostMetricsReceiver(
	logger *zap.Logger,
	config *Config,
	consumer consumer.MetricsConsumerOld,
	tickerFn getTickerC,
) (*Receiver, error) {

	configs := []struct {
		config  hmcomponent.CollectorConfig
		factory hmcomponent.CollectorFactory
	}{
		{config: config.CPUConfig, factory: &cpu.Factory{}},
	}

	collectors := make([]hmcomponent.Collector, 0)
	for _, cfg := range configs {
		if reflect.ValueOf(cfg.config).IsNil() {
			continue
		}

		collector, err := cfg.factory.CreateMetricsCollector(logger, cfg.config)
		if err != nil {
			return nil, fmt.Errorf("cannot create collector: %s", err.Error())
		}
		collectors = append(collectors, collector)
	}

	if tickerFn == nil {
		tickerFn = func() <-chan time.Time { return time.NewTicker(config.ScrapeInterval).C }
	}

	hmr := &Receiver{
		consumer:   consumer,
		tickerFn:   tickerFn,
		config:     config,
		collectors: collectors,
		done:       make(chan struct{}),
	}

	return hmr, nil
}

// Start scrapes Host metrics based on the OS platform.
func (hmr *Receiver) Start(ctx context.Context, host component.Host) error {
	var err = componenterror.ErrAlreadyStarted
	hmr.startOnce.Do(func() {
		err = hmr.initializeCollectors()
		if err != nil {
			return
		}

		go func() {
			tickerC := hmr.tickerFn()
			for {
				select {
				case <-tickerC:
					hmr.scrapeAndExport()

				case <-hmr.done:
					return
				}
			}
		}()

		err = nil
	})
	return err
}

// Shutdown stops and cancels the underlying Host metrics scrapers.
func (hmr *Receiver) Shutdown(context.Context) error {
	var err = componenterror.ErrAlreadyStopped
	hmr.stopOnce.Do(func() {
		defer close(hmr.done)
		err = hmr.closeCollectors()
	})
	return err
}

func (hmr *Receiver) initializeCollectors() error {
	for _, collector := range hmr.collectors {
		err := collector.Initialize()
		if err != nil {
			return err
		}
	}

	return nil
}

func (hmr *Receiver) closeCollectors() error {
	var errs []error

	for _, collector := range hmr.collectors {
		err := collector.Close()
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}

	return nil
}

func (hmr *Receiver) scrapeAndExport() {
	ctx, span := trace.StartSpan(context.Background(), "hostmetricsreceiver.scrapeAndExport")
	defer span.End()

	var errs []error
	var metrics []*metricspb.Metric

	for _, collector := range hmr.collectors {
		m, err := collector.CollectMetrics(ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		metrics = append(metrics, m...)
	}

	if len(errs) > 0 {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping host metrics: %v", componenterror.CombineErrors(errs))})
	}

	if len(metrics) > 0 {
		err := hmr.consumer.ConsumeMetricsData(ctx, consumerdata.MetricsData{Metrics: metrics})
		if err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
			return
		}
	}
}
