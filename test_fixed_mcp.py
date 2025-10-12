#!/usr/bin/env python3
"""
Test script with fixed MCP client using proper async context managers.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parent))

from src.infrastructure.mcp.client import MCPClient, MCPTransportType, MCPServerConfig


async def test_fixed_mcp():
    """Test MCP client with fixed async context management."""
    print("🚀 Testing Fixed MCP Client")
    print("=" * 40)

    # Create server configuration
    server_config = MCPServerConfig(
        name="filesystem-server",
        transport=MCPTransportType.STDIO,
        command="mcp-server-filesystem",
        args=["/Users/mshogin/my/agents"],
        env={}
    )

    # Create client
    client = MCPClient(server_config)

    try:
        print("📝 Step 1: Connecting to MCP server")
        success = await client.connect()

        if not success:
            print("    ❌ Connection failed")
            return False

        print("    ✅ Connected successfully")

        # Test capabilities discovery
        print("\n📝 Step 2: Discovering capabilities")
        tools = client.get_available_tools()
        resources = client.get_available_resources()
        prompts = client.get_available_prompts()

        print(f"    Tools: {len(tools)}")
        print(f"    Resources: {len(resources)}")
        print(f"    Prompts: {len(prompts)}")

        if len(tools) == 0:
            print("    ⚠️  No tools discovered")
            return False

        # Show available tools
        print(f"\n📋 Available Tools:")
        for i, tool in enumerate(tools[:5]):
            print(f"    {i+1}. {tool.name}: {tool.description[:50]}...")

        # Test tool execution
        print(f"\n📝 Step 3: Testing tool execution")

        # Test list_allowed_directories
        if any(tool.name == "list_allowed_directories" for tool in tools):
            print("    Testing list_allowed_directories...")
            result = await client.call_tool("list_allowed_directories", {})
            if result:
                print(f"    ✅ Success: {result}")
            else:
                print(f"    ❌ Failed")

        # Test list_directory
        if any(tool.name == "list_directory" for tool in tools):
            print("    Testing list_directory...")
            result = await client.call_tool("list_directory", {"path": "."})
            if result:
                print(f"    ✅ Success: Found directory listing")
            else:
                print(f"    ❌ Failed")

        print(f"\n🎉 Fixed MCP test completed successfully!")
        return True

    except Exception as e:
        print(f"❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        return False

    finally:
        # Cleanup
        try:
            await client.disconnect()
        except Exception as e:
            print(f"Cleanup error: {e}")


async def main():
    """Main test function."""
    success = await test_fixed_mcp()

    if success:
        print("\n✅ Fixed MCP test passed!")
        sys.exit(0)
    else:
        print("\n❌ Fixed MCP test failed.")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())