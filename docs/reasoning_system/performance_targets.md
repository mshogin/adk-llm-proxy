# Performance Targets and SLA Criteria

## Overview

This document defines performance targets, SLA (Service Level Agreement) criteria, and measurement methodologies for the ADK LLM Proxy reasoning system. Use these targets for capacity planning, monitoring, and production readiness assessments.

---

## Performance Targets by Component

### Agent Execution

| Agent Type | Target Latency (p50) | Target Latency (p99) | Notes |
|------------|---------------------|---------------------|-------|
| **No LLM (rules-only)** | <100ms | <500ms | Pure computation, no external calls |
| **Simple LLM (classification)** | <2s | <5s | Single LLM call, <500 tokens |
| **Medium LLM (synthesis)** | <5s | <10s | Single LLM call, <2K tokens |
| **Complex LLM (reasoning)** | <10s | <20s | Multiple LLM calls or >5K tokens |
| **External API (retrieval)** | <3s | <8s | Single API call (GitLab, YouTrack) |
| **Multi-step (parallel agents)** | <5s | <15s | Parallel execution of 3-5 agents |

**Measurement:**
```go
// Track in audit.agent_runs
duration := run.EndTime.Sub(run.StartTime)
p50 := percentile(durations, 0.50)
p99 := percentile(durations, 0.99)
```

---

### Full Pipeline Execution

| Pipeline Type | Target Latency (p50) | Target Latency (p99) | Agent Count |
|---------------|---------------------|---------------------|-------------|
| **Sequential (no LLM)** | <500ms | <2s | 4-5 agents |
| **Sequential (with LLM)** | <5s | <15s | 4-5 agents |
| **Parallel (no LLM)** | <1s | <3s | 7-10 agents |
| **Parallel (with LLM)** | <8s | <25s | 7-10 agents |
| **Conditional (best case)** | <2s | <5s | 3-4 agents executed |
| **Conditional (worst case)** | <10s | <30s | 8-10 agents executed |

**Factors Affecting Latency:**
- LLM provider response time (2-15s typical)
- External API latency (GitLab, YouTrack: 1-5s)
- Context size (larger contexts = slower serialization)
- Agent count (more agents = longer pipeline)
- Cache hit rate (cache hits = instant response)

**Measurement:**
```bash
# Track total session duration
curl http://localhost:8001/metrics | grep session_duration_seconds

# Sample metrics output:
# session_duration_seconds{quantile="0.5"} 4.2
# session_duration_seconds{quantile="0.99"} 18.7
```

---

### LLM Orchestrator

| Metric | Target | Notes |
|--------|--------|-------|
| **Model selection latency** | <10ms | Decision logic overhead |
| **Cache lookup latency** | <5ms | In-memory cache |
| **Budget check latency** | <1ms | Simple arithmetic |
| **Total orchestration overhead** | <20ms | Sum of all overhead |

**Measurement:**
```go
start := time.Now()
model, provider, err := orchestrator.SelectModel(ctx, req)
latency := time.Since(start)
// Target: <10ms
```

---

### Cache Performance

| Metric | Target | Notes |
|--------|--------|-------|
| **Cache hit rate (classification)** | >60% | Highly cacheable (deterministic) |
| **Cache hit rate (synthesis)** | >40% | Moderately cacheable |
| **Cache hit rate (inference)** | >20% | Context-dependent |
| **Cache lookup latency** | <5ms | In-memory lookup |
| **Cache miss penalty** | 0ms | No penalty (proceed to LLM) |

**Cost Savings:**
- 60% cache hit rate = 60% cost reduction
- Classification caching saves ~$0.0001/query
- Synthesis caching saves ~$0.001/query

**Measurement:**
```go
entries, hits, size := orchestrator.GetCacheStats()
hitRate := float64(hits) / float64(entries + hits) * 100
fmt.Printf("Cache hit rate: %.1f%%\n", hitRate)
```

---

## Cost Targets

### Per-Session Cost

| Pipeline Type | Target Cost (p50) | Target Cost (p99) | Budget Limit |
|---------------|------------------|------------------|--------------|
| **Sequential basic** | $0.02 | $0.05 | $0.50 |
| **Parallel retrieval** | $0.10 | $0.20 | $1.00 |
| **Conditional (adaptive)** | $0.05 | $0.15 | $0.75 |
| **Production full** | $0.15 | $0.50 | $2.00 |

