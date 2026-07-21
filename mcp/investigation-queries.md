# MCP Investigation Queries

These are the exact MCP tool calls used by Claude Code acting as an autonomous SRE
to investigate the silent failure in `activity.check_fraud`.

## Step 1: Service Discovery

```json
{
  "method": "tools/call",
  "params": {
    "name": "signoz_list_services",
    "arguments": {}
  }
}
```

**Result:** Discovered `temporal-starter` service (top-level workflow initiator).

## Step 2: Field Discovery

```json
{
  "method": "tools/call",
  "params": {
    "name": "signoz_get_field_keys",
    "arguments": {
      "signal": "traces"
    }
  }
}
```

**Result:** Discovered 80+ trace fields including:
- `customer.tier` (attribute, string) — enterprise, pro, free
- `fraud.risk_score` (attribute, number)
- `fraud.analysis_duration_ms` (attribute, number)
- `temporalWorkflowID` (attribute, string)
- `duration_nano` (span, number)

## Step 3: Validate Field Values

```json
{
  "method": "tools/call",
  "params": {
    "name": "signoz_get_field_values",
    "arguments": {
      "signal": "traces",
      "name": "customer.tier",
      "fieldContext": "attribute"
    }
  }
}
```

**Result:** Confirmed values — `enterprise`, `pro`, `free`.

## Step 4: Search for Fraud Check Spans

```json
{
  "method": "tools/call",
  "params": {
    "name": "signoz_search_traces",
    "arguments": {
      "filter": "name CONTAINS 'check_fraud'",
      "timeRange": "1h",
      "limit": 5
    }
  }
}
```

**Result:** Found spans on `temporal-worker` service (not `temporal-starter`).
Span name: `activity.check_fraud`, kind: Internal, durations: 120ms–217ms.

## Step 5: P99 Aggregation by Customer Tier

```json
{
  "method": "tools/call",
  "params": {
    "name": "signoz_aggregate_traces",
    "arguments": {
      "searchContext": "P99 execution latency for activity.check_fraud grouped by customer.tier",
      "aggregation": "p99",
      "aggregateOn": "duration_nano",
      "filter": "name = 'activity.check_fraud'",
      "groupBy": "customer.tier",
      "requestType": "scalar",
      "timeRange": "1h",
      "service": "temporal-worker"
    }
  }
}
```

**Result:**

| Customer Tier | P99 Latency |
|---|---|
| pro | 25,199,942,524 ns (~25.20s) |
| free | 25,198,009,983 ns (~25.20s) |
| enterprise | 25,175,404,774 ns (~25.18s) |

## Diagnosis

The P99 latency is nearly identical (~25.2s) across all three tiers — 170x slower
than every other activity (50–150ms). This indicates a **systemic timeout ceiling**
in the fraud check activity, likely from:
- Temporal activity StartToClose timeout set at 25s
- Downstream fraud ML model or external API timing out uniformly
- No errors thrown (silent failure) — the activity completes "successfully" but
  at the timeout boundary

This was surfaced entirely through MCP tool calls without accessing the SigNoz UI.
