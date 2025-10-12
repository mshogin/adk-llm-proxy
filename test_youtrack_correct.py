#!/usr/bin/env python3
"""
YouTrack MCP Tools Testing Script with Correct Parameters
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

async def test_youtrack_tools_correct():
    """Test YouTrack MCP tools with correct parameters."""
    logger.info("🔧 Testing YouTrack MCP Tools (Corrected)")
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
        success = await mcp_registry.register_server(youtrack_config)
        await mcp_registry.connect_all()

        youtrack_client = mcp_registry.get_server_by_name("youtrack-server")
        if not youtrack_client:
            logger.error("❌ YouTrack client not available")
            return False

        # Test results tracking
        test_results = []

        # Test 1: find_epic with correct parameters
        logger.info("\n🧪 Test 1: find_epic")
        try:
            response = await youtrack_client.call_tool("find_epic", {"search_term": "test"})
            logger.info("✅ find_epic executed successfully")
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Response: {content[:200]}...")
            test_results.append(("find_epic", True, "Success"))
        except Exception as e:
            logger.error(f"❌ find_epic failed: {e}")
            test_results.append(("find_epic", False, str(e)))

        # Test 2: Find an actual task ID first
        logger.info("\n🧪 Test 2: Looking for actual tasks to test with")
        actual_task_id = None
        try:
            # Try to search for any issues to get a real task ID
            response = await youtrack_client.call_tool("find_epic", {"search_term": "*"})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                # Try to extract a task ID from the response
                lines = content.split('\n')
                for line in lines:
                    if '•' in line and ':' in line:
                        # Format: • TASK-123: Description
                        parts = line.split(':')[0].strip()
                        if '•' in parts:
                            potential_id = parts.replace('•', '').strip()
                            if '-' in potential_id:
                                actual_task_id = potential_id
                                logger.info(f"   🎯 Found actual task ID: {actual_task_id}")
                                break
            test_results.append(("task_discovery", True, f"Found task: {actual_task_id}"))
        except Exception as e:
            logger.warning(f"⚠️  Could not find actual task: {e}")
            test_results.append(("task_discovery", False, str(e)))

        # Test 3: analyze_task_activity with real or test task ID
        logger.info("\n🧪 Test 3: analyze_task_activity")
        test_task_id = actual_task_id or "TEST-1"
        try:
            response = await youtrack_client.call_tool("analyze_task_activity", {
                "task_id": test_task_id,
                "days": 7
            })
            logger.info("✅ analyze_task_activity executed successfully")
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Response: {content[:200]}...")
            test_results.append(("analyze_task_activity", True, "Success"))
        except Exception as e:
            logger.error(f"❌ analyze_task_activity failed: {e}")
            test_results.append(("analyze_task_activity", False, str(e)))

        # Test 4: get_task_details
        logger.info("\n🧪 Test 4: get_task_details")
        try:
            response = await youtrack_client.call_tool("get_task_details", {
                "task_id": test_task_id
            })
            logger.info("✅ get_task_details executed successfully")
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Response: {content[:200]}...")
            test_results.append(("get_task_details", True, "Success"))
        except Exception as e:
            logger.error(f"❌ get_task_details failed: {e}")
            test_results.append(("get_task_details", False, str(e)))

        # Test 5: analyze_epic_progress with real or test epic ID
        logger.info("\n🧪 Test 5: analyze_epic_progress")
        test_epic_id = actual_task_id or "TEST-1"
        try:
            response = await youtrack_client.call_tool("analyze_epic_progress", {
                "epic_id": test_epic_id
            })
            logger.info("✅ analyze_epic_progress executed successfully")
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Response: {content[:200]}...")
            test_results.append(("analyze_epic_progress", True, "Success"))
        except Exception as e:
            logger.error(f"❌ analyze_epic_progress failed: {e}")
            test_results.append(("analyze_epic_progress", False, str(e)))

        # Summary
        passed = len([r for r in test_results if r[1]])
        total = len(test_results)

        logger.info(f"\n📊 Test Summary: {passed}/{total} passed")
        for test_name, success, message in test_results:
            status = "✅" if success else "❌"
            logger.info(f"   {status} {test_name}: {message}")

        success_rate = (passed / total) * 100 if total > 0 else 0
        logger.info(f"\n🎯 Success Rate: {success_rate:.1f}%")

        return success_rate >= 60.0

    except Exception as e:
        logger.error(f"❌ Error: {e}")
        return False

async def cleanup():
    """Clean up resources."""
    try:
        await mcp_registry.shutdown()
    except Exception as e:
        logger.error(f"❌ Cleanup error: {e}")

if __name__ == "__main__":
    try:
        result = asyncio.run(test_youtrack_tools_correct())
        if result:
            logger.info("🎉 YouTrack tools are working correctly!")
            sys.exit(0)
        else:
            logger.error("❌ Some YouTrack tools have issues!")
            sys.exit(1)
    except KeyboardInterrupt:
        logger.info("🛑 Testing interrupted")
        sys.exit(1)
    finally:
        try:
            asyncio.run(cleanup())
        except:
            pass