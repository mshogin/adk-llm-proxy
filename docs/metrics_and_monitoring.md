# Metrics and Monitoring

Comprehensive documentation for the ADK LLM Proxy metrics system and monitoring capabilities.

## Overview

The metrics system provides complete observability into agent execution, LLM usage, costs, and performance. It includes:

- **Metrics Collection**: Per-agent and per-session metrics
- **Distributed Tracing**: Request flow tracking with trace_id
- **Performance Profiling**: CPU, memory, and latency tracking
- **Cost Reporting**: Budget tracking and per-agent cost breakdown
- **Alerting**: Real-time alerts for budget overruns and SLA violations
- **Structured Logging**: ELK-compatible JSON logs
- **Prometheus Export**: Standard metrics format for monitoring

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Metrics Collection                       │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │   Collector  │  │ LLM Metrics  │  │ Context Metrics │   │
│  └──────────────┘  └──────────────┘  └─────────────────┘   │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │   Tracer     │  │   Profiler   │  │  Cost Reporter  │   │
│  └──────────────┘  └──────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      Export & Alerting                       │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │  Prometheus  │  │   Logging    │  │ Alert Manager   │   │
│  │   Exporter   │  │  (ELK JSON)  │  │                 │   │
│  └──────────────┘  └──────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Metrics Collector

### Purpose
Aggregates metrics for all agents in a session, tracking execution statistics, LLM usage, and costs.

### Data Structure

```go
type SessionMetrics struct {
    SessionID      string
    TraceID        string
    TotalDurationMS int64
    TotalCost       float64
    TotalTokens     int
    AgentMetrics    map[string]*AgentMetrics
}

type AgentMetrics struct {
    AgentID         string
    ExecutionCount  int
    TotalDurationMS int64
    AvgDurationMS   int64
    LLMCalls        int
    TotalTokens     int
    TotalCost       float64
    SuccessCount    int
    FailureCount    int
    LastStatus      string
    LastError       string
}
```

### Usage

```go
collector := metrics.NewCollector("session123", "trace456")

// Record agent execution
run := models.AgentRun{
    AgentID:     "intent_detection",
    StartTime:   time.Now().Add(-100 * time.Millisecond),
    EndTime:     time.Now(),
    Status:      "success",
    OutputSize:  1024,
}
collector.RecordAgentExecution(run)

// Record LLM usage
collector.RecordLLMUsage("intent_detection", 150, 0.0003)

// Get metrics
metrics := collector.GetSessionMetrics()
```

## LLM Metrics Collector

### Purpose
Tracks LLM-specific metrics including cache performance, model selection, and provider statistics.

### Data Structure

```go
type LLMMetrics struct {
    TotalCalls           int
    TotalPromptTokens    int
    TotalCompletionTokens int
    TotalCost            float64
    CacheHits            int
    CacheMisses          int
    CacheHitRate         float64
    CallsByProvider      map[string]*LLMCallMetrics
    ModelDecisions       []ModelSelectionDecision
}

type ModelSelectionDecision struct {
    Timestamp         time.Time
    AgentID           string
    TaskType          string
    ContextSize       int
    SelectedModel     string
    Reason            string
    AlternativeModels []string
    EstimatedCost     float64
}
```

### Cache Metrics

```go
collector := metrics.NewLLMMetricsCollector("session123", "trace456")

// Record cache hit
collector.RecordCacheHit()

// Record cache miss
collector.RecordCacheMiss()

// Record LLM call
collector.RecordLLMCall("openai", "gpt-4o-mini", 100, 50, 0.0003, 150, nil)

// Get cache metrics
cacheMetrics := collector.GetCacheMetrics()
fmt.Printf("Cache hit rate: %.2f%%\n", cacheMetrics.HitRate)
```

## Context Metrics Collector

### Purpose
Tracks context size growth and artifact proliferation to detect bloat.

### Data Structure

```go
type ContextGrowth struct {
    InitialSize      int
    CurrentSize      int
    GrowthRate       float64
    SnapshotCount    int
    AvgDiffSize      int
    MaxContextSize   int
    TotalDiffSize    int
}

type ArtifactGrowth struct {
    InitialCount        int
    CurrentCount        int
    GrowthRate          float64
    AvgArtifactCount    int
    MaxArtifactCount    int
    TotalArtifactCount  int
}
```

### Usage

```go
collector := metrics.NewContextMetricsCollector("session123", "trace456")

// Record context snapshot
collector.RecordContextSnapshot("intent_detection", agentContext, time.Now().Unix())

// Get context growth metrics
growth := collector.GetContextGrowth()
artifactGrowth := collector.GetArtifactGrowth()
```

## Distributed Tracing

### Purpose
Provides complete traceability of requests through the agent pipeline with span hierarchy.

