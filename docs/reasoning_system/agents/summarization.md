# Summarization Agent

## Overview

The **Summarization Agent** generates executive summaries and structured output artifacts from the reasoning process. It synthesizes all outputs from previous agents (intents, hypotheses, conclusions, facts, validation results) into concise summaries, detailed reports, command lists, and context diffs for downstream consumption.

**Agent ID**: `summarization`

**Version**: 1.0.0

**Type**: Rule-based Output Formatting Agent

**Dependencies**: Inference Agent, Validation Agent

## Key Characteristics

- **Zero LLM Calls**: Pure rule-based summarization
- **Deterministic**: Same inputs produce same outputs
- **Fast**: ~50ms execution time
- **Multi-Format**: Generates summaries, reports, commands, diffs
- **Executive-Level**: Concise high-level overviews
- **Structured Artifacts**: Machine-readable output formats
- **Complete Traceability**: Context diff showing all agent changes

## Agent Contract

### Preconditions

- **`reasoning.intents`**: Array of detected intents
- **`reasoning.conclusions`**: Generated conclusions with confidence scores

### Postconditions

- **`reasoning.summary`**: Executive summary string (1-3 sentences)
- **`reasoning.artifacts`**: Array of structured output artifacts (reports, commands, diffs)

### Capabilities

```go
SupportsParallelExecution: false  // Must run after all reasoning agents
SupportsRetry:             true   // Safe to retry on failure
RequiresLLM:               false  // Rule-based summarization
IsDeterministic:           true   // Same input → same output
EstimatedDuration:         50ms   // ~50ms for summarization
```

## Summarization Pipeline

The Summarization Agent performs the following steps in order:

1. **Generate Executive Summary** → Concise 1-3 sentence overview
2. **Generate Structured Report** → Detailed markdown report with sections
3. **Generate Command List** → Executable commands (if applicable)
4. **Generate Context Diff** → Overview of all agent changes
5. **Write Results** → Populate reasoning context with summary and artifacts

```
Reasoning Context → Executive Summary + Report + Commands + Diff → Output Artifacts
```

## Executive Summary Generation

### Summary Components

The executive summary consists of 4 parts:

1. **Intent Summary**: Which intents were detected
2. **Data Summary**: How much data was retrieved and from where
3. **Conclusion Summary**: How many conclusions, average confidence
4. **Validation Summary**: Whether validation passed or had issues

**Example Summary**:
```
Detected intent: query_commits. Retrieved 3 fact(s) from gitlab.
Generated 1 conclusion with high confidence (0.95). Validation passed with no issues.
```

### Intent Summarization

| Scenario | Summary Format | Example |
|----------|----------------|---------|
| **Single Intent** | "Detected intent: {type}." | "Detected intent: query_commits." |
| **Multiple Intents** | "Detected {N} intents: {types}." | "Detected 2 intents: query_commits, query_issues." |

### Data Summarization

| Scenario | Summary Format | Example |
|----------|----------------|---------|
| **No Data** | "No data retrieved." | "No data retrieved." |
| **Single Source** | "Retrieved {N} fact(s) from {source}." | "Retrieved 5 fact(s) from gitlab." |
| **Multiple Sources** | "Retrieved {N} fact(s) ({breakdown})." | "Retrieved 8 fact(s) (5 from gitlab, 3 from youtrack)." |

### Conclusion Summarization

**Confidence Levels**:

| Average Confidence | Level | Description |
|-------------------|-------|-------------|
| **≥ 0.90** | high | Strong evidence, reliable conclusions |
| **≥ 0.70** | good | Solid evidence, trustworthy conclusions |
| **≥ 0.50** | moderate | Some evidence, reasonable conclusions |
| **< 0.50** | low | Weak evidence, uncertain conclusions |

**Examples**:
```
Generated 1 conclusion with high confidence (0.95).
Generated 3 conclusions with good average confidence (0.82).
Generated 2 conclusions with moderate average confidence (0.65).
No conclusions generated.
```

### Validation Summarization

