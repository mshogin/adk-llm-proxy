# Inference Agent

## Overview

The **Inference Agent** makes conclusions based on enriched facts, hypotheses, and user intents. It performs rule-based inference to derive actionable conclusions, assess their confidence, generate alternative interpretations for ambiguous cases, and build step-by-step reasoning traces.

**Agent ID**: `inference`

**Version**: 1.0.0

**Type**: Rule-based Inference Agent

**Dependencies**: Context Synthesizer Agent

## Key Characteristics

- **Zero LLM Calls**: Pure rule-based inference logic
- **Deterministic**: Same inputs produce same conclusions
- **Fast**: ~150ms execution time
- **Evidence-Based**: Conclusions grounded in facts and knowledge
- **Confidence Scoring**: Quantitative assessment of conclusion certainty
- **Alternative Generation**: Multiple interpretations for low-confidence cases
- **Traceability**: Complete inference chain from hypothesis to conclusion

## Agent Contract

### Preconditions

- **`reasoning.intents`**: Array of detected intents with confidence scores
- **`reasoning.hypotheses`**: Reasoning structure with dependencies
- **`enrichment.facts`**: Normalized facts with provenance

### Postconditions

- **`reasoning.conclusions`**: Array of conclusions with confidence scores
- **`reasoning.alternatives`**: Alternative interpretations for ambiguous cases
- **`reasoning.inference_chain`**: Step-by-step reasoning trace

### Capabilities

```go
SupportsParallelExecution: false  // Must run after enrichment
SupportsRetry:             true   // Safe to retry on failure
RequiresLLM:               false  // Rule-based inference (future: LLM for complex cases)
IsDeterministic:           true   // Same input → same output
EstimatedDuration:         150ms  // ~150ms for inference
```

## Inference Pipeline

The Inference Agent performs the following steps in order:

1. **Build Inference Chain** → Create step-by-step reasoning trace from hypotheses
2. **Make Conclusions** → Generate conclusions per intent based on evidence
3. **Generate Alternatives** → Create alternative interpretations for low-confidence conclusions
4. **Write Results** → Populate reasoning context with conclusions, alternatives, and chain

```
Intents + Hypotheses + Facts → Inference Chain → Conclusions + Alternatives
```

## Inference Chain Construction

### Chain Structure

Each step in the inference chain represents verification of a hypothesis:

```go
type InferenceStep struct {
    ID          string   // Unique step identifier
    Description string   // Step description
    Hypothesis  string   // Hypothesis being verified
    Evidence    []string // Supporting fact/knowledge references
    Confidence  float64  // Confidence in this step (0-1)
}
```

### Chain Building Process

1. **Iterate Hypotheses**: Process each hypothesis from reasoning structure
2. **Find Evidence**: Search facts and knowledge for supporting evidence
3. **Calculate Confidence**: Assess confidence based on evidence count/quality
4. **Create Step**: Generate inference step linking hypothesis to evidence

**Example**:
```json
{
  "id": "step0",
  "description": "Verify: Retrieve commit data from GitLab",
  "hypothesis": "h0",
  "evidence": ["fact:f1", "fact:f2", "knowledge:k0"],
  "confidence": 0.90
}
```

## Conclusion Generation

### Conclusion Structure

```go
type Conclusion struct {
    ID          string   // Unique conclusion identifier
    Description string   // Human-readable conclusion
    Confidence  float64  // Confidence score (0-1)
    Evidence    []string // Supporting fact/knowledge references
    Intent      string   // Intent this conclusion addresses
}
```

### Conclusion Generation by Intent

**1. query_commits Intent**

Conclusion based on commit facts from GitLab:

```json
{
  "id": "c0",
  "description": "Found 3 commit(s) from GitLab",
  "confidence": 0.95,
  "evidence": ["fact:f1", "fact:f2", "fact:f3", "knowledge:k0"],
  "intent": "query_commits"
}
```

**Confidence**: 0.95 (high, structured data source)

---

**2. query_issues Intent**

Conclusion based on issue facts from YouTrack:

```json
{
  "id": "c0",
  "description": "Found 5 issue(s) from YouTrack",
  "confidence": 0.95,
  "evidence": ["fact:f4", "fact:f5", "knowledge:k1"],
  "intent": "query_issues"
}
```

