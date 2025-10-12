"""
MCP Tool Discovery and Enumeration System.

This module provides comprehensive tool discovery capabilities for MCP servers,
including enumeration, caching, and capability introspection.
"""
import asyncio
import logging
from typing import Dict, List, Optional, Any, Set, NamedTuple
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass, field
import json

try:
    from mcp.types import Tool, Resource, Prompt
except ImportError:
    # Fallback types for development
    class Tool:
        def __init__(self, name: str, description: str = "", inputSchema: Dict = None):
            self.name = name
            self.description = description
            self.inputSchema = inputSchema or {}

    class Resource:
        def __init__(self, uri: str, name: str = "", description: str = ""):
            self.uri = uri
            self.name = name
            self.description = description

    class Prompt:
        def __init__(self, name: str, description: str = "", arguments: List = None):
            self.name = name
            self.description = description
            self.arguments = arguments or []

from .registry import MCPServerRegistry, MCPServerStatus


class ToolAvailabilityStatus(Enum):
    """Status of tool availability."""
    AVAILABLE = "available"
    UNAVAILABLE = "unavailable"
    ERROR = "error"
    UNKNOWN = "unknown"


@dataclass
class MCPToolInfo:
    """Enhanced tool information with metadata."""
    name: str
    server_name: str
    description: str
    input_schema: Dict[str, Any]
    availability_status: ToolAvailabilityStatus = ToolAvailabilityStatus.UNKNOWN
    last_checked: Optional[datetime] = None
    error_message: Optional[str] = None
    usage_count: int = 0
    last_used: Optional[datetime] = None
    response_time_ms: Optional[float] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        return {
            "name": self.name,
            "server_name": self.server_name,
            "description": self.description,
            "input_schema": self.input_schema,
            "availability_status": self.availability_status.value,
            "last_checked": self.last_checked.isoformat() if self.last_checked else None,
            "error_message": self.error_message,
            "usage_count": self.usage_count,
            "last_used": self.last_used.isoformat() if self.last_used else None,
            "response_time_ms": self.response_time_ms
        }


@dataclass
class MCPResourceInfo:
    """Enhanced resource information with metadata."""
    uri: str
    server_name: str
    name: str
    description: str
    mime_type: Optional[str] = None
    availability_status: ToolAvailabilityStatus = ToolAvailabilityStatus.UNKNOWN
    last_checked: Optional[datetime] = None
    error_message: Optional[str] = None
    access_count: int = 0
    last_accessed: Optional[datetime] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        return {
            "uri": self.uri,
            "server_name": self.server_name,
            "name": self.name,
            "description": self.description,
            "mime_type": self.mime_type,
            "availability_status": self.availability_status.value,
            "last_checked": self.last_checked.isoformat() if self.last_checked else None,
            "error_message": self.error_message,
            "access_count": self.access_count,
            "last_accessed": self.last_accessed.isoformat() if self.last_accessed else None
        }


@dataclass
class MCPPromptInfo:
    """Enhanced prompt information with metadata."""
    name: str
    server_name: str
    description: str
    arguments: List[Dict[str, Any]]
    availability_status: ToolAvailabilityStatus = ToolAvailabilityStatus.UNKNOWN
    last_checked: Optional[datetime] = None
    error_message: Optional[str] = None
    usage_count: int = 0
    last_used: Optional[datetime] = None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        return {
            "name": self.name,
            "server_name": self.server_name,
            "description": self.description,
            "arguments": self.arguments,
            "availability_status": self.availability_status.value,
            "last_checked": self.last_checked.isoformat() if self.last_checked else None,
            "error_message": self.error_message,
            "usage_count": self.usage_count,
            "last_used": self.last_used.isoformat() if self.last_used else None
        }


class DiscoveryResult(NamedTuple):
    """Result of a discovery operation."""
    server_name: str
    tools: List[MCPToolInfo]
    resources: List[MCPResourceInfo]
    prompts: List[MCPPromptInfo]
    discovery_time: datetime
    success: bool
    error_message: Optional[str] = None


