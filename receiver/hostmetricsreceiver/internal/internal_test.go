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

package internal_test

import (
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver/internal"
)

func TestGetInt64TimeSeries(t *testing.T) {
	t1 := time.Date(2020, 01, 01, 0, 0, 0, 0, time.UTC)
	labels := []*metricspb.LabelValue{
		{Value: "label1", HasValue: true},
		{HasValue: false},
	}

	ts := internal.GetInt64TimeSeries(t1, 1000, labels...)

	assert.EqualValues(t, internal.TimeToTimestamp(t1), ts.StartTimestamp)
	assert.EqualValues(t, labels, ts.LabelValues)
	assert.EqualValues(t, 1, len(ts.Points))
	assert.EqualValues(t, 1000, ts.Points[0].Value.(*metricspb.Point_Int64Value).Int64Value)
}

func TestGetDoubleTimeSeries(t *testing.T) {
	t1 := time.Date(2020, 01, 01, 0, 0, 0, 0, time.UTC)
	labels := []*metricspb.LabelValue{
		{Value: "label1", HasValue: true},
		{HasValue: false},
	}

	ts := internal.GetDoubleTimeSeries(t1, 1000, labels...)

	assert.EqualValues(t, internal.TimeToTimestamp(t1), ts.StartTimestamp)
	assert.EqualValues(t, labels, ts.LabelValues)
	assert.EqualValues(t, 1, len(ts.Points))
	assert.EqualValues(t, 1000, ts.Points[0].Value.(*metricspb.Point_DoubleValue).DoubleValue)
}

func TestTimeConverters(t *testing.T) {
	// Ensure that nanoseconds are also preserved.
	t1 := time.Date(2018, 10, 31, 19, 43, 35, 789, time.UTC)

	assert.EqualValues(t, int64(1541015015000000789), t1.UnixNano())
	tp := internal.TimeToTimestamp(t1)
	assert.EqualValues(t, &timestamp.Timestamp{Seconds: 1541015015, Nanos: 789}, tp)
	assert.EqualValues(t, int64(1541015015000000789), internal.TimestampToTime(tp).UnixNano())
}
