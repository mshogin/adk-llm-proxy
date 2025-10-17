# Single-Agent Integration Testing Specification

## Цель

Создать инфраструктуру для изолированного тестирования каждого агента отдельно, без необходимости запускать полный pipeline из 8 агентов.

## Проблема

Текущая архитектура требует запуска всех 8 агентов последовательно для тестирования. Это:
- **Медленно**: полный pipeline занимает несколько секунд
- **Сложно дебажить**: трудно понять где именно возникла проблема
- **Неудобно для разработки**: нельзя работать точечно над одним агентом
- **Усложняет тестирование**: много зависимостей между агентами

## Решение

Создать single-agent workflows и интеграционные тесты для каждого агента отдельно.

## Архитектура

### 1. Single-Agent Workflows

Создать 8 отдельных workflow (по одному на каждый агент):

```
intent_detection_only    → Тестирует IntentDetectionAgent
reasoning_structure_only → Тестирует ReasoningStructureAgent
retrieval_planner_only   → Тестирует RetrievalPlannerAgent
retrieval_executor_only  → Тестирует RetrievalExecutorAgent
context_synthesizer_only → Тестирует ContextSynthesizerAgent
inference_only           → Тестирует InferenceAgent
summarization_only       → Тестирует SummarizationAgent
validation_only          → Тестирует ValidationAgent
```

### 2. Mock Data Providers

Каждый агент имеет preconditions (требуемые входные данные). Нужно создать mock providers, которые подготавливают эти данные:

**Пример: ReasoningStructureAgent**
```go
// Preconditions: reasoning.intents
// Нужен MockIntentProvider, который создаст:
type MockIntentProvider struct{}

func (m *MockIntentProvider) CreateContext() *models.AgentContext {
    ctx := models.NewAgentContext("test-session", "test-trace")
    ctx.Reasoning.Intents = []models.Intent{
        {Type: "query_commits", Confidence: 0.99},
        {Type: "request_help", Confidence: 0.64},
    }
    return ctx
}
```

**Mock providers для всех агентов:**

| Agent | Preconditions | Mock Provider |
|-------|---------------|---------------|
| intent_detection | (нет) | MockUserInputProvider |
| reasoning_structure | reasoning.intents | MockIntentProvider |
| retrieval_planner | reasoning.intents, reasoning.hypotheses | MockReasoningProvider |
| retrieval_executor | retrieval.plans | MockRetrievalPlanProvider |
| context_synthesizer | retrieval.artifacts | MockArtifactProvider |
| inference | reasoning.intents, reasoning.hypotheses, enrichment.facts | MockEnrichmentProvider |
| summarization | reasoning.intents, reasoning.conclusions | MockInferenceProvider |
| validation | (complete context) | MockCompleteContextProvider |

### 3. Single-Agent Workflow Implementation

Каждый single-agent workflow должен:

1. **Создать mock context** с preconditions
2. **Запустить один агент**
3. **Вернуть результат** в reasoning блоке

**Пример реализации:**

