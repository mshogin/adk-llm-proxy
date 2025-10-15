# 🗺️ ADK LLM Proxy Roadmap

A comprehensive roadmap for the ADK LLM Proxy project.

> **Note:** Older completed phases have been archived in [ROADMAP-DONE.md](ROADMAP-DONE.md). This file contains only the 3 most recent completed phases and all active/upcoming phases.

---

## Phase 7: Advanced Features & Polish
**Goal**: Production-ready MCP integration with advanced features

### 7.1 Performance Optimization
- [x] Implement MCP tool result caching
- [x] Add parallel tool execution
- [x] Create tool execution prioritization
- [x] Implement smart tool warming
- [x] Add resource usage optimization

### 7.2 Monitoring & Observability
- [x] Add MCP tool execution metrics
- [x] Create MCP server health monitoring
- [x] Implement tool performance tracking
- [x] Add execution tracing and debugging
- [x] Create MCP analytics dashboard

### 7.3 Security & Compliance
- [x] Implement MCP tool sandboxing
- [x] Add tool permission management
- [x] Create audit logging for MCP operations
- [x] Implement rate limiting and quotas
- [x] Add security scanning for MCP servers

### 7.4 Documentation & Examples
- [x] Create comprehensive MCP integration docs
- [x] Add MCP server development guide
- [x] Create example MCP server implementations
- [x] Add troubleshooting guides
- [x] Create video tutorials and demos

## Phase 8: Architectural Best Practices Enhancement
**Goal**: Strengthen Claude instructions with SOLID, Clean Architecture, interface-driven design, and function composition guidelines

**Current State:**
- ✅ DDD architecture well-documented and implemented
- ✅ Layer separation enforced (domain/application/infrastructure/presentation)
- ⬜ SOLID principles mentioned but not deeply explained
- ⬜ Interface-driven design not explicitly documented
- ⬜ Clean Architecture principles (Uncle Bob) not formally referenced
- ⬜ Function composition guidelines minimal

**Target State:**
- ✅ SOLID principles with practical examples for Python and Golang
- ✅ Interface-first development approach documented
- ✅ Clean Architecture mapping to project structure
- ✅ Small, focused, self-explanatory function guidelines
- ✅ Architecture-first planning process documented
- ✅ Enhanced code review checklist with architectural criteria

### 8.1 Add SOLID Principles Section
- [x] Create "SOLID Principles in Practice" section in CLAUDE.md
- [x] Document Single Responsibility Principle (SRP) with examples
- [x] Document Open/Closed Principle (OCP) with examples
- [x] Document Liskov Substitution Principle (LSP) with examples
- [x] Document Interface Segregation Principle (ISP) with examples
- [x] Document Dependency Inversion Principle (DIP) with examples
- [x] Show anti-patterns (what NOT to do) for each principle
- [x] Map SOLID principles to DDD layers
- [x] Add SOLID checklist items for code review

**Implementation notes:**
- Provide both Python and Golang examples for each principle
- Use existing project structure as reference (e.g., MCP client abstractions)
- Show how DIP enables testability through dependency injection
- Emphasize SRP alignment with service separation in DDD

### 8.2 Add Interface-Driven Design Guidelines
- [x] Create "Interface-Driven Design" section in CLAUDE.md
- [x] Document Python interface patterns (ABC, Protocol, type hints)
- [x] Document Golang interface patterns (implicit implementation, composition)
- [x] Explain when to define interfaces (during planning phase)
- [x] Document interface segregation best practices
- [x] Add dependency injection patterns for both languages
- [x] Show mock-friendly design examples
- [x] Add interface design to planning checklist

**Implementation notes:**
- Include Abstract Base Classes (ABC) examples for Python
- Show Python Protocol (PEP 544) for structural subtyping
- Demonstrate Golang's implicit interface satisfaction
- Emphasize small, focused interfaces (ISP)
- Show constructor injection pattern for testability

### 8.3 Add Clean Architecture Section
- [x] Create "Clean Architecture Principles (Uncle Bob)" section in CLAUDE.md
- [x] Explain the Dependency Rule (dependencies point inward)
- [x] Map Clean Architecture circles to project layers
- [x] Document Entities → src/domain/models/ mapping
- [x] Document Use Cases → src/application/services/ mapping
- [x] Document Interface Adapters → src/infrastructure/, src/presentation/ mapping
- [x] Show dependency flow diagrams (ASCII art or description)
- [x] Explain why domain layer has no external dependencies
- [x] Document dependency inversion usage (interfaces in domain)

**Implementation notes:**
- Reference Robert C. Martin's Clean Architecture book
- Show how current DDD structure already follows Clean Architecture
- Emphasize "dependencies point inward" as core rule
- Illustrate with examples from existing codebase (e.g., reasoning service)

### 8.4 Add Function Composition Guidelines
- [x] Create "Function Composition & Readability" section in CLAUDE.md
- [x] Document function size guidelines (20-30 lines max)
- [x] Explain Single Responsibility per function
- [x] Document self-documenting naming conventions
- [x] Show how to avoid nested logic (extract helper functions)
- [x] Demonstrate function composition for complex operations
- [x] Document early return pattern for flat structure
- [x] Add Python function composition examples
- [x] Add Golang function composition examples

**Implementation notes:**
- Show bad examples (large, nested functions) vs good (small, composed)
- Emphasize "one level of abstraction per function"
- Demonstrate how small functions serve as inline documentation
- Show how to reduce cognitive load through composition

