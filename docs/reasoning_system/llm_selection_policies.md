# LLM Model Selection Policies

This document describes the dynamic LLM model selection strategies used by the ADK LLM Proxy's orchestrator.

## Overview

The LLM Orchestrator automatically selects the most appropriate model for each task based on:
- **Task complexity** (simple validation → critical reasoning)
- **Context size** (token count requirements)
- **Budget constraints** (cost per session/agent)
- **Provider availability** (with automatic fallback chains)

This ensures optimal cost-performance balance while maintaining quality.

## Model Profiles

### Default Model Profiles

| Provider | Model | Quality | Speed | Cost/1K tok | Context Limit | Use Cases |
|----------|-------|---------|-------|-------------|---------------|-----------|
| **Ollama** | mistral | good | fast | $0.00 | 32K | Local, fast, cheap tasks |
| **Ollama** | llama3 | good | fast | $0.00 | 8K | Local inference |
| **DeepSeek** | deepseek-chat | good | fast | $0.0001 | 64K | Classification, extraction, validation |
| **DeepSeek** | deepseek-r1 | premium | medium | $0.0003 | 64K | Complex reasoning |
| **OpenAI** | gpt-4o-mini | good | fast | $0.00015 | 128K | General purpose, cost-effective |
| **OpenAI** | gpt-4o | premium | medium | $0.0025 | 128K | Medium-complex synthesis |
| **OpenAI** | o1-mini | premium | slow | $0.015 | 128K | Deep reasoning with CoT |
| **OpenAI** | o1 | premium | slow | $0.06 | 200K | Critical reasoning, high-stakes |
| **Anthropic** | claude-3-haiku | good | fast | $0.00025 | 200K | Fast, cost-effective |
| **Anthropic** | claude-3-5-sonnet | premium | medium | $0.003 | 200K | Long-context analysis |
| **Anthropic** | claude-opus | premium | slow | $0.015 | 200K | Critical reasoning fallback |

### Model Profile Schema

```go
type ModelProfile struct {
    Provider             string  // Provider name
    Model                string  // Model identifier
    Quality              string  // "good", "premium", "basic"
    Speed                string  // "fast", "medium", "slow"
    CostPer1KTokens      float64 // Cost per 1K tokens (USD)
    ContextLimit         int     // Maximum context window (tokens)
    SupportsStreaming    bool    // Streaming support
    SupportsFunctions    bool    // Function calling support
    SupportsVision       bool    // Vision capabilities
    AverageLatencyMS     int     // Typical response latency
    IsLocal              bool    // Local/cloud deployment
    RequiresAuth         bool    // API key required
    MaxRequestsPerSecond int     // Rate limit (0 = unlimited)
    RequestTimeoutMS     int     // Request timeout
}
```

## Selection Strategies by Task Type

### Task Complexity Classification

The orchestrator classifies tasks into categories based on:
- Input size (word/token count)
- Required reasoning depth
- Output complexity

| Task Type | Context Size | Characteristics | Default Model | Fallback 1 | Fallback 2 |
|-----------|--------------|-----------------|---------------|------------|------------|
| **Intent classification** | <500 tok | Single message, <50 words, pattern matching | deepseek/deepseek-chat | openai/gpt-4o-mini | ollama/mistral |
| **Entity extraction** | <1K tok | Structured extraction, <100 words | deepseek/deepseek-chat | openai/gpt-4o-mini | ollama/llama3 |
| **Simple validation** | <1K tok | Boolean checks, <200 words | deepseek/deepseek-chat | openai/gpt-4o-mini | rules-only |
| **Keyword-based search** | <2K tok | Query generation, keywords | openai/gpt-4o-mini | deepseek/deepseek-chat | ollama/mistral |
| **Short text synthesis** | <2K tok | Summarization, <500 words | openai/gpt-4o-mini | deepseek/deepseek-chat | ollama/llama3 |
| **Query normalization** | <2K tok | SQL/query reformatting | openai/gpt-4o-mini | deepseek/deepseek-chat | - |
| **Fact deduplication** | <3K tok | Simple comparison logic | openai/gpt-4o-mini | deepseek/deepseek-chat | ollama/mistral |
| **Simple inference** | <3K tok | Rule-based conclusions | openai/gpt-4o-mini | deepseek/deepseek-chat | - |
| **Medium synthesis** | <8K tok | Multiple sources, <2000 words | openai/gpt-4o | anthropic/claude-haiku | openai/gpt-4o-mini |
| **Complex retrieval planning** | <8K tok | Multi-step with dependencies | openai/gpt-4o | anthropic/claude-haiku | deepseek/deepseek-r1 |
| **Multi-source correlation** | <16K tok | Cross-reference analysis | openai/gpt-4o | anthropic/claude-sonnet | deepseek/deepseek-r1 |
| **Advanced inference** | <16K tok | Logical reasoning chains | openai/gpt-4o | anthropic/claude-sonnet | deepseek/deepseek-r1 |
| **Long-context analysis** | <32K tok | Large document analysis | anthropic/claude-sonnet | openai/gpt-4o | - |
| **Deep reasoning** | <64K tok | Multi-hop reasoning, ambiguity | openai/o1-mini | anthropic/claude-sonnet | openai/gpt-4o |
| **Critical reasoning** | <100K tok | High-stakes, chain-of-thought | openai/o1 | anthropic/claude-opus | openai/o1-mini |

