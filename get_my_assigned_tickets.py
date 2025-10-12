#!/usr/bin/env python3
"""
Get My Assigned Tickets using the new MCP tool
"""

import asyncio
import logging
import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.config.config import config

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

async def get_my_assigned_tickets():
    """Get tickets assigned to me using the new MCP tool."""
    logger.info("🎫 Getting My Assigned Tickets via MCP")
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
        logger.info("🔗 Connecting to YouTrack...")
        success = await mcp_registry.register_server(youtrack_config)
        await mcp_registry.connect_all()

        youtrack_client = mcp_registry.get_server_by_name("youtrack-server")
        if not youtrack_client:
            logger.error("❌ YouTrack client not available")
            return False

        logger.info("✅ Connected to YouTrack successfully!")

        # Get available tools to confirm our new tool is there
        tools = youtrack_client.get_available_tools()
        tool_names = [tool.name for tool in tools]
        logger.info(f"📋 Available tools: {tool_names}")

        if 'find_assigned_tickets' not in tool_names:
            logger.error("❌ find_assigned_tickets tool not found!")
            return False

        # Test 1: Get open tickets assigned to me
        logger.info("\n🔍 Test 1: Getting open tickets assigned to me...")
        try:
            response = await youtrack_client.call_tool("find_assigned_tickets", {"state": "Open"})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Open tickets:")
                logger.info(content)
            else:
                logger.info("⚠️  No response received")
        except Exception as e:
            logger.error(f"❌ Error getting open tickets: {e}")

        # Test 2: Get all tickets assigned to me (any state)
        logger.info("\n🔍 Test 2: Getting all tickets assigned to me (any state)...")
        try:
            response = await youtrack_client.call_tool("find_assigned_tickets", {"state": "All"})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ All tickets:")
                logger.info(content)
            else:
                logger.info("⚠️  No response received")
        except Exception as e:
            logger.error(f"❌ Error getting all tickets: {e}")

        return True

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
        result = asyncio.run(get_my_assigned_tickets())
        if result:
            logger.info("🎉 Successfully retrieved assigned tickets!")
        else:
            logger.error("❌ Failed to retrieve assigned tickets")
        sys.exit(0)
    except KeyboardInterrupt:
        logger.info("🛑 Interrupted")
        sys.exit(1)
    finally:
        try:
            asyncio.run(cleanup())
        except:
            pass