#!/usr/bin/env python3
"""
Empty Reasoning Workflow Callback
This workflow skips all reasoning steps and passes through the request unchanged.
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
    Empty reasoning workflow that does no processing.

    This workflow skips all reasoning steps and returns immediately,
    passing the request through unchanged to the LLM.

    Args:
        request_data: The request data to process
        analyze_request_intent: Function to analyze user intent (unused)
        generate_reasoning_context: Function to generate reasoning context (unused)
        enhance_messages_with_reasoning: Function to enhance messages (unused)
        discover_reasoning_tools: Function to discover MCP tools (unused)
        execute_reasoning_tools: Function to execute MCP tools (unused)
        stream_reasoning_step: Function to stream reasoning steps (unused)

    Yields:
        Nothing - this is an empty generator
    """
    logger.info("🔄 EMPTY WORKFLOW: Skipping all reasoning steps (no processing)")

    # Make this an async generator by using 'yield' even though we yield nothing
    # This is required for 'async for' to work
    if False:  # Never execute, but makes this an async generator
        yield ""

    return
