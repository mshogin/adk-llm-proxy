# Retrieval Planner Agent

## Overview

The **Retrieval Planner Agent** creates optimized retrieval plans for information gathering based on reasoning structure. It prioritizes structured data sources, generates normalized queries with entity-based filters, and ensures efficient data retrieval strategies.

**Agent ID**: `retrieval_planner`

**Version**: 1.0.0

**Type**: Rule-based Planning Agent

**Dependencies**: Intent Detection Agent, Reasoning Structure Agent

## Key Characteristics

- **Zero LLM Calls**: Pure rule-based plan generation
- **Deterministic**: Same intents produce same retrieval plans
- **Fast**: ~80ms execution time
- **Structured Source Priority**: GitLab/YouTrack > unstructured sources
- **Query Normalization**: Entity-based filters (project, date, status, provider)
- **Priority Sorting**: High-priority plans executed first

## Agent Contract

### Preconditions

- **`reasoning.intents`**: Array of detected intents with confidence scores
- **`reasoning.hypotheses`**: Reasoning structure with dependencies

### Postconditions

- **`retrieval.plans`**: Array of prioritized retrieval plans
- **`retrieval.queries`**: Array of normalized queries with filters

### Capabilities

```go
SupportsParallelExecution: false  // Must run after reasoning structure
SupportsRetry:             true   // Safe to retry on failure
RequiresLLM:               false  // Rule-based, no LLM needed
IsDeterministic:           true   // Same input → same output
EstimatedDuration:         80ms   // ~80ms for plan generation
```

## Retrieval Plans by Intent Type

### query_commits

**Retrieval Plan**:
- **Source**: GitLab (structured data)
- **Priority**: 10 (highest)
- **Filters**: projects, dates
- **Description**: "Retrieve commit data from GitLab"

**Example**:
```json
{
  "id": "plan0",
  "description": "Retrieve commit data from GitLab",
  "sources": ["gitlab"],
  "filters": {
    "projects": ["project1", "project2"],
    "dates": ["last week"]
  },
  "priority": 10
}
```

---

### query_issues

**Retrieval Plan**:
- **Source**: YouTrack (structured data)
- **Priority**: 10 (highest)
- **Filters**: projects, dates, statuses
- **Description**: "Retrieve issue data from YouTrack"

**Example**:
```json
{
  "id": "plan0",
  "description": "Retrieve issue data from YouTrack",
  "sources": ["youtrack"],
  "filters": {
    "projects": ["proj-123"],
    "dates": ["this month"],
    "statuses": ["open", "in-progress"]
  },
  "priority": 10
}
```

---

### query_analytics

**Retrieval Plans** (multiple):

1. **GitLab Metrics**
   - Source: GitLab
   - Priority: 9
   - Filters: projects, dates
   - Description: "Retrieve commit metrics from GitLab"

2. **YouTrack Metrics**
   - Source: YouTrack
   - Priority: 9
   - Filters: projects, dates
   - Description: "Retrieve issue metrics from YouTrack"

---

### query_status

**Retrieval Plans** (multiple):

1. **Recent Commits**
   - Source: GitLab
   - Priority: 8
   - Filters: dates=["last week"]
   - Description: "Retrieve recent commits for activity status"

2. **Open Issues**
   - Source: YouTrack
   - Priority: 8
   - Filters: statuses=["open", "in-progress"]
   - Description: "Retrieve open issues for health status"

## Query Normalization

### Query Structure

```go
type Query struct {
    ID          string                 // Unique query ID
    QueryString string                 // Human-readable query
    Source      string                 // Data source (gitlab, youtrack, etc.)
    Filters     map[string]interface{} // Entity-based filters
    Results     interface{}            // Populated after execution
}
```

### Query String Format

Query strings combine description with filter details:

**Format**: `{description} {filter1:values} {filter2:values}`

**Examples**:
```
"Retrieve commit data from GitLab projects:proj1,proj2 dates:last week"
"Retrieve issue data from YouTrack statuses:open,in-progress"
"Retrieve commit metrics from GitLab projects:repo1 dates:2024-01-15"
```

### Supported Filters

| Filter | Description | Example Values |
|--------|-------------|----------------|
| **projects** | Project identifiers | `["gitlab-repo", "youtrack-project"]` |
| **dates** | Time references | `["last week", "this month", "2024-01-15"]` |
| **statuses** | Status indicators | `["open", "closed", "in-progress"]` |
| **providers** | Service providers | `["gitlab", "youtrack", "openai"]` |

## Source Prioritization

Retrieval plans are prioritized to favor structured data sources:

| Source | Type | Priority | Use Case |
|--------|------|----------|----------|
| **gitlab** | Structured | 10 | Commit data, code metrics |
| **youtrack** | Structured | 10 | Issue data, task tracking |
| **analytics** | Structured | 9 | Aggregated metrics |
| **status** | Mixed | 8 | Health checks, recent activity |
| **generic** | Unstructured | 5 | Fallback for unknown intents |

**Priority Benefits**:
- **Higher accuracy**: Structured sources have reliable schemas
- **Faster execution**: Direct API queries vs full-text search
- **Better filtering**: Entity-based filters reduce noise

## Multiple Intent Handling

