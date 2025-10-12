"""
Unified MCP Tool Registry.

This module provides a unified interface for accessing tools, resources, and prompts
from all connected MCP servers, with intelligent routing, load balancing, and caching.
"""
import asyncio
import logging
from typing import Dict, List, Optional, Any, Set, Union, Callable
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass
import json
import hashlib

from .discovery import MCPToolDiscovery, MCPToolInfo, MCPResourceInfo, MCPPromptInfo, ToolAvailabilityStatus
from .registry import MCPServerRegistry


class ToolExecutionStrategy(Enum):
    """Strategy for executing tools when multiple servers provide the same tool."""
    FIRST_AVAILABLE = "first_available"  # Use first available server
    ROUND_ROBIN = "round_robin"  # Rotate between servers
    FASTEST_RESPONSE = "fastest_response"  # Use server with fastest average response
    LEAST_USED = "least_used"  # Use server with least usage
    RANDOM = "random"  # Random selection


@dataclass
class ToolExecutionResult:
    """Result of a tool execution."""
    success: bool
    result: Any = None
    error_message: Optional[str] = None
    server_name: Optional[str] = None
    execution_time_ms: Optional[float] = None
    tool_name: Optional[str] = None


@dataclass
class CachedResult:
    """Cached tool execution result."""
    result: Any
    timestamp: datetime
    tool_name: str
    arguments_hash: str
    server_name: str
    ttl_seconds: int = 300  # 5 minutes default

    def is_valid(self) -> bool:
        """Check if cached result is still valid."""
        age = datetime.now() - self.timestamp
        return age.total_seconds() < self.ttl_seconds