### Data Structure

```go
type Trace struct {
    TraceID    string
    SessionID  string
    RootSpan   *Span
    Spans      []*Span
    SpanCount  int
    Duration   time.Duration
}

type Span struct {
    SpanID     string
    TraceID    string
    ParentID   string
    Name       string
    StartTime  time.Time
    EndTime    time.Time
    Duration   time.Duration
    Status     SpanStatus
    Attributes map[string]interface{}
    Events     []SpanEvent
    Tags       map[string]string
}
```

### Usage

```go
tracer := metrics.NewTracer("session123", "trace456")

// Start root span
rootSpan := tracer.StartSpan("agent.pipeline", "")

// Start child span
childSpan := tracer.StartSpan("agent.intent_detection", rootSpan.SpanID)

// Add attributes
tracer.AddSpanAttribute(childSpan.SpanID, "agent_id", "intent_detection")

// Add events
tracer.AddSpanEvent(childSpan.SpanID, "intent_found", map[string]interface{}{
    "intent_type": "query_commits",
    "confidence":  0.95,
})

// End spans
tracer.EndSpan(childSpan.SpanID, metrics.SpanStatusOK)
tracer.EndSpan(rootSpan.SpanID, metrics.SpanStatusOK)

// Get complete trace
trace := tracer.GetTrace()
```

### Context Propagation

```go
import "context"

// Add trace and span IDs to context
ctx := metrics.ContextWithTrace(context.Background(), "trace123", "span456")

// Extract from context
traceID := metrics.TraceIDFromContext(ctx)
spanID := metrics.SpanIDFromContext(ctx)
```

## Performance Profiling

### Purpose
Tracks CPU time, memory usage, and latency per agent for identifying bottlenecks.

### Data Structure

```go
type AgentProfile struct {
    AgentID             string
    ExecutionCount      int
    TotalDurationMS     int64
    MinDurationMS       int64
    MaxDurationMS       int64
    AvgDurationMS       int64
    TotalMemoryAllocBytes int64
    AvgMemoryAllocBytes   int64
    MaxMemoryAllocBytes   int64
    AvgGoroutineCount   int
    MaxGoroutineCount   int
    Operations          map[string]*OperationProfile
    ErrorCount          int
    LastError           string
}
```

### Usage

```go
profiler := metrics.NewPerformanceProfiler("session123", "trace456")

// Auto-profiling with deferred completion
done := profiler.StartProfile("intent_detection")
defer done(nil)

// ... agent execution ...

// Get bottlenecks (agents > 500ms avg)
bottlenecks := profiler.GetBottlenecks(500)

// Get memory hogs (agents > 100MB avg)
memoryHogs := profiler.GetMemoryHogs(100 * 1024 * 1024)

// Get performance report
report := profiler.GetPerformanceReport()
```

### Operation-Level Profiling

```go
// Profile specific operations within an agent
profiler.RecordOperation("intent_detection", "regex_matching", 10)
profiler.RecordOperation("intent_detection", "entity_extraction", 20)

profile := profiler.GetAgentProfile("intent_detection")
for name, op := range profile.Operations {
    fmt.Printf("%s: %dms avg\n", name, op.AvgDurationMS)
}
```

## Cost Reporting

### Purpose
Tracks costs per session and per agent with budget alerts.

### Data Structure

```go
type CostReport struct {
    SessionID       string
    TraceID         string
    TotalCost       float64
    BudgetLimit     float64
    BudgetUsage     float64    // percentage (0-100)
    BudgetRemaining float64
    IsOverBudget    bool
    AgentCosts      []*AgentCostData
    AgentCount      int
    TopCostAgents   []*AgentCostData // Top 5 most expensive
    StartTime       time.Time
    EndTime         time.Time
    Duration        time.Duration
}

type AgentCostData struct {
    AgentID          string
    TotalCost        float64
    LLMCalls         int
    TotalTokens      int
    PromptTokens     int
    CompletionTokens int
    AvgCostPerCall   float64
    BudgetUsage      float64 // percentage
}
```

### Usage

```go
reporter := metrics.NewCostReporter("session123", "trace456", 10.0)

// Record agent cost
reporter.RecordAgentCost("intent_detection", 0.0003, 100, 50)

// Get cost report
report := reporter.GetCostReport()
fmt.Printf("Total cost: $%.4f / $%.4f (%.1f%%)\n",
    report.TotalCost, report.BudgetLimit, report.BudgetUsage)

// Check budget alerts
alerts := reporter.CheckBudgetAlerts()
for _, alert := range alerts {
    fmt.Printf("%s: %s\n", alert.Severity, alert.Message)
}

// Format report
formatted := metrics.FormatCostReport(report)
fmt.Println(formatted)
```

## Alerting

