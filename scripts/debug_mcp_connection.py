#!/usr/bin/env python3
"""
Debug MCP server connections to see why they're not working.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.config.config import config

async def debug_mcp_connections():
    """Debug MCP server connections."""
    print("🔍 Debugging MCP Server Connections")
    print("=" * 50)

    try:
        # Check configuration
        enabled_servers = config.get_enabled_mcp_servers()
        print(f"📋 Configuration shows {len(enabled_servers)} enabled servers")

        for server_config in enabled_servers:
            print(f"\n🔧 Server: {server_config.name}")
            print(f"   Enabled: {server_config.enabled}")
            print(f"   Command: {server_config.command}")
            print(f"   Args: {server_config.args}")
            if hasattr(server_config, 'env') and server_config.env:
                print(f"   Environment vars: {list(server_config.env.keys())}")

        # Try to register and connect servers
        print(f"\n🚀 Attempting to register and connect servers...")

        for server_config in enabled_servers:
            if server_config.enabled:
                print(f"\n📡 Registering {server_config.name}...")
                try:
                    success = await mcp_registry.register_server(server_config)
                    print(f"   Registration success: {success}")
                except Exception as e:
                    print(f"   Registration failed: {e}")

        print(f"\n🔗 Connecting to all registered servers...")
        try:
            await mcp_registry.connect_all()
            print("   Connection attempt completed")
        except Exception as e:
            print(f"   Connection failed: {e}")

        # Check what's actually connected
        print(f"\n📊 Server connection status:")
        for server_name in ["youtrack-server", "gitlab-server", "filesystem-server"]:
            client = mcp_registry.get_server_by_name(server_name)
            if client:
                print(f"   ✅ {server_name}: Connected")
                try:
                    tools = client.get_available_tools()
                    print(f"      Tools: {len(tools)}")
                    for tool in tools[:3]:  # Show first 3 tools
                        print(f"         - {tool.name}")
                    if len(tools) > 3:
                        print(f"         ... and {len(tools) - 3} more")
                except Exception as e:
                    print(f"      Error getting tools: {e}")
            else:
                print(f"   ❌ {server_name}: Not connected")

        # If YouTrack is connected, test it directly
        youtrack_client = mcp_registry.get_server_by_name("youtrack-server")
        if youtrack_client:
            print(f"\n🎫 Testing YouTrack directly...")
            try:
                tools = youtrack_client.get_available_tools()
                find_assigned_tool = None
                for tool in tools:
                    if 'find_assigned_tickets' in tool.name:
                        find_assigned_tool = tool
                        break

                if find_assigned_tool:
                    print(f"   ✅ Found find_assigned_tickets tool")
                    print(f"   🚀 Testing direct tool call...")

                    result = await youtrack_client.call_tool("find_assigned_tickets", {"state": "Open"})
                    if result:
                        print(f"   📊 Direct call result: {len(result) if isinstance(result, list) else 'Not a list'}")
                        if isinstance(result, list) and len(result) > 0:
                            content = result[0].text if hasattr(result[0], 'text') else str(result[0])
                            print(f"   📋 Content preview: {content[:200]}...")

                            if 'ticket' in content.lower() or 'st-' in content:
                                print("   🎫 SUCCESS: Direct YouTrack call returns ticket data! ✅")
                            else:
                                print("   ⚠️  Direct call successful but no obvious ticket data")
                    else:
                        print("   ❌ Direct call returned None")
                else:
                    print(f"   ❌ find_assigned_tickets tool not found")
                    print(f"   Available tools: {[tool.name for tool in tools]}")
            except Exception as e:
                print(f"   ❌ Direct YouTrack test failed: {e}")

    except Exception as e:
        print(f"❌ Error debugging MCP connections: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    try:
        asyncio.run(debug_mcp_connections())
    except KeyboardInterrupt:
        print("\n🛑 Debug interrupted by user")