**Cost Breakdown:**
- Intent detection: ~$0.0001 (deepseek-chat)
- Reasoning structure: ~$0.0002 (deepseek-chat)
- Retrieval planning: ~$0.0005 (gpt-4o-mini)
- Inference (simple): ~$0.001 (gpt-4o-mini)
- Inference (complex): ~$0.01 (gpt-4o)
- Summarization: ~$0.0005 (gpt-4o-mini)

**Budget Adherence:**
- 100% adherence (hard limits enforced)
- Warning at 75-80% of budget
- Emergency degradation at >100%

**Measurement:**
```go
sessionUsed, sessionLimit, agentBudgets := orchestrator.GetBudgetStatus()
adherence := 1.0 - (sessionUsed / sessionLimit)
// Target: Never exceed sessionLimit
```

---

### Per-Agent Cost

| Agent | Target Cost | Budget Limit | Model |
|-------|------------|--------------|-------|
| **Intent Detection** | $0.0001 | $0.05 | deepseek-chat |
| **Reasoning Structure** | $0.0002 | $0.05 | deepseek-chat |
| **Retrieval Planner** | $0.0005 | $0.10 | gpt-4o-mini |
| **Context Synthesizer** | $0.002 | $0.10 | gpt-4o-mini |
| **Inference (simple)** | $0.001 | $0.20 | gpt-4o-mini |
| **Inference (complex)** | $0.01 | $0.50 | gpt-4o |
| **Validation** | $0.0003 | $0.05 | deepseek-chat |
| **Summarization** | $0.0005 | $0.05 | gpt-4o-mini |

---

## Resource Utilization

### CPU

| Load Type | Target CPU Usage | Notes |
|-----------|-----------------|-------|
| **Idle** | <5% | No active requests |
| **Light (10 req/s)** | <30% | Typical load |
| **Medium (50 req/s)** | <60% | Moderate load |
| **Heavy (100 req/s)** | <85% | Peak load |

**Scaling Thresholds:**
- Scale up at: 70% sustained CPU usage
- Scale down at: <30% sustained CPU usage

**Measurement:**
```bash
# Check CPU usage
top -p $(pgrep proxy)

# Prometheus metric
process_cpu_seconds_total
```

---

### Memory

| Component | Target Memory | Max Memory | Notes |
|-----------|--------------|------------|-------|
| **Idle proxy** | <100MB | 200MB | No active sessions |
| **Per-session context** | <5MB | 10MB | AgentContext size |
| **Cache** | <100MB | 500MB | Response cache |
| **Total under load** | <300MB | 1GB | 50 concurrent sessions |

**Memory Optimization:**
- Enable context compression
- Limit cache size (max_entries, max_size_mb)
- Externalize large artifacts

**Measurement:**
```bash
# Check memory usage
ps aux | grep proxy

# Prometheus metric
process_resident_memory_bytes
```

---

### Network

| Metric | Target | Notes |
|--------|--------|-------|
| **Throughput** | >10K req/s | Non-streaming mode |
| **Concurrent connections** | >1000 | HTTP keep-alive |
| **Bandwidth (outbound)** | <10MB/s | Streaming responses |

**Measurement:**
```bash
# Check network throughput
netstat -s

# Load test with hey
hey -n 10000 -c 100 http://localhost:8001/v1/chat/completions
```

---

## Availability Targets

### Uptime SLA

| Tier | Uptime Target | Max Downtime/Month | Max Downtime/Year |
|------|---------------|-------------------|------------------|
| **Bronze** | 95% | 36 hours | 18 days |
| **Silver** | 99% | 7.2 hours | 3.6 days |
| **Gold** | 99.9% | 43 minutes | 8.7 hours |
| **Platinum** | 99.99% | 4.3 minutes | 52 minutes |

**Downtime Definition:**
- API returns 5xx errors
- Response time >30s (p99)
- Unable to accept new requests

**Measurement:**
```bash
# Check uptime
curl http://localhost:8001/health

# Calculate uptime percentage
uptime_percent = (total_time - downtime) / total_time * 100
```

---

### Success Rate

| Metric | Target | Notes |
|--------|--------|-------|
| **Overall success rate** | >99% | Non-5xx responses |
| **Agent success rate** | >98% | Agents complete without errors |
| **LLM success rate** | >95% | LLM calls succeed (including retries) |
| **External API success rate** | >90% | GitLab, YouTrack, etc. |

**Success Definition:**
- HTTP 200 response
- Pipeline completes to summarization
- No unhandled exceptions

