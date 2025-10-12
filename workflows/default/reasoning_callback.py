#!/usr/bin/env python3
"""
Default Reasoning Workflow Callback
This module implements the default reasoning workflow that can be customized.
"""

import logging
import asyncio
from typing import Dict, List, Any, AsyncGenerator

logger = logging.getLogger(__name__)


async def reasoning_workflow(
    request_data: Dict[str, Any],
    analyze_request_intent,
    generate_reasoning_context,
    enhance_messages_with_reasoning,
    discover_reasoning_tools,
    execute_reasoning_tools,
    stream_reasoning_step
) -> AsyncGenerator[str, None]:
    """
    Default reasoning workflow callback.

    This workflow follows the standard reasoning pipeline:
    1. Analyze request intent
    2. Discover and execute MCP reasoning tools
    3. Generate reasoning context
    4. Enhance messages with reasoning

    Args:
        request_data: The request data to process
        analyze_request_intent: Function to analyze user intent
        generate_reasoning_context: Function to generate reasoning context
        enhance_messages_with_reasoning: Function to enhance messages
        discover_reasoning_tools: Function to discover MCP tools
        execute_reasoning_tools: Function to execute MCP tools
        stream_reasoning_step: Function to stream reasoning steps

    Yields:
        Reasoning step chunks in SSE format
    """
    try:
        logger.info("🔄 DEFAULT WORKFLOW: Starting reasoning pipeline")

        # Step 1: Analyze request intent
        logger.debug("🔄 WORKFLOW Step 1: Analyzing intent")
        yield await stream_reasoning_step("intent_analysis", {"status": "analyzing user intent..."}, None)

        intent_result = analyze_request_intent(request_data)
        if intent_result.get("status") != "success":
            yield await stream_reasoning_step("intent_analysis", {"status": "failed", "error": intent_result.get("error")}, None)
            return

        yield await stream_reasoning_step("intent_analysis", {
            "status": "completed",
            "complexity": intent_result["intent_analysis"]["complexity"],
            "domains": intent_result["intent_analysis"]["domains"],
            "word_count": intent_result["intent_analysis"]["word_count"]
        }, None)

        await asyncio.sleep(0.1)

        # Step 2: Discover and execute reasoning tools
        reasoning_insights = {}
        yield await stream_reasoning_step("mcp_tool_discovery", {"status": "discovering reasoning tools..."}, None)

        tool_discovery_result = await discover_reasoning_tools(request_data, intent_result["intent_analysis"])

        if tool_discovery_result.get("status") == "success":
            reasoning_tools = tool_discovery_result.get("reasoning_tools", [])
            preferred_tools = tool_discovery_result.get("preferred_tools", [])

            if reasoning_tools:
                yield await stream_reasoning_step("mcp_tool_discovery", {
                    "status": "completed",
                    "tools_found": len(reasoning_tools),
                    "preferred_tools": len(preferred_tools),
                    "tools": reasoning_tools
                }, None)

                await asyncio.sleep(0.1)

                # Execute reasoning tools
                logger.debug("🔄 WORKFLOW Step 2.1: Executing reasoning tools")
                yield await stream_reasoning_step("mcp_tool_execution", {"status": "executing reasoning tools..."}, None)

                tool_execution_result = await execute_reasoning_tools(request_data, reasoning_tools, intent_result["intent_analysis"])

                if tool_execution_result.get("status") == "success":
                    reasoning_insights = tool_execution_result.get("reasoning_insights", {})
                    tools_executed = tool_execution_result.get("tools_executed", [])
                    execution_plan = tool_execution_result.get("execution_plan", {})

                    yield await stream_reasoning_step("mcp_tool_execution", {
                        "status": "completed",
                        "tools_executed": len(tools_executed),
                        "insights_gathered": len(reasoning_insights),
                        "plan_type": execution_plan.get("intent_type", "unknown") if execution_plan else "basic"
                    }, None)
                else:
                    yield await stream_reasoning_step("mcp_tool_execution", {
                        "status": "failed",
                        "error": tool_execution_result.get("error")
                    }, None)
            else:
                yield await stream_reasoning_step("mcp_tool_discovery", {
                    "status": "completed",
                    "tools_found": 0,
                    "reason": "No suitable reasoning tools found"
                }, None)
        else:
            yield await stream_reasoning_step("mcp_tool_discovery", {
                "status": "failed",
                "error": tool_discovery_result.get("error")
            }, None)

        await asyncio.sleep(0.1)

        # Step 3: Generate reasoning context
        logger.debug("🔄 WORKFLOW Step 3: Generating context")
        yield await stream_reasoning_step("context_generation", {"status": "generating reasoning context..."}, None)

        messages = request_data.get("messages", [])
        context_result = generate_reasoning_context(intent_result["intent_analysis"], messages, reasoning_insights)
        if context_result.get("status") != "success":
            yield await stream_reasoning_step("context_generation", {"status": "failed", "error": context_result.get("error")}, None)
            return

        yield await stream_reasoning_step("context_generation", {
            "status": "completed",
            "context_items": len(context_result["reasoning_context"]),
            "enhanced_understanding": context_result["enhanced_understanding"]
        }, None)

        await asyncio.sleep(0.1)

        # Step 4: Enhance messages
        logger.debug("🔄 WORKFLOW Step 4: Enhancing messages")
        yield await stream_reasoning_step("message_enhancement", {"status": "enhancing messages with reasoning..."}, None)

        enhancement_result = enhance_messages_with_reasoning(messages, context_result["reasoning_prompt"])
        if enhancement_result.get("status") != "success":
            yield await stream_reasoning_step("message_enhancement", {"status": "failed", "error": enhancement_result.get("error")}, None)
            return

        yield await stream_reasoning_step("message_enhancement", {
            "status": "completed",
            "original_messages": len(messages),
            "enhanced_messages": len(enhancement_result["enhanced_messages"]),
            "reasoning_added": enhancement_result["reasoning_added"]
        }, None)

        await asyncio.sleep(0.1)

        # Build enhanced request for final display
        enhanced_request_for_display = request_data.copy()
        enhanced_request_for_display["messages"] = enhancement_result["enhanced_messages"]

        # Final reasoning completion
        yield await stream_reasoning_step("reasoning_complete", {
            "status": "reasoning pipeline completed successfully",
            "enhanced_request_ready": True,
            "pipeline": "default workflow: intent → tools → context → enhancement"
        }, enhanced_request_for_display)

        logger.info("🔄 DEFAULT WORKFLOW: Pipeline completed successfully")

    except Exception as e:
        logger.error(f"❌ Error in default workflow: {e}")
        yield await stream_reasoning_step("error", {"status": "failed", "error": str(e)}, None)
