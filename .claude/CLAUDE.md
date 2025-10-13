# ADK LLM Proxy - Project Context for Claude Code

## Project Overview

This is an intelligent LLM proxy server that adds an agent layer on top of LLM APIs. It's built with:
- **Google Agent Development Kit (ADK)** for intelligent processing
- **Domain-Driven Design (DDD)** architecture
- **Model Context Protocol (MCP)** integration for GitLab and YouTrack
- **FastAPI** for the web layer
- **Streaming-first** architecture

## Technology Choice: Golang First

**Default to Golang for all implementations.**

### Directory Structure
```
src/golang/              # Golang-specific code (PRIMARY)
├── cmd/                 # Main applications
│   └── proxy/           # Proxy server implementation
├── internal/            # Private application code
│   ├── domain/         # Domain layer
│   ├── application/    # Application layer
│   ├── infrastructure/ # Infrastructure layer
│   └── presentation/   # Presentation layer
└── pkg/                # Public libraries

workflows/python/        # Python agents when needed
mcps/                    # MCP servers (Python-only due to SDK)
```

### When to Use Python (Exceptions Only)

Use Python **only** when Golang implementation is too complex or impractical:

1. **MCP Servers** - Python MCP SDK is the only official implementation (`mcps/*/`)
2. **Google ADK Integration** - ADK is Python-only (`workflows/python/adk_agent.py`)
3. **Complex AI/ML Workflows** - When you need transformers, langchain, or similar libraries
4. **Quick Prototyping Scripts** - Temporary proof-of-concept scripts (not committed)

**Rule:** If you can implement it reasonably in Golang, implement it in Golang.

### Language-Specific Guidelines

**Golang (Primary):**
- Use for all core services, APIs, CLI tools, and business logic
- Follow standard Go project layout (cmd/, internal/, pkg/)
- Use interfaces for abstraction (Dependency Inversion)
- Error handling: return errors, don't panic
- Concurrency: goroutines + channels

**Python (Exception Cases):**
- Only for MCP servers, ADK agents, or when absolutely necessary
- Follow DDD architecture even in Python components
- Use type hints and async/await
- Keep Python components isolated and callable from Golang (subprocess, gRPC)

## Architecture

### Directory Structure
```
src/
├── application/              # Application layer (use cases, orchestration)
│   └── services/            # Application services (orchestration, coordination)
│       ├── orchestration_service.py
│       ├── preprocessing_service.py
│       ├── postprocessing_service.py
│       └── mcp_tool_selector.py
├── domain/                   # Domain layer (core business logic)
│   ├── models/              # Domain models and entities
│   └── services/            # Domain services (business logic)
│       ├── reasoning_service_impl.py
│       ├── enhanced_reasoning_orchestrator.py
│       └── llm_reasoning_agents.py
├── infrastructure/           # Infrastructure layer (external dependencies)
│   ├── config/              # Configuration management
│   │   └── config.py
│   ├── llm/                 # LLM provider integrations
│   │   ├── openai_client.py
│   │   ├── ollama_client.py
│   │   └── deepseek_client.py
│   └── mcp/                 # MCP infrastructure
│       ├── client.py
│       ├── registry.py
│       ├── discovery.py
│       └── server_base.py
└── presentation/             # Presentation layer (API, UI)
    └── api/                 # FastAPI controllers
        └── streaming_controller.py

tests/                        # All tests mirror src/ structure
├── application/
│   └── services/
│       ├── test_orchestration_service.py
│       └── test_preprocessing_service.py
├── domain/
│   └── services/
│       └── test_reasoning_service.py
├── infrastructure/
│   ├── test_mcp_client.py
│   ├── test_mcp_registry.py
│   └── test_mcp_integration.py
└── integration/              # End-to-end integration tests
    ├── test_full_pipeline.py
    └── test_streaming.py

mcps/                         # MCP servers (separate microservices)
├── gitlab/                   # GitLab MCP server
│   ├── server.py
│   ├── gitlab_client.py
│   ├── requirements.txt
│   └── test_gitlab_server.py  # Server-specific tests
├── youtrack/                 # YouTrack MCP server
│   ├── server.py
│   ├── youtrack_client.py
│   ├── requirements.txt
│   └── test_youtrack_server.py
└── template/                 # Template for new MCP servers

workflows/                    # Reasoning workflow implementations
├── default/                  # Standard reasoning pipeline
├── enhanced/                 # LLM-powered multi-agent reasoning
└── empty/                    # No-op workflow (pass-through)
```

