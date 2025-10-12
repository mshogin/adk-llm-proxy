#!/usr/bin/env python3
"""
MCP Server Base Class
Provides base functionality for MCP server implementations.
"""

import asyncio
import json
import logging
from typing import Any, Callable, Dict, List, Optional
from functools import wraps

logger = logging.getLogger(__name__)


def mcp_tool(name: str = None, description: str = None, input_schema: Dict = None):
    """Decorator to mark a method as an MCP tool with metadata."""
    def decorator(func: Callable) -> Callable:
        func._mcp_tool = True
        func._mcp_tool_name = name or func.__name__
        func._mcp_tool_description = description or func.__doc__ or f"Tool: {func.__name__}"
        func._mcp_tool_input_schema = input_schema
        return func
    return decorator


def mcp_resource(uri: str = None, name: str = None, description: str = None, mime_type: str = "text/plain"):
    """Decorator to mark a method as an MCP resource with metadata."""
    def decorator(func: Callable) -> Callable:
        func._mcp_resource = True
        func._mcp_resource_uri = uri or f"resource://{func.__name__}"
        func._mcp_resource_name = name or func.__name__
        func._mcp_resource_description = description or func.__doc__ or f"Resource: {func.__name__}"
        func._mcp_resource_mime_type = mime_type
        return func
    return decorator


def mcp_prompt(name: str = None, description: str = None, arguments: List = None):
    """Decorator to mark a method as an MCP prompt with metadata."""
    def decorator(func: Callable) -> Callable:
        func._mcp_prompt = True
        func._mcp_prompt_name = name or func.__name__
        func._mcp_prompt_description = description or func.__doc__ or f"Prompt: {func.__name__}"
        func._mcp_prompt_arguments = arguments or []
        return func
    return decorator


