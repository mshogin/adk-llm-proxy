# MCP Server Development Guide

## Overview

This guide covers everything you need to know to develop custom MCP servers for the ADK LLM Proxy. Learn how to create, test, deploy, and maintain production-ready MCP servers.

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Server Architecture](#server-architecture)
4. [Creating Your First Server](#creating-your-first-server)
5. [Tool Development](#tool-development)
6. [Resources & Prompts](#resources--prompts)
7. [Testing](#testing)
8. [Deployment](#deployment)
9. [Best Practices](#best-practices)
10. [Advanced Topics](#advanced-topics)

---

## Introduction

### What is an MCP Server?

An MCP (Model Context Protocol) server is a process that exposes **tools**, **resources**, and **prompts** to LLM applications. Servers run as independent processes and communicate via stdio, SSE, or WebSocket.

### When to Create a Custom Server?

Create an MCP server when you need to:
- Integrate with external APIs (Jira, GitHub, Slack, etc.)
- Access databases or file systems
- Perform computations or data processing
- Provide domain-specific knowledge or context
- Orchestrate complex workflows

### MCP Server vs. API Client

| MCP Server | Traditional API Client |
|-----------|----------------------|
| LLM-discoverable tools | Fixed code integration |
| Self-describing capabilities | Manual documentation |
| Dynamic tool selection | Hardcoded logic |
| Standardized protocol | Custom protocols |
| Isolated process | In-process library |

---

## Getting Started

### Prerequisites

```bash
# Install MCP Python SDK
pip install mcp

# Install development tools
pip install pytest pytest-asyncio black flake8 mypy
```

### Project Structure

```
mcps/your-server/
├── __init__.py
├── server.py              # Main server implementation
├── client.py              # API client for external service
├── requirements.txt       # Python dependencies
├── README.md              # Server documentation
├── config.example.yaml    # Example configuration
└── test_server.py         # Server tests
```

### Using the Template

Generate a new server from the template:

```bash
# Copy template
cp -r mcps/template mcps/your-server

# Update server.py with your implementation
cd mcps/your-server
```

---

## Server Architecture

### MCP Server Components

```
┌─────────────────────────────────────────┐
│         MCP Server Process              │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │  MCP Protocol Handler              │ │
│  │  - Tool registry                   │ │
│  │  - Resource provider               │ │
│  │  - Prompt templates                │ │
│  └───────────────────────────────────┘ │
│                 ↓                       │
│  ┌───────────────────────────────────┐ │
│  │  Business Logic Layer              │ │
│  │  - Tool implementations            │ │
│  │  - Data validation                 │ │
│  │  - Error handling                  │ │
│  └───────────────────────────────────┘ │
│                 ↓                       │
│  ┌───────────────────────────────────┐ │
│  │  External Service Client           │ │
│  │  - API authentication              │ │
│  │  - Request/response handling       │ │
│  │  - Rate limiting                   │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

### Communication Flow

1. **Startup**: Proxy launches server process via stdio
2. **Handshake**: Server announces capabilities (tools, resources, prompts)
3. **Discovery**: Proxy queries available tools
4. **Execution**: Proxy calls tools with arguments
5. **Response**: Server returns structured results
6. **Shutdown**: Graceful cleanup on SIGTERM

---

## Creating Your First Server

### Step 1: Define Your Server Class

```python
# mcps/weather/server.py
from mcp.server import Server
from mcp.types import Tool, TextContent
import httpx

class WeatherMCPServer:
    """MCP server for weather data from OpenWeatherMap API."""

    def __init__(self, api_key: str):
        self.api_key = api_key
        self.base_url = "https://api.openweathermap.org/data/2.5"
        self.server = Server("weather-server")

        # Register tools
        self._register_tools()

    def _register_tools(self):
        """Register all available tools."""
        self.server.add_tool(
            Tool(
                name="get_weather",
                description="Get current weather for a city",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "city": {
                            "type": "string",
                            "description": "City name (e.g., 'London', 'New York')"
                        },
                        "units": {
                            "type": "string",
                            "enum": ["metric", "imperial"],
                            "description": "Temperature units",
                            "default": "metric"
                        }
                    },
                    "required": ["city"]
                }
            ),
            self.get_weather
        )

    async def get_weather(self, city: str, units: str = "metric"):
        """Get current weather for a city."""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/weather",
                    params={
                        "q": city,
                        "appid": self.api_key,
                        "units": units
                    }
                )
                response.raise_for_status()
                data = response.json()

                # Format response
                temp = data["main"]["temp"]
                description = data["weather"][0]["description"]
                humidity = data["main"]["humidity"]

                result = f"""Weather in {city}:
Temperature: {temp}°{'C' if units == 'metric' else 'F'}
Conditions: {description}
Humidity: {humidity}%"""

                return [TextContent(type="text", text=result)]

        except httpx.HTTPError as e:
            return [TextContent(
                type="text",
                text=f"Error fetching weather: {str(e)}"
            )]

    async def run(self):
        """Start the MCP server."""
        async with self.server.stdio_server() as streams:
            await self.server.run(
                streams[0],
                streams[1],
                self.server.create_initialization_options()
            )


# Entry point
if __name__ == "__main__":
    import os
    import asyncio

    api_key = os.getenv("OPENWEATHER_API_KEY")
    if not api_key:
        raise ValueError("OPENWEATHER_API_KEY environment variable required")

    server = WeatherMCPServer(api_key)
    asyncio.run(server.run())
```

### Step 2: Create requirements.txt

```txt
mcp>=1.0.0
httpx>=0.24.0
pydantic>=2.0.0
```

### Step 3: Configure in config.yaml

```yaml
mcp:
  servers:
    weather:
      command: python
      args:
        - -m
        - mcps.weather.server
      env:
        OPENWEATHER_API_KEY: "your-api-key"
```

### Step 4: Test Your Server

```bash
# Install dependencies
cd mcps/weather
pip install -r requirements.txt

# Run server standalone
OPENWEATHER_API_KEY="your-key" python server.py

# Test via proxy
python scripts/debug_mcp_connection.py
```

---

## Tool Development

### Tool Design Principles

1. **Single Responsibility**: Each tool should do one thing well
2. **Clear Interface**: Use descriptive names and parameters
3. **Robust Validation**: Validate inputs, handle errors gracefully
4. **Consistent Output**: Return structured, predictable data
5. **Idempotent**: Same inputs should produce same outputs

### Tool Naming Conventions

```python
# Good names (verb + noun)
get_weather()
create_issue()
search_documents()
analyze_sentiment()
list_projects()

# Bad names (unclear purpose)
weather()
issue()
search()
analyze()
data()
```

### Input Schema Best Practices

```python
{
    "type": "object",
    "properties": {
        # Required parameters first
        "city": {
            "type": "string",
            "description": "City name (e.g., 'London', 'Paris')",  # Examples!
            "minLength": 1
        },

        # Optional parameters with defaults
        "units": {
            "type": "string",
            "enum": ["metric", "imperial"],                        # Use enums
            "description": "Temperature units (metric or imperial)",
            "default": "metric"
        },

        # Complex types
        "date_range": {
            "type": "object",
            "properties": {
                "start": {"type": "string", "format": "date"},
                "end": {"type": "string", "format": "date"}
            }
        }
    },
    "required": ["city"],                                          # Explicit required fields
    "additionalProperties": false                                  # Strict validation
}
```

### Error Handling

```python
async def get_weather(self, city: str):
    """Get weather with proper error handling."""
    try:
        # Validate inputs
        if not city or len(city) < 2:
            return [TextContent(
                type="text",
                text="Error: City name must be at least 2 characters"
            )]

        # Make API call
        response = await self.client.get(f"/weather?q={city}")
        response.raise_for_status()

        # Process result
        data = response.json()
        return [TextContent(type="text", text=format_weather(data))]

    except httpx.HTTPStatusError as e:
        if e.response.status_code == 404:
            return [TextContent(
                type="text",
                text=f"Error: City '{city}' not found"
            )]
        else:
            return [TextContent(
                type="text",
                text=f"API error: {e.response.status_code}"
            )]

    except httpx.NetworkError:
        return [TextContent(
            type="text",
            text="Error: Network connection failed. Please try again."
        )]

    except Exception as e:
        # Log unexpected errors
        logging.error(f"Unexpected error in get_weather: {e}")
        return [TextContent(
            type="text",
            text="An unexpected error occurred. Please contact support."
        )]
```

### Response Formatting

```python
# Good: Structured, readable output
def format_issue(issue_data):
    return f"""Issue: {issue_data['key']}
Title: {issue_data['title']}
Status: {issue_data['status']}
Assignee: {issue_data['assignee']}
Created: {issue_data['created_at']}
Updated: {issue_data['updated_at']}

Description:
{issue_data['description']}
"""

# Better: JSON for complex data
def format_issue_json(issue_data):
    return [{
        "type": "text",
        "text": json.dumps({
            "key": issue_data['key'],
            "title": issue_data['title'],
            "status": issue_data['status'],
            "assignee": issue_data['assignee'],
            "created_at": issue_data['created_at'],
            "updated_at": issue_data['updated_at'],
            "description": issue_data['description']
        }, indent=2)
    }]
```

---

## Resources & Prompts

### Resources

Resources provide context and data to LLMs:

```python
# Register a resource
self.server.add_resource(
    Resource(
        uri="weather://cities/list",
        name="Supported Cities",
        description="List of cities with weather data",
        mimeType="application/json"
    ),
    self.get_cities_list
)

async def get_cities_list(self):
    """Return list of supported cities."""
    cities = ["London", "Paris", "New York", "Tokyo", "Sydney"]
    return [TextContent(
        type="text",
        text=json.dumps(cities)
    )]
```

### Prompts

Prompts provide reusable templates:

```python
# Register a prompt template
self.server.add_prompt(
    Prompt(
        name="weather_report",
        description="Generate a weather report for a city",
        arguments=[
            PromptArgument(
                name="city",
                description="City name",
                required=True
            ),
            PromptArgument(
                name="format",
                description="Report format (brief, detailed)",
                required=False
            )
        ]
    ),
    self.weather_report_prompt
)

async def weather_report_prompt(self, city: str, format: str = "brief"):
    """Generate weather report prompt."""
    weather = await self.get_weather(city)

    if format == "detailed":
        prompt = f"""Generate a detailed weather report for {city}.
Include current conditions, forecast, and recommendations.

Current Data:
{weather[0].text}
"""
    else:
        prompt = f"""Summarize the weather in {city} in one sentence.

Data: {weather[0].text}
"""

    return [TextContent(type="text", text=prompt)]
```

---

## Testing

### Unit Tests

```python
# test_weather_server.py
import pytest
from unittest.mock import AsyncMock, patch
from mcps.weather.server import WeatherMCPServer

@pytest.fixture
async def weather_server():
    """Create a test server instance."""
    server = WeatherMCPServer(api_key="test-key")
    return server

@pytest.mark.asyncio
async def test_get_weather_success(weather_server):
    """Test successful weather retrieval."""
    with patch('httpx.AsyncClient.get') as mock_get:
        # Mock API response
        mock_response = AsyncMock()
        mock_response.json.return_value = {
            "main": {"temp": 20, "humidity": 60},
            "weather": [{"description": "clear sky"}]
        }
        mock_response.raise_for_status = AsyncMock()
        mock_get.return_value = mock_response

        # Call tool
        result = await weather_server.get_weather("London")

        # Verify result
        assert len(result) == 1
        assert "20" in result[0].text
        assert "clear sky" in result[0].text

@pytest.mark.asyncio
async def test_get_weather_city_not_found(weather_server):
    """Test handling of invalid city."""
    with patch('httpx.AsyncClient.get') as mock_get:
        mock_get.side_effect = httpx.HTTPStatusError(
            "Not found",
            request=AsyncMock(),
            response=AsyncMock(status_code=404)
        )

        result = await weather_server.get_weather("InvalidCity")

        assert "not found" in result[0].text.lower()
```

### Integration Tests

```python
# test_weather_integration.py
import pytest
import subprocess
import json

@pytest.mark.integration
async def test_server_startup():
    """Test that server starts correctly."""
    proc = subprocess.Popen(
        ["python", "-m", "mcps.weather.server"],
        env={"OPENWEATHER_API_KEY": "test-key"},
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )

    try:
        # Wait for startup
        await asyncio.sleep(2)
        assert proc.poll() is None, "Server died during startup"
    finally:
        proc.terminate()
        proc.wait()

@pytest.mark.integration
async def test_tool_discovery():
    """Test tool discovery via MCP protocol."""
    from src.infrastructure.mcp.client import MCPClient

    client = MCPClient("weather", command="python", args=["-m", "mcps.weather.server"])
    await client.connect()

    try:
        tools = await client.list_tools()
        assert len(tools) > 0
        assert any(t["name"] == "get_weather" for t in tools)
    finally:
        await client.disconnect()
```

### Running Tests

```bash
# Run all tests
pytest mcps/weather/

# Run with coverage
pytest --cov=mcps.weather --cov-report=html

# Run integration tests only
pytest -m integration

# Run specific test
pytest mcps/weather/test_server.py::test_get_weather_success
```

---

## Deployment

### Production Checklist

- [ ] Environment variables validated
- [ ] Error handling comprehensive
- [ ] Logging configured
- [ ] Rate limiting implemented
- [ ] Timeouts configured
- [ ] Health checks added
- [ ] Documentation complete
- [ ] Tests passing (>80% coverage)
- [ ] Security audit passed

### Configuration Management

```yaml
# Production config
mcp:
  servers:
    weather:
      command: python
      args:
        - -m
        - mcps.weather.server
      env:
        OPENWEATHER_API_KEY: "${OPENWEATHER_API_KEY}"  # From environment
        LOG_LEVEL: "INFO"
        TIMEOUT: "30"
        RATE_LIMIT: "100"
```

### Logging

```python
import logging
import sys

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stderr),
        logging.FileHandler('logs/weather_server.log')
    ]
)

logger = logging.getLogger("weather-mcp")

# Use in server
logger.info("Server starting...")
logger.debug(f"API call: {endpoint}")
logger.error(f"Error: {error_message}")
logger.warning(f"Rate limit approaching: {current_rate}")
```

### Health Monitoring

```python
class WeatherMCPServer:
    def __init__(self, api_key):
        self.api_key = api_key
        self.health = {
            "status": "initializing",
            "last_check": None,
            "error_count": 0,
            "request_count": 0
        }

    async def health_check(self):
        """Perform health check."""
        try:
            # Test API connectivity
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.base_url}/health",
                    timeout=5.0
                )

            self.health["status"] = "healthy"
            self.health["last_check"] = datetime.now()
            return True

        except Exception as e:
            self.health["status"] = "unhealthy"
            self.health["error_count"] += 1
            logger.error(f"Health check failed: {e}")
            return False