### Request Processing Pipeline
Every request flows through this intelligent pipeline:
```
Request → 🔍 Preprocessing → 🧠 Reasoning → 🤖 LLM → ✨ Postprocessing → Response
```

**Key Components:**
1. **Preprocessing**: Context injection, request validation
2. **Reasoning**: Intent analysis, MCP tool discovery/execution, context generation
3. **LLM Forwarding**: Proxy to OpenAI/Ollama/DeepSeek with enhanced context
4. **Postprocessing**: Response analysis, content enhancement

## Coding Standards & Conventions

### Golang Standards (Primary)
- **Naming**: camelCase for private, PascalCase for public
- **Error handling**: Always check errors; return errors with context (`fmt.Errorf("...: %w", err)`)
- **Interfaces**: Small, focused interfaces (Interface Segregation Principle)
- **Comments**: Use godoc-style comments for public APIs
- **Formatting**: Use `gofmt` or `goimports` (enforced automatically)
- **Project structure**: Follow standard Go layout (cmd/, internal/, pkg/)
- **Concurrency**: Use goroutines + channels; avoid shared memory
- **Testing**: Table-driven tests, use `testify` for assertions

**Error Handling Example:**
```go
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("failed to do something: %w", err)
}
```

### Python Standards (Exception Cases Only)
- **Type hints**: Always use for function parameters and return values
- **Async/await**: All I/O operations must be async
- **Error handling**: Specific error types, propagate to caller
- **Docstrings**: Google-style for all public functions
- **Formatting**: Use `black` for code formatting

## SOLID Principles in Practice

The SOLID principles are foundational design principles that make software designs more understandable, flexible, and maintainable. They work hand-in-hand with DDD architecture.

### Single Responsibility Principle (SRP)
**Definition**: A class/module should have only one reason to change. Each component should have a single, well-defined responsibility.

**Golang Example (Good):**
```go
// Good: Separate responsibilities
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) FindByID(id string) (*User, error) {
    // Only responsible for data access
}

type UserValidator struct {}

func (v *UserValidator) Validate(user *User) error {
    // Only responsible for validation
}

type UserService struct {
    repo UserRepository
    validator UserValidator
}

func (s *UserService) CreateUser(user *User) error {
    // Orchestrates validation and persistence
    if err := s.validator.Validate(user); err != nil {
        return err
    }
    return s.repo.Save(user)
}
```

**Anti-pattern (Bad):**
```go
// Bad: Too many responsibilities
type UserService struct {
    db *sql.DB
}

func (s *UserService) CreateUser(user *User) error {
    // Validation logic
    if user.Email == "" {
        return errors.New("invalid email")
    }
    
    // Database logic
    _, err := s.db.Exec("INSERT INTO users...")
    
    // Email logic
    sendWelcomeEmail(user.Email)
    
    // Logging logic
    log.Printf("User created: %s", user.ID)
    
    return err
}
```

**Python Example (Good):**
```python
# Good: Separate responsibilities
class MCPClient:
    """Only responsible for MCP protocol communication"""
    async def send_request(self, request: MCPRequest) -> MCPResponse:
        pass

class MCPRegistry:
    """Only responsible for managing MCP server connections"""
    def register_server(self, server: MCPServer) -> None:
        pass
    
    def get_server(self, name: str) -> MCPServer:
        pass

class MCPToolSelector:
    """Only responsible for selecting appropriate tools"""
    def select_tools(self, intent: str) -> List[MCPTool]:
        pass
```

