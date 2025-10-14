# ADK LLM Proxy - Project Context for Claude Code

## Project Overview

Intelligent LLM proxy server with agent layer on top of LLM APIs. Built with:
- **Google Agent Development Kit (ADK)** for intelligent processing
- **Domain-Driven Design (DDD)** architecture
- **Model Context Protocol (MCP)** integration for GitLab and YouTrack
- **FastAPI** for web layer
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

Use Python **only** when Golang implementation is impractical:
1. **MCP Servers** - Python MCP SDK is the only official implementation
2. **Google ADK Integration** - ADK is Python-only
3. **Complex AI/ML Workflows** - transformers, langchain, etc.

**Rule:** If you can implement it reasonably in Golang, implement it in Golang.

### Golang Guidelines
- Follow standard Go project layout (cmd/, internal/, pkg/)
- Use interfaces for abstraction (Dependency Inversion)
- Error handling: return errors with context, don't panic
- Concurrency: goroutines + channels
- Small, focused interfaces
- Table-driven tests with `testify`

## Architecture

### Request Processing Pipeline
```
Request → Preprocessing → Reasoning → LLM → Postprocessing → Response
```

**DDD Layers:**
1. **Domain** (`internal/domain/`) - Core business logic, interfaces, models
2. **Application** (`internal/application/`) - Use cases, orchestration
3. **Infrastructure** (`internal/infrastructure/`) - External system integrations
4. **Presentation** (`internal/presentation/`) - API controllers, DTOs

**Dependency Rule:** Dependencies always point inward toward domain.

## SOLID Principles (Condensed)

### Single Responsibility (SRP)
Each component has one reason to change.

```go
// Good: Separate responsibilities
type UserRepository struct { db *sql.DB }
func (r *UserRepository) FindByID(id string) (*User, error) { /* data access */ }

type UserValidator struct {}
func (v *UserValidator) Validate(user *User) error { /* validation */ }

type UserService struct { repo UserRepository; validator UserValidator }
func (s *UserService) CreateUser(user *User) error {
    if err := s.validator.Validate(user); err != nil { return err }
    return s.repo.Save(user)
}
```

**DDD Mapping:** Each service has single responsibility per layer.

### Open/Closed (OCP)
Open for extension, closed for modification. Use interfaces.

```go
// Good: Interface allows extension without modification
type LLMProvider interface {
    Name() string
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

type OpenAIProvider struct {}
func (p *OpenAIProvider) StreamCompletion(...) (<-chan CompletionChunk, error) { /* impl */ }

type OllamaProvider struct {}
func (p *OllamaProvider) StreamCompletion(...) (<-chan CompletionChunk, error) { /* impl */ }

// Orchestrator uses interface, never modified for new providers
type Orchestrator struct { providers map[string]LLMProvider }
```

**DDD Mapping:** Interfaces in domain, implementations in infrastructure.

### Liskov Substitution (LSP)
All implementations must honor interface contracts.

```go
// Good: All implementations honor contract
type Workflow interface {
    Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error)
}

type DefaultWorkflow struct{}
func (w *DefaultWorkflow) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningResult, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()  // Respects context
    default:
        return &ReasoningResult{Message: "Hello"}, nil
    }
}
```

**DDD Mapping:** All infrastructure implementations honor domain contracts.

### Interface Segregation (ISP)
Small, focused interfaces. Clients depend only on what they need.

```go
// Good: Small, focused interfaces
type ToolExecutor interface {
    ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error)
}

type ToolDiscoverer interface {
    ListTools(ctx context.Context) ([]Tool, error)
}

type Orchestrator struct {
    executor ToolExecutor  // Only needs execution
}

type Registry struct {
    discoverer ToolDiscoverer  // Only needs discovery
}
```

**Go Proverb:** "The bigger the interface, the weaker the abstraction."

### Dependency Inversion (DIP)
High-level modules depend on abstractions, not concrete implementations.

```go
// Good: Domain defines interface
// File: internal/domain/services/provider.go
type LLMProvider interface {
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

// Application depends on domain interface
type Orchestrator struct {
    provider LLMProvider  // Depends on abstraction
}

// Infrastructure implements domain interface
// File: internal/infrastructure/providers/openai.go
type OpenAIProvider struct {}
func (p *OpenAIProvider) StreamCompletion(...) (<-chan CompletionChunk, error) { /* impl */ }

// Wire in main.go
func main() {
    provider := providers.NewOpenAIProvider(config)
    orchestrator := services.NewOrchestrator(provider)
}
```