**Measurement:**
```go
totalRequests := successCount + errorCount
successRate := float64(successCount) / float64(totalRequests) * 100
// Target: >99%
```

---

## Scalability Targets

### Horizontal Scaling

| Deployment | Instances | Max Throughput | Concurrent Sessions |
|------------|-----------|----------------|-------------------|
| **Development** | 1 | 10 req/s | 10 |
| **Staging** | 2 | 50 req/s | 50 |
| **Production (small)** | 3-5 | 500 req/s | 500 |
| **Production (large)** | 10-20 | 5000 req/s | 5000 |

**Scaling Strategy:**
- Stateless agents (no shared state between instances)
- Shared cache (Redis cluster)
- Load balancer (round-robin or least-connections)

---

### Vertical Scaling

| Instance Size | CPU | Memory | Max Sessions | Cost/Month |
|---------------|-----|--------|--------------|------------|
| **Small** | 2 cores | 4GB | 50 | $50 |
| **Medium** | 4 cores | 8GB | 100 | $100 |
| **Large** | 8 cores | 16GB | 200 | $200 |

**Scaling Decision:**
- Scale horizontally first (better availability)
- Scale vertically for memory-heavy workloads

---

## Quality Targets

### Correctness

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Intent detection accuracy** | >90% | Human evaluation |
| **Entity extraction precision** | >85% | F1 score |
| **Inference quality** | >80% | User feedback |

---

### Consistency

| Metric | Target | Notes |
|--------|--------|-------|
| **Reproducibility (same input)** | 100% | Fixed model + temperature |
| **Contract adherence** | 100% | All postconditions fulfilled |
| **Schema compliance** | 100% | Valid AgentContext structure |

**Measurement:**
```go
func TestReproducibility(t *testing.T) {
    result1 := pipeline.Execute(ctx, input)
    result2 := pipeline.Execute(ctx, input)
    assert.Equal(t, result1, result2) // Must be identical
}
```

---

## Monitoring and Alerting

### Key Metrics to Monitor

**1. Latency Metrics**
```
# Prometheus metrics
session_duration_seconds{quantile="0.5"}   # p50
session_duration_seconds{quantile="0.99"}  # p99
agent_duration_ms{agent="intent_detection"}
```

**2. Throughput Metrics**
```
http_requests_total                        # Total requests
http_requests_per_second                   # Request rate
pipeline_completions_total                 # Completed pipelines
```

**3. Error Metrics**
```
http_errors_total{code="5xx"}              # Server errors
agent_failures_total{agent="inference"}    # Agent failures
budget_exceeded_total                      # Budget violations
```

**4. Cost Metrics**
```
llm_cost_usd_total                         # Total LLM costs
llm_cost_by_agent{agent="inference"}       # Per-agent costs
cache_hit_rate                             # Cache efficiency
```

---

### Alerting Rules

**Critical Alerts** (PagerDuty)

```yaml
# High error rate
alert: HighErrorRate
condition: error_rate > 0.05  # >5% errors
severity: critical
channels: [pagerduty, slack]

# Service down
alert: ServiceDown
condition: up == 0
severity: critical
channels: [pagerduty, slack]

# Budget exceeded
alert: BudgetExceeded
condition: budget_usage >= 1.0
severity: critical
channels: [pagerduty, slack]
```

**Warning Alerts** (Slack)

```yaml
# High latency
alert: HighLatency
condition: p99_latency > 30s
severity: warning
channels: [slack]

# Low cache hit rate
alert: LowCacheHitRate
condition: cache_hit_rate < 0.20
severity: warning
channels: [slack]

# Budget warning
alert: BudgetWarning
condition: budget_usage > 0.75
severity: warning
channels: [slack]
```

---

## Production Readiness Checklist

### Performance

- [ ] **p50 latency <5s** (measured over 1000+ requests)
- [ ] **p99 latency <25s** (measured over 1000+ requests)
- [ ] **Throughput >100 req/s** (load tested with hey/wrk)
- [ ] **Cache hit rate >40%** (for production workload)
- [ ] **Success rate >99%** (measured over 24 hours)

### Cost

- [ ] **Budget controls enforced** (hard limits tested)
- [ ] **Average cost <$0.20/session** (measured over 1000+ sessions)
- [ ] **Budget warnings tested** (alert fires at 75%)
- [ ] **Emergency degradation tested** (non-critical agents skip LLM)

### Reliability

