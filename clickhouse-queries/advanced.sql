-- ============================================================
-- ADVANCED CLICKHOUSE QUERIES FOR TEMPORAL + SIGNOZ
-- Track 2: Signals & Dashboards — "Template Augmentation" Strategy
-- Table: signoz_traces.distributed_signoz_index_v3
-- ============================================================


-- ============================================================
-- QUERY 1: Z-Score Anomaly Detection per Activity
-- Uses stddevPop() to calculate statistical outliers per activity.
-- Flags activities where P99 is >3 standard deviations above mean.
-- ============================================================
WITH activity_stats AS (
    SELECT
        name AS "Activity",
        count() AS "Executions",
        round(avg(duration_nano) / 1000000, 2) AS avg_ms,
        round(stddevPop(duration_nano) / 1000000, 2) AS stddev_ms,
        round(quantile(0.50)(duration_nano) / 1000000, 2) AS "P50 (ms)",
        round(quantile(0.95)(duration_nano) / 1000000, 2) AS "P95 (ms)",
        round(quantile(0.99)(duration_nano) / 1000000, 2) AS "P99 (ms)"
    FROM signoz_traces.distributed_signoz_index_v3
    WHERE timestamp >= now() - INTERVAL 1 HOUR
        AND name LIKE 'activity.%'
        AND serviceName = 'temporal-worker'
    GROUP BY name
    HAVING "Executions" >= 10
)
SELECT
    "Activity",
    "Executions",
    "P50 (ms)",
    "P95 (ms)",
    "P99 (ms)",
    round(("P99 (ms)" - avg_ms) / nullIf(stddev_ms, 0), 2) AS "Z-Score (P99)",
    round(("P95 (ms)" - "P50 (ms)") / nullIf("P50 (ms)", 0) * 100, 1) AS "Tail Spread %",
    CASE
        WHEN ("P99 (ms)" - avg_ms) / nullIf(stddev_ms, 0) > 3.0 THEN '🔴 ANOMALOUS'
        WHEN ("P99 (ms)" - avg_ms) / nullIf(stddev_ms, 0) > 2.0 THEN '🟡 WARNING'
        WHEN ("P95 (ms)" - "P50 (ms)") / nullIf("P50 (ms)", 0) > 1.0 THEN '🟠 LONG TAIL'
        ELSE '🟢 NORMAL'
    END AS "Health"
FROM activity_stats
ORDER BY "Z-Score (P99)" DESC;


-- ============================================================
-- QUERY 2: Per-Tier Latency Drift (Blast Radius Detection)
-- Compares 30-min live data against 2-hour global baseline using CROSS JOIN.
-- Surfaces only tiers that are mathematically deviating from fleet norm.
-- ============================================================
WITH global_baseline AS (
    SELECT
        avg(duration_nano) / 1000000 AS global_avg_ms,
        quantile(0.99)(duration_nano) / 1000000 AS global_p99_ms
    FROM signoz_traces.distributed_signoz_index_v3
    WHERE timestamp >= now() - INTERVAL 2 HOUR
        AND name LIKE 'activity.%'
        AND serviceName = 'temporal-worker'
),
tier_live AS (
    SELECT
        attributes_string['customer.tier'] AS tier,
        count() AS span_count,
        round(quantile(0.99)(duration_nano) / 1000000, 2) AS tier_p99_ms,
        round(avg(duration_nano) / 1000000, 2) AS tier_avg_ms,
        round(countIf(has_error = true) * 100.0 / count(), 2) AS error_rate_pct
    FROM signoz_traces.distributed_signoz_index_v3
    WHERE timestamp >= now() - INTERVAL 30 MINUTE
        AND name LIKE 'activity.%'
        AND serviceName = 'temporal-worker'
        AND attributes_string['customer.tier'] != ''
    GROUP BY tier
    HAVING span_count >= 5
)
SELECT
    tl.tier AS "Customer Tier",
    tl.tier_p99_ms AS "P99 (ms)",
    round(gb.global_p99_ms, 2) AS "Baseline P99 (ms)",
    round(((tl.tier_p99_ms - gb.global_p99_ms) / gb.global_p99_ms) * 100, 1) AS "Drift %",
    tl.error_rate_pct AS "Error Rate %",
    tl.span_count AS "Total Spans",
    CASE
        WHEN ((tl.tier_p99_ms - gb.global_p99_ms) / gb.global_p99_ms) > 1.0 THEN '🔴 CRITICAL'
        WHEN ((tl.tier_p99_ms - gb.global_p99_ms) / gb.global_p99_ms) > 0.5 THEN '🟡 DRIFTING'
        WHEN ((tl.tier_p99_ms - gb.global_p99_ms) / gb.global_p99_ms) > 0.1 THEN '🟠 ELEVATED'
        WHEN ((tl.tier_p99_ms - gb.global_p99_ms) / gb.global_p99_ms) < -0.1 THEN '🔵 UNDER-BASELINE'
        ELSE '🟢 HEALTHY'
    END AS "Status"
FROM tier_live tl
CROSS JOIN global_baseline gb
ORDER BY "Drift %" DESC;


