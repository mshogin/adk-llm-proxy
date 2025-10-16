# Validation Agent

## Overview

The **Validation Agent** validates completeness and consistency of the reasoning pipeline. It performs comprehensive checks on intents, hypotheses, conclusions, facts, and their relationships to ensure the reasoning process is sound and complete. The agent generates validation reports with actionable auto-fix hints for common issues.

**Agent ID**: `validation`

**Version**: 1.0.0

**Type**: Rule-based Quality Assurance Agent

**Dependencies**: Inference Agent

## Key Characteristics

- **Zero LLM Calls**: Pure rule-based validation logic
- **Deterministic**: Same inputs produce same validation results
- **Fast**: ~100ms execution time
- **Comprehensive**: 5 validation checks covering all reasoning components
- **Non-Blocking**: Generates warnings/errors but doesn't block execution
- **Actionable**: Auto-fix hints for every issue detected
- **Traceability**: Detailed validation reports with issue descriptions

## Agent Contract

### Preconditions

- **`reasoning.intents`**: Array of detected intents
- **`reasoning.hypotheses`**: Reasoning structure with dependencies
- **`reasoning.conclusions`**: Generated conclusions

### Postconditions

- **`diagnostics.validation_reports`**: Array of validation reports (one per check)
- **`diagnostics.errors`**: Critical validation errors
- **`diagnostics.warnings`**: Non-critical validation warnings

### Capabilities

```go
SupportsParallelExecution: false  // Must run after inference
SupportsRetry:             true   // Safe to retry on failure
RequiresLLM:               false  // Rule-based validation
IsDeterministic:           true   // Same input → same output
EstimatedDuration:         100ms  // ~100ms for validation
```

## Validation Checks

The Validation Agent performs 5 comprehensive checks:

### 1. Intent Completeness

**Purpose**: Ensure all intents have corresponding conclusions

**Check Logic**:
- Map conclusions by intent type
- Verify each intent has at least one conclusion

**Issue Detected**: `Intent 'query_issues' has no conclusions`

**Auto-Fix Hint**: `Ensure Inference Agent generates conclusions for 'query_issues' intent`

**Severity**: Warning (non-critical, but indicates incomplete reasoning)

---

### 2. Hypothesis Consistency

**Purpose**: Validate hypothesis dependencies form a valid structure

**Check Logic**:
- Build hypothesis ID map
- Check each hypothesis dependency exists

**Issue Detected**: `Hypothesis 'h1' depends on missing hypothesis 'h999'`

**Auto-Fix Hint**: `Add missing hypothesis 'h999' or remove dependency from 'h1'`

**Severity**: Error (critical, breaks reasoning structure)

---

### 3. Dependency Cycle Detection

**Purpose**: Detect circular dependencies in hypothesis graph

**Check Logic**:
- Build adjacency list from hypothesis dependencies
- Perform DFS to detect cycles
- Extract cycle path when found

**Issue Detected**: `Dependency cycle detected: h0 → h1 → h2 → h0`

**Auto-Fix Hint**: `Break cycle by removing dependency from 'h2' to 'h0'`

**Severity**: Error (critical, prevents topological execution)

**Algorithm**: Depth-First Search (DFS) with recursion stack tracking

---

### 4. Conclusion Evidence

**Purpose**: Validate conclusions have supporting evidence

**Check Logic**:
- Check each conclusion has evidence references
- Verify evidence references exist in facts/knowledge

**Issues Detected**:
- `Conclusion 'c0' has no supporting evidence`
- `Conclusion 'c0' references non-existent evidence 'fact:f999'`

**Auto-Fix Hints**:
- `Link conclusion 'c0' to relevant facts or knowledge items`
- `Remove invalid evidence reference 'fact:f999' from conclusion 'c0'`

**Severity**:
- No evidence: Warning (conclusion may be weak)
- Invalid reference: Error (data integrity issue)

---

### 5. Fact Provenance

**Purpose**: Ensure facts have complete provenance metadata

**Check Logic**:
- Check each fact has provenance information
- Validate confidence scores are in range [0.0, 1.0]

**Issues Detected**:
- `Fact 'f1' missing provenance information`
- `Fact 'f1' has invalid confidence score 1.50 (must be 0.0-1.0)`

