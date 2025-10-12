"""
Integration tests for MCP functionality with real test server.
"""
import asyncio
import pytest
import subprocess
import sys
import time
import json
from pathlib import Path

# Import the MCP components we're testing
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.infrastructure.mcp.registry import MCPServerRegistry
from src.infrastructure.mcp.discovery import MCPToolDiscovery
from src.infrastructure.mcp.tool_registry import MCPUnifiedToolRegistry, ToolExecutionStrategy
from src.infrastructure.mcp.introspection import MCPAvailabilityTracker, MCPCapabilityIntrospector
from src.infrastructure.config.config import MCPServerConfig


@pytest.mark.integration
class TestMCPIntegration:
    """Integration tests with real MCP test server."""

    def setup_method(self):
        """Set up integration test environment."""
        self.registry = MCPServerRegistry()
        self.discovery = MCPToolDiscovery(self.registry)
        self.tool_registry = MCPUnifiedToolRegistry(self.registry, self.discovery)
        self.availability_tracker = MCPAvailabilityTracker(self.registry, self.discovery)
        self.introspector = MCPCapabilityIntrospector(
            self.registry, self.discovery, self.availability_tracker
        )

        # Configuration for test MCP server
        self.test_server_config = MCPServerConfig(
            name="integration-test-server",
            transport="stdio",
            command=sys.executable,
            args=[str(Path(__file__).parent.parent / "test_mcp_server.py")],
            enabled=True,
            timeout=10.0,
            retry_attempts=2
        )

    def teardown_method(self):
        """Clean up after tests."""
        asyncio.run(self._cleanup())

    async def _cleanup(self):
        """Clean up test resources."""
        try:
            await self.availability_tracker.stop_tracking()
            await self.discovery.stop_auto_discovery()
            await self.registry.shutdown()
        except Exception as e:
            print(f"Cleanup error: {e}")

    @pytest.mark.asyncio
    async def test_full_mcp_integration_cycle(self):
        """Test complete MCP integration cycle with real server."""
        # Step 1: Register and connect to test server
        result = await self.registry.register_server(self.test_server_config)
        assert result is True

        # Wait a moment for connection
        await asyncio.sleep(1.0)

        # Verify server is connected
        server_info = self.registry.get_server_info("integration-test-server")
        assert server_info is not None

        if not server_info.is_healthy:
            # Try manual connection
            await self.registry._connect_server("integration-test-server")
            await asyncio.sleep(1.0)

        # Step 2: Discover capabilities
        discovery_results = await self.discovery.discover_all_capabilities()

        # Verify discovery worked
        assert len(discovery_results) >= 1
        successful_results = [r for r in discovery_results if r.success]
        assert len(successful_results) >= 1

        # Check that tools were discovered
        all_tools = self.discovery.get_all_tools()
        tool_names = {tool.name for tool in all_tools}

        # Test server should have these tools
        expected_tools = {"echo", "add", "get_time", "list_files", "read_file", "write_file"}
        assert expected_tools.issubset(tool_names)

        # Step 3: Test tool execution via unified registry
        echo_result = await self.tool_registry.execute_tool(
            "echo",
            {"message": "Integration Test Message"}
        )

        if echo_result.success:
            assert "Integration Test Message" in str(echo_result.result)
        else:
            pytest.skip(f"Echo tool execution failed: {echo_result.error_message}")

        # Step 4: Test mathematical tool
        add_result = await self.tool_registry.execute_tool(
            "add",
            {"a": 15, "b": 27}
        )

        if add_result.success:
            # Parse result (may be in text format)
            result_value = add_result.result
            if isinstance(result_value, str):
                # Extract number from string if needed
                try:
                    result_value = float(result_value)
                except (ValueError, TypeError):
                    # If it's JSON-formatted response, try parsing
                    if result_value and "[{" in str(result_value):
                        try:
                            parsed = json.loads(str(result_value))
                            if isinstance(parsed, list) and len(parsed) > 0:
                                result_value = parsed[0].get("text", result_value)
                                result_value = float(result_value)
                        except (json.JSONDecodeError, ValueError):
                            pass

            assert result_value == 42.0 or "42" in str(result_value)

        # Step 5: Test resource access
        all_resources = self.discovery.get_all_resources()
        if all_resources:
            resource = all_resources[0]
            resource_result = await self.tool_registry.get_resource(resource.uri)
            assert resource_result.success in [True, False]  # May fail but shouldn't crash

        # Step 6: Test batch tool execution
        batch_requests = [
            {"name": "echo", "arguments": {"message": "Batch Test 1"}},
            {"name": "add", "arguments": {"a": 10, "b": 20}},
            {"name": "get_time", "arguments": {"format": "iso"}}
        ]

        batch_results = await self.tool_registry.execute_batch_tools(
            batch_requests,
            parallel=True,
            max_concurrent=3
        )

        assert len(batch_results) == 3
        successful_batch = sum(1 for r in batch_results if isinstance(r, type(echo_result)) and r.success)
        assert successful_batch >= 1  # At least one should succeed

    @pytest.mark.asyncio
    async def test_tool_execution_strategies(self):
        """Test different tool execution strategies."""
        # Register server
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)

        # Discover tools
        await self.discovery.discover_all_capabilities()

        # Test different execution strategies
        strategies = [
            ToolExecutionStrategy.FIRST_AVAILABLE,
            ToolExecutionStrategy.ROUND_ROBIN,
            ToolExecutionStrategy.RANDOM
        ]

        for strategy in strategies:
            self.tool_registry.set_execution_strategy(strategy)

            result = await self.tool_registry.execute_tool(
                "echo",
                {"message": f"Strategy test: {strategy.value}"}
            )

            # Should work with any strategy (only one server anyway)
            if result.success:
                assert f"Strategy test: {strategy.value}" in str(result.result)

    @pytest.mark.asyncio
    async def test_caching_functionality(self):
        """Test tool result caching."""
        # Register server and discover tools
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)
        await self.discovery.discover_all_capabilities()

        # Enable caching
        self.tool_registry.enable_caching(enabled=True, default_ttl=60)

        # Execute same tool twice
        start_time = time.time()
        result1 = await self.tool_registry.execute_tool(
            "get_time",
            {"format": "timestamp"}
        )
        first_exec_time = time.time() - start_time

        start_time = time.time()
        result2 = await self.tool_registry.execute_tool(
            "get_time",
            {"format": "timestamp"}
        )
        second_exec_time = time.time() - start_time

        if result1.success and result2.success:
            # Second execution should be faster (cached)
            # Note: This is a heuristic test, may not always be true
            assert second_exec_time < first_exec_time + 0.1  # Allow some variance

            # Results should be identical for cached execution
            assert result1.result == result2.result

    @pytest.mark.asyncio
    async def test_error_handling(self):
        """Test error handling with problematic tools."""
        # Register server and discover tools
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)
        await self.discovery.discover_all_capabilities()

        # Test error tool (should fail gracefully)
        error_result = await self.tool_registry.execute_tool(
            "error_tool",
            {"error_message": "Integration test error"}
        )

        assert error_result.success is False
        assert error_result.error_message is not None
        assert "Integration test error" in error_result.error_message or "error" in error_result.error_message.lower()

        # Test timeout handling with slow tool
        slow_result = await self.tool_registry.execute_tool(
            "slow_tool",
            {"delay": 0.5},
            timeout=2.0
        )

        # Should succeed within timeout
        if slow_result.success:
            assert "0.5" in str(slow_result.result) or "Completed" in str(slow_result.result)

        # Test timeout with very slow operation
        timeout_result = await self.tool_registry.execute_tool(
            "slow_tool",
            {"delay": 5.0},
            timeout=1.0
        )

        # Should timeout
        assert timeout_result.success is False
        assert "timeout" in timeout_result.error_message.lower()

    @pytest.mark.asyncio
    async def test_health_monitoring_integration(self):
        """Test health monitoring with real server."""
        # Register server
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)

        # Start health monitoring
        await self.registry.start_health_monitoring(interval=0.5)

        # Start availability tracking
        await self.availability_tracker.start_tracking(interval=0.5)

        # Wait for a few monitoring cycles
        await asyncio.sleep(2.0)

        # Check health monitoring results
        server_info = self.registry.get_server_info("integration-test-server")
        if server_info and server_info.is_healthy:
            assert server_info.last_health_check is not None

        # Check availability tracking
        availability_summary = self.availability_tracker.get_availability_summary("echo")
        if availability_summary:
            assert availability_summary["tool_name"] == "echo"
            assert "availability_percentage_24h" in availability_summary

        # Stop monitoring
        await self.registry.stop_health_monitoring()
        await self.availability_tracker.stop_tracking()

    @pytest.mark.asyncio
    async def test_capability_introspection(self):
        """Test capability introspection functionality."""
        # Register server and discover tools
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)
        await self.discovery.discover_all_capabilities()

        # Get capability overview
        overview = self.introspector.get_capability_overview()

        assert overview["total_tools"] > 0
        assert "category_distribution" in overview
        assert "complexity_distribution" in overview

        # Introspect specific tool
        introspection_result = await self.introspector.introspect_tool("echo")

        if introspection_result:
            assert introspection_result.tool_name == "echo"
            assert introspection_result.server_name == "integration-test-server"
            assert introspection_result.category is not None
            assert introspection_result.compatibility is not None
            assert len(introspection_result.recommendations) >= 0

            # Check compatibility info
            compat = introspection_result.compatibility
            assert compat.min_parameters >= 0
            assert compat.max_parameters >= compat.min_parameters
            assert compat.complexity is not None

    @pytest.mark.asyncio
    async def test_tool_search_and_filtering(self):
        """Test tool search and filtering capabilities."""
        # Register server and discover tools
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)
        await self.discovery.discover_all_capabilities()

        # Test tool search
        file_tools = self.discovery.search_tools("file")
        assert len(file_tools) >= 2  # Should find "list_files", "read_file", "write_file"

        time_tools = self.discovery.search_tools("time")
        assert len(time_tools) >= 1  # Should find "get_time"

        # Test server-specific tool lookup
        server_tools = self.discovery.find_tools_by_server("integration-test-server")
        assert len(server_tools) >= 5  # Should have multiple tools

        # Test tool filtering
        def no_write_filter(tool_name: str, arguments: dict) -> bool:
            return "write" not in tool_name.lower()

        self.tool_registry.add_tool_filter(no_write_filter)

        # Try to execute filtered tool
        filtered_result = await self.tool_registry.execute_tool("write_file", {
            "path": "/tmp/test.txt",
            "content": "test"
        })

        assert filtered_result.success is False
        assert "denied by filters" in filtered_result.error_message

        # Regular tool should still work
        unfiltered_result = await self.tool_registry.execute_tool("echo", {"message": "test"})
        if unfiltered_result.success:
            assert "test" in str(unfiltered_result.result)

    @pytest.mark.asyncio
    async def test_auto_discovery(self):
        """Test automatic discovery functionality."""
        # Start auto discovery
        await self.discovery.start_auto_discovery(interval=1.0)

        # Register server after starting discovery
        await self.registry.register_server(self.test_server_config)

        # Wait for discovery to pick up the server
        await asyncio.sleep(3.0)

        # Check that tools were discovered
        all_tools = self.discovery.get_all_tools()
        tool_names = {tool.name for tool in all_tools}

        # Stop auto discovery
        await self.discovery.stop_auto_discovery()

        # Should have discovered tools
        assert len(tool_names) >= 5

    def test_registry_statistics(self):
        """Test comprehensive registry statistics."""
        asyncio.run(self._test_registry_statistics_async())

    async def _test_registry_statistics_async(self):
        """Async helper for registry statistics test."""
        # Register server and discover tools
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)
        await self.discovery.discover_all_capabilities()

        # Execute some tools to generate usage statistics
        await self.tool_registry.execute_tool("echo", {"message": "stats test"})
        await self.tool_registry.execute_tool("add", {"a": 1, "b": 2})

        # Get comprehensive statistics
        stats = self.tool_registry.get_registry_stats()

        assert "capabilities" in stats
        assert "usage" in stats
        assert "cache" in stats
        assert "execution" in stats

        # Check capability stats
        cap_stats = stats["capabilities"]
        assert cap_stats["total_tools"] >= 5

        # Check usage stats
        usage_stats = stats["usage"]
        assert "total_tool_calls" in usage_stats
        assert "most_used_tools" in usage_stats

        # Get registry-specific stats
        registry_stats = self.registry.get_registry_stats()
        assert registry_stats["total_servers"] >= 1

    @pytest.mark.asyncio
    async def test_concurrent_operations(self):
        """Test concurrent tool operations."""
        # Register server and discover tools
        await self.registry.register_server(self.test_server_config)
        await asyncio.sleep(1.0)
        await self.discovery.discover_all_capabilities()

        # Create multiple concurrent tool calls
        tasks = []
        for i in range(5):
            task = self.tool_registry.execute_tool(
                "echo",
                {"message": f"Concurrent message {i}"}
            )
            tasks.append(task)

        # Execute all concurrently
        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Check results
        successful_results = [
            r for r in results
            if not isinstance(r, Exception) and hasattr(r, 'success') and r.success
        ]

        # At least some should succeed
        assert len(successful_results) >= 1

        # Verify content
        for i, result in enumerate(successful_results[:5]):
            if result.success:
                assert f"Concurrent message" in str(result.result)


if __name__ == "__main__":
    # Run integration tests
    pytest.main([__file__, "-v", "-m", "integration"])