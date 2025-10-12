# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

This is an **ADK-based LLM Reverse Proxy Server** that provides an intelligent streaming-first proxy for LLM providers (OpenAI, Ollama, DeepSeek). The system adds an intelligent agent layer using Google's Agent Development Kit (ADK) that preprocesses requests, applies reasoning, and postprocesses responses.

**Key Features:**
- **Streaming-only architecture** - no non-streaming support
- **Multi-provider support** - OpenAI, Ollama, DeepSeek
- **Intelligent processing pipeline** - preprocessing → reasoning → LLM → postprocessing
- **MCP (Model Context Protocol) integration** - comprehensive tool discovery and execution
- **OpenAI-compatible API** - drop-in replacement

## Common Development Commands

### Starting the Server
```bash
# Start with OpenAI provider (most common)
python main.py -provider openai -model gpt-4o-mini

# Start with local Ollama
python main.py -provider ollama -model mistral

# Start with DeepSeek
python main.py -provider deepseek -model deepseek-chat
```

### Environment Setup
```bash
# Install dependencies
pip install -r requirements.txt

# Set API keys (required for certain providers)
export OPENAI_API_KEY="your-key-here"
export DEEPSEEK_API_KEY="your-key-here"

# Copy and customize configuration
cp config.yaml.example config.yaml
```

### Testing
```bash
# Run comprehensive MCP tests
python test_comprehensive_mcp.py

# Run specific MCP tests
python test_mcp_server.py
python test_real_mcp.py

# Test the API endpoint
curl -X POST http://localhost:8001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

### Development Tools
```bash
# Debug mode with detailed logging
DEBUG=true python main.py -provider openai -model gpt-4o-mini

# Check health endpoint
curl http://localhost:8001/health

# List available models
curl http://localhost:8001/v1/models
```

### Code Quality
```bash
# Format code (if black is installed)
black .

# Lint code (if flake8 is installed)
flake8 .

# Run pytest tests (if pytest is installed)
pytest tests/ -v
```

## High-Level Architecture

The system follows **Domain-Driven Design** principles with a clean separation of concerns:

```
Request Flow:
Client → FastAPI → Orchestration Service → [Preprocessing → Reasoning → LLM Provider → Postprocessing] → Client

Directory Structure:
src/
├── presentation/        # FastAPI web layer, streaming controllers
├── application/         # Business logic, orchestration services
├── domain/             # Core reasoning and content filtering services
└── infrastructure/     # External integrations (ADK, MCP, providers)
```

### Key Components

**1. Streaming Controller (`src/presentation/api/streaming_controller.py`)**
- FastAPI application with OpenAI-compatible endpoints
- Handles CORS, middleware, and request logging
- **Streaming-only** - no support for non-streaming responses
- Routes all requests through the ADK orchestrator

**2. Orchestration Service (`src/application/services/orchestration_service.py`)**
- Coordinates the complete request/response pipeline
- Manages preprocessing → reasoning → provider → postprocessing flow
- Provides specialized functions for streaming scenarios
- Handles error recovery and fallbacks

**3. Provider Integration (`src/infrastructure/repositories/llm_proxy_repository.py`)**
- Abstracts communication with different LLM providers
- Handles authentication, rate limiting, and error handling
- Supports OpenAI, Ollama, and DeepSeek APIs

**4. Agent Processing Pipeline:**
- **Preprocessing Service**: Request validation, context injection, metadata extraction
- **Reasoning Service**: Intelligent request enhancement with ADK reasoning
- **Postprocessing Service**: Response analysis, filtering, content enhancement

## MCP (Model Context Protocol) Integration

The system includes comprehensive MCP support for connecting to external tools and services:

### Configuration
MCP servers are configured in `config.yaml`:

```yaml
mcp:
  enabled: true
  health_check_interval: 60.0
  connection_timeout: 30.0
  
  servers:
    - name: "filesystem-server"
      transport: "stdio"
      command: "mcp-server-filesystem"
      args: ["/workspace"]
      enabled: true
    
    - name: "web-api-server"
      transport: "sse"
      url: "https://api.example.com/mcp/sse"
      headers:
        Authorization: "Bearer token"