**Confidence**: 0.95 (high, structured data source)

---

**3. query_analytics Intent**

Multiple conclusions (one per source):

```json
[
  {
    "id": "c0",
    "description": "Aggregated 3 metrics from gitlab",
    "confidence": 0.85,
    "evidence": ["fact:f1", "fact:f2", "fact:f3"],
    "intent": "query_analytics"
  },
  {
    "id": "c1",
    "description": "Aggregated 2 metrics from youtrack",
    "confidence": 0.85,
    "evidence": ["fact:f4", "fact:f5"],
    "intent": "query_analytics"
  }
]
```

**Confidence**: 0.85 (good, aggregated metrics)

---

**4. query_status Intent**

Conclusion based on overall activity level:

```json
{
  "id": "c0",
  "description": "System is active with significant data",
  "confidence": 0.90,
  "evidence": ["fact:f1", "fact:f2", ...],
  "intent": "query_status"
}
```

**Status Classification**:
- **>10 facts**: "System is active with significant data" (confidence 0.90)
- **1-10 facts**: "System has limited activity" (confidence 0.75)
- **0 facts**: "No recent activity detected" (confidence 0.80)

## Confidence Scoring

### Evidence-Based Confidence

Confidence is calculated based on supporting evidence count and quality:

| Evidence Count | Confidence | Interpretation |
|----------------|-----------|----------------|
| **0** | 0.20 | Minimal (no supporting evidence) |
| **1** | 0.50 | Moderate (single evidence source) |
| **2** | 0.70 | Good (multiple evidence sources) |
| **3+** | 0.90 | High (strong evidence base) |

### Evidence Matching Algorithm

1. **Extract Keywords**: Remove stop words, filter short words (<3 chars)
2. **Match Facts**: Check if fact content contains any keywords (case-insensitive)
3. **Match Knowledge**: Check if knowledge content contains any keywords
4. **Aggregate**: Collect all matching fact/knowledge IDs as evidence

**Example**:
```go
Hypothesis: "Retrieve commit data from GitLab"
Keywords: [retrieve, commit, data, gitlab]

Fact 1: "Commit message from gitlab" → MATCH
Fact 2: "Issue description" → NO MATCH
Knowledge 1: "Aggregated 5 facts from gitlab" → MATCH

Evidence: ["fact:f1", "knowledge:k1"]
Confidence: 0.70 (2 evidence sources)
```

## Alternative Interpretations

### Alternative Structure

```go
type Alternative struct {
    ID          string  // Unique alternative ID
    Conclusion  string  // Conclusion ID this is alternative to
    Description string  // Alternative interpretation
    Confidence  float64 // Lower confidence than original
}
```

### Generation Rules

Alternatives are generated for **low-confidence conclusions** (confidence < 0.70):

```json
{
  "id": "alt0",
  "conclusion": "c0",
  "description": "Alternative: No commits found (requires more data)",
  "confidence": 0.40
}
```

**Confidence Adjustment**: `alternative_confidence = original_confidence × 0.8`

**Purpose**:
- Acknowledge uncertainty in low-confidence conclusions
- Suggest that more data may change the conclusion
- Enable LLM to consider multiple interpretations

## Output Schema

### Full Reasoning Output

```json
{
  "reasoning": {
    "conclusions": [
      {
        "id": "c0",
        "description": "Found 3 commit(s) from GitLab",
        "confidence": 0.95,
        "evidence": ["fact:f1", "fact:f2", "fact:f3"],
        "intent": "query_commits"
      }
    ],
    "alternatives": [
      {
        "id": "alt0",
        "conclusion": "c1",
        "description": "Alternative: No issues found (requires more data)",
        "confidence": 0.40
      }
    ],
    "inference_chain": [
      {
        "id": "step0",
        "description": "Verify: Retrieve commit data from GitLab",
        "hypothesis": "h0",
        "evidence": ["fact:f1", "fact:f2"],
        "confidence": 0.90
      }
    ]
  }
}
```

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| **Execution Time** | ~150ms | Rule-based, minimal I/O |
| **LLM Calls** | 0 | Fully deterministic (future: LLM for complex cases) |
| **Memory Usage** | <2MB | Lightweight processing |
| **Conclusions per Intent** | 1-N | Based on intent type and data sources |
| **Idempotent** | Yes | Same input → same output |

