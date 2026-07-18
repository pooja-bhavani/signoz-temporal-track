# Temporal Workflow SLO & Root Cause Correlator

**WeMakeDevs x SigNoz Hackathon — Track 2: Signals & Dashboards**

Deep OpenTelemetry instrumentation of Temporal workflows with cross-signal correlation dashboards and predictive SLO alerting in SigNoz.

## Architecture

```
                    ┌────────────────────┐
                    │   Load Generator   │
                    │   Multi-tenant     │
                    │   order traffic    │
                    └────────┬───────────┘
                             │ HTTP POST /order
                             ▼
                    ┌────────────────────┐
                    │   Starter API      │
                    │   :8005            │
                    │   (starts workflows)│
                    └────────┬───────────┘
                             │ gRPC
                             ▼
┌──────────────────────────────────────────────────────┐
│              Temporal Server (:7233)                   │
│   ┌─────────────────────────────────────────────┐    │
│   │       OrderProcessingWorkflow                │    │
│   │                                             │    │
│   │  ValidateOrder → CheckFraud → ProcessPayment│    │
│   │       → ReserveInventory → ScheduleShipping │    │
│   │                                             │    │
│   │  (Saga pattern with compensation)           │    │
│   └─────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────┘
                             │
                             ▼
                    ┌────────────────────┐
                    │   Worker           │
                    │   (executes        │
                    │   activities)      │
                    │   + OTel SDK       │
                    └────────┬───────────┘
                             │ OTLP gRPC
                             ▼
                    ┌────────────────────┐
                    │  OTel Collector    │
                    │  + hostmetrics     │
                    └────────┬───────────┘
                             │ OTLP HTTP
                             ▼
                    ┌────────────────────┐
                    │  SigNoz            │
                    │  Traces + Metrics  │
                    │  + Logs            │
                    └────────────────────┘
```

## What Makes This Win Track 2

### Template Augmentation Strategy
1. **Import** the official SigNoz Temporal Go SDK dashboard template (baseline worker metrics)
2. **Augment** with 7 advanced custom ClickHouse panels that show Query Builder mastery

### Custom OTel Instrumentation (Business Context)
- Every Temporal activity span carries: `customer.id`, `customer.tier`, `order.amount`, `payment.method`
- Temporal SDK's `opentelemetry` contrib package for native tracing + metrics
- Custom search attributes (`customer_tier`, `order_value`) propagated through workflow headers

### Advanced ClickHouse Queries (3 flagship)
1. **Z-Score Anomaly Detection** — rolling 15-min baseline with stddev, flags spikes dynamically (not static thresholds)
2. **Parent-Child Trace Correlation** — self-join on `spanId = parentSpanId` to find which activity causes workflow slowdowns
3. **Predictive SLO Depletion** — 6-hour burn rate extrapolated to "hours until budget exhausted"

### Cross-Signal Correlation
4. **Activity Retries ↔ Error Logs** — correlates retry spikes with log frequency in same time window
5. **Activity Latency vs Worker CPU** — maps P99 execution time against infrastructure metrics
6. **Workflow Step Bottleneck** — identifies which pipeline stage is the throughput limiter
7. **End-to-End SLO by Tier** — per-tier workflow success rate vs targets

## Quick Start

```bash
# 1. Clone
git clone https://github.com/pooja-bhavani/signoz-temporal-track.git
cd signoz-temporal-track

# 2. Set SigNoz endpoint
echo "SIGNOZ_ENDPOINT=http://172.17.0.1:4318" > .env

# 3. Start everything
docker compose up --build -d

# 4. Verify
docker compose logs --tail=5 load-generator
# Should show: OK order=ORD-000001 customer=cust-acme-001 tier=enterprise amount=$...

# 5. Open SigNoz → Services → temporal-worker, temporal-starter
```

## Importing Dashboards

### Step 1: Import official Temporal template
Download from: `https://raw.githubusercontent.com/SigNoz/dashboards/main/temporal.io/temporal-go-sdk-metrics.json`

SigNoz UI → Dashboards → New Dashboard → Import JSON → paste/upload

### Step 2: Create custom panels
Dashboard → Add Panel → ClickHouse Query → paste queries from `clickhouse-queries/advanced.sql`

## Project Structure

```
signoz-temporal-track/
├── docker-compose.yaml
├── otel-collector-config.yaml
├── go.mod
├── shared/
│   ├── telemetry.go           # OTel init (traces + metrics)
│   └── workflows.go           # Shared types
├── worker/
│   ├── main.go                # Temporal worker with OTel interceptor
│   ├── workflows/
│   │   ├── order_processing.go  # Main workflow (5-step pipeline)
│   │   └── activities.go        # Activities with custom spans + business context
│   └── Dockerfile
├── starter/
│   ├── main.go                # HTTP API that starts workflows
│   └── Dockerfile
├── loadgen/
│   ├── main.go                # Multi-tenant traffic generator
│   └── Dockerfile
├── dashboards/
│   └── temporal-go-sdk-metrics.json  # Official SigNoz template
├── clickhouse-queries/
│   └── advanced.sql           # 7 advanced cross-signal queries
└── temporal-config/
    └── development-sql.yaml
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SIGNOZ_ENDPOINT` | `http://host.docker.internal:4318` | SigNoz OTLP HTTP endpoint |
| `RPS` | `3` | Orders per second from load generator |
| `TEMPORAL_ADDRESS` | `temporal-server:7233` | Temporal server address |

## Demo Scenario

1. Start services → load generator creates orders across enterprise/pro/free tiers
2. SigNoz shows Temporal worker metrics + custom activity traces
3. Z-Score panel detects when fraud check latency spikes (high-value orders)
4. Parent-child panel shows fraud check is 80%+ of total workflow time for flagged orders
5. SLO panel shows enterprise tier maintaining 99.9% while free tier burns budget faster
6. Predictive query shows "12.5 hours to budget exhaustion" for free tier

## Tech Stack

- **Go 1.22** + Temporal SDK v1.31
- **Temporal Server** 1.25 (self-hosted, SQLite)
- **OpenTelemetry Go SDK** + Temporal OTel contrib (traces + metrics)
- **OTel Collector Contrib** 0.104.0
- **SigNoz** (ClickHouse backend)
- **Docker Compose**