class MCPUnifiedToolRegistry:
    """
    Unified Tool Registry for MCP servers.

    Provides a single interface for discovering, caching, and executing tools
    from multiple MCP servers with intelligent routing and load balancing.
    """

    def __init__(self, registry: MCPServerRegistry, discovery: MCPToolDiscovery):
        self.registry = registry
        self.discovery = discovery
        self.logger = logging.getLogger("MCPUnifiedToolRegistry")

        # Execution configuration
        self._execution_strategy = ToolExecutionStrategy.FIRST_AVAILABLE
        self._enable_caching = True
        self._default_cache_ttl = 300  # 5 minutes

        # Caching
        self._result_cache: Dict[str, CachedResult] = {}
        self._cache_stats = {"hits": 0, "misses": 0}

        # Load balancing state
        self._round_robin_counters: Dict[str, int] = {}  # tool_name -> counter

        # Tool execution filters
        self._tool_filters: List[Callable[[str, Dict[str, Any]], bool]] = []

    def set_execution_strategy(self, strategy: ToolExecutionStrategy):
        """Set the tool execution strategy."""
        self._execution_strategy = strategy
        self.logger.info(f"Tool execution strategy changed to: {strategy.value}")

    def enable_caching(self, enabled: bool = True, default_ttl: int = 300):
        """Enable or disable result caching."""
        self._enable_caching = enabled
        self._default_cache_ttl = default_ttl

        if not enabled:
            self._result_cache.clear()

        self.logger.info(f"Result caching {'enabled' if enabled else 'disabled'}")

    def add_tool_filter(self, filter_func: Callable[[str, Dict[str, Any]], bool]):
        """
        Add a filter function for tool execution.

        Filter function should return True to allow execution, False to deny.
        """
        self._tool_filters.append(filter_func)

    async def execute_tool(
        self,
        tool_name: str,
        arguments: Dict[str, Any],
        server_name: Optional[str] = None,
        cache_ttl: Optional[int] = None,
        timeout: Optional[float] = None
    ) -> ToolExecutionResult:
        """
        Execute a tool with intelligent server selection and caching.

        Args:
            tool_name: Name of the tool to execute
            arguments: Tool arguments
            server_name: Specific server to use (optional)
            cache_ttl: Cache TTL in seconds (optional)
            timeout: Execution timeout (optional)

        Returns:
            ToolExecutionResult with execution details
        """
        start_time = datetime.now()

        try:
            # Apply filters
            if not self._check_tool_filters(tool_name, arguments):
                return ToolExecutionResult(
                    success=False,
                    error_message="Tool execution denied by filters",
                    tool_name=tool_name
                )

            # Check cache first
            if self._enable_caching:
                cached_result = self._get_cached_result(tool_name, arguments, server_name)
                if cached_result:
                    self._cache_stats["hits"] += 1
                    return ToolExecutionResult(
                        success=True,
                        result=cached_result.result,
                        server_name=cached_result.server_name,
                        tool_name=tool_name
                    )
                self._cache_stats["misses"] += 1

            # Find available servers
            available_servers = self._find_available_servers(tool_name, server_name)
            if not available_servers:
                return ToolExecutionResult(
                    success=False,
                    error_message=f"No available servers for tool: {tool_name}",
                    tool_name=tool_name
                )

            # Select server based on strategy
            selected_server = self._select_server(tool_name, available_servers)

            # Execute tool
            result = await self._execute_on_server(
                selected_server,
                tool_name,
                arguments,
                timeout
            )

            # Calculate execution time
            execution_time = (datetime.now() - start_time).total_seconds() * 1000

            # Cache result if successful
            if result.success and self._enable_caching:
                self._cache_result(tool_name, arguments, result.result, selected_server, cache_ttl)

            # Record usage statistics
            self.discovery.record_tool_usage(tool_name, execution_time)

            # Update result with execution time
            result.execution_time_ms = execution_time
            result.tool_name = tool_name

            return result

        except Exception as e:
            self.logger.error(f"Tool execution failed for {tool_name}: {e}")
            return ToolExecutionResult(
                success=False,
                error_message=str(e),
                tool_name=tool_name,
                execution_time_ms=(datetime.now() - start_time).total_seconds() * 1000
            )

    async def execute_batch_tools(
        self,
        tool_requests: List[Dict[str, Any]],
        parallel: bool = True,
        max_concurrent: int = 10
    ) -> List[ToolExecutionResult]:
        """
        Execute multiple tools in batch.

        Args:
            tool_requests: List of tool requests with 'name' and 'arguments' keys
            parallel: Execute in parallel or sequentially
            max_concurrent: Maximum concurrent executions

        Returns:
            List of ToolExecutionResult objects
        """
        if not parallel:
            # Sequential execution
            results = []
            for request in tool_requests:
                result = await self.execute_tool(
                    request["name"],
                    request.get("arguments", {}),
                    request.get("server_name"),
                    request.get("cache_ttl"),
                    request.get("timeout")
                )
                results.append(result)
            return results

        # Parallel execution with concurrency limit
        semaphore = asyncio.Semaphore(max_concurrent)

        async def execute_with_semaphore(request):
            async with semaphore:
                return await self.execute_tool(
                    request["name"],
                    request.get("arguments", {}),
                    request.get("server_name"),
                    request.get("cache_ttl"),
                    request.get("timeout")
                )

        tasks = [execute_with_semaphore(req) for req in tool_requests]
        return await asyncio.gather(*tasks, return_exceptions=True)

    def _check_tool_filters(self, tool_name: str, arguments: Dict[str, Any]) -> bool:
        """Check if tool execution passes all filters."""
        for filter_func in self._tool_filters:
            try:
                if not filter_func(tool_name, arguments):
                    return False
            except Exception as e:
                self.logger.warning(f"Tool filter error: {e}")
                return False
        return True

    def _get_cached_result(
        self,
        tool_name: str,
        arguments: Dict[str, Any],
        server_name: Optional[str]
    ) -> Optional[CachedResult]:
        """Get cached result if available and valid."""
        cache_key = self._make_cache_key(tool_name, arguments, server_name)
        cached = self._result_cache.get(cache_key)

        if cached and cached.is_valid():
            return cached

        # Remove invalid cache entry
        if cached:
            del self._result_cache[cache_key]

        return None

    def _cache_result(
        self,
        tool_name: str,
        arguments: Dict[str, Any],
        result: Any,
        server_name: str,
        ttl: Optional[int]
    ):
        """Cache tool execution result."""
        cache_key = self._make_cache_key(tool_name, arguments, server_name)
        args_hash = self._hash_arguments(arguments)

        cached_result = CachedResult(
            result=result,
            timestamp=datetime.now(),
            tool_name=tool_name,
            arguments_hash=args_hash,
            server_name=server_name,
            ttl_seconds=ttl or self._default_cache_ttl
        )

        self._result_cache[cache_key] = cached_result

        # Clean up old cache entries periodically
        if len(self._result_cache) > 1000:  # Arbitrary limit
            self._cleanup_cache()

    def _make_cache_key(
        self,
        tool_name: str,
        arguments: Dict[str, Any],
        server_name: Optional[str]
    ) -> str:
        """Create cache key from tool name, arguments, and server."""
        args_hash = self._hash_arguments(arguments)
        server_part = server_name or "any"
        return f"{tool_name}:{server_part}:{args_hash}"

    def _hash_arguments(self, arguments: Dict[str, Any]) -> str:
        """Create hash of arguments for caching."""
        # Sort keys for consistent hashing
        sorted_args = json.dumps(arguments, sort_keys=True, default=str)
        return hashlib.md5(sorted_args.encode()).hexdigest()[:16]

    def _cleanup_cache(self):
        """Remove expired cache entries."""
        now = datetime.now()
        expired_keys = [
            key for key, cached in self._result_cache.items()
            if not cached.is_valid()
        ]

        for key in expired_keys:
            del self._result_cache[key]

        self.logger.debug(f"Cleaned up {len(expired_keys)} expired cache entries")

    def _find_available_servers(
        self,
        tool_name: str,
        preferred_server: Optional[str] = None
    ) -> List[str]:
        """Find servers that have the tool and are available."""
        # Get tool info
        tool = self.discovery.get_tool(tool_name)
        if not tool:
            return []

        # If preferred server specified, check if it's available
        if preferred_server:
            server_info = self.registry.get_server_info(preferred_server)
            if (server_info and server_info.is_healthy and
                self.registry.get_server_by_name(preferred_server)):
                return [preferred_server]
            return []

        # Find all servers that provide this tool
        potential_servers = self.discovery.get_tool_servers(tool_name)
        available_servers = []

        for server_name in potential_servers:
            server_info = self.registry.get_server_info(server_name)
            if server_info and server_info.is_healthy:
                client = self.registry.get_server_by_name(server_name)
                if client:
                    available_servers.append(server_name)

        return available_servers

    def _select_server(self, tool_name: str, available_servers: List[str]) -> str:
        """Select server based on configured strategy."""
        if len(available_servers) == 1:
            return available_servers[0]

        if self._execution_strategy == ToolExecutionStrategy.FIRST_AVAILABLE:
            return available_servers[0]

        elif self._execution_strategy == ToolExecutionStrategy.ROUND_ROBIN:
            counter = self._round_robin_counters.get(tool_name, 0)
            selected_server = available_servers[counter % len(available_servers)]
            self._round_robin_counters[tool_name] = counter + 1
            return selected_server

        elif self._execution_strategy == ToolExecutionStrategy.FASTEST_RESPONSE:
            # Select server with best average response time
            best_server = available_servers[0]
            best_time = float('inf')

            for server_name in available_servers:
                tools = self.discovery.find_tools_by_server(server_name)
                server_tools = [t for t in tools if t.name == tool_name]
                if server_tools and server_tools[0].response_time_ms:
                    if server_tools[0].response_time_ms < best_time:
                        best_time = server_tools[0].response_time_ms
                        best_server = server_name

            return best_server

        elif self._execution_strategy == ToolExecutionStrategy.LEAST_USED:
            # Select server with lowest usage for this tool
            least_used_server = available_servers[0]
            least_usage = float('inf')

            for server_name in available_servers:
                tools = self.discovery.find_tools_by_server(server_name)
                server_tools = [t for t in tools if t.name == tool_name]
                if server_tools:
                    usage = server_tools[0].usage_count
                    if usage < least_usage:
                        least_usage = usage
                        least_used_server = server_name

            return least_used_server

        elif self._execution_strategy == ToolExecutionStrategy.RANDOM:
            import random
            return random.choice(available_servers)

        # Default fallback
        return available_servers[0]

    async def _execute_on_server(
        self,
        server_name: str,
        tool_name: str,
        arguments: Dict[str, Any],
        timeout: Optional[float]
    ) -> ToolExecutionResult:
        """Execute tool on a specific server."""
        try:
            client = self.registry.get_server_by_name(server_name)
            if not client:
                return ToolExecutionResult(
                    success=False,
                    error_message=f"Server {server_name} not available",
                    server_name=server_name
                )

            # Execute with timeout if specified
            if timeout:
                result = await asyncio.wait_for(
                    client.call_tool(tool_name, arguments),
                    timeout=timeout
                )
            else:
                result = await client.call_tool(tool_name, arguments)

            if result is not None:
                return ToolExecutionResult(
                    success=True,
                    result=result,
                    server_name=server_name
                )
            else:
                return ToolExecutionResult(
                    success=False,
                    error_message="Tool returned None result",
                    server_name=server_name
                )

        except asyncio.TimeoutError:
            return ToolExecutionResult(
                success=False,
                error_message=f"Tool execution timed out after {timeout}s",
                server_name=server_name
            )
        except Exception as e:
            return ToolExecutionResult(
                success=False,
                error_message=str(e),
                server_name=server_name
            )

    # Resource and Prompt methods

    async def get_resource(
        self,
        uri: str,
        server_name: Optional[str] = None
    ) -> ToolExecutionResult:
        """Get a resource from MCP servers."""
        try:
            resource_info = self.discovery.get_resource(uri)
            if not resource_info:
                return ToolExecutionResult(
                    success=False,
                    error_message=f"Resource not found: {uri}"
                )

            target_server = server_name or resource_info.server_name
            client = self.registry.get_server_by_name(target_server)

            if not client:
                return ToolExecutionResult(
                    success=False,
                    error_message=f"Server {target_server} not available"
                )

            result = await client.get_resource(uri)

            return ToolExecutionResult(
                success=result is not None,
                result=result,
                server_name=target_server,
                error_message=None if result else "Resource returned None"
            )

        except Exception as e:
            return ToolExecutionResult(
                success=False,
                error_message=str(e),
                server_name=server_name
            )

    async def get_prompt(
        self,
        name: str,
        arguments: Optional[Dict[str, Any]] = None,
        server_name: Optional[str] = None
    ) -> ToolExecutionResult:
        """Get a prompt from MCP servers."""
        try:
            prompt_info = self.discovery.get_prompt(name)
            if not prompt_info:
                return ToolExecutionResult(
                    success=False,
                    error_message=f"Prompt not found: {name}"
                )

            target_server = server_name or prompt_info.server_name
            client = self.registry.get_server_by_name(target_server)

            if not client:
                return ToolExecutionResult(
                    success=False,
                    error_message=f"Server {target_server} not available"
                )

            result = await client.get_prompt(name, arguments)

            return ToolExecutionResult(
                success=result is not None,
                result=result,
                server_name=target_server,
                error_message=None if result else "Prompt returned None"
            )

        except Exception as e:
            return ToolExecutionResult(
                success=False,
                error_message=str(e),
                server_name=server_name
            )

    # Registry management methods

    def get_registry_stats(self) -> Dict[str, Any]:
        """Get comprehensive registry statistics."""
        capability_summary = self.discovery.get_capability_summary()
        usage_stats = self.discovery.get_usage_statistics()

        return {
            "capabilities": capability_summary,
            "usage": usage_stats,
            "cache": {
                "enabled": self._enable_caching,
                "entries": len(self._result_cache),
                "hits": self._cache_stats["hits"],
                "misses": self._cache_stats["misses"],
                "hit_rate": self._cache_stats["hits"] / (self._cache_stats["hits"] + self._cache_stats["misses"]) if (self._cache_stats["hits"] + self._cache_stats["misses"]) > 0 else 0
            },
            "execution": {
                "strategy": self._execution_strategy.value,
                "filters_count": len(self._tool_filters)
            }
        }

    def clear_cache(self):
        """Clear all cached results."""
        self._result_cache.clear()
        self._cache_stats = {"hits": 0, "misses": 0}
        self.logger.info("Tool execution cache cleared")

    def get_tool_info(self, tool_name: str) -> Optional[Dict[str, Any]]:
        """Get detailed information about a tool."""
        tool = self.discovery.get_tool(tool_name)
        if not tool:
            return None

        servers = self.discovery.get_tool_servers(tool_name)
        available_servers = self._find_available_servers(tool_name)

        return {
            "name": tool.name,
            "description": tool.description,
            "input_schema": tool.input_schema,
            "primary_server": tool.server_name,
            "all_servers": list(servers),
            "available_servers": available_servers,
            "availability_status": tool.availability_status.value,
            "usage_count": tool.usage_count,
            "last_used": tool.last_used.isoformat() if tool.last_used else None,
            "response_time_ms": tool.response_time_ms,
            "last_checked": tool.last_checked.isoformat() if tool.last_checked else None,
            "error_message": tool.error_message
        }