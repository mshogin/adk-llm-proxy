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

## Interface-Driven Design

Interface-driven design is a development approach where you **define interfaces first**, then implement them. This approach ensures loose coupling, testability, and adherence to SOLID principles (especially DIP and OCP).

### Core Principle: Design Abstraction Before Implementation

**The Golden Rule**: When adding a new feature or component, always define the interface **before** writing any implementation code.

**Why This Matters:**
1. **Forces you to think about contracts** - What operations are needed? What are the inputs/outputs?
2. **Enables parallel development** - Multiple people can implement different parts simultaneously
3. **Makes testing trivial** - Mock implementations are easy to create
4. **Prevents tight coupling** - Consumers depend on abstractions, not concrete types
5. **Supports multiple implementations** - Easy to swap or add providers/strategies

### When to Define Interfaces

**During the Planning Phase** (before writing any implementation):

1. **Identify external dependencies** - Databases, APIs, file systems, external services
2. **Define domain abstractions** - Core business operations that need multiple implementations
3. **Plan layer boundaries** - How will layers communicate? What contracts do they need?
4. **Consider testability** - What needs to be mocked during testing?

**Example Planning Session:**
```
Feature: Add caching to LLM responses

Step 1: Define interface (abstraction first)
  - What operations? Get, Set, Delete, Clear
  - What data types? Key (string), Value (LLMResponse), TTL (duration)
  - Where does it live? Domain layer (abstraction)

Step 2: Plan implementations
  - RedisCache (infrastructure layer)
  - MemoryCache (infrastructure layer)
  - NoOpCache (for testing)

Step 3: Wire it up
  - Orchestrator depends on CacheInterface (domain)
  - Main.go injects concrete implementation (Redis or Memory)
```

### Python Interface Patterns

Python provides multiple ways to define interfaces. Choose the right tool for the job.

#### 1. Abstract Base Classes (ABC)
**Use when**: You have a base class with shared implementation and require specific methods to be overridden.

```python
from abc import ABC, abstractmethod
from typing import List

class LLMProvider(ABC):
    """Base interface for all LLM providers"""
    
    @abstractmethod
    async def stream_completion(self, request: CompletionRequest) -> AsyncIterator[CompletionChunk]:
        """Stream completion chunks from the LLM"""
        pass
    
    @abstractmethod
    def name(self) -> str:
        """Return provider name"""
        pass

# Concrete implementation
class OpenAIProvider(LLMProvider):
    async def stream_completion(self, request: CompletionRequest) -> AsyncIterator[CompletionChunk]:
        # Implementation
        async for chunk in self._call_openai_api(request):
            yield chunk
    
    def name(self) -> str:
        return "openai"

# Enforces contract at instantiation
provider = OpenAIProvider()  # ✓ OK
```

**Key Features:**
- Cannot instantiate abstract classes
- Raises `TypeError` if abstract methods are not implemented
- Supports inheritance hierarchy
- Can provide shared implementation in base class

#### 2. Protocol (Structural Subtyping - PEP 544)
**Use when**: You want duck typing with type checker support, or defining interfaces for external code you don't control.

```python
from typing import Protocol, AsyncIterator

class CacheInterface(Protocol):
    """Structural interface for caching"""
    
    async def get(self, key: str) -> dict | None:
        ...
    
    async def set(self, key: str, value: dict, ttl: int) -> None:
        ...
    
    async def delete(self, key: str) -> None:
        ...

# No explicit inheritance needed
class RedisCache:
    async def get(self, key: str) -> dict | None:
        # Implementation
        pass
    
    async def set(self, key: str, value: dict, ttl: int) -> None:
        # Implementation
        pass
    
    async def delete(self, key: str) -> None:
        # Implementation
        pass

# Type checker verifies structure matches
def use_cache(cache: CacheInterface) -> None:
    await cache.get("key")  # ✓ Type safe

cache = RedisCache()
use_cache(cache)  # ✓ OK - RedisCache matches CacheInterface structure
```

**Key Features:**
- No explicit inheritance required (duck typing)
- Type checkers (mypy, pyright) verify compatibility
- Great for defining interfaces for third-party code
- More flexible than ABC

