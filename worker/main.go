package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	"github.com/pooja-bhavani/signoz-temporal-track/shared"
	"github.com/pooja-bhavani/signoz-temporal-track/worker/workflows"
)

func main() {
	ctx := context.Background()

	// Initialize OpenTelemetry
	shutdown, err := shared.InitTelemetry(ctx, "temporal-worker")
	if err != nil {
		log.Fatalf("Failed to init telemetry: %v", err)
	}
	defer shutdown()

	// Create Temporal OTel interceptor
	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		log.Fatalf("Failed to create tracing interceptor: %v", err)
	}

	// Create metrics handler
	metricsHandler := opentelemetry.NewMetricsHandler(opentelemetry.MetricsHandlerOptions{})

	temporalAddr := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddr == "" {
		temporalAddr = "localhost:7233"
	}

	// Connect to Temporal
	c, err := client.Dial(client.Options{
		HostPort:       temporalAddr,
		MetricsHandler: metricsHandler,
		Interceptors:   []interceptor.ClientInterceptor{tracingInterceptor},
	})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, shared.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	// Register workflows and activities
	w.RegisterWorkflow(workflows.OrderProcessingWorkflow)
	w.RegisterActivity(&workflows.Activities{})

	// Start worker
	go func() {
		if err := w.Run(worker.InterruptCh()); err != nil {
			log.Fatalf("Worker failed: %v", err)
		}
	}()

	log.Println("Temporal worker started, listening on task queue:", shared.TaskQueue)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down worker...")
}
