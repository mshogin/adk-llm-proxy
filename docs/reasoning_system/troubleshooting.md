# Reasoning System Troubleshooting Guide

## Overview

This guide covers common issues, root causes, and solutions for the ADK LLM Proxy reasoning system. Issues are organized by subsystem.

---

## Agent Pipeline Issues

### Issue: Agent Not Executing

**Symptoms:**
- Agent skipped in pipeline
- No entry in `audit.agent_runs`
- Expected postconditions not populated

**Common Causes:**

1. **Agent disabled in configuration**
   ```yaml
   # Check config
   agents:
     - id: intent_detection
       enabled: false  # ← Agent disabled
   ```
   **Fix:** Set `enabled: true` in pipeline configuration

2. **Preconditions not met**
   ```go
   // Agent requires "reasoning.intents" but previous agent didn't populate it
   func (a *Agent) Preconditions() []string {
       return []string{"reasoning.intents"}  // Missing!
   }
   ```
   **Fix:** Check `audit.agent_runs` to see which agent failed to populate required keys

3. **Dependency not satisfied (parallel mode)**
   ```yaml
   agents:
     - id: inference
       depends_on: [reasoning_structure]  # reasoning_structure failed!
   ```
   **Fix:** Check `audit.agent_runs` for failed dependencies, fix upstream agent

**Debugging Steps:**
```go
// 1. Check agent registration
result, _ := manager.Execute(ctx, agentContext)
fmt.Println("Registered agents:", manager.ListAgents())

// 2. Check audit trail
for _, run := range result.Audit.AgentRuns {
    fmt.Printf("Agent: %s, Status: %s\n", run.AgentID, run.Status)
}

// 3. Check validation reports
for _, report := range result.Diagnostics.ValidationReports {
    fmt.Printf("Violations: %v\n", report.Violations)
}
```

---

### Issue: Pipeline Timeout

**Symptoms:**
- `context.DeadlineExceeded` error
- Pipeline stops mid-execution
- Incomplete `audit.agent_runs`

**Common Causes:**

1. **Agent timeout too short**
   ```yaml
   agents:
     - id: expensive_agent
       timeout: 1000  # 1 second - too short for LLM call
   ```
   **Fix:** Increase timeout based on agent complexity
   ```yaml
   timeout: 30000  # 30 seconds for LLM-heavy agents
   ```

2. **External API timeout**
   - LLM provider slow or unresponsive
   - MCP server timeout

   **Fix:** Check provider health, increase timeout, add retry logic

3. **Infinite loop in agent code**
   ```go
   func (a *Agent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
       // Missing context check!
       for {
           process()  // Never checks ctx.Done()
       }
   }
   ```
   **Fix:** Always check context cancellation
   ```go
   for {
       select {
       case <-ctx.Done():
           return nil, ctx.Err()
       default:
           process()
       }
   }
   ```

**Recommended Timeouts:**
| Agent Type | Recommended Timeout |
|------------|---------------------|
| No LLM (rules-only) | 5s |
| Simple LLM (classification) | 10s |
| Medium LLM (synthesis) | 30s |
| Complex LLM (reasoning) | 60s |
| Multi-agent parallel | 90s |

---

### Issue: Contract Violations

**Symptoms:**
- `postcondition validation failed` error
- `precondition validation failed` error
- Pipeline stops with validation error

**Common Violations:**

1. **Missing Postcondition**
   ```go
   func (a *IntentDetectionAgent) Postconditions() []string {
       return []string{"reasoning.intents"}  // Promises intents
   }

   func (a *IntentDetectionAgent) Execute(...) (*models.AgentContext, error) {
       // Forgot to populate intents!
       return agentContext, nil  // ← Contract violation
   }
   ```
   **Fix:** Always populate all postconditions
   ```go
   func (a *IntentDetectionAgent) Execute(...) (*models.AgentContext, error) {
       agentContext.Reasoning.Intents = []models.Intent{
           {Type: "query", Confidence: 0.95},
       }
       return agentContext, nil  // ✓ Contract fulfilled
   }
   ```