```

---

## Best Practices

### 1. Security

```python
# ✅ DO: Validate all inputs
def validate_city(city: str) -> bool:
    # Whitelist characters
    return bool(re.match(r'^[a-zA-Z\s-]+$', city))

# ✅ DO: Use environment variables for secrets
api_key = os.getenv("API_KEY")

# ❌ DON'T: Hardcode secrets
api_key = "sk-1234567890abcdef"  # NEVER DO THIS

# ✅ DO: Sanitize outputs
def sanitize_output(text: str) -> str:
    # Remove sensitive data
    return re.sub(r'\b\d{16}\b', '[REDACTED]', text)
```

### 2. Performance

```python
# ✅ DO: Use connection pooling
class WeatherServer:
    def __init__(self):
        self.client = httpx.AsyncClient(
            timeout=30.0,
            limits=httpx.Limits(max_connections=100)
        )

    async def cleanup(self):
        await self.client.aclose()

# ✅ DO: Implement caching
from functools import lru_cache
from datetime import datetime, timedelta

class WeatherServer:
    def __init__(self):
        self.cache = {}
        self.cache_ttl = timedelta(minutes=10)

    async def get_weather(self, city: str):
        # Check cache
        cache_key = f"weather:{city}"
        if cache_key in self.cache:
            data, timestamp = self.cache[cache_key]
            if datetime.now() - timestamp < self.cache_ttl:
                return data

        # Fetch fresh data
        data = await self._fetch_weather(city)
        self.cache[cache_key] = (data, datetime.now())
        return data
