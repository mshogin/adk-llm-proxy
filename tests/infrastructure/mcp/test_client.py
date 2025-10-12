"""
Unit tests for MCP client functionality.
"""
import asyncio
import pytest
import json
from unittest.mock import AsyncMock, MagicMock, patch
from datetime import datetime

# Import the MCP components we're testing
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.infrastructure.mcp.client import MCPClient, MCPServerConfig, MCPTransportType
from src.infrastructure.config.config import MCPServerConfig as ConfigMCPServerConfig


class TestMCPServerConfig:
    """Test MCP server configuration."""

    def test_stdio_config_validation(self):
        """Test stdio configuration validation."""
        # Valid stdio config
        config = ConfigMCPServerConfig(
            name="test-server",
            transport="stdio",
            command="python",
            args=["-m", "test_server"]
        )
        assert config.validate() is True

        # Invalid stdio config - no command
        config_invalid = ConfigMCPServerConfig(
            name="test-server",
            transport="stdio"
        )
        with pytest.raises(ValueError, match="command is required for stdio transport"):
            config_invalid.validate()

    def test_sse_config_validation(self):
        """Test SSE configuration validation."""
        # Valid SSE config
        config = ConfigMCPServerConfig(
            name="test-server",
            transport="sse",
            url="https://api.example.com/mcp"
        )
        assert config.validate() is True

        # Invalid SSE config - no URL
        config_invalid = ConfigMCPServerConfig(
            name="test-server",
            transport="sse"
        )
        with pytest.raises(ValueError, match="url is required for sse transport"):
            config_invalid.validate()

    def test_websocket_config_validation(self):
        """Test WebSocket configuration validation."""
        # Valid WebSocket config
        config = ConfigMCPServerConfig(
            name="test-server",
            transport="websocket",
            url="wss://api.example.com/mcp"
        )
        assert config.validate() is True

        # Invalid WebSocket config - no URL
        config_invalid = ConfigMCPServerConfig(
            name="test-server",
            transport="websocket"
        )
        with pytest.raises(ValueError, match="url is required for websocket transport"):
            config_invalid.validate()

    def test_invalid_transport_type(self):
        """Test invalid transport type validation."""
        config = ConfigMCPServerConfig(
            name="test-server",
            transport="invalid"
        )
        with pytest.raises(ValueError, match="Invalid transport type"):
            config.validate()