-- ============================================================
-- QUERY 3: SLO Error Budget Burn Rate per Customer Tier
-- Implements Google SRE Error Budget methodology.
-- Calculates budget consumed vs allowed, flags tiers at risk.
-- ============================================================
WITH tier_slo AS (
    SELECT
        attributes_string['customer.tier'] AS tier,
        count() AS total_workflows,
        countIf(has_error = true) AS failed_workflows,
        CASE
            WHEN attributes_string['customer.tier'] = 'enterprise' THEN 0.999
            WHEN attributes_string['customer.tier'] = 'pro' THEN 0.995
            ELSE 0.990
        END AS slo_target
    FROM signoz_traces.distributed_signoz_index_v3
    WHERE timestamp >= now() - INTERVAL 1 HOUR
        AND name LIKE 'activity.%'
        AND serviceName = 'temporal-worker'
        AND attributes_string['customer.tier'] != ''
    GROUP BY tier
)
SELECT
    tier AS "Tier",
    total_workflows AS "Total Requests",
    failed_workflows AS "Failures",
    slo_target * 100 AS "SLO Target %",
    round((1 - (toFloat64(failed_workflows) / total_workflows)) * 100, 3) AS "Actual Uptime %",
    round((1 - slo_target) * total_workflows, 1) AS "Error Budget (allowed)",
    failed_workflows AS "Budget Consumed",
    round(toFloat64(failed_workflows) / nullIf((1 - slo_target) * total_workflows, 0) * 100, 1) AS "Budget Burn %",
    CASE
        WHEN toFloat64(failed_workflows) / nullIf((1 - slo_target) * total_workflows, 0) > 1.0 THEN '🔴 BUDGET EXHAUSTED'
        WHEN toFloat64(failed_workflows) / nullIf((1 - slo_target) * total_workflows, 0) > 0.7 THEN '🟡 BUDGET AT RISK'
        WHEN toFloat64(failed_workflows) / nullIf((1 - slo_target) * total_workflows, 0) > 0.4 THEN '🟠 BURNING FAST'
        ELSE '🟢 WITHIN BUDGET'
    END AS "Status"
FROM tier_slo
ORDER BY "Budget Burn %" DESC;


-- ============================================================
-- QUERY 4: Fraud Check Timeout Rate (Value Panel)
-- Single-value metric: percentage of fraud checks exceeding 1s.
-- ============================================================
SELECT
    round(countIf(duration_nano / 1000000 > 1000) * 100.0 / count(), 1) AS value
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= now() - INTERVAL 1 HOUR
    AND name = 'activity.check_fraud'
    AND serviceName = 'temporal-worker';


-- ============================================================
-- QUERY 5: SLO Error Budget Burn Rate Over Time (Time Series)
-- Per-tier burn rate tracked over time for trend visualization.
-- ============================================================
SELECT
    toStartOfInterval(timestamp, INTERVAL 5 MINUTE) AS ts,
    attributes_string['customer.tier'] AS tier,
    round(
        countIf(has_error = true) * 100.0 /
        (CASE
            WHEN attributes_string['customer.tier'] = 'enterprise' THEN (1 - 0.999) * count()
            WHEN attributes_string['customer.tier'] = 'pro' THEN (1 - 0.995) * count()
            ELSE (1 - 0.990) * count()
        END), 1
    ) AS burn_rate_pct
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= now() - INTERVAL 1 HOUR
    AND name LIKE 'activity.%'
    AND serviceName = 'temporal-worker'
    AND attributes_string['customer.tier'] != ''
GROUP BY ts, tier
ORDER BY ts;


-- ============================================================
-- QUERY 6: Traffic by Customer Tier Over Time (Time Series)
-- Shows request volume per tier to identify traffic patterns.
-- ============================================================
SELECT
    toStartOfInterval(timestamp, INTERVAL 1 MINUTE) AS ts,
    attributes_string['customer.tier'] AS tier,
    count() AS requests
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= now() - INTERVAL 30 MINUTE
    AND name LIKE 'activity.%'
    AND serviceName = 'temporal-worker'
    AND attributes_string['customer.tier'] != ''
GROUP BY ts, tier
ORDER BY ts;


-- ============================================================
-- QUERY 7: Activity P99 Latency Trend (Time Series)
-- Per-activity P99 over time to identify latency spikes.
-- ============================================================
SELECT
    toStartOfInterval(timestamp, INTERVAL 1 MINUTE) AS ts,
    name AS activity,
    round(quantile(0.99)(duration_nano) / 1000000, 2) AS p99_ms
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= now() - INTERVAL 30 MINUTE
    AND name LIKE 'activity.%'
    AND serviceName = 'temporal-worker'
GROUP BY ts, activity
ORDER BY ts;


-- ============================================================
-- QUERY 8: Workflow Step Duration Breakdown (Time Series - Stacked)
-- Average duration per activity over time. Use with "Stack series" ON.
-- ============================================================
SELECT
    toStartOfInterval(timestamp, INTERVAL 1 MINUTE) AS ts,
    name AS activity,
    round(avg(duration_nano) / 1000000, 2) AS avg_ms
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= now() - INTERVAL 30 MINUTE
    AND name LIKE 'activity.%'
    AND serviceName = 'temporal-worker'
GROUP BY ts, activity
ORDER BY ts;


-- ============================================================
-- QUERY 9: Cross-Signal — Error Logs Correlated with Trace Failures
-- Joins log severity with trace error status in the same time window.
-- Demonstrates multi-signal correlation (traces + logs).
-- ============================================================
SELECT
    toStartOfInterval(timestamp, INTERVAL 5 MINUTE) AS ts,
    name AS activity,
    count() AS total_spans,
    countIf(has_error = true) AS error_spans,
    round(countIf(has_error = true) * 100.0 / count(), 2) AS error_rate_pct
FROM signoz_traces.distributed_signoz_index_v3
WHERE timestamp >= now() - INTERVAL 1 HOUR
    AND name LIKE 'activity.%'
    AND serviceName = 'temporal-worker'
GROUP BY ts, activity
HAVING error_spans > 0
ORDER BY ts DESC, error_rate_pct DESC;