When multiple intents are detected:
1. Each high-confidence intent (≥0.3) generates separate plans
2. Plans are consolidated and sorted by priority
3. Highest-priority plans execute first

**Example**:
```json
{
  "intents": [
    {"type": "query_commits", "confidence": 0.9},
    {"type": "query_issues", "confidence": 0.85}
  ]
}
```

**Result**: 2 plans (GitLab commits at priority 10, YouTrack issues at priority 10)

## Output Schema

### Retrieval Plan

```go
type RetrievalPlan struct {
    ID          string                 // Unique plan ID
    Description string                 // Human-readable description
    Sources     []string               // Data sources to query
    Filters     map[string]interface{} // Entity-based filters
    Priority    int                    // Execution priority (higher first)
}
```

### Full Output Example

```json
{
  "retrieval": {
    "plans": [
      {
        "id": "plan0",
        "description": "Retrieve commit data from GitLab",
        "sources": ["gitlab"],
        "filters": {
          "projects": ["repo1"],
          "dates": ["last week"]
        },
        "priority": 10
      }
    ],
    "queries": [
      {
        "id": "plan0_query",
        "query_string": "Retrieve commit data from GitLab projects:repo1 dates:last week",
        "source": "gitlab",
        "filters": {
          "projects": ["repo1"],
          "dates": ["last week"]
        }
      }
    ]
  }
}
```

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| **Execution Time** | ~80ms | Rule-based, no I/O |
| **LLM Calls** | 0 | Fully deterministic |
| **Memory Usage** | <1MB | Lightweight processing |
| **Plans per Intent** | 1-2 | Based on intent complexity |
| **Idempotent** | Yes | Same input → same output |

## Usage Examples

### Example 1: Commit Query

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
}
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {ID: "h0", Description: "Retrieve commit data"},
}
ctx.Reasoning.Entities = map[string]interface{}{
    "projects": []string{"my-repo"},
    "dates":    []string{"this week"},
}

agent := agents.NewRetrievalPlannerAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// 1 plan: GitLab commits (priority 10)
// 1 query: "Retrieve commit data from GitLab projects:my-repo dates:this week"
```

---

### Example 2: Analytics Query

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_analytics", Confidence: 0.85},
}
```

**Output**:
```go
// 2 plans: GitLab metrics (priority 9) + YouTrack metrics (priority 9)
// 2 queries: One for each source
```

---

### Example 3: Multiple Intents

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
    {Type: "query_issues", Confidence: 0.8},
}
```

**Output**:
```go
// 2 plans: GitLab commits + YouTrack issues (both priority 10)
// Plans sorted by priority (tie broken by order)
```

## Testing Coverage

### Unit Tests (20+ tests)

1. **Agent Initialization**: Metadata, capabilities, contract
2. **Precondition Validation**: Missing intents, missing hypotheses
3. **Plan Generation**: All 4 intent types with validation
4. **Priority Sorting**: Correct ordering by priority
5. **Query Normalization**: Entity filters in query string
6. **Low-Confidence Filtering**: Intents < 0.3 skipped
7. **Multiple Intents**: Correct plan consolidation
8. **Context Isolation**: Input not modified
9. **Audit Trail**: Execution tracking, metrics
10. **Idempotency**: Same input produces same output

## Future Enhancements

1. **Volume Constraints**: Limit result count per source
2. **Time Constraints**: Query timeout limits
3. **Cost Optimization**: Balance speed vs cost
4. **Fallback Sources**: Retry with alternative sources
5. **Parallel Execution**: Execute independent plans concurrently

## Troubleshooting

### Issue: "No intents found in context"

**Cause**: Intent Detection Agent didn't run or produced no intents

**Solution**: Ensure Intent Detection Agent runs first and produces intents

---

### Issue: "No hypotheses found in context"

**Cause**: Reasoning Structure Agent didn't run

**Solution**: Ensure Reasoning Structure Agent runs before Retrieval Planner

---

### Issue: "Empty retrieval plans"

**Cause**: All intents below confidence threshold (< 0.3)

**Solution**: Review intent confidence scores from Intent Detection Agent

## Related Documentation

- [Intent Detection Agent](./intent_detection.md)
- [Reasoning Structure Agent](./reasoning_structure.md)
- [Context Synthesizer Agent](./context_synthesizer.md)
- [Agent Context Schema](../agent_context_schema.md)

## API Reference

### Constructor

```go
func NewRetrievalPlannerAgent() *RetrievalPlannerAgent
```

### Execute

```go
func (a *RetrievalPlannerAgent) Execute(
    ctx context.Context,
    agentContext *models.AgentContext
) (*models.AgentContext, error)
```

**Parameters**:
- `ctx`: Cancellation context
- `agentContext`: Must contain `reasoning.intents` and `reasoning.hypotheses`

**Returns**: Updated context with `retrieval.plans` and `retrieval.queries`

## Changelog

### Version 1.0.0 (2025-01-15)

**Initial Release**:
- Rule-based plan generation for 4 intent types
- Structured data source prioritization (priority 8-10)
- Normalized query generation with entity filters
- Priority-based plan sorting
- Low-confidence intent filtering
- Full test coverage (20+ unit tests)
- Zero LLM calls for fast, deterministic execution