class TestMCPClient:
    """Test MCP client functionality."""

    def setup_method(self):
        """Set up test fixtures."""
        self.stdio_config = ConfigMCPServerConfig(
            name="test-stdio-server",
            transport="stdio",
            command="python",
            args=["test_server.py"]
        )

        self.sse_config = ConfigMCPServerConfig(
            name="test-sse-server",
            transport="sse",
            url="https://api.example.com/mcp"
        )

        self.websocket_config = ConfigMCPServerConfig(
            name="test-websocket-server",
            transport="websocket",
            url="wss://api.example.com/mcp"
        )

    def test_client_initialization(self):
        """Test MCP client initialization."""
        client = MCPClient(self.stdio_config)

        assert client.config == self.stdio_config
        assert client.session is None
        assert client.is_connected is False
        assert len(client._tools) == 0
        assert len(client._resources) == 0
        assert len(client._prompts) == 0

    @pytest.mark.asyncio
    async def test_connect_invalid_config(self):
        """Test connection with invalid configuration."""
        invalid_config = ConfigMCPServerConfig(
            name="invalid-server",
            transport="stdio"
        )
        client = MCPClient(invalid_config)

        result = await client.connect()
        assert result is False
        assert client.is_connected is False

    @pytest.mark.asyncio
    async def test_connect_stdio_success(self):
        """Test successful stdio connection."""
        client = MCPClient(self.stdio_config)

        with patch('src.infrastructure.mcp.client.stdio_client') as mock_stdio:
            # Mock successful connection
            mock_session = AsyncMock()
            mock_stdio.return_value = (mock_session, AsyncMock(), AsyncMock())

            # Mock session initialization and capability discovery
            mock_session.initialize = AsyncMock()
            mock_session.list_tools = AsyncMock(return_value=MagicMock(tools=[]))
            mock_session.list_resources = AsyncMock(return_value=MagicMock(resources=[]))
            mock_session.list_prompts = AsyncMock(return_value=MagicMock(prompts=[]))

            result = await client.connect()

            assert result is True
            assert client.is_connected is True
            assert client.session == mock_session
            mock_session.initialize.assert_called_once()

    @pytest.mark.asyncio
    async def test_connect_stdio_failure(self):
        """Test failed stdio connection."""
        client = MCPClient(self.stdio_config)

        with patch('src.infrastructure.mcp.client.stdio_client') as mock_stdio:
            # Mock connection failure
            mock_stdio.side_effect = Exception("Connection failed")

            result = await client.connect()

            assert result is False
            assert client.is_connected is False
            assert client.session is None

    @pytest.mark.asyncio
    async def test_disconnect(self):
        """Test disconnection."""
        client = MCPClient(self.stdio_config)

        # Set up connected state
        mock_session = AsyncMock()
        client.session = mock_session
        client.is_connected = True

        await client.disconnect()

        mock_session.close.assert_called_once()
        assert client.is_connected is False
        assert client.session is None

    @pytest.mark.asyncio
    async def test_call_tool_success(self):
        """Test successful tool call."""
        client = MCPClient(self.stdio_config)

        # Set up connected state
        mock_session = AsyncMock()
        client.session = mock_session
        client.is_connected = True

        # Mock successful tool call
        expected_result = {"result": "test output"}
        mock_session.call_tool.return_value = MagicMock(content=expected_result)

        result = await client.call_tool("test_tool", {"param": "value"})

        assert result == expected_result
        mock_session.call_tool.assert_called_once_with("test_tool", {"param": "value"})

    @pytest.mark.asyncio
    async def test_call_tool_not_connected(self):
        """Test tool call when not connected."""
        client = MCPClient(self.stdio_config)

        result = await client.call_tool("test_tool", {"param": "value"})

        assert result is None

    @pytest.mark.asyncio
    async def test_call_tool_failure(self):
        """Test failed tool call."""
        client = MCPClient(self.stdio_config)

        # Set up connected state
        mock_session = AsyncMock()
        client.session = mock_session
        client.is_connected = True

        # Mock tool call failure
        mock_session.call_tool.side_effect = Exception("Tool call failed")

        result = await client.call_tool("test_tool", {"param": "value"})

        assert result is None

    @pytest.mark.asyncio
    async def test_get_resource_success(self):
        """Test successful resource retrieval."""
        client = MCPClient(self.stdio_config)

        # Set up connected state
        mock_session = AsyncMock()
        client.session = mock_session
        client.is_connected = True

        # Mock successful resource retrieval
        expected_result = {"content": "resource data"}
        mock_session.read_resource.return_value = MagicMock(contents=expected_result)

        result = await client.get_resource("test://resource")

        assert result == expected_result
        mock_session.read_resource.assert_called_once_with("test://resource")

    @pytest.mark.asyncio
    async def test_get_prompt_success(self):
        """Test successful prompt retrieval."""
        client = MCPClient(self.stdio_config)

        # Set up connected state
        mock_session = AsyncMock()
        client.session = mock_session
        client.is_connected = True

        # Mock successful prompt retrieval
        expected_result = [{"role": "user", "content": "test prompt"}]
        mock_session.get_prompt.return_value = MagicMock(messages=expected_result)

        result = await client.get_prompt("test_prompt", {"param": "value"})

        assert result == expected_result
        mock_session.get_prompt.assert_called_once_with("test_prompt", {"param": "value"})

    @pytest.mark.asyncio
    async def test_health_check_success(self):
        """Test successful health check."""
        client = MCPClient(self.stdio_config)

        # Set up connected state
        mock_session = AsyncMock()
        client.session = mock_session
        client.is_connected = True

        # Mock successful health check (list_tools as proxy)
        mock_session.list_tools.return_value = MagicMock(tools=[])

        result = await client.health_check()

        assert result is True
        mock_session.list_tools.assert_called_once()

    @pytest.mark.asyncio
    async def test_health_check_not_connected(self):
        """Test health check when not connected."""
        client = MCPClient(self.stdio_config)

        result = await client.health_check()

        assert result is False

    @pytest.mark.asyncio
    async def test_health_check_failure(self):
        """Test failed health check."""
        client = MCPClient(self.stdio_config)

        # Set up connected state
        mock_session = AsyncMock()
        client.session = mock_session
        client.is_connected = True

        # Mock health check failure
        mock_session.list_tools.side_effect = Exception("Health check failed")

        result = await client.health_check()

        assert result is False

    def test_tool_availability_tracking(self):
        """Test tool availability checking."""
        client = MCPClient(self.stdio_config)

        # Initially no tools
        assert client.is_tool_available("test_tool") is False

        # Mock some tools
        from src.infrastructure.mcp.client import Tool
        mock_tool = Tool("test_tool", "Test tool description")
        client._tools = [mock_tool]

        assert client.is_tool_available("test_tool") is True
        assert client.is_tool_available("nonexistent_tool") is False

    def test_get_available_capabilities(self):
        """Test getting available capabilities."""
        client = MCPClient(self.stdio_config)

        # Mock capabilities
        from src.infrastructure.mcp.client import Tool, Resource, Prompt
        client._tools = [Tool("tool1", "Tool 1"), Tool("tool2", "Tool 2")]
        client._resources = [Resource("res://1", "Resource 1")]
        client._prompts = [Prompt("prompt1", "Prompt 1")]

        assert len(client.get_available_tools()) == 2
        assert len(client.get_available_resources()) == 1
        assert len(client.get_available_prompts()) == 1

        # Check that we get copies, not original lists
        tools = client.get_available_tools()
        tools.append("fake")
        assert len(client._tools) == 2  # Original unchanged


