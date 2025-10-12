import json
import logging
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

# Global MCP integration components (initialized on import)
_mcp_discovery = None
_mcp_tool_registry = None
_mcp_tool_selector = None

def inject_context(messages: List[Dict[str, Any]], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Inject contextual information into the conversation messages.

    Args:
        messages: List of message objects from the OpenAI request
        context: Optional context dictionary containing additional information

    Returns:
        Dictionary with processed messages and metadata
    """
    try:
        processed_messages = []

        # Process existing messages and optionally override/add system prompt
        if config.ENABLE_CONTEXT_INJECTION:
            # Build the system message content
            system_content = config.SYSTEM_PROMPT_PREFIX

            # Add context information if provided
            if context:
                context_info = "\n\nAdditional Context:\n"
                for key, value in context.items():
                    if isinstance(value, (str, int, float)):
                        context_info += f"- {key}: {value}\n"
                system_content += context_info

            # Add system message at the beginning, replacing any existing one
            processed_messages.append({
                "role": "system",
                "content": system_content
            })

            # Process other messages (skip existing system messages)
            for message in messages:
                if message.get('role') == 'system':
                    continue  # Skip original system message, we're using config

                processed_msg = message.copy()

                # Ensure message content doesn't exceed max length
                if isinstance(processed_msg.get('content'), str):
                    content = processed_msg['content']
                    if len(content) > config.MAX_CONTEXT_LENGTH:
                        logger.warning(f"Message content truncated from {len(content)} to {config.MAX_CONTEXT_LENGTH} characters")
                        processed_msg['content'] = content[:config.MAX_CONTEXT_LENGTH] + "..."

                processed_messages.append(processed_msg)
        else:
            # Context injection disabled - pass through all messages unchanged
            for message in messages:
                processed_msg = message.copy()

                # Still apply length limits
                if isinstance(processed_msg.get('content'), str):
                    content = processed_msg['content']
                    if len(content) > config.MAX_CONTEXT_LENGTH:
                        logger.warning(f"Message content truncated from {len(content)} to {config.MAX_CONTEXT_LENGTH} characters")
                        processed_msg['content'] = content[:config.MAX_CONTEXT_LENGTH] + "..."

                processed_messages.append(processed_msg)

        return {
            "status": "success",
            "messages": processed_messages,
            "original_count": len(messages),
            "processed_count": len(processed_messages),
            "context_injected": config.ENABLE_CONTEXT_INJECTION and any(msg.get('role') == 'system' for msg in processed_messages)
        }

    except Exception as e:
        logger.error(f"Error in context injection: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "messages": messages  # Return original messages on error
        }

def extract_request_metadata(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Extract and analyze metadata from the incoming request.

    Args:
        request_data: The original OpenAI-compatible request

    Returns:
        Dictionary with extracted metadata
    """
    try:
        metadata = {
            "model": request_data.get("model", config.OPENAI_DEFAULT_MODEL),
            "temperature": request_data.get("temperature", 0.7),
            "max_tokens": request_data.get("max_tokens"),
            "stream": request_data.get("stream", False),
            "message_count": len(request_data.get("messages", [])),
            "total_tokens_estimate": 0,
            "has_system_message": False,
            "conversation_type": "chat"
        }

        # Analyze messages
        messages = request_data.get("messages", [])
        total_chars = 0

        for msg in messages:
            if msg.get("role") == "system":
                metadata["has_system_message"] = True

            content = msg.get("content", "")
            if isinstance(content, str):
                total_chars += len(content)

        # Rough token estimation (4 chars ≈ 1 token)
        metadata["total_tokens_estimate"] = total_chars // 4

        # Determine conversation type
        if len(messages) == 1:
            metadata["conversation_type"] = "single_shot"
        elif len(messages) > 5:
            metadata["conversation_type"] = "extended_chat"

        return {
            "status": "success",
            "metadata": metadata
        }

    except Exception as e:
        logger.error(f"Error extracting metadata: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "metadata": {}
        }

def validate_request(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Validate the incoming request for completeness and correctness.

    Args:
        request_data: The OpenAI-compatible request to validate

    Returns:
        Dictionary with validation results
    """
    try:
        errors = []
        warnings = []

        # Check required fields
        if "messages" not in request_data:
            errors.append("Missing 'messages' field")
        elif not isinstance(request_data["messages"], list):
            errors.append("'messages' must be a list")
        elif len(request_data["messages"]) == 0:
            errors.append("'messages' cannot be empty")

        # Check message format
        if "messages" in request_data:
            for i, msg in enumerate(request_data["messages"]):
                if not isinstance(msg, dict):
                    errors.append(f"Message {i} is not a dictionary")
                    continue

                if "role" not in msg:
                    errors.append(f"Message {i} missing 'role' field")
                elif msg["role"] not in ["system", "user", "assistant"]:
                    warnings.append(f"Message {i} has unusual role: {msg['role']}")

                if "content" not in msg:
                    errors.append(f"Message {i} missing 'content' field")

        # Check model parameter
        model = request_data.get("model")
        if model and not isinstance(model, str):
            warnings.append("'model' should be a string")

        # Check temperature
        temp = request_data.get("temperature")
        if temp is not None:
            if not isinstance(temp, (int, float)):
                warnings.append("'temperature' should be a number")
            elif temp < 0 or temp > 2:
                warnings.append("'temperature' should be between 0 and 2")

        is_valid = len(errors) == 0

        return {
            "status": "success",
            "is_valid": is_valid,
            "errors": errors,
            "warnings": warnings,
            "validated_request": request_data if is_valid else None
        }

    except Exception as e:
        logger.error(f"Error validating request: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "is_valid": False
        }


def _initialize_mcp_components():
    """Initialize MCP components for preprocessing integration."""
    global _mcp_discovery, _mcp_tool_registry, _mcp_tool_selector

    try:
        if not _mcp_discovery:
            _mcp_discovery = MCPToolDiscovery(mcp_registry)
            logger.info("MCP Discovery initialized for preprocessing")

        if not _mcp_tool_registry:
            _mcp_tool_registry = MCPUnifiedToolRegistry(mcp_registry, _mcp_discovery)
            logger.info("MCP Tool Registry initialized for preprocessing")

        if not _mcp_tool_selector:
            _mcp_tool_selector = MCPToolSelector(_mcp_tool_registry, mcp_registry, _mcp_discovery)
            logger.info("MCP Tool Selector initialized for preprocessing")

    except Exception as e:
        logger.error(f"Error initializing MCP components: {e}")


async def discover_preprocessing_tools(request_data: Dict[str, Any], intent_analysis: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Discover available MCP tools for preprocessing phase.

    Args:
        request_data: The incoming request data
        intent_analysis: Optional intent analysis from reasoning service

    Returns:
        Dictionary with discovered tools and selection results
    """
    try:
        logger.debug("Discovering MCP tools for preprocessing")

        # Initialize MCP components if needed
        _initialize_mcp_components()

        if not _mcp_tool_selector:
            return {
                "status": "error",
                "error": "MCP Tool Selector not available",
                "available_tools": []
            }

        # Create tool selection context
        context = ToolSelectionContext(
            request_data=request_data,
            intent_analysis=intent_analysis,
            processing_phase=ProcessingPhase.PREPROCESSING
        )

        # Select appropriate tools
        selection_result = await _mcp_tool_selector.select_tools_for_context(context)

        if selection_result.get("status") == "success":
            tools = selection_result.get("selected_tools", [])
            logger.info(f"Discovered {len(tools)} preprocessing tools: {tools}")
        else:
            logger.warning(f"Tool discovery failed: {selection_result.get('error', 'Unknown error')}")

        return selection_result

    except Exception as e:
        logger.error(f"Error discovering preprocessing tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "available_tools": []
        }


async def enrich_context_with_mcp_tools(messages: List[Dict[str, Any]], available_tools: List[str], request_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Enrich context using selected MCP tools for preprocessing.

    Args:
        messages: List of conversation messages
        available_tools: List of selected MCP tool names
        request_data: Original request data

    Returns:
        Dictionary with enriched context and metadata
    """
    try:
        logger.debug(f"Enriching context with {len(available_tools)} MCP tools")

        if not available_tools or not _mcp_tool_selector:
            return {
                "status": "success",
                "enriched_context": {},
                "tools_executed": [],
                "reason": "No tools available for context enrichment"
            }

        # Create execution plan
        context = ToolSelectionContext(
            request_data=request_data,
            processing_phase=ProcessingPhase.PREPROCESSING
        )

        plan_result = await _mcp_tool_selector.create_execution_plan(available_tools, context)

        if plan_result.get("status") != "success":
            return {
                "status": "error",
                "error": f"Failed to create execution plan: {plan_result.get('error')}",
                "enriched_context": {}
            }

        execution_plan = plan_result.get("execution_plan")
        if not execution_plan:
            return {
                "status": "success",
                "enriched_context": {},
                "tools_executed": [],
                "reason": "No execution plan created"
            }

        # Execute tools
        execution_result = await _mcp_tool_selector.execute_tool_plan(execution_plan, context)

        if execution_result.get("status") != "success":
            return {
                "status": "error",
                "error": f"Tool execution failed: {execution_result.get('error')}",
                "enriched_context": {}
            }

        # Process results to create enriched context
        enriched_context = {}
        tools_executed = []

        for result in execution_result.get("results", []):
            if result.success:
                tools_executed.append(result.tool_name)
                # Add tool results to context
                if result.result:
                    enriched_context[f"{result.tool_name}_result"] = result.result

        logger.info(f"Context enriched using {len(tools_executed)} tools: {tools_executed}")

        return {
            "status": "success",
            "enriched_context": enriched_context,
            "tools_executed": tools_executed,
            "execution_stats": {
                "success_count": execution_result.get("success_count", 0),
                "total_count": execution_result.get("total_count", 0),
                "execution_time_ms": execution_result.get("execution_time_ms", 0)
            }
        }

    except Exception as e:
        logger.error(f"Error enriching context with MCP tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "enriched_context": {}
        }


def create_preprocessing_pipeline(request_data: Dict[str, Any], intent_analysis: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Create intelligent preprocessing pipeline with MCP tool integration.

    Args:
        request_data: The incoming request data
        intent_analysis: Optional intent analysis for better tool selection

    Returns:
        Dictionary with pipeline configuration and selected tools
    """
    try:
        logger.debug("Creating preprocessing pipeline with MCP integration")

        # Basic pipeline steps
        pipeline_steps = [
            "validate_request",
            "extract_metadata",
            "discover_mcp_tools",
            "enrich_context",
            "inject_context"
        ]

        # Determine if MCP tools should be used
        use_mcp_tools = True  # Can be configured based on request type

        if intent_analysis:
            # Adjust pipeline based on intent
            complexity = intent_analysis.get("complexity", "simple")
            domains = intent_analysis.get("domains", [])

            # Skip MCP tools for very simple requests
            if complexity == "simple" and not domains:
                use_mcp_tools = False
                pipeline_steps = [step for step in pipeline_steps if "mcp" not in step]

        pipeline_config = {
            "steps": pipeline_steps,
            "use_mcp_tools": use_mcp_tools,
            "parallel_execution": False,  # Sequential for now
            "timeout_ms": 30000,  # 30 seconds
            "enable_caching": True
        }

        logger.info(f"Created preprocessing pipeline with {len(pipeline_steps)} steps (MCP: {use_mcp_tools})")

        return {
            "status": "success",
            "pipeline_config": pipeline_config,
            "estimated_duration_ms": len(pipeline_steps) * 1000
        }

    except Exception as e:
        logger.error(f"Error creating preprocessing pipeline: {e}")
        return {
            "status": "error",
            "error": str(e),
            "pipeline_config": None
        }


async def execute_preprocessing_pipeline(request_data: Dict[str, Any], pipeline_config: Dict[str, Any], intent_analysis: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
    """
    Execute the preprocessing pipeline with MCP tool integration.

    Args:
        request_data: The incoming request data
        pipeline_config: Pipeline configuration
        intent_analysis: Optional intent analysis

    Returns:
        Dictionary with processed request and execution results
    """
    try:
        logger.info("Executing preprocessing pipeline")

        processed_request = request_data.copy()
        execution_results = {}

        steps = pipeline_config.get("steps", [])
        use_mcp_tools = pipeline_config.get("use_mcp_tools", False)

        # Step 1: Validate request
        if "validate_request" in steps:
            validation_result = validate_request(processed_request)
            execution_results["validation"] = validation_result

            if not validation_result.get("is_valid"):
                return {
                    "status": "error",
                    "error": "Request validation failed",
                    "validation_errors": validation_result.get("errors", []),
                    "processed_request": None
                }

        # Step 2: Extract metadata
        if "extract_metadata" in steps:
            metadata_result = extract_request_metadata(processed_request)
            execution_results["metadata"] = metadata_result

        # Step 3: Discover MCP tools
        available_tools = []
        if "discover_mcp_tools" in steps and use_mcp_tools:
            tool_discovery_result = await discover_preprocessing_tools(processed_request, intent_analysis)
            execution_results["tool_discovery"] = tool_discovery_result

            if tool_discovery_result.get("status") == "success":
                available_tools = tool_discovery_result.get("selected_tools", [])

        # Step 4: Enrich context with MCP tools
        enriched_context = {}
        if "enrich_context" in steps and available_tools:
            messages = processed_request.get("messages", [])
            enrichment_result = await enrich_context_with_mcp_tools(messages, available_tools, processed_request)
            execution_results["enrichment"] = enrichment_result

            if enrichment_result.get("status") == "success":
                enriched_context = enrichment_result.get("enriched_context", {})

        # Step 5: Inject context
        if "inject_context" in steps:
            messages = processed_request.get("messages", [])
            # Merge original context with MCP-enriched context
            combined_context = enriched_context.copy()
            if intent_analysis:
                combined_context["intent_analysis"] = intent_analysis

            context_result = inject_context(messages, combined_context)
            execution_results["context_injection"] = context_result

            if context_result.get("status") == "success":
                processed_request["messages"] = context_result.get("messages", messages)

        # Calculate total execution time
        total_time_ms = sum(
            result.get("execution_time_ms", 0)
            for result in execution_results.values()
            if isinstance(result, dict)
        )

        logger.info(f"Preprocessing pipeline completed in {total_time_ms}ms")

        return {
            "status": "success",
            "processed_request": processed_request,
            "execution_results": execution_results,
            "pipeline_stats": {
                "steps_executed": len([r for r in execution_results.values() if r.get("status") == "success"]),
                "total_steps": len(steps),
                "execution_time_ms": total_time_ms,
                "mcp_tools_used": len(available_tools)
            }
        }

    except Exception as e:
        logger.error(f"Error executing preprocessing pipeline: {e}")
        return {
            "status": "error",
            "error": str(e),
            "processed_request": request_data  # Return original on error
        }

# Create the preprocessing agent using new ADK API
from google.adk.agents import Agent

preprocessing_agent = Agent(
    name="preprocessing_agent",
    model="gemini-2.0-flash",
    tools=[
        inject_context,
        extract_request_metadata,
        validate_request,
        discover_preprocessing_tools,
        enrich_context_with_mcp_tools,
        create_preprocessing_pipeline,
        execute_preprocessing_pipeline
    ],
    instruction="""You are an intelligent preprocessing agent for an LLM reverse proxy server with MCP tool integration.

    Your responsibilities:
    1. Validate incoming requests using validate_request()
    2. Extract metadata from requests using extract_request_metadata()
    3. Discover and utilize MCP tools for context enrichment using discover_preprocessing_tools()
    4. Create intelligent preprocessing pipelines using create_preprocessing_pipeline()
    5. Execute preprocessing pipelines with MCP integration using execute_preprocessing_pipeline()
    6. Enrich context using selected MCP tools via enrich_context_with_mcp_tools()
    7. Inject enhanced context and system prompts using inject_context()

    Enhanced processing workflow:
    1. First validate the request to ensure it's properly formatted
    2. Extract metadata for analytics and routing decisions
    3. Analyze request intent to select appropriate MCP tools
    4. Create an intelligent preprocessing pipeline based on request complexity
    5. Execute the pipeline, including MCP tool discovery and execution
    6. Enrich context with results from MCP tools
    7. Inject the enhanced context into the request
    8. Return the processed request with MCP enrichment ready for forwarding to the LLM

    MCP Integration Features:
    - Intelligent tool selection based on request analysis and intent
    - Context enrichment using domain-specific tools (YouTrack, GitLab, RAG, etc.)
    - Parallel and sequential tool execution with error handling
    - Result caching and performance optimization
    - Fallback mechanisms for tool failures

    Always return structured data with clear status indicators and comprehensive error handling.""",
    description="Handles intelligent message preprocessing, validation, context injection, and MCP tool integration for LLM requests"
)