#### 3. Type Hints Only (Lightweight)
**Use when**: You need simple type checking without runtime enforcement.

```python
from typing import Callable, Awaitable

# Function type alias
ToolExecutor = Callable[[str, dict], Awaitable[dict]]

# Usage
async def execute_mcp_tool(tool_name: str, args: dict) -> dict:
    # Implementation
    pass

# Function matches type
executor: ToolExecutor = execute_mcp_tool
```

**Comparison:**

| Pattern | Runtime Check | Inheritance Required | Type Check | Use Case |
|---------|--------------|---------------------|------------|----------|
| **ABC** | ✓ Yes | ✓ Yes | ✓ Yes | Domain interfaces with shared logic |
| **Protocol** | ✗ No | ✗ No | ✓ Yes | Duck-typed interfaces, external code |
| **Type Hints** | ✗ No | ✗ No | ✓ Yes | Simple function types, callbacks |

### Golang Interface Patterns

Go's interface system is based on **implicit satisfaction** - if a type implements all methods, it automatically satisfies the interface.

#### 1. Small, Focused Interfaces (Idiomatic Go)

```go
// Good: Small, focused interface
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// Compose interfaces when needed
type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

**Go Proverb**: *"The bigger the interface, the weaker the abstraction."*

#### 2. Interface Composition

```go
// Domain layer defines focused interfaces
// File: internal/domain/services/provider.go
package services

type Streamer interface {
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

type Namer interface {
    Name() string
}

type HealthChecker interface {
    CheckHealth(ctx context.Context) error
}

// Compose when needed
type LLMProvider interface {
    Streamer
    Namer
    HealthChecker
}

// Clients can depend on minimal interface
type Orchestrator struct {
    streamer Streamer  // Only needs streaming capability
}

// Implementation satisfies all interfaces implicitly
type OpenAIProvider struct {
    apiKey string
}

func (p *OpenAIProvider) StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error) {
    // Implementation
}

func (p *OpenAIProvider) Name() string {
    return "openai"
}

func (p *OpenAIProvider) CheckHealth(ctx context.Context) error {
    // Implementation
}

// Automatically satisfies LLMProvider, Streamer, Namer, HealthChecker
```

#### 3. Interface Segregation Pattern

```go
// Anti-pattern: Fat interface
type MCPServer interface {
    ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error)
    ListTools(ctx context.Context) ([]Tool, error)
    GetResources(ctx context.Context) ([]Resource, error)
    GetPrompts(ctx context.Context) ([]Prompt, error)
    CheckHealth(ctx context.Context) error
    GetConfig() Config
    UpdateConfig(Config) error
    // ... 10 more methods
}

// Better: Segregated interfaces
type ToolExecutor interface {
    ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error)
}

type ToolLister interface {
    ListTools(ctx context.Context) ([]Tool, error)
}

type ResourceProvider interface {
    GetResources(ctx context.Context) ([]Resource, error)
}

// Clients depend only on what they need
type Orchestrator struct {
    executor ToolExecutor  // Minimal dependency
}

type Registry struct {
    lister ToolLister  // Different minimal dependency
}
```

### Dependency Injection Patterns

Dependency injection enables testability and loose coupling. Always inject dependencies through constructors.

#### Python Dependency Injection

```python
# Good: Constructor injection
class OrchestrationService:
    def __init__(
        self,
        reasoning: IReasoningService,
        llm_provider: LLMProvider,
        cache: CacheInterface
    ):
        self.reasoning = reasoning
        self.llm_provider = llm_provider
        self.cache = cache
    
    async def process(self, request: Request) -> Response:
        # Use injected dependencies
        result = await self.reasoning.reason(context)
        return Response(result)

# Composition root (main.py)
def setup():
    # Wire dependencies
    reasoning = ReasoningServiceImpl()
    llm_provider = OpenAIProvider(api_key="...")
    cache = RedisCache(url="redis://localhost")
    
    # Inject dependencies
    orchestrator = OrchestrationService(reasoning, llm_provider, cache)
    return orchestrator

