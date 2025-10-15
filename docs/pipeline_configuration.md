# Pipeline Configuration Guide

This document describes the YAML-based pipeline configuration format for the Reasoning Agent System.

## Overview

The pipeline configuration defines how reasoning agents are orchestrated, including:
- Execution mode (sequential, parallel, or conditional)
- Agent dependencies and execution order
- Timeouts and retry policies
- Execution conditions

## Configuration File Structure

```yaml
pipeline:
  mode: sequential  # sequential | parallel | conditional
  agents:
    - id: agent_id
      enabled: true
      depends_on: []
      timeout: 5s
      retry: 2
      conditions: []
```

## Fields Reference

### Pipeline Section

#### `mode` (required)
- **Type:** string
- **Values:** `sequential`, `parallel`, `conditional`
- **Description:** Defines how agents are executed

**Sequential Mode:**
- Agents execute one after another in the order listed
- Each agent waits for the previous agent to complete
- If an agent fails, the pipeline stops (unless retry succeeds)
- Use when agents have strict dependencies on previous agents' outputs

**Parallel Mode:**
- Agents execute based on their dependency graph
- Agents with no dependencies run immediately
- Agents with dependencies wait for all their dependencies to complete
- Multiple independent agents can run concurrently
- Use for optimal performance when agents have partial dependencies

**Conditional Mode:**
- Each agent checks its conditions before executing
- Agents execute sequentially in the order listed
- Agents skip execution if their conditions are not satisfied
- Use when agent execution depends on runtime context state

#### `agents` (required)
- **Type:** array of AgentConfig
- **Description:** List of agents in the pipeline
- **Minimum:** 1 agent required

### Agent Configuration

#### `id` (required)
- **Type:** string
- **Description:** Unique identifier for the agent
- **Examples:** `intent_detection`, `reasoning_structure`, `summarization`
- **Constraints:**
  - Must be unique within the pipeline
  - Cannot be empty
  - Should match the agent's `AgentID()` implementation

#### `enabled` (required)
- **Type:** boolean
- **Default:** true
- **Description:** Whether the agent is active in the pipeline
- **Usage:**
  - Set to `false` to temporarily disable an agent without removing it
  - Disabled agents are skipped during execution
  - Dependencies on disabled agents cause dependent agents to skip

#### `depends_on` (optional)
- **Type:** array of strings
- **Default:** `[]` (no dependencies)
- **Description:** List of agent IDs that must complete before this agent runs
- **Usage:**
  - Only applicable in `parallel` mode
  - Ignored in `sequential` mode (order is implicit)
  - May be used in `conditional` mode for documentation purposes
- **Constraints:**
  - Referenced agents must exist in the pipeline
  - Cannot depend on itself
  - No circular dependencies allowed (validated at load time)

#### `timeout` (required)
- **Type:** string (duration)
- **Format:** Go duration format (e.g., `5s`, `30s`, `1m`, `500ms`)
- **Default:** `30s` if not specified
- **Description:** Maximum execution time for the agent
- **Behavior:**
  - If agent exceeds timeout, execution is cancelled
  - Timeout includes retry attempts
  - Context cancellation is propagated to the agent

#### `retry` (required)
- **Type:** integer
- **Default:** 0 (no retries)
- **Range:** 0-10 (recommended)
- **Description:** Number of retry attempts on agent failure
- **Behavior:**
  - Agent is retried immediately on failure (with 100ms delay between retries)
  - All retry attempts must complete within the `timeout` period
  - If all retries fail, the pipeline fails (in sequential/conditional) or the agent is marked as failed (in parallel)

#### `conditions` (optional, conditional mode only)
- **Type:** array of strings
- **Default:** `[]` (always execute)
- **Description:** Context keys that must exist for the agent to execute
- **Format:** `namespace.field` (e.g., `reasoning.intents`, `enrichment.facts`)
- **Usage:**
  - Only applicable in `conditional` mode
  - Ignored in `sequential` and `parallel` modes
  - If any condition is not satisfied, the agent is skipped

## Example Configurations

### Example 1: Sequential Pipeline (Default)

