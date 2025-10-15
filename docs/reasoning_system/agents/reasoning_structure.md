# Reasoning Structure Agent

## Overview

The **Reasoning Structure Agent** is the second agent in the reasoning pipeline, responsible for building structured reasoning plans from detected user intents. It generates hypothesis hierarchies, creates dependency graphs between reasoning steps, and ensures logical consistency in the reasoning flow.

**Agent ID**: `reasoning_structure`

**Version**: 1.0.0

**Type**: Rule-based Planning Agent

**Dependencies**: Intent Detection Agent

## Key Characteristics

- **Zero LLM Calls**: Pure rule-based hypothesis generation
- **Deterministic**: Same intents always produce same structure
- **Fast**: ~100ms execution time
- **Structured Output**: Hierarchical hypotheses with explicit dependencies
- **Cycle Detection**: Automatically detects and breaks dependency cycles
- **Intent-Aware**: Generates appropriate hypotheses based on intent types

## Architecture

```
Input: reasoning.intents[] (from Intent Detection Agent)
       reasoning.entities{} (optional)

Processing:
  1. Validate preconditions (intents exist)
  2. Generate hypotheses for each high-confidence intent
  3. Build dependency graph showing hypothesis relationships
  4. Detect cycles in dependency graph
  5. Break cycles if detected (automatic)
  6. Output structured reasoning plan

Output: reasoning.hypotheses[]
        reasoning.dependency_map
```

## Agent Contract

### Preconditions

The agent requires the following context keys to be present:

- **`reasoning.intents`**: Array of detected intents with confidence scores

### Postconditions

The agent guarantees to populate the following context keys:

- **`reasoning.hypotheses`**: Array of reasoning hypotheses
- **`reasoning.dependency_map`**: Graph structure showing dependencies

### Capabilities

```go
SupportsParallelExecution: false  // Must run after intent detection
SupportsRetry:             true   // Safe to retry on failure
RequiresLLM:               false  // Rule-based, no LLM needed
IsDeterministic:           true   // Same input → same output
EstimatedDuration:         100ms  // ~100ms for structure generation
```

## Hypothesis Generation

### Intent Type: `query_commits`

**Use Case**: User wants to query commit history

**Hypotheses Generated** (3 steps):

1. **H0**: Retrieve commit data from GitLab
   - Dependencies: none
   - Description: Fetch raw commit data from source

2. **H1**: Filter and rank commits by relevance
   - Dependencies: [H0]
   - Description: Apply filters and sort commits

3. **H2**: Format commit summary for user
   - Dependencies: [H1]
   - Description: Create human-readable output

**Dependency Chain**: H0 → H1 → H2 (sequential)

**Example Input**:
```json
{
  "intents": [
    {
      "type": "query_commits",
      "confidence": 0.9
    }
  ]
}
```

**Example Output**:
```json
{
  "hypotheses": [
    {
      "id": "h0",
      "description": "Retrieve commit data from GitLab",
      "dependencies": []
    },
    {
      "id": "h1",
      "description": "Filter and rank commits by relevance",
      "dependencies": ["h0"]
    },
    {
      "id": "h2",
      "description": "Format commit summary for user",
      "dependencies": ["h1"]
    }
  ]
}
```

---

### Intent Type: `query_issues`

**Use Case**: User wants to query issue tracking data

**Hypotheses Generated** (3 steps):

1. **H0**: Retrieve issue data from YouTrack
2. **H1**: Apply filters based on entities (status, date, project)
3. **H2**: Format issue summary for user

**Dependency Chain**: H0 → H1 → H2 (sequential)

---

### Intent Type: `query_analytics`

**Use Case**: User requests statistics, metrics, or trends

**Hypotheses Generated** (4 steps):

1. **H0**: Identify relevant data sources (commits, issues, metrics)
2. **H1**: Aggregate data from multiple sources
3. **H2**: Calculate statistics and trends
4. **H3**: Generate analytics report with visualizations

**Dependency Chain**: H0 → H1 → H2 → H3 (sequential)

---

### Intent Type: `query_status`

**Use Case**: User wants system status or health check

**Hypotheses Generated** (3 steps):

1. **H0**: Check system health and status (independent)
2. **H1**: Gather recent events and changes (independent)
3. **H2**: Synthesize overall status report (depends on H0 and H1)

**Dependency Chain**:
```
H0 ─┐
    ├→ H2
H1 ─┘
```
*Note: H0 and H1 can run in parallel*

---

### Intent Type: `command_action`

**Use Case**: User wants to execute a command

**Hypotheses Generated** (3 steps):

1. **H0**: Validate command preconditions and permissions
2. **H1**: Execute command action
3. **H2**: Verify command execution result

**Dependency Chain**: H0 → H1 → H2 (sequential)

---

### Intent Type: `request_help`

**Use Case**: User asks for help or documentation

**Hypotheses Generated** (3 steps):

1. **H0**: Identify help topic and user context
2. **H1**: Retrieve relevant documentation and examples
3. **H2**: Format helpful response with examples