2. **Empty Array Violation**
   ```go
   // Some validation requires non-empty arrays
   agentContext.Reasoning.Intents = []models.Intent{}  // Empty!
   ```
   **Fix:** Ensure at least one element
   ```go
   if len(intents) == 0 {
       intents = []models.Intent{{Type: "unknown", Confidence: 0.0}}
   }
   agentContext.Reasoning.Intents = intents
   ```

3. **Wrong Data Type**
   ```go
   // Postcondition expects array, agent returns nil
   agentContext.Reasoning.Intents = nil  // Type mismatch
   ```
   **Fix:** Initialize empty array instead of nil
   ```go
   agentContext.Reasoning.Intents = []models.Intent{}
   ```

4. **Namespace Violation**
   ```go
   // IntentDetectionAgent writes to wrong namespace
   agentContext.Enrichment.Facts = facts  // ← Wrong namespace!
   ```
   **Fix:** Write only to designated namespace
   ```go
   // Intent agent writes to reasoning namespace only
   agentContext.Reasoning.Intents = intents  // ✓ Correct namespace
   ```

**Debugging:**
```bash
# Enable contract validation in config
go test -v -run TestContractViolations
```

---

## Context Validation Errors

### Issue: ContextViolationError

**Symptoms:**
- `agent wrote to unauthorized key` error
- Namespace isolation violated

**Cause:**
Agent writing outside designated namespace

```go
// Intent agent writing to enrichment namespace (wrong!)
agentContext.Enrichment.Facts = []models.Fact{...}
```

**Fix:**
Respect namespace boundaries
```go
// Each agent has designated namespace
// Intent Detection → reasoning.intents, reasoning.entities
// Retrieval Planner → retrieval.plans, retrieval.queries
// Context Synthesizer → enrichment.facts, enrichment.derived_knowledge
```

**Namespace Map:**
| Agent | Allowed Namespaces |
|-------|-------------------|
| IntentDetectionAgent | `reasoning.intents`, `reasoning.entities` |
| ReasoningStructureAgent | `reasoning.hypotheses` |
| RetrievalPlannerAgent | `retrieval.plans`, `retrieval.queries` |
| ContextSynthesizerAgent | `enrichment.facts`, `enrichment.derived_knowledge` |
| InferenceAgent | `reasoning.conclusions`, `reasoning.inference_chain` |
| ValidationAgent | `diagnostics.validation_reports` |
| SummarizationAgent | `reasoning.summary` |

---

### Issue: Context Size Exceeded

**Symptoms:**
- `context size limit exceeded` error
- Performance degradation
- Memory issues

**Causes:**
- Large artifacts in context (logs, raw data)
- Too many facts accumulated
- Large audit trail

**Fix 1: Externalize artifacts**
```go
// Instead of storing large content in context
agentContext.Enrichment.Facts = []models.Fact{
    {Content: largeContent},  // ← Bad
}

// Store reference to external storage
agentContext.Enrichment.Facts = []models.Fact{
    {Content: "s3://bucket/artifact-123.json", External: true},  // ✓ Good
}
```

**Fix 2: Enable compression**
```yaml
pipeline:
  context:
    compression: true
    max_size_mb: 10
```

**Fix 3: Prune old facts**
```go
// Keep only most recent facts
if len(agentContext.Enrichment.Facts) > 100 {
    agentContext.Enrichment.Facts = agentContext.Enrichment.Facts[len(agentContext.Enrichment.Facts)-100:]
}
```

---

## LLM Orchestrator Issues

### Issue: Budget Exceeded

**Symptoms:**
- `session budget exceeded` error
- `agent budget exceeded` error
- Requests blocked despite available session budget

**Causes:**

1. **Session budget exhausted**
   ```go
   sessionUsed := 0.95  // $0.95
   sessionLimit := 1.00 // $1.00
   // 95% spent, request blocked
   ```
   **Fix:** Increase session budget or optimize model selection
   ```yaml
   budget:
     session_budget_usd: 2.00  # Increase budget
     warning_threshold: 0.80   # Warn earlier
   ```

