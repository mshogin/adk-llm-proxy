#!/usr/bin/env python3

import asyncio
import logging
from typing import Any, Dict, List, Optional

from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent, Resource, Prompt

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Server("template-mcp-server")

@app.list_tools()
async def list_tools() -> List[Tool]:
    """List available tools for this MCP server."""
    return [
        Tool(
            name="echo",
            description="Echo back the input text",
            inputSchema={
                "type": "object",
                "properties": {
                    "text": {
                        "type": "string",
                        "description": "Text to echo back"
                    }
                },
                "required": ["text"]
            }
        )
    ]

@app.call_tool()
async def call_tool(name: str, arguments: Dict[str, Any]) -> List[TextContent]:
    """Handle tool calls."""
    if name == "echo":
        text = arguments.get("text", "")
        logger.info(f"Echo tool called with text: {text}")
        return [TextContent(type="text", text=f"Echo: {text}")]

    raise ValueError(f"Unknown tool: {name}")

@app.list_resources()
async def list_resources() -> List[Resource]:
    """List available resources for this MCP server."""
    return []

@app.read_resource()
async def read_resource(uri: str) -> str:
    """Read a resource."""
    raise ValueError(f"Resource not found: {uri}")

@app.list_prompts()
async def list_prompts() -> List[Prompt]:
    """List available prompts for this MCP server."""
    return []

@app.get_prompt()
async def get_prompt(name: str, arguments: Optional[Dict[str, str]] = None) -> str:
    """Get a prompt."""
    raise ValueError(f"Prompt not found: {name}")

async def main():
    """Main entry point for the MCP server."""
    async with stdio_server() as (read_stream, write_stream):
        await app.run(
            read_stream,
            write_stream,
            app.create_initialization_options()
        )

if __name__ == "__main__":
    asyncio.run(main())