# Testing with mocks
def test_orchestration():
    mock_reasoning = MockReasoningService()
    mock_llm = MockLLMProvider()
    mock_cache = MockCache()
    
    orchestrator = OrchestrationService(mock_reasoning, mock_llm, mock_cache)
    # Test with mocks instead of real dependencies
```

#### Golang Dependency Injection

```go
// Good: Constructor injection
type Orchestrator struct {
    reasoning services.ReasoningService
    provider  services.LLMProvider
    cache     services.Cache
}

func NewOrchestrator(
    reasoning services.ReasoningService,
    provider services.LLMProvider,
    cache services.Cache,
) *Orchestrator {
    return &Orchestrator{
        reasoning: reasoning,
        provider:  provider,
        cache:     cache,
    }
}

func (o *Orchestrator) Process(ctx context.Context, req *Request) (*Response, error) {
    // Use injected dependencies
    result, err := o.reasoning.Reason(ctx, req)
    return &Response{Data: result}, err
}

// Composition root (main.go)
func main() {
    // Wire dependencies
    reasoning := domain.NewReasoningService()
    provider := providers.NewOpenAIProvider(config.APIKey)
    cache := infrastructure.NewRedisCache(config.RedisURL)
    
    // Inject dependencies
    orchestrator := services.NewOrchestrator(reasoning, provider, cache)
    handler := api.NewHandler(orchestrator)
    
    // Start server
    http.ListenAndServe(":8080", handler)
}

// Testing with mocks
func TestOrchestrator(t *testing.T) {
    mockReasoning := &MockReasoningService{}
    mockProvider := &MockLLMProvider{}
    mockCache := &MockCache{}
    
    orchestrator := NewOrchestrator(mockReasoning, mockProvider, mockCache)
    // Test with mocks
}
```

### Mock-Friendly Design

Good interface design makes testing easy. Follow these patterns:

#### Python Mock Example

```python
# Interface-based design
class IReasoningService(ABC):
    @abstractmethod
    async def reason(self, context: ReasoningContext) -> ReasoningResult:
        pass

# Mock implementation for testing
class MockReasoningService(IReasoningService):
    def __init__(self, return_value: ReasoningResult):
        self.return_value = return_value
        self.calls = []
    
    async def reason(self, context: ReasoningContext) -> ReasoningResult:
        self.calls.append(context)
        return self.return_value

# Test using mock
async def test_orchestration():
    expected_result = ReasoningResult(message="test")
    mock = MockReasoningService(expected_result)
    
    orchestrator = OrchestrationService(reasoning=mock)
    result = await orchestrator.process(request)
    
    assert result.message == "test"
    assert len(mock.calls) == 1  # Verify reasoning was called
```

#### Golang Mock Example

```go
// Interface enables mocking
type ReasoningService interface {
    Reason(ctx context.Context, req *ReasoningContext) (*ReasoningResult, error)
}

// Mock implementation
type MockReasoningService struct {
    ReturnValue *ReasoningResult
    ReturnError error
    Calls       []*ReasoningContext
}

func (m *MockReasoningService) Reason(ctx context.Context, req *ReasoningContext) (*ReasoningResult, error) {
    m.Calls = append(m.Calls, req)
    return m.ReturnValue, m.ReturnError
}

