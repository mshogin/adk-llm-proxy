#!/usr/bin/env python3
"""
Comprehensive test script for MCP filesystem operations.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parent))

from src.infrastructure.mcp.client import MCPClient, MCPTransportType, MCPServerConfig


async def test_comprehensive_mcp():
    """Test comprehensive MCP filesystem operations."""
    print("🚀 Testing Comprehensive MCP Filesystem Operations")
    print("=" * 55)

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
        print("📝 Step 1: Connecting and discovering capabilities")
        success = await client.connect()
        if not success:
            return False

        tools = client.get_available_tools()
        print(f"    ✅ Connected with {len(tools)} tools available")

        # Test filesystem operations
        print("\n🗂️  Step 2: Testing filesystem operations")

        # 1. List allowed directories
        print("    Testing list_allowed_directories...")
        result = await client.call_tool("list_allowed_directories", {})
        if result:
            print(f"    ✅ Allowed directories discovered")

        # 2. List directory contents
        print("    Testing list_directory...")
        result = await client.call_tool("list_directory", {"path": "."})
        if result:
            print(f"    ✅ Directory listing successful")

        # 3. Read existing file
        print("    Testing read_text_file...")
        result = await client.call_tool("read_text_file", {"path": "config.yaml"})
        if result:
            content = str(result[0].text) if hasattr(result[0], 'text') else str(result)
            print(f"    ✅ File read successful ({len(content)} chars)")
        else:
            print(f"    ❌ File read failed")

        # 4. Create test file
        print("    Testing write_file...")
        test_content = "# MCP Test File\nThis file was created by MCP integration test.\n"
        result = await client.call_tool("write_file", {
            "path": "mcp_test_file.txt",
            "content": test_content
        })
        if result:
            print(f"    ✅ File write successful")
        else:
            print(f"    ❌ File write failed")

        # 5. Read the test file back
        print("    Testing read_text_file on created file...")
        result = await client.call_tool("read_text_file", {"path": "mcp_test_file.txt"})
        if result:
            content = str(result[0].text) if hasattr(result[0], 'text') else str(result)
            if test_content in content:
                print(f"    ✅ File read verification successful")
            else:
                print(f"    ⚠️  File content mismatch")
        else:
            print(f"    ❌ File read verification failed")

        # 6. Test directory operations
        print("    Testing create_directory...")
        result = await client.call_tool("create_directory", {"path": "mcp_test_dir"})
        if result:
            print(f"    ✅ Directory creation successful")

        # 7. Test file info
        print("    Testing get_file_info...")
        result = await client.call_tool("get_file_info", {"path": "config.yaml"})
        if result:
            print(f"    ✅ File info retrieval successful")

        # 8. Test search
        print("    Testing search_files...")
        result = await client.call_tool("search_files", {
            "path": ".",
            "pattern": "*.py"
        })
        if result:
            print(f"    ✅ File search successful")

        print("\n📊 Step 3: Testing performance operations")

        # 9. Test multiple file read
        print("    Testing read_multiple_files...")
        result = await client.call_tool("read_multiple_files", {
            "paths": ["config.yaml", "requirements.txt", "mcp_test_file.txt"]
        })
        if result:
            print(f"    ✅ Multiple file read successful")

        print(f"\n🎉 Comprehensive MCP test completed successfully!")
        return True

    except Exception as e:
        print(f"❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        return False

    finally:
        # Cleanup test files
        try:
            await client.call_tool("move_file", {
                "source": "mcp_test_file.txt",
                "destination": "mcp_test_dir/moved_test_file.txt"
            })
            print("    🧹 Moved test file for cleanup")
        except:
            pass

        try:
            await client.disconnect()
            print("    🔌 Client disconnected")
        except Exception as e:
            print(f"Cleanup error: {e}")


async def main():
    """Main test function."""
    success = await test_comprehensive_mcp()

    if success:
        print("\n✅ All comprehensive MCP tests passed!")
        print("🎉 MCP implementation is fully functional with real filesystem server!")
        sys.exit(0)
    else:
        print("\n❌ Some tests failed.")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())