### Task Detection Heuristics

```go
func DetectTaskType(input *ReasoningInput) TaskType {
    wordCount := countWords(input.Prompt)
    tokenCount := estimateTokens(input.Prompt)

    // Intent classification
    if isSingleSentence(input.Prompt) && wordCount < 50 {
        return TaskTypeIntentClassification
    }

    // Entity extraction
    if hasStructuredRequest(input.Prompt) && wordCount < 100 {
        return TaskTypeEntityExtraction
    }

    // Simple validation
    if isBooleanQuestion(input.Prompt) && wordCount < 200 {
        return TaskTypeValidation
    }

    // Medium synthesis
    if hasMultipleSources(input.Context) && wordCount < 2000 {
        return TaskTypeMediumSynthesis
    }

    // Deep reasoning
    if requiresMultiHop(input.Prompt) || hasAmbiguity(input.Prompt) {
        return TaskTypeDeepReasoning
    }

    // Default to simple inference
    return TaskTypeInference
}
```

## Budget-Aware Selection

### Budget Constraints

```go
type BudgetConstraints struct {
    SessionBudgetUSD  float64  // Max cost per session
    AgentBudgetUSD    float64  // Max cost per agent
    WarningThreshold  float64  // Warning at X% of budget
    CriticalAgents    []string // Agents that bypass budget
}
```

### Budget Control Strategy

1. **Normal Operation** (budget <80% spent)
   - Use default model from selection table
   - Cache responses aggressively

2. **Warning State** (budget 80-100% spent)
   - Downgrade to cheaper models (e.g., gpt-4o → gpt-4o-mini)
   - Increase cache reliance
   - Log budget warnings

3. **Emergency State** (budget >100% spent)
   - Critical agents continue (e.g., validation, summarization)
   - Non-critical agents skip LLM, use deterministic fallback
   - Cache-only mode for optional operations

### Cost Calculation

```go
func (o *LLMOrchestrator) CalculateCost(provider, model string, tokens int) float64 {
    profile := o.profiles[fmt.Sprintf("%s/%s", provider, model)]
    return profile.CostPer1KTokens * (float64(tokens) / 1000.0)
}

func (o *LLMOrchestrator) TrackUsage(agentID, provider, model string, tokens int) float64 {
    cost := o.CalculateCost(provider, model, tokens)

    o.sessionBudgetUsed += cost
    o.agentBudgetUsed[agentID] += cost

    return cost
}
```

## Fallback Chain Strategy

### Fallback Sequence

1. **Default Model** - Optimal for task type
2. **Fallback 1** - Alternative provider/model
3. **Fallback 2** - Last resort (often local or cheapest)
4. **Deterministic** - Rule-based, no LLM (if applicable)

### Fallback Triggers

- **Provider unavailable** (network error, 503)
- **Rate limited** (429 Too Many Requests)
- **Timeout** (request exceeds configured timeout)
- **Budget exceeded** (agent/session budget limit)
- **Context too large** (exceeds model context limit)

### Example Fallback Chain

```
Task: Intent Classification
Input: 450 tokens

1. Try: deepseek/deepseek-chat (default, $0.0001/1K, 64K limit)
   → Success ✓

If failed:
2. Try: openai/gpt-4o-mini (fallback 1, $0.00015/1K, 128K limit)
   → Success ✓

If failed:
3. Try: ollama/mistral (fallback 2, $0.00, 32K limit, local)
   → Success ✓

If failed:
4. Use: rule-based classifier (deterministic, no cost)
```

## Caching Strategy

### Cache Configuration

```go
type CacheConfig struct {
    Enabled            bool // Enable caching
    ClassificationTTL  int  // TTL for classification tasks (seconds)
    SynthesisTTL       int  // TTL for synthesis tasks
    InferenceTTL       int  // TTL for inference tasks
}
```

### Default Cache TTL by Task Type

| Task Type | TTL | Rationale |
|-----------|-----|-----------|
| Intent classification | 24h | Stable, repeatable |
| Entity extraction | 24h | Deterministic |
| Simple validation | 12h | Mostly deterministic |
| Short synthesis | 1h | Context-dependent |
| Medium synthesis | 1h | Context-dependent |
| Inference | 30min | Context-sensitive |
| Deep reasoning | 15min | Highly context-sensitive |

