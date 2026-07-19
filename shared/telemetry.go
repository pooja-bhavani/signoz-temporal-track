package shared

import (
	"context"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var OtelLogger log.Logger

func InitTelemetry(ctx context.Context, serviceName string) (func(), error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(5*time.Second)),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Metric exporter
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// Log exporter (direct OTLP gRPC → SigNoz Logs)
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	OtelLogger = lp.Logger(serviceName)

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tp.Shutdown(ctx)
		mp.Shutdown(ctx)
		lp.Shutdown(ctx)
	}

	return shutdown, nil
}

// LogInfo emits a structured INFO log to SigNoz
func LogInfo(ctx context.Context, msg string, attrs ...otellog.KeyValue) {
	if OtelLogger == nil {
		return
	}
	var record otellog.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(otellog.SeverityInfo)
	record.SetSeverityText("INFO")
	record.SetBody(otellog.StringValue(msg))
	record.AddAttributes(attrs...)
	OtelLogger.Emit(ctx, record)
}

// LogError emits a structured ERROR log to SigNoz
func LogError(ctx context.Context, msg string, attrs ...otellog.KeyValue) {
	if OtelLogger == nil {
		return
	}
	var record otellog.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(otellog.SeverityError)
	record.SetSeverityText("ERROR")
	record.SetBody(otellog.StringValue(msg))
	record.AddAttributes(attrs...)
	OtelLogger.Emit(ctx, record)
}

// LogWarn emits a structured WARN log to SigNoz
func LogWarn(ctx context.Context, msg string, attrs ...otellog.KeyValue) {
	if OtelLogger == nil {
		return
	}
	var record otellog.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(otellog.SeverityWarn)
	record.SetSeverityText("WARN")
	record.SetBody(otellog.StringValue(msg))
	record.AddAttributes(attrs...)
	OtelLogger.Emit(ctx, record)
}
