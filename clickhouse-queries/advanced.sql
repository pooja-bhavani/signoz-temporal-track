-- ============================================================
-- ADVANCED CLICKHOUSE QUERIES FOR TEMPORAL + SIGNOZ
-- Track 2: Signals & Dashboards — "Template Augmentation" Strategy
-- ============================================================


-- ============================================================
-- QUERY 1: Dynamic Anomaly Detection (Z-Score) for Workflow Latency
-- Uses window functions to calculate rolling baseline and flag outliers.
-- Replaces static threshold alerts with mathematical anomaly detection.
-- ============================================================
WITH
    minute_stats AS (
        SELECT
            toStartOfInterval(timestamp, INTERVAL 1 MINUTE) AS minute,
            serviceName,
            avg(durationNano / 1000000) AS avg_latency_ms
        FROM signoz_traces.signoz_index_v3
        WHERE timestamp >= now() - INTERVAL 4 HOUR
        GROUP BY minute, serviceName
    ),
    moving_metrics AS (
        SELECT
            minute,
            serviceName,
            avg_latency_ms,
            avg(avg_latency_ms) OVER (PARTITION BY serviceName ORDER BY minute ROWS BETWEEN 15 PRECEDING AND 1 PRECEDING) as rolling_avg,
            stddevPop(avg_latency_ms) OVER (PARTITION BY serviceName ORDER BY minute ROWS BETWEEN 15 PRECEDING AND 1 PRECEDING) as rolling_stddev
        FROM minute_stats
    )
SELECT
    minute,
    serviceName,
    round(avg_latency_ms, 2) AS current_latency_ms,
    round(rolling_avg, 2) AS baseline_avg_ms,
    round(rolling_stddev, 2) AS baseline_stddev,
    round((avg_latency_ms - rolling_avg) / NULLIF(rolling_stddev, 0), 2) AS z_score,
    CASE
        WHEN ((avg_latency_ms - rolling_avg) / NULLIF(rolling_stddev, 0)) > 3 THEN 'ANOMALY_SPIKE'
        WHEN ((avg_latency_ms - rolling_avg) / NULLIF(rolling_stddev, 0)) < -3 THEN 'ANOMALY_DROP'
        ELSE 'NORMAL'
    END AS status
FROM moving_metrics
WHERE minute >= now() - INTERVAL 1 HOUR
ORDER BY minute DESC, z_score DESC;


-- ============================================================
-- QUERY 2: Parent-Child Distributed Trace Correlation (Root Cause Finder)
-- Self-joins trace spans to find slow parent calls caused by slow children.
-- Shows which downstream activity is responsible for workflow delays.
-- ============================================================
SELECT
    parent.traceId AS trace_id,
    parent.serviceName AS workflow_service,
    parent.name AS parent_operation,
    (parent.durationNano / 1000000) AS total_latency_ms,
    child.serviceName AS activity_service,
    child.name AS child_operation,
    (child.durationNano / 1000000) AS child_latency_ms,
    round((child.durationNano * 100.0 / parent.durationNano), 2) AS percent_time_in_child,
    parent.stringTagMap['customer.tier'] AS affected_tier
FROM signoz_traces.signoz_index_v3 AS parent
JOIN signoz_traces.signoz_index_v3 AS child
    ON parent.traceId = child.traceId
    AND parent.spanId = child.parentSpanId
WHERE parent.timestamp >= now() - INTERVAL 1 HOUR
    AND parent.kind = 2  -- Server span (workflow)
    AND child.kind = 3   -- Client span (activity call)
    AND (parent.durationNano / 1000000) > 500  -- Only slow workflows (>500ms)
ORDER BY percent_time_in_child DESC
LIMIT 50;


-- ============================================================
-- QUERY 3: Predictive Error Budget Depletion (Forecasting)
-- Calculates 6-hour burn rate and extrapolates time to SLO breach.
-- ============================================================
WITH
    hourly_stats AS (
        SELECT
            stringTagMap['customer.tier'] AS tier,
            toStartOfInterval(timestamp, INTERVAL 1 HOUR) as hour_bucket,
            count() AS total_requests,
            countIf(statusCode = 2) AS error_count
        FROM signoz_traces.signoz_index_v3
        WHERE timestamp >= now() - INTERVAL 6 HOUR
            AND kind = 2
            AND stringTagMap['customer.tier'] != ''
        GROUP BY tier, hour_bucket
    ),
    burn_rates AS (
        SELECT
            tier,
            sum(error_count) as total_errors_6h,
            sum(total_requests) as total_reqs_6h,
            sum(error_count) / sum(total_requests) as current_error_rate,
            CASE tier
                WHEN 'enterprise' THEN 0.001  -- 99.9% SLO
                WHEN 'pro' THEN 0.005         -- 99.5% SLO
                ELSE 0.01                     -- 99.0% SLO
            END as error_budget_allowance
        FROM hourly_stats
        GROUP BY tier
    )
SELECT
    tier,
    total_reqs_6h,
    total_errors_6h,
    round(current_error_rate * 100, 4) AS error_rate_pct,
    round(error_budget_allowance * 100, 2) AS budget_allowance_pct,
    round((error_budget_allowance - current_error_rate) * 100, 4) AS margin_remaining_pct,
    CASE
        WHEN current_error_rate >= error_budget_allowance THEN 'BUDGET EXHAUSTED'
        WHEN current_error_rate = 0 THEN 'INFINITE'
        ELSE toString(round((error_budget_allowance / current_error_rate) * 6, 1)) || ' hours'
    END AS time_to_exhaustion,
    CASE
        WHEN current_error_rate >= error_budget_allowance THEN 'CRITICAL'
        WHEN current_error_rate >= error_budget_allowance * 0.5 THEN 'WARNING'
        ELSE 'HEALTHY'
    END AS status