### Cache Key Generation

```go
func (o *LLMOrchestrator) GetCacheKey(req *LLMRequest, model string) string {
    data := fmt.Sprintf("%s|%s|%d|%.2f|%s",
        req.Prompt, model, req.MaxTokens, req.Temperature, req.TaskType)
    hash := sha256.Sum256([]byte(data))
    return fmt.Sprintf("%x", hash)
}
```

### Cache Hit Targets

- **Simple tasks** (classification, extraction): >60% hit rate
- **Medium tasks** (synthesis, validation): >40% hit rate
- **Complex tasks** (inference, reasoning): >20% hit rate

## Throttling and Rate Limits

### Per-Provider Limits

| Provider | Model | Max RPS | Timeout |
|----------|-------|---------|---------|
| Ollama | mistral | 50 | 10s |
| Ollama | llama3 | 50 | 10s |
| DeepSeek | deepseek-chat | 20 | 15s |
| DeepSeek | deepseek-r1 | 10 | 30s |
| OpenAI | gpt-4o-mini | 50 | 15s |
| OpenAI | gpt-4o | 20 | 20s |
| OpenAI | o1-mini | 10 | 60s |
| OpenAI | o1 | 5 | 120s |
| Anthropic | claude-3-haiku | 50 | 15s |
| Anthropic | claude-3-5-sonnet | 20 | 30s |
| Anthropic | claude-opus | 10 | 30s |

### Throttling Algorithm

Token bucket algorithm with per-provider/model limits:

```go
type rateLimiter struct {
    maxRequests    int           // Requests per second
    tokens         int           // Available tokens
    lastRefill     time.Time     // Last refill time
    requestTimeout time.Duration // Request timeout
}

func (r *rateLimiter) tryAcquire() bool {
    now := time.Now()
    elapsed := now.Sub(r.lastRefill)

    // Refill tokens based on elapsed time
    tokensToAdd := int(elapsed.Seconds() * float64(r.maxRequests))
    if tokensToAdd > 0 {
        r.tokens = min(r.tokens + tokensToAdd, r.maxRequests)
        r.lastRefill = now
    }

    // Try to acquire token
    if r.tokens > 0 {
        r.tokens--
        return true
    }
    return false
}
```

## Security Filtering

### PII Masking

Before sending requests to LLM providers, sensitive information is automatically masked:

| PII Type | Detection | Replacement | Example |
|----------|-----------|-------------|---------|
| Email | Regex pattern | Partial mask | `john.doe@example.com` → `j***e@example.com` |
| Phone | US/intl formats | `[PHONE]` | `555-123-4567` → `[PHONE]` |
| SSN | XXX-XX-XXXX | `[SSN]` | `123-45-6789` → `[SSN]` |
| Credit Card | 4×4 digits | `[CC-XXXX-1234]` | `1234-5678-9012-3456` → `[CC-XXXX-3456]` |
| API Keys | Common prefixes | `prefix=[REDACTED]` | `api_key=sk_1234...` → `api_key=[REDACTED]` |

### Field Truncation

Large fields are truncated to prevent context explosion:

- **Default limit**: 10,000 characters per field
- **Truncation**: Preserves word boundaries
- **Indicator**: `... [TRUNCATED]` suffix

## Decision Logging

Every model selection is logged for audit and analysis:

```go
type LLMDecision struct {
    Timestamp time.Time `json:"timestamp"`
    AgentID   string    `json:"agent_id"`
    TaskType  string    `json:"task_type"`
    Selected  string    `json:"selected"`    // provider/model
    Reason    string    `json:"reason"`      // Why this model
}
```

### Decision Reasons

- `default_for_task_type` - Default model for this task complexity
- `fallback_1_default_unavailable` - Primary failed, using fallback
- `fallback_2_all_others_unavailable` - All options exhausted
- `budget_downgrade` - Cheaper model due to budget constraints
- `cache_hit` - Using cached response
- `deterministic_fallback` - Using rules instead of LLM

## Usage Examples

### Example 1: Simple Intent Classification

```go
req := &LLMRequest{
    Prompt:      "What GitLab projects did we work on last week?",
    TaskType:    TaskTypeIntentClassification,
    AgentID:     "intent_agent",
    MaxTokens:   100,
    Temperature: 0.0,
    ContextSize: 450,
}

model, provider, err := orchestrator.SelectModel(ctx, req)
// Result: model="deepseek-chat", provider="deepseek"
// Reason: "default_for_task_type (cost: $0.000045, quality: good, speed: fast)"
```

### Example 2: Complex Reasoning with Budget

