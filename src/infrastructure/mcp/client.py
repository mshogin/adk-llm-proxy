"""
MCP Client implementation for connecting to and communicating with MCP servers.
"""
import asyncio
import logging
from typing import Dict, List, Optional, Any, Union
from enum import Enum
import json

try:
    from mcp import ClientSession, StdioServerParameters
    from mcp.client.stdio import stdio_client
    from mcp.client.sse import sse_client
    from mcp.types import Tool, Resource, Prompt
except ImportError as e:
    logging.warning(f"MCP library not available: {e}")
    # Define minimal interfaces for development
    class ClientSession:
        pass
    class StdioServerParameters:
        pass

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


class MCPTransportType(Enum):
    """Supported MCP transport types."""
    STDIO = "stdio"
    SSE = "sse"
    WEBSOCKET = "websocket"


class MCPServerConfig:
    """Configuration for an MCP server connection."""

    def __init__(
        self,
        name: str,
        transport: MCPTransportType,
        command: Optional[str] = None,
        args: Optional[List[str]] = None,
        env: Optional[Dict[str, str]] = None,
        url: Optional[str] = None,
        headers: Optional[Dict[str, str]] = None
    ):
        self.name = name
        self.transport = transport
        self.command = command
        self.args = args or []
        self.env = env or {}
        self.url = url
        self.headers = headers or {}

    def validate(self) -> bool:
        """Validate the server configuration."""
        if self.transport == MCPTransportType.STDIO:
            return bool(self.command)
        elif self.transport in [MCPTransportType.SSE, MCPTransportType.WEBSOCKET]:
            return bool(self.url)
        return False