2. **Per-agent budget exhausted**
   ```go
   agentBudgets := map[string]float64{
       "inference": 0.11,  // Exceeded $0.10 limit
   }
   ```
   **Fix:** Increase agent budget or use cheaper models
   ```yaml
   budget:
     agent_budget_usd: 0.20  # Increase per-agent limit
   ```

3. **Expensive model selection**
   - Using GPT-4o or o1 for simple tasks

   **Fix:** Review model selection logic
   ```go
   // Check decisions
   for _, decision := range orchestrator.GetDecisions() {
       fmt.Printf("Agent: %s, Model: %s, Cost: %.4f\n",
           decision.AgentID, decision.Selected, decision.Cost)
   }
   ```

**Budget Optimization:**
```yaml
# Use cheaper models by default
budget:
  session_budget_usd: 1.00
  agent_budget_usd: 0.10
  warning_threshold: 0.80
  critical_agents:
    - validation      # Only critical agents bypass budget
    - summarization
```

---

### Issue: Cache Miss Rate Too High

**Symptoms:**
- Cache hit rate <10%
- High costs despite caching enabled
- Repeated identical requests not cached

**Causes:**

1. **Variable prompts**
   ```go
   // Timestamp in prompt breaks caching
   prompt := fmt.Sprintf("Analyze data at %v", time.Now())
   ```
   **Fix:** Normalize prompts
   ```go
   prompt := "Analyze data"  // Remove timestamps
   ```

2. **Different temperatures**
   ```go
   // Different temperatures create different cache keys
   req1 := LLMRequest{Prompt: "test", Temperature: 0.7}
   req2 := LLMRequest{Prompt: "test", Temperature: 0.8}  // ← Cache miss
   ```
   **Fix:** Use consistent temperature
   ```go
   temperature := 0.0  // Deterministic for caching
   ```

3. **TTL too short**
   ```yaml
   cache:
     classification_ttl: 60  # 1 minute - too short!
   ```
   **Fix:** Increase TTL for stable tasks
   ```yaml
   cache:
     classification_ttl: 86400  # 24 hours
     synthesis_ttl: 3600         # 1 hour
   ```

**Monitor cache performance:**
```go
entries, hits, size := orchestrator.GetCacheStats()
hitRate := float64(hits) / float64(entries+hits) * 100
fmt.Printf("Cache hit rate: %.1f%%\n", hitRate)
```

**Target hit rates:**
- Classification: >60%
- Synthesis: >40%
- Inference: >20%

---

### Issue: All Requests Use Expensive Models

**Symptoms:**
- Every request uses GPT-4o or Claude Opus
- High costs for simple tasks
- Model selection ignores task complexity

**Causes:**

1. **Task type incorrectly classified**
   ```go
   // Simple classification marked as complex reasoning
   taskType := TaskTypeDeepReasoning  // Wrong!
   ```
   **Fix:** Review task detection heuristics
   ```go
   func detectTaskType(prompt string) TaskType {
       if len(words(prompt)) < 50 {
           return TaskTypeIntentClassification  // Simple
       }
       // ...
   }
   ```

2. **Default model override**
   ```yaml
   llm:
     default_model: "gpt-4o"  # Expensive default
   ```
   **Fix:** Use cost-effective defaults
   ```yaml
   llm:
     default_model: "gpt-4o-mini"  # Cheap default
   ```

3. **Context size overestimated**
   ```go
   // Incorrectly calculates large context
   contextSize := len(allText)  // ← Overestimate
   ```
   **Fix:** Use token count, not character count
   ```go
   contextSize := estimateTokens(text)  // More accurate
   ```

**Debug model selection:**
```go
decisions := orchestrator.GetDecisions()
for _, d := range decisions {
    fmt.Printf("Task: %s, Selected: %s, Reason: %s\n",
        d.TaskType, d.Selected, d.Reason)
}
```

---

### Issue: Frequent Provider Fallbacks

**Symptoms:**
- Most requests use fallback models
- `fallback_1_default_unavailable` in decision logs
- Slow response times

**Causes:**