```go
// Session budget: $1.00
// Already spent: $0.85 (85%)

req := &LLMRequest{
    Prompt:      largeAnalysisPrompt, // 15K tokens
    TaskType:    TaskTypeAdvancedInference,
    AgentID:     "inference_agent",
    MaxTokens:   2000,
    Temperature: 0.7,
    ContextSize: 15000,
}

model, provider, err := orchestrator.SelectModel(ctx, req)
// Result: model="gpt-4o-mini", provider="openai"  // Downgraded!
// Reason: "budget_downgrade (session 85% spent, using cheaper model)"
```

### Example 3: Fallback Chain

```go
// DeepSeek unavailable (503), OpenAI rate limited (429)

req := &LLMRequest{
    Prompt:      "Categorize this issue",
    TaskType:    TaskTypeEntityExtraction,
    AgentID:     "extraction_agent",
}

model, provider, err := orchestrator.SelectModel(ctx, req)
// Try 1: deepseek/deepseek-chat → 503 error
// Try 2: openai/gpt-4o-mini → 429 rate limit
// Result: model="mistral", provider="ollama"
// Reason: "fallback_2_all_others_unavailable (local fallback)"
```

## Performance Metrics

### Target Metrics

- **Model selection latency**: <10ms
- **Cache lookup**: <5ms
- **Budget check**: <1ms
- **Throttle check**: <1ms
- **Total orchestration overhead**: <20ms

### Cost Savings

With aggressive caching and smart selection:
- **Classification tasks**: 60-80% cost reduction (cache hits)
- **Simple tasks**: 90% cost reduction (cheap models)
- **Complex tasks**: 30-50% cost reduction (right-sized models)

### Decision Quality

- **Correct model selection**: >95% of tasks use appropriate model tier
- **Budget adherence**: 100% (hard limits enforced)
- **Fallback success**: >99% of failed requests recover via fallback

## Configuration

### Orchestrator Configuration

```go
orchestrator := NewLLMOrchestratorWithConfig(
    BudgetConstraints{
        SessionBudgetUSD:  1.00,      // $1 per session
        AgentBudgetUSD:    0.10,      // $0.10 per agent
        WarningThreshold:  0.80,      // Warn at 80%
        CriticalAgents:    []string{"validation", "summarization"},
    },
    CacheConfig{
        Enabled:            true,
        ClassificationTTL:  86400,    // 24h
        SynthesisTTL:       3600,     // 1h
        InferenceTTL:       1800,     // 30min
    },
)
```

### Runtime Updates

```go
// Update rate limits dynamically
orchestrator.UpdateProviderThrottle("openai", "gpt-4o-mini", 100, 10000)

// Get current stats
stats := orchestrator.GetThrottleStats()
budgetUsed, budgetLimit, agentBudgets := orchestrator.GetBudgetStatus()
```

## Monitoring and Observability

### Key Metrics to Track

1. **Cost metrics**: Total spend, spend by agent, spend by model
2. **Performance**: Selection latency, cache hit rate, throttle delays
3. **Quality**: Fallback frequency, budget violations, decision reasons
4. **Availability**: Provider uptime, fallback success rate

### Decision Audit Trail

All decisions are logged to `llm.decisions` in AgentContext:

```json
{
  "llm": {
    "decisions": [
      {
        "timestamp": "2025-10-15T13:30:00Z",
        "agent_id": "intent_agent",
        "task_type": "intent_classification",
        "selected": "deepseek/deepseek-chat",
        "reason": "default_for_task_type (cost: $0.000045)"
      }
    ]
  }
}
```

## Best Practices

1. **Choose task types carefully** - Accurate classification ensures optimal model selection
2. **Set realistic budgets** - Balance cost vs quality requirements
3. **Monitor cache hit rates** - Low hit rates indicate poor caching strategy
4. **Review decision logs** - Identify patterns in model selection
5. **Test fallback chains** - Ensure graceful degradation works
6. **Update model profiles** - Keep costs and limits current
7. **Use critical agents sparingly** - Budget bypass should be rare

## Troubleshooting

### Common Issues

**Issue**: All requests use expensive models
- **Cause**: Task types incorrectly classified as complex
- **Fix**: Review task detection heuristics, adjust thresholds

**Issue**: Frequent fallback to local models
- **Cause**: Cloud providers rate limited or unavailable
- **Fix**: Increase rate limits, check API key validity, add retries

**Issue**: Budget exceeded frequently
- **Cause**: Budget too low or tasks too complex
- **Fix**: Increase budget or use more caching

**Issue**: Cache hit rate <10%
- **Cause**: Highly variable prompts, low TTL
- **Fix**: Increase TTL for stable task types, normalize prompts

## References

- Model profiles: `src/golang/internal/domain/models/llm_profile.go`
- Orchestrator: `src/golang/internal/application/services/llm_orchestrator.go`
- Throttler: `src/golang/internal/application/services/llm_throttler.go`
- Security filter: `src/golang/internal/application/services/llm_security_filter.go`
