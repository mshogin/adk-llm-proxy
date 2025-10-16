# Agent Contracts Documentation

## Overview

Agent contracts define the formal interface between agents in the reasoning pipeline. Each agent declares preconditions (required inputs) and postconditions (guaranteed outputs), enabling validation and ensuring pipeline integrity.

## Contract Principles

1. **Preconditions**: Keys that must exist in context before agent execution
2. **Postconditions**: Keys the agent promises to populate
3. **Validation**: Automatic checking of contract fulfillment
4. **Namespace Isolation**: Agents only write to designated namespaces
5. **Fail-Fast**: Contract violations stop pipeline immediately

## Agent Interface

```go
type ReasoningAgent interface {
    AgentID() string                // Unique agent identifier
    Preconditions() []string        // Required context keys
    Postconditions() []string       // Guaranteed output keys
    Execute(ctx context.Context, agentContext *AgentContext) (*AgentContext, error)
}
```

## Standard Agents and Their Contracts

### 1. Intent Detection Agent

**Agent ID**: `intent_detection`

**Purpose**: Detect user intents and extract entities from input text

**Preconditions**:
- None (entry point agent)

**Postconditions**:
- `reasoning.intents` - Array of detected intents with confidence scores
- `reasoning.entities` - Map of extracted entities

**Input Requirements**:
- User input text must be present in context metadata

**Example Contract**:
```go
func (a *IntentDetectionAgent) Preconditions() []string {
    return []string{} // No preconditions
}

func (a *IntentDetectionAgent) Postconditions() []string {
    return []string{"reasoning.intents", "reasoning.entities"}
}
```

**Guaranteed Output**:
```go
ctx.Reasoning.Intents = []models.Intent{
    {Type: "query_commits", Confidence: 0.95},
}
ctx.Reasoning.Entities = map[string]models.Entity{
    "project": {Value: "my-project", Type: "identifier"},
}
```

---

### 2. Reasoning Structure Agent

**Agent ID**: `reasoning_structure`

**Purpose**: Build reasoning goal hierarchy and generate hypotheses

**Preconditions**:
- `reasoning.intents` - Detected intents from intent detection

**Postconditions**:
- `reasoning.hypotheses` - Generated hypotheses with dependencies

**Example Contract**:
```go
func (a *ReasoningStructureAgent) Preconditions() []string {
    return []string{"reasoning.intents"}
}

func (a *ReasoningStructureAgent) Postconditions() []string {
    return []string{"reasoning.hypotheses"}
}
```

**Guaranteed Output**:
```go
ctx.Reasoning.Hypotheses = []models.Hypothesis{
    {
        ID: "h1",
        Description: "User wants to analyze commit activity",
        Dependencies: []string{},
        Confidence: 0.90,
    },
}
```

---

### 3. Retrieval Planner Agent

**Agent ID**: `retrieval_planner`

**Purpose**: Generate retrieval plans for data sources

**Preconditions**:
- `reasoning.intents` - User intents
- `reasoning.hypotheses` - Reasoning hypotheses

**Postconditions**:
- `retrieval.plans` - Retrieval strategies
- `retrieval.queries` - Generated queries

**Example Contract**:
```go
func (a *RetrievalPlannerAgent) Preconditions() []string {
    return []string{"reasoning.intents", "reasoning.hypotheses"}
}

func (a *RetrievalPlannerAgent) Postconditions() []string {
    return []string{"retrieval.plans", "retrieval.queries"}
}
```

**Guaranteed Output**:
```go
ctx.Retrieval.Plans = []models.RetrievalPlan{
    {
        ID: "plan-1",
        Description: "Query GitLab commits",
        Sources: []string{"gitlab"},
        Priority: 1,
    },
}
```

---

### 4. Context Synthesizer Agent

**Agent ID**: `context_synthesizer`

**Purpose**: Normalize and merge facts from multiple sources

**Preconditions**:
- `retrieval.plans` - Retrieval plans
- `retrieval.queries` - Queries to execute

