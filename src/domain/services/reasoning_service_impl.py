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
import importlib
import sys
from pathlib import Path
from typing import Dict, List, Any, Optional, AsyncGenerator
from google.adk.agents import Agent
from google.adk.tools import ToolContext
from src.application.services.mcp_tool_selector import (
    MCPToolSelector, ToolSelectionContext, ProcessingPhase
)
from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.mcp.tool_registry import MCPUnifiedToolRegistry
from src.infrastructure.mcp.discovery import MCPToolDiscovery
from src.infrastructure.config.config import config

logger = logging.getLogger(__name__)

# Workflow callback cache
_workflow_callback = None

def load_workflow_callback():
    """
    Load the reasoning workflow callback from the configured path.

    Returns:
        The workflow callback function or None if not found
    """
    global _workflow_callback

    if _workflow_callback is not None:
        return _workflow_callback

    try:
        workflow_path = config.REASONING_WORKFLOW
        logger.info(f"🔄 Loading reasoning workflow from: {workflow_path}")

        # Convert path like "workflows/default" to module path
        # Support both relative and absolute paths
        if workflow_path.startswith("/"):
            # Absolute path
            module_path = Path(workflow_path)
        else:
            # Relative path from project root
            project_root = Path(__file__).parent.parent.parent.parent
            module_path = project_root / workflow_path

        # Add the workflow directory to sys.path if not already there
        workflow_dir = module_path.parent
        if str(workflow_dir) not in sys.path:
            sys.path.insert(0, str(workflow_dir))

        # Import the module
        module_name = module_path.name
        module = importlib.import_module(module_name)

        # Get the reasoning_workflow function
        if hasattr(module, 'reasoning_workflow'):
            _workflow_callback = module.reasoning_workflow
            logger.info(f"✅ Successfully loaded workflow callback from {workflow_path}")
            return _workflow_callback
        else:
            logger.error(f"❌ Module {module_name} does not have 'reasoning_workflow' function")
            return None

    except Exception as e:
        logger.error(f"❌ Error loading workflow callback from {workflow_path}: {e}")
        logger.info("⚠️  Falling back to default workflow")
        return None

# Global MCP integration components for reasoning
_reasoning_mcp_discovery = None
_reasoning_mcp_tool_registry = None
_reasoning_mcp_tool_selector = None

