#!/usr/bin/env python3
"""
Enhanced Reasoning Workflow Callback
This workflow uses the LLM-powered enhanced reasoning orchestrator.
"""

import logging
from typing import Dict, Any, AsyncGenerator

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
    Enhanced reasoning workflow using multi-agent LLM-powered reasoning.

    This workflow uses the enhanced_reasoning_orchestrator which provides:
    - LLM Intent Analysis
    - LLM Plan Generation
    - LLM Recursive Execution
    - LLM Context Evaluation

    Args:
        request_data: The request data to process
        analyze_request_intent: Function to analyze user intent (unused)
        generate_reasoning_context: Function to generate reasoning context (unused)
        enhance_messages_with_reasoning: Function to enhance messages
        discover_reasoning_tools: Function to discover MCP tools (unused)
        execute_reasoning_tools: Function to execute MCP tools (unused)
        stream_reasoning_step: Function to stream reasoning steps

    Yields:
        Reasoning step chunks in SSE format
    """
    try:
        logger.info("🔄 ENHANCED WORKFLOW: Starting multi-agent reasoning pipeline")
        import ipdb; ipdb.set_trace()

        # Use the new enhanced multi-agent reasoning system
        from src.domain.services.enhanced_reasoning_orchestrator import enhanced_reasoning_pipeline

        async for chunk in enhanced_reasoning_pipeline(request_data):
            yield chunk

        # Get the collected context from enhanced reasoning and create enhanced request
        from src.domain.services.enhanced_reasoning_orchestrator import get_enhanced_reasoning_context
        messages = request_data.get("messages", [])

        try:
            enhanced_context = get_enhanced_reasoning_context()
            if enhanced_context and enhanced_context.collected_context:
                # Build reasoning context from collected MCP results
                reasoning_context_parts = []
                reasoning_context_parts.append("Enhanced Multi-Agent Reasoning Results:")
                reasoning_context_parts.append("="*50)

                for item in enhanced_context.collected_context:
                    if item.get('success') and item.get('result'):
                        tool_name = item.get('tool_name', 'unknown_tool')
                        result_content = str(item.get('result', ''))
                        reasoning_context_parts.append(f"\n🛠️ Tool: {tool_name}")
                        reasoning_context_parts.append(f"Result: {result_content}")
                        reasoning_context_parts.append("-" * 30)

                reasoning_context = "\n".join(reasoning_context_parts)
                logger.info(f"🔄 Enhanced reasoning collected {len(enhanced_context.collected_context)} context items")
            else:
                reasoning_context = "Enhanced multi-agent reasoning completed with LLM-powered analysis."
                logger.info("🔄 No specific context collected from enhanced reasoning")
        except Exception as e:
            logger.warning(f"Failed to get enhanced reasoning context: {e}")
            reasoning_context = "Enhanced multi-agent reasoning completed with LLM-powered analysis."

        enhancement_result = enhance_messages_with_reasoning(messages, reasoning_context)
        if enhancement_result.get("status") == "success":
            enhanced_request_for_display = request_data.copy()
            enhanced_request_for_display["messages"] = enhancement_result["enhanced_messages"]

            # Final completion with enhanced request preview
            context_items_count = len(enhanced_context.collected_context) if enhanced_context and enhanced_context.collected_context else 0
            yield await stream_reasoning_step("enhanced_reasoning_complete", {
                "status": "enhanced reasoning pipeline completed successfully",
                "enhanced_request_ready": True,
                "pipeline": "LLM Intent → LLM Plan → LLM Execution → LLM Evaluation → Enhancement",
                "context_items": context_items_count
            }, enhanced_request_for_display)

        logger.info("🔄 ENHANCED WORKFLOW: Pipeline completed successfully")

    except Exception as e:
        logger.error(f"❌ Error in enhanced workflow: {e}")
        yield await stream_reasoning_step("error", {"status": "failed", "error": str(e)}, None)