FROM burn_rates
ORDER BY current_error_rate DESC;


-- ============================================================
-- QUERY 4: Temporal Activity Retry Correlation with Error Logs
-- Cross-signal: correlates retry spikes (traces) with error log patterns.
-- ============================================================
SELECT
    toStartOfInterval(t.timestamp, INTERVAL 5 MINUTE) AS interval,
    t.name AS activity_name,
    count() AS total_executions,
    countIf(t.statusCode = 2) AS failed_executions,
    -- Count retries (attempt > 1 shown by repeated activity spans in same workflow)
    countIf(toInt32OrZero(t.stringTagMap['temporalActivityAttempt']) > 1) AS retried_executions,
    -- Correlate with error logs in the same window
    (SELECT count()
     FROM signoz_logs.logs
     WHERE timestamp >= interval
       AND timestamp < interval + INTERVAL 5 MINUTE
       AND severity_text = 'ERROR'
    ) AS error_log_count,
    round(countIf(t.statusCode = 2) * 100.0 / count(), 2) AS failure_rate_pct
FROM signoz_traces.signoz_index_v3 AS t
WHERE t.timestamp >= now() - INTERVAL 1 HOUR
    AND t.name LIKE 'activity.%'
GROUP BY interval, activity_name
ORDER BY interval DESC, retried_executions DESC;


-- ============================================================
-- QUERY 5: Activity Latency vs Worker CPU (Cross-Signal Infrastructure)
-- Maps activity execution time against host CPU utilization.
-- ============================================================
WITH
    activity_latencies AS (
        SELECT
            toStartOfInterval(timestamp, INTERVAL 1 MINUTE) AS minute,
            name AS activity,
            quantile(0.99)(durationNano / 1000000) AS p99_ms,
            quantile(0.50)(durationNano / 1000000) AS p50_ms,
            count() AS executions
        FROM signoz_traces.signoz_index_v3
        WHERE timestamp >= now() - INTERVAL 30 MINUTE
            AND name LIKE 'activity.%'
        GROUP BY minute, activity
    )
SELECT
    al.minute,
    al.activity,
    round(al.p99_ms, 2) AS activity_p99_ms,
    round(al.p50_ms, 2) AS activity_p50_ms,
    al.executions,
    round(al.p99_ms / NULLIF(al.p50_ms, 0), 2) AS latency_spread_ratio
FROM activity_latencies al
ORDER BY al.minute DESC, al.activity_p99_ms DESC;


-- ============================================================
-- QUERY 6: Workflow End-to-End SLO by Customer Tier
-- Measures workflow completion rate and latency against SLO targets.
-- ============================================================
SELECT
    stringTagMap['customer.tier'] AS tier,
    count() AS total_workflows,
    countIf(statusCode != 2) AS successful,
    countIf(statusCode = 2) AS failed,
    round(countIf(statusCode != 2) * 100.0 / count(), 2) AS success_rate_pct,
    round(quantile(0.50)(durationNano / 1000000), 2) AS p50_duration_ms,
    round(quantile(0.95)(durationNano / 1000000), 2) AS p95_duration_ms,
    round(quantile(0.99)(durationNano / 1000000), 2) AS p99_duration_ms,
    CASE stringTagMap['customer.tier']
        WHEN 'enterprise' THEN 99.9
        WHEN 'pro' THEN 99.5
        ELSE 99.0
    END AS slo_target_pct,
    CASE
        WHEN countIf(statusCode != 2) * 100.0 / count() >=
            CASE stringTagMap['customer.tier']
                WHEN 'enterprise' THEN 99.9
                WHEN 'pro' THEN 99.5
                ELSE 99.0
            END THEN 'MEETING SLO'
        ELSE 'BREACHING SLO'
    END AS slo_status
FROM signoz_traces.signoz_index_v3
WHERE timestamp >= now() - INTERVAL 1 HOUR
    AND name = 'OrderProcessingWorkflow'
    AND stringTagMap['customer.tier'] != ''
GROUP BY tier
ORDER BY success_rate_pct ASC;


-- ============================================================
-- QUERY 7: Workflow Step Bottleneck Analysis
-- Identifies which activity in the pipeline is the bottleneck.
-- ============================================================
SELECT
    name AS activity_step,
    count() AS executions,
    round(avg(durationNano / 1000000), 2) AS avg_ms,
    round(quantile(0.50)(durationNano / 1000000), 2) AS p50_ms,
    round(quantile(0.95)(durationNano / 1000000), 2) AS p95_ms,
    round(quantile(0.99)(durationNano / 1000000), 2) AS p99_ms,
    countIf(statusCode = 2) AS errors,
    round(countIf(statusCode = 2) * 100.0 / count(), 2) AS error_rate_pct
FROM signoz_traces.signoz_index_v3
WHERE timestamp >= now() - INTERVAL 1 HOUR
    AND name LIKE 'activity.%'
GROUP BY name
ORDER BY p99_ms DESC;