// Test using mock
func TestOrchestrator(t *testing.T) {
    mock := &MockReasoningService{
        ReturnValue: &ReasoningResult{Message: "test"},
        ReturnError: nil,
    }
    
    orchestrator := NewOrchestrator(mock, nil, nil)
    result, err := orchestrator.Process(context.Background(), &Request{})
    
    assert.NoError(t, err)
    assert.Equal(t, "test", result.Message)
    assert.Len(t, mock.Calls, 1)  // Verify reasoning was called
}
```

### Interface Design Checklist

When designing a new interface, ask yourself:

- [ ] **Is it small and focused?** (ISP - Interface Segregation Principle)
- [ ] **Does it define a single capability?** (SRP - Single Responsibility Principle)
- [ ] **Is it defined in the domain layer?** (DIP - Dependency Inversion Principle)
- [ ] **Can it have multiple implementations?** (OCP - Open/Closed Principle)
- [ ] **Is it easy to mock for testing?** (Testability)
- [ ] **Does it avoid exposing implementation details?** (Abstraction)
- [ ] **Are all methods related to the same concept?** (Cohesion)
- [ ] **Would a client ever need only some of these methods?** (If yes, split it)

### Summary: Interface-First Development Workflow

**Step-by-step process for adding new functionality:**

1. **Define the interface** in `internal/domain/services/` (Go) or `src/domain/services/` (Python)
2. **Document the contract** - What are the inputs, outputs, and error conditions?
3. **Create mock implementation** for testing
4. **Write tests** using the mock
5. **Implement in infrastructure** layer (e.g., `internal/infrastructure/providers/`)
6. **Wire dependencies** in composition root (`main.go` or `main.py`)
7. **Run tests** - Both unit tests (with mocks) and integration tests (with real implementation)

**Benefits:**
- ✓ Testable from day one
- ✓ Loosely coupled architecture
- ✓ Easy to swap implementations
- ✓ Parallel development possible
- ✓ Forced to think about contracts before code
- ✓ SOLID principles automatically enforced

## Clean Architecture Principles (Uncle Bob)

Clean Architecture, introduced by Robert C. Martin ("Uncle Bob"), provides a blueprint for organizing code to achieve independence from frameworks, UI, databases, and external systems. The core idea: **dependencies flow inward** toward the business logic.

### The Dependency Rule

**The Golden Rule of Clean Architecture:**

> **Dependencies can only point inward.** Inner layers cannot know anything about outer layers.

```
┌─────────────────────────────────────────────┐
│  External Interfaces (Frameworks, DBs, UI)  │  ← Outermost
├─────────────────────────────────────────────┤
│  Interface Adapters (Controllers, Gateways) │
├─────────────────────────────────────────────┤
│  Application Business Rules (Use Cases)     │
├─────────────────────────────────────────────┤
│  Enterprise Business Rules (Entities)       │  ← Innermost
└─────────────────────────────────────────────┘

       Dependencies flow ↓ inward only
```

**What this means:**
- **Entities** (innermost) know nothing about Use Cases, Controllers, or Databases
- **Use Cases** know about Entities, but not about Controllers or Databases
- **Interface Adapters** know about Use Cases and Entities, but not about specific Frameworks
- **Frameworks** (outermost) know about everything, but nothing depends on them

**Why this matters:**
- Business logic is **independent** of frameworks, databases, and UI
- Business logic is **testable** without external dependencies
- You can **swap** frameworks, databases, or UI without touching business logic
- Changes in external systems don't ripple into core business logic

### Clean Architecture Circles Mapped to DDD Layers

Our project structure follows Clean Architecture principles through DDD layers:

```
┌───────────────────────────────────────────────────────────────┐
│  Frameworks & Drivers (External Systems)                      │
│  - FastAPI framework                                          │
│  - HTTP clients (httpx)                                       │
│  - MCP SDK                                                    │
│  - Database drivers                                           │
│  - External APIs (OpenAI, GitLab, YouTrack)                   │
└───────────────────────────────────────────────────────────────┘
                          ↑ depends on ↑
┌───────────────────────────────────────────────────────────────┐
│  Interface Adapters (src/infrastructure/ + src/presentation/) │
│  - src/presentation/api/                                      │
│    - streaming_controller.py (HTTP handlers)                  │
│    - Request/Response DTOs                                    │
│  - src/infrastructure/                                        │
│    - llm/openai_client.py (implements domain interfaces)      │
│    - mcp/client.py (MCP protocol implementation)              │
│    - config/config.py (external config loading)               │
└───────────────────────────────────────────────────────────────┘
                          ↑ depends on ↑
┌───────────────────────────────────────────────────────────────┐
│  Application Business Rules (src/application/)                │
│  - src/application/services/                                  │
│    - orchestration_service.py (coordinates use cases)         │
│    - preprocessing_service.py (use case logic)                │
│    - postprocessing_service.py (use case logic)               │
│    - mcp_tool_selector.py (tool selection use case)           │
└───────────────────────────────────────────────────────────────┘
                          ↑ depends on ↑