```

### 3. Error Recovery

```python
# ✅ DO: Implement retry logic
from tenacity import retry, stop_after_attempt, wait_exponential

@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=2, max=10)
)
async def fetch_with_retry(url: str):
    """Fetch with automatic retry."""
    async with httpx.AsyncClient() as client:
        response = await client.get(url)
        response.raise_for_status()
        return response.json()

# ✅ DO: Provide fallback mechanisms
async def get_weather(self, city: str):
    try:
        return await self.fetch_from_primary_api(city)
    except Exception as e:
        logger.warning(f"Primary API failed: {e}, trying fallback")
        try:
            return await self.fetch_from_fallback_api(city)
        except Exception:
            return self.get_cached_weather(city)
```

### 4. Documentation

```python
class WeatherMCPServer:
    """
    MCP server for OpenWeatherMap integration.

    Provides tools for:
    - Current weather conditions
    - Weather forecasts
    - Historical weather data

    Configuration:
        OPENWEATHER_API_KEY: API key from OpenWeatherMap
        CACHE_TTL: Cache duration in seconds (default: 600)

    Example:
        >>> server = WeatherMCPServer(api_key="your-key")
        >>> await server.run()
    """

    async def get_weather(self, city: str, units: str = "metric"):
        """
        Get current weather for a city.

        Args:
            city: City name (e.g., "London", "New York")
            units: Temperature units ("metric" or "imperial")

        Returns:
            List[TextContent]: Formatted weather information

        Raises:
            ValueError: If city name is invalid
            HTTPError: If API request fails

        Example:
            >>> result = await server.get_weather("London")
            >>> print(result[0].text)
            Weather in London:
            Temperature: 15°C
            ...
        """