**Dependency Chain**: H0 → H1 → H2 (sequential)

---

### Intent Type: `conversation`

**Use Case**: General conversation or greetings

**Hypotheses Generated** (1 step):

1. **H0**: Generate appropriate conversational response

**Dependency Chain**: Single step (no dependencies)

---

## Dependency Graph Structure

The dependency graph is represented as:

```go
type DependencyGraph struct {
    Nodes []string            // List of all hypothesis IDs
    Edges map[string][]string // from -> [to1, to2, ...]
}
```

### Graph Properties

- **Nodes**: All hypothesis IDs (h0, h1, h2, ...)
- **Edges**: Directed edges showing "A must complete before B"
- **Acyclic**: Cycles are automatically detected and broken

### Example: Linear Chain

```
H0 → H1 → H2

Nodes: ["h0", "h1", "h2"]
Edges: {
  "h0": ["h1"],
  "h1": ["h2"]
}
```

### Example: Parallel Start

```
H0 ─┐
    ├→ H2
H1 ─┘

Nodes: ["h0", "h1", "h2"]
Edges: {
  "h0": ["h2"],
  "h1": ["h2"]
}
```

## Cycle Detection & Breaking

### Detection Algorithm

Uses **Depth-First Search (DFS)** with recursion stack tracking:

1. Visit each node in the graph
2. Track visited nodes and nodes in current recursion stack
3. If a node in recursion stack is visited again, a cycle is detected
4. Return all detected cycles

### Cycle Breaking Strategy

When cycles are detected:

1. **Log Warning**: Cycle information added to diagnostics
2. **Break Cycle**: Remove edge from second-to-last to last node in cycle
3. **Verify**: Re-run cycle detection to ensure graph is acyclic
4. **Continue**: Execution continues with broken cycle

**Example**:
```
Before: H0 → H1 → H2 → H0 (cycle!)
After:  H0 → H1 → H2 (cycle broken)
```

## Confidence Filtering

- **Threshold**: 0.3 minimum confidence
- **Behavior**: Intents with confidence < 0.3 are skipped
- **Rationale**: Low-confidence intents produce unreliable hypotheses

**Example**:
```json
{
  "intents": [
    {"type": "query_commits", "confidence": 0.9},  // ✓ Generates hypotheses
    {"type": "query_issues", "confidence": 0.2}    // ✗ Skipped (too low)
  ]
}
```

## Multiple Intent Handling

When multiple intents are detected, the agent:

1. **Processes All High-Confidence Intents**: Each intent ≥ 0.3 confidence
2. **Generates Separate Hypothesis Sets**: Each intent gets its own hypotheses
3. **Assigns Unique IDs**: Hypotheses have globally unique IDs (h0, h1, h2, ...)
4. **No Cross-Intent Dependencies**: Hypotheses from different intents are independent

**Example**:
```json
{
  "intents": [
    {"type": "query_commits", "confidence": 0.9},
    {"type": "query_issues", "confidence": 0.7}
  ]
}
```

**Output**: 6 hypotheses total (3 for commits, 3 for issues)

## Output Schema

### Hypothesis Structure

```go
type Hypothesis struct {
    ID           string   // Unique identifier (h0, h1, h2, ...)
    Description  string   // Human-readable description
    Dependencies []string // IDs of hypotheses this depends on
}
```

### Dependency Graph Structure

```go
type DependencyGraph struct {
    Nodes []string            // All hypothesis IDs
    Edges map[string][]string // Adjacency list (from -> [to])
}
```

### Full Output Example

```json
{
  "reasoning": {
    "hypotheses": [
      {
        "id": "h0",
        "description": "Retrieve commit data from GitLab",
        "dependencies": []
      },
      {
        "id": "h1",
        "description": "Filter and rank commits by relevance",
        "dependencies": ["h0"]
      },
      {
        "id": "h2",
        "description": "Format commit summary for user",
        "dependencies": ["h1"]
      }
    ],
    "dependency_map": {
      "nodes": ["h0", "h1", "h2"],
      "edges": {
        "h0": ["h1"],
        "h1": ["h2"]
      }
    }
  }
}
```

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| **Execution Time** | ~100ms | Rule-based, no I/O |
| **LLM Calls** | 0 | Fully deterministic |
| **Memory Usage** | <1MB | Lightweight processing |
| **Scalability** | O(n*m) | n=intents, m=hypotheses per intent |
| **Idempotent** | Yes | Same input → same output |

## Testing Coverage

### Unit Tests (25+ tests)

1. **Agent Initialization**: Metadata, capabilities, contract
2. **Precondition Validation**: Missing intents, nil context
3. **Hypothesis Generation**: All 7 intent types
4. **Dependency Graph**: Structure validation, edge correctness
5. **Cycle Detection**: No cycles, cycles present
6. **Cycle Breaking**: Automated cycle resolution
7. **Multiple Intents**: Correct hypothesis separation
8. **Confidence Filtering**: Low-confidence intents skipped
9. **Context Isolation**: Input context not modified
10. **Audit Trail**: Execution tracking, performance metrics
11. **Idempotency**: Same input produces same output

