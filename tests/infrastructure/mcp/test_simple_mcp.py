#!/usr/bin/env python3
"""
Simple test script for MCP connection without complex async context management.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parent))

from mcp import StdioServerParameters, ClientSession
from mcp.client.stdio import stdio_client


async def test_simple_mcp():
    """Test MCP connection with proper async context management."""
    print("🚀 Testing Simple MCP Connection")
    print("=" * 40)

    server_params = StdioServerParameters(
        command="mcp-server-filesystem",
        args=["/Users/mshogin/my/agents"],
        env={}
    )

    try:
        async with stdio_client(server_params) as (read, write):
            # Use ClientSession as an async context manager to ensure proper lifecycle handling
            async with ClientSession(read, write) as session:
                print("📝 Step 1: Initializing session")
                await asyncio.wait_for(session.initialize(), timeout=10.0)
                print("    ✅ Session initialized")

                print("📝 Step 2: Listing tools")
                tools_result = await asyncio.wait_for(session.list_tools(), timeout=10.0)
                tools = tools_result.tools if hasattr(tools_result, 'tools') else []
                print(f"    ✅ Found {len(tools)} tools")

                # Show first few tools
                for i, tool in enumerate(tools[:3]):
                    print(f"    - {tool.name}: {tool.description[:50]}...")

                print("📝 Step 3: Testing tool execution")
                # Test list_allowed_directories
                result = await asyncio.wait_for(
                    session.call_tool("list_allowed_directories", {}),
                    timeout=10.0
                )
                print(f"    ✅ list_allowed_directories: {len(str(result))} chars returned")

                print("\n🎉 Simple MCP test completed successfully!")
                return True

    except asyncio.TimeoutError:
        print("❌ Test timed out")
        return False
    except Exception as e:
        print(f"❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test function."""
    success = await test_simple_mcp()

    if success:
        print("\n✅ Simple MCP test passed! MCP connection is working.")
        sys.exit(0)
    else:
        print("\n❌ Simple MCP test failed.")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