**DDD Layer Mapping:**
- Each service in `src/domain/services/` should have a single business responsibility
- Each service in `src/application/services/` should orchestrate a single use case
- Each client in `src/infrastructure/` should integrate with a single external system

### Open/Closed Principle (OCP)
**Definition**: Software entities should be open for extension but closed for modification. Use interfaces and composition to add new behavior without changing existing code.

**Golang Example (Good):**
```go
// Good: Open for extension via interface
type LLMProvider interface {
    Name() string
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

// Easy to add new providers without modifying existing code
type OpenAIProvider struct { /* ... */ }
func (p *OpenAIProvider) StreamCompletion(...) { /* ... */ }

type AnthropicProvider struct { /* ... */ }
func (p *AnthropicProvider) StreamCompletion(...) { /* ... */ }

type OllamaProvider struct { /* ... */ }
func (p *OllamaProvider) StreamCompletion(...) { /* ... */ }

// Orchestrator works with interface, never modified
type Orchestrator struct {
    providers map[string]LLMProvider
}

func (o *Orchestrator) GetProvider(name string) LLMProvider {
    return o.providers[name]
}
```

**Anti-pattern (Bad):**
```go
// Bad: Must modify existing code to add new provider
type Orchestrator struct {
    openai *OpenAIClient
    anthropic *AnthropicClient
}

func (o *Orchestrator) StreamCompletion(provider string, req *Request) {
    switch provider {
    case "openai":
        return o.openai.Stream(req)
    case "anthropic":
        return o.anthropic.Stream(req)
    // Must modify this function for each new provider!
    default:
        return nil
    }
}
```

**Python Example (Good):**
```python
# Good: Open for extension via ABC
from abc import ABC, abstractmethod

class WorkflowInterface(ABC):
    @abstractmethod
    async def execute(self, context: ReasoningContext) -> ReasoningResult:
        pass

class DefaultWorkflow(WorkflowInterface):
    async def execute(self, context: ReasoningContext) -> ReasoningResult:
        return ReasoningResult(message="Default reasoning")

class EnhancedWorkflow(WorkflowInterface):
    async def execute(self, context: ReasoningContext) -> ReasoningResult:
        # LLM-powered reasoning
        pass

# Orchestrator uses interface, never needs modification
class ReasoningService:
    def __init__(self, workflow: WorkflowInterface):
        self.workflow = workflow
    
    async def reason(self, context: ReasoningContext) -> ReasoningResult:
        return await self.workflow.execute(context)
```

**DDD Layer Mapping:**
- Define interfaces in `src/domain/services/` (e.g., `IReasoningService`, `ILLMProvider`)
- Implement in `src/infrastructure/` (e.g., `OpenAIClient`, `OllamaClient`)
- Orchestrate in `src/application/services/` using interfaces only

### Liskov Substitution Principle (LSP)
**Definition**: Subtypes must be substitutable for their base types. Any implementation of an interface should work correctly when used in place of the interface.

**Golang Example (Good):**
```go
// Good: All implementations honor the contract
type Workflow interface {
    // Returns result or error, never panics
    Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error)
}

type DefaultWorkflow struct{}
func (w *DefaultWorkflow) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error) {
    // Always returns result, respects context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        return &ReasoningResult{Message: "Hello World"}, nil
    }
}

type AdvancedWorkflow struct{}
func (w *AdvancedWorkflow) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error) {
    // Same contract: returns result or error, respects context
    result, err := w.callAgents(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("agent call failed: %w", err)
    }
    return result, nil
}

// Client code works with any implementation
func ProcessRequest(workflow Workflow, input *ReasoningInput) {
    result, err := workflow.Execute(context.Background(), input)
    // Works identically regardless of workflow implementation
}
```

