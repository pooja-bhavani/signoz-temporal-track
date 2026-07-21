# SigNoz MCP Integration — Autonomous SRE with Claude Code

This directory contains the MCP (Model Context Protocol) configuration and artifacts
from using Claude Code as an autonomous SRE to investigate and operationalize
monitoring for Temporal workflow failures.

## What is SigNoz MCP?

SigNoz MCP exposes observability tools (trace search, log aggregation, metric queries,
field discovery) as MCP tool calls that AI agents can invoke programmatically. This
enables AI-powered SRE workflows where Claude Code can:

1. **Discover** — List services, fields, and field values without human intervention
2. **Investigate** — Query P99 latencies, error rates, and anomalies across dimensions
3. **Operationalize** — Generate ClickHouse SQL for permanent dashboard panels

## Setup

### 1. Enable SigNoz MCP Server

The MCP server runs alongside SigNoz. It's available on the same host as your
SigNoz instance:

```bash
# MCP endpoint (requires API key)
http://<SIGNOZ_HOST>:8000/mcp
```

### 2. Get Your API Key

In SigNoz UI: **Settings → API Keys → Create New Key**

### 3. Configure Claude Code

Add to your project's `.claude/settings.json` or use the config in `mcp-config.json`:

```json
{
  "mcpServers": {
    "signoz": {
      "type": "sse",
      "url": "http://localhost:8000/mcp",
      "headers": {
        "SIGNOZ-API-KEY": "<YOUR_API_KEY>"
      }
    }
  }
}
```

If SigNoz is on a remote host (e.g., EC2), tunnel the port:

```bash
ssh -f -N -L 8000:localhost:8000 -i "your-key.pem" ubuntu@<EC2_IP>
```

### 4. Install SigNoz Skills (Optional)

For structured query generation workflows:

```bash
# In Claude Code
/install-skill signoz-generating-queries
/install-skill signoz-writing-clickhouse-queries
```

## Available MCP Tools

| Tool | Purpose |
|---|---|
| `signoz_list_services` | Discover all instrumented services |
| `signoz_get_service_top_operations` | Get top operations for a service |
| `signoz_get_field_keys` | Discover available fields for a signal |
| `signoz_get_field_values` | Get distinct values for a field |
| `signoz_search_traces` | Find specific traces/spans with filters |
| `signoz_aggregate_traces` | Compute P50/P95/P99, counts, grouped by fields |
| `signoz_search_logs` | Search log entries by text, severity, fields |
| `signoz_aggregate_logs` | Aggregate log data (counts, patterns) |
| `signoz_query_metrics` | Query pre-aggregated metrics with formulas |
| `signoz_list_metrics` | Discover available metrics |
| `signoz_get_trace_details` | Get full trace details by trace ID |
| `signoz_execute_builder_query` | Advanced Query Builder v5 queries |

## Investigation Workflow

See `investigation-queries.md` for the complete autonomous SRE workflow that:

1. Discovered `temporal-worker` service and `customer.tier` attribute via MCP
2. Queried P99 latency for `activity.check_fraud` grouped by customer tier
3. Uncovered a **silent failure** — 25s timeout ceiling across all tiers (170x slower
   than other activities, zero errors thrown)
4. Generated `p99-fraud-check-panel.sql` to operationalize as a permanent dashboard panel

## Files

```
mcp/
├── README.md                    # This file
├── mcp-config.json              # Claude Code MCP server configuration
├── investigation-queries.md     # Full MCP tool call sequence with results
└── p99-fraud-check-panel.sql    # ClickHouse SQL generated from investigation
```