### Scalability

| Input Size | Execution Time | Memory Usage |
|-----------|---------------|--------------|
| 10 facts | ~50ms | <1MB |
| 100 facts | ~150ms | ~1MB |
| 1000 facts | ~500ms | ~5MB |

## Usage Examples

### Example 1: Commit Query

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
}
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {ID: "h0", Description: "Retrieve commit data from GitLab"},
}
ctx.Enrichment.Facts = []models.Fact{
    {ID: "f1", Content: "feat: add login", Source: "gitlab", Confidence: 0.95},
    {ID: "f2", Content: "fix: auth bug", Source: "gitlab", Confidence: 0.95},
}

agent := agents.NewInferenceAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// 1 conclusion
result.Reasoning.Conclusions = [
    {
        ID: "c0",
        Description: "Found 2 commit(s) from GitLab",
        Confidence: 0.95,
        Evidence: ["fact:f1", "fact:f2"],
        Intent: "query_commits",
    },
]

// 1 inference step
result.Reasoning.InferenceChain = [
    {
        ID: "step0",
        Description: "Verify: Retrieve commit data from GitLab",
        Hypothesis: "h0",
        Evidence: ["fact:f1", "fact:f2"],
        Confidence: 0.70,
    },
]

// 0 alternatives (high confidence)
result.Reasoning.Alternatives = []
```

---

### Example 2: Multiple Intents with Low Confidence

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.9},
    {Type: "query_issues", Confidence: 0.5}, // Low confidence
}
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {ID: "h0", Description: "Retrieve data"},
}
ctx.Enrichment.Facts = []models.Fact{
    {ID: "f1", Content: "Commit message", Source: "gitlab"},
}

agent := agents.NewInferenceAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// 2 conclusions
result.Reasoning.Conclusions = [
    {
        ID: "c0",
        Description: "Found 1 commit(s) from GitLab",
        Confidence: 0.95,
        Intent: "query_commits",
    },
    {
        ID: "c1",
        Description: "No issues found matching criteria",
        Confidence: 0.80,
        Intent: "query_issues",
    },
]

// 1 alternative (for no-match conclusion with moderate confidence)
result.Reasoning.Alternatives = []
```

---

### Example 3: Status Query with High Activity

**Input**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_status", Confidence: 0.75},
}
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {ID: "h0", Description: "Check system health"},
}
ctx.Enrichment.Facts = []models.Fact{
    // 15 facts from various sources
    {ID: "f1", Content: "Commit 1", Source: "gitlab"},
    {ID: "f2", Content: "Issue 1", Source: "youtrack"},
    // ... (13 more facts)
}

agent := agents.NewInferenceAgent()
result, err := agent.Execute(context.Background(), ctx)
```

**Output**:
```go
// 1 conclusion
result.Reasoning.Conclusions = [
    {
        ID: "c0",
        Description: "System is active with significant data",
        Confidence: 0.90,
        Evidence: ["fact:f1", "fact:f2", ...], // All 15 facts
        Intent: "query_status",
    },
]
```

## Testing Coverage

### Unit Tests (24+ tests)

1. **Agent Initialization**: Metadata, capabilities, contract
2. **Precondition Validation**: Missing intents, hypotheses, facts, nil reasoning context
3. **Conclusion Generation**: Commit queries, issue queries, analytics queries, status queries
4. **Confidence Scoring**: Evidence-based confidence calculation
5. **Evidence Matching**: Keyword extraction, overlap detection
6. **Inference Chain**: Chain construction, hypothesis-evidence linking
7. **Alternative Generation**: Low-confidence alternatives
8. **Context Isolation**: Input not modified
9. **Audit Trail**: Execution tracking, metrics
10. **Idempotency**: Same input produces same output
11. **Multiple Intents**: Handling multiple concurrent intents

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| **"no intents found"** | Empty intents array | Ensure Intent Detection Agent produced intents |
| **"no hypotheses found"** | Empty hypotheses array | Ensure Reasoning Structure Agent produced hypotheses |
| **"no facts found"** | Empty facts array | Ensure Context Synthesizer produced facts |
| **"reasoning context is nil"** | Missing reasoning context | Ensure reasoning agents ran first |

### Error Examples

```go
// Missing intents
ctx.Reasoning.Intents = []
→ Error: "no intents found (required: reasoning.intents)"

