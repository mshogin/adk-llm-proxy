"""
MCP (Model Context Protocol) infrastructure package.

This package provides comprehensive MCP integration including client connections,
server registry, tool discovery, unified tool access, health monitoring,
and advanced capability introspection.
"""

from .client import MCPClient, MCPServerConfig as ClientConfig, MCPTransportType
from .registry import MCPServerRegistry, MCPServerStatus, MCPServerInfo, mcp_registry
from .discovery import MCPToolDiscovery, MCPToolInfo, MCPResourceInfo, MCPPromptInfo, ToolAvailabilityStatus
from .tool_registry import MCPUnifiedToolRegistry, ToolExecutionStrategy, ToolExecutionResult
from .introspection import (
    MCPAvailabilityTracker, MCPCapabilityIntrospector,
    ToolIntrospectionResult, ToolCompatibilityInfo,
    ToolCategory, SchemaComplexity
)
from .server_base import MCPServerBase, mcp_tool, mcp_resource, mcp_prompt

__all__ = [
    # Client
    "MCPClient",
    "ClientConfig",
    "MCPTransportType",

    # Registry
    "MCPServerRegistry",
    "MCPServerStatus",
    "MCPServerInfo",
    "mcp_registry",

    # Discovery
    "MCPToolDiscovery",
    "MCPToolInfo",
    "MCPResourceInfo",
    "MCPPromptInfo",
    "ToolAvailabilityStatus",

    # Unified tool registry
    "MCPUnifiedToolRegistry",
    "ToolExecutionStrategy",
    "ToolExecutionResult",

    # Introspection and availability tracking
    "MCPAvailabilityTracker",
    "MCPCapabilityIntrospector",
    "ToolIntrospectionResult",
    "ToolCompatibilityInfo",
    "ToolCategory",
    "SchemaComplexity",

    # Server base (for MCP server implementations)
    "MCPServerBase",
    "mcp_tool",
    "mcp_resource",
    "mcp_prompt"
]