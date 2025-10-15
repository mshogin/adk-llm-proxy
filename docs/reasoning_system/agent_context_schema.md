# AgentContext Schema Documentation

## Overview

`AgentContext` is a versioned, namespaced context storage system for agent execution in the Reasoning Enrichment Agent System. It provides isolated namespaces for different types of data, enforces access control, tracks changes, and manages size limits.

## Architecture

### Design Principles

1. **Namespace Isolation**: Each agent can only write to its designated namespaces
2. **Versioned Schema**: Supports migration as schema evolves
3. **Complete Audit Trail**: All changes are tracked with diffs
4. **Size Management**: Automatic limits and artifact externalization
5. **Type Safety**: Strongly-typed Go structs with validation

### Location

- **Implementation**: `src/golang/internal/domain/models/agent_context.go`
- **Validation**: `src/golang/internal/domain/models/context_validator.go`
- **Diff Tracking**: `src/golang/internal/domain/models/context_diff.go`
- **Size Limits**: `src/golang/internal/domain/models/context_limits.go`

## Core Structure

```go
type AgentContext struct {
    Version     string              // Schema version (e.g., "1.0.0")
    Metadata    *MetadataContext    // Session and trace information
    Reasoning   *ReasoningContext   // Reasoning data
    Enrichment  *EnrichmentContext  // Facts and knowledge
    Retrieval   *RetrievalContext   // Retrieval plans and artifacts
    LLM         *LLMContext         // LLM usage and decisions
    Diagnostics *DiagnosticsContext // Errors, warnings, performance
    Audit       *AuditContext       // Agent runs and diffs
}
```

## Namespaces

### 1. Metadata

**Purpose**: Session and trace information (mostly read-only)

```go
type MetadataContext struct {
    SessionID string    // Unique session identifier
    TraceID   string    // Distributed tracing ID
    CreatedAt time.Time // Context creation timestamp
    Locale    string    // User locale (optional, writable)
}
```

**Write Access**: Orchestrator only (except `Locale`)
**Read Access**: All agents

---

### 2. Reasoning

**Purpose**: Core reasoning data - intents, hypotheses, conclusions

```go
type ReasoningContext struct {
    Intents          []Intent              // Detected user intents
    Entities         map[string]interface{} // Extracted entities
    Hypotheses       []Hypothesis          // Reasoning hypotheses
    Conclusions      []Conclusion          // Final conclusions
    DependencyMap    interface{}           // Dependency graph
    ConfidenceScores map[string]float64    // Confidence tracking
    Summary          string                // Final summary
}
```

**Key Types**:

```go
type Intent struct {
    Type       string   // Intent type (e.g., "query", "analysis")
    Confidence float64  // Confidence score (0.0-1.0)
    Entities   []string // Associated entities
}

type Hypothesis struct {
    ID           string   // Unique identifier
    Description  string   // Hypothesis description
    Dependencies []string // Dependent hypothesis IDs
}

type Conclusion struct {
    ID         string   // Unique identifier
    Content    string   // Conclusion text
    Confidence float64  // Confidence score
    BasedOn    []string // Source hypothesis/fact IDs
}
```

**Write Access**:
- Intent Detection Agent: `intents`, `entities`
- Reasoning Structure Agent: `hypotheses`, `dependency_map`
- Inference Agent: `conclusions`, `confidence_scores`
- Summarization Agent: `summary`

**Read Access**: All agents

---

### 3. Enrichment

**Purpose**: Facts, derived knowledge, and relationships

```go
type EnrichmentContext struct {
    Facts            []Fact         // Factual information
    DerivedKnowledge []Knowledge    // Inferred knowledge
    Relationships    []Relationship // Entity relationships
    ContextLinks     []string       // External context links
}
```

**Key Types**:

```go
type Fact struct {
    ID         string                 // Unique identifier
    Content    string                 // Fact content
    Source     string                 // Data source
    Timestamp  time.Time              // When fact was captured
    Confidence float64                // Confidence score
    Provenance map[string]interface{} // Source metadata
}

type Knowledge struct {
    ID          string   // Unique identifier
    Content     string   // Knowledge statement
    DerivedFrom []string // Source fact IDs
}

type Relationship struct {
    From string // Source entity
    To   string // Target entity
    Type string // Relationship type
}
```