// Missing hypotheses
ctx.Reasoning.Hypotheses = []
→ Error: "no hypotheses found (required: reasoning.hypotheses)"

// Missing facts
ctx.Enrichment.Facts = []
→ Error: "no facts found (required: enrichment.facts)"
```

## Future Enhancements

1. **LLM-Based Inference**: Use LLM for complex cases requiring deep reasoning
2. **Probabilistic Reasoning**: Bayesian inference for updating beliefs
3. **Causal Inference**: Identify cause-effect relationships between facts
4. **Temporal Reasoning**: Time-based ordering and sequencing
5. **Contradiction Detection**: Identify conflicting facts/conclusions
6. **Confidence Propagation**: Propagate confidence through dependency chains
7. **Explanation Generation**: Natural language explanations for conclusions

## Troubleshooting

### Issue: "All conclusions have low confidence"

**Cause**: Weak evidence matching (keywords not overlapping with facts)

**Solution**: Review hypothesis phrasing and fact content; consider semantic matching

---

### Issue: "Too many alternatives generated"

**Cause**: Many low-confidence conclusions

**Solution**: Improve evidence quality or adjust confidence threshold

---

### Issue: "Inference chain is empty"

**Cause**: No hypotheses provided or all hypotheses filtered out

**Solution**: Check Reasoning Structure Agent output

## Related Documentation

- [Context Synthesizer Agent](./context_synthesizer.md)
- [Validation Agent](./validation.md)
- [Agent Context Schema](../agent_context_schema.md)
- [Reasoning System Overview](../README.md)

## API Reference

### Constructor

```go
func NewInferenceAgent() *InferenceAgent
```

### Execute

```go
func (a *InferenceAgent) Execute(
    ctx context.Context,
    agentContext *models.AgentContext
) (*models.AgentContext, error)
```

**Parameters**:
- `ctx`: Cancellation context
- `agentContext`: Must contain `reasoning.intents`, `reasoning.hypotheses`, `enrichment.facts`

**Returns**: Updated context with `reasoning.conclusions`, `reasoning.alternatives`, `reasoning.inference_chain`

### Internal Methods

```go
// Build step-by-step reasoning trace
func (a *InferenceAgent) buildInferenceChain(
    hypotheses []models.Hypothesis,
    facts []models.Fact,
    knowledge []models.Knowledge
) []models.InferenceStep

// Generate conclusions per intent
func (a *InferenceAgent) makeConclusions(
    intents []models.Intent,
    hypotheses []models.Hypothesis,
    facts []models.Fact,
    knowledge []models.Knowledge,
    chain []models.InferenceStep
) []models.Conclusion

// Generate alternative interpretations
func (a *InferenceAgent) generateAlternatives(
    conclusions []models.Conclusion,
    facts []models.Fact
) []models.Alternative

// Find supporting evidence for hypothesis
func (a *InferenceAgent) findSupportingEvidence(
    h models.Hypothesis,
    facts []models.Fact,
    knowledge []models.Knowledge
) []string

// Calculate confidence based on evidence
func (a *InferenceAgent) calculateEvidenceConfidence(
    h models.Hypothesis,
    facts []models.Fact,
    knowledge []models.Knowledge
) float64

// Extract keywords from text
func (a *InferenceAgent) extractKeywords(text string) []string

// Check keyword overlap
func (a *InferenceAgent) hasKeywordOverlap(
    keywords []string,
    text string
) bool
```

## Changelog

### Version 1.0.0 (2025-01-15)

**Initial Release**:
- Rule-based inference for 4 intent types (query_commits, query_issues, query_analytics, query_status)
- Evidence-based confidence scoring (0.20-0.90)
- Alternative generation for low-confidence conclusions (<0.70)
- Inference chain construction with hypothesis-evidence linking
- Keyword-based evidence matching
- Full test coverage (24+ unit tests)
- Zero LLM calls for fast, deterministic inference
