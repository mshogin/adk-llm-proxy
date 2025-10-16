# Integration Tests for LLM Providers

This directory contains integration tests that verify the ADK LLM Proxy works correctly with real LLM provider APIs.

## Overview

Integration tests in this directory:
- Test real API calls to OpenAI, Anthropic, DeepSeek, and Ollama
- Verify streaming and non-streaming modes
- Test LLM orchestrator with real providers
- Validate cost tracking, budget enforcement, and caching
- Test concurrent request handling
- Verify fallback chains work in production

## Prerequisites

### API Keys

To run tests with cloud providers, you need valid API keys:

```bash
# OpenAI (required for most tests)
export OPENAI_API_KEY="sk-..."

# Anthropic (optional)
export ANTHROPIC_API_KEY="sk-ant-..."

# DeepSeek (optional)
export DEEPSEEK_API_KEY="sk-..."
```

### Local Services

For Ollama tests, you need a running local instance:

```bash
# Install Ollama (macOS)
brew install ollama

# Start Ollama service
ollama serve

# Pull the mistral model
ollama pull mistral
```

## Running Tests

### Run All Integration Tests

```bash
# From project root
go test ./tests/golang/integration/... -v

# With timeout (recommended for API tests)
go test ./tests/golang/integration/... -v -timeout 5m
```

### Run Specific Provider Tests

```bash
# OpenAI only
go test ./tests/golang/integration/... -v -run TestOpenAIProvider

# Anthropic only
go test ./tests/golang/integration/... -v -run TestAnthropicProvider

# DeepSeek only
go test ./tests/golang/integration/... -v -run TestDeepSeekProvider

# Ollama only
go test ./tests/golang/integration/... -v -run TestOllamaProvider

# Orchestrator tests
go test ./tests/golang/integration/... -v -run TestLLMOrchestrator
```

### Skip Tests Without API Keys

Tests automatically skip if required API keys are not set:

```bash
# This will skip OpenAI, Anthropic, and DeepSeek tests
go test ./tests/golang/integration/... -v

# Output:
# --- SKIP: TestOpenAIProvider_Integration (0.00s)
#     llm_providers_integration_test.go:18: Skipping test: OPENAI_API_KEY environment variable not set
```

## Test Coverage

### TestOpenAIProvider_Integration
**Tests:** OpenAI API integration
- Non-streaming completion
- Streaming completion
- Timeout handling
- Error handling

**Models tested:** `gpt-4o-mini`

**Skip condition:** No `OPENAI_API_KEY`

### TestAnthropicProvider_Integration
**Tests:** Anthropic API integration
- Non-streaming completion
- Streaming completion with Claude Haiku

**Models tested:** `claude-3-haiku-20240307`

**Skip condition:** No `ANTHROPIC_API_KEY`

### TestDeepSeekProvider_Integration
**Tests:** DeepSeek API integration
- Non-streaming completion
- Streaming completion with DeepSeek Chat

**Models tested:** `deepseek-chat`

**Skip condition:** No `DEEPSEEK_API_KEY`

### TestOllamaProvider_Integration
**Tests:** Ollama local provider integration
- Non-streaming completion
- Streaming completion with Mistral

**Models tested:** `mistral`

**Skip condition:** Ollama service not running

### TestLLMOrchestrator_Integration
**Tests:** LLM orchestrator with real providers
- Model selection for intent classification (cheap models)
- Model selection for advanced reasoning (expensive models)
- Budget tracking across multiple agents
- Decision logging
- Caching functionality
- Cache statistics

**Skip condition:** No API keys available

### TestProviderFallback_Integration
**Tests:** Fallback chain functionality
- Primary model unavailable → fallback to secondary
- Verifies fallback selection logic

**Skip condition:** No `OPENAI_API_KEY`

### TestConcurrentRequests_Integration
**Tests:** Concurrent API calls
- 5 parallel requests to same provider
- Thread-safety of orchestrator
- No race conditions

**Skip condition:** No `OPENAI_API_KEY`

### TestCostTracking_Integration
**Tests:** Accurate cost calculation
- Multiple agents with different models
- Session budget aggregation
- Per-agent budget tracking
- Verification of cost per 1K tokens

**Skip condition:** No `OPENAI_API_KEY`

## Expected Costs

Running all integration tests will incur small API costs:

| Provider | Tests | Tokens | Estimated Cost |
|----------|-------|--------|---------------|
| OpenAI | 8 tests | ~5,000 | $0.001 |
| Anthropic | 2 tests | ~500 | $0.0001 |
| DeepSeek | 2 tests | ~500 | $0.00005 |
| Ollama | 2 tests | ~500 | $0 (local) |