**Write Access**: Context Synthesizer Agent
**Read Access**: All agents

---

### 4. Retrieval

**Purpose**: Information retrieval plans, queries, and artifacts

```go
type RetrievalContext struct {
    Plans     []RetrievalPlan // Retrieval execution plans
    Queries   []Query         // Search queries
    Artifacts []Artifact      // Retrieved artifacts
}
```

**Key Types**:

```go
type RetrievalPlan struct {
    ID          string                 // Unique identifier
    Description string                 // Plan description
    Sources     []string               // Target sources
    Filters     map[string]interface{} // Query filters
    Priority    int                    // Execution priority
}

type Query struct {
    ID          string                 // Unique identifier
    QueryString string                 // Search query
    Source      string                 // Target source
    Filters     map[string]interface{} // Query filters
    Results     interface{}            // Query results
}

type Artifact struct {
    ID      string      // Unique identifier
    Type    string      // Artifact type
    Content interface{} // Artifact data
    Source  string      // Data source
}
```

**Write Access**: Retrieval Planner Agent
**Read Access**: All agents

---

### 5. LLM

**Purpose**: LLM provider info, usage tracking, and selection decisions

```go
type LLMContext struct {
    Provider  string                 // Current provider
    Model     string                 // Current model
    Usage     *LLMUsage              // Token usage and cost
    Decisions []LLMDecision          // Model selection decisions
    Cache     map[string]interface{} // Response cache metadata
}
```

**Key Types**:

```go
type LLMUsage struct {
    TotalTokens      int                // Total tokens used
    PromptTokens     int                // Prompt tokens
    CompletionTokens int                // Completion tokens
    CostUSD          float64            // Total cost in USD
    ByAgent          map[string]float64 // Cost breakdown by agent
}

type LLMDecision struct {
    Timestamp  time.Time // When decision was made
    AgentID    string    // Agent making request
    TaskType   string    // Task complexity type
    Selected   string    // Selected model
    Reason     string    // Selection reason
    Complexity string    // Task complexity level
}
```

**Write Access**: Inference Agent, LLM Orchestrator
**Read Access**: All agents

---

### 6. Diagnostics

**Purpose**: Errors, warnings, performance metrics, validation reports

```go
type DiagnosticsContext struct {
    Errors            []ErrorReport      // Error reports
    Warnings          []Warning          // Warning messages
    Performance       *PerformanceData   // Performance metrics
    ValidationReports []ValidationReport // Validation results
}
```

**Key Types**:

```go
type ErrorReport struct {
    Timestamp time.Time   // When error occurred
    AgentID   string      // Agent that errored
    Message   string      // Error message
    Severity  string      // Error severity
    Details   interface{} // Additional details
}

type PerformanceData struct {
    TotalDurationMS int64                    // Total execution time
    AgentMetrics    map[string]*AgentMetrics // Per-agent metrics
}

type AgentMetrics struct {
    DurationMS int64   // Agent execution time
    LLMCalls   int     // Number of LLM calls
    Status     string  // Execution status
    Tokens     int     // Tokens used (optional)
    Cost       float64 // Cost incurred (optional)
}

type ValidationReport struct {
    Timestamp time.Time // Validation timestamp
    AgentID   string    // Validator agent
    Passed    bool      // Validation passed
    Issues    []string  // Validation issues
    AutoFixes []string  // Suggested fixes
}
```

**Write Access**: All agents (errors/warnings), Validation Agent (reports)
**Read Access**: All agents

---

### 7. Audit

**Purpose**: Complete audit trail of agent executions and context changes

```go
type AuditContext struct {
    AgentRuns []AgentRun    // All agent executions
    Diffs     []ContextDiff // Context change diffs
}
```

**Key Types**:

```go
type AgentRun struct {
    Timestamp   time.Time // Execution start time
    AgentID     string    // Agent identifier
    Status      string    // Execution status
    DurationMS  int64     // Execution duration
    KeysWritten []string  // Context keys modified
    Error       string    // Error message (if failed)
}

type ContextDiff struct {
    Timestamp time.Time              // Diff timestamp
    AgentID   string                 // Agent that made changes
    Changes   map[string]interface{} // Changed data by namespace
}
```

**Write Access**: All agents (via audit system)
**Read Access**: All agents

## Access Control

### Agent Permissions

Default permissions for standard agents:

| Agent                | Allowed Namespaces                          |
|----------------------|---------------------------------------------|
| intent_detection     | reasoning, diagnostics, audit               |
| reasoning_structure  | reasoning, diagnostics, audit               |
| retrieval_planner    | retrieval, diagnostics, audit               |
| context_synthesizer  | enrichment, diagnostics, audit              |
| inference            | reasoning, enrichment, llm, diagnostics, audit |
| validation           | diagnostics, audit                          |
| summarization        | reasoning, diagnostics, audit               |
| orchestrator         | * (all namespaces)                          |

### Permission Enforcement

```go
// Register agent permissions
validator := models.NewContextValidator()
validator.RegisterAgent("intent_detection", []string{"reasoning", "diagnostics", "audit"})

// Validate write access
err := validator.ValidateWrite("intent_detection", "reasoning", "intents")
// Returns: nil (allowed)

err := validator.ValidateWrite("intent_detection", "llm", "provider")
// Returns: ContextViolationError (not allowed)
```

### Safe Writes

Use `SafeSet` for validated writes:

```go
validator := models.NewContextValidator()
validator.RegisterAgent("intent_detection", []string{"reasoning"})

ctx := models.NewAgentContext("session-1", "trace-1")

// Safe write
intents := []models.Intent{{Type: "query", Confidence: 0.95}}
err := validator.SafeSet(ctx, "intent_detection", "reasoning", "intents", intents)
// Success: intents written to context
```

## Size Management

### Default Limits

```go
limits := models.DefaultContextLimits()
// MaxTotalSize: 10 MB
// MaxNamespaceSize: 2 MB
// MaxArrayItems: 1000
// ArtifactExternalizationThreshold: 100 KB
// MaxInlineArtifactSize: 50 KB
```

### Size Checking

```go
checker := models.NewContextSizeChecker(limits)

// Validate context size
err := checker.Check(ctx)
if err != nil {
    // Handle size violation
}

// Check if artifacts should be externalized
if checker.ShouldExternalizeArtifacts(ctx) {
    externalizer := NewS3ArtifactExternalizer()
    models.ExternalizeArtifacts(ctx, externalizer, limits)
}
```

### Context Statistics

```go
stats, err := models.GetStats(ctx)

fmt.Printf("Total size: %d bytes\n", stats.TotalSize)
fmt.Printf("Reasoning size: %d bytes\n", stats.NamespaceSizes["reasoning"])
fmt.Printf("Intents count: %d\n", stats.ArrayCounts["intents"])
fmt.Printf("Externalized artifacts: %d\n", stats.ExternalizedCount)
```

## Change Tracking

### Diff Tracking

```go
tracker, _ := models.NewDiffTracker(ctx)

// Agent modifies context
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query", Confidence: 0.95},
}

// Capture changes
diff, _ := tracker.Capture("intent_detection", ctx)

// View changes
fmt.Println(diff.Summary())
// Output: [intent_detection] Changes: reasoning.intents_added=1
```

### Audit Trail

```go
// Add agent run to audit
ctx.Audit.AgentRuns = append(ctx.Audit.AgentRuns, models.AgentRun{
    Timestamp:   time.Now(),
    AgentID:     "intent_detection",
    Status:      "success",
    DurationMS:  120,
    KeysWritten: []string{"reasoning.intents", "reasoning.entities"},
})

// Add diff to audit
ctx.Audit.Diffs = append(ctx.Audit.Diffs, *diff)
```

## Serialization

### JSON Serialization