**Dependency Flow:**
```
Presentation → Application → Domain ← Infrastructure
                              ↑
                      (interfaces defined here)
```

**Key Rules:**
1. Domain defines interfaces
2. Infrastructure implements interfaces
3. Application depends on domain interfaces only
4. Dependencies point inward toward domain

## Interface-Driven Design

**Golden Rule:** Define interfaces **before** implementation.

### Golang Interface Patterns

#### 1. Small, Focused Interfaces
```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
type Closer interface { Close() error }

// Compose when needed
type ReadWriteCloser interface { Reader; Writer; Closer }
```

#### 2. Interface Composition
```go
// Domain layer defines focused interfaces
type Streamer interface {
    StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan CompletionChunk, error)
}

type Namer interface { Name() string }
type HealthChecker interface { CheckHealth(ctx context.Context) error }

// Compose
type LLMProvider interface { Streamer; Namer; HealthChecker }

// Clients depend on minimal interface
type Orchestrator struct {
    streamer Streamer  // Only needs streaming
}
```

### Dependency Injection Pattern

```go
// Constructor injection
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
    return &Orchestrator{reasoning, provider, cache}
}

// Testing with mocks
func TestOrchestrator(t *testing.T) {
    mock := &MockLLMProvider{}
    orchestrator := NewOrchestrator(nil, mock, nil)
    // Test with mock
}
```

### Interface Design Checklist
- [ ] Small and focused? (ISP)
- [ ] Single capability? (SRP)
- [ ] Defined in domain layer? (DIP)
- [ ] Multiple implementations possible? (OCP)
- [ ] Easy to mock? (Testability)
- [ ] No implementation details exposed? (Abstraction)

## Clean Architecture (Condensed)

**Dependency Rule:** Dependencies can only point inward.

```
┌─────────────────────────────────────┐
│  Frameworks & Drivers (External)    │  ← Outermost
├─────────────────────────────────────┤
│  Interface Adapters (Infra + Pres) │
├─────────────────────────────────────┤
│  Application Business Rules (App)   │
├─────────────────────────────────────┤
│  Enterprise Business Rules (Domain) │  ← Innermost (no external deps)
└─────────────────────────────────────┘
       Dependencies flow ↓ inward
```

### Layer Responsibilities

**Domain Layer** (`internal/domain/`):
- Core business logic, domain models
- Defines interfaces for external dependencies
- Zero external dependencies

**Application Layer** (`internal/application/`):
- Use cases, orchestration
- Depends on domain interfaces only

**Infrastructure Layer** (`internal/infrastructure/`):
- Implements domain interfaces
- External system integrations

**Presentation Layer** (`internal/presentation/`):
- API controllers, DTOs
- Converts between HTTP and domain models

### Example: Caching Feature

```go
// Step 1: Domain defines interface
// internal/domain/services/cache.go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// Step 2: Application uses interface
// internal/application/services/orchestrator.go
type Orchestrator struct {
    cache services.Cache  // Depends on abstraction
}

// Step 3: Infrastructure implements
// internal/infrastructure/cache/redis.go
type RedisCache struct { client *redis.Client }
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
    return c.client.Get(ctx, key).Bytes()
}

// Step 4: Wire in main.go
func main() {
    cache := cache.NewRedisCache(config.RedisURL)
    orchestrator := services.NewOrchestrator(cache)
}
```

**Benefits:**
- ✓ Testable with mocks
- ✓ Can swap Redis for Memcached without changing use case
- ✓ Domain remains independent

## Function Composition & Readability

### Core Principles
1. Functions do ONE thing (SRP at function level)
2. Functions are small (5-30 lines max)
3. Function names are self-explanatory
4. One level of abstraction per function
5. Compose complex operations from simple functions

### Small Functions Pattern

