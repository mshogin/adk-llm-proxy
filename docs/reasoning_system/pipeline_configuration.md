# Pipeline Configuration Documentation

## Overview

The ReasoningManager orchestrates agent execution in sequential, parallel, or conditional modes. Pipeline configuration defines agent ordering, dependencies, timeouts, retry policies, and execution options.

## Configuration Structure

```yaml
pipeline:
  mode: sequential  # sequential | parallel | conditional
  agents:
    - id: intent_detection
      enabled: true
      timeout: 5000  # milliseconds
      retry: 2       # max retries
      depends_on: [] # agent dependencies
    - id: reasoning_structure
      enabled: true
      timeout: 5000
      retry: 1
      depends_on: [intent_detection]
  options:
    validate_contract: true
    fail_on_violation: true
    track_performance: true
```

## Execution Modes

### 1. Sequential Mode

**Description**: Agents execute one after another in order

**Use When**:
- Linear dependency chain
- Each agent depends on previous agent's output
- Simple reasoning pipelines

**Configuration**:
```go
config := PipelineConfig{
    Mode: SequentialMode,
    Agents: []AgentConfig{
        {ID: "intent_detection", Enabled: true},
        {ID: "reasoning_structure", Enabled: true},
        {ID: "inference", Enabled: true},
    },
}
```

**Example Pipeline**:
```
Input → Intent Detection → Reasoning Structure → Inference → Output
```

**Advantages**:
- Simple and predictable
- Easy to debug
- Guaranteed execution order

**Disadvantages**:
- Slower (no parallelism)
- Blocked by slow agents

---

### 2. Parallel Mode

**Description**: Independent agents execute concurrently

**Use When**:
- Agents have no dependencies
- Multiple independent data sources
- Performance optimization needed

**Configuration**:
```go
config := PipelineConfig{
    Mode: ParallelMode,
    Agents: []AgentConfig{
        {ID: "root", Enabled: true, DependsOn: []string{}},
        {ID: "branch1", Enabled: true, DependsOn: []string{"root"}},
        {ID: "branch2", Enabled: true, DependsOn: []string{"root"}},
        {ID: "merge", Enabled: true, DependsOn: []string{"branch1", "branch2"}},
    },
}
```

**Example Pipeline**:
```
       ┌─ Branch1 ─┐
Root ──┤           ├─→ Merge → Output
       └─ Branch2 ─┘
```

**Advantages**:
- Faster (parallelism)
- Efficient resource usage
- Scalable

**Disadvantages**:
- More complex
- Requires careful dependency management
- **Warning**: Current implementation has race condition in audit trail

---

### 3. Conditional Mode

**Description**: Agents execute only if preconditions are met

**Use When**:
- Optional agent execution
- Different paths based on context
- Dynamic pipeline adaptation

**Configuration**:
```go
config := PipelineConfig{
    Mode: ConditionalMode,
    Agents: []AgentConfig{
        {ID: "intent_detection", Enabled: true},
        {ID: "retrieval", Enabled: true, Condition: "has_query_intent"},
        {ID: "inference", Enabled: true},
    },
}
```

**Example Pipeline**:
```
Intent Detection → [Retrieval?] → Inference → Output
                     (only if query intent detected)
```

**Advantages**:
- Flexible
- Efficient (skips unnecessary work)
- Adaptive

**Disadvantages**:
- More complex logic
- Harder to debug
- Non-deterministic execution

---

## Agent Configuration

### AgentConfig Structure

```go
type AgentConfig struct {
    ID         string   // Unique agent identifier
    Enabled    bool     // Enable/disable agent
    Timeout    int      // Timeout in milliseconds
    Retry      int      // Max retry attempts
    DependsOn  []string // Agent dependencies (parallel mode)
    Condition  string   // Execution condition (conditional mode)
}
```

### Configuration Options

#### ID (Required)

Unique identifier matching registered agent's `AgentID()`

```yaml
id: intent_detection
```

#### Enabled (Default: true)

Enable or disable agent execution

```yaml
enabled: false  # Skip this agent
```

#### Timeout (Default: 5000ms)

Maximum execution time before cancellation

```yaml
timeout: 10000  # 10 seconds
```

#### Retry (Default: 0)

Number of retry attempts on failure

```yaml
retry: 3  # Retry up to 3 times
```

#### DependsOn (Parallel mode only)

List of agent IDs that must complete first