**Anti-pattern (Bad):**
```go
// Bad: Violates contract expectations
type BrokenWorkflow struct{}
func (w *BrokenWorkflow) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error) {
    // Violates LSP: panics instead of returning error
    if input == nil {
        panic("input cannot be nil")
    }
    
    // Violates LSP: ignores context cancellation
    time.Sleep(10 * time.Minute)
    
    return &ReasoningResult{}, nil
}
```

**Python Example (Good):**
```python
# Good: All implementations honor the contract
class MCPServerBase(ABC):
    @abstractmethod
    async def execute_tool(self, tool_name: str, args: dict) -> dict:
        """Returns dict result or raises MCPException"""
        pass

class GitLabMCPServer(MCPServerBase):
    async def execute_tool(self, tool_name: str, args: dict) -> dict:
        # Honors contract: returns dict or raises MCPException
        if tool_name not in self.tools:
            raise MCPException(f"Unknown tool: {tool_name}")
        return await self._call_tool(tool_name, args)

class YouTrackMCPServer(MCPServerBase):
    async def execute_tool(self, tool_name: str, args: dict) -> dict:
        # Same contract: returns dict or raises MCPException
        try:
            return await self.client.call(tool_name, args)
        except Exception as e:
            raise MCPException(f"Tool execution failed: {e}")
```

**DDD Layer Mapping:**
- Define clear contracts in domain interfaces
- All infrastructure implementations must honor these contracts
- Never throw unexpected exceptions or ignore parameters

### Interface Segregation Principle (ISP)
**Definition**: Clients should not be forced to depend on interfaces they don't use. Keep interfaces small and focused.

**Golang Example (Good):**
```go
// Good: Small, focused interfaces
type ToolExecutor interface {
    ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error)
}

type ToolDiscoverer interface {
    ListTools(ctx context.Context) ([]Tool, error)
}

type HealthChecker interface {
    CheckHealth(ctx context.Context) error
}

// Clients depend only on what they need
type Orchestrator struct {
    executor ToolExecutor  // Only needs execution
}

type Registry struct {
    discoverer ToolDiscoverer  // Only needs discovery
    health HealthChecker       // Only needs health checks
}

// Implementation can implement multiple interfaces
type MCPServer struct { /* ... */ }
func (s *MCPServer) ExecuteTool(...) (any, error) { /* ... */ }
func (s *MCPServer) ListTools(...) ([]Tool, error) { /* ... */ }
func (s *MCPServer) CheckHealth(...) error { /* ... */ }
```

**Anti-pattern (Bad):**
```go
// Bad: Fat interface forces all clients to depend on everything
type MCPServer interface {
    ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error)
    ListTools(ctx context.Context) ([]Tool, error)
    CheckHealth(ctx context.Context) error
    GetConfig() Config
    UpdateConfig(Config) error
    GetMetrics() Metrics
    // ... 15 more methods
}

// Client only needs ExecuteTool but must mock everything
type Orchestrator struct {
    server MCPServer  // Depends on entire fat interface
}
```

**Python Example (Good):**
```python
# Good: Small, focused interfaces
class ToolExecutorInterface(Protocol):
    async def execute_tool(self, tool_name: str, args: dict) -> dict:
        pass

class ToolDiscoveryInterface(Protocol):
    async def list_tools(self) -> List[ToolInfo]:
        pass

# Clients depend only on what they need
class OrchestrationService:
    def __init__(self, executor: ToolExecutorInterface):
        self.executor = executor  # Only needs execution

class MCPRegistry:
    def __init__(self, discoverer: ToolDiscoveryInterface):
        self.discoverer = discoverer  # Only needs discovery
```

**DDD Layer Mapping:**
- Create small, focused interfaces in `src/domain/services/`
- Avoid "God interfaces" with many unrelated methods
- Use interface composition when needed

### Dependency Inversion Principle (DIP)
**Definition**: High-level modules should not depend on low-level modules. Both should depend on abstractions. This is the foundation of DDD architecture.

