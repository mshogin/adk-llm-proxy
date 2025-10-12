"""
MCP Server Registry for managing active connections and health monitoring.
"""
import asyncio
import logging
from typing import Dict, List, Optional, Set
from datetime import datetime, timedelta
from enum import Enum

from .client import MCPClient, MCPTransportType
from .client import MCPServerConfig


class MCPServerStatus(Enum):
    """Status of an MCP server connection."""
    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    ERROR = "error"
    DISABLED = "disabled"


class MCPServerInfo:
    """Information about an MCP server in the registry."""

    def __init__(self, config: MCPServerConfig):
        self.config = config
        self.client: Optional[MCPClient] = None
        self.status = MCPServerStatus.DISCONNECTED
        self.last_health_check: Optional[datetime] = None
        self.last_error: Optional[str] = None
        self.connection_attempts = 0
        self.last_connection_attempt: Optional[datetime] = None
        self.tools_count = 0
        self.resources_count = 0
        self.prompts_count = 0

    @property
    def is_healthy(self) -> bool:
        """Check if the server is healthy."""
        return (
            self.status == MCPServerStatus.CONNECTED and
            self.client is not None and
            self.client.is_connected
        )

    @property
    def should_retry_connection(self) -> bool:
        """Check if connection should be retried."""
        if not self.config.enabled:
            return False

        if self.connection_attempts >= self.config.retry_attempts:
            return False

        if self.last_connection_attempt is None:
            return True

        retry_time = self.last_connection_attempt + timedelta(seconds=self.config.retry_delay)
        return datetime.now() >= retry_time

    def update_capabilities(self):
        """Update capability counts from the client."""
        if self.client:
            self.tools_count = len(self.client.get_available_tools())
            self.resources_count = len(self.client.get_available_resources())
            self.prompts_count = len(self.client.get_available_prompts())


