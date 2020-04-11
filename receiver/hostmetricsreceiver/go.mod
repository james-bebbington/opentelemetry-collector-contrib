module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver

go 1.14

require (
	github.com/census-instrumentation/opencensus-proto v0.2.1
	github.com/golang/protobuf v1.3.5
	github.com/open-telemetry/opentelemetry-collector v0.3.1-0.20200411005314-55bf5e69393d
	github.com/stretchr/testify v1.5.1
	go.opencensus.io v0.22.3
	go.uber.org/zap v1.14.1
	k8s.io/client-go v12.0.0+incompatible // indirect
)