| Scenario | Summary Format | Example |
|----------|----------------|---------|
| **No Issues** | "Validation passed with no issues." | "Validation passed with no issues." |
| **Errors Only** | "Validation detected {N} error(s)." | "Validation detected 2 error(s)." |
| **Warnings Only** | "Validation detected {N} warning(s)." | "Validation detected 1 warning(s)." |
| **Both** | "Validation detected {N} error(s) and {M} warning(s)." | "Validation detected 1 error(s) and 2 warning(s)." |

## Structured Artifacts

### Artifact Types

The agent generates 1-3 artifacts depending on context:

| Artifact Type | ID | Always Present | Description |
|--------------|-----|----------------|-------------|
| **Report** | `report` | ✓ Yes | Detailed markdown report with sections |
| **Command List** | `commands` | ✗ Conditional | Executable commands (if conclusions suggest actions) |
| **Context Diff** | `context_diff` | ✗ Conditional | Agent execution trail (if audit available) |

### Report Artifact

**Structure**:
```markdown
# Reasoning Report

## Detected Intents
- **query_commits** (confidence: 0.90)

## Retrieved Data
- **gitlab**: 3 fact(s)

## Conclusions
- Found 3 commit(s) from GitLab (confidence: 0.95)

## Validation Results
### Errors
- (none)

### Warnings
- (none)
```

**Sections** (conditional based on available data):
1. **Detected Intents**: List of intents with confidence scores
2. **Retrieved Data**: Fact counts by source
3. **Conclusions**: Conclusion descriptions with confidence
4. **Validation Results**: Errors and warnings (if any)

### Command List Artifact

**Generated When**: Conclusions contain action keywords ("commit", "issue", etc.)

**Format**: Newline-separated list of executable commands

**Example**:
```
git log
youtrack issues list
```

**Command Mapping**:
| Conclusion Keyword | Generated Command |
|-------------------|------------------|
| `commit` | `git log` |
| `issue` | `youtrack issues list` |

### Context Diff Artifact

**Generated When**: Audit trail available (`audit.agent_runs` not empty)

**Format**: List of agents and the context keys they wrote

**Example**:
```
Context Changes:
- intent_detection: reasoning.intents
- reasoning_structure: reasoning.hypotheses
- retrieval_planner: retrieval.plans, retrieval.queries
- context_synthesizer: enrichment.facts, enrichment.derived_knowledge
- inference: reasoning.conclusions, reasoning.inference_chain
- validation: diagnostics.validation_reports, diagnostics.errors
```

## Output Schema

### Full Output Example

```json
{
  "reasoning": {
    "summary": "Detected intent: query_commits. Retrieved 3 fact(s) from gitlab. Generated 1 conclusion with high confidence (0.95). Validation passed with no issues.",
    "artifacts": [
      {
        "id": "report",
        "type": "report",
        "source": "summarization",
        "content": "# Reasoning Report\n\n## Detected Intents\n- **query_commits** (confidence: 0.90)\n..."
      },
      {
        "id": "commands",
        "type": "command_list",
        "source": "summarization",
        "content": "git log"
      },
      {
        "id": "context_diff",
        "type": "diff",
        "source": "summarization",
        "content": "Context Changes:\n- intent_detection: reasoning.intents\n- inference: reasoning.conclusions"
      }
    ]
  }
}
```

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| **Execution Time** | ~50ms | Rule-based, minimal I/O |
| **LLM Calls** | 0 | Fully deterministic |
| **Memory Usage** | <1MB | Lightweight string processing |
| **Artifacts Generated** | 1-3 | Report always, commands/diff conditional |
| **Summary Length** | 1-3 sentences | Executive-level brevity |
| **Idempotent** | Yes | Same input → same output |

### Scalability

| Input Size | Execution Time | Memory Usage |
|-----------|---------------|--------------|
| 10 conclusions, 10 facts | ~30ms | <1MB |
| 50 conclusions, 100 facts | ~50ms | ~1MB |
| 200 conclusions, 500 facts | ~100ms | ~2MB |

## Usage Examples

### Example 1: Simple Query with Single Source

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
}
ctx.Reasoning.Conclusions = []models.Conclusion{
    {
        ID:          "c0",
        Description: "Found 3 commit(s) from GitLab",
        Confidence:  0.95,
        Intent:      "query_commits",
    },
}
ctx.Enrichment.Facts = []models.Fact{
    {ID: "f1", Source: "gitlab"},
    {ID: "f2", Source: "gitlab"},
    {ID: "f3", Source: "gitlab"},
}