**Auto-Fix Hints**:
- `Add provenance metadata (artifact_id, source) to fact 'f1'`
- `Normalize confidence score for fact 'f1' to valid range [0.0, 1.0]`

**Severity**:
- Missing provenance: Warning (traceability issue)
- Invalid confidence: Error (data validity issue)

## Output Schema

### Validation Report Structure

```go
type ValidationReport struct {
    Timestamp time.Time // When validation was performed
    AgentID   string    // "validation"
    Passed    bool      // true if check passed
    Issues    []string  // List of issues found
    AutoFixes []string  // Suggested fixes for issues
}
```

### Full Output Example

```json
{
  "diagnostics": {
    "validation_reports": [
      {
        "timestamp": "2025-01-15T10:30:00Z",
        "agent_id": "validation",
        "passed": false,
        "issues": [
          "Intent 'query_issues' has no conclusions"
        ],
        "auto_fixes": [
          "Ensure Inference Agent generates conclusions for 'query_issues' intent"
        ]
      },
      {
        "timestamp": "2025-01-15T10:30:00Z",
        "agent_id": "validation",
        "passed": false,
        "issues": [
          "Hypothesis 'h1' depends on missing hypothesis 'h999'"
        ],
        "auto_fixes": [
          "Add missing hypothesis 'h999' or remove dependency from 'h1'"
        ]
      }
    ],
    "errors": [
      {
        "timestamp": "2025-01-15T10:30:00Z",
        "agent_id": "validation",
        "message": "Hypothesis 'h1' depends on missing hypothesis 'h999'",
        "severity": "error"
      }
    ],
    "warnings": [
      {
        "timestamp": "2025-01-15T10:30:00Z",
        "agent_id": "validation",
        "message": "Intent 'query_issues' has no conclusions"
      }
    ]
  }
}
```

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| **Execution Time** | ~100ms | Rule-based, minimal I/O |
| **LLM Calls** | 0 | Fully deterministic |
| **Memory Usage** | <2MB | Lightweight processing |
| **Checks Performed** | 5 | Comprehensive coverage |
| **Idempotent** | Yes | Same input → same output |

### Scalability

| Input Size | Execution Time | Memory Usage |
|-----------|---------------|--------------|
| 10 hypotheses | ~50ms | <1MB |
| 100 hypotheses | ~100ms | ~1MB |
| 1000 hypotheses | ~500ms | ~5MB |

## Usage Examples

### Example 1: All Checks Pass

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
}
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {ID: "h0", Description: "Retrieve commits"},
}
ctx.Reasoning.Conclusions = []models.Conclusion{
    {
        ID: "c0",
        Intent: "query_commits",
        Evidence: []string{"fact:f1"},
    },
}
ctx.Enrichment.Facts = []models.Fact{
    {
        ID: "f1",
        Confidence: 0.95,
        Provenance: map[string]interface{}{"source": "gitlab"},
    },
}

agent := agents.NewValidationAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// All 5 validation reports pass
for _, report := range result.Diagnostics.ValidationReports {
    assert.True(report.Passed)
    assert.Empty(report.Issues)
}

// No errors or warnings
assert.Empty(result.Diagnostics.Errors)
assert.Empty(result.Diagnostics.Warnings)
```

---

### Example 2: Missing Conclusion for Intent

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits"},
    {Type: "query_issues"}, // No conclusion for this intent
}
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {ID: "h0"},
}
ctx.Reasoning.Conclusions = []models.Conclusion{
    {ID: "c0", Intent: "query_commits"},
    // Missing conclusion for query_issues
}
```

**Output**:
```go
// Intent completeness check fails
report := result.Diagnostics.ValidationReports[0]
assert.False(report.Passed)
assert.Contains(report.Issues[0], "query_issues")
assert.Contains(report.AutoFixes[0], "Inference Agent")

// Warning generated
assert.Len(result.Diagnostics.Warnings, 1)
assert.Contains(result.Diagnostics.Warnings[0].Message, "query_issues")
```

---

### Example 3: Dependency Cycle

**Input**:
```go
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {ID: "h0", Dependencies: []string{"h1"}},
    {ID: "h1", Dependencies: []string{"h2"}},
    {ID: "h2", Dependencies: []string{"h0"}}, // Cycle: h0 → h1 → h2 → h0
}
```

