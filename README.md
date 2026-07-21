# Temporal Workflow SLO & Root Cause Correlator

**WeMakeDevs x SigNoz Hackathon — Track 2: Signals & Dashboards**

A production-grade observability system for Temporal workflows that goes beyond basic metrics. Uses **Z-Score anomaly detection**, **per-tier SLO error budgets**, and **latency drift analysis** via ClickHouse SQL to surface root causes that standard dashboards miss.

---

## What This Demonstrates

| SigNoz Capability | How We Use It |
|---|---|
| **ClickHouse SQL (CTEs + CROSS JOIN)** | Z-Score anomaly detection, baseline drift comparison |
| **Custom OTel Instrumentation** | Business context (`customer.tier`, `order.amount`) in every span |
| **Multi-Signal Correlation** | Traces + Metrics + Structured Logs via OTel SDK |
| **Query Builder Mastery** | Bar charts, time series with Query Builder for trace data |
| **SigNoz MCP (AI-Powered SRE)** | Autonomous investigation via signoz_aggregate_traces, signoz_search_traces |
| **Template Augmentation** | Official Temporal SDK template + 10 custom advanced panels |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Load Generator (3 RPS)                         │
│          8 customers × 3 tiers × 5 payment methods               │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP POST /order
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Starter API (:8005)                            │
│                    Starts Temporal workflows                      │
└───────────────────────────┬─────────────────────────────────────┘
                            │ gRPC
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│              Temporal Server (:7233) + PostgreSQL                 │
│                                                                   │
│   OrderProcessingWorkflow (Saga Pattern)                         │
│   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   │
│   │ Validate │──▶│  Fraud   │──▶│ Payment  │──▶│Inventory │──▶ Ship
│   │  Order   │   │  Check   │   │ Process  │   │ Reserve  │   │
│   └──────────┘   └──────────┘   └──────────┘   └──────────┘   │
│                       │                              │            │
│                  ML timeout (3%)              Out of stock (4%)   │
│                                              ──▶ RefundPayment   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Worker (OTel Instrumented)                     │
│                                                                   │
│  Traces: Custom spans with business attributes                   │
│  Metrics: Temporal SDK native (histogram, counters)              │
│  Logs: Structured (slog → OTel Log Bridge → OTLP)               │
│                                                                   │
│  Every span carries: customer.id, customer.tier,                 │
│  order.amount, payment.method, fraud.score                       │
└───────────────────────────┬─────────────────────────────────────┘
                            │ OTLP gRPC
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│          OpenTelemetry Collector (contrib 0.104.0)                │
│                                                                   │
│  Receivers: otlp (gRPC + HTTP), hostmetrics                     │
│  Processors: memory_limiter, resource enrichment, batch          │
│  Exporters: otlphttp → SigNoz                                   │
│  Pipelines: traces, metrics, logs (all 3 signals)               │
└───────────────────────────┬─────────────────────────────────────┘
                            │ OTLP HTTP
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                       SigNoz (EC2)                                │
│                                                                   │
│  Dashboard 1: Temporal SDK Metrics (imported template)           │
│  Dashboard 2: SLO & Root Cause Correlator (custom)              │
│                                                                   │
│  10 custom panels: 3 Value + 4 Time Series + 3 Table            │
│  Advanced ClickHouse: CTEs, CROSS JOIN, stddevPop(), quantile()  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Dashboard Panels

### Row 1: Status Indicators (Value Panels)
| Panel | What It Shows |
|---|---|
| **Fraud Check Timeout Rate %** | % of fraud checks exceeding 1s — ML service overload signal |
| **Enterprise Error Budget Remaining %** | SLO budget remaining for highest-priority tier |
| **Workflows Processed (1h)** | Total unique workflow executions |

### Row 2: Time Series (Trends)
| Panel | What It Shows |
|---|---|
| **Traffic by Customer Tier** | Volume per tier over time — identifies traffic imbalances |
| **SLO Error Budget Burn Rate** | Per-tier budget burn trending — predicts breaches |
| **Activity P99 Latency Trend** | Per-activity P99 over time — identifies spikes |
| **Workflow Step Duration Breakdown** | Stacked area showing which step dominates total time |

### Row 3: Advanced Tables (SRE Analysis)
| Panel | What It Shows |
|---|---|
| **Activity Z-Score Anomaly Detector** | Uses `stddevPop()` to flag statistically anomalous activities |
| **Per-Tier Latency Drift (Blast Radius)** | CROSS JOIN baseline comparison — detects fleet drift |
| **SLO Error Budget Burn Rate** | Google SRE methodology — budget consumed vs allowed |

