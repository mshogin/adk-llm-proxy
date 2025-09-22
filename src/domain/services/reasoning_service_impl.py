#!/usr/bin/env python3
"""
Reasoning Module for LLM Reverse Proxy
This module performs intelligent reasoning on user requests before sending to LLM.
It enriches the context and streams reasoning steps to the caller.
"""

import logging
import json
import time
import asyncio
from typing import Dict, List, Any, Optional, AsyncGenerator
from google.adk.agents import Agent
from google.adk.tools import ToolContext

logger = logging.getLogger(__name__)

def analyze_request_intent(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """Analyze the user's request to understand intent and complexity."""
    try:
        logger.debug("🧠 REASONING: Analyzing request intent")

        messages = request_data.get("messages", [])
        if not messages:
            return {"status": "error", "error": "No messages to analyze"}

        # Get the last user message
        user_messages = [msg for msg in messages if msg.get("role") == "user"]
        if not user_messages:
            return {"status": "error", "error": "No user messages found"}

        last_message = user_messages[-1].get("content", "")

        # Analyze intent
        intent_analysis = {
            "message_length": len(last_message),
            "word_count": len(last_message.split()),
            "contains_question": "?" in last_message,
            "contains_request": any(word in last_message.lower() for word in ["please", "can you", "help", "create", "make", "build"]),
            "complexity": "simple" if len(last_message.split()) < 10 else "complex",
            "domains": [],
            "needs_context": len(messages) > 2
        }

        # Detect domains
        if any(word in last_message.lower() for word in ["code", "programming", "function", "api"]):
            intent_analysis["domains"].append("programming")
        if any(word in last_message.lower() for word in ["explain", "what", "how", "why"]):
            intent_analysis["domains"].append("explanation")
        if any(word in last_message.lower() for word in ["create", "generate", "build", "make"]):
            intent_analysis["domains"].append("creation")

        logger.debug(f"🧠 Intent analysis: {intent_analysis}")

        return {
            "status": "success",
            "intent_analysis": intent_analysis,
            "original_message": last_message
        }

    except Exception as e:
        logger.error(f"❌ Error analyzing request intent: {e}")
        return {"status": "error", "error": str(e)}

def generate_reasoning_context(intent_analysis: Dict[str, Any], messages: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Generate additional context based on reasoning analysis."""
    try:
        logger.debug("🧠 REASONING: Generating reasoning context")

        reasoning_context = []

        # Add context based on complexity
        if intent_analysis.get("complexity") == "complex":
            reasoning_context.append("This appears to be a complex request that may require detailed analysis.")

        # Add context based on domains
        domains = intent_analysis.get("domains", [])
        if "programming" in domains:
            reasoning_context.append("Programming context: Consider code quality, best practices, and maintainability.")
        if "explanation" in domains:
            reasoning_context.append("Explanation context: Provide clear, step-by-step explanations with examples.")
        if "creation" in domains:
            reasoning_context.append("Creation context: Focus on creating practical, working solutions.")

        # Add conversation context
        if intent_analysis.get("needs_context"):
            reasoning_context.append("Conversation context: This is part of an ongoing conversation. Consider previous messages.")

        # Generate reasoning prompt (without request analysis - that's shown in unified analysis at end)
        reasoning_prompt = "\n".join([
            "# Internal Reasoning Context",
            "The following context has been generated through intelligent analysis:",
            "",
            *[f"- {ctx}" for ctx in reasoning_context],
            "",
            "Please consider this context when formulating your response."
        ])

        logger.debug(f"🧠 Generated reasoning context: {len(reasoning_context)} items")

        return {
            "status": "success",
            "reasoning_context": reasoning_context,
            "reasoning_prompt": reasoning_prompt,
            "enhanced_understanding": True
        }

    except Exception as e:
        logger.error(f"❌ Error generating reasoning context: {e}")
        return {"status": "error", "error": str(e)}

def enhance_messages_with_reasoning(messages: List[Dict[str, Any]], reasoning_context: str) -> Dict[str, Any]:
    """Enhance the messages array with reasoning context by combining with existing system messages."""
    try:
        logger.debug("🧠 REASONING: Enhancing messages with reasoning context")

        enhanced_messages = []
        system_content_parts = []

        # First pass: collect all system message content and non-system messages
        for message in messages:
            if message.get("role") == "system":
                # Collect system content to combine later
                system_content_parts.append(message.get("content", ""))
            else:
                enhanced_messages.append(message)

        # Add reasoning context to system content parts
        system_content_parts.append(reasoning_context)

        # Create unified system message combining all system content
        unified_system_content = "\n\n".join(filter(None, system_content_parts))

        # Insert unified system message at the beginning
        if unified_system_content.strip():
            unified_system_message = {
                "role": "system",
                "content": unified_system_content
            }
            enhanced_messages.insert(0, unified_system_message)

        logger.debug(f"🧠 Enhanced messages: unified {len(system_content_parts)} system parts into single message")

        return {
            "status": "success",
            "enhanced_messages": enhanced_messages,
            "reasoning_added": True
        }

    except Exception as e:
        logger.error(f"❌ Error enhancing messages with reasoning: {e}")
        return {"status": "error", "error": str(e)}

async def stream_reasoning_step(step_name: str, step_data: Dict[str, Any], enhanced_request: Dict[str, Any] = None) -> str:
    """Generate a streaming message for a reasoning step as chat content."""
    from src.domain.services.content_filter_service import add_reasoning_markers

    # Convert reasoning step to visible chat content
    if step_data.get("status") == "analyzing user intent...":
        content = "🧠 **Reasoning**: Analyzing your request..."
    elif step_data.get("status") == "generating reasoning context...":
        content = "🧠 **Reasoning**: Generating intelligent context..."
    elif step_data.get("status") == "enhancing messages with reasoning...":
        content = "🧠 **Reasoning**: Enhancing request with insights..."
    elif step_data.get("status") == "reasoning pipeline completed successfully":
        # Include the FULL enhanced message being sent to LLM (no truncation)
        enhanced_messages_text = ""
        if enhanced_request and "messages" in enhanced_request:
            messages_preview = []
            for msg in enhanced_request["messages"]:
                role = msg.get("role", "unknown")
                content_full = msg.get("content", "")  # No truncation
                messages_preview.append(f"**{role.title()}**: {content_full}")
            enhanced_messages_text = f"\n\n```\nEnhanced Request to LLM:\n{chr(10).join(messages_preview)}\n```\n\n"

        content = f"🧠 **Reasoning**: Analysis complete, sending to LLM...{enhanced_messages_text}---\n\n"
    elif step_data.get("status") == "completed":
        if step_name == "intent_analysis":
            complexity = step_data.get("complexity", "unknown")
            domains = step_data.get("domains", [])
            content = f"🧠 **Analysis**: {complexity.title()} request detected" + (f" ({', '.join(domains)})" if domains else "") + "\n"
        elif step_name == "context_generation":
            items = step_data.get("context_items", 0)
            content = f"🧠 **Context**: Generated {items} reasoning insights\n"
        elif step_name == "message_enhancement":
            original = step_data.get("original_messages", 0)
            enhanced = step_data.get("enhanced_messages", 0)
            content = f"🧠 **Enhancement**: Request enriched ({original} → {enhanced} messages)\n"
        else:
            content = f"🧠 **{step_name.replace('_', ' ').title()}**: {step_data.get('status', 'completed')}\n"
    else:
        content = f"🧠 **{step_name.replace('_', ' ').title()}**: {step_data.get('status', 'processing...')}\n"

    # Don't add reasoning markers to individual steps - we'll add them at pipeline level
    # marked_content = add_reasoning_markers(content)

    # Format as standard chat completion chunk
    reasoning_chunk = {
        "id": f"reasoning-{int(time.time())}",
        "object": "chat.completion.chunk",
        "created": int(time.time()),
        "model": "reasoning-engine",
        "choices": [
            {
                "index": 0,
                "delta": {
                    "content": content
                },
                "finish_reason": None
            }
        ]
    }
    return f"data: {json.dumps(reasoning_chunk)}\n\n"

async def reasoning_pipeline(request_data: Dict[str, Any], enhanced_request: Dict[str, Any] = None) -> AsyncGenerator[str, None]:
    """
    Execute the reasoning pipeline with streaming updates to the caller.

    Pipeline:
    1. Analyze request intent
    2. Generate reasoning context
    3. Enhance messages with reasoning
    4. Return enhanced request data
    """
    try:
        logger.info("🧠 REASONING: Starting reasoning pipeline")

        # Start reasoning block
        from src.domain.services.content_filter_service import REASONING_START_MARKER
        start_chunk = {
            "id": f"reasoning-start-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "reasoning-engine",
            "choices": [
                {
                    "index": 0,
                    "delta": {
                        "content": f"{REASONING_START_MARKER}\n"
                    },
                    "finish_reason": None
                }
            ]
        }
        yield f"data: {json.dumps(start_chunk)}\n\n"

        # Step 1: Analyze request intent
        logger.debug("🧠 REASONING Step 1: Analyzing intent")
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

        # Small delay for streaming effect
        await asyncio.sleep(0.1)

        # Step 2: Generate reasoning context
        logger.debug("🧠 REASONING Step 2: Generating context")
        yield await stream_reasoning_step("context_generation", {"status": "generating reasoning context..."}, None)

        messages = request_data.get("messages", [])
        context_result = generate_reasoning_context(intent_result["intent_analysis"], messages)
        if context_result.get("status") != "success":
            yield await stream_reasoning_step("context_generation", {"status": "failed", "error": context_result.get("error")}, None)
            return

        yield await stream_reasoning_step("context_generation", {
            "status": "completed",
            "context_items": len(context_result["reasoning_context"]),
            "enhanced_understanding": context_result["enhanced_understanding"]
        }, None)

        await asyncio.sleep(0.1)

        # Step 3: Enhance messages
        logger.debug("🧠 REASONING Step 3: Enhancing messages")
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

        # Final reasoning completion with enhanced request preview
        yield await stream_reasoning_step("reasoning_complete", {
            "status": "reasoning pipeline completed successfully",
            "enhanced_request_ready": True,
            "pipeline": "preprocessing → reasoning → LLM → postprocessing"
        }, enhanced_request_for_display)

        # End reasoning block
        from src.domain.services.content_filter_service import REASONING_END_MARKER
        end_chunk = {
            "id": f"reasoning-end-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "reasoning-engine",
            "choices": [
                {
                    "index": 0,
                    "delta": {
                        "content": f"{REASONING_END_MARKER}\n"
                    },
                    "finish_reason": None
                }
            ]
        }
        yield f"data: {json.dumps(end_chunk)}\n\n"

        logger.info("🧠 REASONING: Pipeline completed successfully")

    except Exception as e:
        logger.error(f"❌ Error in reasoning pipeline: {e}")
        yield await stream_reasoning_step("error", {"status": "failed", "error": str(e)}, None)

def apply_reasoning_to_request(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Apply reasoning enhancements to the request data.
    This is the non-streaming version for the orchestrator.
    """
    try:
        logger.debug("🧠 REASONING: Applying reasoning to request")

        # Analyze intent
        intent_result = analyze_request_intent(request_data)
        if intent_result.get("status") != "success":
            return {"status": "error", "error": f"Intent analysis failed: {intent_result.get('error')}"}

        # Generate context
        messages = request_data.get("messages", [])
        context_result = generate_reasoning_context(intent_result["intent_analysis"], messages)
        if context_result.get("status") != "success":
            return {"status": "error", "error": f"Context generation failed: {context_result.get('error')}"}

        # Enhance messages
        enhancement_result = enhance_messages_with_reasoning(messages, context_result["reasoning_prompt"])
        if enhancement_result.get("status") != "success":
            return {"status": "error", "error": f"Message enhancement failed: {enhancement_result.get('error')}"}

        # Return enhanced request
        enhanced_request = request_data.copy()
        enhanced_request["messages"] = enhancement_result["enhanced_messages"]

        return {
            "status": "success",
            "enhanced_request": enhanced_request,
            "reasoning_metadata": {
                "intent_analysis": intent_result["intent_analysis"],
                "reasoning_context": context_result["reasoning_context"],
                "original_message_count": len(messages),
                "enhanced_message_count": len(enhancement_result["enhanced_messages"])
            }
        }

    except Exception as e:
        logger.error(f"❌ Error applying reasoning to request: {e}")
        return {"status": "error", "error": str(e)}

# Create ADK agent for reasoning
def create_reasoning_agent():
    """Create ADK agent for reasoning with proper tools."""
    try:
        reasoning_agent = Agent(
            name="reasoning_agent",
            tools=[
                analyze_request_intent,
                generate_reasoning_context,
                enhance_messages_with_reasoning,
                apply_reasoning_to_request
            ]
        )
        logger.info("✅ Reasoning agent created successfully")
        return reasoning_agent
    except Exception as e:
        logger.error(f"❌ Error creating reasoning agent: {e}")
        return None

# Global reasoning agent instance
reasoning_agent = create_reasoning_agent()