class MCPServerRegistry:
    """
    Registry for managing MCP server connections, health checking, and lifecycle.
    """

    def __init__(self):
        self.logger = logging.getLogger("MCPServerRegistry")
        self._servers: Dict[str, MCPServerInfo] = {}
        self._health_check_task: Optional[asyncio.Task] = None
        self._health_check_interval = 60.0
        self._shutdown = False

    async def register_server(self, config: MCPServerConfig) -> bool:
        """
        Register a new MCP server configuration.

        Args:
            config: Server configuration

        Returns:
            bool: True if registration successful
        """
        try:
            # Validate configuration
            config.validate()

            # Create server info
            server_info = MCPServerInfo(config)

            if not config.enabled:
                server_info.status = MCPServerStatus.DISABLED
                self.logger.info(f"Registered disabled MCP server: {config.name}")
            else:
                self.logger.info(f"Registered MCP server: {config.name}")

            self._servers[config.name] = server_info

            # Try to connect if enabled
            if config.enabled:
                await self._connect_server(config.name)

            return True

        except Exception as e:
            self.logger.error(f"Failed to register MCP server {config.name}: {e}")
            return False

    async def unregister_server(self, name: str) -> bool:
        """
        Unregister and disconnect an MCP server.

        Args:
            name: Server name

        Returns:
            bool: True if unregistration successful
        """
        server_info = self._servers.get(name)
        if not server_info:
            return False

        try:
            # Disconnect if connected
            await self._disconnect_server(name)

            # Remove from registry
            del self._servers[name]
            self.logger.info(f"Unregistered MCP server: {name}")
            return True

        except Exception as e:
            self.logger.error(f"Failed to unregister MCP server {name}: {e}")
            return False

    async def connect_all(self):
        """Connect to all enabled servers."""
        connect_tasks = []

        for name, server_info in self._servers.items():
            if server_info.config.enabled and server_info.status == MCPServerStatus.DISCONNECTED:
                connect_tasks.append(self._connect_server(name))

        if connect_tasks:
            self.logger.info(f"Connecting to {len(connect_tasks)} MCP servers...")
            results = await asyncio.gather(*connect_tasks, return_exceptions=True)

            success_count = sum(1 for result in results if isinstance(result, bool) and result)
            self.logger.info(f"Successfully connected to {success_count}/{len(connect_tasks)} MCP servers")

    async def disconnect_all(self):
        """Disconnect from all servers."""
        disconnect_tasks = []

        for name, server_info in self._servers.items():
            if server_info.status == MCPServerStatus.CONNECTED:
                disconnect_tasks.append(self._disconnect_server(name))

        if disconnect_tasks:
            self.logger.info(f"Disconnecting from {len(disconnect_tasks)} MCP servers...")
            await asyncio.gather(*disconnect_tasks, return_exceptions=True)

    async def _connect_server(self, name: str) -> bool:
        """Connect to a specific server."""
        server_info = self._servers.get(name)
        if not server_info:
            return False

        if not server_info.config.enabled:
            server_info.status = MCPServerStatus.DISABLED
            return False

        try:
            server_info.status = MCPServerStatus.CONNECTING
            server_info.last_connection_attempt = datetime.now()
            server_info.connection_attempts += 1

            # Create client
            server_info.client = MCPClient(server_info.config)

            # Attempt connection
            success = await server_info.client.connect()

            if success:
                server_info.status = MCPServerStatus.CONNECTED
                server_info.last_error = None
                server_info.connection_attempts = 0  # Reset on successful connection
                server_info.update_capabilities()

                self.logger.info(
                    f"Connected to MCP server '{name}' "
                    f"({server_info.tools_count} tools, "
                    f"{server_info.resources_count} resources, "
                    f"{server_info.prompts_count} prompts)"
                )
                return True
            else:
                raise Exception("Connection failed")

        except Exception as e:
            error_msg = str(e)
            server_info.status = MCPServerStatus.ERROR
            server_info.last_error = error_msg
            server_info.client = None

            self.logger.error(f"Failed to connect to MCP server '{name}': {error_msg}")
            return False

    async def _disconnect_server(self, name: str) -> bool:
        """Disconnect from a specific server."""
        server_info = self._servers.get(name)
        if not server_info:
            return False

        try:
            if server_info.client:
                await server_info.client.disconnect()

            server_info.status = MCPServerStatus.DISCONNECTED
            server_info.client = None
            server_info.tools_count = 0
            server_info.resources_count = 0
            server_info.prompts_count = 0

            self.logger.info(f"Disconnected from MCP server: {name}")
            return True

        except Exception as e:
            self.logger.error(f"Error disconnecting from MCP server {name}: {e}")
            return False

    async def start_health_monitoring(self, interval: float = 60.0):
        """Start health monitoring for all servers."""
        if self._health_check_task and not self._health_check_task.done():
            return

        self._health_check_interval = interval
        self._shutdown = False
        self._health_check_task = asyncio.create_task(self._health_check_loop())
        self.logger.info(f"Started MCP health monitoring (interval: {interval}s)")

    async def stop_health_monitoring(self):
        """Stop health monitoring."""
        self._shutdown = True

        if self._health_check_task and not self._health_check_task.done():
            self._health_check_task.cancel()
            try:
                await self._health_check_task
            except asyncio.CancelledError:
                pass

        self.logger.info("Stopped MCP health monitoring")

    async def _health_check_loop(self):
        """Main health check loop."""
        while not self._shutdown:
            try:
                await self._perform_health_checks()
                await asyncio.sleep(self._health_check_interval)
            except asyncio.CancelledError:
                break
            except Exception as e:
                self.logger.error(f"Error in health check loop: {e}")
                await asyncio.sleep(5)  # Short delay before retrying

    async def _perform_health_checks(self):
        """Perform health checks on all servers."""
        check_tasks = []

        for name, server_info in self._servers.items():
            if server_info.config.enabled:
                check_tasks.append(self._health_check_server(name))

        if check_tasks:
            await asyncio.gather(*check_tasks, return_exceptions=True)

    async def _health_check_server(self, name: str):
        """Perform health check on a specific server."""
        server_info = self._servers.get(name)
        if not server_info:
            return

        try:
            if server_info.status == MCPServerStatus.CONNECTED and server_info.client:
                # Perform health check
                is_healthy = await server_info.client.health_check()
                server_info.last_health_check = datetime.now()

                if not is_healthy:
                    self.logger.warning(f"MCP server '{name}' failed health check")
                    server_info.status = MCPServerStatus.ERROR
                    server_info.last_error = "Health check failed"

                    # Try to reconnect
                    await self._disconnect_server(name)
                    if server_info.should_retry_connection:
                        await self._connect_server(name)

            elif (server_info.status in [MCPServerStatus.DISCONNECTED, MCPServerStatus.ERROR] and
                  server_info.should_retry_connection):
                # Try to reconnect
                await self._connect_server(name)

        except Exception as e:
            self.logger.error(f"Error during health check for '{name}': {e}")

    def get_server_info(self, name: str) -> Optional[MCPServerInfo]:
        """Get information about a server."""
        return self._servers.get(name)

    def get_all_servers(self) -> Dict[str, MCPServerInfo]:
        """Get information about all servers."""
        return self._servers.copy()

    def get_connected_servers(self) -> Dict[str, MCPServerInfo]:
        """Get all connected servers."""
        return {
            name: info for name, info in self._servers.items()
            if info.status == MCPServerStatus.CONNECTED
        }

    def get_server_by_name(self, name: str) -> Optional[MCPClient]:
        """Get MCP client for a specific server."""
        server_info = self._servers.get(name)
        if server_info and server_info.is_healthy:
            return server_info.client
        return None

    def get_all_available_tools(self) -> Dict[str, List[str]]:
        """Get all available tools from all connected servers."""
        tools = {}
        for name, server_info in self._servers.items():
            if server_info.is_healthy and server_info.client:
                server_tools = [tool.name for tool in server_info.client.get_available_tools()]
                if server_tools:
                    tools[name] = server_tools
        return tools

    def find_servers_with_tool(self, tool_name: str) -> List[str]:
        """Find all servers that provide a specific tool."""
        servers = []
        for name, server_info in self._servers.items():
            if (server_info.is_healthy and
                server_info.client and
                server_info.client.is_tool_available(tool_name)):
                servers.append(name)
        return servers

    def get_registry_stats(self) -> Dict[str, any]:
        """Get registry statistics."""
        total_servers = len(self._servers)
        connected_servers = len(self.get_connected_servers())
        total_tools = sum(info.tools_count for info in self._servers.values())
        total_resources = sum(info.resources_count for info in self._servers.values())
        total_prompts = sum(info.prompts_count for info in self._servers.values())

        status_counts = {}
        for status in MCPServerStatus:
            count = sum(1 for info in self._servers.values() if info.status == status)
            status_counts[status.value] = count

        return {
            "total_servers": total_servers,
            "connected_servers": connected_servers,
            "total_tools": total_tools,
            "total_resources": total_resources,
            "total_prompts": total_prompts,
            "status_counts": status_counts,
            "health_monitoring_active": self._health_check_task is not None and not self._health_check_task.done()
        }

    async def shutdown(self):
        """Shutdown the registry and all connections."""
        self.logger.info("Shutting down MCP registry...")

        # Stop health monitoring
        await self.stop_health_monitoring()

        # Disconnect all servers
        await self.disconnect_all()

        self.logger.info("MCP registry shutdown complete")


# Global registry instance
mcp_registry = MCPServerRegistry()