```yaml
pipeline:
  mode: sequential
  agents:
    - id: intent_detection
      enabled: true
      timeout: 5s
      retry: 2

    - id: reasoning_structure
      enabled: true
      timeout: 10s
      retry: 2

    - id: retrieval_planner
      enabled: true
      timeout: 5s
      retry: 2

    - id: context_synthesizer
      enabled: true
      timeout: 15s
      retry: 2

    - id: inference
      enabled: true
      timeout: 20s
      retry: 1

    - id: validation
      enabled: true
      timeout: 5s
      retry: 1

    - id: summarization
      enabled: true
      timeout: 10s
      retry: 1
```

**Execution:** Agents run in order: `intent_detection` → `reasoning_structure` → `retrieval_planner` → `context_synthesizer` → `inference` → `validation` → `summarization`

### Example 2: Parallel Pipeline with Dependencies

```yaml
pipeline:
  mode: parallel
  agents:
    # Level 0: No dependencies
    - id: intent_detection
      enabled: true
      depends_on: []
      timeout: 5s
      retry: 2

    # Level 1: Depend on intent_detection
    - id: reasoning_structure
      enabled: true
      depends_on:
        - intent_detection
      timeout: 10s
      retry: 2

    - id: retrieval_planner
      enabled: true
      depends_on:
        - intent_detection
      timeout: 5s
      retry: 2

    # Level 2: Depend on level 1 agents
    - id: context_synthesizer
      enabled: true
      depends_on:
        - retrieval_planner
      timeout: 15s
      retry: 2

    # Level 3: Depend on multiple agents
    - id: inference
      enabled: true
      depends_on:
        - context_synthesizer
        - reasoning_structure
      timeout: 20s
      retry: 1

    # Level 4: Depend on inference
    - id: validation
      enabled: true
      depends_on:
        - inference
      timeout: 5s
      retry: 1

    - id: summarization
      enabled: true
      depends_on:
        - validation
      timeout: 10s
      retry: 1
```

**Execution:**
- **Level 0:** `intent_detection` runs first
- **Level 1:** `reasoning_structure` and `retrieval_planner` run in parallel after `intent_detection`
- **Level 2:** `context_synthesizer` runs after `retrieval_planner`
- **Level 3:** `inference` runs after both `context_synthesizer` and `reasoning_structure` complete
- **Level 4:** `validation` and `summarization` run sequentially

### Example 3: Conditional Pipeline

```yaml
pipeline:
  mode: conditional
  agents:
    # Always runs (no conditions)
    - id: intent_detection
      enabled: true
      conditions: []
      timeout: 5s
      retry: 2

    # Runs only if intents were detected
    - id: entity_extraction
      enabled: true
      conditions:
        - reasoning.intents
      timeout: 5s
      retry: 1

    # Runs only if entities were extracted
    - id: validation
      enabled: true
      conditions:
        - reasoning.entities
      timeout: 5s
      retry: 1

    # Runs only if validation passed
    - id: summarization
      enabled: true
      conditions:
        - reasoning.conclusions
      timeout: 10s
      retry: 0
```

**Execution:**
- `intent_detection` always runs
- `entity_extraction` runs only if `reasoning.intents` exists in context
- `validation` runs only if `reasoning.entities` exists in context
- `summarization` runs only if `reasoning.conclusions` exists in context

### Example 4: Minimal Configuration

```yaml
pipeline:
  mode: sequential
  agents:
    - id: simple_agent
      enabled: true
      timeout: 30s
      retry: 0
```

**Notes:**
- Uses default timeout of 30s
- No retries on failure
- Sequential mode with a single agent

## Loading Configuration

### Go Code

```go
import "github.com/mshogin/agents/internal/infrastructure/config"

// Load from file
pipelineConfig, err := config.LoadPipelineConfig("config/pipeline.yaml")
if err != nil {
    log.Fatalf("Failed to load pipeline config: %v", err)
}

// Create reasoning manager
manager := services.NewReasoningManager(*pipelineConfig)
```

### Configuration Validation

The configuration loader performs the following validations:

1. **Mode validation:** Must be `sequential`, `parallel`, or `conditional`
2. **Agent count:** At least one agent must be configured
3. **Unique agent IDs:** No duplicate agent IDs allowed
4. **Dependency validation:**
   - All referenced dependencies must exist
   - Agents cannot depend on themselves
   - No circular dependencies
5. **Timeout format:** Must be valid Go duration (e.g., `5s`, `1m`)
6. **Timeout value:** Must be positive

**Error Examples:**

```
invalid execution mode: unknown (valid: sequential, parallel, conditional)
no agents configured
duplicate agent ID: intent_detection
agent reasoning_structure depends on unknown agent: nonexistent
agent validation cannot depend on itself
circular dependency detected involving agent: agent_a
invalid timeout format: abc (examples: 5s, 30s, 1m)
timeout must be positive: -5s
```

## Agent Context Keys

Agents can read from and write to namespaced context keys. Common keys used in conditions:

### Reasoning Namespace
- `reasoning.intents` - Detected user intents
- `reasoning.entities` - Extracted entities
- `reasoning.hypotheses` - Generated hypotheses
- `reasoning.conclusions` - Reasoning conclusions
- `reasoning.summary` - Final summary

### Enrichment Namespace
- `enrichment.facts` - Collected facts
- `enrichment.derived_knowledge` - Derived knowledge
- `enrichment.relationships` - Entity relationships

### Retrieval Namespace
- `retrieval.plans` - Retrieval plans
- `retrieval.queries` - Generated queries
- `retrieval.artifacts` - Retrieved artifacts

### LLM Namespace
- `llm.usage` - LLM token usage
- `llm.decisions` - Model selection decisions

## Performance Considerations

### Sequential Mode
- **Latency:** Sum of all agent execution times
- **Throughput:** One pipeline at a time (single-threaded)
- **Use when:** Agents have strict dependencies

### Parallel Mode
- **Latency:** Longest path through the dependency graph
- **Throughput:** Multiple agents can run concurrently
- **Use when:** Agents have partial dependencies
- **Note:** Overhead of goroutine spawning and synchronization

### Conditional Mode
- **Latency:** Sum of executed agents only (skipped agents have zero latency)
- **Throughput:** One pipeline at a time (single-threaded)
- **Use when:** Agent execution depends on runtime conditions

## Best Practices

1. **Keep timeouts reasonable:**
   - Simple agents (classification, extraction): 5-10s
   - Medium complexity (synthesis, planning): 10-20s
   - Complex agents (inference, reasoning): 20-30s

2. **Use retries sparingly:**
   - Idempotent operations: 1-2 retries
   - Non-idempotent or slow operations: 0 retries
   - Use exponential backoff for external API calls (handled in agent implementation)

3. **Organize agents by complexity:**
   - Fast, simple agents first
   - Slow, complex agents later
   - This enables early failure detection

4. **Use parallel mode for I/O-bound agents:**
   - Agents making external API calls
   - Agents performing database queries
   - Agents with no data dependencies

5. **Document agent contracts:**
   - Clearly define preconditions (required context keys)
   - Clearly define postconditions (guaranteed output keys)
   - Use contract validation in development

## Troubleshooting

### Pipeline fails immediately
- Check agent IDs match registered agents
- Verify all agents are registered before execution
- Check timeout values are positive and reasonable

### Circular dependency error
- Review `depends_on` fields
- Draw dependency graph to visualize cycles
- Remove or reorder dependencies to break cycle

### Agents skip execution in parallel mode
- Check that dependencies are enabled
- Verify dependency agent IDs are correct
- Review audit trail to see execution order

### Agents skip execution in conditional mode
- Verify condition keys exist in context
- Check preconditions of previous agents
- Enable contract validation to debug

### Timeouts occur frequently
- Increase timeout values
- Optimize agent implementation
- Check for network latency or API rate limits

## See Also

- [Agent Context Schema](./agent_context_schema.md) - Context namespace documentation
- [Agent Contracts](./agent_contracts.md) - Preconditions and postconditions
- [Reasoning Manager API](../src/golang/internal/application/services/reasoning_manager.go) - Implementation details