### 8.5 Update Code Review Checklist
- [x] Add SOLID principles compliance checks to existing checklist
- [x] Add "Interfaces defined before implementations" check
- [x] Add "Dependency Rule respected" check
- [x] Add "Functions are small and focused (<30 lines)" check
- [x] Add "Function names are self-explanatory" check
- [x] Add "Abstractions at proper layer" check
- [x] Add "No business logic in infrastructure" check
- [x] Add "Testable design (DI, mockable interfaces)" check

**Implementation notes:**
- Extend existing "Code Review Checklist" section
- Make checklist actionable and measurable
- Link checklist items to documentation sections for reference

### 8.6 Add Planning Phase Guidelines
- [x] Create "Architecture-First Planning" section in CLAUDE.md
- [x] Document Step 1: Define interfaces first
- [x] Document Step 2: Map to DDD layers
- [x] Document Step 3: Design for testability
- [x] Document Step 4: Plan function composition
- [x] Document Step 5: Validate against SOLID & Clean Architecture
- [x] Add example planning session (e.g., adding caching feature)
- [x] Create planning template for future features

**Implementation notes:**
- Place before "Development Workflow" section (planning comes first)
- Show complete planning example with interface definitions
- Demonstrate how to validate architecture decisions early
- Emphasize "think before code" approach

## Phase 9: Golang High-Performance Proxy Implementation
**Goal**: Create production-ready Golang implementation of ADK LLM Proxy with OpenAI-compatible API, custom reasoning workflows, and async streaming for Emacs gptel integration

**Current State:**
- ✅ Python implementation fully functional with MCP integration
- ✅ DDD architecture well-established
- ⬜ No Golang implementation exists
- ⬜ Need high-performance alternative for production use
- ⬜ Want single-binary deployment option

**Target State:**
- ✅ Golang proxy with 10K+ req/s throughput
- ✅ OpenAI API compatibility (works with gptel, curl, etc.)
- ✅ Multi-provider support (OpenAI, Anthropic, DeepSeek, Ollama)
- ✅ 3 reasoning workflows (default, basic, advanced)
- ✅ Async streaming (reasoning + inference in parallel)
- ✅ Single binary deployment
- ✅ Emacs gptel integration tested

**Use Case:** Emacs gptel → Golang Proxy (reasoning) → LLM Provider

**Technology Stack:**
- **Language**: Go 1.21+
- **Router**: `go-chi/chi` (lightweight, fast)
- **Config**: `gopkg.in/yaml.v3`
- **Testing**: `testing` + `testify`
- **Streaming**: Server-Sent Events (SSE)
- **Concurrency**: Goroutines + channels

### 9.1 Project Structure & Foundation
- [x] Create `src/golang/` directory structure (cmd, internal, pkg)
- [x] Initialize Go module (`go mod init`)
- [x] Create `cmd/proxy/main.go` entry point
- [x] Set up `internal/` with DDD layers (domain, application, infrastructure, presentation)
- [x] Create `pkg/workflows/` for public workflow implementations
- [x] Add Go dependencies (chi, yaml.v3, testify)
- [x] Create `config.yaml` for Golang proxy
- [x] Set up `tests/golang/` directory mirroring `src/golang/`
- [x] Add `Makefile` targets for Go (build, test, run)

**Implementation notes:**
- Follow Go project layout (cmd/, internal/, pkg/)
- Mirror DDD architecture from Python version
- Use Go modules for dependency management
- Keep internal/ private, pkg/ public
- Single binary output: `bin/proxy`

**Files to create:**
```
src/golang/
├── cmd/proxy/main.go
├── internal/
│   ├── domain/{models,services}/
│   ├── application/services/
│   ├── infrastructure/{config,providers,agents}/
│   └── presentation/api/
├── pkg/workflows/
└── go.mod
```

### 9.2 Domain Layer: Interfaces & Models
- [x] Define `ILLMProvider` interface in `internal/domain/services/provider.go`
- [x] Define `IWorkflow` interface in `internal/domain/services/workflow.go`
- [x] Define `IReasoningService` interface in `internal/domain/services/reasoning.go`
- [x] Create `CompletionRequest` model in `internal/domain/models/request.go`
- [x] Create `CompletionChunk` model (OpenAI-compatible) in `internal/domain/models/response.go`
- [x] Create `ReasoningResult` model in `internal/domain/models/reasoning_result.go`
- [x] Add unit tests for model serialization
- [x] Validate OpenAI schema compatibility

**Implementation notes:**
- Use **interfaces** for all abstractions (Dependency Inversion Principle)
- Keep domain layer **pure** (no external deps except stdlib)
- Use `context.Context` for cancellation
- Add JSON tags for OpenAI API compatibility
- Models must be **immutable** where possible

**Interface examples:**
```go
type LLMProvider interface {
    Name() string
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

type Workflow interface {
    Name() string
    Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error)
}
```

### 9.3 Infrastructure: LLM Provider Clients
- [x] Implement `OpenAIProvider` in `internal/infrastructure/providers/openai.go`
- [x] Implement `AnthropicProvider` in `internal/infrastructure/providers/anthropic.go`
- [x] Implement `DeepSeekProvider` in `internal/infrastructure/providers/deepseek.go`
- [x] Implement `OllamaProvider` in `internal/infrastructure/providers/ollama.go`
- [x] Create HTTP client pool with connection reuse
- [x] Implement SSE parsing for streaming responses
- [x] Add retry logic with exponential backoff
- [x] Handle provider-specific auth (API keys, headers)
- [x] Add unit tests with mocked HTTP responses
- [x] Add integration tests with real APIs (optional)

