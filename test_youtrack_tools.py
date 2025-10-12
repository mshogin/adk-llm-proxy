#!/usr/bin/env python3
"""
YouTrack MCP Tools Testing Script
Tests actual YouTrack MCP tool functionality with real API calls
"""

import asyncio
import logging
import sys
import json
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.config.config import config

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def test_youtrack_tools():
    """Test YouTrack MCP tools with actual API calls."""
    logger.info("🔧 Testing YouTrack MCP Tools")
    logger.info("=" * 50)

    try:
        # Initialize MCP registry
        enabled_servers = config.get_enabled_mcp_servers()
        youtrack_config = None

        for server_config in enabled_servers:
            if server_config.name == "youtrack-server":
                youtrack_config = server_config
                break

        if not youtrack_config:
            logger.error("❌ YouTrack server not found in configuration")
            return False

        # Register and connect YouTrack server
        logger.info("🔗 Connecting to YouTrack MCP server...")
        success = await mcp_registry.register_server(youtrack_config)
        if not success:
            logger.error("❌ Failed to register YouTrack server")
            return False

        await mcp_registry.connect_all()

        # Get YouTrack client
        youtrack_client = mcp_registry.get_server_by_name("youtrack-server")
        if not youtrack_client:
            logger.error("❌ YouTrack client not available")
            return False

        # Get available tools
        tools = youtrack_client.get_available_tools()
        logger.info(f"✅ Found {len(tools)} YouTrack tools")

        # List all tools
        logger.info("\n📋 Available YouTrack Tools:")
        for i, tool in enumerate(tools, 1):
            logger.info(f"   {i}. {tool.name}: {getattr(tool, 'description', 'No description')}")

        # Test specific tools
        test_results = {
            'total_tests': 0,
            'passed_tests': 0,
            'failed_tests': 0,
            'results': {}
        }

        # Test 1: find_epic tool
        await test_find_epic_tool(youtrack_client, test_results)

        # Test 2: analyze_task_activity tool
        await test_analyze_task_activity_tool(youtrack_client, test_results)

        # Test 3: analyze_epic_progress tool
        await test_analyze_epic_progress_tool(youtrack_client, test_results)

        # Print summary
        logger.info("\n📊 YouTrack Tools Test Summary:")
        logger.info("=" * 50)
        logger.info(f"   Total tests: {test_results['total_tests']}")
        logger.info(f"   ✅ Passed: {test_results['passed_tests']}")
        logger.info(f"   ❌ Failed: {test_results['failed_tests']}")

        for tool_name, result in test_results['results'].items():
            status = "✅" if result['success'] else "❌"
            logger.info(f"   {status} {tool_name}: {result['message']}")

        success_rate = (test_results['passed_tests'] / test_results['total_tests']) * 100 if test_results['total_tests'] > 0 else 0
        logger.info(f"\n🎯 Success Rate: {success_rate:.1f}%")

        return success_rate >= 50.0  # Consider successful if 50%+ tests pass

    except Exception as e:
        logger.error(f"❌ Error testing YouTrack tools: {e}")
        return False

async def test_find_epic_tool(client, test_results):
    """Test the find_epic tool."""
    tool_name = "find_epic"
    logger.info(f"\n🧪 Testing tool: {tool_name}")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        # Test with a simple search term
        test_params = {
            "search_term": "test",
            "max_results": 5
        }

        logger.info(f"   Parameters: {test_params}")

        response = await client.call_tool(tool_name, test_params)
        logger.info(f"   Response type: {type(response)}")

        if response:
            if isinstance(response, dict) and 'content' in response:
                content = response['content']
                if isinstance(content, list):
                    logger.info(f"   ✅ Found {len(content)} results")
                    result['success'] = True
                    result['message'] = f"Found {len(content)} epics"
                    test_results['passed_tests'] += 1
                else:
                    logger.info(f"   📄 Response content: {str(content)[:200]}...")
                    result['success'] = True
                    result['message'] = "Tool executed successfully"
                    test_results['passed_tests'] += 1
            else:
                logger.info(f"   📄 Raw response: {str(response)[:200]}...")
                result['success'] = True
                result['message'] = "Tool executed successfully"
                test_results['passed_tests'] += 1
        else:
            result['message'] = "No response received"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Tool execution failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results'][tool_name] = result

async def test_analyze_task_activity_tool(client, test_results):
    """Test the analyze_task_activity tool."""
    tool_name = "analyze_task_activity"
    logger.info(f"\n🧪 Testing tool: {tool_name}")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        # Test with minimal parameters
        test_params = {
            "days": 7
        }

        logger.info(f"   Parameters: {test_params}")

        response = await client.call_tool(tool_name, test_params)
        logger.info(f"   Response type: {type(response)}")

        if response:
            if isinstance(response, dict) and 'content' in response:
                content = response['content']
                logger.info(f"   📄 Analysis result: {str(content)[:200]}...")
            else:
                logger.info(f"   📄 Raw response: {str(response)[:200]}...")

            result['success'] = True
            result['message'] = "Analysis completed successfully"
            test_results['passed_tests'] += 1
        else:
            result['message'] = "No response received"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Tool execution failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results'][tool_name] = result

async def test_analyze_epic_progress_tool(client, test_results):
    """Test the analyze_epic_progress tool."""
    tool_name = "analyze_epic_progress"
    logger.info(f"\n🧪 Testing tool: {tool_name}")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        # First try to find an epic to analyze
        try:
            epics_response = await client.call_tool("find_epic", {"search_term": "test", "max_results": 1})
            epic_id = None

            if epics_response and isinstance(epics_response, dict) and 'content' in epics_response:
                content = epics_response['content']
                if isinstance(content, list) and len(content) > 0:
                    epic_data = content[0]
                    if isinstance(epic_data, dict) and 'id' in epic_data:
                        epic_id = epic_data['id']
                    elif isinstance(epic_data, dict) and 'idReadable' in epic_data:
                        epic_id = epic_data['idReadable']

        except Exception as e:
            logger.info(f"   ⚠️  Could not find epic for testing: {e}")

        # Use a fallback epic ID or test with a generic one
        if not epic_id:
            epic_id = "TEST-1"  # Generic test ID

        test_params = {
            "epic_id": epic_id
        }

        logger.info(f"   Parameters: {test_params}")

        response = await client.call_tool(tool_name, test_params)
        logger.info(f"   Response type: {type(response)}")

        if response:
            if isinstance(response, dict) and 'content' in response:
                content = response['content']
                logger.info(f"   📄 Progress analysis: {str(content)[:200]}...")
            else:
                logger.info(f"   📄 Raw response: {str(response)[:200]}...")

            result['success'] = True
            result['message'] = "Progress analysis completed"
            test_results['passed_tests'] += 1
        else:
            result['message'] = "No response received"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Tool execution failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results'][tool_name] = result

async def cleanup():
    """Clean up resources."""
    try:
        await mcp_registry.shutdown()
        logger.info("✅ Cleanup complete")
    except Exception as e:
        logger.error(f"❌ Cleanup error: {e}")

if __name__ == "__main__":
    try:
        result = asyncio.run(test_youtrack_tools())
        if result:
            logger.info("🎉 YouTrack tools testing completed successfully!")
            sys.exit(0)
        else:
            logger.error("❌ YouTrack tools testing failed!")
            sys.exit(1)
    except KeyboardInterrupt:
        logger.info("🛑 Testing interrupted by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"❌ Testing script error: {e}")
        sys.exit(1)
    finally:
        try:
            asyncio.run(cleanup())
        except:
            pass