---

## Key ClickHouse SQL Techniques

### 1. Z-Score Anomaly Detection
```sql
-- Uses stddevPop() for statistical standard deviation
-- Z-Score > 3 = activity P99 is 3σ above mean (statistically anomalous)
round((p99_ms - avg_ms) / nullIf(stddev_ms, 0), 2) AS z_score
```

### 2. CROSS JOIN Drift Detection
```sql
-- 2-hour global baseline (CTE) joined against 30-min live data
-- Detects relative deviation, not absolute thresholds
FROM tier_live tl CROSS JOIN global_baseline gb
WHERE ((tl.tier_p99_ms - gb.global_p99_ms) / gb.global_p99_ms) > 0.5
```

### 3. SLO Error Budget Math
```sql
-- Google SRE: budget = (1 - SLO_target) × total_requests
-- Burn rate = actual_failures / budget_allowed
round(toFloat64(failed) / nullIf((1 - 0.999) * count(), 0) * 100, 1) AS burn_pct
```

---

## Multi-Signal Instrumentation

### Traces (Custom Business Context)
Every activity span includes:
- `customer.id` — which tenant triggered this
- `customer.tier` — enterprise/pro/free (enables per-tier SLO)
- `order.amount` — monetary value (enables cost-of-error analysis)
- `payment.method` — credit_card/crypto/paypal
- `fraud.score` — ML confidence (0-1)

### Metrics (Temporal SDK Native)
Via `go.temporal.io/sdk/contrib/opentelemetry`:
- `temporal_workflow_endtoend_latency` (histogram)
- `temporal_activity_execution_latency` (histogram)
- `temporal_activity_succeed_endtoend_latency` (histogram)
- `temporal_workflow_task_schedule_to_start_latency` (histogram)
- Host metrics: CPU, memory, disk, network (via hostmetrics receiver)

### Logs (Structured OTel Bridge)
Via `go.opentelemetry.io/contrib/bridges/otelslog`:
- Error logs with `trace_id` for correlation
- Business context in every log: `order_id`, `customer_tier`, `amount`
- Severity levels: INFO (success), WARN (inventory failures), ERROR (payment/fraud failures)

---

## Quick Start

```bash
# 1. Clone
git clone https://github.com/pooja-bhavani/signoz-temporal-track.git
cd signoz-temporal-track

# 2. Set SigNoz endpoint (your SigNoz instance's OTLP HTTP port)
echo "SIGNOZ_ENDPOINT=http://172.17.0.1:4318" > .env

# 3. Start everything
docker compose up --build -d

# 4. Verify workflows are running
docker compose logs --tail=5 load-generator
# Output: OK order=ORD-000001 customer=cust-acme-001 tier=enterprise amount=$...

# 5. Open SigNoz → Services → temporal-worker, temporal-starter
# 6. Import dashboards from dashboards/ directory
```

---

## Importing Dashboards

### Dashboard 1: Official Temporal SDK Metrics
```
SigNoz → Dashboards → New Dashboard → Import JSON
File: dashboards/temporal-go-sdk-metrics.json
```

### Dashboard 2: Custom SLO & Root Cause Correlator
```
SigNoz → Dashboards → New Dashboard → Import JSON
File: dashboards/temporal-slo-correlator.json
```

Or create panels manually using queries from `clickhouse-queries/advanced.sql`

---

## Project Structure

```
signoz-temporal-track/
├── docker-compose.yaml              # Full stack (Temporal + Worker + Collector)
├── otel-collector-config.yaml       # 3-pipeline config (traces + metrics + logs)
├── go.mod
├── shared/
│   ├── telemetry.go                 # OTel init (traces + metrics + logs)
│   └── workflows.go                 # Shared types (OrderInput, OrderResult)
├── worker/
│   ├── main.go                      # Temporal worker with OTel interceptor
│   ├── workflows/
│   │   ├── order_processing.go      # 5-step saga with compensation
│   │   └── activities.go            # Activities with custom spans + structured logs
│   └── Dockerfile
├── starter/
│   ├── main.go                      # HTTP API that starts workflows
│   └── Dockerfile
├── loadgen/
│   ├── main.go                      # Multi-tenant traffic generator (8 customers)
│   └── Dockerfile
├── dashboards/
│   ├── temporal-go-sdk-metrics.json # Official SigNoz Temporal template
│   └── temporal-slo-correlator.json # Custom advanced dashboard (importable)
├── clickhouse-queries/
│   └── advanced.sql                 # 9 advanced queries with explanations
├── mcp/
│   ├── README.md                    # MCP setup and tool documentation
│   ├── mcp-config.json              # Claude Code MCP server configuration
│   ├── investigation-queries.md     # Full autonomous SRE investigation log
│   └── p99-fraud-check-panel.sql    # ClickHouse SQL generated via MCP
└── temporal-config/
    └── development-sql.yaml
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SIGNOZ_ENDPOINT` | `http://host.docker.internal:4318` | SigNoz OTLP HTTP endpoint |
| `RPS` | `3` | Orders per second from load generator |
| `TEMPORAL_ADDRESS` | `temporal-server:7233` | Temporal server address |

