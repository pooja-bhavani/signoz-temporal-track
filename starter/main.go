package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"

	"github.com/pooja-bhavani/signoz-temporal-track/shared"
)

var temporalClient client.Client

func main() {
	ctx := context.Background()

	// Initialize OpenTelemetry
	shutdown, err := shared.InitTelemetry(ctx, "temporal-starter")
	if err != nil {
		log.Fatalf("Failed to init telemetry: %v", err)
	}
	defer shutdown()

	// Create Temporal OTel interceptor
	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		log.Fatalf("Failed to create tracing interceptor: %v", err)
	}

	metricsHandler := opentelemetry.NewMetricsHandler(opentelemetry.MetricsHandlerOptions{})

	temporalAddr := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddr == "" {
		temporalAddr = "localhost:7233"
	}

	temporalClient, err = client.Dial(client.Options{
		HostPort:       temporalAddr,
		MetricsHandler: metricsHandler,
		Interceptors:   []interceptor.ClientInterceptor{tracingInterceptor},
	})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer temporalClient.Close()

	http.HandleFunc("/order", handleOrder)
	http.HandleFunc("/health", handleHealth)

	go func() {
		log.Println("Starter API listening on :8005")
		if err := http.ListenAndServe(":8005", nil); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down starter...")
}

func handleOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var input shared.OrderInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	workflowID := fmt.Sprintf("order-%s", input.OrderID)

	we, err := temporalClient.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: shared.TaskQueue,
	}, "OrderProcessingWorkflow", input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"workflow_id": we.GetID(),
		"run_id":     we.GetRunID(),
		"status":     "started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