```go
// pkg/workflows/intent_detection_only.go
type IntentDetectionOnlyWorkflow struct {
    agent *agents.IntentDetectionAgent
}

func (w *IntentDetectionOnlyWorkflow) Execute(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error) {
    // 1. Create context with user input
    agentContext := w.createContextFromInput(input)

    // 2. Execute only IntentDetectionAgent
    resultContext, err := w.agent.Execute(ctx, agentContext)
    if err != nil {
        return nil, fmt.Errorf("agent failed: %w", err)
    }

    // 3. Build detailed result
    result := w.buildDetailedResult(agentContext, resultContext)

    return result, nil
}

func (w *IntentDetectionOnlyWorkflow) buildDetailedResult(before, after *models.AgentContext) *models.ReasoningResult {
    result := models.NewReasoningResult("intent_detection_only", "Single agent test")

    message := "=== INTENT DETECTION AGENT TEST ===\n\n"

    // Show INPUT
    message += "📥 INPUT:\n"
    if before.LLM != nil && before.LLM.Cache != nil {
        if input, ok := before.LLM.Cache["original_user_input"].(string); ok {
            message += fmt.Sprintf("  User message: \"%s\"\n", input)
        }
    }
    message += "\n"

    // Show OUTPUT
    message += "📤 OUTPUT:\n"
    if len(after.Reasoning.Intents) > 0 {
        for _, intent := range after.Reasoning.Intents {
            message += fmt.Sprintf("  • %s (confidence: %.2f)\n", intent.Type, intent.Confidence)
        }
    }

    // Show entities
    if len(after.Reasoning.Entities) > 0 {
        message += "\n  Entities:\n"
        for key, values := range after.Reasoning.Entities {
            message += fmt.Sprintf("    • %s: %v\n", key, values)
        }
    }
    message += "\n"

    // Show agent trace (LLM calls, etc.)
    if after.LLM != nil && after.LLM.Cache != nil {
        if traces, ok := after.LLM.Cache["agent_traces"].([]interface{}); ok {
            for _, trace := range traces {
                if traceMap, ok := trace.(map[string]interface{}); ok {
                    if triggered, ok := traceMap["llm_fallback_triggered"].(bool); ok && triggered {
                        message += "💬 LLM FALLBACK TRIGGERED:\n"
                        if reason, ok := traceMap["llm_trigger_reason"].(string); ok {
                            message += fmt.Sprintf("  Reason: %s\n", reason)
                        }
                        if llmCalls, ok := traceMap["llm_calls_made"].(int); ok {
                            message += fmt.Sprintf("  LLM calls: %d\n", llmCalls)
                        }
                        message += "\n"
                    }
                }
            }
        }
    }

    // Show system prompt that would go to LLM
    message += "=== 📤 SYSTEM PROMPT FOR LLM ===\n\n"
    systemPrompt := w.buildSystemPrompt(after)
    message += systemPrompt
    message += "\n"

    // Show metrics
    if after.Diagnostics != nil && after.Diagnostics.Performance != nil {
        message += fmt.Sprintf("\n⏱️  Duration: %dms\n", after.Diagnostics.Performance.TotalDurationMS)
        if metrics, ok := after.Diagnostics.Performance.AgentMetrics["intent_detection"]; ok {
            message += fmt.Sprintf("   LLM calls: %d\n", metrics.LLMCalls)
            if metrics.Cost > 0 {
                message += fmt.Sprintf("   Cost: $%.6f\n", metrics.Cost)
            }
        }
    }

    result.Message = message

    return result
}
```

### 4. Response Format

Reasoning блок для single-agent workflow должен содержать:

```
=== [AGENT NAME] AGENT TEST ===

📥 INPUT:
  [Детальное описание входных данных]
  [Показать все preconditions]

📤 OUTPUT:
  [Детальное описание выходных данных]
  [Показать все postconditions]

💬 LLM INTERACTION (если был вызов):
  Reason: [Почему вызвали LLM]
  Calls: [Количество вызовов]
  Model: [Какая модель использовалась]
  Tokens: [Использовано токенов]
  Cost: [Стоимость]

=== 📤 SYSTEM PROMPT FOR LLM ===
[Финальный промпт который ушел бы в LLM]

⏱️ METRICS:
  Duration: [время выполнения]
  LLM calls: [количество]
  Cost: [стоимость]
```

### 5. Integration Tests

Создать интеграционные тесты для каждого агента в `tests/golang/integration/single_agent/`:

```go
// tests/golang/integration/single_agent/intent_detection_test.go
func TestIntentDetectionAgent_QueryCommits(t *testing.T) {
    // 1. Setup test server with intent_detection_only workflow
    server := setupTestServer(t, "intent_detection_only")
    defer server.Close()

    // 2. Send test request
    req := createTestRequest("What are my recent commits?")
    resp := sendRequest(t, server, req)

    // 3. Parse response
    reasoning := extractReasoningBlock(t, resp)

    // 4. Validate
    assert.Contains(t, reasoning, "📥 INPUT:")
    assert.Contains(t, reasoning, "User message: \"What are my recent commits?\"")
    assert.Contains(t, reasoning, "📤 OUTPUT:")
    assert.Contains(t, reasoning, "query_commits")
    assert.Contains(t, reasoning, "confidence:")
    assert.Contains(t, reasoning, "=== 📤 SYSTEM PROMPT FOR LLM ===")

    // 5. Validate only one agent ran
    agentExecutions := extractAgentExecutions(t, reasoning)
    assert.Equal(t, 1, len(agentExecutions))
    assert.Equal(t, "intent_detection", agentExecutions[0].AgentID)
}

func TestIntentDetectionAgent_AmbiguousIntent(t *testing.T) {
    // Test case: ambiguous input should trigger clarification questions
    server := setupTestServer(t, "intent_detection_only")
    defer server.Close()

    req := createTestRequest("Show me something")
    resp := sendRequest(t, server, req)
    reasoning := extractReasoningBlock(t, resp)

    // Should have multiple intents with similar confidence
    assert.Contains(t, reasoning, "confidence:")
    // Should have clarification questions
    assert.Contains(t, reasoning, "Clarification")
}

func TestIntentDetectionAgent_LLMFallback(t *testing.T) {
    // Test case: low confidence should trigger LLM fallback
    server := setupTestServer(t, "intent_detection_only")
    defer server.Close()

    req := createTestRequest("Can you help me understand how to configure the proxy?")
    resp := sendRequest(t, server, req)
    reasoning := extractReasoningBlock(t, resp)

    // Should show LLM fallback triggered
    assert.Contains(t, reasoning, "💬 LLM FALLBACK TRIGGERED:")
    assert.Contains(t, reasoning, "LLM calls:")
}
```

### 6. Helper Utilities

**TestServer** - утилита для запуска тест-сервера с single-agent workflow:

```go
// tests/golang/integration/single_agent/test_server.go
type TestServer struct {
    httpServer *httptest.Server
    config     *config.Config
}

func NewTestServer(workflowName string) *TestServer {
    // Create config with single workflow
    cfg := &config.Config{
        Server: config.ServerConfig{
            Host: "localhost",
            Port: 0, // random port
        },
        Workflows: config.WorkflowsConfig{
            Default: workflowName,
            Enabled: []string{workflowName},
        },
    }

    // Create and start server
    server := setupHTTPServer(cfg)

    return &TestServer{
        httpServer: httptest.NewServer(server),
        config:     cfg,
    }
}

func (ts *TestServer) URL() string {
    return ts.httpServer.URL
}

func (ts *TestServer) Close() {
    ts.httpServer.Close()
}
```

**MockContextBuilder** - строитель mock contexts с preconditions:

```go
// tests/golang/integration/single_agent/mock_context_builder.go
type MockContextBuilder struct {
    ctx *models.AgentContext
}

func NewMockContextBuilder() *MockContextBuilder {
    return &MockContextBuilder{
        ctx: models.NewAgentContext("test-session", "test-trace"),
    }
}

func (b *MockContextBuilder) WithIntents(intents ...models.Intent) *MockContextBuilder {
    b.ctx.Reasoning.Intents = intents
    return b
}

func (b *MockContextBuilder) WithHypotheses(hypotheses ...models.Hypothesis) *MockContextBuilder {
    b.ctx.Reasoning.Hypotheses = hypotheses
    return b
}

func (b *MockContextBuilder) WithPlans(plans ...models.RetrievalPlan) *MockContextBuilder {
    b.ctx.Retrieval.Plans = plans
    return b
}

func (b *MockContextBuilder) Build() *models.AgentContext {
    return b.ctx
}

// Example usage:
ctx := NewMockContextBuilder().
    WithIntents(
        models.Intent{Type: "query_commits", Confidence: 0.99},
    ).
    WithHypotheses(
        models.Hypothesis{ID: "h0", Description: "Retrieve commits from GitLab"},
    ).
    Build()
```

**ResponseValidator** - валидатор ответов агента:

