module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerthrifthttpexporter

go 1.14

require (
	github.com/apache/thrift v0.14.1
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/google/go-cmp v0.5.4
	github.com/jaegertracing/jaeger v1.23.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.14.1-0.20201117192738-131ff3e248b6
	go.uber.org/zap v1.17.0
	google.golang.org/protobuf v1.25.0
)