```yaml
depends_on:
  - intent_detection
  - reasoning_structure
```

#### Condition (Conditional mode only)

Condition for agent execution

```yaml
condition: has_query_intent
```

---

## Execution Options

```go
type AgentExecutionOptions struct {
    ValidateContract bool // Enable contract validation
    FailOnViolation  bool // Stop pipeline on violation
    TrackPerformance bool // Track execution metrics
}
```

### ValidateContract

**Description**: Enable pre/postcondition validation

**Default**: `false`

**Example**:
```go
options := AgentExecutionOptions{
    ValidateContract: true,
}
```

**Effect**:
- Checks preconditions before execution
- Verifies postconditions after execution
- Logs validation results

### FailOnViolation

**Description**: Stop pipeline on contract violation

**Default**: `false`

**Example**:
```go
options := AgentExecutionOptions{
    ValidateContract: true,
    FailOnViolation:  true, // Stop on first violation
}
```

**Effect**:
- Pipeline stops immediately on violation
- Error returned to caller
- Partial results available in context

### TrackPerformance

**Description**: Collect performance metrics

**Default**: `false`

**Example**:
```go
options := AgentExecutionOptions{
    TrackPerformance: true,
}
```

**Effect**:
- Records execution duration per agent
- Tracks memory usage
- Populates `diagnostics.performance`

---

## Complete Configuration Examples

### Example 1: Simple Sequential Pipeline

```go
config := PipelineConfig{
    Mode: SequentialMode,
    Agents: []AgentConfig{
        {ID: "intent_detection", Enabled: true, Timeout: 5000},
        {ID: "reasoning_structure", Enabled: true, Timeout: 5000},
        {ID: "inference", Enabled: true, Timeout: 5000},
        {ID: "summarization", Enabled: true, Timeout: 5000},
    },
    Options: AgentExecutionOptions{
        ValidateContract: true,
        FailOnViolation:  true,
        TrackPerformance: true,
    },
}
```

### Example 2: Parallel Pipeline with Merge

```go
config := PipelineConfig{
    Mode: ParallelMode,
    Agents: []AgentConfig{
        {ID: "intent_detection", Enabled: true, DependsOn: []string{}},
        {ID: "gitlab_retrieval", Enabled: true, DependsOn: []string{"intent_detection"}},
        {ID: "youtrack_retrieval", Enabled: true, DependsOn: []string{"intent_detection"}},
        {ID: "synthesis", Enabled: true, DependsOn: []string{"gitlab_retrieval", "youtrack_retrieval"}},
        {ID: "inference", Enabled: true, DependsOn: []string{"synthesis"}},
    },
    Options: AgentExecutionOptions{
        ValidateContract: false, // Disable for performance
        TrackPerformance: true,
    },
}
```

### Example 3: Conditional Execution

```go
config := PipelineConfig{
    Mode: ConditionalMode,
    Agents: []AgentConfig{
        {ID: "intent_detection", Enabled: true},
        {ID: "retrieval", Enabled: true, Condition: "requires_retrieval"},
        {ID: "inference", Enabled: true},
        {ID: "validation", Enabled: true, Condition: "high_stakes"},
    },
    Options: AgentExecutionOptions{
        ValidateContract: true,
        FailOnViolation:  false, // Continue on validation errors
        TrackPerformance: true,
    },
}
```

---

## Pipeline Manager Usage

### Initialization

```go
// Create manager with configuration
manager := NewReasoningManager(config)

// Register agents
manager.RegisterAgent(NewIntentDetectionAgent())
manager.RegisterAgent(NewReasoningStructureAgent())
manager.RegisterAgent(NewInferenceAgent())
```

### Execution

```go
// Create context
ctx := context.Background()
agentContext := models.NewAgentContext("session-123", "trace-456")

// Execute pipeline
result, err := manager.Execute(ctx, agentContext)
if err != nil {
    log.Error("Pipeline failed:", err)
    return
}

// Access results
intents := result.Reasoning.Intents
conclusions := result.Reasoning.Conclusions
```

### Monitoring

```go
// Check execution audit
for _, run := range result.Audit.AgentRuns {
    log.Info("Agent:", run.AgentID,
             "Duration:", run.EndTime.Sub(run.StartTime),
             "Status:", run.Status)
}

// Check performance metrics
for _, metric := range result.Diagnostics.Performance {
    log.Info("Agent:", metric.AgentID,
             "Duration:", metric.DurationMS, "ms",
             "Memory:", metric.MemoryBytes, "bytes")
}
```