### Integration Tests

- **Full Pipeline**: Intent Detection → Reasoning Structure
- **Error Handling**: Graceful failure on invalid input
- **Performance**: <100ms execution time validation

## Usage Examples

### Example 1: Simple Commit Query

**Input**:
```go
ctx := models.NewAgentContext("session-123", "trace-456")
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
}

agent := agents.NewReasoningStructureAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// result.Reasoning.Hypotheses has 3 hypotheses
// result.Reasoning.DependencyMap has linear chain
```

---

### Example 2: Multiple Intents

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
    {Type: "query_analytics", Confidence: 0.8},
}

result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// result.Reasoning.Hypotheses has 7 hypotheses total
// 3 from query_commits + 4 from query_analytics
```

---

### Example 3: Status Query with Parallel Start

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_status", Confidence: 0.85},
}

result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// Hypotheses:
// h0: Check system health (no deps)
// h1: Gather recent events (no deps)
// h2: Synthesize status report (depends on h0, h1)
//
// Dependency graph shows h0 and h1 can run in parallel
```

---

### Example 4: Handling Low Confidence

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
    {Type: "conversation", Confidence: 0.2}, // Too low
}

result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// Only query_commits hypotheses generated
// conversation intent skipped due to low confidence
```

## Future Enhancements

### Planned (Not Yet Implemented)

1. **LLM Fallback for Complex Structures**
   - Use LLM for ambiguous or complex intent combinations
   - Generate custom hypotheses for unknown patterns

2. **Hypothesis Optimization**
   - Merge redundant hypotheses across intents
   - Parallelize independent hypothesis chains

3. **Entity-Aware Hypothesis Generation**
   - Use extracted entities to refine hypothesis descriptions
   - Generate entity-specific validation hypotheses

4. **Confidence Propagation**
   - Propagate intent confidence to generated hypotheses
   - Prioritize high-confidence hypothesis execution

5. **Custom Hypothesis Templates**
   - User-defined hypothesis templates per intent type
   - Configurable dependency patterns

## Troubleshooting

### Issue: "No intents found in context"

**Cause**: Reasoning context missing or empty intents array

**Solution**:
- Ensure Intent Detection Agent runs first
- Verify `reasoning.intents` is populated
- Check preconditions are satisfied

---

### Issue: "Cycle detected in dependency graph"

**Cause**: Hypothesis dependencies form a loop

**Solution**:
- Check `diagnostics.warnings` for cycle details
- Cycles are automatically broken
- Review hypothesis generation logic if cycles persist

---

### Issue: "Empty hypotheses array"

**Cause**: All intents below confidence threshold (< 0.3)

**Solution**:
- Check intent confidence scores
- Review Intent Detection Agent output
- Lower confidence threshold if needed (not recommended)

---

### Issue: "Hypothesis IDs not unique"

**Cause**: Internal error in hypothesis generation

**Solution**:
- This should never happen (IDs auto-incremented)
- File a bug report if observed

## Related Documentation

- [Intent Detection Agent](./intent_detection.md) - Required precursor agent
- [Agent Context Schema](../agent_context_schema.md) - Context structure details
- [Pipeline Configuration](../pipeline_configuration.md) - Agent orchestration
- [Retrieval Planner Agent](./retrieval_planner.md) - Next agent in pipeline

## API Reference

### Constructor

```go
func NewReasoningStructureAgent() *ReasoningStructureAgent
```

Creates a new reasoning structure agent with default configuration.

**Returns**: Configured agent instance

---

### AgentID

```go
func (a *ReasoningStructureAgent) AgentID() string
```

Returns the unique agent identifier.

**Returns**: `"reasoning_structure"`

---

### Execute

```go
func (a *ReasoningStructureAgent) Execute(
    ctx context.Context,
    agentContext *models.AgentContext
) (*models.AgentContext, error)
```

Executes the reasoning structure generation process.

**Parameters**:
- `ctx`: Cancellation context
- `agentContext`: Input agent context (must contain `reasoning.intents`)

**Returns**:
- Updated context with `reasoning.hypotheses` and `reasoning.dependency_map`
- Error if preconditions not met or execution fails

---

### GetMetadata

```go
func (a *ReasoningStructureAgent) GetMetadata() services.AgentMetadata
```

Returns agent metadata for introspection.

**Returns**: AgentMetadata with ID, name, description, version, tags, dependencies

---

### GetCapabilities

```go
func (a *ReasoningStructureAgent) GetCapabilities() services.AgentCapabilities
```

Returns agent capabilities declaration.

**Returns**: AgentCapabilities with execution characteristics

## Changelog

### Version 1.0.0 (2025-01-15)

**Initial Release**:
- Rule-based hypothesis generation for 7 intent types
- Dependency graph construction with cycle detection
- Automatic cycle breaking
- Low-confidence intent filtering (threshold: 0.3)
- Full test coverage (25+ unit tests)
- Zero LLM calls for fast, deterministic execution