class MCPToolDiscovery:
    """
    Advanced MCP Tool Discovery System.

    Provides comprehensive tool enumeration, caching, and capability introspection
    across all connected MCP servers.
    """

    def __init__(self, registry: MCPServerRegistry):
        self.registry = registry
        self.logger = logging.getLogger("MCPToolDiscovery")

        # Unified registries
        self._tools: Dict[str, MCPToolInfo] = {}  # tool_name -> tool_info
        self._resources: Dict[str, MCPResourceInfo] = {}  # uri -> resource_info
        self._prompts: Dict[str, MCPPromptInfo] = {}  # prompt_name -> prompt_info

        # Server mappings for conflict resolution
        self._tool_servers: Dict[str, Set[str]] = {}  # tool_name -> set of server_names
        self._resource_servers: Dict[str, Set[str]] = {}  # uri -> set of server_names
        self._prompt_servers: Dict[str, Set[str]] = {}  # prompt_name -> set of server_names

        # Caching
        self._cache_ttl = timedelta(minutes=5)
        self._last_discovery: Dict[str, datetime] = {}  # server_name -> last_discovery_time

        # Discovery task
        self._discovery_task: Optional[asyncio.Task] = None
        self._auto_discovery_interval = 300.0  # 5 minutes

    async def start_auto_discovery(self, interval: float = 300.0):
        """Start automatic tool discovery."""
        if self._discovery_task and not self._discovery_task.done():
            return

        self._auto_discovery_interval = interval
        self._discovery_task = asyncio.create_task(self._discovery_loop())
        self.logger.info(f"Started automatic tool discovery (interval: {interval}s)")

    async def stop_auto_discovery(self):
        """Stop automatic tool discovery."""
        if self._discovery_task and not self._discovery_task.done():
            self._discovery_task.cancel()
            try:
                await self._discovery_task
            except asyncio.CancelledError:
                pass

        self.logger.info("Stopped automatic tool discovery")

    async def _discovery_loop(self):
        """Main discovery loop."""
        while True:
            try:
                await self.discover_all_capabilities()
                await asyncio.sleep(self._auto_discovery_interval)
            except asyncio.CancelledError:
                break
            except Exception as e:
                self.logger.error(f"Error in discovery loop: {e}")
                await asyncio.sleep(30)  # Short delay before retrying

    async def discover_all_capabilities(self) -> List[DiscoveryResult]:
        """Discover capabilities from all connected servers."""
        connected_servers = self.registry.get_connected_servers()

        if not connected_servers:
            self.logger.info("No connected MCP servers for capability discovery")
            return []

        # Perform discovery on all servers concurrently
        discovery_tasks = [
            self._discover_server_capabilities(name, info)
            for name, info in connected_servers.items()
        ]

        results = await asyncio.gather(*discovery_tasks, return_exceptions=True)

        # Process results
        successful_results = []
        for result in results:
            if isinstance(result, DiscoveryResult):
                successful_results.append(result)
                if result.success:
                    await self._process_discovery_result(result)
            elif isinstance(result, Exception):
                self.logger.error(f"Discovery task failed: {result}")

        self.logger.info(
            f"Capability discovery complete: {len(successful_results)} servers processed, "
            f"{len(self._tools)} tools, {len(self._resources)} resources, {len(self._prompts)} prompts"
        )

        return successful_results

    async def _discover_server_capabilities(self, server_name: str, server_info) -> DiscoveryResult:
        """Discover capabilities from a specific server."""
        start_time = datetime.now()

        try:
            if not server_info.client or not server_info.is_healthy:
                return DiscoveryResult(
                    server_name=server_name,
                    tools=[],
                    resources=[],
                    prompts=[],
                    discovery_time=start_time,
                    success=False,
                    error_message="Server not healthy or client unavailable"
                )

            # Check cache validity
            if self._is_cache_valid(server_name):
                self.logger.debug(f"Using cached capabilities for server: {server_name}")
                return self._get_cached_result(server_name, start_time)

            client = server_info.client

            # Discover tools
            tools = []
            try:
                raw_tools = client.get_available_tools()
                for tool in raw_tools:
                    tool_info = MCPToolInfo(
                        name=tool.name,
                        server_name=server_name,
                        description=getattr(tool, 'description', ''),
                        input_schema=getattr(tool, 'inputSchema', {}),
                        availability_status=ToolAvailabilityStatus.AVAILABLE,
                        last_checked=start_time
                    )
                    tools.append(tool_info)
            except Exception as e:
                self.logger.error(f"Failed to discover tools from {server_name}: {e}")

            # Discover resources
            resources = []
            try:
                raw_resources = client.get_available_resources()
                for resource in raw_resources:
                    # Convert AnyUrl to string for string operations
                    uri_str = str(resource.uri)
                    resource_info = MCPResourceInfo(
                        uri=resource.uri,
                        server_name=server_name,
                        name=getattr(resource, 'name', uri_str.split('/')[-1]),
                        description=getattr(resource, 'description', ''),
                        mime_type=getattr(resource, 'mimeType', None),
                        availability_status=ToolAvailabilityStatus.AVAILABLE,
                        last_checked=start_time
                    )
                    resources.append(resource_info)
            except Exception as e:
                self.logger.error(f"Failed to discover resources from {server_name}: {e}")

            # Discover prompts
            prompts = []
            try:
                raw_prompts = client.get_available_prompts()
                for prompt in raw_prompts:
                    prompt_info = MCPPromptInfo(
                        name=prompt.name,
                        server_name=server_name,
                        description=getattr(prompt, 'description', ''),
                        arguments=getattr(prompt, 'arguments', []),
                        availability_status=ToolAvailabilityStatus.AVAILABLE,
                        last_checked=start_time
                    )
                    prompts.append(prompt_info)
            except Exception as e:
                self.logger.error(f"Failed to discover prompts from {server_name}: {e}")

            # Update last discovery time
            self._last_discovery[server_name] = start_time

            return DiscoveryResult(
                server_name=server_name,
                tools=tools,
                resources=resources,
                prompts=prompts,
                discovery_time=start_time,
                success=True
            )

        except Exception as e:
            self.logger.error(f"Discovery failed for server {server_name}: {e}")
            return DiscoveryResult(
                server_name=server_name,
                tools=[],
                resources=[],
                prompts=[],
                discovery_time=start_time,
                success=False,
                error_message=str(e)
            )

    def _is_cache_valid(self, server_name: str) -> bool:
        """Check if cached capabilities are still valid."""
        last_discovery = self._last_discovery.get(server_name)
        if not last_discovery:
            return False

        return datetime.now() - last_discovery < self._cache_ttl

    def _get_cached_result(self, server_name: str, discovery_time: datetime) -> DiscoveryResult:
        """Get cached discovery result for a server."""
        # Filter existing capabilities by server
        tools = [tool for tool in self._tools.values() if tool.server_name == server_name]
        resources = [res for res in self._resources.values() if res.server_name == server_name]
        prompts = [prompt for prompt in self._prompts.values() if prompt.server_name == server_name]

        return DiscoveryResult(
            server_name=server_name,
            tools=tools,
            resources=resources,
            prompts=prompts,
            discovery_time=discovery_time,
            success=True
        )

    async def _process_discovery_result(self, result: DiscoveryResult):
        """Process discovery result and update registries."""
        server_name = result.server_name

        # Clear existing entries for this server
        self._clear_server_capabilities(server_name)

        # Add tools
        for tool in result.tools:
            self._register_tool(tool)

        # Add resources
        for resource in result.resources:
            self._register_resource(resource)

        # Add prompts
        for prompt in result.prompts:
            self._register_prompt(prompt)

        self.logger.debug(
            f"Updated capabilities for {server_name}: "
            f"{len(result.tools)} tools, {len(result.resources)} resources, {len(result.prompts)} prompts"
        )

    def _clear_server_capabilities(self, server_name: str):
        """Clear all capabilities for a specific server."""
        # Remove tools
        tools_to_remove = [name for name, tool in self._tools.items() if tool.server_name == server_name]
        for tool_name in tools_to_remove:
            self._unregister_tool(tool_name, server_name)

        # Remove resources
        resources_to_remove = [uri for uri, res in self._resources.items() if res.server_name == server_name]
        for uri in resources_to_remove:
            self._unregister_resource(uri, server_name)

        # Remove prompts
        prompts_to_remove = [name for name, prompt in self._prompts.items() if prompt.server_name == server_name]
        for prompt_name in prompts_to_remove:
            self._unregister_prompt(prompt_name, server_name)

    def _register_tool(self, tool: MCPToolInfo):
        """Register a tool in the unified registry."""
        # Handle name conflicts by preferring the first server or using server-qualified names
        if tool.name in self._tools:
            existing_tool = self._tools[tool.name]
            if existing_tool.server_name != tool.server_name:
                # Name conflict - use server-qualified name
                qualified_name = f"{tool.server_name}.{tool.name}"
                tool.name = qualified_name
                self.logger.warning(f"Tool name conflict resolved: {tool.name}")

        self._tools[tool.name] = tool

        # Update server mapping
        if tool.name not in self._tool_servers:
            self._tool_servers[tool.name] = set()
        self._tool_servers[tool.name].add(tool.server_name)

    def _register_resource(self, resource: MCPResourceInfo):
        """Register a resource in the unified registry."""
        self._resources[resource.uri] = resource

        # Update server mapping
        if resource.uri not in self._resource_servers:
            self._resource_servers[resource.uri] = set()
        self._resource_servers[resource.uri].add(resource.server_name)

    def _register_prompt(self, prompt: MCPPromptInfo):
        """Register a prompt in the unified registry."""
        # Handle name conflicts similar to tools
        if prompt.name in self._prompts:
            existing_prompt = self._prompts[prompt.name]
            if existing_prompt.server_name != prompt.server_name:
                qualified_name = f"{prompt.server_name}.{prompt.name}"
                prompt.name = qualified_name
                self.logger.warning(f"Prompt name conflict resolved: {prompt.name}")

        self._prompts[prompt.name] = prompt

        # Update server mapping
        if prompt.name not in self._prompt_servers:
            self._prompt_servers[prompt.name] = set()
        self._prompt_servers[prompt.name].add(prompt.server_name)

    def _unregister_tool(self, tool_name: str, server_name: str):
        """Unregister a tool from a specific server."""
        if tool_name in self._tool_servers:
            self._tool_servers[tool_name].discard(server_name)
            if not self._tool_servers[tool_name]:
                # No more servers provide this tool
                self._tools.pop(tool_name, None)
                del self._tool_servers[tool_name]

    def _unregister_resource(self, uri: str, server_name: str):
        """Unregister a resource from a specific server."""
        if uri in self._resource_servers:
            self._resource_servers[uri].discard(server_name)
            if not self._resource_servers[uri]:
                # No more servers provide this resource
                self._resources.pop(uri, None)
                del self._resource_servers[uri]

    def _unregister_prompt(self, prompt_name: str, server_name: str):
        """Unregister a prompt from a specific server."""
        if prompt_name in self._prompt_servers:
            self._prompt_servers[prompt_name].discard(server_name)
            if not self._prompt_servers[prompt_name]:
                # No more servers provide this prompt
                self._prompts.pop(prompt_name, None)
                del self._prompt_servers[prompt_name]

    # Public API methods

    def get_all_tools(self) -> List[MCPToolInfo]:
        """Get all available tools from all servers."""
        return list(self._tools.values())

    def get_all_resources(self) -> List[MCPResourceInfo]:
        """Get all available resources from all servers."""
        return list(self._resources.values())

    def get_all_prompts(self) -> List[MCPPromptInfo]:
        """Get all available prompts from all servers."""
        return list(self._prompts.values())

    def get_tool(self, name: str) -> Optional[MCPToolInfo]:
        """Get a specific tool by name."""
        return self._tools.get(name)

    def get_resource(self, uri: str) -> Optional[MCPResourceInfo]:
        """Get a specific resource by URI."""
        return self._resources.get(uri)

    def get_prompt(self, name: str) -> Optional[MCPPromptInfo]:
        """Get a specific prompt by name."""
        return self._prompts.get(name)

    def find_tools_by_server(self, server_name: str) -> List[MCPToolInfo]:
        """Find all tools provided by a specific server."""
        return [tool for tool in self._tools.values() if tool.server_name == server_name]

    def find_resources_by_server(self, server_name: str) -> List[MCPResourceInfo]:
        """Find all resources provided by a specific server."""
        return [res for res in self._resources.values() if res.server_name == server_name]

    def find_prompts_by_server(self, server_name: str) -> List[MCPPromptInfo]:
        """Find all prompts provided by a specific server."""
        return [prompt for prompt in self._prompts.values() if prompt.server_name == server_name]

    def search_tools(self, query: str, case_sensitive: bool = False) -> List[MCPToolInfo]:
        """Search tools by name or description."""
        if not case_sensitive:
            query = query.lower()

        results = []
        for tool in self._tools.values():
            search_text = f"{tool.name} {tool.description}"
            if not case_sensitive:
                search_text = search_text.lower()

            if query in search_text:
                results.append(tool)

        return results

    def get_tool_servers(self, tool_name: str) -> Set[str]:
        """Get all servers that provide a specific tool."""
        return self._tool_servers.get(tool_name, set()).copy()

    def get_capability_summary(self) -> Dict[str, Any]:
        """Get a summary of all discovered capabilities."""
        return {
            "total_tools": len(self._tools),
            "total_resources": len(self._resources),
            "total_prompts": len(self._prompts),
            "servers_discovered": len(self._last_discovery),
            "tools_by_server": {
                server: len(self.find_tools_by_server(server))
                for server in self._last_discovery.keys()
            },
            "last_discovery": {
                server: timestamp.isoformat()
                for server, timestamp in self._last_discovery.items()
            },
            "cache_ttl_minutes": self._cache_ttl.total_seconds() / 60
        }

    async def update_tool_availability(self, tool_name: str) -> bool:
        """Update availability status of a specific tool."""
        tool = self.get_tool(tool_name)
        if not tool:
            return False

        try:
            # Get the server client
            server_info = self.registry.get_server_info(tool.server_name)
            if not server_info or not server_info.is_healthy:
                tool.availability_status = ToolAvailabilityStatus.UNAVAILABLE
                tool.error_message = "Server not healthy"
                return False

            # Check if tool is still available
            client = server_info.client
            available_tools = client.get_available_tools()
            tool_names = [t.name for t in available_tools]

            if tool_name in tool_names or tool.name.split('.')[-1] in tool_names:
                tool.availability_status = ToolAvailabilityStatus.AVAILABLE
                tool.error_message = None
                tool.last_checked = datetime.now()
                return True
            else:
                tool.availability_status = ToolAvailabilityStatus.UNAVAILABLE
                tool.error_message = "Tool no longer available on server"
                tool.last_checked = datetime.now()
                return False

        except Exception as e:
            tool.availability_status = ToolAvailabilityStatus.ERROR
            tool.error_message = str(e)
            tool.last_checked = datetime.now()
            return False

    def record_tool_usage(self, tool_name: str, response_time_ms: Optional[float] = None):
        """Record usage statistics for a tool."""
        tool = self.get_tool(tool_name)
        if tool:
            tool.usage_count += 1
            tool.last_used = datetime.now()
            if response_time_ms is not None:
                tool.response_time_ms = response_time_ms

    def get_usage_statistics(self) -> Dict[str, Any]:
        """Get usage statistics for all tools."""
        stats = {
            "total_tool_calls": sum(tool.usage_count for tool in self._tools.values()),
            "most_used_tools": [],
            "average_response_times": {},
            "tools_by_availability": {}
        }

        # Most used tools (top 10)
        sorted_tools = sorted(
            self._tools.values(),
            key=lambda t: t.usage_count,
            reverse=True
        )
        stats["most_used_tools"] = [
            {
                "name": tool.name,
                "server": tool.server_name,
                "usage_count": tool.usage_count,
                "last_used": tool.last_used.isoformat() if tool.last_used else None
            }
            for tool in sorted_tools[:10]
        ]

        # Average response times
        for tool in self._tools.values():
            if tool.response_time_ms is not None:
                stats["average_response_times"][tool.name] = tool.response_time_ms

        # Tools by availability status
        for status in ToolAvailabilityStatus:
            count = sum(1 for tool in self._tools.values() if tool.availability_status == status)
            stats["tools_by_availability"][status.value] = count

        return stats