**Implementation notes:**
- Use `net/http.Client` with custom `Transport` for pooling
- Parse SSE stream line-by-line (`data: {...}\n\n`)
- Use **goroutines + channels** for async streaming
- Handle rate limiting (429) and retries
- Provider-specific URL patterns:
  - OpenAI: `https://api.openai.com/v1/chat/completions`
  - Anthropic: `https://api.anthropic.com/v1/messages`
  - Ollama: `http://localhost:11434/v1/chat/completions`

**Validation:**
- Test streaming and non-streaming modes
- Test timeout and cancellation
- Benchmark throughput (>1K req/s per provider)

### 9.4 Workflows: Default, Basic, Advanced
- [x] Create `pkg/workflows/workflow.go` base interface
- [x] Implement **Default Workflow** in `pkg/workflows/default.go` (returns "Hello World")
- [x] Implement **Basic Workflow** in `pkg/workflows/basic.go` (intent detection via regex/keywords)
- [x] Implement **Advanced Workflow** in `pkg/workflows/advanced.go` (multi-agent orchestration)
- [x] Create Python ADK agent wrapper script (`workflows/python/adk_agent.py`)
- [x] Implement ADK agent caller in `internal/infrastructure/agents/adk_agent.go` (subprocess)
- [x] Implement OpenAI agent caller in `internal/infrastructure/agents/openai_agent.go` (native SDK)
- [x] Add parallel execution with goroutines in advanced workflow
- [x] Add timeout handling for agent calls
- [x] Create unit tests for each workflow
- [x] Benchmark workflow execution time

**Implementation notes:**

**Default Workflow:**
```go
func (w *DefaultWorkflow) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error) {
    return &ReasoningResult{Message: "Hello World"}, nil
}
```

**Basic Workflow:**
```go
// Detect intent using regex/keywords (no LLM call)
func (w *BasicWorkflow) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error) {
    intent := detectIntent(input.Messages)  // Simple pattern matching
    return &ReasoningResult{Message: fmt.Sprintf("Intent: %s", intent)}, nil
}
```

**Advanced Workflow:**
```go
// Multi-agent: ADK (Python subprocess) + OpenAI (native Go)
func (w *AdvancedWorkflow) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error) {
    adkChan := make(chan *AgentResult)
    openaiChan := make(chan *AgentResult)

    // Parallel agent execution
    go w.callADKAgent(ctx, input, adkChan)
    go w.callOpenAIAgent(ctx, input, openaiChan)

    // Wait for both with timeout
    adk := <-adkChan
    openai := <-openaiChan

    return w.aggregateResults(adk, openai), nil
}
```

**ADK Agent (Python subprocess):**
- Create `workflows/python/adk_agent.py` wrapper
- Use `os/exec` to call Python script with JSON stdin/stdout
- Parse JSON response from subprocess

**Validation:**
- Test each workflow independently
- Test workflow selection based on config
- Test ADK subprocess communication
- Benchmark: Default <1ms, Basic <5ms, Advanced <500ms

### 9.5 Application Layer: Orchestration & Streaming
- [x] Create `Orchestrator` in `internal/application/services/orchestrator.go`
- [x] Implement `ProcessRequest()` with async reasoning + inference
- [x] Create `StreamEvent` model for SSE events
- [x] Implement event channel for reasoning/completion streaming
- [x] Add workflow selection logic (from config or header)
- [x] Add provider selection logic (based on model name)
- [x] Implement graceful error handling (send errors as events)
- [x] Add context cancellation handling (client disconnect)
- [x] Create streaming coordinator in `internal/application/services/streaming.go`
- [x] Add unit tests for orchestration pipeline
- [x] Add integration tests for full request flow

**Implementation notes:**

**Async Pipeline:**
```go
func (o *Orchestrator) ProcessRequest(ctx context.Context, req *CompletionRequest, workflow string) (<-chan StreamEvent, error) {
    eventChan := make(chan StreamEvent, 10)

    go func() {
        defer close(eventChan)

        // Phase 1: Reasoning (async)
        wf := o.workflows[workflow]
        reasoningResult, err := wf.Execute(ctx, extractInput(req))
        eventChan <- StreamEvent{Type: "reasoning", Data: reasoningResult}

        // Phase 2: LLM Inference (async streaming)
        provider := o.getProvider(req.Model)
        chunkChan, _ := provider.StreamCompletion(ctx, req)

        for chunk := range chunkChan {
            eventChan <- StreamEvent{Type: "completion", Data: chunk}
        }

        eventChan <- StreamEvent{Type: "done"}
    }()

    return eventChan, nil
}
```

**Event Types:**
- `reasoning`: Workflow result
- `completion`: LLM chunk
- `error`: Error message
- `done`: Stream complete

**Validation:**
- Test reasoning + inference parallelism
- Test client disconnect (context cancellation)
- Test concurrent request handling (100+ parallel)
- Benchmark latency and throughput