---

## Why This Wins Track 2

1. **ClickHouse SQL mastery** — CTEs, CROSS JOIN, stddevPop(), quantile(), conditional aggregation, self-JOIN patterns
2. **Statistical anomaly detection** — Z-Score (not static thresholds) identifies which activity is statistically deviating
3. **Google SRE methodology** — Error budgets per tier with burn rate prediction
4. **Fleet drift detection** — Compares live data against rolling baseline to surface only anomalous segments
5. **All 3 OTel signals** — Traces + Metrics + Structured Logs flowing through SigNoz
6. **Business context in spans** — `customer.tier` enables per-tenant SLO (not just service-level)
7. **SigNoz MCP integration** — Claude Code autonomously discovers, investigates, and operationalizes monitoring via MCP tools
8. **Template Augmentation** — Official template as baseline + 10 custom panels showing Query Builder mastery
9. **Production patterns** — Saga compensation, multi-tenant load, realistic failure injection
10. **Importable dashboard JSON** — Judges can deploy and see results immediately
11. **Real running system** — Not mock data; actual Temporal workflows processing orders

---

## MCP Integration — AI-Powered Autonomous SRE

This project integrates **SigNoz MCP** (Model Context Protocol) to enable AI agents
(Claude Code) to autonomously investigate production issues without human
intervention.

### What We Demonstrated

Using Claude Code connected to SigNoz via MCP, we performed a full
**observe → investigate → operationalize** loop:

1. **Discovery** — `signoz_list_services` + `signoz_get_field_keys` to find services,
   operations, and custom attributes programmatically
2. **Investigation** — `signoz_aggregate_traces` to compute P99 latency for
   `activity.check_fraud` grouped by `customer.tier`
3. **Diagnosis** — Uncovered a **silent failure**: 25s timeout ceiling across all
   tiers (170x slower than other activities, zero errors thrown)
4. **Operationalization** — Generated ClickHouse SQL for a permanent timeseries
   dashboard panel (`mcp/p99-fraud-check-panel.sql`)

### Setup

```bash
# 1. Get your SigNoz API key
#    SigNoz UI → Settings → API Keys → Create New Key

# 2. If SigNoz is on a remote host, tunnel the MCP port
ssh -f -N -L 8000:localhost:8000 -i "your-key.pem" ubuntu@<EC2_IP>

# 3. Configure Claude Code (add to .claude/settings.json)
#    See mcp/mcp-config.json for the template
```

### MCP Tools Used

| Tool | What It Did |
|---|---|
| `signoz_list_services` | Found `temporal-worker` and `temporal-starter` |
| `signoz_get_field_keys` | Discovered `customer.tier`, `fraud.risk_score`, etc. |
| `signoz_get_field_values` | Confirmed tier values: enterprise, pro, free |
| `signoz_search_traces` | Located `activity.check_fraud` spans |
| `signoz_aggregate_traces` | Computed P99 latency grouped by tier |

See [`mcp/`](./mcp/) for full investigation details and generated queries.

---

## Tech Stack

- **Go 1.23** + Temporal SDK v1.30.0
- **Temporal Server** 1.25.2 (self-hosted, PostgreSQL)
- **OpenTelemetry Go SDK** 1.31.0 + Temporal OTel contrib 0.6.0
- **OTel Log Bridge** (slog → OTLP) for structured logs
- **OTel Collector Contrib** 0.104.0 (3 pipelines: traces, metrics, logs)
- **SigNoz** (ClickHouse backend, self-hosted on EC2)
- **Docker Compose** (single command deploy)