**Golang Example (Good):**
```go
// Good: Domain layer defines interface (abstraction)
// File: internal/domain/services/provider.go
package services

type LLMProvider interface {
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

// File: internal/application/services/orchestrator.go
package services

// Application layer depends on domain interface
type Orchestrator struct {
    provider services.LLMProvider  // Depends on abstraction
}

func NewOrchestrator(provider services.LLMProvider) *Orchestrator {
    return &Orchestrator{provider: provider}
}

// File: internal/infrastructure/providers/openai.go
package providers

// Infrastructure layer implements domain interface
type OpenAIProvider struct { /* ... */ }

func (p *OpenAIProvider) StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error) {
    // Implementation details
}

// File: cmd/proxy/main.go
// Dependency injection at composition root
func main() {
    // Wire dependencies
    provider := providers.NewOpenAIProvider(config)
    orchestrator := services.NewOrchestrator(provider)
    handler := api.NewHandler(orchestrator)
    // ...
}
```

**Anti-pattern (Bad):**
```go
// Bad: High-level depends on low-level concrete types
package services

import "internal/infrastructure/providers"  // ❌ Application depends on infrastructure

type Orchestrator struct {
    provider *providers.OpenAIProvider  // ❌ Depends on concrete type
}

func (o *Orchestrator) Process(req *Request) {
    // Tightly coupled to OpenAI implementation
    o.provider.CallOpenAI(req)
}
```

**Python Example (Good):**
```python
# Good: Domain layer defines interface
# File: src/domain/services/reasoning_interface.py
from abc import ABC, abstractmethod

class IReasoningService(ABC):
    @abstractmethod
    async def reason(self, context: ReasoningContext) -> ReasoningResult:
        pass

# File: src/application/services/orchestration_service.py
# Application layer depends on domain interface
class OrchestrationService:
    def __init__(self, reasoning: IReasoningService):
        self.reasoning = reasoning  # Depends on abstraction
    
    async def process(self, request: Request) -> Response:
        result = await self.reasoning.reason(context)
        return Response(result)

# File: src/domain/services/reasoning_service_impl.py
# Domain implementation
class ReasoningServiceImpl(IReasoningService):
    async def reason(self, context: ReasoningContext) -> ReasoningResult:
        # Business logic
        pass

# File: main.py
# Dependency injection at composition root
def setup():
    reasoning = ReasoningServiceImpl()
    orchestrator = OrchestrationService(reasoning)
    return orchestrator
```

**DDD Layer Mapping - Dependency Flow:**
```
Presentation → Application → Domain ← Infrastructure
                              ↑
                          (interfaces defined here)
```

**Key Rules:**
1. **Domain layer** defines interfaces (abstractions)
2. **Infrastructure layer** implements interfaces (concrete)
3. **Application layer** depends on domain interfaces only
4. **Presentation layer** depends on application layer only
5. Dependencies always point **inward** toward domain

**Testability Benefits:**
```go
// Easy to test with mocks
type MockProvider struct{}
func (m *MockProvider) StreamCompletion(...) (<-chan CompletionChunk, error) {
    // Test implementation
}

func TestOrchestrator(t *testing.T) {
    mockProvider := &MockProvider{}
    orchestrator := NewOrchestrator(mockProvider)
    // Test with mock instead of real provider
}
```

### SOLID Principles Summary

| Principle | Focus | Key Question |
|-----------|-------|--------------|
| **SRP** | Single Responsibility | Does this class have only one reason to change? |
| **OCP** | Open/Closed | Can I add new behavior without modifying existing code? |
| **LSP** | Liskov Substitution | Can I substitute any implementation without breaking the system? |
| **ISP** | Interface Segregation | Are my interfaces small and focused? |
| **DIP** | Dependency Inversion | Do I depend on abstractions or concrete implementations? |

**SOLID ↔ DDD Mapping:**
- **SRP** → Each service has one clear responsibility per layer
- **OCP** → Interfaces in domain, implementations in infrastructure
- **LSP** → All implementations honor domain contracts
- **ISP** → Small, focused interfaces in domain layer
- **DIP** → Dependencies point inward toward domain