```

---

## Advanced Topics

### Custom Transports

```python
# SSE Transport
async def run_sse(self, host: str = "localhost", port: int = 8080):
    """Run server with SSE transport."""
    app = create_sse_app(self.server)
    await asyncio.create_server(app, host, port)

# WebSocket Transport
async def run_websocket(self, host: str = "localhost", port: int = 8080):
    """Run server with WebSocket transport."""
    async with websockets.serve(self.handle_websocket, host, port):
        await asyncio.Future()  # Run forever
```

### Streaming Responses

```python
async def stream_large_dataset(self, query: str):
    """Stream large results incrementally."""
    async for chunk in self.fetch_data_stream(query):
        yield TextContent(
            type="text",
            text=json.dumps(chunk)
        )
```

### Multi-Tool Workflows

```python
# Complex workflow combining multiple tools
async def analyze_project(self, project_id: str):
    """Analyze project using multiple tools."""
    # Step 1: Get project details
    project = await self.get_project(project_id)

    # Step 2: Get related issues
    issues = await self.get_issues(project_id)

    # Step 3: Get recent commits
    commits = await self.get_commits(project_id)

    # Step 4: Analyze and combine results
    analysis = self.combine_analysis(project, issues, commits)

    return [TextContent(type="text", text=analysis)]
```

---

## Resources

- **MCP Specification**: https://modelcontextprotocol.io/
- **Python SDK**: https://github.com/modelcontextprotocol/python-sdk
- **Example Servers**: https://github.com/modelcontextprotocol/servers
- **Best Practices**: https://modelcontextprotocol.io/docs/best-practices

---

## Support

Need help?

1. Review the [MCP Integration Guide](MCP_INTEGRATION.md)
2. Check existing servers in `mcps/` for examples
3. Open an issue on GitHub with:
   - Server code
   - Error logs
   - Configuration
   - Steps to reproduce

---

**Last Updated**: October 2025
**Version**: 1.0.0