**Output**:
```go
// Dependency cycle check fails
report := result.Diagnostics.ValidationReports[2] // Third check
assert.False(report.Passed)
assert.Contains(report.Issues[0], "cycle")
assert.Contains(report.Issues[0], "h0 → h1 → h2 → h0")
assert.Contains(report.AutoFixes[0], "Break cycle")

// Error generated
assert.Len(result.Diagnostics.Errors, 1)
assert.Equal("error", result.Diagnostics.Errors[0].Severity)
```

---

### Example 4: Invalid Confidence Score

**Input**:
```go
ctx.Enrichment.Facts = []models.Fact{
    {
        ID: "f1",
        Confidence: 1.5, // Invalid (>1.0)
        Provenance: map[string]interface{}{"source": "test"},
    },
}
```

**Output**:
```go
// Fact provenance check fails
report := result.Diagnostics.ValidationReports[4] // Fifth check
assert.False(report.Passed)
assert.Contains(report.Issues[0], "invalid confidence")
assert.Contains(report.AutoFixes[0], "Normalize confidence")

// Error generated
assert.Len(result.Diagnostics.Errors, 1)
```

## Testing Coverage

### Unit Tests (23+ tests)

1. **Agent Initialization**: Metadata, capabilities, contract
2. **Precondition Validation**: Missing intents, hypotheses, conclusions
3. **Intent Completeness**: Missing conclusions, complete coverage
4. **Hypothesis Consistency**: Missing dependencies, valid structure
5. **Dependency Cycles**: Cycle detection, no cycles, DFS algorithm
6. **Conclusion Evidence**: Missing evidence, invalid references, valid evidence
7. **Fact Provenance**: Missing provenance, invalid confidence, valid facts
8. **Evidence Existence**: Fact/knowledge reference validation
9. **Context Isolation**: Input not modified
10. **Audit Trail**: Execution tracking, metrics
11. **Idempotency**: Same input produces same output
12. **Multiple Failures**: Handling multiple simultaneous issues

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| **"no intents found"** | Empty intents array | Ensure Intent Detection Agent produced intents |
| **"no hypotheses found"** | Empty hypotheses array | Ensure Reasoning Structure Agent produced hypotheses |
| **"no conclusions found"** | Empty conclusions array | Ensure Inference Agent produced conclusions |
| **"reasoning context is nil"** | Missing reasoning context | Ensure reasoning agents ran first |

### Error Examples

```go
// Missing preconditions
ctx.Reasoning.Intents = []
→ Error: "no intents found (required: reasoning.intents)"

ctx.Reasoning.Hypotheses = []
→ Error: "no hypotheses found (required: reasoning.hypotheses)"

ctx.Reasoning.Conclusions = []
→ Error: "no conclusions found (required: reasoning.conclusions)"
```

## Validation Severity Levels

| Severity | Description | Action Required |
|----------|-------------|-----------------|
| **Error** | Critical issue that breaks reasoning integrity | Fix immediately; may cause downstream failures |
| **Warning** | Non-critical issue that may affect reasoning quality | Review and fix when possible; execution continues |

**Error Examples**:
- Missing hypothesis dependency
- Dependency cycle
- Invalid evidence reference
- Invalid confidence score

**Warning Examples**:
- Missing conclusion for intent
- Conclusion has no evidence
- Missing fact provenance

## Auto-Fix Patterns

### Pattern 1: Add Missing Resource

**Issue**: Hypothesis depends on non-existent hypothesis
**Auto-Fix**: `Add missing hypothesis 'h999' or remove dependency from 'h1'`

### Pattern 2: Remove Invalid Reference

**Issue**: Conclusion references non-existent evidence
**Auto-Fix**: `Remove invalid evidence reference 'fact:f999' from conclusion 'c0'`

### Pattern 3: Break Cycle

**Issue**: Circular dependency detected
**Auto-Fix**: `Break cycle by removing dependency from 'h2' to 'h0'`

### Pattern 4: Normalize Value

**Issue**: Confidence score out of range
**Auto-Fix**: `Normalize confidence score for fact 'f1' to valid range [0.0, 1.0]`