### DDD Principles & File Organization

**CRITICAL: Follow strict layer separation and file placement rules**

#### Layer Responsibilities

1. **Domain Layer** (`src/domain/`)
   - Core business logic and rules
   - Domain models and entities
   - Domain services (pure business logic, no external dependencies)
   - Independent of frameworks, databases, or external systems
   - Files: `src/domain/models/*.py`, `src/domain/services/*.py`

2. **Application Layer** (`src/application/`)
   - Use cases and application workflows
   - Orchestrates domain objects and services
   - Coordinates infrastructure services
   - Transaction management and cross-cutting concerns
   - Files: `src/application/services/*.py`

3. **Infrastructure Layer** (`src/infrastructure/`)
   - External system integrations (APIs, databases, file systems)
   - Framework implementations (FastAPI, httpx, MCP)
   - Configuration and environment management
   - Technical utilities and helpers
   - Files: `src/infrastructure/{config,llm,mcp,adk}/*.py`

4. **Presentation Layer** (`src/presentation/`)
   - API controllers and endpoints
   - Request/response DTOs
   - Input validation and serialization
   - HTTP/WebSocket handling
   - Files: `src/presentation/api/*.py`

#### File Placement Rules

**DO:**
- Place business logic in `src/domain/services/`
- Place orchestration in `src/application/services/`
- Place API integrations in `src/infrastructure/`
- Place HTTP endpoints in `src/presentation/api/`
- Create subdirectories for related functionality (e.g., `src/infrastructure/mcp/`)

**DON'T:**
- Mix layers (e.g., HTTP code in domain layer)
- Put business logic in infrastructure
- Access infrastructure directly from domain
- Create files outside the layer structure

#### Creating New Files - Decision Tree

**When creating a new file, ask yourself:**

1. **Is it a test?**
   - YES → `tests/` directory (mirror src/ structure)
   - NO → Continue to step 2

2. **What does it do?**
   - **Business logic/rules** → `src/domain/services/`
   - **Orchestration/workflows** → `src/application/services/`
   - **External API/database** → `src/infrastructure/`
   - **HTTP endpoint** → `src/presentation/api/`
   - **MCP server** → `mcps/{server-name}/`
   - **Reasoning workflow** → `workflows/{workflow-name}/`

3. **Does it need a subdirectory?**
   - Group related files together
   - Example: `src/infrastructure/llm/` for all LLM provider clients
   - Example: `src/domain/models/` for all domain models

**Quick Examples:**
- New LLM provider (e.g., Anthropic) → `src/infrastructure/llm/anthropic_client.py`
- New reasoning algorithm → `src/domain/services/reasoning_algorithm.py`
- New preprocessing step → `src/application/services/preprocessing_service.py` (or new file if complex)
- New API endpoint → `src/presentation/api/new_controller.py`
- Test for reasoning → `tests/domain/services/test_reasoning_algorithm.py`

### Return Value Pattern
Most service functions return a status dictionary:
```python
{"status": "success", "data": {...}}
{"status": "error", "error": "Error message"}
```

### MCP Integration
- Use `mcp_registry` for managing MCP server connections
- MCP servers are configured in `config.yaml` under `mcp.servers`
- Each MCP server provides **tools**, **resources**, and **prompts**

## Important Files

### Configuration
- **config.yaml**: Main configuration (LLM providers, MCP servers, processing settings)
- **src/infrastructure/config/config.py**: Config loading and validation

### Entry Points
- **main.py**: Main server entry point (supports --provider, --model, --prompt)
- **src/presentation/api/streaming_controller.py**: FastAPI app with streaming endpoints

