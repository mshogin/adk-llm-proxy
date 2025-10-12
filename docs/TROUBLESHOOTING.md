# MCP Integration Troubleshooting Guide

## Overview

This guide helps you diagnose and resolve common issues with MCP server integration in the ADK LLM Proxy.

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Connection Issues](#connection-issues)
3. [Tool Discovery Problems](#tool-discovery-problems)
4. [Execution Errors](#execution-errors)
5. [Performance Issues](#performance-issues)
6. [Configuration Problems](#configuration-problems)
7. [Authentication Errors](#authentication-errors)
8. [Logging & Debugging](#logging--debugging)

---

## Quick Diagnostics

### Health Check Command

```bash
# Check all MCP servers
make check-mcp

# Expected output:
# ✅ YouTrack Server: Connected (7 tools available)
# ✅ GitLab Server: Connected (7 tools available)
# ✅ RAG Server: Connected (5 tools available)
```

### Debug MCP Connection

```bash
# Run connection debug script
python scripts/debug_mcp_connection.py

# This shows:
# - Server connection status
# - Available tools
# - Tool schemas
# - Connection errors
```

### Check Server Logs

```bash
# View recent logs
tail -f logs/mcp_*.log

# Check for errors
grep -i error logs/mcp_*.log

# Check for warnings
grep -i warning logs/mcp_*.log
```

---

## Connection Issues

### Problem: MCP Server Won't Connect

**Symptom:**
```
Error: Failed to connect to MCP server 'youtrack'
Connection timeout after 30 seconds
```

**Diagnosis:**

```bash
# 1. Check if server process is running
ps aux | grep mcps

# 2. Try starting server manually
python -m mcps.youtrack.server

# 3. Check for port conflicts
lsof -i :8000
```

**Solutions:**

1. **Check Configuration:**
```yaml
# config.yaml - Verify command and args
mcp:
  servers:
    youtrack:
      command: python              # Correct Python executable
      args:
        - -m
        - mcps.youtrack.server    # Correct module path
```

2. **Verify Python Environment:**
```bash
# Check Python version
python --version  # Should be 3.9+

# Check if MCP package is installed
python -c "import mcp; print(mcp.__version__)"

# Install if missing
pip install mcp
```

3. **Check Environment Variables:**
```bash
# Verify required environment variables
echo $YOUTRACK_BASE_URL
echo $YOUTRACK_TOKEN

# Set if missing
export YOUTRACK_BASE_URL="https://your-instance.youtrack.cloud"
export YOUTRACK_TOKEN="your-api-token"
```

4. **Increase Timeout:**
```yaml
mcp:
  discovery:
    timeout: 60  # Increase from default 30 seconds
```

### Problem: Server Disconnects Randomly

**Symptom:**
```
Warning: MCP server 'gitlab' disconnected unexpectedly
Attempting reconnection...
```

**Solutions:**

1. **Check Server Logs:**
```bash
tail -f logs/mcp_gitlab.log
# Look for:
# - Out of memory errors
# - API rate limits
# - Network timeouts
```

2. **Increase Resource Limits:**
```yaml
mcp:
  servers:
    gitlab:
      resource_limits:
        max_memory: 1024M      # Increase memory
        max_execution_time: 300s  # Increase timeout
```

3. **Enable Auto-Reconnect:**
```yaml
mcp:
  discovery:
    auto_reconnect: true
    reconnect_interval: 30
```

---

## Tool Discovery Problems

### Problem: Tools Not Discovered

**Symptom:**
```
MCP server 'youtrack' connected but no tools available
Tools list is empty
```

**Diagnosis:**

```bash
# Test tool discovery directly
python scripts/debug_mcp_connection.py

# Output should show:
# Server: youtrack
# Tools: 7 available
#   - find_epic
#   - get_epic_details
#   ...
```

**Solutions:**

1. **Check Server Implementation:**
```python
# Verify tools are registered in server.py
def _register_tools(self):
    self.server.add_tool(
        Tool(name="find_epic", ...),
        self.find_epic
    )
```

2. **Verify Tool Schemas:**
```python
# Ensure inputSchema is valid JSON Schema
inputSchema={
    "type": "object",
    "properties": {
        "query": {"type": "string"}  # Valid schema
    },
    "required": ["query"]
}
```

3. **Check for Registration Errors:**
```bash
# Look for errors during tool registration
python -m mcps.youtrack.server 2>&1 | grep -i "error\|warning"
```

### Problem: Tools Discovered But Not Selectable

**Symptom:**
```
User query: "Find epic AUTH-123"
No suitable tools found for this query
```

**Solutions:**

1. **Improve Tool Descriptions:**
```python
# Bad: Vague description
Tool(
    name="find_epic",
    description="Find epic"  # Too vague!
)

# Good: Clear, descriptive
Tool(
    name="find_epic",
    description="Search for YouTrack epics by name, ID, or tag. "
                "Returns epic details including status, tasks, and progress."
)
```

2. **Check Tool Selector Logic:**
```bash
# Enable debug logging
DEBUG=true python main.py

# Check tool selection reasoning in logs
grep "Tool selection" logs/mcp_selector.log
```

---

## Execution Errors

### Problem: Tool Execution Timeout

**Symptom:**
```
Error: Tool execution timeout after 120 seconds
Tool: analyze_epic_progress
```

**Solutions:**

1. **Increase Execution Timeout:**
```yaml
mcp:
  execution:
    timeout: 300  # 5 minutes for slow operations
```

2. **Optimize Slow Tools:**
```python
# Add pagination for large results
async def list_tasks(self, epic_id: str, limit: int = 50):
    """List tasks with pagination to avoid timeouts."""
    tasks = []
    offset = 0

    while len(tasks) < limit:
        batch = await self.client.get_tasks(
            epic_id,
            limit=50,
            offset=offset
        )
        if not batch:
            break
        tasks.extend(batch)
        offset += 50

    return tasks[:limit]
```

3. **Implement Caching:**
```python
# Cache expensive operations
from functools import lru_cache
from datetime import datetime, timedelta

class MyServer:
    def __init__(self):
        self.cache = {}
        self.cache_ttl = timedelta(minutes=10)

    async def expensive_operation(self, key: str):
        # Check cache
        if key in self.cache:
            data, timestamp = self.cache[key]
            if datetime.now() - timestamp < self.cache_ttl:
                return data

        # Fetch and cache
        data = await self._fetch_data(key)
        self.cache[key] = (data, datetime.now())
        return data
```

### Problem: Tool Returns Error

**Symptom:**
```
Tool 'find_epic' returned error:
"API error: 401 Unauthorized"
```

**Solutions:**

1. **Verify API Credentials:**
```bash
# Test API access directly
curl -H "Authorization: Bearer $YOUTRACK_TOKEN" \
     "https://your-instance.youtrack.cloud/api/issues"

# Should return JSON, not 401
```

2. **Check Token Expiration:**
```python
# Add token validation
async def validate_token(self):
    """Check if API token is still valid."""
    try:
        response = await self.client.get("/api/user")
        return response.status_code == 200
    except Exception as e:
        logger.error(f"Token validation failed: {e}")
        return False
```

3. **Improve Error Handling:**
```python
# Good error handling
async def find_epic(self, query: str):
    try:
        result = await self.api_call(query)
        return result
    except httpx.HTTPStatusError as e:
        if e.response.status_code == 401:
            return [TextContent(
                type="text",
                text="Authentication error. Please check YOUTRACK_TOKEN."
            )]
        elif e.response.status_code == 404:
            return [TextContent(
                type="text",
                text=f"Epic not found: {query}"
            )]
        else:
            return [TextContent(
                type="text",
                text=f"API error: {e.response.status_code}"
            )]
```

---

## Performance Issues

### Problem: Slow Response Times

**Symptom:**
```
Request took 45 seconds to complete
Multiple tool executions taking too long
```

**Solutions:**

1. **Enable Parallel Execution:**
```yaml
mcp:
  execution:
    parallel_limit: 10  # Execute up to 10 tools concurrently
```

2. **Implement Connection Pooling:**
```python
# Reuse HTTP connections
import httpx

class MyServer:
    def __init__(self):
        self.client = httpx.AsyncClient(
            timeout=30.0,
            limits=httpx.Limits(
                max_connections=100,
                max_keepalive_connections=20
            )
        )

    async def cleanup(self):
        await self.client.aclose()
```

3. **Enable Caching:**
```yaml
mcp:
  cache:
    enabled: true
    ttl: 3600      # 1 hour
    max_size: 1000
```

4. **Profile Slow Operations:**
```python
import time

async def my_tool(self, arg: str):
    start = time.time()

    result = await self.slow_operation(arg)

    duration = time.time() - start
    logger.info(f"Operation took {duration:.2f}s")

    return result
```

### Problem: High Memory Usage

**Symptom:**
```
MCP server consuming > 2GB memory
Out of memory errors
```

**Solutions:**

1. **Implement Streaming:**
```python
# Stream large results instead of loading all at once
async def get_large_dataset(self):
    async for chunk in self.fetch_data_stream():
        yield chunk  # Stream chunks
```

2. **Limit Result Sizes:**
```python
# Add limits to prevent huge results
async def search(self, query: str, limit: int = 100):
    """Search with maximum result limit."""
    if limit > 1000:
        limit = 1000  # Cap at reasonable size

    return await self.api.search(query, limit=limit)
```

3. **Clear Caches Periodically:**
```python
# Implement cache eviction
class MyServer:
    def __init__(self):
        self.cache = {}
        self.max_cache_size = 1000

    def cleanup_cache(self):
        if len(self.cache) > self.max_cache_size:
            # Remove oldest entries
            sorted_items = sorted(
                self.cache.items(),
                key=lambda x: x[1][1]  # Sort by timestamp
            )
            self.cache = dict(sorted_items[-self.max_cache_size:])
```

---

## Configuration Problems

### Problem: Config File Not Found

**Symptom:**
```
Error: Configuration file 'config.yaml' not found
```

**Solutions:**

1. **Check File Location:**
```bash
# Verify config.yaml exists
ls -la config.yaml

# Check working directory
pwd  # Should be project root
```

2. **Use Default Config:**
```bash
# Copy from example
cp config.example.yaml config.yaml

# Edit with your settings
nano config.yaml
```

### Problem: Invalid YAML Syntax

**Symptom:**
```
Error: YAML parsing failed at line 15
Unexpected token ':'
```

**Solutions:**

1. **Validate YAML Syntax:**
```bash
# Use YAML validator
python -c "import yaml; yaml.safe_load(open('config.yaml'))"

# Or use online validator
# https://www.yamllint.com/
```

2. **Common YAML Mistakes:**
```yaml
# ❌ Wrong: Missing space after colon
servers:youtrack:

# ✅ Correct: Space after colon
servers:
  youtrack:

# ❌ Wrong: Mixed tabs and spaces
	servers:
  youtrack:

# ✅ Correct: Consistent indentation (2 or 4 spaces)
  servers:
    youtrack:
```

---

## Authentication Errors

### Problem: API Authentication Failed

**Symptom:**
```
401 Unauthorized: Invalid or expired token
```

**Solutions:**

1. **Generate New Token:**
```bash
# YouTrack: Profile → Authentication → New Token
# GitLab: Settings → Access Tokens → Create

# Set environment variable (example)
export YOUTRACK_TOKEN="perm:YOUR-TOKEN-HERE"
```

2. **Check Token Permissions:**
```
Required permissions for YouTrack:
- Read Issues
- Read Projects
- Read Users

Required permissions for GitLab:
- read_api
- read_repository
```

3. **Test Token Directly:**
```bash
# Test YouTrack token
curl -H "Authorization: Bearer $YOUTRACK_TOKEN" \
     "https://your-instance.youtrack.cloud/api/admin/projects" \
     | jq .

# Test GitLab token
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
     "https://gitlab.com/api/v4/projects" \
     | jq .
```

---

## Logging & Debugging

### Enable Debug Logging

```bash
# Method 1: Environment variable
DEBUG=true python main.py

# Method 2: Config file
# config.yaml
logging:
  level: DEBUG
  mcp_debug: true
```

### View Structured Logs

```bash
# Follow MCP logs
tail -f logs/mcp_*.log

# Filter by server
grep "youtrack" logs/mcp_*.log

# Filter by error level
grep -E "ERROR|CRITICAL" logs/mcp_*.log

# View specific tool executions
grep "Tool execution" logs/mcp_*.log
```

### Debug Tool Selection

```bash
# Enable tool selection debugging
python scripts/debug_mcp_tool_selection.py

# Output shows:
# - User query analysis
# - Tool matching process
# - Selected tools and reasoning
# - Execution plan
```

### Test Individual Tools

```python
# Create test script
from src.infrastructure.mcp.client import MCPClient

async def test_tool():
    client = MCPClient("youtrack", ...)
    await client.connect()

    result = await client.call_tool("find_epic", {"query": "AUTH-123"})
    print(result)

    await client.disconnect()

# Run test
python -c "import asyncio; asyncio.run(test_tool())"
```

---

## Common Error Messages

### Error: `ModuleNotFoundError: No module named 'mcp'`

**Solution:**
```bash
pip install mcp
```

### Error: `asyncio.TimeoutError: Task took longer than 30 seconds`

**Solution:**
```yaml
# Increase timeout in config.yaml
mcp:
  execution:
    timeout: 120
```

### Error: `JSONDecodeError: Expecting value: line 1 column 1`

**Solution:**
```python
# Server is returning non-JSON response
# Check server implementation:
async def my_tool(self):
    # ❌ Wrong: Plain string
    return "result"

    # ✅ Correct: TextContent
    return [TextContent(type="text", text="result")]
```

### Error: `BrokenPipeError: [Errno 32] Broken pipe`

**Solution:**
```python
# Server process died unexpectedly
# Check server logs for crash reason:
tail -f logs/mcp_server.log

# Common causes:
# - Unhandled exception
# - Out of memory
# - Invalid tool implementation
```

---

## Getting Help

### 1. Check Documentation

- [MCP Integration Guide](MCP_INTEGRATION.md)
- [Server Development Guide](MCP_SERVER_DEVELOPMENT.md)
- [GitHub Issues](https://github.com/your-org/adk-llm-proxy/issues)

### 2. Collect Debug Information

When reporting issues, include:

```bash
# 1. Version information
python --version
pip show mcp

# 2. Configuration (sanitize tokens!)
cat config.yaml | grep -v "TOKEN\|PASSWORD"

# 3. Error logs
tail -100 logs/mcp_*.log

# 4. Health check output
make check-mcp

# 5. Debug script output
python scripts/debug_mcp_connection.py
```

### 3. Open GitHub Issue

Include:
- Clear problem description
- Steps to reproduce
- Expected vs actual behavior
- Debug information (above)
- MCP server code (if custom server)

---

## Preventive Measures

### Regular Maintenance

```bash
# 1. Check server health daily
make check-mcp

# 2. Monitor log sizes
du -sh logs/

# 3. Rotate logs regularly
find logs/ -name "*.log" -mtime +7 -delete

# 4. Update dependencies
pip install --upgrade mcp

# 5. Test critical tools
python scripts/verify_mcp_servers.py
```

### Monitoring Setup

```yaml
# Enable metrics collection
mcp:
  metrics:
    enabled: true
    export_interval: 60

  alerts:
    connection_failures: 3
    timeout_threshold: 30
    error_rate_threshold: 0.1
```

---

**Last Updated**: October 2025
**Version**: 1.0.0