### Pattern 5: Add Metadata

**Issue**: Missing provenance information
**Auto-Fix**: `Add provenance metadata (artifact_id, source) to fact 'f1'`

## Integration with Pipeline

The Validation Agent fits into the reasoning pipeline as a quality assurance checkpoint:

```
Intent Detection → Reasoning Structure → Retrieval Planner
→ Context Synthesizer → Inference → **Validation** → Summarization
```

**Execution Mode**: Sequential (runs after Inference Agent)

**Failure Handling**: Non-blocking (generates reports but doesn't halt pipeline)

**Output Usage**: Diagnostics consumed by Summarization Agent and returned to user

## Future Enhancements

1. **Semantic Validation**: Check logical consistency of reasoning chains
2. **Temporal Validation**: Verify time-based ordering of facts
3. **Cross-Agent Validation**: Validate data flow between agents
4. **Statistical Validation**: Detect outliers in confidence scores
5. **Auto-Repair**: Automatically fix common issues
6. **Severity Scoring**: Quantitative risk assessment per issue
7. **Custom Validation Rules**: User-defined validation logic

## Troubleshooting

### Issue: "Too many validation warnings"

**Cause**: Reasoning pipeline producing incomplete/inconsistent data

**Solution**: Review earlier agents (Intent Detection, Reasoning Structure, Inference) for issues

---

### Issue: "Validation reports all pass but reasoning seems wrong"

**Cause**: Validation checks structural integrity, not semantic correctness

**Solution**: Add semantic validation checks or use LLM-based validation

---

### Issue: "False positive validation errors"

**Cause**: Edge case not handled by validation rules

**Solution**: Review validation logic; may need to relax constraints

## Related Documentation

- [Inference Agent](./inference.md)
- [Summarization Agent](./summarization.md)
- [Agent Context Schema](../agent_context_schema.md)
- [Reasoning System Overview](../README.md)

## API Reference

### Constructor

```go
func NewValidationAgent() *ValidationAgent
```

### Execute

```go
func (a *ValidationAgent) Execute(
    ctx context.Context,
    agentContext *models.AgentContext
) (*models.AgentContext, error)
```

**Parameters**:
- `ctx`: Cancellation context
- `agentContext`: Must contain `reasoning.intents`, `reasoning.hypotheses`, `reasoning.conclusions`

**Returns**: Updated context with `diagnostics.validation_reports`, `diagnostics.errors`, `diagnostics.warnings`

### Internal Methods

```go
// Validate intent completeness
func (a *ValidationAgent) validateIntentCompleteness(
    ctx *models.AgentContext
) (models.ValidationReport, []models.ErrorReport, []models.Warning)

// Validate hypothesis consistency
func (a *ValidationAgent) validateHypothesisConsistency(
    ctx *models.AgentContext
) (models.ValidationReport, []models.ErrorReport, []models.Warning)

// Detect dependency cycles
func (a *ValidationAgent) detectDependencyCycles(
    ctx *models.AgentContext
) (models.ValidationReport, []models.ErrorReport, []models.Warning)

// Validate conclusion evidence
func (a *ValidationAgent) validateConclusionEvidence(
    ctx *models.AgentContext
) (models.ValidationReport, []models.ErrorReport, []models.Warning)

// Validate fact provenance
func (a *ValidationAgent) validateFactProvenance(
    ctx *models.AgentContext
) (models.ValidationReport, []models.ErrorReport, []models.Warning)

// Check if evidence exists
func (a *ValidationAgent) evidenceExists(
    evidenceRef string,
    ctx *models.AgentContext
) bool

// DFS cycle detection
func (a *ValidationAgent) detectCycleDFS(
    node string,
    graph map[string][]string,
    visited, recStack map[string]bool,
    path []string
) []string
```

## Changelog

### Version 1.0.0 (2025-01-15)

**Initial Release**:
- 5 comprehensive validation checks (intent, hypothesis, cycle, evidence, provenance)
- Auto-fix hints for all detected issues
- Error/warning severity classification
- Cycle detection using DFS algorithm
- Evidence reference validation
- Confidence score range validation
- Full test coverage (23+ unit tests)
- Zero LLM calls for fast, deterministic validation
- Non-blocking execution (generates reports without halting pipeline)