┌───────────────────────────────────────────────────────────────┐
│  Enterprise Business Rules (src/domain/)                      │
│  - src/domain/models/ (entities, value objects)               │
│  - src/domain/services/ (business logic, interfaces)          │
│    - reasoning_service_impl.py (core reasoning logic)         │
│    - Interfaces: IReasoningService, ILLMProvider, etc.        │
└───────────────────────────────────────────────────────────────┘
                    ↑ innermost - no dependencies ↑
```

### Layer-by-Layer Breakdown

#### 1. Entities (Domain Models)
**Location**: `src/domain/models/` or `internal/domain/models/`

**Responsibility**: Core business objects and rules that are fundamental to the business, independent of any application.

**Examples in this project:**
```python
# src/domain/models/reasoning_context.py
@dataclass
class ReasoningContext:
    """Core entity representing reasoning state"""
    messages: List[Message]
    intent: Optional[str]
    selected_tools: List[str]
    metadata: dict

# src/domain/models/completion_request.py
@dataclass
class CompletionRequest:
    """Core entity for LLM requests"""
    model: str
    messages: List[Message]
    temperature: float
    stream: bool
```

**Go example:**
```go
// internal/domain/models/request.go
package models

type CompletionRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Temperature float64   `json:"temperature"`
    Stream      bool      `json:"stream"`
}
```

**Key characteristics:**
- No dependencies on any other layer
- Pure data structures with business rules
- No framework code, no database code, no UI code
- Can be used in any context

#### 2. Use Cases (Application Services)
**Location**: `src/application/services/` or `internal/application/services/`

**Responsibility**: Application-specific business rules. Orchestrates flow of data to/from entities and directs entities to use their business rules.

**Examples in this project:**
```python
# src/application/services/orchestration_service.py
class OrchestrationService:
    """Use case: Process user request through full pipeline"""
    
    def __init__(
        self,
        reasoning: IReasoningService,      # Domain interface
        preprocessing: PreprocessingService,
        postprocessing: PostprocessingService
    ):
        self.reasoning = reasoning
        self.preprocessing = preprocessing
        self.postprocessing = postprocessing
    
    async def process_request(self, request: CompletionRequest) -> CompletionResponse:
        # Orchestrate use case
        preprocessed = await self.preprocessing.process(request)
        reasoning_result = await self.reasoning.reason(preprocessed)
        final_result = await self.postprocessing.process(reasoning_result)
        return final_result
```

**Go example:**
```go
// internal/application/services/orchestrator.go
package services

type Orchestrator struct {
    reasoning domain.ReasoningService  // Domain interface
    provider  domain.LLMProvider       // Domain interface
}

func (o *Orchestrator) ProcessRequest(ctx context.Context, req *domain.CompletionRequest) (*domain.Response, error) {
    // Orchestrate use case
    reasoningResult, err := o.reasoning.Reason(ctx, req)
    if err != nil {
        return nil, err
    }
    
    return o.provider.Complete(ctx, reasoningResult)
}
```

**Key characteristics:**
- Depends on domain layer (entities + interfaces)
- Coordinates business logic, doesn't implement it
- No knowledge of HTTP, databases, or frameworks
- Uses interfaces for external dependencies (Dependency Inversion)

#### 3. Interface Adapters (Infrastructure + Presentation)
**Location**: 
- `src/infrastructure/` or `internal/infrastructure/` (adapters TO external systems)
- `src/presentation/` or `internal/presentation/` (adapters FROM external systems)

**Responsibility**: Convert data between the format most convenient for use cases/entities and the format most convenient for external systems (web, database, APIs).

**Examples in this project:**

**Infrastructure (Outbound Adapters):**
```python
# src/infrastructure/llm/openai_client.py
class OpenAIClient:
    """Adapter TO OpenAI API - implements domain interface"""
    
    async def stream_completion(self, request: CompletionRequest) -> AsyncIterator[CompletionChunk]:
        # Adapt domain model to OpenAI API format
        openai_request = {
            "model": request.model,
            "messages": [self._to_openai_message(m) for m in request.messages],
            "stream": True
        }
        
        # Call external API
        async for chunk in self.client.chat.completions.create(**openai_request):
            # Adapt OpenAI format back to domain model
            yield self._to_domain_chunk(chunk)