1. **Primary provider down**
   - DeepSeek API unavailable
   - OpenAI rate limited

   **Check:**
   ```bash
   curl -I https://api.deepseek.com/v1/chat/completions
   curl -I https://api.openai.com/v1/chat/completions
   ```

   **Fix:** Check API status, validate API keys
   ```bash
   export OPENAI_API_KEY="sk-..."
   ./proxy --config config.yaml
   ```

2. **Rate limits exceeded**
   ```
   HTTP 429 Too Many Requests
   ```
   **Fix:** Increase throttle limits or add delay
   ```yaml
   providers:
     openai:
       max_requests_per_second: 10  # Reduce rate
   ```

3. **Network issues**
   ```
   dial tcp: i/o timeout
   ```
   **Fix:** Check network, add retry logic, increase timeout
   ```yaml
   providers:
     openai:
       timeout_ms: 30000  # 30 seconds
       retry: 3
   ```

**Monitor provider health:**
```go
stats := orchestrator.GetProviderStats()
for provider, stat := range stats {
    fmt.Printf("%s: uptime %.1f%%, fallbacks: %d\n",
        provider, stat.UptimePercent, stat.FallbackCount)
}
```

---

## Performance Issues

### Issue: High Latency (>2s for non-LLM pipeline)

**Symptoms:**
- Slow pipeline execution
- Non-LLM agents taking >500ms each
- Full pipeline >2s without LLM calls

**Causes:**

1. **Synchronous execution (should be parallel)**
   ```go
   // Sequential execution (slow)
   result1 := agent1.Execute(ctx, agentContext)
   result2 := agent2.Execute(ctx, result1)
   ```
   **Fix:** Use parallel execution where possible
   ```yaml
   pipeline:
     mode: parallel
     agents:
       - id: retrieval_gitlab
         depends_on: [intent_detection]
       - id: retrieval_youtrack
         depends_on: [intent_detection]  # Parallel with gitlab
   ```

2. **Large context serialization**
   - Copying/cloning large contexts

   **Fix:** Use context references, enable compression
   ```yaml
   pipeline:
     context:
       compression: true
       lazy_load: true
   ```

3. **Inefficient validation**
   - Validating on every operation

   **Fix:** Disable validation in production
   ```yaml
   pipeline:
     options:
       validate_contract: false  # Disable in prod
   ```

**Benchmark agents:**
```bash
go test -bench=. -benchmem ./internal/application/services/
```

**Target latencies:**
- Single agent (no LLM): <500ms
- Full pipeline (no LLM): <2s
- With LLM: <10s

---

### Issue: High Memory Usage

**Symptoms:**
- Memory usage >500MB under load
- OOM errors
- Garbage collection pressure

**Causes:**

1. **Context accumulation**
   - Large contexts not released

   **Fix:** Clear context after use
   ```go
   defer func() {
       agentContext = nil  // Allow GC
   }()
   ```

2. **Cache unbounded growth**
   - No cache eviction

   **Fix:** Set cache size limits
   ```yaml
   cache:
     max_entries: 1000
     max_size_mb: 100
   ```

3. **Goroutine leaks**
   - Goroutines not terminating

   **Fix:** Always use context cancellation
   ```go
   ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
   defer cancel()
   ```

**Monitor memory:**
```bash
go tool pprof http://localhost:8001/debug/pprof/heap
```

---

## Provider/API Issues

### Issue: Authentication Failures

**Symptoms:**
- `401 Unauthorized`
- `invalid API key`

**Causes:**
- Missing or invalid API key
- API key not loaded from environment

**Fix:**
```bash
# Check environment variables
echo $OPENAI_API_KEY
echo $ANTHROPIC_API_KEY

# Set if missing
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Verify config loads env vars
grep -A2 "api_key" config.yaml
```

**Config:**
```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"  # Expands from env
```

---

### Issue: Rate Limiting (429)

**Symptoms:**
- `429 Too Many Requests`
- Frequent fallbacks
- Slow responses

**Fix:**
```yaml
providers:
  openai:
    max_requests_per_second: 5  # Reduce rate
    retry: 3                     # Auto-retry
    backoff_ms: 1000            # 1s backoff
```