---

## Advanced Configuration

### Retry Policies

```go
config := AgentConfig{
    ID:      "flaky_agent",
    Enabled: true,
    Timeout: 5000,
    Retry:   3,  // Retry up to 3 times
}
```

**Retry Behavior**:
1. Initial attempt
2. If fails, retry #1 (after 1s delay)
3. If fails, retry #2 (after 2s delay)
4. If fails, retry #3 (after 4s delay)
5. If still fails, report error

### Timeout Handling

```go
config := AgentConfig{
    ID:      "slow_agent",
    Enabled: true,
    Timeout: 30000,  // 30 seconds
}
```

**Timeout Behavior**:
- Context cancelled after timeout
- Agent receives `context.DeadlineExceeded` error
- Pipeline continues with partial results (if `FailOnViolation: false`)

### Graceful Degradation

```go
config := PipelineConfig{
    Mode: SequentialMode,
    Agents: []AgentConfig{
        {ID: "critical_agent", Enabled: true},
        {ID: "optional_agent", Enabled: true},
    },
    Options: AgentExecutionOptions{
        ValidateContract: true,
        FailOnViolation:  false,  // Continue on optional agent failure
    },
}
```

---

## Performance Tuning

### Optimization Guidelines

1. **Disable Validation in Production**:
   ```go
   ValidateContract: false  // Save ~5-10% overhead
   ```

2. **Use Parallel Mode**:
   ```go
   Mode: ParallelMode  // 2-3x faster for independent agents
   ```

3. **Adjust Timeouts**:
   ```go
   Timeout: 2000  // Shorter timeout for fast agents
   ```

4. **Minimize Retries**:
   ```go
   Retry: 1  // Only retry once
   ```

### Performance Targets

- **Single agent (no LLM)**: <500ms
- **Full pipeline (no LLM)**: <2s
- **With LLM calls**: <10s (depends on model)

---

## Troubleshooting

### Issue: Agent Not Executing

**Symptoms**: Agent skipped, no entry in audit trail

**Causes**:
1. `Enabled: false` in config
2. Preconditions not met (conditional mode)
3. Dependency not satisfied (parallel mode)

**Solutions**:
1. Check agent is enabled
2. Verify preconditions are populated
3. Check `depends_on` agents completed

### Issue: Pipeline Timeout

**Symptoms**: `context.DeadlineExceeded` error

**Causes**:
1. Agent timeout too short
2. Slow external API calls
3. Infinite loop in agent code

**Solutions**:
1. Increase timeout value
2. Optimize external calls
3. Add timeout to agent logic

### Issue: Contract Violations

**Symptoms**: `postcondition validation failed` error

**Causes**:
1. Agent didn't populate postcondition
2. Empty array when non-empty expected
3. Wrong data type

**Solutions**:
1. Always populate postconditions
2. Ensure at least one element
3. Check data types match schema

---

## Best Practices

### 1. Start with Sequential Mode

✅ **DO**: Use sequential mode initially
```go
Mode: SequentialMode  // Simple and reliable
```

❌ **DON'T**: Jump to parallel mode
```go
Mode: ParallelMode  // More complex, potential race conditions
```

### 2. Enable Validation in Development

✅ **DO**: Validate contracts during development
```go
ValidateContract: true   // Catch errors early
FailOnViolation:  true   // Stop on first error
```

❌ **DON'T**: Disable validation during development
```go
ValidateContract: false  // Miss contract violations
```

### 3. Set Reasonable Timeouts

✅ **DO**: Set timeouts based on agent complexity
```go
{ID: "intent_detection", Timeout: 2000},  // Fast agent
{ID: "inference", Timeout: 10000},        // Slow agent
```

❌ **DON'T**: Use same timeout for all agents
```go
{ID: "intent_detection", Timeout: 30000},  // Too long for fast agent
```

### 4. Monitor Performance

✅ **DO**: Track performance metrics
```go
TrackPerformance: true  // Identify bottlenecks
```

---

## References

- [AgentContext Schema](./agent_context_schema.md)
- [Agent Contracts](./agent_contracts.md)
- Source: `src/golang/internal/application/services/reasoning_manager.go`
- Config: `src/golang/internal/infrastructure/config/pipeline_config.go`
- Tests: `src/golang/internal/application/services/reasoning_manager_test.go`
