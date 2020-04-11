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

package internal

import (
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
)

// GetInt64TimeSeries returns a TimeSeries with the specified startTime, current value and labelValues.
func GetInt64TimeSeries(startTime time.Time, val uint64, labelVals ...(*metricspb.LabelValue)) *metricspb.TimeSeries {
	return &metricspb.TimeSeries{
		StartTimestamp: TimeToTimestamp(startTime),
		LabelValues:    labelVals,
		Points:         []*metricspb.Point{{Timestamp: TimeToTimestamp(time.Now()), Value: &metricspb.Point_Int64Value{Int64Value: int64(val)}}},
	}
}

// GetDoubleTimeSeries returns a TimeSeries with the specified startTime, current value and labelValues.
func GetDoubleTimeSeries(startTime time.Time, val float64, labelVals ...(*metricspb.LabelValue)) *metricspb.TimeSeries {
	return &metricspb.TimeSeries{
		StartTimestamp: TimeToTimestamp(startTime),
		LabelValues:    labelVals,
		Points:         []*metricspb.Point{{Timestamp: TimeToTimestamp(time.Now()), Value: &metricspb.Point_DoubleValue{DoubleValue: val}}},
	}
}

// TimeToTimestamp converts a time.Time to a timestamp.Timestamp pointer.
func TimeToTimestamp(t time.Time) *timestamp.Timestamp {
	if t.IsZero() {
		return nil
	}
	nanoTime := t.UnixNano()
	return &timestamp.Timestamp{
		Seconds: nanoTime / 1e9,
		Nanos:   int32(nanoTime % 1e9),
	}
}

// TimestampToTime converts a timestamp.Timestamp pointer to a time.Time.
func TimestampToTime(ts *timestamp.Timestamp) (t time.Time) {
	if ts == nil {
		return
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos))
}