class MCPClient:
    """
    MCP Client for connecting to and communicating with MCP servers.

    Supports multiple transport types: stdio, SSE, and WebSocket.
    """

    def __init__(self, config: MCPServerConfig):
        self.config = config
        self.session: Optional[ClientSession] = None
        self.is_connected = False
        self._tools: List[Tool] = []
        self._resources: List[Resource] = []
        self._prompts: List[Prompt] = []
        self._stdio_context = None  # For managing stdio connection context
        self._session_context = None  # For managing ClientSession context
        self._connection_task = None  # For managing the connection task
        self.logger = logging.getLogger(f"MCPClient.{config.name}")

    async def connect(self) -> bool:
        """
        Connect to the MCP server based on the configured transport.

        Returns:
            bool: True if connection successful, False otherwise
        """
        try:
            if not self.config.validate():
                self.logger.error(f"Invalid configuration for server {self.config.name}")
                return False

            if self.config.transport == MCPTransportType.STDIO:
                return await self._connect_stdio()
            elif self.config.transport == MCPTransportType.SSE:
                return await self._connect_sse()
            elif self.config.transport == MCPTransportType.WEBSOCKET:
                self.logger.error("WebSocket transport not available in current MCP version")
                return False

        except Exception as e:
            self.logger.error(f"Failed to connect to MCP server {self.config.name}: {e}")
            return False

        return False

    async def _connect_stdio(self) -> bool:
        """Connect via stdio transport."""
        try:
            server_params = StdioServerParameters(
                command=self.config.command,
                args=self.config.args,
                env=self.config.env
            )

            # Create and enter stdio context
            self._stdio_context = stdio_client(server_params)
            read_stream, write_stream = await self._stdio_context.__aenter__()

            # Create and enter ClientSession context
            self._session_context = ClientSession(read_stream, write_stream)
            self.session = await self._session_context.__aenter__()

            self.is_connected = True
            self.logger.info(f"Connected to MCP server {self.config.name} via stdio")

            # Initialize session capabilities
            await self._initialize_session()
            return True

        except Exception as e:
            self.logger.error(f"Failed to connect via stdio: {e}")
            # Clean up on failure
            await self._cleanup_contexts()
            return False

    async def _connect_sse(self) -> bool:
        """Connect via SSE transport."""
        try:
            self.session, write, read = await sse_client(
                self.config.url,
                headers=self.config.headers
            )
            self.is_connected = True
            self.logger.info(f"Connected to MCP server {self.config.name} via SSE")

            await self._initialize_session()
            return True

        except Exception as e:
            self.logger.error(f"Failed to connect via SSE: {e}")
            return False


    async def _initialize_session(self):
        """Initialize the MCP session and discover capabilities."""
        if not self.session:
            return

        try:
            # Initialize the session
            await self.session.initialize()

            # Discover available tools, resources, and prompts
            await self._discover_capabilities()

        except Exception as e:
            self.logger.error(f"Failed to initialize session: {e}")

    async def _discover_capabilities(self):
        """Discover tools, resources, and prompts available on the server."""
        if not self.session:
            return

        try:
            # List available tools
            tools_result = await self.session.list_tools()
            self._tools = tools_result.tools if hasattr(tools_result, 'tools') else []
            self.logger.info(f"Discovered {len(self._tools)} tools")
        except Exception as e:
            self.logger.warning(f"Failed to discover tools: {e}")
            self._tools = []

        try:
            # List available resources
            resources_result = await self.session.list_resources()
            self._resources = resources_result.resources if hasattr(resources_result, 'resources') else []
            self.logger.info(f"Discovered {len(self._resources)} resources")
        except Exception as e:
            self.logger.debug(f"Failed to discover resources (optional): {e}")
            self._resources = []

        try:
            # List available prompts
            prompts_result = await self.session.list_prompts()
            self._prompts = prompts_result.prompts if hasattr(prompts_result, 'prompts') else []
            self.logger.info(f"Discovered {len(self._prompts)} prompts")
        except Exception as e:
            self.logger.debug(f"Failed to discover prompts (optional): {e}")
            self._prompts = []

    async def _cleanup_contexts(self):
        """Clean up async contexts safely."""
        # Close session context first with timeout
        if self._session_context:
            try:
                await asyncio.wait_for(
                    self._session_context.__aexit__(None, None, None),
                    timeout=1.0
                )
            except (asyncio.TimeoutError, asyncio.CancelledError, RuntimeError) as e:
                # These errors are expected during shutdown
                self.logger.debug(f"Session context cleanup (expected during shutdown): {type(e).__name__}")
            except Exception as e:
                self.logger.debug(f"Session context cleanup: {e}")
            finally:
                self._session_context = None
                self.session = None

        # Close stdio context last with timeout
        if self._stdio_context:
            try:
                await asyncio.wait_for(
                    self._stdio_context.__aexit__(None, None, None),
                    timeout=1.0
                )
            except (asyncio.TimeoutError, asyncio.CancelledError, RuntimeError) as e:
                # These errors are expected during shutdown
                self.logger.debug(f"Stdio context cleanup (expected during shutdown): {type(e).__name__}")
            except Exception as e:
                self.logger.debug(f"Stdio context cleanup: {e}")
            finally:
                self._stdio_context = None

    async def disconnect(self):
        """Disconnect from the MCP server."""
        if not self.is_connected:
            return

        self.is_connected = False

        # Properly clean up async contexts
        await self._cleanup_contexts()

        self.logger.info(f"Disconnected from MCP server {self.config.name}")

    async def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Call a tool on the MCP server.

        Args:
            tool_name: Name of the tool to call
            arguments: Arguments to pass to the tool

        Returns:
            Tool execution result or None if failed
        """
        if not self.is_connected or not self.session:
            self.logger.error("Not connected to MCP server")
            return None

        try:
            result = await self.session.call_tool(tool_name, arguments)
            self.logger.debug(f"Tool {tool_name} executed successfully")
            return result.content if hasattr(result, 'content') else result

        except Exception as e:
            self.logger.error(f"Failed to call tool {tool_name}: {e}")
            return None

    async def get_resource(self, uri: str) -> Optional[Any]:
        """
        Get a resource from the MCP server.

        Args:
            uri: URI of the resource to retrieve

        Returns:
            Resource content or None if failed
        """
        if not self.is_connected or not self.session:
            self.logger.error("Not connected to MCP server")
            return None

        try:
            result = await self.session.read_resource(uri)
            self.logger.debug(f"Resource {uri} retrieved successfully")
            return result.contents if hasattr(result, 'contents') else result

        except Exception as e:
            self.logger.error(f"Failed to get resource {uri}: {e}")
            return None

    async def get_prompt(self, name: str, arguments: Optional[Dict[str, Any]] = None) -> Optional[Any]:
        """
        Get a prompt from the MCP server.

        Args:
            name: Name of the prompt
            arguments: Optional arguments for the prompt

        Returns:
            Prompt content or None if failed
        """
        if not self.is_connected or not self.session:
            self.logger.error("Not connected to MCP server")
            return None

        try:
            result = await self.session.get_prompt(name, arguments or {})
            self.logger.debug(f"Prompt {name} retrieved successfully")
            return result.messages if hasattr(result, 'messages') else result

        except Exception as e:
            self.logger.error(f"Failed to get prompt {name}: {e}")
            return None

    def get_available_tools(self) -> List[Tool]:
        """Get list of available tools."""
        return self._tools.copy()

    def get_available_resources(self) -> List[Resource]:
        """Get list of available resources."""
        return self._resources.copy()

    def get_available_prompts(self) -> List[Prompt]:
        """Get list of available prompts."""
        return self._prompts.copy()

    def is_tool_available(self, tool_name: str) -> bool:
        """Check if a specific tool is available."""
        return any(tool.name == tool_name for tool in self._tools)

    async def health_check(self) -> bool:
        """
        Perform a health check on the MCP server connection.

        Returns:
            bool: True if healthy, False otherwise
        """
        if not self.is_connected or not self.session:
            return False

        try:
            # Try to list tools as a simple health check
            await self.session.list_tools()
            return True
        except Exception as e:
            self.logger.warning(f"Health check failed: {e}")
            return False