agent := agents.NewSummarizationAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
result.Reasoning.Summary =
    "Detected intent: query_commits. " +
    "Retrieved 3 fact(s) from gitlab. " +
    "Generated 1 conclusion with high confidence (0.95). " +
    "Validation passed with no issues."

result.Reasoning.Artifacts = [
    {
        ID:      "report",
        Type:    "report",
        Source:  "summarization",
        Content: "# Reasoning Report\n\n## Detected Intents\n...",
    },
    {
        ID:      "commands",
        Type:    "command_list",
        Source:  "summarization",
        Content: "git log",
    },
]
```

---

### Example 2: Multiple Intents and Sources

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
    {Type: "query_issues", Confidence: 0.8},
}
ctx.Reasoning.Conclusions = []models.Conclusion{
    {Description: "Found 3 commit(s) from GitLab", Confidence: 0.95},
    {Description: "Found 2 issue(s) from YouTrack", Confidence: 0.90},
}
ctx.Enrichment.Facts = []models.Fact{
    {Source: "gitlab"},
    {Source: "gitlab"},
    {Source: "gitlab"},
    {Source: "youtrack"},
    {Source: "youtrack"},
}

agent := agents.NewSummarizationAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
result.Reasoning.Summary =
    "Detected 2 intents: query_commits, query_issues. " +
    "Retrieved 5 fact(s) (3 from gitlab, 2 from youtrack). " +
    "Generated 2 conclusions with high average confidence (0.93). " +
    "Validation passed with no issues."
```

---

### Example 3: With Validation Errors

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
}
ctx.Reasoning.Conclusions = []models.Conclusion{
    {Description: "Found commits", Confidence: 0.60},
}
ctx.Diagnostics.Errors = []models.ErrorReport{
    {Message: "Missing hypothesis dependency"},
}
ctx.Diagnostics.Warnings = []models.Warning{
    {Message: "Low confidence conclusion"},
}

agent := agents.NewSummarizationAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
result.Reasoning.Summary =
    "Detected intent: query_commits. " +
    "No data retrieved. " +
    "Generated 1 conclusion with moderate confidence (0.60). " +
    "Validation detected 1 error(s) and 1 warning(s)."

// Report includes validation section
result.Reasoning.Artifacts[0].Content contains:
    "## Validation Results\n" +
    "### Errors\n" +
    "- Missing hypothesis dependency\n" +
    "### Warnings\n" +
    "- Low confidence conclusion"
```

---

### Example 4: With Audit Trail (Context Diff)

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
}
ctx.Reasoning.Conclusions = []models.Conclusion{
    {Description: "Found commits", Confidence: 0.95},
}
ctx.Audit.AgentRuns = []models.AgentRun{
    {AgentID: "intent_detection", KeysWritten: []string{"reasoning.intents"}},
    {AgentID: "inference", KeysWritten: []string{"reasoning.conclusions"}},
}

agent := agents.NewSummarizationAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// Context diff artifact generated
result.Reasoning.Artifacts contains:
    {
        ID:      "context_diff",
        Type:    "diff",
        Source:  "summarization",
        Content: "Context Changes:\n" +
                 "- intent_detection: reasoning.intents\n" +
                 "- inference: reasoning.conclusions",
    }
```

## Testing Coverage

### Unit Tests (35+ tests)

1. **Agent Initialization**: Metadata, capabilities, contract
2. **Precondition Validation**: Missing intents, conclusions, nil reasoning
3. **Intent Summarization**: Single intent, multiple intents
4. **Data Summarization**: Single source, multiple sources, no data
5. **Conclusion Summarization**: High/good/moderate/low confidence levels
6. **Validation Summarization**: No issues, errors, warnings, both
7. **Report Generation**: Basic report, report with validation
8. **Command List Generation**: Commit commands, issue commands, no commands
9. **Context Diff Generation**: With audit, without audit
10. **Artifact Generation**: All artifact types, conditional artifacts
11. **Context Isolation**: Input not modified
12. **Audit Trail**: Execution tracking, metrics
13. **Idempotency**: Same input produces same output
14. **Performance**: Benchmark < 100ms execution time
15. **Full Pipeline**: Complete output validation

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| **"no intents found"** | Empty intents array | Ensure Intent Detection Agent produced intents |
| **"no conclusions found"** | Empty conclusions array | Ensure Inference Agent produced conclusions |
| **"reasoning context is nil"** | Missing reasoning context | Ensure reasoning agents ran first |