**Postconditions**:
- `enrichment.facts` - Collected and normalized facts
- `enrichment.derived_knowledge` - Inferred information

**Example Contract**:
```go
func (a *ContextSynthesizerAgent) Preconditions() []string {
    return []string{"retrieval.plans", "retrieval.queries"}
}

func (a *ContextSynthesizerAgent) Postconditions() []string {
    return []string{"enrichment.facts", "enrichment.derived_knowledge"}
}
```

**Guaranteed Output**:
```go
ctx.Enrichment.Facts = []models.Fact{
    {
        ID: "fact-1",
        Content: "15 commits in last week",
        Source: "gitlab",
        Confidence: 1.0,
    },
}
```

---

### 5. Inference Agent

**Agent ID**: `inference`

**Purpose**: Make conclusions based on facts and hypotheses

**Preconditions**:
- `reasoning.hypotheses` - Generated hypotheses
- `enrichment.facts` - Collected facts

**Postconditions**:
- `reasoning.conclusions` - Final conclusions
- `reasoning.inference_chain` - Reasoning steps

**Example Contract**:
```go
func (a *InferenceAgent) Preconditions() []string {
    return []string{"reasoning.hypotheses", "enrichment.facts"}
}

func (a *InferenceAgent) Postconditions() []string {
    return []string{"reasoning.conclusions", "reasoning.inference_chain"}
}
```

**Guaranteed Output**:
```go
ctx.Reasoning.Conclusions = []models.Conclusion{
    {
        ID: "c1",
        Description: "High development activity detected",
        Confidence: 0.88,
        SupportingEvidence: []string{"fact-1"},
    },
}
```

---

### 6. Validation Agent

**Agent ID**: `validation`

**Purpose**: Validate reasoning chain completeness and consistency

**Preconditions**:
- `reasoning.intents` - User intents
- `reasoning.conclusions` - Generated conclusions

**Postconditions**:
- `diagnostics.validation_reports` - Validation results

**Example Contract**:
```go
func (a *ValidationAgent) Preconditions() []string {
    return []string{"reasoning.intents", "reasoning.conclusions"}
}

func (a *ValidationAgent) Postconditions() []string {
    return []string{"diagnostics.validation_reports"}
}
```

**Guaranteed Output**:
```go
ctx.Diagnostics.ValidationReports = []models.ValidationReport{
    {
        AgentID: "validation",
        Passed: true,
        Violations: []string{},
        Timestamp: time.Now(),
    },
}
```

---

### 7. Summarization Agent

**Agent ID**: `summarization`

**Purpose**: Generate executive summary of reasoning process

**Preconditions**:
- `reasoning.conclusions` - Final conclusions
- `enrichment.facts` - Supporting facts

**Postconditions**:
- `reasoning.summary` - Executive summary

**Example Contract**:
```go
func (a *SummarizationAgent) Preconditions() []string {
    return []string{"reasoning.conclusions", "enrichment.facts"}
}

func (a *SummarizationAgent) Postconditions() []string {
    return []string{"reasoning.summary"}
}
```

**Guaranteed Output**:
```go
ctx.Reasoning.Summary = "Analysis shows high development activity with 15 commits in the last week across 3 active projects."
```

---

## Contract Validation

### Validation Process

1. **Pre-execution**: Check all preconditions exist and are non-empty
2. **Post-execution**: Verify all postconditions are fulfilled
3. **Namespace Check**: Ensure agent only wrote to allowed namespaces
4. **Failure Handling**: Stop pipeline on violation if `FailOnViolation` is true

### Validation Configuration

```go
type AgentExecutionOptions struct {
    ValidateContract bool // Enable contract validation
    FailOnViolation  bool // Stop pipeline on violation
    TrackPerformance bool // Track execution metrics
}
```

### Example Validation

```go
// Enable validation
options := domainservices.AgentExecutionOptions{
    ValidateContract: true,
    FailOnViolation:  true,
}

// Execute with validation
result, err := manager.Execute(ctx, agentContext)
if err != nil {
    // Contract violation occurred
    log.Error("Contract violation:", err)
}
```

