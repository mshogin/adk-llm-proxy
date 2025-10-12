"""
Tests for MCP tool discovery and execution functionality.
"""
import asyncio
import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from datetime import datetime, timedelta

# Import the MCP components we're testing
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.infrastructure.mcp.discovery import (
    MCPToolDiscovery, MCPToolInfo, MCPResourceInfo, MCPPromptInfo,
    ToolAvailabilityStatus, DiscoveryResult
)
from src.infrastructure.mcp.registry import MCPServerRegistry, MCPServerStatus
from src.infrastructure.mcp.tool_registry import (
    MCPUnifiedToolRegistry, ToolExecutionStrategy, ToolExecutionResult
)
from src.infrastructure.config.config import MCPServerConfig


class TestMCPToolDiscovery:
    """Test MCP tool discovery functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.registry = MCPServerRegistry()
        self.discovery = MCPToolDiscovery(self.registry)

        self.test_config = MCPServerConfig(
            name="test-server",
            transport="stdio",
            command="python",
            args=["test_server.py"],
            enabled=True
        )

    def teardown_method(self):
        """Clean up after tests."""
        asyncio.create_task(self.discovery.stop_auto_discovery())
        asyncio.create_task(self.registry.shutdown())

    @pytest.mark.asyncio
    async def test_discover_server_capabilities_success(self):
        """Test successful capability discovery from a server."""
        # Register server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        # Mock healthy server with client
        mock_client = AsyncMock()
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        # Mock capabilities
        from src.infrastructure.mcp.client import Tool, Resource, Prompt
        mock_tools = [
            Tool("echo", "Echo tool", {"type": "object", "properties": {"message": {"type": "string"}}}),
            Tool("add", "Add tool", {"type": "object", "properties": {"a": {"type": "number"}, "b": {"type": "number"}}})
        ]
        mock_resources = [
            Resource("test://config", "Test Config", "Test configuration resource")
        ]
        mock_prompts = [
            Prompt("greeting", "Greeting prompt", [{"name": "name", "required": False}])
        ]

        mock_client.get_available_tools.return_value = mock_tools
        mock_client.get_available_resources.return_value = mock_resources
        mock_client.get_available_prompts.return_value = mock_prompts

        # Test discovery
        result = await self.discovery._discover_server_capabilities("test-server", server_info)

        assert result.success is True
        assert result.server_name == "test-server"
        assert len(result.tools) == 2
        assert len(result.resources) == 1
        assert len(result.prompts) == 1

        # Check tool details
        echo_tool = result.tools[0]
        assert echo_tool.name == "echo"
        assert echo_tool.server_name == "test-server"
        assert echo_tool.availability_status == ToolAvailabilityStatus.AVAILABLE

    @pytest.mark.asyncio
    async def test_discover_server_capabilities_failure(self):
        """Test capability discovery failure."""
        # Register server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        # Mock unhealthy server
        server_info.client = None
        server_info.status = MCPServerStatus.ERROR

        # Test discovery
        result = await self.discovery._discover_server_capabilities("test-server", server_info)

        assert result.success is False
        assert result.error_message is not None
        assert len(result.tools) == 0

    @pytest.mark.asyncio
    async def test_discover_all_capabilities(self):
        """Test discovering capabilities from all servers."""
        # Register multiple servers
        config2 = MCPServerConfig(
            name="test-server-2",
            transport="stdio",
            command="python",
            args=["test_server2.py"],
            enabled=True
        )

        await self.registry.register_server(self.test_config)
        await self.registry.register_server(config2)

        # Mock both servers as connected
        for server_name in ["test-server", "test-server-2"]:
            server_info = self.registry._servers[server_name]
            mock_client = AsyncMock()
            server_info.client = mock_client
            server_info.status = MCPServerStatus.CONNECTED

            # Mock different tools for each server
            if server_name == "test-server":
                from src.infrastructure.mcp.client import Tool
                mock_client.get_available_tools.return_value = [
                    Tool("echo", "Echo tool")
                ]
            else:
                mock_client.get_available_tools.return_value = [
                    Tool("calculate", "Calculator tool")
                ]

            mock_client.get_available_resources.return_value = []
            mock_client.get_available_prompts.return_value = []

        # Test discovery
        results = await self.discovery.discover_all_capabilities()

        assert len(results) == 2
        assert all(result.success for result in results)

        # Check that tools were registered
        all_tools = self.discovery.get_all_tools()
        tool_names = {tool.name for tool in all_tools}
        assert "echo" in tool_names
        assert "calculate" in tool_names

    @pytest.mark.asyncio
    async def test_name_conflict_resolution(self):
        """Test handling of tool name conflicts."""
        # Register two servers
        config2 = MCPServerConfig(
            name="test-server-2",
            transport="stdio",
            command="python",
            args=["test_server2.py"],
            enabled=True
        )

        await self.registry.register_server(self.test_config)
        await self.registry.register_server(config2)

        # Mock both servers with same tool name
        for server_name in ["test-server", "test-server-2"]:
            server_info = self.registry._servers[server_name]
            mock_client = AsyncMock()
            server_info.client = mock_client
            server_info.status = MCPServerStatus.CONNECTED

            # Both servers have "echo" tool
            from src.infrastructure.mcp.client import Tool
            mock_client.get_available_tools.return_value = [
                Tool("echo", f"Echo tool from {server_name}")
            ]
            mock_client.get_available_resources.return_value = []
            mock_client.get_available_prompts.return_value = []

        # Test discovery
        await self.discovery.discover_all_capabilities()

        # Check conflict resolution
        all_tools = self.discovery.get_all_tools()
        tool_names = {tool.name for tool in all_tools}

        # Should have original "echo" and qualified "test-server-2.echo"
        assert "echo" in tool_names
        assert "test-server-2.echo" in tool_names

    @pytest.mark.asyncio
    async def test_caching_functionality(self):
        """Test discovery result caching."""
        # Register server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        # Mock connected server
        mock_client = AsyncMock()
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        from src.infrastructure.mcp.client import Tool
        mock_client.get_available_tools.return_value = [Tool("echo", "Echo tool")]
        mock_client.get_available_resources.return_value = []
        mock_client.get_available_prompts.return_value = []

        # First discovery
        result1 = await self.discovery._discover_server_capabilities("test-server", server_info)
        assert result1.success is True

        # Mock cache as valid
        self.discovery._last_discovery["test-server"] = datetime.now()

        # Second discovery should use cache
        with patch.object(self.discovery, '_get_cached_result') as mock_cache:
            mock_cache.return_value = result1

            result2 = await self.discovery._discover_server_capabilities("test-server", server_info)
            mock_cache.assert_called_once()

    @pytest.mark.asyncio
    async def test_auto_discovery(self):
        """Test automatic discovery functionality."""
        # Register server
        await self.registry.register_server(self.test_config)

        with patch.object(self.discovery, 'discover_all_capabilities') as mock_discover:
            mock_discover.return_value = []

            # Start auto discovery with short interval
            await self.discovery.start_auto_discovery(interval=0.1)

            # Wait for at least one discovery cycle
            await asyncio.sleep(0.2)

            # Stop auto discovery
            await self.discovery.stop_auto_discovery()

            # Verify discovery was called
            assert mock_discover.call_count >= 1

    def test_tool_search(self):
        """Test tool search functionality."""
        # Add some test tools manually
        tool1 = MCPToolInfo(
            name="file_reader",
            server_name="test-server",
            description="Read files from filesystem",
            input_schema={}
        )
        tool2 = MCPToolInfo(
            name="web_search",
            server_name="test-server",
            description="Search the web for information",
            input_schema={}
        )

        self.discovery._tools = {
            "file_reader": tool1,
            "web_search": tool2
        }

        # Test search by name
        results = self.discovery.search_tools("file")
        assert len(results) == 1
        assert results[0].name == "file_reader"

        # Test search by description
        results = self.discovery.search_tools("web")
        assert len(results) == 1
        assert results[0].name == "web_search"

        # Test case insensitive search
        results = self.discovery.search_tools("READ")
        assert len(results) == 1

    def test_get_capability_summary(self):
        """Test capability summary generation."""
        # Add test data
        tool = MCPToolInfo(
            name="test_tool",
            server_name="test-server",
            description="Test tool",
            input_schema={}
        )
        self.discovery._tools = {"test_tool": tool}
        self.discovery._last_discovery = {"test-server": datetime.now()}

        summary = self.discovery.get_capability_summary()

        assert summary["total_tools"] == 1
        assert summary["total_resources"] == 0
        assert summary["total_prompts"] == 0
        assert summary["servers_discovered"] == 1
        assert "test-server" in summary["tools_by_server"]

    @pytest.mark.asyncio
    async def test_update_tool_availability(self):
        """Test tool availability updates."""
        # Add test tool
        tool = MCPToolInfo(
            name="test_tool",
            server_name="test-server",
            description="Test tool",
            input_schema={}
        )
        self.discovery._tools = {"test_tool": tool}

        # Register server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        # Mock healthy server
        mock_client = AsyncMock()
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        # Mock tool availability check
        from src.infrastructure.mcp.client import Tool
        mock_client.get_available_tools.return_value = [Tool("test_tool", "Test tool")]

        # Test availability update
        result = await self.discovery.update_tool_availability("test_tool")

        assert result is True
        assert tool.availability_status == ToolAvailabilityStatus.AVAILABLE
        assert tool.last_checked is not None

    def test_record_tool_usage(self):
        """Test tool usage recording."""
        # Add test tool
        tool = MCPToolInfo(
            name="test_tool",
            server_name="test-server",
            description="Test tool",
            input_schema={}
        )
        self.discovery._tools = {"test_tool": tool}

        # Initially no usage
        assert tool.usage_count == 0
        assert tool.last_used is None

        # Record usage
        self.discovery.record_tool_usage("test_tool", 150.5)

        assert tool.usage_count == 1
        assert tool.last_used is not None
        assert tool.response_time_ms == 150.5

        # Record more usage
        self.discovery.record_tool_usage("test_tool", 200.0)
        assert tool.usage_count == 2
        assert tool.response_time_ms == 200.0


class TestMCPUnifiedToolRegistry:
    """Test unified tool registry functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.registry = MCPServerRegistry()
        self.discovery = MCPToolDiscovery(self.registry)
        self.tool_registry = MCPUnifiedToolRegistry(self.registry, self.discovery)

        self.test_config = MCPServerConfig(
            name="test-server",
            transport="stdio",
            command="python",
            args=["test_server.py"],
            enabled=True
        )

    def teardown_method(self):
        """Clean up after tests."""
        asyncio.create_task(self.registry.shutdown())

    @pytest.mark.asyncio
    async def test_execute_tool_success(self):
        """Test successful tool execution."""
        # Set up server and tool
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        mock_client = AsyncMock()
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        # Mock tool availability
        tool = MCPToolInfo(
            name="echo",
            server_name="test-server",
            description="Echo tool",
            input_schema={}
        )
        self.discovery._tools = {"echo": tool}
        self.discovery._tool_servers = {"echo": {"test-server"}}

        # Mock successful tool execution
        mock_client.call_tool.return_value = "Hello World"

        with patch.object(self.registry, 'get_server_by_name', return_value=mock_client):
            result = await self.tool_registry.execute_tool("echo", {"message": "Hello World"})

            assert result.success is True
            assert result.result == "Hello World"
            assert result.server_name == "test-server"
            assert result.execution_time_ms is not None

    @pytest.mark.asyncio
    async def test_execute_tool_not_found(self):
        """Test tool execution when tool not found."""
        result = await self.tool_registry.execute_tool("nonexistent_tool", {})

        assert result.success is False
        assert "No available servers" in result.error_message

    @pytest.mark.asyncio
    async def test_execute_tool_with_caching(self):
        """Test tool execution with result caching."""
        # Enable caching
        self.tool_registry.enable_caching(enabled=True, default_ttl=300)

        # Set up server and tool
        await self.registry.register_server(self.test_config)

        tool = MCPToolInfo(
            name="echo",
            server_name="test-server",
            description="Echo tool",
            input_schema={}
        )
        self.discovery._tools = {"echo": tool}
        self.discovery._tool_servers = {"echo": {"test-server"}}

        mock_client = AsyncMock()
        mock_client.call_tool.return_value = "Cached Result"

        with patch.object(self.registry, 'get_server_by_name', return_value=mock_client):
            # First execution
            result1 = await self.tool_registry.execute_tool("echo", {"message": "test"})
            assert result1.success is True

            # Second execution should use cache
            result2 = await self.tool_registry.execute_tool("echo", {"message": "test"})
            assert result2.success is True
            assert result2.result == "Cached Result"

            # Mock should only be called once
            mock_client.call_tool.assert_called_once()

    @pytest.mark.asyncio
    async def test_batch_tool_execution(self):
        """Test batch tool execution."""
        # Set up server and tools
        await self.registry.register_server(self.test_config)

        tools = {
            "echo": MCPToolInfo("echo", "test-server", "Echo tool", {}),
            "add": MCPToolInfo("add", "test-server", "Add tool", {})
        }
        self.discovery._tools = tools
        self.discovery._tool_servers = {
            "echo": {"test-server"},
            "add": {"test-server"}
        }

        mock_client = AsyncMock()
        mock_client.call_tool.side_effect = lambda name, args: f"Result for {name}"

        with patch.object(self.registry, 'get_server_by_name', return_value=mock_client):
            batch_requests = [
                {"name": "echo", "arguments": {"message": "hello"}},
                {"name": "add", "arguments": {"a": 1, "b": 2}}
            ]

            results = await self.tool_registry.execute_batch_tools(batch_requests, parallel=True)

            assert len(results) == 2
            assert all(isinstance(result, ToolExecutionResult) for result in results)
            assert all(result.success for result in results)

    def test_execution_strategy_configuration(self):
        """Test execution strategy configuration."""
        # Test default strategy
        assert self.tool_registry._execution_strategy == ToolExecutionStrategy.FIRST_AVAILABLE

        # Test strategy change
        self.tool_registry.set_execution_strategy(ToolExecutionStrategy.ROUND_ROBIN)
        assert self.tool_registry._execution_strategy == ToolExecutionStrategy.ROUND_ROBIN

    def test_server_selection_strategies(self):
        """Test different server selection strategies."""
        available_servers = ["server1", "server2", "server3"]

        # Test first available
        self.tool_registry._execution_strategy = ToolExecutionStrategy.FIRST_AVAILABLE
        selected = self.tool_registry._select_server("test_tool", available_servers)
        assert selected == "server1"

        # Test round robin
        self.tool_registry._execution_strategy = ToolExecutionStrategy.ROUND_ROBIN
        selected1 = self.tool_registry._select_server("test_tool", available_servers)
        selected2 = self.tool_registry._select_server("test_tool", available_servers)
        selected3 = self.tool_registry._select_server("test_tool", available_servers)
        selected4 = self.tool_registry._select_server("test_tool", available_servers)

        # Should cycle through servers
        assert selected1 == "server1"
        assert selected2 == "server2"
        assert selected3 == "server3"
        assert selected4 == "server1"  # Back to first

    def test_caching_functionality(self):
        """Test result caching functionality."""
        # Test cache key generation
        key1 = self.tool_registry._make_cache_key("echo", {"message": "hello"}, "server1")
        key2 = self.tool_registry._make_cache_key("echo", {"message": "hello"}, "server1")
        key3 = self.tool_registry._make_cache_key("echo", {"message": "world"}, "server1")

        assert key1 == key2  # Same arguments should produce same key
        assert key1 != key3  # Different arguments should produce different keys

        # Test argument hashing
        hash1 = self.tool_registry._hash_arguments({"b": 2, "a": 1})
        hash2 = self.tool_registry._hash_arguments({"a": 1, "b": 2})
        assert hash1 == hash2  # Order shouldn't matter

    def test_tool_filtering(self):
        """Test tool filtering functionality."""
        # Add a filter that blocks tools containing "dangerous"
        def safety_filter(tool_name: str, arguments: dict) -> bool:
            return "dangerous" not in tool_name.lower()

        self.tool_registry.add_tool_filter(safety_filter)

        # Test filter application
        assert self.tool_registry._check_tool_filters("safe_tool", {}) is True
        assert self.tool_registry._check_tool_filters("dangerous_tool", {}) is False

    def test_get_registry_stats(self):
        """Test registry statistics."""
        # Add some test data
        tool = MCPToolInfo("test_tool", "test-server", "Test tool", {})
        tool.usage_count = 5
        self.discovery._tools = {"test_tool": tool}

        stats = self.tool_registry.get_registry_stats()

        assert "capabilities" in stats
        assert "usage" in stats
        assert "cache" in stats
        assert "execution" in stats

        assert stats["capabilities"]["total_tools"] == 1
        assert stats["execution"]["strategy"] == ToolExecutionStrategy.FIRST_AVAILABLE.value


if __name__ == "__main__":
    pytest.main([__file__, "-v"])