### Purpose
Real-time alerts for budget overruns, SLA violations, error rates, and memory exhaustion.

### Alert Types

| Alert Type | Thresholds | Severity Levels |
|-----------|-----------|----------------|
| Budget Overrun | 50%, 80%, 100% | Info, Warning, Critical |
| SLA Violation (Agent) | 1s, 5s | Warning, Critical |
| SLA Violation (Session) | 10s, 30s | Warning, Critical |
| Error Rate | 10%, 25% | Warning, Critical |
| Memory Exhaustion | 100MB, 500MB | Warning, Critical |

### Configuration

```go
config := metrics.AlertConfig{
    BudgetInfoThreshold:              50.0,  // 50%
    BudgetWarningThreshold:           80.0,  // 80%
    BudgetCriticalThreshold:          100.0, // 100%
    AgentDurationWarningThreshold:    1000,  // 1s
    AgentDurationCriticalThreshold:   5000,  // 5s
    SessionDurationWarningThreshold:  10000, // 10s
    SessionDurationCriticalThreshold: 30000, // 30s
    ErrorRateWarningThreshold:        10.0,  // 10%
    ErrorRateCriticalThreshold:       25.0,  // 25%
    MemoryWarningThresholdMB:         100,   // 100MB
    MemoryCriticalThresholdMB:        500,   // 500MB
    DeduplicationWindowSeconds:       300,   // 5min
}

alertManager := metrics.NewAlertManager(
    "session123",
    "trace456",
    config,
    costReporter,
    profiler,
)
```

### Usage

```go
// Check for alerts
alerts := alertManager.CheckAlerts()

for _, alert := range alerts {
    fmt.Printf("[%s] %s: %s\n", alert.Severity, alert.Type, alert.Message)

    // Access alert details
    for key, value := range alert.Details {
        fmt.Printf("  %s: %v\n", key, value)
    }
}

// Get only active alerts
activeAlerts := alertManager.GetActiveAlerts()

// Get alert history
history := alertManager.GetAlertHistory()

// Resolve an alert
alertManager.ResolveAlert(alert.ID)

// Clear resolved alerts
alertManager.ClearResolvedAlerts()
```

## Structured Logging

### Purpose
ELK-compatible JSON logs for easy parsing and analysis.

### Log Format

```json
{
  "@timestamp": "2025-01-16T10:30:45.123456789Z",
  "level": "INFO",
  "message": "Agent execution completed",
  "logger": "adk_llm_proxy",
  "source_file": "/path/to/file.go",
  "source_line": 42,
  "session_id": "session123",
  "trace_id": "trace456",
  "span_id": "span789",
  "agent_id": "intent_detection",
  "fields": {
    "duration_ms": 150,
    "status": "success"
  }
}
```

### Usage

```go
import "github.com/mshogin/agents/internal/infrastructure/logging"

// Create logger
logger := logging.NewStructuredLogger(os.Stdout, logging.InfoLevel)

// Log with fields
logger.Info("Agent execution started", map[string]interface{}{
    "session_id": "session123",
    "trace_id":   "trace456",
    "agent_id":   "intent_detection",
})

// Log error with stack trace
err := errors.New("validation failed")
logger.Error("Agent execution failed", err, map[string]interface{}{
    "agent_id": "validation",
})

// Create context logger with pre-set fields
ctx := logger.NewContext(map[string]interface{}{
    "session_id": "session123",
    "trace_id":   "trace456",
})

ctx.Info("Processing request")
ctx.Warn("High token usage", map[string]interface{}{
    "tokens": 5000,
})
```

### Global Logger

```go
// Use global logger
logging.Info("Application started")
logging.Warn("High memory usage")
logging.Error("Failed to connect", err)

// Set custom global logger
customLogger := logging.NewStructuredLogger(file, logging.DebugLevel)
logging.SetDefaultLogger(customLogger)
```

## Prometheus Export

### Purpose
Exposes metrics in Prometheus text format for monitoring.

### Metric Types

- **Counter**: Monotonically increasing values (e.g., total requests)
- **Gauge**: Values that can go up and down (e.g., memory usage)
- **Histogram**: Distribution of values (e.g., request duration)
- **Summary**: Similar to histogram with percentiles

### Usage

```go
exporter := metrics.NewPrometheusExporter("adk_llm_proxy")

// Create adapter for your metrics
adapter := metrics.NewSimpleCollectorAdapter("agent_metrics")

// Add metrics
adapter.AddMetric(metrics.CreateAgentMetric(
    "intent_detection",
    "agent_duration_ms",
    150.5,
    "Agent execution duration in milliseconds",
))

adapter.AddMetric(metrics.CreateLLMMetric(
    "openai",
    "gpt-4o-mini",
    "llm_tokens_total",
    1500,
    "Total tokens used",
))

// Register adapter
exporter.RegisterCollector(adapter)

// Export metrics
output := exporter.Export()
fmt.Println(output)
```