```

**Presentation (Inbound Adapters):**
```python
# src/presentation/api/streaming_controller.py
@app.post("/v1/chat/completions")
async def chat_completions(request: Request):
    """Adapter FROM HTTP - converts HTTP request to domain model"""
    
    # Parse HTTP request body
    body = await request.json()
    
    # Convert to domain model
    completion_request = CompletionRequest(
        model=body["model"],
        messages=[Message(**m) for m in body["messages"]],
        stream=body.get("stream", False)
    )
    
    # Call use case
    result = await orchestrator.process_request(completion_request)
    
    # Convert domain model to HTTP response
    return JSONResponse(content=result.to_dict())
```

**Go example:**
```go
// internal/infrastructure/providers/openai.go
package providers

type OpenAIProvider struct {
    client *http.Client
}

// Implements domain.LLMProvider interface
func (p *OpenAIProvider) StreamCompletion(ctx context.Context, req *domain.CompletionRequest) (<-chan domain.CompletionChunk, error) {
    // Adapt domain model to OpenAI API format
    openaiReq := p.toOpenAIRequest(req)
    
    // Call external API
    resp, err := p.client.Post(openaiURL, "application/json", openaiReq)
    
    // Adapt response back to domain model
    return p.parseSSEStream(resp.Body), nil
}
```

**Key characteristics:**
- Implements domain interfaces (Dependency Inversion)
- Handles format conversion (domain ↔ external)
- Contains framework-specific code
- Can be replaced without changing use cases or entities

#### 4. Frameworks & Drivers (External Layer)
**Location**: Not in your codebase - external dependencies

**Examples:**
- FastAPI framework
- OpenAI SDK
- GitLab Python library
- Redis client
- PostgreSQL driver

**Key characteristics:**
- External libraries and frameworks
- Your code depends on them, but they don't depend on your code
- Changes here (e.g., switching FastAPI to Flask) only affect adapter layer

### Dependency Flow: Why Domain Defines Interfaces

**The Key to Clean Architecture: Dependency Inversion**

```
❌ Bad (Traditional Layered Architecture):

Use Case → OpenAI Client (concrete)
                ↓
         OpenAI SDK (external)

Problem: Use case depends on concrete implementation.
Can't test without real OpenAI API. Can't swap providers easily.

✓ Good (Clean Architecture with DIP):

Use Case → ILLMProvider (interface)
                ↑ implements
         OpenAI Client (concrete)
                ↓
         OpenAI SDK (external)

Solution: Use case depends on abstraction.
Easy to test with mocks. Easy to swap providers.
```

**In code:**

```python
# Domain layer defines interface
# src/domain/services/llm_provider.py
from abc import ABC, abstractmethod

class ILLMProvider(ABC):
    """Domain abstraction - no implementation details"""
    
    @abstractmethod
    async def stream_completion(self, request: CompletionRequest) -> AsyncIterator[CompletionChunk]:
        pass

# Application layer depends on domain interface
# src/application/services/orchestration_service.py
class OrchestrationService:
    def __init__(self, llm_provider: ILLMProvider):  # Depends on abstraction
        self.llm_provider = llm_provider

# Infrastructure layer implements domain interface
# src/infrastructure/llm/openai_client.py
class OpenAIClient(ILLMProvider):  # Implements domain abstraction
    async def stream_completion(self, request: CompletionRequest):
        # Implementation using OpenAI SDK
        pass
```

**Why this works:**
1. **Domain layer** defines what it needs (interface)
2. **Infrastructure layer** provides what domain needs (implementation)
3. **Dependency points inward** (infrastructure → domain), not outward
4. **Use cases remain testable** (can inject mock implementation)

### Why Domain Layer Has No External Dependencies

**Principle**: The domain layer should have **zero dependencies** on external systems.

**Golang example (domain layer):**
```go
// internal/domain/services/reasoning.go
package services

