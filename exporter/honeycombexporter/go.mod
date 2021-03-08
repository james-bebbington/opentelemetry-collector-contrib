module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/honeycombexporter

go 1.14

require (
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/google/go-cmp v0.5.3
	github.com/honeycombio/libhoney-go v1.15.0
	github.com/klauspost/compress v1.11.12
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/collector v0.14.1-0.20201117192738-131ff3e248b6
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.33.2
	google.golang.org/grpc/examples v0.0.0-20200728194956-1c32b02682df // indirect
	google.golang.org/protobuf v1.25.0
)