### 9.6 Presentation Layer: OpenAI-Compatible API
- [x] Create HTTP handler in `internal/presentation/api/handlers.go`
- [x] Implement `POST /v1/chat/completions` (OpenAI-compatible)
- [x] Implement `GET /health` endpoint
- [x] Implement `GET /workflows` endpoint (list available workflows)
- [x] Add SSE streaming response handler
- [x] Add non-streaming (buffered) response handler
- [x] Create middleware in `internal/presentation/api/middleware.go` (logging, CORS, recovery)
- [x] Add workflow selection via `X-Workflow` header
- [x] Implement graceful shutdown (SIGTERM/SIGINT)
- [x] Add request validation middleware
- [x] Create unit tests for handlers
- [x] Create integration tests with httptest

**Implementation notes:**

**OpenAI-Compatible Endpoint:**
```go
// POST /v1/chat/completions
func (h *Handler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
    var req models.CompletionRequest
    json.NewDecoder(r.Body).Decode(&req)

    workflow := r.Header.Get("X-Workflow")
    if workflow == "" {
        workflow = h.config.Workflows.Default
    }

    if req.Stream {
        h.streamResponse(w, r, &req, workflow)
    } else {
        h.bufferResponse(w, r, &req, workflow)
    }
}
```

**SSE Streaming:**
```go
func (h *Handler) streamResponse(w http.ResponseWriter, r *http.Request, req *CompletionRequest, workflow string) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")

    flusher := w.(http.Flusher)
    eventChan, _ := h.orchestrator.ProcessRequest(r.Context(), req, workflow)

    for event := range eventChan {
        fmt.Fprintf(w, "data: %s\n\n", formatSSE(event))
        flusher.Flush()
    }
}
```

**Endpoints:**
- `POST /v1/chat/completions` (OpenAI-compatible)
- `GET /health` (health check)
- `GET /workflows` (list workflows)

**Validation:**
- Test with `curl` (OpenAI request format)
- Test SSE streaming in browser
- Test workflow selection via header
- Load test with `hey` or `wrk` (10K+ req/s target)

### 9.7 Configuration & CLI
- [x] Create config loader in `internal/infrastructure/config/config.go`
- [x] Define YAML config structure (server, providers, workflows, advanced)
- [x] Implement environment variable expansion (`${VAR}`)
- [x] Add CLI flags in `cmd/proxy/main.go` (--config, --host, --port, --workflow)
- [x] Implement CLI flag overrides for config
- [x] Add config validation (required fields, valid values)
- [x] Create example `config.yaml` with all providers
- [x] Document config options in README
- [x] Add unit tests for config loading
- [x] Test environment variable expansion

**Implementation notes:**

**Config Structure:**
```go
type Config struct {
    Server struct {
        Host string `yaml:"host"`
        Port int    `yaml:"port"`
    } `yaml:"server"`

    Providers map[string]ProviderConfig `yaml:"providers"`

    Workflows struct {
        Default string   `yaml:"default"`
        Enabled []string `yaml:"enabled"`
    } `yaml:"workflows"`

    Advanced AdvancedConfig `yaml:"advanced"`
}
```

**Example config.yaml:**
```yaml
server:
  host: "0.0.0.0"
  port: 8001

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    enabled: true
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com/v1"
    enabled: true
  ollama:
    base_url: "http://localhost:11434/v1"
    enabled: true

workflows:
  default: "basic"
  enabled: ["default", "basic", "advanced"]

advanced:
  adk_agent_path: "workflows/python/adk_agent.py"
  openai_api_key: "${OPENAI_API_KEY}"
```

**CLI Flags:**
```bash
./proxy --config config.yaml --port 8001 --workflow basic
```

**Validation:**
- Config loads from YAML
- CLI flags override config values
- Environment variables expand correctly
- Invalid config shows helpful errors

### 9.8 Testing, Documentation & Emacs Integration
- [x] Create integration test in `tests/golang/integration/e2e_test.go`
- [x] Test OpenAI-compatible request/response
- [x] Test SSE streaming events (reasoning + completion + done)
- [x] Test all 3 workflows (default, basic, advanced)
- [x] Test provider selection based on model name
- [x] Create `README_GOLANG.md` with setup instructions
- [x] Document CLI usage and config options
- [x] Add Emacs gptel configuration example
- [x] Create `examples/emacs-gptel-config.el`
- [x] Test gptel integration in Emacs
- [x] Add performance benchmarks to README
- [x] Create troubleshooting guide
- [x] Add comparison table (Python vs Golang)

**Implementation notes:**

**E2E Test:**
```go
func TestChatCompletionsStreaming(t *testing.T) {
    server := setupTestServer()
    defer server.Close()

    req := makeOpenAIRequest("gpt-4o-mini", "Hello", true)
    resp := sendRequest(server, req)

    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

    events := parseSSEEvents(resp.Body)
    assert.Contains(t, events, "reasoning")
    assert.Contains(t, events, "completion")
    assert.Contains(t, events, "done")
}
```

**Emacs gptel Config:**
```elisp
;; .emacs.d/init.el or examples/emacs-gptel-config.el
(use-package gptel
  :config
  (setq gptel-backend
        (gptel-make-openai "ADK Proxy (Golang)"
          :host "localhost:8001"
          :endpoint "/v1/chat/completions"
          :stream t
          :key "dummy"  ; Not required for local proxy
          :models '("gpt-4o-mini" "gpt-4o" "claude-3-5-sonnet" "deepseek-chat")))

  ;; Set workflow via custom header
  (setq gptel-api-extra-headers
        '(("X-Workflow" . "basic"))))  ; or "default", "advanced"

;; Usage:
;; M-x gptel
;; Type your prompt, press C-c RET
```