### Core Services
- **src/application/services/orchestration_service.py**: Main pipeline orchestrator
- **src/domain/services/reasoning_service_impl.py**: Reasoning implementation
- **src/application/services/postprocessing_service.py**: Response postprocessing
- **src/infrastructure/mcp/**: MCP client, registry, server management

### MCP Servers
- **mcps/gitlab/server.py**: GitLab integration (commits, MRs, code analysis)
- **mcps/youtrack/server.py**: YouTrack integration (epics, tasks, analytics)

## Development Workflow

### Running the Server
```bash
# Start with OpenAI
python main.py -provider openai -model gpt-4o-mini

# Start with Ollama (local)
python main.py -provider ollama -model mistral

# Prompt mode (single request, no server)
python main.py --prompt "Your question" -provider openai -model gpt-4o-mini
```

### Testing

**Test Organization:**
- **Golang tests**: `tests/golang/` mirroring `src/golang/` structure
- **Python tests**: Only for MCP servers (inside `mcps/*/`) or ADK agents
- **Test naming**: `*_test.go` (Golang) or `test_*.py` (Python)
- **Never** create test files in project root

**Golang Testing:**
```
tests/golang/
├── internal/
│   ├── domain/
│   ├── application/
│   ├── infrastructure/
│   └── presentation/
└── integration/
    └── e2e_test.go
```

**Testing Guidelines:**
- Use table-driven tests in Golang
- Mock external dependencies (LLM APIs, MCP servers)
- Test error cases and edge cases
- Integration tests for full request flow

### Configuration Changes
After changing `config.yaml`:
1. Restart the server (no hot-reload for config)
2. Verify MCP servers connect: check startup logs
3. Test with: `curl -X POST http://localhost:8000/v1/chat/completions ...`

### Adding New MCP Servers
1. Create new directory under `mcps/your-server/`
2. Implement `server.py` extending `MCPServerBase`
3. Add configuration to `config.yaml` under `mcp.servers`
4. Register in `mcp_registry` on startup

### Creating Custom Workflows
1. Create directory: `workflows/custom/`
2. Add `reasoning_callback.py` with `reasoning_workflow()` function
3. Update `config.yaml`: `processing.reasoning_workflow: "workflows/custom"`
4. Restart server

## Common Pitfalls & Solutions

### Async Issues
- **Problem**: "object async_generator can't be used in 'await' expression"
- **Solution**: Use `async for chunk in generator:` not `await generator`

### MCP Connection Failures
- **Problem**: MCP server won't connect
- **Solution**: Check `command` and `args` in config.yaml; verify Python module path

### Import Errors
- **Problem**: Module not found errors
- **Solution**: Always run from project root; check PYTHONPATH

### Streaming Not Working
- **Problem**: Response not streaming
- **Solution**: Check `stream: true` in request; verify SSE format in response

### Warning/Error Noise
- **Problem**: Too many asyncio warnings during shutdown
- **Solution**: Warnings are suppressed in main.py; cleanup happens automatically

## Environment Variables

Required for operation:
- **OPENAI_API_KEY**: For OpenAI provider (stored in config.yaml or env)
- **GITLAB_URL**: GitLab instance URL (in config.yaml)
- **GITLAB_TOKEN**: GitLab API token (in config.yaml)
- **YOUTRACK_BASE_URL**: YouTrack instance URL (in config.yaml)
- **YOUTRACK_TOKEN**: YouTrack API token (in config.yaml)

Optional:
- **LLM_PROVIDER**: Override provider from command line
- **LLM_MODEL**: Override model from command line
- **DEBUG**: Enable debug mode ("true"/"1"/"yes"/"on")

## Code Review Checklist

When reviewing or writing code:

### Architecture & Design
- [ ] **Golang-first approach** (Python only for MCP servers, ADK, or when truly necessary)
- [ ] **Proper layer separation** (domain/application/infrastructure/presentation)
- [ ] **Files in correct directories** (follow DDD structure, Go project layout)
- [ ] **Test files properly located** (`tests/golang/` mirroring `src/golang/`)
- [ ] **Interfaces defined before implementations** (design abstraction first)
- [ ] **Dependency Rule respected** (dependencies point inward toward domain)
- [ ] **Abstractions at proper layer** (interfaces in domain, implementations in infrastructure)

