#!/usr/bin/env python3
"""
Direct test script for MCP client without configuration system.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parent))

from src.infrastructure.mcp.client import MCPClient, MCPTransportType, MCPServerConfig


async def test_direct_mcp_connection():
    """Test MCP client directly with filesystem server."""
    print("🚀 Testing Direct MCP Client Connection")
    print("=" * 50)

    # Create server configuration manually
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
        # Test connection
        print("📝 Step 1: Connecting to MCP filesystem server")
        success = await client.connect()

        if success:
            print("    ✅ Connection successful")

            # Get available tools
            tools = client.get_available_tools()
            resources = client.get_available_resources()
            prompts = client.get_available_prompts()

            print(f"\n🔍 Step 2: Discovery Results")
            print(f"    Tools: {len(tools)}")
            print(f"    Resources: {len(resources)}")
            print(f"    Prompts: {len(prompts)}")

            # Show first few tools
            print(f"\n📋 Available Tools:")
            for i, tool in enumerate(tools[:5]):
                print(f"    {i+1}. {tool.name}: {tool.description[:60]}...")

            # Test tool execution
            print(f"\n⚡ Step 3: Testing tool execution")

            # Test list_allowed_directories
            result = await client.call_tool("list_allowed_directories", {})
            if result:
                print(f"    ✅ list_allowed_directories: {result}")
            else:
                print(f"    ❌ list_allowed_directories failed")

            # Test list_directory
            result = await client.call_tool("list_directory", {"path": "."})
            if result:
                print(f"    ✅ list_directory: Found {len(str(result).split('\\n'))} entries")
            else:
                print(f"    ❌ list_directory failed")

            # Test read_text_file
            result = await client.call_tool("read_text_file", {"path": "README.md"})
            if result:
                content_preview = str(result)[:100] + "..." if len(str(result)) > 100 else str(result)
                print(f"    ✅ read_text_file: {content_preview}")
            else:
                print(f"    ❌ read_text_file failed")

            print(f"\n🎉 Direct MCP test completed successfully!")
            return True

        else:
            print("    ❌ Connection failed")
            return False

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
    success = await test_direct_mcp_connection()

    if success:
        print("\n✅ Direct MCP test passed! Client implementation is working.")
        sys.exit(0)
    else:
        print("\n❌ Direct MCP test failed. Check the output above for details.")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())