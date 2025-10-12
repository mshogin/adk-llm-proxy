#!/usr/bin/env python3
"""
GitLab MCP Tools Testing with Real Projects
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

async def test_gitlab_real_projects():
    """Test GitLab MCP tools with real project IDs."""
    logger.info("🔧 Testing GitLab MCP Tools with Real Projects")
    logger.info("=" * 60)

    try:
        # Initialize MCP registry
        enabled_servers = config.get_enabled_mcp_servers()
        gitlab_config = None

        for server_config in enabled_servers:
            if server_config.name == "gitlab-server":
                gitlab_config = server_config
                break

        if not gitlab_config:
            logger.error("❌ GitLab server not found in configuration")
            return False

        # Register and connect GitLab server
        success = await mcp_registry.register_server(gitlab_config)
        await mcp_registry.connect_all()

        gitlab_client = mcp_registry.get_server_by_name("gitlab-server")
        if not gitlab_client:
            logger.error("❌ GitLab client not available")
            return False

        # Real project IDs from your GitLab instance
        real_projects = [
            {"id": "27479", "name": "pipeline", "path": "icad/st-tarifficator/cicd/pipeline"},
            {"id": "27478", "name": "deploy", "path": "icad/st-tarifficator/cicd/deploy"},
            {"id": "27448", "name": "testownscript", "path": "icad/st-tarifficator/docs/testownscript"},
            {"id": "27436", "name": "Models", "path": "wbwh/sku-saas/models"},
            {"id": "27369", "name": "ST-docs", "path": "icad/st-tarifficator/docs/st-docs"}
        ]

        # Test find_project tool
        logger.info("\n🧪 Test 1: Find Project")
        try:
            response = await gitlab_client.call_tool("find_project", {"search_term": "pipeline"})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Find project works:")
                logger.info(f"   📄 Result: {content[:400]}...")
        except Exception as e:
            logger.error(f"❌ Find project failed: {e}")

        # Test with a real project
        test_project = real_projects[0]  # Use pipeline project
        logger.info(f"\n🎯 Using test project: {test_project['name']} (ID: {test_project['id']})")

        # Test get_recent_commits
        logger.info("\n🧪 Test 2: Get Recent Commits")
        try:
            response = await gitlab_client.call_tool("get_recent_commits", {"project_id": test_project["id"]})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Get recent commits works:")
                logger.info(f"   📄 Commits: {content[:500]}...")
        except Exception as e:
            logger.error(f"❌ Get recent commits failed: {e}")

        # Test get_merge_requests
        logger.info("\n🧪 Test 3: Get Merge Requests")
        try:
            response = await gitlab_client.call_tool("get_merge_requests", {"project_id": test_project["id"]})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Get merge requests works:")
                logger.info(f"   📄 MRs: {content[:500]}...")
        except Exception as e:
            logger.error(f"❌ Get merge requests failed: {e}")

        # Test analyze_commit_messages
        logger.info("\n🧪 Test 4: Analyze Commit Messages")
        try:
            response = await gitlab_client.call_tool("analyze_commit_messages", {"project_id": test_project["id"]})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Analyze commit messages works:")
                logger.info(f"   📄 Analysis: {content[:500]}...")
        except Exception as e:
            logger.error(f"❌ Analyze commit messages failed: {e}")

        # Test analyze_branch_activity
        logger.info("\n🧪 Test 5: Analyze Branch Activity")
        try:
            response = await gitlab_client.call_tool("analyze_branch_activity", {"project_id": test_project["id"]})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Analyze branch activity works:")
                logger.info(f"   📄 Branch Analysis: {content[:500]}...")
        except Exception as e:
            logger.error(f"❌ Analyze branch activity failed: {e}")

        # Test track_developer_activity
        logger.info("\n🧪 Test 6: Track Developer Activity")
        try:
            response = await gitlab_client.call_tool("track_developer_activity", {"project_id": test_project["id"]})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Track developer activity works:")
                logger.info(f"   📄 Developer Activity: {content[:500]}...")
        except Exception as e:
            logger.error(f"❌ Track developer activity failed: {e}")

        # Test link_commits_to_tasks (YouTrack integration)
        logger.info("\n🧪 Test 7: Link Commits to Tasks (YouTrack Integration)")
        try:
            response = await gitlab_client.call_tool("link_commits_to_tasks", {"project_id": test_project["id"]})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Link commits to tasks works:")
                logger.info(f"   📄 Task Links: {content[:500]}...")
        except Exception as e:
            logger.error(f"❌ Link commits to tasks failed: {e}")

        # Test get_code_metrics
        logger.info("\n🧪 Test 8: Get Code Metrics")
        try:
            response = await gitlab_client.call_tool("get_code_metrics", {"project_id": test_project["id"]})
            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info("✅ Get code metrics works:")
                logger.info(f"   📄 Code Metrics: {content[:500]}...")
        except Exception as e:
            logger.error(f"❌ Get code metrics failed: {e}")

        logger.info("\n🎉 GitLab MCP tools testing with real data completed!")
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
        result = asyncio.run(test_gitlab_real_projects())
        if result:
            logger.info("🎯 Real project testing completed!")
        else:
            logger.error("❌ Real project testing failed!")
        sys.exit(0)
    except KeyboardInterrupt:
        logger.info("🛑 Testing interrupted")
        sys.exit(1)
    finally:
        try:
            asyncio.run(cleanup())
        except:
            pass