module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter

go 1.14

require (
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/hashicorp/go-hclog v0.16.1
	github.com/jaegertracing/jaeger v1.23.0
	github.com/logzio/jaeger-logzio v0.0.0-20201026090333-8336e3e13ec6
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.14.1-0.20201117192738-131ff3e248b6
	go.uber.org/zap v1.17.0
	google.golang.org/protobuf v1.25.0
)