```go
// Serialize to JSON
data, err := ctx.Serialize()

// Deserialize from JSON
restored, err := models.Deserialize(data)
```

### Cloning

```go
// Create deep copy
clone, err := ctx.Clone()

// Modify clone without affecting original
clone.Reasoning.Summary = "modified"
```

## Best Practices

### 1. Use Namespace Isolation

```go
// ✓ Good: Agent writes to its designated namespace
validator.SafeSet(ctx, "intent_detection", "reasoning", "intents", intents)

// ✗ Bad: Direct write bypasses validation
ctx.Reasoning.Intents = intents
```

### 2. Track All Changes

```go
// Always capture diffs after agent execution
diff, err := tracker.Capture(agentID, ctx)
if err == nil {
    ctx.Audit.Diffs = append(ctx.Audit.Diffs, *diff)
}
```

### 3. Monitor Context Size

```go
// Check size periodically
if checker.ShouldExternalizeArtifacts(ctx) {
    models.ExternalizeArtifacts(ctx, externalizer, limits)
}

stats, _ := models.GetStats(ctx)
if stats.TotalSize > 8*1024*1024 { // 8 MB warning threshold
    logger.Warn("Context approaching size limit")
}
```

### 4. Preserve Provenance

```go
// Always track source of facts
fact := models.Fact{
    ID:      "f1",
    Content: "User committed 5 times",
    Source:  "gitlab",
    Timestamp: time.Now(),
    Confidence: 1.0,
    Provenance: map[string]interface{}{
        "query": "commits by user X",
        "project": "project-123",
    },
}
```

### 5. Use Confidence Scores

```go
// Track confidence for intents
intent := models.Intent{
    Type:       "query",
    Confidence: 0.95,
}

// Track confidence for conclusions
conclusion := models.Conclusion{
    ID:         "c1",
    Content:    "User is active contributor",
    Confidence: 0.88,
    BasedOn:    []string{"f1", "f2", "f3"},
}
```

## Error Handling

### Common Errors

```go
// ContextViolationError: Agent accessing unauthorized namespace
_, err := validator.SafeSet(ctx, "intent_detection", "llm", "provider", "openai")
if err, ok := err.(*models.ContextViolationError); ok {
    fmt.Printf("Violation: %s tried to write %s.%s\n", err.AgentID, err.Namespace, err.Key)
}

// ContextSizeError: Size limit exceeded
err = checker.Check(ctx)
if err, ok := err.(*models.ContextSizeError); ok {
    fmt.Printf("Size limit exceeded: %s (current: %d, max: %d)\n",
        err.Limit, err.Current, err.Maximum)
}
```

## Migration Strategy

### Version Handling

```go
// Check version before processing
if ctx.Version != "1.0.0" {
    migrated, err := MigrateContext(ctx, "1.0.0")
    if err != nil {
        return err
    }
    ctx = migrated
}
```

### Schema Evolution

When adding new fields:

1. Add field to struct with `json:"field_name,omitempty"` tag
2. Update version number in `NewAgentContext()`
3. Write migration function for old versions
4. Update validation in `context_validator.go`
5. Update tests

## Performance Considerations

### Memory Usage

- **Typical context**: 10-100 KB
- **Large context**: 1-2 MB
- **Maximum**: 10 MB (hard limit)

### Optimization Tips

1. **Externalize large artifacts** early (>50 KB)
2. **Limit array sizes** (e.g., keep only recent 100 diffs)
3. **Use pointers** for large structs to avoid copies
4. **Serialize infrequently** (only when needed)

### Benchmarks

```
Operation              Time       Memory
--------------------  ---------  --------
NewAgentContext       ~10 µs     ~5 KB
Clone                 ~100 µs    2x size
Serialize (JSON)      ~500 µs    1.5x size
Deserialize (JSON)    ~800 µs    2x size
Diff Capture          ~50 µs     ~1 KB
Size Check            ~200 µs    0 KB
```

## Examples

See `src/golang/internal/domain/models/*_test.go` for comprehensive examples of all operations.
