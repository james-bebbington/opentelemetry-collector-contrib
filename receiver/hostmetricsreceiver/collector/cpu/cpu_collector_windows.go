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
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/process"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal/windows"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
)

var cpus = float64(runtime.NumCPU())

var hostUtilizationDescriptor = &windows.PerfCounterDescriptor{
	Name:             "host/cpu/utilization",
	Path:             `\Processor(*)\% Processor Time`,
	CollectOnStartup: true,
}

var hostUtilizationTotalDescriptor = &windows.PerfCounterDescriptor{
	Name:             "host/cpu/utilization",
	Path:             `\Processor(_Total)\% Processor Time`,
	CollectOnStartup: true,
}

var processUtilizationDescriptor = &windows.PerfCounterDescriptor{
	Name:             "process/cpu/utilization",
	Path:             `\Process(*)\% Processor Time`,
	CollectOnStartup: true,
	// "Process\% Processor time" is reported per cpu so we need to
	// scale the value based on #cpu cores
	TransformFn: func(val float64) float64 { return val / cpus },
}

// Collector for CPU Metrics
type Collector struct {
	startTime time.Time

	hostUtilizationCounter    *windows.PerfCounter
	processUtilizationCounter *windows.PerfCounter

	config *Config

	initializeOnce sync.Once
	closeOnce      sync.Once
}

// NewCPUCollector creates a set of CPU related metrics
func NewWindowsCPUCollector(cfg *Config) *Collector {
	return &Collector{
		config: cfg,
	}
}

// Initialize
func (c *Collector) Initialize() error {
	var err = componenterror.ErrAlreadyStarted
	c.initializeOnce.Do(func() {
		if c.config.ReportPerCPU {
			c.hostUtilizationCounter, err = windows.NewPerfCounter(hostUtilizationDescriptor)
		} else {
			c.hostUtilizationCounter, err = windows.NewPerfCounter(hostUtilizationTotalDescriptor)
		}
		if err != nil {
			return
		}

		if c.config.ReportPerProcess {
			c.processUtilizationCounter, err = windows.NewPerfCounter(processUtilizationDescriptor)
			if err != nil {
				return
			}
		}

		c.startTime = time.Now()
		err = nil
	})
	return err
}

// Close
func (c *Collector) Close() error {
	var err = componenterror.ErrAlreadyStarted
	c.closeOnce.Do(func() {
		var errs []error

		if c.hostUtilizationCounter != nil {
			err = c.hostUtilizationCounter.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}

		if c.processUtilizationCounter != nil {
			err = c.processUtilizationCounter.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			err = componenterror.CombineErrors(errs)
		} else {
			err = nil
		}
	})
	return err
}

// CollectMetrics returns a list of collected CPU related metrics.
func (c *Collector) CollectMetrics(ctx context.Context) ([]*metricspb.Metric, error) {
	if c.startTime.IsZero() {
		return nil, errors.New("cpu collector has not been initialized")
	}

	return collectMetrics(c, ctx)
}

func (c *Collector) getPerCoreMetrics(ctx context.Context) ([]*metricspb.Metric, error) {
	_, span := trace.StartSpan(ctx, "cpucollector.getPerCoreMetrics")
	defer span.End()

	var errs []error
	metrics := []*metricspb.Metric{}

	// host/cpu/time

	cpuTimes, err := cpu.Times(c.config.ReportPerCPU)
	if err != nil {
		errs = append(errs, err)
	} else {
		metrics = append(metrics, c.createCpuCounterMetric(cpuTimes))
	}

	// host/cpu/utilization

	metric, err := c.hostUtilizationCounter.CollectDataAndConvertToMetric(c.startTime, MetricCPUUtilization)
	if err != nil {
		errs = append(errs, err)
	} else {
		metrics = append(metrics, metric)
	}

	if len(errs) > 0 {
		err = componenterror.CombineErrors(errs)
	}

	return metrics, err
}

func (c *Collector) getPerProcessMetrics(ctx context.Context) ([]*metricspb.Metric, error) {
	if !c.config.ReportPerProcess {
		return []*metricspb.Metric{}, nil
	}

	_, span := trace.StartSpan(ctx, "cpucollector.getPerProcessMetrics")
	defer span.End()

	var errs []error
	metrics := []*metricspb.Metric{}

	// process/cpu/utilization

	metric, err := c.processUtilizationCounter.CollectDataAndConvertToMetric(c.startTime, MetricProcessCPUUtilization)
	if err != nil {
		errs = append(errs, err)
	} else {
		metrics = append(metrics, metric)
	}

	metrics = append(metrics, c.createCpuPerProcessMetric(ctx))

	// var err error
	if len(errs) > 0 {
		err = componenterror.CombineErrors(errs)
	}

	return metrics, err
}

func (c *Collector) createCpuCounterMetric(cpuTimes []cpu.TimesStat) *metricspb.Metric {
	var timeseries = []*metricspb.TimeSeries{}
	for _, cpuTime := range cpuTimes {
		for _, stateTime := range []struct {
			label *metricspb.LabelValue
			val   float64
		}{
			{LabelValueCPUUser, cpuTime.User},
			{LabelValueCPUSystem, cpuTime.System},
			{LabelValueCPUIdle, cpuTime.Idle},
			{LabelValueCPUInterrupt, cpuTime.Irq},
			{LabelValueCPUIOWait, cpuTime.Iowait},
		} {
			timeseries = append(
				timeseries,
				internal.GetDoubleTimeSeries(
					c.startTime,
					stateTime.val,
					stateTime.label,
					&metricspb.LabelValue{Value: cpuTime.CPU, HasValue: true},
				),
			)
		}
	}

	metric := &metricspb.Metric{
		MetricDescriptor: MetricCPUSeconds,
		Timeseries:       timeseries,
	}

	return metric
}

func (c *Collector) createCpuPerProcessMetric(ctx context.Context) *metricspb.Metric {
	_, span := trace.StartSpan(ctx, "cpucollector.getPerProcessMetrics.gops")
	defer span.End()

	procs, _ := process.Processes()
	var timeseries = []*metricspb.TimeSeries{}
	for _, p := range procs {
		name, err := p.Name()
		if name == "[System Process]" {
			continue
		}

		if err != nil {
			print(err.Error())
			continue
		}

		var m *cpu.TimesStat
		m, err = p.Times()
		if err != nil {
			print(name + ": ")
			print(err.Error())
			continue // TODO log properly
		}

		timeseries = append(
			timeseries,
			internal.GetDoubleTimeSeries(
				c.startTime,
				float64(m.Total()),
				&metricspb.LabelValue{Value: name, HasValue: true},
			),
		)
	}

	metric := &metricspb.Metric{
		MetricDescriptor: MetricProcessCPUUtilizationGops,
		Timeseries:       timeseries,
	}

	return metric
}

var MetricProcessCPUUtilizationGops *metricspb.MetricDescriptor

func init() {
	metricProcessCPUUtilizationGops := *MetricProcessCPUUtilization
	metricProcessCPUUtilizationGops.Name = "process_gops/cpu/utilization"
	MetricProcessCPUUtilizationGops = &metricProcessCPUUtilizationGops
}
