-- ============================================================
-- P99 Fraud Check Latency by Customer Tier (Timeseries Panel)
-- Generated via Claude Code + SigNoz MCP autonomous SRE workflow
-- Dashboard: Temporal Workflow SLO & Root Cause Correlator
-- ============================================================
-- This query was produced by:
--   1. Connecting Claude Code to SigNoz MCP server (localhost:8000)
--   2. Using signoz_aggregate_traces to investigate silent failures
--   3. Discovering the 25s timeout ceiling in activity.check_fraud
--   4. Writing ClickHouse SQL to operationalize the finding as a dashboard panel
-- ============================================================

SELECT
    toStartOfInterval(timestamp, INTERVAL 1 MINUTE) AS ts,
    attributes_string['customer.tier'] AS customer_tier,
    toFloat64(quantile(0.99)(duration_nano)) AS value
FROM signoz_traces.distributed_signoz_index_v3
WHERE
    timestamp BETWEEN $start_datetime AND $end_datetime
    AND ts_bucket_start BETWEEN $start_timestamp - 1800 AND $end_timestamp
    AND name = 'activity.check_fraud'
    AND mapContains(attributes_string, 'customer.tier')
GROUP BY ts, customer_tier
ORDER BY ts ASC
SETTINGS log_comment = 'signoz-writing-clickhouse-queries skill | 2026-07-21'
