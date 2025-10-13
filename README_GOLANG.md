# ADK LLM Proxy - Golang Implementation

High-performance OpenAI-compatible LLM proxy with reasoning workflows, built in Go.

## Features

- **OpenAI API Compatible** - Works with gptel, curl, and any OpenAI-compatible client
- **Multi-Provider Support** - OpenAI, Anthropic, DeepSeek, Ollama
- **3 Reasoning Workflows**:
  - `default`: Simple pass-through
  - `basic`: Intent detection via regex/keywords
  - `advanced`: Multi-agent orchestration (ADK + OpenAI)
- **Async Streaming** - SSE streaming for real-time responses
- **High Performance** - Connection pooling, goroutines, 10K+ req/s capable
- **Clean Architecture** - DDD layers (domain, application, infrastructure, presentation)

## Quick Start

### Prerequisites

- Go 1.21+ installed
- OpenAI API key (for OpenAI provider)
- Set environment variables:
  ```bash
  export OPENAI_API_KEY=your-key-here
  ```

### Build and Run

```bash
# Build
make go-build

# Run with default config
make go-run

# Or run directly
./bin/proxy --config config-golang.yaml --port 8001
```

### Test the Proxy

```bash
# Health check
curl http://localhost:8001/health

# List available workflows
curl http://localhost:8001/workflows

# Test streaming completion
curl -X POST http://localhost:8001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-Workflow: basic" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello, how are you?"}],
    "stream": true
  }'
```

## Configuration

Edit `config-golang.yaml` to configure providers, workflows, and performance settings.

### Key Configuration Sections

**Providers:**
```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    enabled: true
```

**Workflows:**
```yaml
workflows:
  default: "basic"  # Options: default, basic, advanced
  enabled:
    - "default"
    - "basic"
    - "advanced"
```

### CLI Flags

```bash
./bin/proxy \
  --config config-golang.yaml \  # Config file path
  --host 0.0.0.0 \                # Server host (overrides config)
  --port 8001 \                   # Server port (overrides config)
  --workflow basic                # Default workflow (overrides config)
```

## Emacs gptel Integration

### Installation

1. Install gptel in Emacs:
   ```elisp
   M-x package-install RET gptel RET
   ```

2. Add configuration to your `init.el`:
   ```elisp
   (use-package gptel
     :config
     (setq gptel-backend
           (gptel-make-openai "ADK Proxy"
             :host "localhost:8001"
             :endpoint "/v1/chat/completions"
             :stream t
             :key "dummy"
             :models '("gpt-4o-mini" "gpt-4o" "claude-3-5-sonnet-20241022")))
     (setq gptel-model "gpt-4o-mini")
     (setq gptel-api-extra-headers '(("X-Workflow" . "basic"))))
   ```

   See `examples/emacs-gptel-config.el` for full configuration.

3. Start the proxy:
   ```bash
   make go-run
   ```

4. Use gptel in Emacs:
   ```
   M-x gptel
   ```

### Workflow Selection

Change workflow by modifying the `X-Workflow` header:

```elisp
;; Use basic workflow (intent detection)
(setq gptel-api-extra-headers '(("X-Workflow" . "basic")))

;; Use advanced workflow (multi-agent)
(setq gptel-api-extra-headers '(("X-Workflow" . "advanced")))
```

## API Endpoints

### POST /v1/chat/completions

OpenAI-compatible chat completions endpoint.

**Request:**
```json
{
  "model": "gpt-4o-mini",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true
}
```

**Response (streaming):**
```
data: {"type":"reasoning","data":{"workflow_name":"basic","message":"Detected intent: greeting","intent":"greeting","confidence":0.95}}

data: {"type":"completion","data":{"id":"chatcmpl-123","object":"chat.completion.chunk","model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"Hello"}}]}}

data: {"type":"done","data":{"status":"complete"}}
```

### GET /health

Health check endpoint.

**Response:**
```json
{"status":"ok"}
```

### GET /workflows

List available workflows.

**Response:**
```json
{
  "workflows": ["default", "basic", "advanced"],
  "default_workflow": "basic"
}
```

## Architecture

```
src/golang/
├── cmd/proxy/          # Main entry point
├── internal/
│   ├── domain/         # Core business logic (interfaces, models)
│   ├── application/    # Use cases (orchestration, streaming)
│   ├── infrastructure/ # External systems (providers, config, agents)
│   └── presentation/   # API layer (HTTP handlers)
└── pkg/workflows/      # Public workflow implementations
```

### Design Principles

- **DDD (Domain-Driven Design)** - Clear layer separation
- **SOLID Principles** - Interface-based design, dependency injection
- **Clean Architecture** - Dependencies point inward toward domain
- **High Performance** - Connection pooling, goroutines, channels

## Development

### Run Tests

```bash
# All tests
make go-test

# With coverage
make go-test-coverage
```

### Lint Code

```bash
make go-lint
```

### Clean Build Artifacts

```bash
make go-clean
```

## Workflows

### Default Workflow

Simple pass-through that returns "Hello World". Useful for testing.

```bash
curl -X POST http://localhost:8001/v1/chat/completions \
  -H "X-Workflow: default" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"test"}],"stream":true}'
```

### Basic Workflow

Intent detection using regex and keywords. No LLM calls for reasoning - very fast.

**Detected Intents:**
- `question` - Questions (what, why, how, ?)
- `greeting` - Greetings (hello, hi, hey)
- `code_request` - Code requests (implement, write, create)
- `explanation` - Explanations (explain, describe)
- `debug` - Debugging (fix, error, bug)
- `general` - Everything else

### Advanced Workflow

Multi-agent orchestration:
- **ADK Agent** - Python subprocess executing Google ADK
- **OpenAI Agent** - Native Go LLM call for fast reasoning

Agents run in parallel for maximum performance.

## Troubleshooting

### Proxy won't start

```bash
# Check if port is in use
lsof -i :8001

# Check config is valid
cat config-golang.yaml

# Check environment variables
echo $OPENAI_API_KEY
```

### Emacs gptel not connecting

```bash
# Test proxy manually
curl http://localhost:8001/health

# Test with streaming
curl -X POST http://localhost:8001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hi"}],"stream":true}'

# Check Emacs gptel configuration
# In Emacs: M-x describe-variable RET gptel-backend RET
```

### Provider errors

```bash
# Check API key is set
echo $OPENAI_API_KEY

# Check provider is enabled in config
grep -A5 "openai:" config-golang.yaml

# Test provider directly
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

## Performance

**Target Benchmarks:**
- Throughput: >10K req/s (non-streaming)
- Latency: <10ms (p50), <50ms (p99)
- Memory: <100MB (idle), <500MB (load)
- Startup: <100ms

## Comparison: Python vs Golang

| Feature | Python Implementation | Golang Implementation |
|---------|----------------------|----------------------|
| Performance | ~1K req/s | >10K req/s |
| Memory Usage | ~200MB | ~50MB |
| Startup Time | ~2s | <100ms |
| Binary Size | N/A (interpreted) | ~15MB (single binary) |
| Deployment | Requires Python + deps | Single binary |
| MCP Integration | Native | Via Python subprocess |
| Concurrency | asyncio (single-threaded) | Goroutines (multi-threaded) |

## License

See main project LICENSE file.
