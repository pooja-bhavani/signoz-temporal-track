module github.com/pooja-bhavani/signoz-temporal-track

go 1.23

require (
	go.opentelemetry.io/contrib/bridges/otelslog v0.6.0
	go.opentelemetry.io/otel v1.31.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.7.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.31.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.31.0
	go.opentelemetry.io/otel/sdk v1.31.0
	go.opentelemetry.io/otel/sdk/log v0.7.0
	go.opentelemetry.io/otel/sdk/metric v1.31.0
	go.opentelemetry.io/otel/trace v1.31.0
	go.temporal.io/api v1.44.0
	go.temporal.io/sdk v1.30.0
	go.temporal.io/sdk/contrib/opentelemetry v0.6.0
	google.golang.org/grpc v1.67.1
)
