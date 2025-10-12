"""
Tests for MCP server registry, connection/disconnection functionality.
"""
import asyncio
import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from datetime import datetime

# Import the MCP components we're testing
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.infrastructure.mcp.registry import (
    MCPServerRegistry, MCPServerStatus, MCPServerInfo
)
from src.infrastructure.config.config import MCPServerConfig


class TestMCPServerRegistry:
    """Test MCP server registry functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.registry = MCPServerRegistry()

        self.test_config = MCPServerConfig(
            name="test-server",
            transport="stdio",
            command="python",
            args=["test_server.py"],
            enabled=True
        )

        self.disabled_config = MCPServerConfig(
            name="disabled-server",
            transport="stdio",
            command="python",
            args=["test_server.py"],
            enabled=False
        )

        self.sse_config = MCPServerConfig(
            name="sse-server",
            transport="sse",
            url="https://api.example.com/mcp",
            enabled=True
        )

    def teardown_method(self):
        """Clean up after tests."""
        asyncio.create_task(self.registry.shutdown())

    @pytest.mark.asyncio
    async def test_register_enabled_server(self):
        """Test registering an enabled server."""
        with patch.object(self.registry, '_connect_server') as mock_connect:
            mock_connect.return_value = True

            result = await self.registry.register_server(self.test_config)

            assert result is True
            assert "test-server" in self.registry._servers

            server_info = self.registry._servers["test-server"]
            assert server_info.config == self.test_config
            assert server_info.status != MCPServerStatus.DISABLED

            mock_connect.assert_called_once_with("test-server")

    @pytest.mark.asyncio
    async def test_register_disabled_server(self):
        """Test registering a disabled server."""
        result = await self.registry.register_server(self.disabled_config)

        assert result is True
        assert "disabled-server" in self.registry._servers

        server_info = self.registry._servers["disabled-server"]
        assert server_info.config == self.disabled_config
        assert server_info.status == MCPServerStatus.DISABLED

    @pytest.mark.asyncio
    async def test_register_invalid_server(self):
        """Test registering an invalid server configuration."""
        invalid_config = MCPServerConfig(
            name="invalid-server",
            transport="stdio",
            enabled=True
            # Missing required command
        )

        result = await self.registry.register_server(invalid_config)

        assert result is False
        assert "invalid-server" not in self.registry._servers

    @pytest.mark.asyncio
    async def test_unregister_server(self):
        """Test unregistering a server."""
        # First register a server
        await self.registry.register_server(self.test_config)
        assert "test-server" in self.registry._servers

        with patch.object(self.registry, '_disconnect_server') as mock_disconnect:
            mock_disconnect.return_value = True

            result = await self.registry.unregister_server("test-server")

            assert result is True
            assert "test-server" not in self.registry._servers
            mock_disconnect.assert_called_once_with("test-server")

    @pytest.mark.asyncio
    async def test_unregister_nonexistent_server(self):
        """Test unregistering a server that doesn't exist."""
        result = await self.registry.unregister_server("nonexistent-server")
        assert result is False

    @pytest.mark.asyncio
    async def test_connect_server_success(self):
        """Test successful server connection."""
        # Register server first
        await self.registry.register_server(self.test_config)

        with patch('src.infrastructure.mcp.client.MCPClient') as mock_client_class:
            mock_client = AsyncMock()
            mock_client_class.return_value = mock_client
            mock_client.connect.return_value = True

            # Mock tools for capability discovery
            from src.infrastructure.mcp.client import Tool
            mock_client.get_available_tools.return_value = [Tool("test_tool", "Test tool")]
            mock_client.get_available_resources.return_value = []
            mock_client.get_available_prompts.return_value = []

            result = await self.registry._connect_server("test-server")

            assert result is True

            server_info = self.registry._servers["test-server"]
            assert server_info.status == MCPServerStatus.CONNECTED
            assert server_info.client == mock_client
            assert server_info.tools_count == 1

    @pytest.mark.asyncio
    async def test_connect_server_failure(self):
        """Test failed server connection."""
        # Register server first
        await self.registry.register_server(self.test_config)

        with patch('src.infrastructure.mcp.client.MCPClient') as mock_client_class:
            mock_client = AsyncMock()
            mock_client_class.return_value = mock_client
            mock_client.connect.return_value = False

            result = await self.registry._connect_server("test-server")

            assert result is False

            server_info = self.registry._servers["test-server"]
            assert server_info.status == MCPServerStatus.ERROR
            assert server_info.client is None

    @pytest.mark.asyncio
    async def test_connect_disabled_server(self):
        """Test connecting to a disabled server."""
        # Register disabled server
        await self.registry.register_server(self.disabled_config)

        result = await self.registry._connect_server("disabled-server")

        assert result is False

        server_info = self.registry._servers["disabled-server"]
        assert server_info.status == MCPServerStatus.DISABLED

    @pytest.mark.asyncio
    async def test_disconnect_server_success(self):
        """Test successful server disconnection."""
        # Set up connected server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        mock_client = AsyncMock()
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED
        server_info.tools_count = 5

        result = await self.registry._disconnect_server("test-server")

        assert result is True
        assert server_info.status == MCPServerStatus.DISCONNECTED
        assert server_info.client is None
        assert server_info.tools_count == 0

        mock_client.disconnect.assert_called_once()

    @pytest.mark.asyncio
    async def test_disconnect_server_failure(self):
        """Test server disconnection with error."""
        # Set up connected server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        mock_client = AsyncMock()
        mock_client.disconnect.side_effect = Exception("Disconnect failed")
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        result = await self.registry._disconnect_server("test-server")

        # Should still return True even if disconnect fails
        assert result is True
        assert server_info.status == MCPServerStatus.DISCONNECTED
        assert server_info.client is None

    @pytest.mark.asyncio
    async def test_connect_all_servers(self):
        """Test connecting to all enabled servers."""
        # Register multiple servers
        await self.registry.register_server(self.test_config)
        await self.registry.register_server(self.disabled_config)
        await self.registry.register_server(self.sse_config)

        # Mock connection attempts
        with patch.object(self.registry, '_connect_server') as mock_connect:
            mock_connect.return_value = True

            await self.registry.connect_all()

            # Should only try to connect to enabled servers
            expected_calls = ["test-server", "sse-server"]
            actual_calls = [call[0][0] for call in mock_connect.call_args_list]

            assert set(actual_calls) == set(expected_calls)
            assert mock_connect.call_count == 2

    @pytest.mark.asyncio
    async def test_disconnect_all_servers(self):
        """Test disconnecting all connected servers."""
        # Set up multiple connected servers
        await self.registry.register_server(self.test_config)
        await self.registry.register_server(self.sse_config)

        # Set servers as connected
        self.registry._servers["test-server"].status = MCPServerStatus.CONNECTED
        self.registry._servers["sse-server"].status = MCPServerStatus.CONNECTED

        with patch.object(self.registry, '_disconnect_server') as mock_disconnect:
            mock_disconnect.return_value = True

            await self.registry.disconnect_all()

            # Should disconnect both connected servers
            expected_calls = ["test-server", "sse-server"]
            actual_calls = [call[0][0] for call in mock_disconnect.call_args_list]

            assert set(actual_calls) == set(expected_calls)
            assert mock_disconnect.call_count == 2

    @pytest.mark.asyncio
    async def test_health_monitoring(self):
        """Test health monitoring functionality."""
        # Register and set up server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        mock_client = AsyncMock()
        mock_client.health_check.return_value = True
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        # Start health monitoring
        await self.registry.start_health_monitoring(interval=0.1)

        # Wait for at least one health check
        await asyncio.sleep(0.2)

        # Verify health check was called
        mock_client.health_check.assert_called()

        # Stop monitoring
        await self.registry.stop_health_monitoring()

    @pytest.mark.asyncio
    async def test_health_check_failure_recovery(self):
        """Test health check failure and recovery."""
        # Register and set up server
        await self.registry.register_server(self.test_config)
        server_info = self.registry._servers["test-server"]

        mock_client = AsyncMock()
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        # Mock failing health check
        mock_client.health_check.return_value = False

        with patch.object(self.registry, '_disconnect_server') as mock_disconnect:
            with patch.object(self.registry, '_connect_server') as mock_connect:
                mock_disconnect.return_value = True
                mock_connect.return_value = True

                # Simulate health check failure
                await self.registry._health_check_server("test-server")

                # Should disconnect and attempt reconnection
                mock_disconnect.assert_called_once_with("test-server")
                mock_connect.assert_called_once_with("test-server")

    def test_get_server_info(self):
        """Test getting server information."""
        asyncio.run(self.registry.register_server(self.test_config))

        server_info = self.registry.get_server_info("test-server")
        assert server_info is not None
        assert server_info.config == self.test_config

        # Test nonexistent server
        assert self.registry.get_server_info("nonexistent") is None

    def test_get_all_servers(self):
        """Test getting all servers."""
        asyncio.run(self.registry.register_server(self.test_config))
        asyncio.run(self.registry.register_server(self.disabled_config))

        all_servers = self.registry.get_all_servers()
        assert len(all_servers) == 2
        assert "test-server" in all_servers
        assert "disabled-server" in all_servers

        # Verify we get a copy, not the original
        all_servers["fake"] = "fake"
        assert "fake" not in self.registry._servers

    def test_get_connected_servers(self):
        """Test getting only connected servers."""
        asyncio.run(self.registry.register_server(self.test_config))
        asyncio.run(self.registry.register_server(self.disabled_config))

        # Set one server as connected
        self.registry._servers["test-server"].status = MCPServerStatus.CONNECTED

        connected_servers = self.registry.get_connected_servers()
        assert len(connected_servers) == 1
        assert "test-server" in connected_servers
        assert "disabled-server" not in connected_servers

    def test_get_server_by_name(self):
        """Test getting MCP client by server name."""
        asyncio.run(self.registry.register_server(self.test_config))

        # No client when disconnected
        assert self.registry.get_server_by_name("test-server") is None

        # Mock connected state with healthy client
        server_info = self.registry._servers["test-server"]
        mock_client = MagicMock()
        server_info.client = mock_client
        server_info.status = MCPServerStatus.CONNECTED

        # Should return client when healthy
        client = self.registry.get_server_by_name("test-server")
        assert client == mock_client

    def test_get_registry_stats(self):
        """Test getting registry statistics."""
        asyncio.run(self.registry.register_server(self.test_config))
        asyncio.run(self.registry.register_server(self.disabled_config))

        # Set one server as connected with tools
        server_info = self.registry._servers["test-server"]
        server_info.status = MCPServerStatus.CONNECTED
        server_info.tools_count = 5
        server_info.resources_count = 2

        stats = self.registry.get_registry_stats()

        assert stats["total_servers"] == 2
        assert stats["connected_servers"] == 1
        assert stats["total_tools"] == 5
        assert stats["total_resources"] == 2
        assert stats["status_counts"]["connected"] == 1
        assert stats["status_counts"]["disabled"] == 1

    @pytest.mark.asyncio
    async def test_shutdown(self):
        """Test registry shutdown."""
        # Register servers
        await self.registry.register_server(self.test_config)

        # Set up connected state
        server_info = self.registry._servers["test-server"]
        server_info.status = MCPServerStatus.CONNECTED
        mock_client = AsyncMock()
        server_info.client = mock_client

        # Start health monitoring
        await self.registry.start_health_monitoring()

        with patch.object(self.registry, 'disconnect_all') as mock_disconnect_all:
            await self.registry.shutdown()

            # Should stop health monitoring and disconnect all servers
            mock_disconnect_all.assert_called_once()
            assert self.registry._health_check_task.done()