**Alternative:** Increase account tier with provider

---

### Issue: Context Length Exceeded

**Symptoms:**
- `context_length_exceeded` error
- Request rejected by provider

**Causes:**
- Prompt + context >128K tokens (GPT-4o-mini limit)

**Fix:**
1. Truncate context
   ```go
   if tokenCount > 120000 {
       context = truncate(context, 120000)
   }
   ```

2. Use model with larger context
   ```yaml
   # Use Claude Sonnet (200K tokens) for large context
   ```

3. Summarize context
   ```go
   if tokenCount > limit {
       context = summarize(context)
   }
   ```

---

## Configuration Issues

### Issue: Config Not Loading

**Symptoms:**
- `config.yaml not found`
- Config values not applied

**Causes:**

1. **Wrong path**
   ```bash
   ./proxy --config wrong-path.yaml
   ```
   **Fix:**
   ```bash
   ./proxy --config config.yaml
   ```

2. **Invalid YAML syntax**
   ```yaml
   providers
     openai:  # Missing colon after providers
   ```
   **Fix:**
   ```yaml
   providers:
     openai:  # Correct
   ```

3. **Environment variables not expanded**
   ```yaml
   api_key: "${OPENAI_API_KEY}"  # Not expanded
   ```
   **Fix:** Ensure environment variable is set
   ```bash
   export OPENAI_API_KEY="sk-..."
   ```

**Validate config:**
```bash
# Test config loading
go run cmd/proxy/main.go --config config.yaml --validate
```

---

### Issue: Pipeline Configuration Error

**Symptoms:**
- `agent not found`
- `circular dependency detected`
- `invalid agent configuration`

**Causes:**

1. **Agent ID mismatch**
   ```yaml
   agents:
     - id: intent_detect  # Wrong ID
   ```
   ```go
   func (a *IntentDetectionAgent) AgentID() string {
       return "intent_detection"  # Actual ID
   }
   ```
   **Fix:** Match IDs exactly

2. **Circular dependency**
   ```yaml
   agents:
     - id: agent_a
       depends_on: [agent_b]
     - id: agent_b
       depends_on: [agent_a]  # Circular!
   ```
   **Fix:** Remove circular dependency

3. **Missing dependency**
   ```yaml
   agents:
     - id: inference
       depends_on: [reasoning_structure]  # Not defined
   ```
   **Fix:** Define all dependencies

**Validate pipeline:**
```bash
go test -v -run TestPipelineConfig
```

---

## Testing and Debugging

### Debugging Agent Execution

**Enable verbose logging:**
```go
import "log"

func (a *Agent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
    log.Printf("[%s] Starting execution", a.AgentID())
    log.Printf("[%s] Preconditions: %v", a.AgentID(), a.Preconditions())

    // Process

    log.Printf("[%s] Postconditions populated: %v", a.AgentID(), a.Postconditions())
    return agentContext, nil
}
```

**Check audit trail:**
```go
result, _ := manager.Execute(ctx, agentContext)

// Print execution order and timing
for _, run := range result.Audit.AgentRuns {
    duration := run.EndTime.Sub(run.StartTime)
    fmt.Printf("Agent: %s\n", run.AgentID)
    fmt.Printf("  Status: %s\n", run.Status)
    fmt.Printf("  Duration: %v\n", duration)
    fmt.Printf("  Keys Written: %v\n", run.OutputKeys)
    if run.Error != "" {
        fmt.Printf("  Error: %s\n", run.Error)
    }
}
```

---

### Testing Individual Agents

**Unit test template:**
```go
func TestAgentContract(t *testing.T) {
    agent := NewMyAgent()
    ctx := context.Background()
    agentContext := models.NewAgentContext("test-session", "test-trace")

    // Populate preconditions
    agentContext.Reasoning.Intents = []models.Intent{
        {Type: "test", Confidence: 0.9},
    }

    // Execute
    result, err := agent.Execute(ctx, agentContext)
    require.NoError(t, err)

    // Verify postconditions
    assert.NotEmpty(t, result.Reasoning.Conclusions)
    assert.Greater(t, result.Reasoning.Conclusions[0].Confidence, 0.0)
}
```