```

### MCP Architecture
Located in `src/infrastructure/mcp/`:
- **Client**: Core MCP client with multi-transport support (stdio, SSE, WebSocket)
- **Registry**: Server connection management and lifecycle
- **Discovery**: Automatic tool enumeration from all connected servers
- **Tool Registry**: Unified interface for executing tools across servers
- **Health Monitoring**: Advanced health checks with alerting
- **Introspection**: Deep analysis of tool capabilities and performance

### MCP Usage in Processing Pipeline
The MCP system integrates into all three processing phases:
- **Preprocessing**: Context enrichment using MCP tools
- **Reasoning**: Dynamic tool selection during reasoning
- **Postprocessing**: Response validation and enhancement

## Configuration Management

### Configuration Files
- **`config.yaml`** - Main configuration file (copy from `config.yaml.example`)
- **`.env`** - Environment variables (optional)
- **`src/infrastructure/config/config.py`** - Configuration loader with validation

### Configuration Hierarchy
1. Environment variables (highest priority)
2. `config.yaml` file
3. Default values (lowest priority)

### Key Settings
```yaml
providers:
  openai:
    api_key: "your-key"
    endpoint: "https://api.openai.com/v1"
    default_model: "gpt-4o-mini"
    
server:
  host: "0.0.0.0"
  port: 8000
  debug: false
  
processing:
  enable_context_injection: true
  enable_response_analytics: true
  max_context_length: 4000
```

## Development Workflow

### Project Structure
- **Main Entry Point**: `main.py` - CLI interface with provider selection
- **Core Server**: `src/presentation/api/streaming_controller.py` - FastAPI application
- **Business Logic**: `src/application/services/` - Orchestration and processing services
- **Domain Logic**: `src/domain/services/` - Reasoning and content filtering
- **Infrastructure**: `src/infrastructure/` - External integrations (ADK, MCP, providers)

### Adding New Providers
1. Extend `Config` class in `src/infrastructure/config/config.py`
2. Add provider handling in `src/infrastructure/repositories/llm_proxy_repository.py`
3. Update `main.py` argument parser
4. Add configuration section in `config.yaml.example`

### Debugging Tips
- Use `DEBUG=true` environment variable for detailed logging
- Check `/health` endpoint for system status
- Monitor MCP server health through the registry
- All requests include ADK orchestrator tracing

## Key Implementation Details

### Streaming Architecture
The system is **streaming-only** by design:
- No support for non-streaming responses
- All processing happens during stream setup
- Content filtering occurs before streaming to LLM
- Postprocessing results are appended after main response

### Agent Processing Flow
```python
# Preprocessing Phase
orchestrator_input = {"request_data": request, "provider": provider}
preprocessing_result = await llm_proxy_orchestrator_preprocessing_only(orchestrator_input)

# Reasoning Phase  
reasoning_request = preprocessing_result["processed_request"]
async for reasoning_step in reasoning_pipeline(reasoning_request):
    yield reasoning_step  # Stream reasoning steps to client

# LLM Streaming Phase
filtered_request = filter_messages_for_llm(enhanced_request["messages"])
# Stream from provider...

# Postprocessing Phase
postprocessing_result = await execute_postprocessing_agent(content)
unified_analysis = create_unified_analysis(reasoning_metadata, content)
```

### Content Filtering
The system filters out reasoning and analysis content before sending to LLM providers:
- **`filter_messages_for_llm()`** - Removes internal processing messages
- **`add_analysis_markers()`** - Marks postprocessing content for clear separation
- Ensures clean conversation flow while preserving intelligent enhancements

### Error Handling
- Comprehensive error handling at each pipeline stage
- Graceful degradation when services are unavailable
- Detailed error reporting with context preservation
- Automatic retry logic for transient failures

## MCP Roadmap Integration

The project includes a comprehensive MCP roadmap (`MCP_ROADMAP.md`) with planned features:

**Phase 0**: MCP Client Foundation (✅ Complete)
- Protocol implementation, configuration, tool discovery, testing

**Phase 1**: Internal MCP Server Framework (planned)
- Host MCP servers within the project (`mcps/` directory)

**Phase 2**: YouTrack MCP Server (planned)
- Epic/task tracking and analysis

**Phase 3**: GitLab MCP Server (planned) 
- Code analysis correlation with YouTrack

**Phase 4**: RAG MCP Server (planned)
- Vector and graph database integration

**Phase 5**: Intelligent MCP Orchestration (planned)
- Full integration into preprocessing/reasoning/postprocessing

## Troubleshooting

### Common Issues

**"Google ADK not found"**
```bash
pip install google-adk
```

**"Can't connect to provider"**
- Check API key configuration
- Verify base URL settings
- For Ollama: ensure `ollama serve` is running

**MCP connection failures**
- Verify MCP server installation: `pip install mcp-server-filesystem`
- Check server command paths in `config.yaml`
- Review MCP server logs in debug mode

**Import errors**
- Always run from project root directory
- Check Python path includes current directory
- Verify all dependencies are installed

### Health Checks
```bash
# Basic health check
curl http://localhost:8001/health

# Detailed server info
curl http://localhost:8001/

# Test MCP integration
python test_comprehensive_mcp.py
```

The system provides comprehensive logging and error reporting to help diagnose issues quickly.