class MCPServerBase:
    """Base class for MCP servers."""

    def __init__(self, name: str, version: str = "1.0.0"):
        """Initialize MCP server.

        Args:
            name: Server name
            version: Server version (default: "1.0.0")
        """
        self.name = name
        self.version = version
        self.logger = logging.getLogger(f"mcp.{name}")
        self._tools = {}
        self._resources = {}
        self._prompts = {}
        self._register_handlers()

    def _register_handlers(self):
        """Register all decorated methods as tools, resources, and prompts."""
        for attr_name in dir(self):
            if attr_name.startswith('_'):
                continue

            attr = getattr(self, attr_name)
            if callable(attr):
                if hasattr(attr, '_mcp_tool'):
                    self._tools[attr_name] = attr
                    self.logger.debug(f"Registered tool: {attr_name}")
                elif hasattr(attr, '_mcp_resource'):
                    self._resources[attr_name] = attr
                    self.logger.debug(f"Registered resource: {attr_name}")
                elif hasattr(attr, '_mcp_prompt'):
                    self._prompts[attr_name] = attr
                    self.logger.debug(f"Registered prompt: {attr_name}")

    async def handle_request(self, method: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """Handle an MCP request.

        Args:
            method: Method name
            params: Request parameters

        Returns:
            Response dictionary
        """
        try:
            if method == "initialize":
                return await self.initialize(params)
            elif method == "tools/list":
                return await self.list_tools()
            elif method == "tools/call":
                return await self.call_tool(params.get("name"), params.get("arguments", {}))
            elif method == "resources/list":
                return await self.list_resources()
            elif method == "resources/read":
                return await self.read_resource(params.get("uri"))
            elif method == "prompts/list":
                return await self.list_prompts()
            elif method == "prompts/get":
                return await self.get_prompt(params.get("name"), params.get("arguments", {}))
            else:
                return {"error": f"Unknown method: {method}"}
        except Exception as e:
            self.logger.error(f"Error handling request {method}: {e}")
            return {"error": str(e)}

    async def initialize(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Handle MCP initialize request.

        Args:
            params: Initialization parameters

        Returns:
            Server capabilities
        """
        return {
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "tools": {},
                "resources": {},
                "prompts": {}
            },
            "serverInfo": {
                "name": self.name,
                "version": self.version
            }
        }

    async def list_tools(self) -> Dict[str, Any]:
        """List available tools."""
        tools = []
        for name, handler in self._tools.items():
            tool_info = {
                "name": getattr(handler, '_mcp_tool_name', name),
                "description": getattr(handler, '_mcp_tool_description', handler.__doc__ or f"Tool: {name}"),
            }
            if hasattr(handler, '_mcp_tool_input_schema') and handler._mcp_tool_input_schema:
                tool_info["inputSchema"] = handler._mcp_tool_input_schema
            tools.append(tool_info)
        return {"tools": tools}

    async def call_tool(self, name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """Call a tool.

        Args:
            name: Tool name
            arguments: Tool arguments

        Returns:
            Tool result
        """
        if name not in self._tools:
            return {"error": f"Tool not found: {name}"}

        try:
            handler = self._tools[name]
            result = await handler(**arguments) if asyncio.iscoroutinefunction(handler) else handler(**arguments)
            return {"content": [{"type": "text", "text": json.dumps(result)}]}
        except Exception as e:
            self.logger.error(f"Error calling tool {name}: {e}")
            return {"error": str(e)}

    async def list_resources(self) -> Dict[str, Any]:
        """List available resources."""
        resources = []
        for name, handler in self._resources.items():
            resource_info = {
                "uri": getattr(handler, '_mcp_resource_uri', f"{self.name}://{name}"),
                "name": getattr(handler, '_mcp_resource_name', name),
                "description": getattr(handler, '_mcp_resource_description', handler.__doc__ or f"Resource: {name}"),
                "mimeType": getattr(handler, '_mcp_resource_mime_type', "text/plain"),
            }
            resources.append(resource_info)
        return {"resources": resources}

    async def read_resource(self, uri: str) -> Dict[str, Any]:
        """Read a resource.

        Args:
            uri: Resource URI

        Returns:
            Resource content
        """
        # Extract resource name from URI
        if not uri or '://' not in uri:
            return {"error": "Invalid resource URI"}

        resource_name = uri.split('://')[-1]
        if resource_name not in self._resources:
            return {"error": f"Resource not found: {resource_name}"}

        try:
            handler = self._resources[resource_name]
            result = await handler() if asyncio.iscoroutinefunction(handler) else handler()
            return {"contents": [{"uri": uri, "mimeType": "text/plain", "text": json.dumps(result)}]}
        except Exception as e:
            self.logger.error(f"Error reading resource {resource_name}: {e}")
            return {"error": str(e)}

    async def list_prompts(self) -> Dict[str, Any]:
        """List available prompts."""
        prompts = []
        for name, handler in self._prompts.items():
            prompt_info = {
                "name": getattr(handler, '_mcp_prompt_name', name),
                "description": getattr(handler, '_mcp_prompt_description', handler.__doc__ or f"Prompt: {name}"),
            }
            if hasattr(handler, '_mcp_prompt_arguments') and handler._mcp_prompt_arguments:
                prompt_info["arguments"] = handler._mcp_prompt_arguments
            prompts.append(prompt_info)
        return {"prompts": prompts}

    async def get_prompt(self, name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """Get a prompt.

        Args:
            name: Prompt name
            arguments: Prompt arguments

        Returns:
            Prompt messages
        """
        if name not in self._prompts:
            return {"error": f"Prompt not found: {name}"}

        try:
            handler = self._prompts[name]
            result = await handler(**arguments) if asyncio.iscoroutinefunction(handler) else handler(**arguments)
            return {"messages": result if isinstance(result, list) else [{"role": "user", "content": {"type": "text", "text": str(result)}}]}
        except Exception as e:
            self.logger.error(f"Error getting prompt {name}: {e}")
            return {"error": str(e)}