def analyze_request_intent(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """Analyze the user's request to understand intent and complexity with enhanced domain detection."""
    try:
        logger.debug("🧠 REASONING: Analyzing request intent with enhanced detection")

        messages = request_data.get("messages", [])
        if not messages:
            return {"status": "error", "error": "No messages to analyze"}

        # Get the last user message
        user_messages = [msg for msg in messages if msg.get("role") == "user"]
        if not user_messages:
            return {"status": "error", "error": "No user messages found"}

        last_message = user_messages[-1].get("content", "")
        message_lower = last_message.lower()

        # Enhanced intent analysis
        intent_analysis = {
            "message_length": len(last_message),
            "word_count": len(last_message.split()),
            "contains_question": "?" in last_message,
            "contains_request": any(word in message_lower for word in ["please", "can you", "help", "create", "make", "build", "show", "get", "find", "list"]),
            "complexity": "simple" if len(last_message.split()) < 10 else "complex",
            "domains": [],
            "needs_context": len(messages) > 2,
            "intent_type": "unknown",
            "target_tools": [],
            "action_verbs": [],
            "entities": []
        }

        # Enhanced domain detection with specific patterns
        domain_patterns = {
            "programming": ["code", "programming", "function", "api", "script", "debug", "compile", "repository", "commit", "branch"],
            "explanation": ["explain", "what", "how", "why", "describe", "tell me", "show me how"],
            "creation": ["create", "generate", "build", "make", "develop", "implement", "design"],
            "project_management": ["ticket", "tickets", "task", "tasks", "epic", "epics", "issue", "issues", "project", "assigned", "assign", "milestone", "sprint"],
            "data_retrieval": ["show", "get", "fetch", "find", "list", "display", "retrieve", "search"],
            "analysis": ["analyze", "report", "metrics", "statistics", "performance", "status"],
            "version_control": ["git", "gitlab", "github", "commit", "merge", "branch", "repository", "repo", "pull request", "pr"]
        }

        # Detect domains and extract entities
        for domain, keywords in domain_patterns.items():
            if any(keyword in message_lower for keyword in keywords):
                intent_analysis["domains"].append(domain)

        # Detect intent types and target tools
        if any(word in message_lower for word in ["ticket", "tickets", "assigned", "task", "epic"]):
            intent_analysis["intent_type"] = "task_management"
            intent_analysis["target_tools"].append("youtrack")

            # Detect specific YouTrack actions
            if any(phrase in message_lower for phrase in ["assigned to me", "my tickets", "my tasks"]):
                intent_analysis["action_verbs"].append("find_assigned")
                intent_analysis["entities"].append("current_user")
            elif "create" in message_lower:
                intent_analysis["action_verbs"].append("create_ticket")

        elif any(word in message_lower for word in ["repository", "repo", "commit", "branch", "gitlab", "git"]):
            intent_analysis["intent_type"] = "version_control"
            intent_analysis["target_tools"].append("gitlab")

            # Detect specific GitLab actions
            if any(word in message_lower for word in ["analyze", "metrics", "statistics"]):
                intent_analysis["action_verbs"].append("analyze_repository")
            elif "list" in message_lower or "show" in message_lower:
                intent_analysis["action_verbs"].append("list_repositories")

        elif any(word in message_lower for word in ["file", "files", "directory", "folder", "filesystem"]):
            intent_analysis["intent_type"] = "file_management"
            intent_analysis["target_tools"].append("filesystem")

        # Extract action verbs for general understanding
        action_verbs = ["show", "get", "find", "list", "create", "update", "delete", "search", "analyze"]
        for verb in action_verbs:
            if verb in message_lower and verb not in intent_analysis["action_verbs"]:
                intent_analysis["action_verbs"].append(verb)

        # Set default intent type if not detected
        if intent_analysis["intent_type"] == "unknown":
            if intent_analysis["action_verbs"]:
                intent_analysis["intent_type"] = "general_query"
            else:
                intent_analysis["intent_type"] = "conversation"

        logger.debug(f"🧠 Enhanced intent analysis: {intent_analysis}")

        return {
            "status": "success",
            "intent_analysis": intent_analysis,
            "original_message": last_message
        }

    except Exception as e:
        logger.error(f"❌ Error analyzing request intent: {e}")
        return {"status": "error", "error": str(e)}

def generate_reasoning_context(intent_analysis: Dict[str, Any], messages: List[Dict[str, Any]], reasoning_insights: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
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

        # Add MCP tool insights if available
        if reasoning_insights:
            reasoning_context.append("MCP Tool Insights: Enhanced with external tool analysis:")
            for insight_key, insight_value in reasoning_insights.items():
                if isinstance(insight_value, dict):
                    # Format structured insights
                    insight_summary = f"  - {insight_key}: {insight_value.get('summary', str(insight_value)[:100])}"
                else:
                    insight_summary = f"  - {insight_key}: {str(insight_value)[:100]}"
                reasoning_context.append(insight_summary)

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


def _initialize_reasoning_mcp_components():
    """Initialize MCP components for reasoning integration."""
    global _reasoning_mcp_discovery, _reasoning_mcp_tool_registry, _reasoning_mcp_tool_selector

    try:
        if not _reasoning_mcp_discovery:
            _reasoning_mcp_discovery = MCPToolDiscovery(mcp_registry)
            logger.info("MCP Discovery initialized for reasoning")

        if not _reasoning_mcp_tool_registry:
            _reasoning_mcp_tool_registry = MCPUnifiedToolRegistry(mcp_registry, _reasoning_mcp_discovery)
            logger.info("MCP Tool Registry initialized for reasoning")

        if not _reasoning_mcp_tool_selector:
            _reasoning_mcp_tool_selector = MCPToolSelector(_reasoning_mcp_tool_registry, mcp_registry, _reasoning_mcp_discovery)
            logger.info("MCP Tool Selector initialized for reasoning")

    except Exception as e:
        logger.error(f"Error initializing reasoning MCP components: {e}")


async def _ensure_mcp_discovery_populated():
    """Ensure MCP tool discovery has been run and tools are available."""
    global _reasoning_mcp_discovery

    try:
        if _reasoning_mcp_discovery:
            # Check if we already have tools discovered
            all_tools = _reasoning_mcp_discovery.get_all_tools()
            if not all_tools:
                logger.info("🔍 REASONING: Running MCP tool discovery to populate tools")
                await _reasoning_mcp_discovery.discover_all_capabilities()

                # Check results
                all_tools = _reasoning_mcp_discovery.get_all_tools()
                logger.info(f"🔍 REASONING: Discovery complete - found {len(all_tools)} tools")

                for tool in all_tools:
                    logger.debug(f"  - {tool.server_name}: {tool.name}")
            else:
                logger.debug(f"🔍 REASONING: Using existing {len(all_tools)} discovered tools")

    except Exception as e:
        logger.error(f"❌ Error ensuring MCP discovery: {e}")


async def discover_reasoning_tools(request_data: Dict[str, Any], intent_analysis: Dict[str, Any]) -> Dict[str, Any]:
    """
    Discover and select MCP tools that can enhance reasoning with intelligent tool targeting.

    Args:
        request_data: The request data for context
        intent_analysis: Intent analysis results

    Returns:
        Dictionary with discovered reasoning tools
    """
    try:
        logger.debug("🧠 REASONING: Discovering MCP tools with intelligent targeting")

        # Initialize MCP components if needed
        _initialize_reasoning_mcp_components()

        # Ensure MCP discovery is populated with tools
        await _ensure_mcp_discovery_populated()

        if not _reasoning_mcp_tool_selector:
            return {
                "status": "error",
                "error": "MCP Tool Selector not available",
                "reasoning_tools": []
            }

        # Enhanced tool discovery based on intent analysis
        target_tools = intent_analysis.get("target_tools", [])
        intent_type = intent_analysis.get("intent_type", "unknown")
        action_verbs = intent_analysis.get("action_verbs", [])

        logger.debug(f"🧠 Intent-based targeting: type={intent_type}, tools={target_tools}, actions={action_verbs}")

        # Create tool selection context for reasoning phase
        context = ToolSelectionContext(
            request_data=request_data,
            intent_analysis=intent_analysis,
            processing_phase=ProcessingPhase.REASONING
        )

        # Select appropriate reasoning tools with enhanced targeting
        selection_result = await _reasoning_mcp_tool_selector.select_tools_for_context(context)

        if selection_result.get("status") == "success":
            tools = selection_result.get("selected_tools", [])
            logger.debug(f"🧠 Discovered {len(tools)} reasoning tools: {tools}")

            # Enhanced tool filtering based on intent
            reasoning_tools = []
            preferred_tools = []

            for tool_name in tools:
                tool_info = _reasoning_mcp_tool_registry.get_tool_info(tool_name)
                if not tool_info:
                    continue

                tool_description = tool_info.get("description", "").lower()
                server_name = tool_info.get("server_name", "")

                # Prioritize tools based on intent targeting
                if target_tools:
                    for target in target_tools:
                        if target in server_name.lower() or target in tool_name.lower():
                            preferred_tools.append(tool_name)
                            logger.debug(f"🎯 Preferred tool found: {tool_name} (matches {target})")
                            break

                # Include tools that match general reasoning criteria
                if any(keyword in tool_description for keyword in ["analyze", "understand", "interpret", "reason", "think", "find", "search", "list", "get"]):
                    if tool_name not in preferred_tools:
                        reasoning_tools.append(tool_name)

            # Combine preferred tools first, then general reasoning tools
            final_tools = preferred_tools + reasoning_tools

            # Remove duplicates while preserving order
            unique_tools = []
            seen = set()
            for tool in final_tools:
                if tool not in seen:
                    unique_tools.append(tool)
                    seen.add(tool)

            logger.info(f"🧠 Tool selection: {len(preferred_tools)} preferred + {len(reasoning_tools)} general = {len(unique_tools)} total")

            return {
                "status": "success",
                "reasoning_tools": unique_tools,
                "preferred_tools": preferred_tools,
                "general_tools": reasoning_tools,
                "all_selected_tools": tools,
                "selection_metadata": selection_result,
                "intent_targeting": {
                    "target_tools": target_tools,
                    "intent_type": intent_type,
                    "action_verbs": action_verbs
                }
            }
        else:
            logger.warning(f"🧠 Reasoning tool discovery failed: {selection_result.get('error', 'Unknown error')}")
            return selection_result

    except Exception as e:
        logger.error(f"❌ Error discovering reasoning tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "reasoning_tools": []
        }


async def generate_execution_plan(request_data: Dict[str, Any], intent_analysis: Dict[str, Any], reasoning_tools: List[str]) -> Dict[str, Any]:
    """
    Generate an intelligent execution plan based on intent analysis and available tools.

    Args:
        request_data: The request data
        intent_analysis: Intent analysis results
        reasoning_tools: List of available reasoning tools

    Returns:
        Dictionary with execution plan
    """
    try:
        logger.debug("🧠 REASONING: Generating intelligent execution plan")

        intent_type = intent_analysis.get("intent_type", "unknown")
        action_verbs = intent_analysis.get("action_verbs", [])
        target_tools = intent_analysis.get("target_tools", [])
        entities = intent_analysis.get("entities", [])

        execution_plan = {
            "intent_type": intent_type,
            "steps": [],
            "tool_sequence": [],
            "expected_output": "",
            "reasoning": ""
        }

        # Generate plan based on intent type
        if intent_type == "task_management":
            if "find_assigned" in action_verbs and "current_user" in entities:
                execution_plan.update({
                    "steps": [
                        "Authenticate with YouTrack API",
                        "Retrieve current user information",
                        "Find tickets assigned to current user",
                        "Format and present results"
                    ],
                    "tool_sequence": ["find_assigned_tickets", "get_user_details"] if "youtrack" in target_tools else [],
                    "expected_output": "List of tickets assigned to the current user with details like title, status, and priority",
                    "reasoning": "User wants to see their assigned tickets, so we'll use YouTrack MCP tools to find and retrieve this information"
                })
            elif "create_ticket" in action_verbs:
                execution_plan.update({
                    "steps": [
                        "Extract ticket details from request",
                        "Validate ticket information",
                        "Create new ticket in YouTrack",
                        "Confirm creation and return ticket details"
                    ],
                    "tool_sequence": ["create_ticket"] if "youtrack" in target_tools else [],
                    "expected_output": "Confirmation of ticket creation with ticket ID and details",
                    "reasoning": "User wants to create a new ticket, so we'll use YouTrack MCP tools to create it"
                })

        elif intent_type == "version_control":
            if "analyze_repository" in action_verbs:
                execution_plan.update({
                    "steps": [
                        "Connect to GitLab API",
                        "Analyze repository metrics",
                        "Generate repository statistics",
                        "Present analysis results"
                    ],
                    "tool_sequence": ["analyze_repository", "get_repository_stats"] if "gitlab" in target_tools else [],
                    "expected_output": "Repository analysis including commits, contributors, and code metrics",
                    "reasoning": "User wants repository analysis, so we'll use GitLab MCP tools to gather and analyze repository data"
                })
            elif "list_repositories" in action_verbs:
                execution_plan.update({
                    "steps": [
                        "Connect to GitLab API",
                        "Retrieve user's repositories",
                        "Format repository list",
                        "Present results"
                    ],
                    "tool_sequence": ["list_repositories"] if "gitlab" in target_tools else [],
                    "expected_output": "List of repositories with basic information",
                    "reasoning": "User wants to see repositories, so we'll use GitLab MCP tools to list them"
                })

        elif intent_type == "file_management":
            execution_plan.update({
                "steps": [
                    "Analyze file system request",
                    "Execute file operations",
                    "Return file information"
                ],
                "tool_sequence": ["list_files", "read_file"] if "filesystem" in target_tools else [],
                "expected_output": "File system information or file contents",
                "reasoning": "User wants file system operations, so we'll use filesystem MCP tools"
            })

        elif intent_type == "general_query":
            execution_plan.update({
                "steps": [
                    "Analyze query context",
                    "Determine appropriate tools",
                    "Execute relevant operations",
                    "Synthesize results"
                ],
                "tool_sequence": reasoning_tools[:3],  # Use first 3 reasoning tools
                "expected_output": "Contextual response based on available information",
                "reasoning": "General query requires analysis of available tools and context"
            })

        else:  # conversation or unknown
            execution_plan.update({
                "steps": [
                    "Process conversational request",
                    "Provide direct response"
                ],
                "tool_sequence": [],
                "expected_output": "Direct conversational response",
                "reasoning": "Simple conversation that doesn't require external tools"
            })

        # Filter tool sequence to only include available tools
        available_tool_names = [tool.lower() for tool in reasoning_tools]
        filtered_sequence = []
        for tool in execution_plan["tool_sequence"]:
            # Check if any available tool contains this tool name
            if any(tool.lower() in available_tool.lower() for available_tool in reasoning_tools):
                # Find the exact match
                for available_tool in reasoning_tools:
                    if tool.lower() in available_tool.lower():
                        filtered_sequence.append(available_tool)
                        break

        execution_plan["tool_sequence"] = filtered_sequence

        logger.info(f"🧠 Generated execution plan: {len(execution_plan['steps'])} steps, {len(execution_plan['tool_sequence'])} tools")

        return {
            "status": "success",
            "execution_plan": execution_plan
        }

    except Exception as e:
        logger.error(f"❌ Error generating execution plan: {e}")
        return {
            "status": "error",
            "error": str(e)
        }


async def execute_reasoning_tools(request_data: Dict[str, Any], reasoning_tools: List[str], intent_analysis: Dict[str, Any]) -> Dict[str, Any]:
    """
    Execute reasoning tools with intelligent orchestration based on execution plan.

    Args:
        request_data: The request data
        reasoning_tools: List of selected reasoning tool names
        intent_analysis: Intent analysis results

    Returns:
        Dictionary with reasoning tool execution results
    """
    try:
        logger.debug(f"🧠 REASONING: Executing {len(reasoning_tools)} reasoning tools with intelligent orchestration")

        if not reasoning_tools or not _reasoning_mcp_tool_selector:
            return {
                "status": "success",
                "reasoning_insights": {},
                "tools_executed": [],
                "execution_plan": None,
                "reason": "No reasoning tools to execute"
            }

        # Generate intelligent execution plan
        plan_generation_result = await generate_execution_plan(request_data, intent_analysis, reasoning_tools)

        if plan_generation_result.get("status") != "success":
            logger.warning(f"Failed to generate execution plan: {plan_generation_result.get('error')}")
            # Fall back to basic execution
            execution_plan = {
                "intent_type": "unknown",
                "steps": ["Execute available tools"],
                "tool_sequence": reasoning_tools,
                "expected_output": "Tool execution results",
                "reasoning": "Fallback execution without plan"
            }
        else:
            execution_plan = plan_generation_result["execution_plan"]

        logger.info(f"🧠 Execution plan: {execution_plan['intent_type']} → {len(execution_plan['tool_sequence'])} tools")

        # Create execution context
        context = ToolSelectionContext(
            request_data=request_data,
            intent_analysis=intent_analysis,
            processing_phase=ProcessingPhase.REASONING
        )

        # Use planned tool sequence if available, otherwise use all reasoning tools
        tools_to_execute = execution_plan["tool_sequence"] if execution_plan["tool_sequence"] else reasoning_tools

        if not tools_to_execute:
            return {
                "status": "success",
                "reasoning_insights": {},
                "tools_executed": [],
                "execution_plan": execution_plan,
                "reason": "No tools selected for execution based on plan"
            }

        # Create MCP execution plan
        mcp_plan_result = await _reasoning_mcp_tool_selector.create_execution_plan(tools_to_execute, context)

        if mcp_plan_result.get("status") != "success":
            return {
                "status": "error",
                "error": f"Failed to create MCP execution plan: {mcp_plan_result.get('error')}",
                "reasoning_insights": {},
                "execution_plan": execution_plan
            }

        mcp_execution_plan = mcp_plan_result.get("execution_plan")
        if not mcp_execution_plan:
            return {
                "status": "success",
                "reasoning_insights": {},
                "tools_executed": [],
                "execution_plan": execution_plan,
                "reason": "No MCP execution plan created"
            }

        # Execute reasoning tools according to plan
        execution_result = await _reasoning_mcp_tool_selector.execute_tool_plan(mcp_execution_plan, context)

        if execution_result.get("status") != "success":
            return {
                "status": "error",
                "error": f"Reasoning tool execution failed: {execution_result.get('error')}",
                "reasoning_insights": {},
                "execution_plan": execution_plan
            }

        # Process results to extract reasoning insights with plan context
        reasoning_insights = {}
        tools_executed = []
        plan_context = {
            "intent_type": execution_plan["intent_type"],
            "expected_output": execution_plan["expected_output"],
            "reasoning": execution_plan["reasoning"]
        }

        for result in execution_result.get("results", []):
            if result.success and result.result:
                tools_executed.append(result.tool_name)

                # Enhanced insight processing with plan context
                insight_key = f"{result.tool_name}_insight"
                insight_value = {
                    "result": result.result,
                    "tool_name": result.tool_name,
                    "execution_time_ms": result.execution_time_ms,
                    "plan_context": plan_context
                }

                # Add specific formatting based on intent type
                if execution_plan["intent_type"] == "task_management":
                    if "find_assigned" in intent_analysis.get("action_verbs", []):
                        insight_value["formatted_for"] = "ticket_list_display"
                elif execution_plan["intent_type"] == "version_control":
                    insight_value["formatted_for"] = "repository_analysis"

                reasoning_insights[insight_key] = insight_value

        logger.info(f"🧠 Intelligent execution completed: {len(tools_executed)} tools, plan type: {execution_plan['intent_type']}")

        return {
            "status": "success",
            "reasoning_insights": reasoning_insights,
            "tools_executed": tools_executed,
            "execution_plan": execution_plan,
            "execution_stats": {
                "success_count": execution_result.get("success_count", 0),
                "total_count": execution_result.get("total_count", 0),
                "execution_time_ms": execution_result.get("execution_time_ms", 0),
                "plan_based": True
            }
        }

    except Exception as e:
        logger.error(f"❌ Error executing reasoning tools: {e}")
        return {
            "status": "error",
            "error": str(e),
            "reasoning_insights": {},
            "execution_plan": None
        }


async def validate_reasoning_results(reasoning_metadata: Dict[str, Any], enhanced_request: Dict[str, Any]) -> Dict[str, Any]:
    """
    Validate reasoning results to ensure quality and correctness.

    Args:
        reasoning_metadata: Metadata from the reasoning process
        enhanced_request: The enhanced request after reasoning

    Returns:
        Dictionary with validation results
    """
    try:
        logger.debug("🧠 REASONING: Validating reasoning results")

        validation_results = {
            "overall_status": "valid",
            "confidence_score": 0.0,
            "validation_checks": [],
            "issues": [],
            "recommendations": []
        }

        total_checks = 0
        passed_checks = 0

        # Check 1: Intent analysis quality
        intent_analysis = reasoning_metadata.get("intent_analysis", {})
        if intent_analysis:
            total_checks += 1
            if (intent_analysis.get("complexity") and
                intent_analysis.get("word_count", 0) > 0):
                passed_checks += 1
                validation_results["validation_checks"].append("Intent analysis complete")
            else:
                validation_results["issues"].append("Intent analysis incomplete or missing data")

        # Check 2: Reasoning context enhancement
        reasoning_context = reasoning_metadata.get("reasoning_context", [])
        if reasoning_context:
            total_checks += 1
            if len(reasoning_context) >= 2:
                passed_checks += 1
                validation_results["validation_checks"].append("Reasoning context adequately enriched")
            else:
                validation_results["issues"].append("Reasoning context seems minimal")

        # Check 3: Message enhancement validation
        original_count = reasoning_metadata.get("original_message_count", 0)
        enhanced_count = reasoning_metadata.get("enhanced_message_count", 0)

        if original_count > 0:
            total_checks += 1
            if enhanced_count >= original_count:
                passed_checks += 1
                validation_results["validation_checks"].append("Message structure properly enhanced")
            else:
                validation_results["issues"].append("Message count decreased during reasoning")
                validation_results["recommendations"].append("Review message enhancement logic")

        # Check 4: MCP tools integration validation
        mcp_tools_used = reasoning_metadata.get("mcp_tools_used", 0)
        reasoning_insights = reasoning_metadata.get("reasoning_insights", {})

        total_checks += 1
        if mcp_tools_used > 0 and reasoning_insights:
            passed_checks += 1
            validation_results["validation_checks"].append(f"MCP tools successfully integrated ({mcp_tools_used} tools)")
        elif mcp_tools_used == 0:
            # This is okay if no suitable tools were available
            passed_checks += 1
            validation_results["validation_checks"].append("MCP integration handled appropriately")
        else:
            validation_results["issues"].append("MCP tools used but no insights captured")

        # Check 5: Enhanced request structure validation
        enhanced_messages = enhanced_request.get("messages", [])
        if enhanced_messages:
            total_checks += 1

            # Check for system message with reasoning context
            has_system_message = any(msg.get("role") == "system" for msg in enhanced_messages)
            system_content_length = 0

            if has_system_message:
                for msg in enhanced_messages:
                    if msg.get("role") == "system":
                        system_content_length += len(msg.get("content", ""))

            if has_system_message and system_content_length > 100:  # Reasonable minimum
                passed_checks += 1
                validation_results["validation_checks"].append("Enhanced request has substantial system context")
            else:
                validation_results["issues"].append("Enhanced request lacks sufficient system context")
                validation_results["recommendations"].append("Ensure reasoning context is properly injected")

        # Check 6: Reasoning pipeline completeness
        if "intent_analysis" in reasoning_metadata and "reasoning_context" in reasoning_metadata:
            total_checks += 1
            passed_checks += 1
            validation_results["validation_checks"].append("Reasoning pipeline completed all major steps")
        else:
            total_checks += 1
            validation_results["issues"].append("Reasoning pipeline appears incomplete")

        # Calculate confidence score
        if total_checks > 0:
            base_confidence = passed_checks / total_checks

            # Adjust confidence based on additional factors
            if len(validation_results["issues"]) == 0:
                validation_results["confidence_score"] = min(1.0, base_confidence + 0.1)
            elif len(validation_results["issues"]) <= 2:
                validation_results["confidence_score"] = base_confidence
            else:
                validation_results["confidence_score"] = max(0.1, base_confidence - 0.2)
        else:
            validation_results["confidence_score"] = 0.5  # Default neutral confidence

        # Determine overall status
        if validation_results["confidence_score"] >= 0.8:
            validation_results["overall_status"] = "valid"
        elif validation_results["confidence_score"] >= 0.6:
            validation_results["overall_status"] = "acceptable"
        else:
            validation_results["overall_status"] = "invalid"

        # Add quality recommendations
        if validation_results["confidence_score"] < 0.8:
            if not reasoning_insights:
                validation_results["recommendations"].append("Consider enabling more MCP tools for reasoning enhancement")
            if len(reasoning_context) < 3:
                validation_results["recommendations"].append("Enhance reasoning context with more detailed analysis")

        logger.debug(f"🧠 Reasoning validation completed: {validation_results['overall_status']} (confidence: {validation_results['confidence_score']:.2f})")

        return {
            "status": "success",
            "validation_results": validation_results,
            "is_valid": validation_results["overall_status"] in ["valid", "acceptable"],
            "confidence_score": validation_results["confidence_score"]
        }

    except Exception as e:
        logger.error(f"❌ Error validating reasoning results: {e}")
        return {
            "status": "error",
            "error": str(e),
            "is_valid": False,
            "confidence_score": 0.0
        }


async def enhance_reasoning_with_validation(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Apply reasoning to request with validation of results.

    Args:
        request_data: The original request data

    Returns:
        Dictionary with enhanced request and validation results
    """
    try:
        logger.debug("🧠 REASONING: Applying reasoning with validation")

        # Apply standard reasoning
        reasoning_result = await apply_reasoning_to_request(request_data)

        if reasoning_result.get("status") != "success":
            return reasoning_result

        # Validate reasoning results
        reasoning_metadata = reasoning_result.get("reasoning_metadata", {})
        enhanced_request = reasoning_result.get("enhanced_request", request_data)

        validation_result = await validate_reasoning_results(reasoning_metadata, enhanced_request)

        # Combine results
        return {
            "status": "success",
            "enhanced_request": enhanced_request,
            "reasoning_metadata": reasoning_metadata,
            "validation": validation_result,
            "quality_assured": validation_result.get("is_valid", False)
        }

    except Exception as e:
        logger.error(f"❌ Error in validated reasoning: {e}")
        return {
            "status": "error",
            "error": str(e),
            "enhanced_request": request_data
        }

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
        elif step_name == "mcp_tool_discovery":
            tools_found = step_data.get("tools_found", 0)
            preferred = step_data.get("preferred_tools", 0)
            if preferred > 0:
                content = f"🧠 **Tool Discovery**: Found {tools_found} tools ({preferred} preferred for intent)\n"
            else:
                content = f"🧠 **Tool Discovery**: Found {tools_found} reasoning tools\n"
        elif step_name == "execution_planning":
            intent_type = step_data.get("intent_type", "unknown")
            steps_planned = step_data.get("steps_planned", 0)
            tools_selected = step_data.get("tools_selected", 0)
            plan_reasoning = step_data.get("plan_reasoning", "")
            content = f"🧠 **Plan**: {intent_type.replace('_', ' ').title()} → {steps_planned} steps, {tools_selected} tools\n"
            if plan_reasoning:
                content += f"🧠 **Strategy**: {plan_reasoning}\n"
        elif step_name == "mcp_tool_execution":
            tools_executed = step_data.get("tools_executed", 0)
            insights_gathered = step_data.get("insights_gathered", 0)
            plan_type = step_data.get("plan_type", "unknown")
            content = f"🧠 **Execution**: {tools_executed} tools executed, {insights_gathered} insights gathered ({plan_type})\n"
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
    Execute reasoning pipeline using configured workflow callback.

    Workflows are loaded from the path specified in config.REASONING_WORKFLOW.
    """
    try:
        logger.info("🧠 REASONING: Starting reasoning pipeline")

        # Load and use workflow callback
        workflow_callback = load_workflow_callback()
        if workflow_callback:
            logger.info("🔄 Using workflow callback")
            try:
                async for chunk in workflow_callback(
                    request_data,
                    analyze_request_intent,
                    generate_reasoning_context,
                    enhance_messages_with_reasoning,
                    discover_reasoning_tools,
                    execute_reasoning_tools,
                    stream_reasoning_step
                ):
                    yield chunk
                return
            except Exception as e:
                logger.error(f"❌ Error executing workflow callback: {e}")
                raise

        # If no workflow callback loaded, raise an error
        logger.error("❌ No workflow callback loaded")
        raise RuntimeError("No reasoning workflow configured. Set 'reasoning_workflow' in config.yaml")

    except Exception as e:
        logger.error(f"❌ Error in reasoning pipeline: {e}")
        yield await stream_reasoning_step("error", {"status": "failed", "error": str(e)}, None)

async def apply_reasoning_to_request(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Apply reasoning enhancements to the request data.
    This is the non-streaming version (consumes workflow chunks without streaming them).
    """
    try:
        logger.debug("🧠 REASONING: Applying reasoning to request (non-streaming)")

        # Load workflow callback
        workflow_callback = load_workflow_callback()
        if not workflow_callback:
            logger.error("❌ No workflow callback loaded")
            return {"status": "error", "error": "No reasoning workflow configured"}

        # Run workflow pipeline (consume all chunks but don't yield)
        try:
            async for chunk in workflow_callback(
                request_data,
                analyze_request_intent,
                generate_reasoning_context,
                enhance_messages_with_reasoning,
                discover_reasoning_tools,
                execute_reasoning_tools,
                stream_reasoning_step
            ):
                pass  # Consume all chunks

            # Extract enhanced request from the workflow
            # The workflow should have enhanced the messages
            # For now, use the standard approach
            intent_result = analyze_request_intent(request_data)
            if intent_result.get("status") != "success":
                return {"status": "error", "error": f"Intent analysis failed: {intent_result.get('error')}"}

            # Discover and execute reasoning tools
            reasoning_insights = {}
            try:
                tool_discovery_result = await discover_reasoning_tools(request_data, intent_result["intent_analysis"])
                if tool_discovery_result.get("status") == "success":
                    reasoning_tools = tool_discovery_result.get("reasoning_tools", [])
                    if reasoning_tools:
                        tool_execution_result = await execute_reasoning_tools(request_data, reasoning_tools, intent_result["intent_analysis"])
                        if tool_execution_result.get("status") == "success":
                            reasoning_insights = tool_execution_result.get("reasoning_insights", {})
            except Exception as e:
                logger.warning(f"MCP tool execution failed in reasoning: {e}")

            # Generate context with insights
            messages = request_data.get("messages", [])
            context_result = generate_reasoning_context(intent_result["intent_analysis"], messages, reasoning_insights)
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
                    "reasoning_insights": reasoning_insights,
                    "original_message_count": len(messages),
                    "enhanced_message_count": len(enhancement_result["enhanced_messages"]),
                    "mcp_tools_used": len(reasoning_insights)
                }
            }

        except Exception as e:
            logger.error(f"❌ Error executing workflow: {e}")
            return {"status": "error", "error": str(e)}

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
                apply_reasoning_to_request,
                discover_reasoning_tools,
                execute_reasoning_tools,
                validate_reasoning_results,
                enhance_reasoning_with_validation
            ]
        )
        logger.info("✅ Reasoning agent created successfully")
        return reasoning_agent
    except Exception as e:
        logger.error(f"❌ Error creating reasoning agent: {e}")
        return None

# Global reasoning agent instance
reasoning_agent = create_reasoning_agent()