```go
// tests/golang/integration/single_agent/response_validator.go
type ResponseValidator struct {
    reasoning string
}

func NewResponseValidator(reasoning string) *ResponseValidator {
    return &ResponseValidator{reasoning: reasoning}
}

func (v *ResponseValidator) AssertHasInput() *ResponseValidator {
    if !strings.Contains(v.reasoning, "📥 INPUT:") {
        panic("Response missing INPUT section")
    }
    return v
}

func (v *ResponseValidator) AssertHasOutput() *ResponseValidator {
    if !strings.Contains(v.reasoning, "📤 OUTPUT:") {
        panic("Response missing OUTPUT section")
    }
    return v
}

func (v *ResponseValidator) AssertHasSystemPrompt() *ResponseValidator {
    if !strings.Contains(v.reasoning, "=== 📤 SYSTEM PROMPT FOR LLM ===") {
        panic("Response missing system prompt section")
    }
    return v
}

func (v *ResponseValidator) AssertSingleAgent(expectedAgentID string) *ResponseValidator {
    // Parse agent executions from reasoning block
    agents := extractAgentExecutions(v.reasoning)
    if len(agents) != 1 {
        panic(fmt.Sprintf("Expected 1 agent, got %d", len(agents)))
    }
    if agents[0].AgentID != expectedAgentID {
        panic(fmt.Sprintf("Expected agent %s, got %s", expectedAgentID, agents[0].AgentID))
    }
    return v
}

// Example usage:
NewResponseValidator(reasoning).
    AssertHasInput().
    AssertHasOutput().
    AssertHasSystemPrompt().
    AssertSingleAgent("intent_detection")
```

## Файловая структура

```
src/golang/pkg/workflows/
├── intent_detection_only.go
├── reasoning_structure_only.go
├── retrieval_planner_only.go
├── retrieval_executor_only.go
├── context_synthesizer_only.go
├── inference_only.go
├── summarization_only.go
└── validation_only.go

tests/golang/integration/single_agent/
├── test_server.go                    # TestServer utility
├── mock_context_builder.go           # MockContextBuilder
├── response_validator.go             # ResponseValidator
├── helpers.go                        # Common test helpers
├── intent_detection_test.go          # IntentDetection tests
├── reasoning_structure_test.go       # ReasoningStructure tests
├── retrieval_planner_test.go         # RetrievalPlanner tests
├── retrieval_executor_test.go        # RetrievalExecutor tests
├── context_synthesizer_test.go       # ContextSynthesizer tests
├── inference_test.go                 # Inference tests
├── summarization_test.go             # Summarization tests
└── validation_test.go                # Validation tests

tests/golang/fixtures/
├── intent_fixtures.go                # Mock intents
├── hypothesis_fixtures.go            # Mock hypotheses
├── plan_fixtures.go                  # Mock plans
├── artifact_fixtures.go              # Mock artifacts
└── context_fixtures.go               # Complete mock contexts
```

## Usage Examples

### Запуск сервера с одним агентом

```bash
# Start server with intent_detection_only workflow
./bin/proxy --config config.yaml --workflow intent_detection_only

# Test with curl
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Workflow: intent_detection_only" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "What are my recent commits?"}],
    "stream": true
  }'
```

### Запуск интеграционных тестов

```bash
# Run all single-agent tests
go test ./tests/golang/integration/single_agent/... -v

# Run tests for specific agent
go test ./tests/golang/integration/single_agent/intent_detection_test.go -v

# Run with coverage
go test ./tests/golang/integration/single_agent/... -cover

# Run specific test case
go test ./tests/golang/integration/single_agent/intent_detection_test.go -run TestIntentDetectionAgent_QueryCommits -v
```

## Success Criteria

- ✅ 8 single-agent workflows реализованы
- ✅ Mock data providers для всех preconditions созданы
- ✅ TestServer, MockContextBuilder, ResponseValidator реализованы
- ✅ Интеграционные тесты для всех 8 агентов написаны
- ✅ Каждый тест проверяет INPUT, OUTPUT, system prompt
- ✅ Каждый тест проверяет что выполнился только один агент
- ✅ Все тесты проходят
- ✅ Документация по использованию написана

## Benefits

1. **Быстрая разработка**: можно работать над одним агентом изолированно
2. **Простой дебаггинг**: легко найти проблему в конкретном агенте
3. **Быстрые тесты**: тестирование одного агента занимает миллисекунды
4. **Итеративная разработка**: можно улучшать агенты по одному
5. **Ясная обратная связь**: четко видно что агент получает и что возвращает
6. **Упрощенное тестирование**: не нужно настраивать полный pipeline