class TestMCPServerInfo:
    """Test MCP server info functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.config = MCPServerConfig(
            name="test-server",
            transport="stdio",
            command="python",
            args=["test_server.py"],
            retry_attempts=3,
            retry_delay=1.0
        )
        self.server_info = MCPServerInfo(self.config)

    def test_server_info_initialization(self):
        """Test server info initialization."""
        assert self.server_info.config == self.config
        assert self.server_info.client is None
        assert self.server_info.status == MCPServerStatus.DISCONNECTED
        assert self.server_info.connection_attempts == 0
        assert self.server_info.tools_count == 0

    def test_is_healthy(self):
        """Test health status checking."""
        # Initially not healthy
        assert self.server_info.is_healthy is False

        # Set up connected state
        mock_client = MagicMock()
        mock_client.is_connected = True
        self.server_info.client = mock_client
        self.server_info.status = MCPServerStatus.CONNECTED

        assert self.server_info.is_healthy is True

        # Not healthy if status is error
        self.server_info.status = MCPServerStatus.ERROR
        assert self.server_info.is_healthy is False

    def test_should_retry_connection(self):
        """Test connection retry logic."""
        # Should retry initially
        assert self.server_info.should_retry_connection is True

        # Should not retry if disabled
        self.server_info.config.enabled = False
        assert self.server_info.should_retry_connection is False
        self.server_info.config.enabled = True

        # Should not retry if max attempts reached
        self.server_info.connection_attempts = 5
        assert self.server_info.should_retry_connection is False
        self.server_info.connection_attempts = 1

        # Should not retry immediately after attempt
        self.server_info.last_connection_attempt = datetime.now()
        assert self.server_info.should_retry_connection is False

    def test_update_capabilities(self):
        """Test capability count updates."""
        # Mock client with capabilities
        mock_client = MagicMock()

        # Mock tools, resources, prompts
        from src.infrastructure.mcp.client import Tool, Resource, Prompt
        mock_client.get_available_tools.return_value = [
            Tool("tool1", "Tool 1"),
            Tool("tool2", "Tool 2")
        ]
        mock_client.get_available_resources.return_value = [
            Resource("res://1", "Resource 1")
        ]
        mock_client.get_available_prompts.return_value = [
            Prompt("prompt1", "Prompt 1"),
            Prompt("prompt2", "Prompt 2"),
            Prompt("prompt3", "Prompt 3")
        ]

        self.server_info.client = mock_client
        self.server_info.update_capabilities()

        assert self.server_info.tools_count == 2
        assert self.server_info.resources_count == 1
        assert self.server_info.prompts_count == 3


if __name__ == "__main__":
    pytest.main([__file__, "-v"])