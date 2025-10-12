#!/usr/bin/env python3
"""
Debug MCP tool selection to see what's happening.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.mcp.discovery import MCPToolDiscovery
from src.infrastructure.mcp.tool_registry import MCPUnifiedToolRegistry
from src.application.services.mcp_tool_selector import MCPToolSelector, ToolSelectionContext, ProcessingPhase
from src.infrastructure.config.config import config

async def debug_mcp_tool_selection():
    """Debug the MCP tool selection process."""
    print("🔍 Debugging MCP Tool Selection")
    print("=" * 50)

    try:
        # Check if MCP servers are enabled
        enabled_servers = config.get_enabled_mcp_servers()
        print(f"📋 Enabled MCP servers: {len(enabled_servers)}")
        for server in enabled_servers:
            print(f"   - {server.name}: {server.enabled}")

        # Initialize MCP components
        print("\n🔧 Initializing MCP components...")
        discovery = MCPToolDiscovery(mcp_registry)
        await discovery.discover_all_capabilities()

        all_tools = discovery.get_all_tools()
        print(f"🛠️  Discovered {len(all_tools)} total tools")

        for tool in all_tools:
            print(f"   - {tool.server_name}: {tool.name}")

        # Check YouTrack specifically
        youtrack_tools = [tool for tool in all_tools if 'youtrack' in tool.server_name.lower()]
        print(f"\n🎫 YouTrack tools: {len(youtrack_tools)}")
        for tool in youtrack_tools:
            print(f"   - {tool.name}: {tool.description[:100] if tool.description else 'No description'}")

        # Test tool selection
        print("\n🎯 Testing tool selection for task management...")
        tool_registry = MCPUnifiedToolRegistry(mcp_registry, discovery)
        tool_selector = MCPToolSelector(tool_registry, mcp_registry, discovery)

        # Create context for task management request
        context = ToolSelectionContext(
            request_data={
                "messages": [{"role": "user", "content": "Show me my assigned tickets from YouTrack"}]
            },
            intent_analysis={"intent_type": "task_management", "target_tools": ["youtrack"], "action_verbs": ["find_assigned"]},
            processing_phase=ProcessingPhase.REASONING
        )

        selection_result = await tool_selector.select_tools_for_context(context)
        print(f"🔍 Tool selection result status: {selection_result.get('status')}")

        if selection_result.get('status') == 'success':
            selected_tools = selection_result.get('selected_tools', [])
            print(f"✅ Selected {len(selected_tools)} tools:")
            for tool in selected_tools:
                print(f"   - {tool}")

            if selected_tools:
                # Test execution plan creation
                print("\n📋 Creating execution plan...")
                plan_result = await tool_selector.create_execution_plan(selected_tools, context)
                print(f"📊 Execution plan status: {plan_result.get('status')}")

                if plan_result.get('status') == 'success':
                    execution_plan = plan_result.get('execution_plan')
                    if execution_plan:
                        print("✅ Execution plan created successfully")
                        print(f"🎯 Plan has {len(execution_plan.tool_calls)} tool calls")

                        # Test actual execution
                        print("\n🚀 Testing actual tool execution...")
                        execution_result = await tool_selector.execute_tool_plan(execution_plan, context)
                        print(f"🎯 Execution result status: {execution_result.get('status')}")

                        if execution_result.get('status') == 'success':
                            results = execution_result.get('results', [])
                            print(f"📊 Got {len(results)} execution results")

                            for i, result in enumerate(results):
                                print(f"\n📋 Result {i+1}:")
                                print(f"   Tool: {getattr(result, 'tool_name', 'unknown')}")
                                print(f"   Success: {getattr(result, 'success', False)}")
                                if hasattr(result, 'result'):
                                    result_content = str(result.result)[:300]
                                    print(f"   Content: {result_content}...")

                                    if 'ticket' in result_content.lower() or 'st-' in result_content:
                                        print("🎫 FOUND TICKET DATA! ✅")
                        else:
                            print(f"❌ Execution failed: {execution_result.get('error')}")
                    else:
                        print("❌ No execution plan created")
                else:
                    print(f"❌ Plan creation failed: {plan_result.get('error')}")
            else:
                print("⚠️  No tools selected")
        else:
            print(f"❌ Tool selection failed: {selection_result.get('error')}")

    except Exception as e:
        print(f"❌ Error debugging MCP tool selection: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    try:
        asyncio.run(debug_mcp_tool_selection())
    except KeyboardInterrupt:
        print("\n🛑 Debug interrupted by user")