---

### Reproducing Issues

**Capture context state:**
```go
// Save context before failure
contextJSON, _ := json.MarshalIndent(agentContext, "", "  ")
ioutil.WriteFile("debug_context.json", contextJSON, 0644)
```

**Replay with saved context:**
```go
// Load saved context
data, _ := ioutil.ReadFile("debug_context.json")
var agentContext models.AgentContext
json.Unmarshal(data, &agentContext)

// Replay pipeline
result, err := manager.Execute(ctx, &agentContext)
```

---

## Common Error Messages

### Error: `postcondition validation failed: reasoning.intents not found`

**Cause:** Agent didn't populate promised field

**Fix:** Ensure all postconditions are populated:
```go
func (a *Agent) Execute(...) (*models.AgentContext, error) {
    agentContext.Reasoning.Intents = []models.Intent{{Type: "query", Confidence: 0.9}}
    return agentContext, nil
}
```

---

### Error: `agent wrote to unauthorized key: enrichment.facts`

**Cause:** Namespace violation

**Fix:** Write only to designated namespace:
```go
// IntentDetectionAgent should write to reasoning namespace
agentContext.Reasoning.Intents = intents  // ✓ Correct
```

---

### Error: `circular dependency detected: [agent_a, agent_b, agent_a]`

**Cause:** Circular dependency in pipeline config

**Fix:** Break circular dependency:
```yaml
# Before (circular):
agents:
  - id: agent_a
    depends_on: [agent_b]
  - id: agent_b
    depends_on: [agent_a]

# After (linear):
agents:
  - id: agent_a
    depends_on: []
  - id: agent_b
    depends_on: [agent_a]
```

---

### Error: `budget exceeded: session spent $1.02, limit $1.00`

**Cause:** Session budget exhausted

**Fix:** Increase budget or optimize model selection:
```yaml
budget:
  session_budget_usd: 2.00  # Increase
```

---

### Error: `context deadline exceeded`

**Cause:** Agent timeout

**Fix:** Increase timeout:
```yaml
agents:
  - id: slow_agent
    timeout: 60000  # 60 seconds
```

---

## Best Practices

### Prevent Issues

1. **Enable validation in development**
   ```yaml
   pipeline:
     options:
       validate_contract: true
       fail_on_violation: true
   ```

2. **Use reasonable timeouts**
   - Fast agents: 5s
   - LLM agents: 30s
   - Complex agents: 60s

3. **Monitor budget usage**
   ```go
   sessionUsed, sessionLimit, _ := orchestrator.GetBudgetStatus()
   fmt.Printf("Budget: $%.2f / $%.2f (%.0f%%)\n",
       sessionUsed, sessionLimit, sessionUsed/sessionLimit*100)
   ```

4. **Test contracts in CI/CD**
   ```bash
   go test -v ./internal/domain/services/agents/
   ```

5. **Log all decisions**
   ```go
   for _, decision := range orchestrator.GetDecisions() {
       log.Printf("Decision: %+v", decision)
   }
   ```

---

## Getting Help

### Collect Debug Information

```bash
# 1. Check version and config
./proxy --version
cat config.yaml

# 2. Run with debug logging
export DEBUG=true
./proxy --config config.yaml

# 3. Check test output
go test -v ./...

# 4. Collect metrics
curl http://localhost:8001/metrics

# 5. Save context snapshot
# (see "Reproducing Issues" section)
```

### Report Issues

Include:
1. Error message and stack trace
2. `config.yaml` (redact API keys)
3. Context snapshot (if applicable)
4. Agent execution order from `audit.agent_runs`
5. LLM decisions from `llm.decisions`

---

## References

- [AgentContext Schema](./agent_context_schema.md)
- [Agent Contracts](./agent_contracts.md)
- [Pipeline Configuration](./pipeline_configuration.md)
- [LLM Selection Policies](./llm_selection_policies.md)
- Source: `src/golang/internal/application/services/`
- Tests: `tests/golang/`