```go
// Good: Composed pipeline (high-level)
func ProcessUser(user *User) error {
    if err := ValidateUser(user); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    NormalizeUser(user)
    AssignPermissions(user)
    return SaveUser(user)
}

// Each helper does ONE thing (low-level)
func ValidateUser(user *User) error {
    if user.Email == "" { return errors.New("email required") }
    if !isValidEmail(user.Email) { return errors.New("invalid email") }
    return nil
}

func NormalizeUser(user *User) {
    user.Email = strings.ToLower(user.Email)
    user.Name = strings.TrimSpace(user.Name)
}

func AssignPermissions(user *User) {
    if user.IsAdmin {
        user.Permissions = []string{"read", "write", "delete", "admin"}
    } else {
        user.Permissions = []string{"read"}
    }
}
```

### Naming Patterns

| Pattern | Example | Use When |
|---------|---------|----------|
| **verb_noun** | `calculateTotal`, `sendEmail` | Action on a thing |
| **is_/has_/can_** | `isValid`, `hasPermission` | Boolean predicates |
| **get_/fetch_** | `getUserByID`, `fetchData` | Retrieving data |
| **create_/build_** | `createSession`, `buildRequest` | Constructing objects |

### Early Return Pattern

```go
// Good: Flat structure with guard clauses
func GetUserDiscount(user *User) float64 {
    if !user.IsAuthenticated { return 0.00 }
    if !user.SubscriptionActive { return 0.05 }
    if user.LoyaltyPoints > 1000 { return 0.30 }
    if user.LoyaltyPoints > 500 { return 0.20 }
    return 0.10
}
```

### Function Checklist
- [ ] <30 lines? (ideally 5-15)
- [ ] Does ONE thing?
- [ ] Name self-explanatory?
- [ ] Nesting minimized? (max 2 levels)
- [ ] Abstraction level consistent?
- [ ] Easy to test?
- [ ] Reusable?

## Architecture-First Planning

**Rule:** "Hours of planning save weeks of refactoring."

### Planning Process

**Step 1: Define Interfaces**
- What operations are needed?
- What are inputs/outputs/errors?
- Where should interface live? (domain layer)

**Step 2: Map to DDD Layers**
```
Is it core business logic?
├─ YES → Domain Layer (internal/domain/)
└─ NO → Is it orchestration?
    ├─ YES → Application Layer (internal/application/)
    └─ NO → Is it external integration?
        ├─ YES → Infrastructure Layer (internal/infrastructure/)
        └─ NO → Presentation Layer (internal/presentation/)
```

**Step 3: Design for Testability**
- Plan mocks for interfaces
- Identify unit vs integration tests
- Define test data needs

**Step 4: Plan Function Composition**
- Break down into 5-15 line functions
- Each function has single responsibility
- Clear pipeline structure

**Step 5: Validate Against SOLID & Clean Architecture**
- [ ] SRP - Single responsibility per component
- [ ] OCP - Can extend without modifying
- [ ] LSP - All implementations honor contracts
- [ ] ISP - Interfaces small and focused
- [ ] DIP - Dependencies point to abstractions
- [ ] Dependency Rule - All deps point inward
- [ ] Domain purity - No external deps in domain

### Planning Template

```markdown
Feature: [Name]

Interfaces:
- [List interfaces with method signatures]
- Location: internal/domain/services/

Layer Mapping:
- Domain: [Components]
- Application: [Components]
- Infrastructure: [Components]
- Presentation: [Components]

Testing Strategy:
- Unit tests: [What to mock]
- Integration tests: [End-to-end scenarios]

Function Composition:
- Main function (high-level, 5-10 lines)
- Helper functions (low-level, 5-15 lines each)

Validation:
- ✓ SOLID principles satisfied
- ✓ Clean Architecture followed
- ✓ Testable with mocks
```

## DDD Principles & File Organization

### Layer Responsibilities

1. **Domain** (`internal/domain/`) - Core business logic, independent of frameworks
2. **Application** (`internal/application/`) - Use cases, orchestration
3. **Infrastructure** (`internal/infrastructure/`) - External integrations
4. **Presentation** (`internal/presentation/`) - API endpoints, DTOs

### File Placement Rules

**DO:**
- Business logic → `internal/domain/services/`
- Orchestration → `internal/application/services/`
- API integrations → `internal/infrastructure/`
- HTTP endpoints → `internal/presentation/api/`

**DON'T:**
- Mix layers
- Put business logic in infrastructure
- Access infrastructure directly from domain
- Create files outside layer structure

### Test Organization
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

**Test naming:** `*_test.go` (Golang)

## Coding Standards

