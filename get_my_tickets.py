#!/usr/bin/env python3
"""
Get My Assigned YouTrack Tickets
Retrieves tickets assigned to the current user from YouTrack MCP
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

async def get_my_assigned_tickets():
    """Get tickets assigned to me from YouTrack."""
    logger.info("🎫 Getting My Assigned YouTrack Tickets")
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

        # Get available tools to see what we can use
        tools = youtrack_client.get_available_tools()
        logger.info(f"📋 Available tools: {[tool.name for tool in tools]}")

        # Strategy 1: Try to use a search tool to find assigned tickets
        my_tickets = []

        # Let's try different approaches to find assigned tickets
        search_queries = [
            "assignee: me",
            "for: me",
            "assigned to: me",
            "#Unresolved assignee: me",
            "#Unresolved for: me"
        ]

        logger.info("\n🔍 Searching for tickets assigned to me...")

        for query in search_queries:
            try:
                logger.info(f"   Trying query: '{query}'")

                # Check if we have a generic search tool
                if any(tool.name in ['find_epic', 'search_issues'] for tool in tools):
                    # Try with find_epic first (even though it's for epics, it might work for general search)
                    response = await youtrack_client.call_tool("find_epic", {"search_term": query})

                    if response and isinstance(response, list) and len(response) > 0:
                        content = response[0].text if hasattr(response[0], 'text') else str(response[0])

                        if "Error" not in content and "No epics found" not in content:
                            logger.info(f"✅ Found results with query: '{query}'")
                            logger.info(f"📄 Response: {content[:300]}...")
                            my_tickets.append({
                                'query': query,
                                'results': content
                            })
                            break
                        else:
                            logger.info(f"   ⚠️  No results: {content[:100]}...")

            except Exception as e:
                logger.warning(f"   ❌ Query '{query}' failed: {e}")
                continue

        # Strategy 2: Try to get user info and then search by specific user
        logger.info("\n👤 Getting current user information...")
        try:
            # Let's check if there are any tools that might give us user info
            # or try to infer from available resources
            resources = youtrack_client.get_available_resources()
            logger.info(f"📚 Available resources: {len(resources)}")

            for resource in resources:
                logger.info(f"   Resource: {resource.uri if hasattr(resource, 'uri') else resource}")

        except Exception as e:
            logger.warning(f"⚠️  Could not get user info: {e}")

        # Strategy 3: Try different search patterns
        if not my_tickets:
            logger.info("\n🔍 Trying alternative search approaches...")

            alt_queries = [
                "*",  # Get all tickets and then filter
                "State: Open",
                "State: {Open}",
                "#Unresolved"
            ]

            for query in alt_queries:
                try:
                    logger.info(f"   Trying broad query: '{query}'")
                    response = await youtrack_client.call_tool("find_epic", {"search_term": query})

                    if response and isinstance(response, list) and len(response) > 0:
                        content = response[0].text if hasattr(response[0], 'text') else str(response[0])

                        if "Error" not in content and len(content) > 50:
                            logger.info(f"✅ Got broad results, checking for assignee info...")
                            logger.info(f"📄 Sample: {content[:400]}...")

                            # Look for any indication of assignment in the response
                            if any(word in content.lower() for word in ['assign', 'shogin', 'mikhail', 'me']):
                                my_tickets.append({
                                    'query': f'broad_search_{query}',
                                    'results': content
                                })
                            break

                except Exception as e:
                    logger.warning(f"   ❌ Broad query '{query}' failed: {e}")
                    continue

        # Display results
        if my_tickets:
            logger.info(f"\n🎯 Found {len(my_tickets)} result set(s):")
            for i, ticket_set in enumerate(my_tickets, 1):
                logger.info(f"\n--- Result Set {i} (Query: {ticket_set['query']}) ---")
                logger.info(ticket_set['results'])
                logger.info("-" * 60)
        else:
            logger.info("\n⚠️  No tickets found with the attempted queries.")
            logger.info("This could mean:")
            logger.info("   • No tickets are currently assigned to you")
            logger.info("   • The search syntax needs adjustment for your YouTrack instance")
            logger.info("   • Your user permissions may limit ticket visibility")

        # Try one more approach - get details of a known ticket to see format
        logger.info("\n🔍 Trying to get details of a known ticket to understand format...")
        try:
            # Use the task we found earlier
            response = await youtrack_client.call_tool("get_task_details", {"task_id": "PMA-16688"})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("📋 Sample ticket format:")
                logger.info(content[:500] + "...")
        except Exception as e:
            logger.warning(f"Could not get sample ticket: {e}")

        return len(my_tickets) > 0

    except Exception as e:
        logger.error(f"❌ Error getting assigned tickets: {e}")
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
            logger.info("🎉 Successfully retrieved ticket information!")
        else:
            logger.info("ℹ️  Ticket search completed (see results above)")
        sys.exit(0)
    except KeyboardInterrupt:
        logger.info("🛑 Search interrupted")
        sys.exit(1)
    finally:
        try:
            asyncio.run(cleanup())
        except:
            pass