// Only imports from standard library or other domain packages
import (
    "context"
    "internal/domain/models"
)

// Interface defines what domain needs
type ReasoningService interface {
    Reason(ctx context.Context, input *models.ReasoningInput) (*models.ReasoningResult, error)
}

// NO imports like:
// import "github.com/openai/openai-go"  ❌
// import "gorm.io/gorm"  ❌
// import "github.com/gin-gonic/gin"  ❌
```

**Benefits:**
1. **Testable** - No need to mock external systems for domain tests
2. **Portable** - Business logic can be reused in any context
3. **Independent** - Changes in frameworks don't affect business logic
4. **Fast builds** - Fewer dependencies = faster compilation
5. **Clear separation** - Forces you to think about abstractions

### Dependency Inversion in Action

**Example: Adding caching to LLM responses**

**Step 1: Domain defines interface**
```go
// internal/domain/services/cache.go
package services

type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}
```

**Step 2: Use case depends on interface**
```go
// internal/application/services/orchestrator.go
type Orchestrator struct {
    cache services.Cache  // Depends on domain interface
}

func (o *Orchestrator) ProcessRequest(ctx context.Context, req *domain.Request) (*domain.Response, error) {
    // Check cache
    cached, err := o.cache.Get(ctx, req.CacheKey())
    if err == nil {
        return parseCached(cached), nil
    }
    
    // Process and cache
    result := o.process(ctx, req)
    o.cache.Set(ctx, req.CacheKey(), result, 1*time.Hour)
    return result, nil
}
```

**Step 3: Infrastructure implements interface**
```go
// internal/infrastructure/cache/redis.go
package cache

import "github.com/redis/go-redis/v9"

type RedisCache struct {
    client *redis.Client
}

// Implements services.Cache interface
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
    return c.client.Get(ctx, key).Bytes()
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
    return c.client.Set(ctx, key, value, ttl).Err()
}
```

**Step 4: Wire in main.go**
```go
// cmd/proxy/main.go
func main() {
    // Create concrete implementation
    redisCache := cache.NewRedisCache(config.RedisURL)
    
    // Inject into use case (dependency inversion)
    orchestrator := services.NewOrchestrator(redisCache)
    
    // Use case doesn't know it's Redis - only knows Cache interface
}
```

**Result:**
- ✓ Use case testable with mock cache
- ✓ Can swap Redis for Memcached without changing use case
- ✓ Domain layer remains independent
- ✓ Dependencies point inward (infrastructure → domain)

### Clean Architecture Checklist

When building a new feature, verify:

- [ ] **Entities** - Are domain models free of framework code?
- [ ] **Interfaces in domain** - Does domain define interfaces for external dependencies?
- [ ] **Use cases independent** - Can use cases be tested without external systems?
- [ ] **Dependency direction** - Do all dependencies point inward?
- [ ] **No framework in business logic** - Is business logic framework-agnostic?
- [ ] **Adapter layer** - Are all external systems accessed through adapters?
- [ ] **Testability** - Can you test business logic with mocks?
- [ ] **Replaceability** - Can you swap frameworks/databases without touching business logic?

### Summary: Clean Architecture Benefits

**What you gain:**
- ✓ **Independent of Frameworks** - Business logic doesn't depend on FastAPI, Flask, or any framework
- ✓ **Testable** - Business logic can be tested without UI, database, or external systems
- ✓ **Independent of UI** - Can change from REST to GraphQL without touching business logic
- ✓ **Independent of Database** - Can swap PostgreSQL for MongoDB without changing use cases
- ✓ **Independent of External Systems** - Business logic doesn't know about OpenAI, GitLab, or YouTrack

**How to achieve it:**
1. **Define interfaces in domain** layer
2. **Implement interfaces in infrastructure** layer
3. **Inject dependencies** at composition root (main.go/main.py)
4. **Follow Dependency Rule** - dependencies always point inward
5. **Keep domain pure** - no external dependencies in domain layer

**Clean Architecture + DDD + SOLID = Rock-Solid Codebase**

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