### SOLID Principles
- [ ] **Single Responsibility (SRP)** - Each class/module has one reason to change
- [ ] **Open/Closed (OCP)** - Can add new behavior without modifying existing code (via interfaces)
- [ ] **Liskov Substitution (LSP)** - All implementations honor their interface contracts
- [ ] **Interface Segregation (ISP)** - Interfaces are small and focused, not fat interfaces
- [ ] **Dependency Inversion (DIP)** - Depend on abstractions, not concrete implementations

### Code Quality
- [ ] **Functions are small and focused** (<30 lines, single responsibility per function)
- [ ] **Function names are self-explanatory** (clear intent, no need for excessive comments)
- [ ] **No business logic in infrastructure** (domain services only)
- [ ] **No external dependencies in domain** (interfaces only, no imports from infrastructure)
- [ ] **Error handling** (return errors with context, don't panic in Go)
- [ ] **Comments on public APIs** (godoc-style in Go, docstrings in Python)

### Testing & Safety
- [ ] **Testable design** (dependency injection, mockable interfaces)
- [ ] **Tests included** (unit + integration where appropriate)
- [ ] **Concurrency safe** (proper goroutine management, channel usage in Go)
- [ ] **Config changes documented**

## Git Workflow

- **Main branch**: `main`
- **Commit messages**: Use conventional commits (feat:, fix:, docs:, etc.)
- **Reference tasks**: Include YouTrack task IDs in commits when applicable (e.g., "PROJ-123")
- **Test files**: NEVER commit test files in root directory; they must be in `tests/`

## Performance Considerations

- Use **async** everywhere for I/O operations
- MCP connections are **persistent** and reused
- Streaming reduces latency (no buffering)
- Configure timeouts in config.yaml (default: 30s)
- Enable caching in MCP servers when possible

## Security Notes

- API keys stored in config.yaml (DO NOT commit!)
- Use environment variables for sensitive data
- MCP servers run in isolated processes (stdio transport)
- Token validation happens per-provider
- SSL verification enabled by default

## When to Use What

### When to modify preprocessing
Add new preprocessing logic in `src/application/services/preprocessing_service.py` when:
- Adding context injection
- Validating requests
- Transforming input format

### When to modify reasoning
Modify reasoning in `src/domain/services/reasoning_service_impl.py` when:
- Changing intent analysis
- Adding new reasoning steps
- Modifying MCP tool discovery/execution

### When to create custom workflow
Create a custom workflow (`workflows/custom/`) when:
- Need completely different reasoning flow
- Want to skip certain steps
- Building domain-specific reasoning logic

### When to modify postprocessing
Update `src/application/services/postprocessing_service.py` when:
- Adding response analytics
- Enhancing output format
- Adding metadata to responses

### When to add MCP server
Create new MCP server (`mcps/new-server/`) when:
- Integrating new external system
- Adding new data sources
- Building custom tools/resources



## Helpful Commands

### Golang Commands (Primary)
```bash
# Build
go build -o bin/proxy ./src/golang/cmd/proxy

# Run
./bin/proxy --config config.yaml --port 8001

# Test
go test ./src/golang/...
go test -cover ./src/golang/...

# Format & Lint
gofmt -w src/golang/
golangci-lint run ./src/golang/...

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o bin/proxy-linux ./src/golang/cmd/proxy
GOOS=darwin GOARCH=amd64 go build -o bin/proxy-darwin ./src/golang/cmd/proxy
```

### Python Commands (MCP/ADK Only)
```bash
# Test MCP servers
pytest mcps/gitlab/test_gitlab_server.py
pytest mcps/youtrack/test_youtrack_server.py

# Format
black mcps/
```

## Getting Help

- **Architecture questions**: Check `articles/*.md` for detailed explanations
- **MCP integration**: See `mcps/*/README.md` for server-specific docs
- **Workflows**: See `workflows/README.md` for reasoning customization
- **Main README**: `/README.md` for quick start and overview
