# ADK LLM Proxy - Project Context for Claude Code

## Project Overview

This is an intelligent LLM proxy server that adds an agent layer on top of LLM APIs. It's built with:
- **Google Agent Development Kit (ADK)** for intelligent processing
- **Domain-Driven Design (DDD)** architecture
- **Model Context Protocol (MCP)** integration for GitLab and YouTrack
- **FastAPI** for the web layer
- **Streaming-first** architecture

## Architecture

### Directory Structure
```
src/
├── application/     # Business logic & orchestration (use cases, services)
├── domain/          # Core business logic (reasoning services, domain models)
├── infrastructure/  # External integrations (ADK, LLM APIs, MCP, config)
└── presentation/    # Web layer (FastAPI controllers, endpoints)

mcps/
├── gitlab/          # GitLab MCP server (repository analysis, commit tracking)
├── youtrack/        # YouTrack MCP server (epic/task tracking, analytics)
└── template/        # Template for creating new MCP servers

workflows/
├── default/         # Standard reasoning pipeline
├── enhanced/        # LLM-powered multi-agent reasoning
└── empty/           # No-op workflow (pass-through)
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

### Python Style
- **Type hints**: Always use type hints for function parameters and return values
- **Async/await**: All I/O operations must be async (httpx, MCP calls, etc.)
- **Error handling**: Use try/except with specific error types; return status dicts
- **Docstrings**: Use Google-style docstrings for all public functions

### DDD Principles
- **Domain models** go in `src/domain/`
- **Application services** (orchestration) go in `src/application/services/`
- **Infrastructure** (external APIs, config) goes in `src/infrastructure/`
- **Presentation** (FastAPI routes) goes in `src/presentation/api/`

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
- Tests go in `tests/` directory
- MCP integration tests: `tests/test_mcp_*.py`
- Use `pytest` with `pytest-asyncio` for async tests
- Manual test scripts available: `test_*.py` in root

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
- [ ] Type hints on all functions
- [ ] Async/await for all I/O
- [ ] Error handling with try/except
- [ ] Return status dicts ({"status": "success/error", ...})
- [ ] Docstrings on public functions
- [ ] Proper layer separation (domain/application/infrastructure)
- [ ] No blocking I/O operations
- [ ] Streaming properly implemented (async generators)
- [ ] MCP registry used correctly for server management
- [ ] Config changes documented

## Git Workflow

- **Main branch**: `dev` (NOT main/master)
- **Commit messages**: Use conventional commits (feat:, fix:, docs:, etc.)
- **Reference tasks**: Include YouTrack task IDs in commits (e.g., "PROJ-123")

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

```bash
# Install dependencies
pip install -r requirements.txt

# Run tests
pytest tests/

# Check code style
black src/ mcps/
flake8 src/ mcps/

# Test MCP connection
python test_mcp_server.py

# Test GitLab integration
python test_gitlab_integration.py

# Test YouTrack integration
python test_youtrack_integration.py

# Start server on custom port
python main.py -provider openai -model gpt-4o-mini
# (Port configured in config.yaml: server.port)
```

## Getting Help

- **Architecture questions**: Check `articles/*.md` for detailed explanations
- **MCP integration**: See `mcps/*/README.md` for server-specific docs
- **Workflows**: See `workflows/README.md` for reasoning customization
- **Main README**: `/README.md` for quick start and overview