**Total estimated cost per run:** ~$0.0012 (~$0.001)

## Interpreting Results

### Successful Test Output

```
=== RUN   TestOpenAIProvider_Integration
=== RUN   TestOpenAIProvider_Integration/Non-Streaming
    llm_providers_integration_test.go:55: OpenAI response: Hello from integration test
=== RUN   TestOpenAIProvider_Integration/Streaming
    llm_providers_integration_test.go:75: OpenAI streaming: received 12 chunks
=== RUN   TestOpenAIProvider_Integration/Timeout
--- PASS: TestOpenAIProvider_Integration (2.34s)
    --- PASS: TestOpenAIProvider_Integration/Non-Streaming (1.12s)
    --- PASS: TestOpenAIProvider_Integration/Streaming (1.18s)
    --- PASS: TestOpenAIProvider_Integration/Timeout (0.04s)
```

### Common Failures

**API Key Invalid:**
```
Error: 401 Unauthorized
Fix: Check your API key is correct and active
```

**Rate Limit Exceeded:**
```
Error: 429 Too Many Requests
Fix: Wait a few seconds and retry, or reduce concurrency
```

**Timeout:**
```
Error: context deadline exceeded
Fix: Increase timeout or check network connectivity
```

**Model Not Found:**
```
Error: model not found
Fix: Verify model name is correct and available in your account
```

**Ollama Not Running:**
```
Skipping Ollama test: service not available (connection refused)
Fix: Start Ollama service with `ollama serve`
```

## Performance Benchmarks

Expected latencies (approximate):

| Provider | Non-Streaming | Streaming (first chunk) |
|----------|--------------|------------------------|
| OpenAI | 500-1500ms | 200-800ms |
| Anthropic | 600-1800ms | 300-1000ms |
| DeepSeek | 800-2000ms | 400-1200ms |
| Ollama | 200-800ms | 100-400ms |

**Note:** Latencies vary based on:
- Network conditions
- Provider load
- Model size
- Context size

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Integration Tests
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          DEEPSEEK_API_KEY: ${{ secrets.DEEPSEEK_API_KEY }}
        run: |
          go test ./tests/golang/integration/... -v -timeout 5m
```

### Add secrets to GitHub repository:
1. Go to Settings → Secrets → Actions
2. Add `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `DEEPSEEK_API_KEY`

## Troubleshooting

### All tests skip with "no API keys available"

**Problem:** No API keys are set

**Solution:**
```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export DEEPSEEK_API_KEY="sk-..."
```

### Tests timeout

**Problem:** Network issues or provider API slow

**Solution:**
```bash
# Increase timeout
go test ./tests/golang/integration/... -v -timeout 10m
```

### Concurrent test fails

**Problem:** Rate limiting from provider

**Solution:** Reduce concurrency in test or add delays

### Cost tracking test has precision errors

**Problem:** Floating point precision in cost calculation

**Solution:** Use `assert.InDelta()` for cost comparisons (already done)

## Best Practices

1. **Run integration tests before release**: Verify all providers work
2. **Monitor API costs**: Track spending in provider dashboards
3. **Use test API keys**: Separate from production keys
4. **Run in CI/CD**: Automate testing on every commit
5. **Check rate limits**: Be aware of provider rate limits
6. **Cache responses**: Use caching to reduce test costs
7. **Skip expensive tests locally**: Run full suite in CI only

## Adding New Provider Tests

To add integration tests for a new provider:

1. Create test function: `TestXXXProvider_Integration`
2. Skip if API key not available
3. Test non-streaming mode
4. Test streaming mode
5. Test timeout handling
6. Add to orchestrator tests
7. Document in this README

Example:
```go
func TestNewProvider_Integration(t *testing.T) {
    skipIfNoAPIKey(t, "NEW_PROVIDER_API_KEY")

    config := providers.ProviderConfig{
        APIKey:  os.Getenv("NEW_PROVIDER_API_KEY"),
        BaseURL: "https://api.newprovider.com/v1",
        Enabled: true,
        Timeout: 30000,
    }

    provider := providers.NewProviderX(config)
    // ... test implementation
}
```

## References

- [Provider Implementation Guide](../../../docs/providers.md)
- [LLM Orchestrator Documentation](../../../docs/reasoning_system/llm_selection_policies.md)
- [Pipeline Configuration](../../../docs/reasoning_system/pipeline_configuration.md)
- [Troubleshooting Guide](../../../docs/reasoning_system/troubleshooting.md)