- [ ] **Uptime >99.9%** (measured over 30 days)
- [ ] **Graceful degradation tested** (LLM provider outage)
- [ ] **Retry logic tested** (transient failures recover)
- [ ] **Circuit breakers tested** (failing providers isolated)

### Scalability

- [ ] **Horizontal scaling tested** (2+ instances behind load balancer)
- [ ] **Shared cache tested** (Redis cluster)
- [ ] **Stateless agents verified** (no shared state between instances)
- [ ] **Load tested** (10x expected traffic)

### Monitoring

- [ ] **Prometheus metrics exported** (all key metrics)
- [ ] **Grafana dashboards created** (latency, cost, errors)
- [ ] **Alerting configured** (PagerDuty + Slack)
- [ ] **Log aggregation tested** (ELK or similar)

### Quality

- [ ] **Contract validation passing** (all agents fulfill contracts)
- [ ] **Reproducibility tested** (same input = same output)
- [ ] **Test coverage >80%** (unit + integration)
- [ ] **Load tested** (sustained load for 1 hour)

---

## Benchmarking Guide

### 1. Latency Benchmarking

```bash
# Single request latency
time curl -X POST http://localhost:8001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o-mini", "messages": [{"role": "user", "content": "test"}]}'

# p50/p99 latency (load test)
hey -n 1000 -c 10 -m POST -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o-mini", "messages": [{"role": "user", "content": "test"}]}' \
  http://localhost:8001/v1/chat/completions
```

### 2. Throughput Benchmarking

```bash
# Max throughput
wrk -t12 -c400 -d30s --latency \
  -s post.lua http://localhost:8001/v1/chat/completions

# post.lua
wrk.method = "POST"
wrk.body = '{"model": "gpt-4o-mini", "messages": [{"role": "user", "content": "test"}]}'
wrk.headers["Content-Type"] = "application/json"
```

### 3. Cost Benchmarking

```bash
# Run 100 sessions, collect cost data
for i in {1..100}; do
  curl -X POST http://localhost:8001/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d "{\"model\": \"gpt-4o-mini\", \"messages\": [{\"role\": \"user\", \"content\": \"test $i\"}]}"
done

# Get cost stats
curl http://localhost:8001/debug/budget
```

### 4. Cache Benchmarking

```bash
# Cache cold (first run)
time curl ... # Slow

# Cache warm (second run, same prompt)
time curl ... # Fast

# Calculate hit rate
curl http://localhost:8001/metrics | grep cache_hit_rate
```

---

## Performance Tuning

### 1. Reduce Latency

**Enable parallel execution:**
```yaml
pipeline:
  mode: parallel  # 50% faster for multi-agent pipelines
```

**Reduce timeouts:**
```yaml
agents:
  - id: fast_agent
    timeout: 3000  # 3s instead of 5s
```

**Use cheaper models:**
```yaml
llm:
  selection:
    defaults:
      intent_classification: "deepseek/deepseek-chat"  # Fastest
```

### 2. Reduce Cost

**Enable aggressive caching:**
```yaml
llm:
  cache:
    enabled: true
    classification_ttl: 86400  # 24 hours
```

**Use cheaper models:**
```yaml
llm:
  selection:
    defaults:
      inference: "openai/gpt-4o-mini"  # $0.00015 vs $0.0025
```

**Set strict budgets:**
```yaml
llm:
  budget:
    session_budget_usd: 0.50  # Hard limit
```

### 3. Improve Cache Hit Rate

**Normalize prompts:**
```go
// Remove timestamps
prompt = strings.ReplaceAll(prompt, time.Now().String(), "")

// Lowercase
prompt = strings.ToLower(prompt)

// Trim whitespace
prompt = strings.TrimSpace(prompt)
```

**Increase TTLs:**
```yaml
llm:
  cache:
    classification_ttl: 86400  # 24h instead of 1h
```

### 4. Improve Success Rate

**Enable retries:**
```yaml
agents:
  - id: flaky_agent
    retry: 3  # Retry up to 3 times
```

**Add fallback chains:**
```yaml
llm:
  selection:
    fallbacks:
      gpt_4o: ["anthropic/claude-sonnet", "openai/gpt-4o-mini"]
```

---

## References

- [Pipeline Configuration](./pipeline_configuration.md)
- [LLM Selection Policies](./llm_selection_policies.md)
- [Troubleshooting Guide](./troubleshooting.md)
- [Example Configurations](../../examples/pipelines/)
- Source: `src/golang/internal/application/services/`
- Metrics: `src/golang/internal/infrastructure/metrics/`