---

## Writing Custom Agents

### Step 1: Define Contract

```go
type MyCustomAgent struct {
    id string
}

func (a *MyCustomAgent) AgentID() string {
    return a.id
}

func (a *MyCustomAgent) Preconditions() []string {
    return []string{"reasoning.intents"} // What you need
}

func (a *MyCustomAgent) Postconditions() []string {
    return []string{"custom.output"} // What you promise
}
```

### Step 2: Implement Execute

```go
func (a *MyCustomAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
    // Check preconditions (automatic if validation enabled)
    if len(agentContext.Reasoning.Intents) == 0 {
        return nil, errors.New("missing intents")
    }

    // Process
    result := processIntents(agentContext.Reasoning.Intents)

    // Write postconditions
    agentContext.Custom.Output = result

    return agentContext, nil
}
```

### Step 3: Register Agent

```go
manager := NewReasoningManager(config)
agent := &MyCustomAgent{id: "my_custom_agent"}
err := manager.RegisterAgent(agent)
if err != nil {
    log.Fatal(err)
}
```

---

## Best Practices

### 1. Minimal Preconditions

✅ **DO**: Require only what you actually need
```go
func (a *Agent) Preconditions() []string {
    return []string{"reasoning.intents"} // Only intents needed
}
```

❌ **DON'T**: Require unnecessary data
```go
func (a *Agent) Preconditions() []string {
    return []string{"reasoning.intents", "enrichment.facts", "retrieval.plans"} // Too many
}
```

### 2. Guaranteed Postconditions

✅ **DO**: Always fulfill your promises
```go
func (a *Agent) Execute(ctx, agentContext) (*AgentContext, error) {
    // Always set postcondition before returning
    agentContext.Reasoning.Conclusions = computeConclusions()
    return agentContext, nil
}
```

❌ **DON'T**: Return without fulfilling postconditions
```go
func (a *Agent) Execute(ctx, agentContext) (*AgentContext, error) {
    if someCondition {
        return agentContext, nil // Forgot to set conclusions!
    }
}
```

### 3. Error Handling

✅ **DO**: Return errors for failures
```go
func (a *Agent) Execute(ctx, agentContext) (*AgentContext, error) {
    result, err := processData()
    if err != nil {
        return nil, fmt.Errorf("processing failed: %w", err)
    }
    agentContext.Output = result
    return agentContext, nil
}
```

### 4. Namespace Isolation

✅ **DO**: Write only to your designated namespace
```go
// Intent detection agent writes to reasoning namespace
agentContext.Reasoning.Intents = detectedIntents
```

❌ **DON'T**: Write to foreign namespaces
```go
// Intent detection agent should NOT write to enrichment
agentContext.Enrichment.Facts = facts // WRONG!
```

---

## Contract Violations

### Common Violations

#### 1. Missing Postcondition

**Error**: `postcondition validation failed: reasoning.intents not found`

**Cause**: Agent didn't populate promised field

**Fix**: Ensure all postconditions are always set

#### 2. Empty Array

**Error**: `postcondition validation failed: reasoning.intents is empty`

**Cause**: Agent set array to empty value

**Fix**: Populate with at least one element or change contract

#### 3. Wrong Type

**Error**: `postcondition validation failed: expected array, got nil`

**Cause**: Field not initialized

**Fix**: Initialize field before returning

#### 4. Namespace Violation

**Error**: `namespace violation: agent wrote to unauthorized key`

**Cause**: Agent wrote outside designated namespace

**Fix**: Write only to postcondition namespaces

---

## Testing Contracts

### Unit Test Example

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
}
```

---

## References

- [AgentContext Schema](./agent_context_schema.md)
- [Pipeline Configuration](./pipeline_configuration.md)
- Source: `src/golang/internal/domain/services/reasoning_agent.go`
- Tests: `src/golang/internal/application/services/reasoning_manager_test.go`