**README_GOLANG.md Contents:**
- Quick start (build, run, test)
- Configuration guide
- Workflow descriptions
- API endpoints
- Emacs integration
- Performance benchmarks
- Troubleshooting
- Python vs Golang comparison

**Validation:**
- All tests pass (`go test ./...`)
- E2E test with real LLM providers
- gptel works in Emacs
- README instructions accurate
- Performance benchmarks documented (throughput, latency)

**Performance Targets:**
- Throughput: >10K req/s (non-streaming)
- Latency: <10ms (p50), <50ms (p99)
- Memory: <100MB (idle), <500MB (load)
- Startup: <100ms

---

## Phase 10: Reasoning Enrichment Agent System
**Goal**: Build modular multi-agent reasoning system with dynamic LLM selection, versioned context, and complete traceability

**Current State:**
- ✅ Basic reasoning workflows (default, basic, advanced) implemented
- ✅ Single-agent orchestration with Python ADK + OpenAI
- ⬜ No structured multi-agent pipeline
- ⬜ No dynamic LLM model selection by cost/quality
- ⬜ No versioned reasoning context
- ⬜ No agent-level tracing and metrics

**Target State:**
- ✅ Modular agent system: Intent Detection → Structure → Retrieval → Synthesis → Inference → Validation → Summarization
- ✅ Versioned AgentContext with namespace isolation and audit trail
- ✅ Dynamic LLM Orchestrator with cost-aware model selection and fallback chains
- ✅ Complete traceability: agent runs, diffs, performance, token usage, costs
- ✅ Sequential, parallel, and conditional execution modes
- ✅ <500ms non-LLM latency, budget controls per session/agent

**Architecture:**
```
User Input → Intent Detection → Reasoning Structure → Retrieval Planner
→ Context Synthesizer → Inference → Validation → Summarization → Output
                ↓
         AgentContext (versioned, namespaced, audited)
                ↓
         LLM Orchestrator (dynamic model selection, cost tracking)
```

### 10.1 AgentContext Infrastructure

**Goal**: Create versioned, namespaced context store with audit and validation