### Error Examples

```go
// Missing intents
ctx.Reasoning.Intents = []
→ Error: "no intents found (required: reasoning.intents)"

// Missing conclusions
ctx.Reasoning.Conclusions = []
→ Error: "no conclusions found (required: reasoning.conclusions)"

// Nil reasoning context
ctx.Reasoning = nil
→ Error: "reasoning context is nil"
```

## Future Enhancements

1. **Multi-Format Output**: JSON, XML, YAML output formats
2. **Template Engine**: Customizable report templates
3. **Localization**: Multi-language summaries
4. **Natural Language Generation**: LLM-based narrative summaries
5. **Visual Artifacts**: Graphs, charts, diagrams
6. **Executive Briefings**: Slide-deck style summaries
7. **Actionable Insights**: Automated recommendations
8. **Trend Analysis**: Compare with previous sessions

## Troubleshooting

### Issue: "Summary too long"

**Cause**: Too many intents/conclusions generating verbose summary

**Solution**: Implement truncation or prioritization logic

---

### Issue: "Commands not generated"

**Cause**: Conclusion descriptions don't contain action keywords

**Solution**: Review command mapping keywords; expand keyword list

---

### Issue: "Report missing sections"

**Cause**: Optional data (facts, validation) not present in context

**Solution**: Conditional rendering works as designed; populate upstream agents

## Related Documentation

- [Inference Agent](./inference.md)
- [Validation Agent](./validation.md)
- [Agent Context Schema](../agent_context_schema.md)
- [Reasoning System Overview](../README.md)

## API Reference

### Constructor

```go
func NewSummarizationAgent() *SummarizationAgent
```

### Execute

```go
func (a *SummarizationAgent) Execute(
    ctx context.Context,
    agentContext *models.AgentContext
) (*models.AgentContext, error)
```

**Parameters**:
- `ctx`: Cancellation context
- `agentContext`: Must contain `reasoning.intents`, `reasoning.conclusions`

**Returns**: Updated context with `reasoning.summary`, `reasoning.artifacts`

### Internal Methods

```go
// Generate executive summary
func (a *SummarizationAgent) generateExecutiveSummary(
    ctx *models.AgentContext
) string

// Summarize intents
func (a *SummarizationAgent) summarizeIntents(
    intents []models.Intent
) string

// Summarize data
func (a *SummarizationAgent) summarizeData(
    ctx *models.AgentContext
) string

// Summarize conclusions
func (a *SummarizationAgent) summarizeConclusions(
    conclusions []models.Conclusion
) string

// Summarize validation
func (a *SummarizationAgent) summarizeValidation(
    ctx *models.AgentContext
) string

// Generate artifacts
func (a *SummarizationAgent) generateArtifacts(
    ctx *models.AgentContext
) []models.Artifact

// Generate detailed report
func (a *SummarizationAgent) generateReport(
    ctx *models.AgentContext
) string

// Generate command list
func (a *SummarizationAgent) generateCommandList(
    ctx *models.AgentContext
) string

// Generate context diff
func (a *SummarizationAgent) generateContextDiff(
    ctx *models.AgentContext
) string
```

## Changelog

### Version 1.0.0 (2025-01-15)

**Initial Release**:
- Executive summary generation (1-3 sentences)
- Structured report artifact (markdown format)
- Command list generation (conditional, based on conclusions)
- Context diff artifact (audit trail visualization)
- Confidence level classification (high/good/moderate/low)
- Multi-source data summarization
- Validation results summary
- Full test coverage (35+ unit tests)
- Zero LLM calls for fast, deterministic summarization
- <50ms execution time target
