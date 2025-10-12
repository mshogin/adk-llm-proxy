#!/usr/bin/env python3
"""
GitLab MCP Tools Testing Script
Tests actual GitLab MCP tool functionality with real API calls
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

async def test_gitlab_tools():
    """Test GitLab MCP tools with actual API calls."""
    logger.info("🔧 Testing GitLab MCP Tools")
    logger.info("=" * 50)

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
        logger.info("🔗 Connecting to GitLab MCP server...")
        success = await mcp_registry.register_server(gitlab_config)
        if not success:
            logger.error("❌ Failed to register GitLab server")
            return False

        await mcp_registry.connect_all()

        # Get GitLab client
        gitlab_client = mcp_registry.get_server_by_name("gitlab-server")
        if not gitlab_client:
            logger.error("❌ GitLab client not available")
            return False

        # Get available tools
        tools = gitlab_client.get_available_tools()
        logger.info(f"✅ Found {len(tools)} GitLab tools")

        # List all tools
        logger.info("\n📋 Available GitLab Tools:")
        for i, tool in enumerate(tools, 1):
            logger.info(f"   {i}. {tool.name}: {getattr(tool, 'description', 'No description')}")

        # Test results tracking
        test_results = {
            'total_tests': 0,
            'passed_tests': 0,
            'failed_tests': 0,
            'results': {}
        }

        # Test 1: List projects
        await test_list_projects(gitlab_client, test_results)

        # Test 2: Search projects
        await test_search_projects(gitlab_client, test_results)

        # Test 3: Get project details (if we found a project)
        await test_project_details(gitlab_client, test_results)

        # Test 4: Analyze commit messages
        await test_analyze_commits(gitlab_client, test_results)

        # Test 5: Analyze branch activity
        await test_analyze_branches(gitlab_client, test_results)

        # Test 6: Analyze code complexity
        await test_analyze_complexity(gitlab_client, test_results)

        # Print summary
        logger.info("\n📊 GitLab Tools Test Summary:")
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
        logger.error(f"❌ Error testing GitLab tools: {e}")
        return False

async def test_list_projects(client, test_results):
    """Test the list projects functionality."""
    logger.info(f"\n🧪 Testing: List Projects")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        # Check if we have a list_projects tool
        tools = client.get_available_tools()
        list_tools = [t for t in tools if 'list' in t.name.lower() or 'project' in t.name.lower()]

        if list_tools:
            tool_name = list_tools[0].name
            logger.info(f"   Using tool: {tool_name}")

            response = await client.call_tool(tool_name, {})
            logger.info(f"   Response type: {type(response)}")

            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Projects found: {len(content.splitlines())} lines")
                logger.info(f"   📄 Sample: {content[:200]}...")

                result['success'] = True
                result['message'] = f"Listed projects successfully"
                test_results['passed_tests'] += 1
            else:
                result['message'] = "No projects data received"
                test_results['failed_tests'] += 1
        else:
            result['message'] = "No list projects tool found"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ List projects failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results']['list_projects'] = result

async def test_search_projects(client, test_results):
    """Test the search projects functionality."""
    logger.info(f"\n🧪 Testing: Search Projects")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        # Check if we have a search_projects tool
        tools = client.get_available_tools()
        search_tools = [t for t in tools if 'search' in t.name.lower() and 'project' in t.name.lower()]

        if search_tools:
            tool_name = search_tools[0].name
            logger.info(f"   Using tool: {tool_name}")

            # Try searching for common terms
            search_terms = ["finance", "api", "web"]
            for term in search_terms:
                try:
                    response = await client.call_tool(tool_name, {"query": term})

                    if response and isinstance(response, list) and len(response) > 0:
                        content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                        if "Error" not in content and len(content) > 50:
                            logger.info(f"   ✅ Search '{term}' successful")
                            logger.info(f"   📄 Results: {content[:200]}...")
                            result['success'] = True
                            result['message'] = f"Search functionality working"
                            test_results['passed_tests'] += 1
                            break
                except Exception as e:
                    logger.info(f"   ⚠️  Search '{term}' failed: {e}")
                    continue

            if not result['success']:
                result['message'] = "Search attempts failed"
                test_results['failed_tests'] += 1
        else:
            result['message'] = "No search projects tool found"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Search projects failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results']['search_projects'] = result

async def test_project_details(client, test_results):
    """Test getting project details."""
    logger.info(f"\n🧪 Testing: Get Project Details")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        # Check if we have a get project details tool
        tools = client.get_available_tools()
        detail_tools = [t for t in tools if 'project' in t.name.lower() and ('detail' in t.name.lower() or 'get' in t.name.lower())]

        if detail_tools:
            tool_name = detail_tools[0].name
            logger.info(f"   Using tool: {tool_name}")

            # Try with some common project IDs or names
            test_projects = ["1", "2", "finance", "api"]

            for proj in test_projects:
                try:
                    # Try different parameter names
                    for param_name in ["project_id", "project", "id", "name"]:
                        try:
                            response = await client.call_tool(tool_name, {param_name: proj})

                            if response and isinstance(response, list) and len(response) > 0:
                                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                                if "Error" not in content and len(content) > 50:
                                    logger.info(f"   ✅ Project details for '{proj}' successful")
                                    logger.info(f"   📄 Details: {content[:200]}...")
                                    result['success'] = True
                                    result['message'] = f"Project details functionality working"
                                    test_results['passed_tests'] += 1
                                    return
                        except Exception:
                            continue
                except Exception as e:
                    continue

            if not result['success']:
                result['message'] = "Project details attempts failed"
                test_results['failed_tests'] += 1
        else:
            result['message'] = "No project details tool found"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Project details failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results']['project_details'] = result

async def test_analyze_commits(client, test_results):
    """Test commit message analysis."""
    logger.info(f"\n🧪 Testing: Analyze Commit Messages")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        tools = client.get_available_tools()
        commit_tools = [t for t in tools if 'commit' in t.name.lower() and 'analyze' in t.name.lower()]

        if commit_tools:
            tool_name = commit_tools[0].name
            logger.info(f"   Using tool: {tool_name}")

            # Try with minimal parameters
            test_params = {"project_id": "1"}

            response = await client.call_tool(tool_name, test_params)

            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Analysis: {content[:300]}...")

                result['success'] = True
                result['message'] = "Commit analysis completed"
                test_results['passed_tests'] += 1
            else:
                result['message'] = "No analysis data received"
                test_results['failed_tests'] += 1
        else:
            result['message'] = "No commit analysis tool found"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Commit analysis failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results']['analyze_commits'] = result

async def test_analyze_branches(client, test_results):
    """Test branch activity analysis."""
    logger.info(f"\n🧪 Testing: Analyze Branch Activity")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        tools = client.get_available_tools()
        branch_tools = [t for t in tools if 'branch' in t.name.lower() and 'analyze' in t.name.lower()]

        if branch_tools:
            tool_name = branch_tools[0].name
            logger.info(f"   Using tool: {tool_name}")

            test_params = {"project_id": "1"}

            response = await client.call_tool(tool_name, test_params)

            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Branch Analysis: {content[:300]}...")

                result['success'] = True
                result['message'] = "Branch analysis completed"
                test_results['passed_tests'] += 1
            else:
                result['message'] = "No branch analysis data received"
                test_results['failed_tests'] += 1
        else:
            result['message'] = "No branch analysis tool found"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Branch analysis failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results']['analyze_branches'] = result

async def test_analyze_complexity(client, test_results):
    """Test code complexity analysis."""
    logger.info(f"\n🧪 Testing: Analyze Code Complexity")

    test_results['total_tests'] += 1
    result = {'success': False, 'message': ''}

    try:
        tools = client.get_available_tools()
        complexity_tools = [t for t in tools if 'complexity' in t.name.lower() or ('code' in t.name.lower() and 'analyze' in t.name.lower())]

        if complexity_tools:
            tool_name = complexity_tools[0].name
            logger.info(f"   Using tool: {tool_name}")

            test_params = {"project_id": "1"}

            response = await client.call_tool(tool_name, test_params)

            if response and isinstance(response, list) and len(response) > 0:
                content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                logger.info(f"   📄 Complexity Analysis: {content[:300]}...")

                result['success'] = True
                result['message'] = "Code complexity analysis completed"
                test_results['passed_tests'] += 1
            else:
                result['message'] = "No complexity analysis data received"
                test_results['failed_tests'] += 1
        else:
            result['message'] = "No complexity analysis tool found"
            test_results['failed_tests'] += 1

    except Exception as e:
        logger.error(f"   ❌ Complexity analysis failed: {e}")
        result['message'] = f"Execution failed: {str(e)}"
        test_results['failed_tests'] += 1

    test_results['results']['analyze_complexity'] = result

async def cleanup():
    """Clean up resources."""
    try:
        await mcp_registry.shutdown()
        logger.info("✅ Cleanup complete")
    except Exception as e:
        logger.error(f"❌ Cleanup error: {e}")

if __name__ == "__main__":
    try:
        result = asyncio.run(test_gitlab_tools())
        if result:
            logger.info("🎉 GitLab tools testing completed successfully!")
            sys.exit(0)
        else:
            logger.error("❌ GitLab tools testing had issues!")
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