- [x] Define AgentContext schema in `src/golang/internal/domain/models/agent_context.go`
- [x] Implement namespaces: metadata, reasoning, enrichment, retrieval, llm, diagnostics, audit
- [x] Add version field and migration system for context schema changes
- [x] Create context validator for namespace isolation (agents can't write to foreign keys)
- [x] Implement diff tracker for capturing changes between agent runs
- [x] Add context snapshot/restore functionality
- [x] Create context serialization (JSON, MessagePack) with compression
- [x] Implement context size limits and artifact externalization
- [x] Add utility functions: safe_set, merge, append with validation
- [x] Create unit tests for context operations and validation
- [x] Document AgentContext schema and access patterns

**Implementation notes:**
- Store in `src/golang/internal/domain/models/agent_context.go`
- Namespace structure:
  - `metadata`: session_id, created_at, version, trace_id, locale
  - `reasoning`: intents, entities, hypotheses, conclusions, confidence_scores
  - `enrichment`: facts, derived_knowledge, relationships, context_links
  - `retrieval`: plans, queries, artifacts
  - `llm`: provider, model, usage, cost, decisions
  - `diagnostics`: errors, warnings, performance, validation_reports
  - `audit`: agent_runs[], diffs[]
- Each agent gets own sub-namespace (e.g., `reasoning.intent_detection`)
- Violations of namespace isolation raise `ContextViolationError`

**Files to create:**
```
src/golang/internal/domain/models/
├── agent_context.go          # Core context model
├── context_validator.go      # Namespace validation
└── context_diff.go           # Diff tracking

tests/golang/internal/domain/models/
├── agent_context_test.go
├── context_validator_test.go
└── context_diff_test.go
```

### 10.2 Pipeline Orchestration

**Goal**: Implement flexible agent pipeline with sequential, parallel, and conditional execution

- [x] Define Agent interface in `src/golang/internal/domain/services/reasoning_agent.go`
- [x] Create ReasoningManager in `src/golang/internal/application/services/reasoning_manager.go`
- [x] Implement agent contract validation (preconditions, postconditions)
- [x] Add sequential execution mode (default pipeline)
- [x] Add parallel execution mode for independent agents
- [x] Add conditional execution (run agent if context keys present)
- [x] Implement dependency graph resolution and cycle detection
- [x] Add agent execution metadata tracking (duration, status, output keys)
- [x] Implement retry policy with configurable limits per agent
- [x] Add graceful degradation on agent failures
- [x] Create pipeline configuration loader (YAML/JSON)
- [x] Add unit tests for orchestration modes
- [x] Add integration tests for full pipeline execution
- [x] Document pipeline configuration format

**Implementation notes:**
- Agent interface contract:
  ```go
  type ReasoningAgent interface {
      AgentID() string
      Preconditions() []string  // Required context keys
      Postconditions() []string // Guaranteed output keys
      Execute(ctx context.Context, agentContext *AgentContext) (*AgentContext, error)
  }
  ```
- Pipeline config format:
  ```yaml
  pipeline:
    mode: sequential  # or parallel, conditional
    agents:
      - id: intent_detection
        enabled: true
        timeout: 5s
        retry: 2
      - id: reasoning_structure
        enabled: true
        depends_on: [intent_detection]
  ```
- Track execution in `audit.agent_runs` with timing, status, keys written

**Files to create:**
```
src/golang/internal/domain/services/
└── reasoning_agent.go         # Agent interface

src/golang/internal/application/services/
├── reasoning_manager.go       # Pipeline orchestrator
└── agent_executor.go          # Agent execution wrapper

src/golang/internal/infrastructure/config/
└── pipeline_config.go         # Pipeline configuration loader

tests/golang/internal/application/services/
├── reasoning_manager_test.go
└── agent_executor_test.go
```

### 10.3 LLM Orchestrator

**Goal**: Implement dynamic LLM selection with cost tracking, caching, and fallback chains

- [x] Create LLMOrchestrator in `src/golang/internal/application/services/llm_orchestrator.go`
- [x] Define model profile schema (provider, quality, speed, cost, context_limit)
- [x] Implement provider registry (local Ollama, OpenAI, Anthropic, DeepSeek)
- [x] Add model selection strategy based on task complexity and budget
- [x] Implement cost calculation per request and aggregation per session/agent
- [x] Add response caching with TTL (same input + params = cached response)
- [x] Implement fallback chain (local → mini cloud → large cloud)
- [x] Add throttling and timeout controls per provider
- [x] Implement token usage tracking and limits
- [x] Add decision logging (why model X was selected) to `llm.decisions`
- [x] Create security filters (PII masking, sensitive field truncation)
- [x] Add unit tests for model selection logic
- [ ] Add integration tests with real providers
- [ ] Document model profiles and selection policies

**Implementation notes:**
- Model selection strategy based on task complexity, context size, and budget:
  ```
  Task Type                    | Context  | Default Model           | Fallback 1              | Fallback 2           | Cost/1K tok
  -----------------------------|----------|------------------------|------------------------|---------------------|-------------
  Intent classification        | <500 tok | deepseek/deepseek-chat | openai/gpt-4o-mini     | ollama/mistral      | $0.0001
  Entity extraction            | <1K tok  | deepseek/deepseek-chat | openai/gpt-4o-mini     | ollama/llama3       | $0.0001
  Simple validation            | <1K tok  | deepseek/deepseek-chat | openai/gpt-4o-mini     | rules-only          | $0.0001
  Keyword-based search         | <2K tok  | openai/gpt-4o-mini     | deepseek/deepseek-chat | ollama/mistral      | $0.00015
  Short text synthesis         | <2K tok  | openai/gpt-4o-mini     | deepseek/deepseek-chat | ollama/llama3       | $0.00015
  Query normalization          | <2K tok  | openai/gpt-4o-mini     | deepseek/deepseek-chat | -                   | $0.00015
  Fact deduplication           | <3K tok  | openai/gpt-4o-mini     | deepseek/deepseek-chat | ollama/mistral      | $0.00015
  Simple inference (rules)     | <3K tok  | openai/gpt-4o-mini     | deepseek/deepseek-chat | -                   | $0.00015
  Medium synthesis             | <8K tok  | openai/gpt-4o          | anthropic/claude-haiku | openai/gpt-4o-mini  | $0.0025
  Complex retrieval planning   | <8K tok  | openai/gpt-4o          | anthropic/claude-haiku | deepseek/deepseek-r1| $0.0025
  Multi-source correlation     | <16K tok | openai/gpt-4o          | anthropic/claude-sonnet| deepseek/deepseek-r1| $0.0025
  Advanced inference           | <16K tok | openai/gpt-4o          | anthropic/claude-sonnet| deepseek/deepseek-r1| $0.0025
  Long-context analysis        | <32K tok | anthropic/claude-sonnet| openai/gpt-4o          | -                   | $0.003
  Deep reasoning (multi-step)  | <64K tok | openai/o1-mini         | anthropic/claude-sonnet| openai/gpt-4o       | $0.015
  Critical reasoning (CoT)     | <100K tok| openai/o1              | anthropic/claude-opus  | openai/o1-mini      | $0.06
  ```

- **Task complexity detection heuristics:**
  - Intent classification: single user message, <50 words
  - Entity extraction: structured extraction, <100 words
  - Simple validation: boolean checks, <200 words
  - Short synthesis: summarization, <500 words context
  - Medium synthesis: multiple sources, <2000 words context
  - Complex planning: multi-step with dependencies
  - Advanced inference: requires logical reasoning chains
  - Deep reasoning: multi-hop reasoning, ambiguity resolution
  - Critical reasoning: high-stakes decisions, need chain-of-thought

- **Budget-aware selection:**
  - If session budget >80% spent: downgrade to cheaper model or skip LLM
  - If agent budget exceeded: use deterministic fallback or cache-only
  - Emergency degradation: critical agents get priority, non-critical agents skip LLM

- **Caching strategy:**
  - Cache TTL by task type: classification (24h), synthesis (1h), inference (30min)
  - Cache hit rate target: >40% for repeated patterns
  - Cache key includes: prompt hash + model + temperature + max_tokens

- Cost tracking in `llm.usage`:
  ```go
  type LLMUsage struct {
      TotalTokens      int                `json:"total_tokens"`
      PromptTokens     int                `json:"prompt_tokens"`
      CompletionTokens int                `json:"completion_tokens"`
      CostUSD          float64            `json:"cost_usd"`
      ByAgent          map[string]float64 `json:"by_agent"`
  }
  ```
- Cache key: hash(prompt + model + temperature + other params)

**Files to create:**
```
src/golang/internal/application/services/
├── llm_orchestrator.go        # Dynamic model selection
└── llm_cache.go              # Response caching

src/golang/internal/domain/models/
└── llm_profile.go            # Model profile definition

src/golang/internal/infrastructure/llm/
├── provider_registry.go      # Provider management
└── cost_calculator.go        # Cost tracking

tests/golang/internal/application/services/
├── llm_orchestrator_test.go
└── llm_cache_test.go
```

### 10.4 Core Reasoning Agents

**Goal**: Implement 7 specialized reasoning agents following single-responsibility principle

**10.4.1 Intent Detection Agent**
- [ ] Create IntentDetectionAgent in `src/golang/internal/domain/services/agents/intent_detection.go`
- [ ] Implement rule-based intent classification (regex, keywords)
- [ ] Add lightweight local model for intent detection (Ollama)
- [ ] Fallback to cloud LLM for low-confidence cases (< 0.8)
- [ ] Extract entities: projects, dates, providers, statuses
- [ ] Output: `reasoning.intents[]`, `reasoning.entities{}`, confidence scores
- [ ] Add clarification questions for ambiguous intents
- [ ] Create unit tests with various input patterns
- [ ] Document supported intent types and confidence thresholds

**10.4.2 Reasoning Structure Agent**
- [ ] Create ReasoningStructureAgent in `src/golang/internal/domain/services/agents/reasoning_structure.go`
- [ ] Build reasoning goal hierarchy from detected intents
- [ ] Generate hypotheses and assumptions
- [ ] Create dependency graph between reasoning steps
- [ ] Output: `reasoning.hypotheses[]`, dependency map, expected artifacts
- [ ] Add cycle detection in dependency graph
- [ ] Create unit tests for structure generation
- [ ] Document reasoning structure format

**10.4.3 Information Retrieval Planner**
- [ ] Create RetrievalPlannerAgent in `src/golang/internal/domain/services/agents/retrieval_planner.go`
- [ ] Generate retrieval plans for RAG and analytics sources
- [ ] Prioritize structured data sources (GitLab, YouTrack) over unstructured
- [ ] Create normalized queries with filters (project, date, status, provider)
- [ ] Output: `retrieval.plans[]`, `retrieval.queries[]`
- [ ] Add time and volume constraints per source
- [ ] Create unit tests for plan generation
- [ ] Document query format and source types

**10.4.4 Context Synthesizer Agent**
- [ ] Create ContextSynthesizerAgent in `src/golang/internal/domain/services/agents/context_synthesizer.go`
- [ ] Normalize and merge facts from multiple sources
- [ ] Implement deduplication logic
- [ ] Unify schemas across different data sources
- [ ] Track fact provenance (source, timestamp, confidence)
- [ ] Output: `enrichment.facts[]`, `enrichment.derived_knowledge[]`, relationships
- [ ] Create unit tests for normalization and deduplication
- [ ] Document fact schema and relationship types

**10.4.5 Inference Agent**
- [ ] Create InferenceAgent in `src/golang/internal/domain/services/agents/inference.go`
- [ ] Make conclusions based on facts, hypotheses, and goals
- [ ] Assess conclusion confidence scores
- [ ] Generate alternative interpretations for ambiguous cases
- [ ] Use deterministic rules for simple inferences
- [ ] Use LLM for complex synthesis (selected by task complexity)
- [ ] Output: `reasoning.conclusions[]`, updated confidence scores
- [ ] Create unit tests for inference logic
- [ ] Document inference rules and confidence calculation

**10.4.6 Validation Agent**
- [ ] Create ValidationAgent in `src/golang/internal/domain/services/agents/validation.go`
- [ ] Check completeness of required slots per intent
- [ ] Validate logical consistency of reasoning chain
- [ ] Detect dependency cycles and missing artifacts
- [ ] Generate auto-fix hints for common issues
- [ ] Output: `diagnostics.validation_reports[]`, errors, warnings
- [ ] Create unit tests for validation rules
- [ ] Document validation criteria and auto-fix patterns

**10.4.7 Summarization Agent**
- [ ] Create SummarizationAgent in `src/golang/internal/domain/services/agents/summarization.go`
- [ ] Generate executive summary of reasoning process
- [ ] Create structured output artifacts (reports, command lists)
- [ ] Generate context diff overview
- [ ] Format output for downstream systems
- [ ] Output: `reasoning.summary`, final artifacts
- [ ] Create unit tests for summarization formats
- [ ] Document output formats and templates

**Implementation notes for all agents:**
- All agents implement `ReasoningAgent` interface
- Store in `src/golang/internal/domain/services/agents/`
- Each agent writes only to designated namespace
- Agent execution tracked in `audit.agent_runs`
- Tests in `tests/golang/internal/domain/services/agents/`

**Files to create:**
```
src/golang/internal/domain/services/agents/
├── intent_detection.go
├── reasoning_structure.go
├── retrieval_planner.go
├── context_synthesizer.go
├── inference.go
├── validation.go
└── summarization.go

tests/golang/internal/domain/services/agents/
├── intent_detection_test.go
├── reasoning_structure_test.go
├── retrieval_planner_test.go
├── context_synthesizer_test.go
├── inference_test.go
├── validation_test.go
└── summarization_test.go
```

### 10.5 Observability & Metrics

**Goal**: Comprehensive monitoring, tracing, and performance analysis

- [ ] Create metrics collector in `src/golang/internal/infrastructure/metrics/collector.go`
- [ ] Track per-agent metrics: duration, status, output size, LLM calls
- [ ] Track LLM metrics: tokens, cost, cache hits, model selection decisions
- [ ] Track context metrics: size, artifact count, diff size
- [ ] Implement distributed tracing with trace_id
- [ ] Add Prometheus metrics export
- [ ] Create structured logging with ELK-compatible format
- [ ] Implement performance profiling per agent
- [ ] Add cost reporting per session and per agent
- [ ] Create real-time monitoring dashboard (optional)
- [ ] Add alerting on budget overruns and SLA violations
- [ ] Create unit tests for metrics collection
- [ ] Document metrics format and export endpoints

**Implementation notes:**
- Metrics structure:
  ```go
  type Metrics struct {
      SessionID        string                    `json:"session_id"`
      TraceID          string                    `json:"trace_id"`
      AgentMetrics     map[string]*AgentMetrics  `json:"agent_metrics"`
      TotalDurationMS  int64                     `json:"total_duration_ms"`
      TotalCostUSD     float64                   `json:"total_cost_usd"`
  }

  type AgentMetrics struct {
      DurationMS int64   `json:"duration_ms"`
      LLMCalls   int     `json:"llm_calls"`
      Status     string  `json:"status"`
      Tokens     int     `json:"tokens,omitempty"`
      Cost       float64 `json:"cost,omitempty"`
  }
  ```
- Export to Prometheus, JSON logs, and in-context `diagnostics.performance`

**Files to create:**
```
src/golang/internal/infrastructure/metrics/
├── collector.go              # Metrics aggregation
├── prometheus_exporter.go   # Prometheus integration
└── performance_profiler.go  # Agent profiling

src/golang/internal/infrastructure/logging/
└── structured_logger.go     # ELK-compatible logging

tests/golang/internal/infrastructure/metrics/
└── collector_test.go
```

### 10.6 Testing, Validation & Documentation

**Goal**: Comprehensive testing, quality assurance, and complete documentation

- [ ] Create test fixtures for AgentContext in `tests/fixtures/`
- [ ] Add unit tests for all agents (80%+ coverage per agent)
- [ ] Add integration tests for full pipeline execution
- [ ] Create negative test cases (missing keys, invalid inputs, cycles)
- [ ] Add performance benchmarks (latency, LLM call frequency, cost per session)
- [ ] Implement reproducibility tests (same input → same output with fixed config)
- [ ] Create validation rule tests (slot completeness, logical consistency)
- [ ] Add cost budget enforcement tests
- [ ] Create documentation: AgentContext schema, agent contracts, pipeline config
- [ ] Document LLM selection policies and model profiles
- [ ] Add troubleshooting guide for common issues
- [ ] Create example pipeline configurations
- [ ] Document performance targets and SLA criteria
- [ ] Add architecture diagrams (agent flow, context structure, LLM selection)

**Implementation notes:**
- Coverage targets: 80%+ per module, 60%+ integration
- Performance targets:
  - Non-LLM pipeline: <2s for typical inputs
  - Single agent (no LLM): <500ms
  - Full pipeline with LLM: <10s for complex reasoning
- Reproducibility: deterministic with fixed model + temperature
- Budget controls: configurable limits per session/agent with hard stops

**Files to create:**
```
tests/golang/fixtures/
├── agent_context_fixtures.go
└── test_data.go

tests/golang/integration/
├── full_pipeline_test.go
├── parallel_execution_test.go
└── conditional_execution_test.go

tests/golang/performance/
├── benchmark_latency_test.go
├── benchmark_llm_calls_test.go
└── benchmark_cost_test.go

docs/reasoning_system/
├── agent_context_schema.md
├── agent_contracts.md
├── pipeline_configuration.md
├── llm_selection_policies.md
└── troubleshooting.md
```

**Performance Targets:**
- Average agent execution (no LLM): <500ms
- Full pipeline (no LLM): <2s for typical inputs
- LLM call minimization: <30% of simple tasks use LLM
- Cost budget: <$0.10 per complex reasoning session
- Horizontal scalability: stateless agents, queue-based execution
- Trace coverage: 100% of agent runs logged with trace_id

**Success Criteria:**
- [ ] All agents independent, communicate only via AgentContext
- [ ] Pipeline configurable with sequential/parallel/conditional modes
- [ ] Dynamic LLM selection works, decisions logged in context
- [ ] Performance targets met for non-LLM operations
- [ ] Budget controls enforced, cost tracking accurate
- [ ] Full reasoning traces reproducible
- [ ] Test coverage: 80%+ modules, 60%+ integration
- [ ] Validation catches incomplete slots, cycles, inconsistencies

**Risks & Mitigations:**
- **Risk**: LLM cost overrun → **Mitigation**: Strict budget limits, caching, degradation
- **Risk**: Context size explosion → **Mitigation**: Size limits, compression, external artifact storage
- **Risk**: Agent contract violations → **Mitigation**: Automated schema validation in CI/CD
- **Risk**: External provider failures → **Mitigation**: Fallback chains, offline mode for critical functions

---

## 🎯 Implementation Notes

**Each phase should be implemented incrementally**, with thorough testing before moving to the next phase. Every checkbox represents a discrete, implementable task that can be completed in a single focused session.

**Testing Strategy:**
- Unit tests for each component
- Integration tests with real APIs
- End-to-end workflow testing
- Performance and load testing

**Success Criteria:**
- Clean DDD architecture maintained
- All tests passing
- Documentation complete and accurate
- Performance targets met
