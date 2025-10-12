import json
import logging
import time
import asyncio
from typing import Dict, List, Any, Optional
from google.adk.agents import LlmAgent
from google.adk.tools import ToolContext
from src.infrastructure.config.config import config
from src.application.services.mcp_tool_selector import (
    MCPToolSelector, ToolSelectionContext, ProcessingPhase
)
from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.mcp.tool_registry import MCPUnifiedToolRegistry
from src.infrastructure.mcp.discovery import MCPToolDiscovery

logger = logging.getLogger(__name__)

# Global MCP integration components for postprocessing
_postprocessing_mcp_discovery = None
_postprocessing_mcp_tool_registry = None
_postprocessing_mcp_tool_selector = None

def create_unified_analysis(request_metadata: Optional[Dict[str, Any]], response_content: str, response_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Create a unified analysis combining request analysis and response analysis.

    Args:
        request_metadata: Metadata from the original request (reasoning analysis)
        response_content: The complete response content from the LLM
        response_metadata: Optional metadata about the response

    Returns:
        Dictionary with unified analysis results
    """
    try:
        # Generate response analysis
        response_analysis = analyze_response_content(response_content, response_metadata)

        # Combine request and response analysis
        unified_analysis = {
            "request_analysis": request_metadata.get("intent_analysis", {}) if request_metadata else {},
            "response_analysis": response_analysis.get("analysis", {}),
            "created_at": time.time(),
            "status": "success"
        }

        # Create analysis display content
        request_intent = unified_analysis["request_analysis"]
        response_info = unified_analysis["response_analysis"]

        analysis_content = "\n\n**Request & Response Analysis:**\n"

        # Request analysis section
        if request_intent:
            analysis_content += f"🔍 Intent: {request_intent.get('complexity', 'unknown')} request"
            domains = request_intent.get('domains', [])
            if domains:
                analysis_content += f" ({', '.join(domains)})"
            analysis_content += f"\n🎯 Complexity: {request_intent.get('word_count', 0)} words\n"

        # Response analysis section
        if response_info:
            analysis_content += f"📊 Quality Score: {response_info.get('quality_score', 0)}/100\n"
            analysis_content += f"📝 Content Type: {response_info.get('content_type', 'text')}\n"
            analysis_content += f"📏 Word Count: {response_info.get('length_words', 0)} words\n"
            analysis_content += f"😊 Sentiment: {response_info.get('sentiment', 'neutral')}\n"

        analysis_content += "mshogin"

        return {
            "status": "success",
            "unified_analysis": unified_analysis,
            "analysis_content": analysis_content,
            "should_display": True
        }

    except Exception as e:
        logger.error(f"Error creating unified analysis: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "analysis_content": "",
            "should_display": False
        }

def analyze_response_content(content: str, metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Analyze the content of the response for quality, safety, and insights.

    Args:
        content: The complete response content from the LLM
        metadata: Optional metadata about the response

    Returns:
        Dictionary with analysis results
    """
    logger.debug("ANALYZE_RESPONSE_CONTENT")
    try:
        analysis = {
            "length_chars": len(content),
            "length_words": len(content.split()),
            "length_sentences": content.count('.') + content.count('!') + content.count('?'),
            "has_code_blocks": '```' in content,
            "has_links": 'http' in content.lower(),
            "language_detected": "english",  # Simple detection
            "sentiment": "neutral",
            "quality_score": 0.0,
            "safety_flags": [],
            "content_type": "text"
        }

        # Basic quality scoring
        quality_score = 50.0  # Base score

        # Length quality
        if 50 <= analysis["length_words"] <= 500:
            quality_score += 20
        elif analysis["length_words"] > 500:
            quality_score += 15
        elif analysis["length_words"] >= 10:
            quality_score += 10

        # Structure quality
        if analysis["length_sentences"] >= 2:
            quality_score += 15

        if analysis["has_code_blocks"]:
            quality_score += 10
            analysis["content_type"] = "technical"

        # Simple sentiment analysis (very basic)
        positive_words = ["good", "great", "excellent", "helpful", "wonderful", "perfect"]
        negative_words = ["bad", "terrible", "awful", "horrible", "wrong", "error"]

        content_lower = content.lower()
        positive_count = sum(1 for word in positive_words if word in content_lower)
        negative_count = sum(1 for word in negative_words if word in content_lower)

        if positive_count > negative_count:
            analysis["sentiment"] = "positive"
            quality_score += 5
        elif negative_count > positive_count:
            analysis["sentiment"] = "negative"

        # Safety checks (basic)
        safety_keywords = ["password", "secret", "private", "confidential", "hack", "illegal"]
        for keyword in safety_keywords:
            if keyword in content_lower:
                analysis["safety_flags"].append(f"Contains: {keyword}")

        analysis["quality_score"] = min(100.0, quality_score)

        return {
            "status": "success",
            "analysis": analysis,
            "timestamp": time.time()
        }

    except Exception as e:
        logger.error(f"Error analyzing response content: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "analysis": {}
        }

def enhance_response(content: str, enhancement_type: str = "standard") -> Dict[str, Any]:
    """
    Enhance the response content based on specified enhancement type.

    Args:
        content: The original response content
        enhancement_type: Type of enhancement to apply

    Returns:
        Dictionary with enhanced content
    """
    try:
        enhanced_content = content
        enhancements_applied = []

        if enhancement_type == "standard":
            # Basic enhancements

            # Ensure proper sentence endings
            if enhanced_content and not enhanced_content.rstrip().endswith(('.', '!', '?')):
                enhanced_content = enhanced_content.rstrip() + "."
                enhancements_applied.append("Added sentence ending")

            # Remove excessive whitespace
            lines = enhanced_content.split('\n')
            cleaned_lines = []
            for line in lines:
                cleaned_line = ' '.join(line.split())  # Remove extra spaces
                cleaned_lines.append(cleaned_line)
            enhanced_content = '\n'.join(cleaned_lines)

            # Remove excessive empty lines
            while '\n\n\n' in enhanced_content:
                enhanced_content = enhanced_content.replace('\n\n\n', '\n\n')
                enhancements_applied.append("Cleaned whitespace")

        elif enhancement_type == "verbose":
            # Add more detailed explanations
            if len(content.split()) < 20:
                enhanced_content += "\n\nWould you like me to elaborate on any particular aspect of this response?"
                enhancements_applied.append("Added elaboration prompt")

        elif enhancement_type == "concise":
            # Make response more concise
            sentences = content.split('. ')
            if len(sentences) > 3:
                enhanced_content = '. '.join(sentences[:3]) + "."
                enhancements_applied.append("Condensed response")

        return {
            "status": "success",
            "enhanced_content": enhanced_content,
            "original_length": len(content),
            "enhanced_length": len(enhanced_content),
            "enhancements_applied": enhancements_applied
        }

    except Exception as e:
        logger.error(f"Error enhancing response: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "enhanced_content": content  # Return original on error
        }

def log_interaction(request_metadata: Dict[str, Any], response_metadata: Dict[str, Any],
                   analysis: Dict[str, Any]) -> Dict[str, Any]:
    """
    Log the complete interaction for analytics and monitoring.

    Args:
        request_metadata: Metadata from the original request
        response_metadata: Metadata from the OpenAI response
        analysis: Analysis of the response content

    Returns:
        Dictionary with logging results
    """
    try:
        if not config.ENABLE_RESPONSE_ANALYTICS:
            return {
                "status": "skipped",
                "message": "Analytics disabled"
            }

        interaction_log = {
            "timestamp": time.time(),
            "request": {
                "model": request_metadata.get("model"),
                "message_count": request_metadata.get("message_count", 0),
                "estimated_tokens": request_metadata.get("total_tokens_estimate", 0),
                "conversation_type": request_metadata.get("conversation_type"),
                "has_system_message": request_metadata.get("has_system_message", False)
            },
            "response": {
                "response_id": response_metadata.get("response_id"),
                "model_used": response_metadata.get("model_used"),
                "finish_reason": response_metadata.get("finish_reason"),
                "total_tokens": response_metadata.get("total_tokens", 0),
                "prompt_tokens": response_metadata.get("prompt_tokens", 0),
                "completion_tokens": response_metadata.get("completion_tokens", 0)
            },
            "analysis": {
                "quality_score": analysis.get("quality_score", 0),
                "content_type": analysis.get("content_type", "text"),
                "sentiment": analysis.get("sentiment", "neutral"),
                "safety_flags": analysis.get("safety_flags", []),
                "length_words": analysis.get("length_words", 0)
            }
        }

        # Log to appropriate destination (file, database, etc.)
        logger.info(f"Interaction logged: {json.dumps(interaction_log, indent=2)}")

        return {
            "status": "success",
            "logged": True,
            "log_entry": interaction_log
        }

    except Exception as e:
        logger.error(f"Error logging interaction: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "logged": False
        }

def add_chat_content(content: str, content_type: str = "analysis", metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Add custom content to the chat that will be visible to the user.

    Args:
        content: The original response content
        content_type: Type of content to add (analysis, summary, note, etc.)
        metadata: Optional metadata for the content

    Returns:
        Dictionary with the content to be added to the chat
    """
    try:
        chat_content = ""

        if content_type == "analysis":
            # Add analysis information
            analysis_result = analyze_response_content(content, metadata)
            if analysis_result.get("status") == "success":
                analysis = analysis_result.get("analysis", {})
                chat_content = f"\n\n---\n\n**Response Analysis:**\n"
                chat_content += f"📊 Quality Score: {analysis.get('quality_score', 0):.1f}/100\n"
                chat_content += f"📝 Content Type: {analysis.get('content_type', 'text')}\n"
                chat_content += f"📏 Word Count: {analysis.get('length_words', 0)}\n"
                chat_content += f"😊 Sentiment: {analysis.get('sentiment', 'neutral')}\n"

                if analysis.get('safety_flags'):
                    chat_content += f"⚠️ Safety Flags: {', '.join(analysis['safety_flags'])}\n"

        elif content_type == "summary":
            # Add a summary of the response
            words = content.split()
            chat_content = f"\n\n---\n\n**Summary:**\n"
            chat_content += f"📄 Response length: {len(words)} words\n"
            chat_content += f"📊 Key points: {min(3, len(words) // 10)} main ideas\n"

            # Extract first sentence as summary
            sentences = content.split('. ')
            if sentences:
                chat_content += f"💡 Main point: {sentences[0]}.\n"

        elif content_type == "custom":
            # Add custom content from metadata
            if metadata and "custom_message" in metadata:
                chat_content = f"\n\n---\n\n{metadata['custom_message']}\n"

        elif content_type == "enhancement":
            # Add enhancement information
            enhancement_result = enhance_response(content, "standard")
            if enhancement_result.get("status") == "success":
                enhancements = enhancement_result.get("enhancements_applied", [])
                if enhancements:
                    chat_content = f"\n\n---\n\n**Enhancements Applied:**\n"
                    for enhancement in enhancements:
                        chat_content += f"✅ {enhancement}\n"

        chat_content += "mshogin"
        return {
            "status": "success",
            "chat_content": chat_content,
            "content_type": content_type,
            "should_add": len(chat_content.strip()) > 0
        }

    except Exception as e:
        logger.error(f"Error adding chat content: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "chat_content": "",
            "should_add": False
        }

def filter_response(content: str, filter_type: str = "basic") -> Dict[str, Any]:
    """
    Apply content filtering to the response.

    Args:
        content: The response content to filter
        filter_type: Type of filtering to apply

    Returns:
        Dictionary with filtering results
    """
    try:
        filtered_content = content
        filters_applied = []

        if filter_type == "basic":
            # Basic safety filtering

            # Remove potential sensitive information patterns
            import re

            # Email patterns (simple)
            email_pattern = r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'
            if re.search(email_pattern, filtered_content):
                filtered_content = re.sub(email_pattern, '[EMAIL]', filtered_content)
                filters_applied.append("Masked email addresses")

            # Phone number patterns (simple US format)
            phone_pattern = r'\b\d{3}-\d{3}-\d{4}\b|\b\(\d{3}\)\s*\d{3}-\d{4}\b'
            if re.search(phone_pattern, filtered_content):
                filtered_content = re.sub(phone_pattern, '[PHONE]', filtered_content)
                filters_applied.append("Masked phone numbers")

            # Credit card patterns (simple)
            cc_pattern = r'\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b'
            if re.search(cc_pattern, filtered_content):
                filtered_content = re.sub(cc_pattern, '[CARD]', filtered_content)
                filters_applied.append("Masked credit card numbers")

        return {
            "status": "success",
            "filtered_content": filtered_content,
            "original_content": content,
            "filters_applied": filters_applied,
            "content_modified": len(filters_applied) > 0
        }

    except Exception as e:
        logger.error(f"Error filtering response: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "filtered_content": content  # Return original on error
        }


def _initialize_postprocessing_mcp_components():
    """Initialize MCP components for postprocessing integration."""
    global _postprocessing_mcp_discovery, _postprocessing_mcp_tool_registry, _postprocessing_mcp_tool_selector

    try:
        if not _postprocessing_mcp_discovery:
            _postprocessing_mcp_discovery = MCPToolDiscovery(mcp_registry)
            logger.info("MCP Discovery initialized for postprocessing")

        if not _postprocessing_mcp_tool_registry:
            _postprocessing_mcp_tool_registry = MCPUnifiedToolRegistry(mcp_registry, _postprocessing_mcp_discovery)
            logger.info("MCP Tool Registry initialized for postprocessing")

        if not _postprocessing_mcp_tool_selector:
            _postprocessing_mcp_tool_selector = MCPToolSelector(_postprocessing_mcp_tool_registry, mcp_registry, _postprocessing_mcp_discovery)
            logger.info("MCP Tool Selector initialized for postprocessing")

    except Exception as e:
        logger.error(f"Error initializing postprocessing MCP components: {e}")


async def discover_postprocessing_tools(response_content: str, request_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Discover MCP tools suitable for postprocessing the response.

    Args:
        response_content: The LLM response content
        request_metadata: Optional metadata from the original request

    Returns:
        Dictionary with discovered postprocessing tools
    """
    try:
        logger.debug("Discovering MCP tools for postprocessing")

        # Initialize MCP components if needed
        _initialize_postprocessing_mcp_components()

        if not _postprocessing_mcp_tool_selector:
            return {
                "status": "error",
                "error": "MCP Tool Selector not available",
                "postprocessing_tools": []
            }

        # Create pseudo-request data from response content for tool selection
        pseudo_request = {
            "messages": [{"role": "assistant", "content": response_content}],
            "metadata": request_metadata or {}
        }

        # Create tool selection context for postprocessing phase
        context = ToolSelectionContext(
            request_data=pseudo_request,
            intent_analysis=request_metadata.get("intent_analysis") if request_metadata else None,
            processing_phase=ProcessingPhase.POSTPROCESSING
        )

        # Select appropriate postprocessing tools
        selection_result = await _postprocessing_mcp_tool_selector.select_tools_for_context(context)

        if selection_result.get("status") == "success":
            tools = selection_result.get("selected_tools", [])
            logger.debug(f"Discovered {len(tools)} postprocessing tools: {tools}")

            # Filter tools that are suitable for postprocessing
            postprocessing_tools = []
            for tool_name in tools:
                tool_info = _postprocessing_mcp_tool_registry.get_tool_info(tool_name)
                if tool_info and any(keyword in tool_info.get("description", "").lower()
                                   for keyword in ["validate", "enhance", "format", "improve", "quality", "review"]):
                    postprocessing_tools.append(tool_name)

            return {
                "status": "success",
                "postprocessing_tools": postprocessing_tools,
                "all_selected_tools": tools,
                "selection_metadata": selection_result
            }
        else:
            logger.warning(f"Postprocessing tool discovery failed: {selection_result.get('error', 'Unknown error')}")
            return selection_result

    except Exception as e:
        logger.error(f"Error discovering postprocessing tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "postprocessing_tools": []
        }


async def execute_postprocessing_tools(response_content: str, postprocessing_tools: List[str], request_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Execute postprocessing tools to validate and enhance the response.

    Args:
        response_content: The response content to process
        postprocessing_tools: List of selected postprocessing tool names
        request_metadata: Optional metadata from the original request

    Returns:
        Dictionary with postprocessing tool execution results
    """
    try:
        logger.debug(f"Executing {len(postprocessing_tools)} postprocessing tools")

        if not postprocessing_tools or not _postprocessing_mcp_tool_selector:
            return {
                "status": "success",
                "postprocessing_results": {},
                "tools_executed": [],
                "reason": "No postprocessing tools to execute"
            }

        # Create pseudo-request data for tool execution
        pseudo_request = {
            "messages": [{"role": "assistant", "content": response_content}],
            "metadata": request_metadata or {}
        }

        # Create execution plan
        context = ToolSelectionContext(
            request_data=pseudo_request,
            intent_analysis=request_metadata.get("intent_analysis") if request_metadata else None,
            processing_phase=ProcessingPhase.POSTPROCESSING
        )

        plan_result = await _postprocessing_mcp_tool_selector.create_execution_plan(postprocessing_tools, context)

        if plan_result.get("status") != "success":
            return {
                "status": "error",
                "error": f"Failed to create postprocessing execution plan: {plan_result.get('error')}",
                "postprocessing_results": {}
            }

        execution_plan = plan_result.get("execution_plan")
        if not execution_plan:
            return {
                "status": "success",
                "postprocessing_results": {},
                "tools_executed": [],
                "reason": "No execution plan created"
            }

        # Execute postprocessing tools
        execution_result = await _postprocessing_mcp_tool_selector.execute_tool_plan(execution_plan, context)

        if execution_result.get("status") != "success":
            return {
                "status": "error",
                "error": f"Postprocessing tool execution failed: {execution_result.get('error')}",
                "postprocessing_results": {}
            }

        # Process results to extract postprocessing insights
        postprocessing_results = {}
        tools_executed = []

        for result in execution_result.get("results", []):
            if result.success and result.result:
                tools_executed.append(result.tool_name)
                # Add postprocessing results
                postprocessing_results[f"{result.tool_name}_result"] = result.result

        logger.debug(f"Postprocessing completed using {len(tools_executed)} tools: {tools_executed}")

        return {
            "status": "success",
            "postprocessing_results": postprocessing_results,
            "tools_executed": tools_executed,
            "execution_stats": {
                "success_count": execution_result.get("success_count", 0),
                "total_count": execution_result.get("total_count", 0),
                "execution_time_ms": execution_result.get("execution_time_ms", 0)
            }
        }

    except Exception as e:
        logger.error(f"Error executing postprocessing tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "postprocessing_results": {}
        }


async def validate_response_with_mcp_tools(response_content: str, validation_tools: List[str], request_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Validate response using MCP validation tools.

    Args:
        response_content: The response content to validate
        validation_tools: List of validation tool names
        request_metadata: Optional metadata from the original request

    Returns:
        Dictionary with validation results
    """
    try:
        logger.debug(f"Validating response with {len(validation_tools)} tools")

        if not validation_tools or not _postprocessing_mcp_tool_selector:
            return {
                "status": "success",
                "validation_results": {"overall_status": "skipped", "reason": "No validation tools available"},
                "is_valid": True,
                "confidence_score": 0.0
            }

        # Execute validation tools
        validation_execution = await execute_postprocessing_tools(response_content, validation_tools, request_metadata)

        if validation_execution.get("status") != "success":
            return {
                "status": "error",
                "error": f"Validation execution failed: {validation_execution.get('error')}",
                "is_valid": False
            }

        # Process validation results
        validation_results = validation_execution.get("postprocessing_results", {})
        tools_executed = validation_execution.get("tools_executed", [])

        # Aggregate validation results
        validation_scores = []
        validation_issues = []

        for tool_name in tools_executed:
            result = validation_results.get(f"{tool_name}_result")
            if isinstance(result, dict):
                # Extract validation score if available
                if "validation_score" in result:
                    validation_scores.append(result["validation_score"])
                elif "quality_score" in result:
                    validation_scores.append(result["quality_score"] / 100.0)  # Normalize to 0-1

                # Extract issues if available
                if "issues" in result:
                    validation_issues.extend(result["issues"])
                elif "errors" in result:
                    validation_issues.extend(result["errors"])

        # Calculate overall validation status
        overall_score = sum(validation_scores) / len(validation_scores) if validation_scores else 0.5
        is_valid = overall_score >= 0.6 and len(validation_issues) == 0  # 60% threshold

        logger.debug(f"Response validation completed: {is_valid} (score: {overall_score:.2f})")

        return {
            "status": "success",
            "validation_results": {
                "overall_status": "valid" if is_valid else "invalid",
                "confidence_score": overall_score,
                "validation_scores": validation_scores,
                "issues": validation_issues,
                "tools_used": tools_executed
            },
            "is_valid": is_valid,
            "confidence_score": overall_score
        }

    except Exception as e:
        logger.error(f"Error validating response with MCP tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "is_valid": False,
            "confidence_score": 0.0
        }


async def enhance_response_with_mcp_tools(response_content: str, enhancement_tools: List[str], request_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Enhance response using MCP enhancement tools.

    Args:
        response_content: The original response content
        enhancement_tools: List of enhancement tool names
        request_metadata: Optional metadata from the original request

    Returns:
        Dictionary with enhanced response and metadata
    """
    try:
        logger.debug(f"Enhancing response with {len(enhancement_tools)} tools")

        if not enhancement_tools or not _postprocessing_mcp_tool_selector:
            return {
                "status": "success",
                "enhanced_content": response_content,
                "enhancements_applied": [],
                "reason": "No enhancement tools available"
            }

        # Execute enhancement tools
        enhancement_execution = await execute_postprocessing_tools(response_content, enhancement_tools, request_metadata)

        if enhancement_execution.get("status") != "success":
            return {
                "status": "error",
                "error": f"Enhancement execution failed: {enhancement_execution.get('error')}",
                "enhanced_content": response_content
            }

        # Process enhancement results
        enhancement_results = enhancement_execution.get("postprocessing_results", {})
        tools_executed = enhancement_execution.get("tools_executed", [])

        # Apply enhancements
        enhanced_content = response_content
        enhancements_applied = []

        for tool_name in tools_executed:
            result = enhancement_results.get(f"{tool_name}_result")
            if isinstance(result, dict):
                # Check for enhanced content
                if "enhanced_content" in result:
                    enhanced_content = result["enhanced_content"]
                    enhancements_applied.append(f"{tool_name}: Content enhanced")
                elif "formatted_content" in result:
                    enhanced_content = result["formatted_content"]
                    enhancements_applied.append(f"{tool_name}: Content formatted")
                elif "improvements" in result:
                    improvements = result["improvements"]
                    if isinstance(improvements, list):
                        enhancements_applied.extend([f"{tool_name}: {imp}" for imp in improvements])

        logger.debug(f"Response enhancement completed: {len(enhancements_applied)} enhancements applied")

        return {
            "status": "success",
            "enhanced_content": enhanced_content,
            "original_content": response_content,
            "enhancements_applied": enhancements_applied,
            "tools_used": tools_executed,
            "execution_stats": enhancement_execution.get("execution_stats", {})
        }

    except Exception as e:
        logger.error(f"Error enhancing response with MCP tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "enhanced_content": response_content
        }


async def create_postprocessing_pipeline(response_content: str, request_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Create intelligent postprocessing pipeline with MCP tool integration.

    Args:
        response_content: The response content to process
        request_metadata: Optional metadata from the original request

    Returns:
        Dictionary with pipeline configuration and selected tools
    """
    try:
        logger.debug("Creating postprocessing pipeline with MCP integration")

        # Basic pipeline steps
        pipeline_steps = [
            "analyze_response",
            "discover_mcp_tools",
            "validate_response",
            "enhance_response",
            "filter_response",
            "log_interaction"
        ]

        # Determine if MCP tools should be used
        use_mcp_tools = True  # Can be configured based on response type or request

        # Analyze response to determine pipeline optimization
        response_analysis = analyze_response_content(response_content, request_metadata)

        if response_analysis.get("status") == "success":
            analysis = response_analysis.get("analysis", {})

            # Skip MCP tools for very short responses
            if analysis.get("length_words", 0) < 5:
                use_mcp_tools = False
                pipeline_steps = [step for step in pipeline_steps if "mcp" not in step]

        pipeline_config = {
            "steps": pipeline_steps,
            "use_mcp_tools": use_mcp_tools,
            "parallel_execution": False,  # Sequential for now
            "timeout_ms": 30000,  # 30 seconds
            "enable_validation": True,
            "enable_enhancement": True,
            "enable_caching": True
        }

        logger.info(f"Created postprocessing pipeline with {len(pipeline_steps)} steps (MCP: {use_mcp_tools})")

        return {
            "status": "success",
            "pipeline_config": pipeline_config,
            "estimated_duration_ms": len(pipeline_steps) * 1000
        }

    except Exception as e:
        logger.error(f"Error creating postprocessing pipeline: {e}")
        return {
            "status": "error",
            "error": str(e),
            "pipeline_config": None
        }


async def execute_postprocessing_pipeline(response_content: str, pipeline_config: Dict[str, Any], request_metadata: Optional[Dict[str, Any]] = None, response_metadata: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Execute the postprocessing pipeline with MCP tool integration.

    Args:
        response_content: The response content to process
        pipeline_config: Pipeline configuration
        request_metadata: Optional metadata from the original request
        response_metadata: Optional metadata from the response

    Returns:
        Dictionary with processed response and execution results
    """
    try:
        logger.info("Executing postprocessing pipeline")

        processed_content = response_content
        execution_results = {}

        steps = pipeline_config.get("steps", [])
        use_mcp_tools = pipeline_config.get("use_mcp_tools", False)

        # Step 1: Analyze response
        if "analyze_response" in steps:
            analysis_result = analyze_response_content(processed_content, response_metadata)
            execution_results["analysis"] = analysis_result

        # Step 2: Discover MCP tools
        available_tools = []
        if "discover_mcp_tools" in steps and use_mcp_tools:
            tool_discovery_result = await discover_postprocessing_tools(processed_content, request_metadata)
            execution_results["tool_discovery"] = tool_discovery_result

            if tool_discovery_result.get("status") == "success":
                available_tools = tool_discovery_result.get("postprocessing_tools", [])

        # Step 3: Validate response
        validation_results = {}
        if "validate_response" in steps and available_tools:
            # Use validation-specific tools
            validation_tools = [tool for tool in available_tools if "validate" in tool.lower() or "check" in tool.lower()]
            if validation_tools:
                validation_results = await validate_response_with_mcp_tools(processed_content, validation_tools, request_metadata)
                execution_results["validation"] = validation_results

        # Step 4: Enhance response
        enhancement_results = {}
        if "enhance_response" in steps and available_tools:
            # Use enhancement-specific tools
            enhancement_tools = [tool for tool in available_tools if "enhance" in tool.lower() or "improve" in tool.lower() or "format" in tool.lower()]
            if enhancement_tools:
                enhancement_results = await enhance_response_with_mcp_tools(processed_content, enhancement_tools, request_metadata)
                execution_results["enhancement"] = enhancement_results

                if enhancement_results.get("status") == "success":
                    processed_content = enhancement_results.get("enhanced_content", processed_content)

        # Step 5: Filter response
        if "filter_response" in steps:
            filter_result = filter_response(processed_content, "basic")
            execution_results["filtering"] = filter_result

            if filter_result.get("status") == "success":
                processed_content = filter_result.get("filtered_content", processed_content)

        # Step 6: Log interaction
        if "log_interaction" in steps and request_metadata and response_metadata:
            # Combine analysis results for logging
            combined_analysis = {}
            if execution_results.get("analysis", {}).get("status") == "success":
                combined_analysis = execution_results["analysis"].get("analysis", {})

            log_result = log_interaction(request_metadata, response_metadata, combined_analysis)
            execution_results["logging"] = log_result

        # Calculate total execution time
        total_time_ms = sum(
            result.get("execution_time_ms", 0)
            for result in execution_results.values()
            if isinstance(result, dict) and "execution_time_ms" in result
        )

        logger.info(f"Postprocessing pipeline completed in {total_time_ms}ms")

        return {
            "status": "success",
            "processed_content": processed_content,
            "original_content": response_content,
            "execution_results": execution_results,
            "pipeline_stats": {
                "steps_executed": len([r for r in execution_results.values() if r.get("status") == "success"]),
                "total_steps": len(steps),
                "execution_time_ms": total_time_ms,
                "mcp_tools_used": len(available_tools),
                "content_modified": processed_content != response_content
            }
        }

    except Exception as e:
        logger.error(f"Error executing postprocessing pipeline: {e}")
        return {
            "status": "error",
            "error": str(e),
            "processed_content": response_content  # Return original on error
        }

# Create the postprocessing agent using new ADK API
from google.adk.agents import Agent

postprocessing_agent = Agent(
    name="postprocessing_agent",
    model="gemini-2.0-flash",
    tools=[
        analyze_response_content,
        enhance_response,
        log_interaction,
        filter_response,
        add_chat_content,
        discover_postprocessing_tools,
        execute_postprocessing_tools,
        validate_response_with_mcp_tools,
        enhance_response_with_mcp_tools,
        create_postprocessing_pipeline,
        execute_postprocessing_pipeline
    ],
    instruction="""You are an intelligent postprocessing agent for an LLM reverse proxy server with MCP tool integration.

    Your responsibilities:
    1. Analyze response content using analyze_response_content()
    2. Discover and utilize MCP tools for response processing using discover_postprocessing_tools()
    3. Validate responses using MCP validation tools via validate_response_with_mcp_tools()
    4. Enhance responses with MCP enhancement tools via enhance_response_with_mcp_tools()
    5. Create intelligent postprocessing pipelines using create_postprocessing_pipeline()
    6. Execute postprocessing pipelines with MCP integration using execute_postprocessing_pipeline()
    7. Apply content filtering using filter_response()
    8. Log interactions for analytics using log_interaction()

    Enhanced postprocessing workflow:
    1. First analyze the response content for quality, safety, and characteristics
    2. Discover appropriate MCP tools based on response type and request context
    3. Create an intelligent postprocessing pipeline optimized for the specific response
    4. Execute the pipeline, including MCP tool validation and enhancement
    5. Validate response quality and safety using specialized tools
    6. Enhance the response with formatting, improvements, and optimizations
    7. Apply content filtering for security and compliance
    8. Log the complete interaction for monitoring and analytics

    MCP Integration Features:
    - Intelligent tool selection based on response analysis and request context
    - Response validation using domain-specific validation tools
    - Content enhancement using specialized formatting and improvement tools
    - Quality assurance through automated review tools
    - Parallel and sequential tool execution with error handling
    - Result aggregation and confidence scoring
    - Fallback mechanisms for tool failures

    Preserve the original response integrity while adding significant value through intelligent analysis,
    validation, and enhancement. Always return processed content that meets quality standards and is ready
    for delivery to the client.""",
    description="Handles intelligent response postprocessing, analysis, filtering, enhancement, and MCP tool integration"
)