@pytest.mark.asyncio
class TestMCPClientIntegration:
    """Integration tests for MCP client with mock server."""

    async def test_full_connection_cycle(self):
        """Test full connection, operation, and disconnection cycle."""
        config = ConfigMCPServerConfig(
            name="integration-test-server",
            transport="stdio",
            command="python",
            args=["test_server.py"]
        )

        client = MCPClient(config)

        with patch('src.infrastructure.mcp.client.stdio_client') as mock_stdio:
            # Mock session and connection
            mock_session = AsyncMock()
            mock_stdio.return_value = (mock_session, AsyncMock(), AsyncMock())

            # Mock capabilities
            from src.infrastructure.mcp.client import Tool
            mock_tools = [
                Tool("echo", "Echo tool"),
                Tool("add", "Addition tool")
            ]
            mock_session.initialize = AsyncMock()
            mock_session.list_tools = AsyncMock(return_value=MagicMock(tools=mock_tools))
            mock_session.list_resources = AsyncMock(return_value=MagicMock(resources=[]))
            mock_session.list_prompts = AsyncMock(return_value=MagicMock(prompts=[]))

            # Mock tool execution
            mock_session.call_tool.return_value = MagicMock(content="Hello World")

            # Test connection
            assert await client.connect() is True
            assert client.is_connected is True
            assert len(client.get_available_tools()) == 2

            # Test tool execution
            result = await client.call_tool("echo", {"message": "Hello World"})
            assert result == "Hello World"

            # Test health check
            assert await client.health_check() is True

            # Test disconnection
            await client.disconnect()
            assert client.is_connected is False
            assert client.session is None

    async def test_connection_retry_logic(self):
        """Test connection retry and error handling."""
        config = ConfigMCPServerConfig(
            name="retry-test-server",
            transport="stdio",
            command="nonexistent_command"
        )

        client = MCPClient(config)

        # Test that connection fails gracefully
        result = await client.connect()
        assert result is False
        assert client.is_connected is False

    async def test_concurrent_operations(self):
        """Test concurrent tool calls."""
        config = ConfigMCPServerConfig(
            name="concurrent-test-server",
            transport="stdio",
            command="python",
            args=["test_server.py"]
        )

        client = MCPClient(config)

        with patch('src.infrastructure.mcp.client.stdio_client') as mock_stdio:
            # Mock session
            mock_session = AsyncMock()
            mock_stdio.return_value = (mock_session, AsyncMock(), AsyncMock())
            mock_session.initialize = AsyncMock()
            mock_session.list_tools = AsyncMock(return_value=MagicMock(tools=[]))
            mock_session.list_resources = AsyncMock(return_value=MagicMock(resources=[]))
            mock_session.list_prompts = AsyncMock(return_value=MagicMock(prompts=[]))

            # Mock concurrent tool calls
            async def mock_tool_call(name, args):
                await asyncio.sleep(0.1)  # Simulate work
                return MagicMock(content=f"Result for {name}")

            mock_session.call_tool.side_effect = mock_tool_call

            # Connect
            await client.connect()

            # Execute multiple concurrent tool calls
            tasks = [
                client.call_tool("tool1", {}),
                client.call_tool("tool2", {}),
                client.call_tool("tool3", {})
            ]

            results = await asyncio.gather(*tasks)

            assert len(results) == 3
            assert all(result is not None for result in results)

            await client.disconnect()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])