### Golang Standards
- **Naming:** camelCase (private), PascalCase (public)
- **Error handling:** Always check; return with context
```go
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("failed to do something: %w", err)
}
```
- **Interfaces:** Small and focused
- **Comments:** godoc-style for public APIs
- **Formatting:** `gofmt` or `goimports`
- **Testing:** Table-driven tests, `testify` for assertions

### Return Value Pattern
Most service functions return:
```go
// Success
return &Result{Status: "success", Data: data}, nil

// Error
return nil, fmt.Errorf("operation failed: %w", err)
```

## Important Files

### Configuration
- **config.yaml:** Main config (LLM providers, MCP servers)
- **internal/infrastructure/config/config.go:** Config loading

### Entry Points
- **cmd/proxy/main.go:** Main server entry
- **internal/presentation/api/:** API controllers

### Core Services
- **internal/application/services/orchestration_service.go:** Pipeline orchestrator
- **internal/domain/services/:** Business logic
- **internal/infrastructure/:** External integrations

## Development Workflow

### Running
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
```

### Testing Guidelines
- Use table-driven tests
- Mock external dependencies
- Test error cases and edge cases
- Integration tests for full flow

### Adding Features
1. Define interface in domain layer
2. Create mock implementation
3. Write tests with mock
4. Implement in infrastructure
5. Wire in main.go
6. Run tests

## Code Review Checklist

### Architecture & Design
- [ ] **Golang-first** (Python only for MCP/ADK)
- [ ] **Proper layer separation** (domain/application/infrastructure/presentation)
- [ ] **Files in correct directories** (follow DDD + Go layout)
- [ ] **Test files in tests/golang/**
- [ ] **Interfaces defined before implementations**
- [ ] **Dependency Rule respected** (deps point inward)

### SOLID Principles
- [ ] **SRP** - One reason to change
- [ ] **OCP** - Extend via interfaces
- [ ] **LSP** - Implementations honor contracts
- [ ] **ISP** - Small, focused interfaces
- [ ] **DIP** - Depend on abstractions

### Code Quality
- [ ] **Functions small** (<30 lines, single responsibility)
- [ ] **Function names self-explanatory**
- [ ] **No business logic in infrastructure**
- [ ] **No external deps in domain**
- [ ] **Error handling** (return errors with context)
- [ ] **Comments on public APIs** (godoc-style)

### Testing & Safety
- [ ] **Testable design** (DI, mockable interfaces)
- [ ] **Tests included**
- [ ] **Concurrency safe** (proper goroutine/channel usage)

## Common Issues & Solutions

### MCP Integration
- MCP servers configured in `config.yaml`
- Each server provides tools, resources, prompts
- Use `mcp_registry` for connections

### Configuration Changes
After changing `config.yaml`:
1. Restart server (no hot-reload)
2. Verify MCP servers connect
3. Test with curl

### Import Errors
- Always run from project root
- Check PYTHONPATH for Python components

## Environment Variables

**Required:**
- **OPENAI_API_KEY:** OpenAI provider
- **GITLAB_URL, GITLAB_TOKEN:** GitLab MCP
- **YOUTRACK_BASE_URL, YOUTRACK_TOKEN:** YouTrack MCP

**Optional:**
- **LLM_PROVIDER, LLM_MODEL:** Override defaults
- **DEBUG:** Enable debug mode

## Git Workflow

- **Main branch:** `main`
- **Commits:** Conventional commits (feat:, fix:, docs:)
- **Task references:** Include YouTrack IDs when applicable
- **Test files:** Must be in `tests/`, never in root

## Performance & Security

**Performance:**
- Use async for I/O
- MCP connections are persistent
- Streaming reduces latency
- Configure timeouts in config.yaml

**Security:**
- API keys in config.yaml (DO NOT commit!)
- Use environment variables
- MCP servers in isolated processes
- SSL verification enabled

## Helpful Commands

```bash
# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o bin/proxy-linux ./src/golang/cmd/proxy
GOOS=darwin GOARCH=amd64 go build -o bin/proxy-darwin ./src/golang/cmd/proxy

# Test coverage
go test -cover ./src/golang/...
go test -coverprofile=coverage.out ./src/golang/...
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. ./src/golang/...
```

## Getting Help

- **Architecture questions:** Check `articles/*.md`
- **MCP integration:** See `mcps/*/README.md`
- **Workflows:** See `workflows/README.md`
- **Main README:** `/README.md` for quick start