### Example Output

```
# Generated at 2025-01-16T10:30:45Z
# Metrics from ADK LLM Proxy

# HELP adk_llm_proxy_agent_duration_ms Agent execution duration in milliseconds
# TYPE adk_llm_proxy_agent_duration_ms gauge
adk_llm_proxy_agent_duration_ms{agent_id="intent_detection",status="success"} 150.5

# HELP adk_llm_proxy_llm_tokens_total Total tokens used
# TYPE adk_llm_proxy_llm_tokens_total gauge
adk_llm_proxy_llm_tokens_total{provider="openai",model="gpt-4o-mini"} 1500
```

### HTTP Endpoint

```go
import "net/http"

func metricsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain; version=0.0.4")
    w.Write([]byte(exporter.Export()))
}

http.HandleFunc("/metrics", metricsHandler)
http.ListenAndServe(":9090", nil)
```

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'adk_llm_proxy'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
    metrics_path: /metrics
```

## Best Practices

### 1. Session Lifecycle

```go
// Initialize at session start
collector := metrics.NewCollector(sessionID, traceID)
tracer := metrics.NewTracer(sessionID, traceID)
profiler := metrics.NewPerformanceProfiler(sessionID, traceID)
costReporter := metrics.NewCostReporter(sessionID, traceID, budgetLimit)
alertManager := metrics.NewDefaultAlertManager(sessionID, traceID, costReporter, profiler)

// Record throughout session
// ...

// Finalize at session end
costReporter.Finalize()

// Generate reports
sessionMetrics := collector.GetSessionMetrics()
trace := tracer.GetTrace()
perfReport := profiler.GetPerformanceReport()
costReport := costReporter.GetCostReport()
alerts := alertManager.GetActiveAlerts()
```

### 2. Agent Execution Pattern

```go
// Start span
span := tracer.StartSpan("agent.intent_detection", parentSpanID)

// Start profiling
done := profiler.StartProfile("intent_detection")

// Execute agent
result, err := executeAgent()

// End profiling
done(err)

// Record metrics
collector.RecordAgentExecution(agentRun)
collector.RecordLLMUsage("intent_detection", tokens, cost)

// End span
tracer.EndSpan(span.SpanID, spanStatus)

// Record cost
costReporter.RecordAgentCost("intent_detection", cost, promptTokens, completionTokens)

// Check alerts
alerts := alertManager.CheckAlerts()
```

### 3. Error Handling

```go
if err != nil {
    // Log error
    logger.Error("Agent execution failed", err, map[string]interface{}{
        "agent_id":   "intent_detection",
        "session_id": sessionID,
        "trace_id":   traceID,
    })

    // Record error in span
    tracer.AddSpanAttribute(span.SpanID, "error", err.Error())
    tracer.EndSpan(span.SpanID, metrics.SpanStatusError)

    // Record in profiling
    done(err)
}
```

### 4. Performance Optimization

- Use buffer pools for frequent allocations
- Return deep copies to prevent race conditions
- Use read locks when only reading data
- Batch metric updates when possible
- Set appropriate alert deduplication windows

### 5. Monitoring Integration

```go
// Export to Prometheus
go func() {
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        output := exporter.Export()
        // Write to metrics endpoint
    }
}()

// Send structured logs to ELK
logger := logging.NewStructuredLogger(elkWriter, logging.InfoLevel)

// Check alerts periodically
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        alerts := alertManager.CheckAlerts()
        for _, alert := range alerts {
            // Send to alerting system (PagerDuty, Slack, etc.)
        }
    }
}()
```

## Troubleshooting

### High Memory Usage

1. Check agent profiles: `profiler.GetMemoryHogs(100 * 1024 * 1024)`
2. Analyze context growth: `contextCollector.GetContextGrowth()`
3. Review artifact proliferation: `contextCollector.GetArtifactGrowth()`

### Performance Bottlenecks

1. Identify slow agents: `profiler.GetBottlenecks(500)`
2. Check operation breakdown: `profile.Operations`
3. Review trace spans for duration anomalies

### Budget Overruns

1. Check cost report: `costReporter.GetCostReport()`
2. Review top cost agents: `report.TopCostAgents`
3. Analyze LLM usage: `llmCollector.GetCallsByProvider()`

### Alert Fatigue

1. Adjust thresholds in AlertConfig
2. Increase deduplication window
3. Use alert severity appropriately (Info vs Warning vs Critical)

## References

- [Prometheus Text Format](https://prometheus.io/docs/instrumenting/exposition_formats/)
- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/)
- [Elasticsearch Common Schema](https://www.elastic.co/guide/en/ecs/current/index.html)
