# MCP Integration Guide

## Overview

The ADK LLM Proxy implements comprehensive Model Context Protocol (MCP) integration, enabling intelligent tool discovery, execution, and orchestration across multiple MCP servers. This guide covers architecture, configuration, usage, and best practices.

## Table of Contents

1. [Architecture](#architecture)
2. [Getting Started](#getting-started)
3. [Configuration](#configuration)
4. [MCP Servers](#mcp-servers)
5. [Tool Discovery & Execution](#tool-discovery--execution)
6. [Intelligent Orchestration](#intelligent-orchestration)
7. [Performance & Optimization](#performance--optimization)
8. [Security](#security)
9. [Monitoring & Debugging](#monitoring--debugging)
10. [Best Practices](#best-practices)

---

## Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     ADK LLM Proxy                           │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │Preprocessing │→ │  Reasoning   │→ │Postprocessing│     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
│         │                  │                  │             │
│         └──────────────────┼──────────────────┘             │
│                            ↓                                │
│                   ┌─────────────────┐                       │
│                   │ MCP Orchestrator│                       │
│                   └─────────────────┘                       │
│                            │                                │
│         ┌──────────────────┼──────────────────┐            │
│         ↓                  ↓                  ↓             │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐       │
│  │  YouTrack  │    │   GitLab   │    │    RAG     │       │
│  │MCP Server  │    │ MCP Server │    │ MCP Server │       │
│  └────────────┘    └────────────┘    └────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

#### 1. MCP Client (`src/infrastructure/mcp/client.py`)
- Manages connections to MCP servers
- Handles stdio, SSE, and WebSocket transports
- Provides tool enumeration and execution APIs
- Maintains connection health and auto-reconnect

#### 2. MCP Registry (`src/infrastructure/mcp/registry.py`)
- Centralized registry of all MCP servers
- Dynamic server discovery and registration
- Tool aggregation across servers
- Server lifecycle management

#### 3. MCP Tool Selector (`src/application/services/mcp_tool_selector.py`)
- Intelligent tool selection based on intent
- LLM-powered capability matching
- Multi-tool orchestration planning
- Fallback and error handling

#### 4. Reasoning Service (`src/domain/services/reasoning_service_impl.py`)
- Integrates MCP tools into reasoning pipeline
- Dynamic tool discovery and execution
- Context enrichment from tool results
- Multi-agent reasoning coordination

---

## Getting Started

### Prerequisites

```bash
# Install required packages
pip install mcp anthropic google-adk

# Optional integrations
pip install youtrack-rest-api python-gitlab chromadb neo4j
```

### Quick Start

1. **Configure MCP Servers** (`config.yaml`):

```yaml
mcp:
  servers:
    youtrack:
      command: python
      args:
        - -m
        - mcps.youtrack.server
      env:
        YOUTRACK_BASE_URL: "https://your-instance.youtrack.cloud"
        YOUTRACK_TOKEN: "your-token"

    gitlab:
      command: python
      args:
        - -m
        - mcps.gitlab.server
      env:
        GITLAB_URL: "https://gitlab.com"
        GITLAB_TOKEN: "your-token"
```

2. **Start the Proxy**:

```bash
python main.py -provider openai -model gpt-4o-mini
```

3. **Test MCP Integration**:

```bash
# Check MCP connectivity
make check-mcp

# Test tool discovery
python scripts/debug_mcp_connection.py
```

---

## Configuration

### config.yaml Structure

```yaml
mcp:
  # Server definitions
  servers:
    server_name:
      command: <executable>        # Python, node, etc.
      args: [<arg1>, <arg2>]      # Server startup arguments
      env:                         # Environment variables
        KEY: value

  # Discovery settings
  discovery:
    enabled: true
    interval: 60                   # Rediscovery interval (seconds)
    timeout: 30                    # Discovery timeout (seconds)

  # Tool execution
  execution:
    timeout: 120                   # Tool execution timeout (seconds)
    max_retries: 3                 # Retry failed executions
    parallel_limit: 5              # Max parallel tool calls

  # Caching
  cache:
    enabled: true
    ttl: 3600                      # Cache TTL (seconds)
    max_size: 1000                 # Max cached results
```

### Environment Variables

MCP servers often require sensitive credentials. Use environment variables:

```bash
# YouTrack
export YOUTRACK_BASE_URL="https://your-instance.youtrack.cloud"
export YOUTRACK_TOKEN="your-api-token"

# GitLab
export GITLAB_URL="https://gitlab.com"
export GITLAB_TOKEN="your-personal-access-token"

# RAG (optional)
export CHROMA_HOST="localhost"
export CHROMA_PORT="8000"
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USER="neo4j"
export NEO4J_PASSWORD="password"
```

---

## MCP Servers

### Built-in Servers

#### 1. YouTrack MCP Server (`mcps/youtrack/`)

**Purpose**: Epic and task tracking, project management analytics

**Available Tools**:
- `find_epic` - Search for epics by name, ID, or tag
- `get_epic_details` - Get detailed epic information
- `list_epic_tasks` - List all tasks in an epic
- `get_task_details` - Get task information
- `analyze_task_activity` - Analyze recent task activity
- `analyze_epic_progress` - Track epic progress over time
- `generate_epic_report` - Comprehensive epic analytics

**Example Usage**:
```python
# Via API
{
  "model": "gpt-4o-mini",
  "messages": [
    {"role": "user", "content": "What's the status of the Authentication epic?"}
  ]
}

# The proxy automatically discovers and uses YouTrack tools
```

#### 2. GitLab MCP Server (`mcps/gitlab/`)

**Purpose**: Code repository analysis, commit tracking, code-to-task correlation

**Available Tools**:
- `find_project` - Search for GitLab projects
- `get_recent_commits` - Get commit history
- `analyze_commit_messages` - Analyze commit patterns
- `get_merge_requests` - Track merge requests
- `link_commits_to_tasks` - Correlate commits with YouTrack tasks
- `analyze_code_complexity` - Code complexity metrics
- `get_code_metrics` - Lines changed, files modified

**Example Usage**:
```python
{
  "model": "gpt-4o-mini",
  "messages": [
    {"role": "user", "content": "Show me commits related to AUTH-123"}
  ]
}
```

#### 3. RAG MCP Server (`mcps/rag/`)

**Purpose**: Document search, knowledge extraction, semantic queries

**Available Tools**:
- `scan_folder` - Index documents from folders
- `semantic_search` - Find relevant documents
- `build_knowledge_graph` - Extract entities and relationships
- `rag_query` - Unified semantic search
- `summarize_knowledge` - Synthesize information

---

## Tool Discovery & Execution

### Automatic Tool Discovery

The proxy automatically discovers tools from all configured MCP servers at startup:

1. **Server Connection**: MCP clients connect to each server via stdio/SSE/WebSocket
2. **Tool Enumeration**: Request `tools/list` from each server
3. **Registry Population**: Tools are registered in the central registry
4. **Capability Analysis**: Tool descriptions and schemas are analyzed
5. **Ready State**: Tools are available for reasoning and execution

### Manual Tool Discovery

```python
# Check available tools
from src.infrastructure.mcp.registry import mcp_registry

# List all tools
tools = await mcp_registry.list_all_tools()
for tool in tools:
    print(f"{tool['name']}: {tool['description']}")

# Get specific tool
tool = await mcp_registry.get_tool("find_epic")
print(tool['inputSchema'])
```

### Tool Execution Flow

```
User Query
    ↓
Intent Analysis (LLM)
    ↓
Tool Selection (MCPToolSelector)
    ↓
Tool Execution (MCP Client)
    ↓
Result Processing
    ↓
Context Enrichment
    ↓
Final Response
```

---

## Intelligent Orchestration

### Multi-Tool Coordination

The proxy can orchestrate multiple tools in a single request:

**Example: Cross-Platform Analysis**

```
User: "What work was done on the Authentication epic last week?"

Orchestration:
1. YouTrack: find_epic("Authentication") → Epic ID
2. YouTrack: analyze_epic_progress(epic_id, days=7) → Tasks completed
3. GitLab: link_commits_to_tasks(task_ids) → Related commits
4. GitLab: analyze_commit_messages(commits) → Commit analysis
5. Synthesis: Combine data into comprehensive report
```

### Tool Selection Intelligence

The `MCPToolSelector` uses LLM-powered reasoning to select optimal tools:

```python
# Tool selection considers:
- User intent and query semantics
- Tool capabilities and descriptions
- Historical success rates
- Tool dependencies and ordering
- Context and conversation history
```

### Reasoning Workflows

Custom reasoning workflows can be configured:

```python
# workflows/custom/reasoning_callback.py
async def reasoning_workflow(context):
    # Custom MCP tool orchestration
    tools = await discover_relevant_tools(context)
    results = await execute_tools_parallel(tools)
    enriched = await enhance_context(results)
    return enriched
```

---

## Performance & Optimization

### Caching Strategies

1. **Tool Result Caching**:
```yaml
mcp:
  cache:
    enabled: true
    ttl: 3600                    # 1 hour cache
    strategy: lru                # LRU eviction
```

2. **Connection Pooling**:
- Persistent MCP server connections
- Connection reuse across requests
- Auto-reconnect on failure

3. **Parallel Execution**:
```python
# Execute independent tools in parallel
results = await asyncio.gather(
    mcp_client.call_tool("find_epic", args1),
    mcp_client.call_tool("get_commits", args2),
    mcp_client.call_tool("semantic_search", args3)
)
```

### Performance Metrics

Monitor MCP performance:

```python
# Enable metrics
mcp:
  metrics:
    enabled: true
    export_interval: 60

# Metrics tracked:
- Tool execution time
- Success/failure rates
- Cache hit rates
- Server health status
```

---

## Security

### Tool Sandboxing

MCP tools run in isolated processes:

```yaml
mcp:
  security:
    sandboxing:
      enabled: true
      resource_limits:
        max_memory: 512M
        max_cpu: 50%
        max_execution_time: 120s
```

### Permission Management

Control tool access:

```yaml
mcp:
  permissions:
    youtrack:
      allowed_operations: [read]
      rate_limit: 100/minute

    gitlab:
      allowed_operations: [read, write]
      rate_limit: 50/minute
```

### Audit Logging

All MCP operations are logged:

```python
# Audit log format
{
  "timestamp": "2025-10-13T12:00:00Z",
  "server": "youtrack",
  "tool": "find_epic",
  "user": "user@example.com",
  "args": {...},
  "result": "success",
  "duration_ms": 245
}
```

---

## Monitoring & Debugging

### Health Checks

```bash
# Check MCP server health
make check-mcp

# Output:
# ✅ YouTrack Server: Connected (3 tools available)
# ✅ GitLab Server: Connected (7 tools available)
# ❌ RAG Server: Disconnected (reconnecting...)
```

### Debug Mode

Enable detailed MCP logging:

```bash
DEBUG=true python main.py
```

Debug output includes:
- Tool discovery process
- Tool selection reasoning
- Execution traces
- Error details

### Common Issues

**Problem**: MCP server won't connect

```bash
# Check server process
ps aux | grep mcps

# Test server directly
python -m mcps.youtrack.server

# Check logs
tail -f logs/mcp_youtrack.log
```

**Problem**: Tool execution timeout

```yaml
# Increase timeout
mcp:
  execution:
    timeout: 300  # 5 minutes
```

**Problem**: High latency

```bash
# Enable caching
mcp:
  cache:
    enabled: true

# Use parallel execution
mcp:
  execution:
    parallel_limit: 10
```

---

## Best Practices

### 1. Server Configuration

✅ **DO**:
- Use environment variables for secrets
- Set appropriate timeouts
- Enable caching for read-heavy operations
- Configure rate limits

❌ **DON'T**:
- Hardcode credentials in config.yaml
- Use overly long timeouts
- Disable security features
- Skip health monitoring

### 2. Tool Design

When creating custom MCP servers:

✅ **DO**:
- Provide clear tool descriptions
- Use strict JSON schemas for inputs
- Return structured, consistent data
- Handle errors gracefully
- Implement idempotent operations

❌ **DON'T**:
- Use vague tool names or descriptions
- Return unstructured text
- Throw exceptions without context
- Perform destructive operations without confirmation

### 3. Performance

✅ **DO**:
- Use parallel execution for independent tools
- Cache frequently accessed data
- Implement pagination for large results
- Monitor and optimize slow tools

❌ **DON'T**:
- Execute tools sequentially when parallel is possible
- Fetch all data without pagination
- Ignore performance metrics
- Block on long-running operations

### 4. Error Handling

✅ **DO**:
- Implement retry logic with exponential backoff
- Provide fallback tools
- Log errors with context
- Return user-friendly error messages

❌ **DON'T**:
- Fail silently
- Retry indefinitely
- Expose internal error details to users
- Skip error logging

### 5. Testing

✅ **DO**:
- Test MCP servers independently
- Write integration tests
- Mock external dependencies
- Test error scenarios

❌ **DON'T**:
- Test only happy paths
- Skip integration testing
- Rely on production systems for tests

---

## Advanced Topics

### Custom MCP Server Development

See [MCP Server Development Guide](MCP_SERVER_DEVELOPMENT.md) for detailed instructions on creating custom servers.

### Workflow Customization

See [Reasoning Workflows Guide](../workflows/README.md) for customizing the reasoning pipeline.

### Troubleshooting

See [Troubleshooting Guide](TROUBLESHOOTING.md) for common issues and solutions.

---

## Resources

- **MCP Protocol Specification**: https://modelcontextprotocol.io/
- **Google ADK Documentation**: https://cloud.google.com/vertex-ai/docs/agent-builder
- **Project Repository**: https://github.com/your-org/adk-llm-proxy
- **Issue Tracker**: https://github.com/your-org/adk-llm-proxy/issues

---

## Support

For questions or issues:

1. Check the [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Review [GitHub Issues](https://github.com/your-org/adk-llm-proxy/issues)
3. Open a new issue with:
   - MCP server configuration
   - Error logs
   - Steps to reproduce
   - Expected vs actual behavior

---

**Last Updated**: October 2025
**Version**: 1.0.0
**Status**: Production Ready
