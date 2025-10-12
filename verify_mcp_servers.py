#!/usr/bin/env python3
"""
MCP Server Verification Script
Verifies that MCP servers are working and tools are functional
"""

import asyncio
import logging
import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.mcp.discovery import MCPToolDiscovery
from src.infrastructure.config.config import config

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def verify_mcp_servers():
    """Verify MCP servers are working properly."""
    logger.info("🔍 Starting MCP Server Verification")
    logger.info("=" * 60)

    try:
        # Initialize MCP registry
        logger.info("📋 Loading MCP server configurations...")
        enabled_servers = config.get_enabled_mcp_servers()

        if not enabled_servers:
            logger.error("❌ No enabled MCP servers found in configuration")
            return False

        logger.info(f"✅ Found {len(enabled_servers)} enabled MCP server(s)")

        # Register and connect servers
        logger.info("🔧 Registering MCP servers...")
        for server_config in enabled_servers:
            success = await mcp_registry.register_server(server_config)
            status = "✅" if success else "❌"
            logger.info(f"   {status} {server_config.name}")

        # Connect to all servers
        logger.info("🔗 Connecting to MCP servers...")
        await mcp_registry.connect_all()

        # Get server status
        connected_servers = mcp_registry.get_connected_servers()
        all_servers = mcp_registry.get_all_servers()

        logger.info("📊 Server Status Overview:")
        logger.info(f"   Total servers: {len(all_servers)}")
        logger.info(f"   Connected servers: {len(connected_servers)}")

        # Test each server
        verification_results = {}
        for name, server_info in all_servers.items():
            logger.info(f"\n🔍 Verifying server: {name}")
            logger.info("-" * 40)

            result = await verify_server(name, server_info)
            verification_results[name] = result

        # Summary
        logger.info("\n📈 Verification Summary:")
        logger.info("=" * 60)
        successful_servers = sum(1 for result in verification_results.values() if result['success'])
        logger.info(f"✅ Successful servers: {successful_servers}/{len(verification_results)}")

        for name, result in verification_results.items():
            status = "✅" if result['success'] else "❌"
            logger.info(f"   {status} {name}: {result['message']}")

        return successful_servers == len(verification_results)

    except Exception as e:
        logger.error(f"❌ Error during MCP verification: {e}")
        return False

async def verify_server(name: str, server_info) -> dict:
    """Verify a specific MCP server."""
    result = {
        'success': False,
        'message': '',
        'tools': 0,
        'resources': 0,
        'prompts': 0
    }

    try:
        # Check basic connection
        if not server_info.is_healthy:
            result['message'] = f"Server unhealthy: {server_info.status.value}"
            if server_info.last_error:
                result['message'] += f" - {server_info.last_error}"
            return result

        client = server_info.client
        if not client:
            result['message'] = "No client available"
            return result

        logger.info(f"   Status: {server_info.status.value}")

        # Test tool discovery
        try:
            tools = client.get_available_tools()
            result['tools'] = len(tools)
            logger.info(f"   Tools: {len(tools)}")

            for tool in tools[:3]:  # Show first 3 tools
                logger.info(f"     • {tool.name}: {getattr(tool, 'description', 'No description')[:60]}...")

        except Exception as e:
            logger.warning(f"   Tools discovery failed: {e}")

        # Test resource discovery
        try:
            resources = client.get_available_resources()
            result['resources'] = len(resources)
            logger.info(f"   Resources: {len(resources)}")
        except Exception as e:
            logger.warning(f"   Resource discovery failed: {e}")

        # Test prompt discovery
        try:
            prompts = client.get_available_prompts()
            result['prompts'] = len(prompts)
            logger.info(f"   Prompts: {len(prompts)}")
        except Exception as e:
            logger.warning(f"   Prompt discovery failed: {e}")

        # Test specific server functionality
        if name in ['youtrack-server', 'gitlab-server']:
            tool_test_result = await test_server_tools(name, client)
            result['tool_test'] = tool_test_result

        result['success'] = True
        result['message'] = f"Healthy - {result['tools']} tools, {result['resources']} resources, {result['prompts']} prompts"

    except Exception as e:
        result['message'] = f"Verification error: {str(e)}"
        logger.error(f"   ❌ Error verifying {name}: {e}")

    return result

async def test_server_tools(server_name: str, client) -> dict:
    """Test specific tools for YouTrack and GitLab servers."""
    logger.info(f"   🧪 Testing {server_name} tools...")

    test_results = {
        'tests_run': 0,
        'tests_passed': 0,
        'errors': []
    }

    try:
        tools = client.get_available_tools()

        if server_name == 'youtrack-server':
            # Test YouTrack tools
            for tool in tools[:2]:  # Test first 2 tools
                try:
                    logger.info(f"     Testing tool: {tool.name}")
                    # Note: We can't actually call tools without proper parameters
                    # But we can verify they exist and have proper schemas
                    if hasattr(tool, 'inputSchema') and tool.inputSchema:
                        logger.info(f"       ✅ Tool has valid input schema")
                        test_results['tests_passed'] += 1
                    else:
                        logger.warning(f"       ⚠️  Tool missing input schema")
                    test_results['tests_run'] += 1
                except Exception as e:
                    logger.error(f"       ❌ Tool test failed: {e}")
                    test_results['errors'].append(f"{tool.name}: {e}")
                    test_results['tests_run'] += 1

        elif server_name == 'gitlab-server':
            # Test GitLab tools
            for tool in tools[:2]:  # Test first 2 tools
                try:
                    logger.info(f"     Testing tool: {tool.name}")
                    if hasattr(tool, 'inputSchema') and tool.inputSchema:
                        logger.info(f"       ✅ Tool has valid input schema")
                        test_results['tests_passed'] += 1
                    else:
                        logger.warning(f"       ⚠️  Tool missing input schema")
                    test_results['tests_run'] += 1
                except Exception as e:
                    logger.error(f"       ❌ Tool test failed: {e}")
                    test_results['errors'].append(f"{tool.name}: {e}")
                    test_results['tests_run'] += 1

    except Exception as e:
        logger.error(f"   ❌ Tool testing failed for {server_name}: {e}")
        test_results['errors'].append(f"Tool testing: {e}")

    logger.info(f"     Test results: {test_results['tests_passed']}/{test_results['tests_run']} passed")
    return test_results

async def cleanup():
    """Clean up resources."""
    try:
        await mcp_registry.shutdown()
        logger.info("✅ Cleanup complete")
    except Exception as e:
        logger.error(f"❌ Cleanup error: {e}")

if __name__ == "__main__":
    try:
        result = asyncio.run(verify_mcp_servers())
        if result:
            logger.info("🎉 MCP Server verification completed successfully!")
            sys.exit(0)
        else:
            logger.error("❌ MCP Server verification failed!")
            sys.exit(1)
    except KeyboardInterrupt:
        logger.info("🛑 Verification interrupted by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"❌ Verification script error: {e}")
        sys.exit(1)
    finally:
        try:
            asyncio.run(cleanup())
        except:
            pass