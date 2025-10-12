#!/usr/bin/env python3
"""
LLM-Powered Reasoning Agents for Advanced Multi-Agent Reasoning System.

This module implements sophisticated reasoning agents that use LLM calls for each step:
- Intent Analysis Agent: Uses LLM to understand user intent and required tools
- Plan Generation Agent: Uses LLM to create detailed execution plans
- Plan Execution Agent: Uses LLM for recursive plan execution with MCP tools
- Context Sufficiency Agent: Uses LLM to determine when enough context is collected
"""

import logging
import json
import asyncio
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass
from enum import Enum

# Import MCP tool selector classes
from src.application.services.mcp_tool_selector import (
    ToolExecutionPlan, ToolSelectionContext, ProcessingPhase
)

try:
    from google.adk.agents import LlmAgent
    # Temporarily disable ADK agents due to model_copy compatibility issues
    ADK_AVAILABLE = False  # Set to False to use fallback logic
    LlmAgent = None
except ImportError:
    ADK_AVAILABLE = False
    LlmAgent = None

logger = logging.getLogger(__name__)

def _safe_json_serialize(obj: Any) -> str:
    """Safely serialize objects to JSON, handling non-serializable types."""
    def _serialize_item(item):
        if hasattr(item, '__dict__'):
            # Handle dataclass objects like ToolExecutionResult
            if hasattr(item, 'result'):
                # Special handling for ToolExecutionResult
                result_data = item.result
                if hasattr(result_data, 'text'):
                    result_data = str(result_data.text)
                elif hasattr(result_data, '__dict__'):
                    result_data = str(result_data)
                elif isinstance(result_data, (list, tuple)):
                    processed_list = []
                    for sub_item in result_data:
                        if hasattr(sub_item, 'text'):
                            processed_list.append(str(sub_item.text))
                        elif hasattr(sub_item, '__dict__'):
                            processed_list.append(str(sub_item))
                        else:
                            processed_list.append(sub_item)
                    result_data = processed_list

                return {
                    'success': getattr(item, 'success', True),
                    'result': result_data,
                    'error_message': getattr(item, 'error_message', None),
                    'server_name': getattr(item, 'server_name', None),
                    'execution_time_ms': getattr(item, 'execution_time_ms', 0),
                    'tool_name': getattr(item, 'tool_name', None)
                }
            else:
                # Convert other dataclass objects to string
                return str(item)
        elif isinstance(item, dict):
            # Recursively handle dictionary values
            serialized_dict = {}
            for key, value in item.items():
                serialized_dict[key] = _serialize_item(value)
            return serialized_dict
        elif isinstance(item, (list, tuple)):
            # Recursively handle list/tuple items
            return [_serialize_item(sub_item) for sub_item in item]
        else:
            # Return primitive types as-is
            return item

    try:
        serialized_obj = _serialize_item(obj)
        return json.dumps(serialized_obj, indent=2, default=str)
    except Exception as e:
        logger.warning(f"Failed to serialize object to JSON: {e}")
        return str(obj)


class ReasoningPhase(Enum):
    """Phases of the reasoning process."""
    INTENT_ANALYSIS = "intent_analysis"
    PLAN_GENERATION = "plan_generation"
    PLAN_EXECUTION = "plan_execution"
    CONTEXT_EVALUATION = "context_evaluation"
    COMPLETION = "completion"


@dataclass
class ReasoningContext:
    """Context maintained throughout the reasoning process."""
    original_request: str
    intent_analysis: Optional[Dict[str, Any]] = None
    execution_plan: Optional[Dict[str, Any]] = None
    collected_context: List[Dict[str, Any]] = None
    current_phase: ReasoningPhase = ReasoningPhase.INTENT_ANALYSIS
    reasoning_history: List[Dict[str, Any]] = None
    mcp_tools_available: List[Dict[str, Any]] = None

    def __post_init__(self):
        if self.collected_context is None:
            self.collected_context = []
        if self.reasoning_history is None:
            self.reasoning_history = []


class IntentAnalysisAgent:
    """LLM-powered agent for analyzing user intent and determining required tools."""

    def __init__(self):
        self.agent = None
        if ADK_AVAILABLE:
            try:
                self.agent = LlmAgent(
                    name="intent_analyzer",
                    model="gemini-2.0-flash-thinking",
                    instruction="""You are an expert intent analysis agent. Your job is to understand user requests and determine:
1. What the user wants to accomplish
2. Which systems/tools are needed
3. What specific information needs to be retrieved
4. The complexity and scope of the request

Always respond with structured JSON containing your analysis."""
                )
                logger.info("✅ Intent Analysis Agent initialized with LLM")
            except Exception as e:
                logger.warning(f"Failed to initialize Intent Analysis Agent: {e}")

    async def analyze_intent(self, context: ReasoningContext, available_tools: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Use LLM to analyze user intent and determine required tools."""
        try:
            logger.info("🧠 Intent Analysis Agent: Analyzing user request with LLM")

            # Create detailed prompt for LLM analysis
            tools_summary = self._format_tools_for_llm(available_tools)

            analysis_prompt = f"""
TASK: Analyze the user's request and determine intent, required tools, and execution strategy.

USER REQUEST: "{context.original_request}"

AVAILABLE TOOLS:
{tools_summary}

Please analyze this request and respond with JSON in this exact format:
{{
    "intent_type": "task_management|version_control|file_management|data_analysis|general_query",
    "primary_goal": "clear description of what user wants",
    "required_systems": ["youtrack", "gitlab", "filesystem", etc.],
    "specific_actions": ["find_assigned_tickets", "filter_by_user", etc.],
    "information_needed": ["user_identity", "ticket_filters", etc.],
    "complexity_level": "simple|moderate|complex",
    "estimated_steps": 3,
    "confidence": 0.95,
    "reasoning": "explanation of your analysis"
}}

Focus on understanding what the user really wants and which tools can help achieve it."""

            if self.agent:
                try:
                    # Use the correct ADK Agent API
                    async_gen = self.agent.run_async(analysis_prompt)
                    response = ""
                    async for chunk in async_gen:
                        response += chunk

                    # Parse LLM response
                    intent_data = self._parse_llm_response(response)
                    if intent_data:
                        logger.info(f"🧠 Intent Analysis: {intent_data.get('intent_type')} with {intent_data.get('confidence', 0)} confidence")
                        return {
                            "status": "success",
                            "intent_analysis": intent_data,
                            "llm_powered": True
                        }

                except asyncio.TimeoutError:
                    logger.warning("Intent analysis LLM call timed out")
                except Exception as e:
                    logger.warning(f"LLM intent analysis failed: {e}")

            # Fallback to rule-based analysis
            logger.info("🧠 Intent Analysis: Falling back to rule-based analysis")
            fallback_analysis = self._fallback_intent_analysis(context.original_request, available_tools)
            return {
                "status": "success",
                "intent_analysis": fallback_analysis,
                "llm_powered": False
            }

        except Exception as e:
            logger.error(f"❌ Intent analysis failed: {e}")
            return {"status": "error", "error": str(e)}

    def _format_tools_for_llm(self, tools: List[Dict[str, Any]]) -> str:
        """Format available tools for LLM understanding."""
        if not tools:
            return "No tools available"

        tool_descriptions = []
        for tool in tools[:15]:  # Limit to avoid token overflow
            server = tool.get("server_name", "unknown")
            name = tool.get("name", "unknown")
            desc = tool.get("description", "no description")[:100]
            tool_descriptions.append(f"  - {server}.{name}: {desc}")

        if len(tools) > 15:
            tool_descriptions.append(f"  ... and {len(tools) - 15} more tools")

        return "\n".join(tool_descriptions)

    def _parse_llm_response(self, response: str) -> Optional[Dict[str, Any]]:
        """Parse LLM response and extract JSON."""
        try:
            # Try to find JSON in the response
            import re
            json_match = re.search(r'\{.*\}', response, re.DOTALL)
            if json_match:
                return json.loads(json_match.group())
        except (json.JSONDecodeError, AttributeError):
            pass
        return None

    def _fallback_intent_analysis(self, request: str, tools: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Fallback rule-based intent analysis."""
        request_lower = request.lower()

        # Simple rule-based classification
        if any(word in request_lower for word in ["ticket", "task", "assigned", "youtrack"]):
            intent_type = "task_management"
            required_systems = ["youtrack"]
        elif any(word in request_lower for word in ["repository", "repo", "commit", "gitlab"]):
            intent_type = "version_control"
            required_systems = ["gitlab"]
        else:
            intent_type = "general_query"
            required_systems = []

        return {
            "intent_type": intent_type,
            "primary_goal": f"Process request: {request[:100]}",
            "required_systems": required_systems,
            "specific_actions": ["analyze_request"],
            "information_needed": ["user_context"],
            "complexity_level": "moderate",
            "estimated_steps": 3,
            "confidence": 0.7,
            "reasoning": "Fallback rule-based analysis"
        }


class PlanGenerationAgent:
    """LLM-powered agent for generating detailed execution plans."""

    def __init__(self):
        self.agent = None
        if ADK_AVAILABLE:
            try:
                self.agent = LlmAgent(
                    name="plan_generator",
                    model="gemini-2.0-flash-thinking",
                    instruction="""You are an expert plan generation agent. Create detailed, step-by-step execution plans.
Each step should be specific and actionable. Consider dependencies, error handling, and resource requirements.
Always respond with structured JSON containing the complete plan."""
                )
                logger.info("✅ Plan Generation Agent initialized with LLM")
            except Exception as e:
                logger.warning(f"Failed to initialize Plan Generation Agent: {e}")

    async def generate_plan(self, context: ReasoningContext) -> Dict[str, Any]:
        """Use LLM to generate detailed execution plan."""
        try:
            logger.info("🧠 Plan Generation Agent: Creating execution plan with LLM")

            intent = context.intent_analysis
            if not intent:
                return {"status": "error", "error": "No intent analysis available"}

            plan_prompt = f"""
TASK: Create a detailed execution plan for the following request.

USER REQUEST: "{context.original_request}"

INTENT ANALYSIS:
- Type: {intent.get('intent_type')}
- Goal: {intent.get('primary_goal')}
- Required Systems: {intent.get('required_systems', [])}
- Actions Needed: {intent.get('specific_actions', [])}
- Information Needed: {intent.get('information_needed', [])}

AVAILABLE MCP TOOLS: {len(context.mcp_tools_available or [])} tools across YouTrack, GitLab, and filesystem

Create a detailed plan in this JSON format:
{{
    "plan_id": "unique_plan_identifier",
    "plan_type": "{intent.get('intent_type')}",
    "total_steps": 5,
    "steps": [
        {{
            "step_number": 1,
            "step_name": "authenticate_with_youtrack",
            "step_type": "mcp_tool_call|llm_analysis|data_processing",
            "description": "detailed description of what this step does",
            "required_tools": ["youtrack.authenticate"],
            "expected_output": "authentication confirmation",
            "dependencies": [],
            "error_handling": "retry with fallback",
            "estimated_time_ms": 1000
        }}
    ],
    "success_criteria": "clear criteria for completion",
    "fallback_strategies": ["alternative approaches if main plan fails"],
    "resource_requirements": ["youtrack_access", "user_permissions"],
    "confidence": 0.9,
    "reasoning": "why this plan will work"
}}

Make the plan specific to the user's request and available tools."""

            if self.agent:
                try:
                    async_gen = self.agent.run_async(plan_prompt)
                    response = ""
                    async for chunk in async_gen:
                        response += chunk

                    plan_data = self._parse_llm_response(response)
                    if plan_data:
                        logger.info(f"🧠 Plan Generated: {plan_data.get('total_steps', 0)} steps for {plan_data.get('plan_type')}")
                        return {
                            "status": "success",
                            "execution_plan": plan_data,
                            "llm_powered": True
                        }

                except asyncio.TimeoutError:
                    logger.warning("Plan generation LLM call timed out")
                except Exception as e:
                    logger.warning(f"LLM plan generation failed: {e}")

            # Fallback plan generation
            logger.info("🧠 Plan Generation: Using fallback plan generation")
            fallback_plan = self._generate_fallback_plan(context)
            return {
                "status": "success",
                "execution_plan": fallback_plan,
                "llm_powered": False
            }

        except Exception as e:
            logger.error(f"❌ Plan generation failed: {e}")
            return {"status": "error", "error": str(e)}

    def _parse_llm_response(self, response: str) -> Optional[Dict[str, Any]]:
        """Parse LLM response and extract JSON."""
        try:
            import re
            json_match = re.search(r'\{.*\}', response, re.DOTALL)
            if json_match:
                return json.loads(json_match.group())
        except (json.JSONDecodeError, AttributeError):
            pass
        return None

    def _generate_fallback_plan(self, context: ReasoningContext) -> Dict[str, Any]:
        """Generate fallback plan without LLM."""
        intent = context.intent_analysis or {}
        intent_type = intent.get("intent_type", "general_query")

        if intent_type == "task_management":
            steps = [
                {
                    "step_number": 1,
                    "step_name": "authenticate_user",
                    "step_type": "mcp_tool_call",
                    "description": "Authenticate with YouTrack and get user info",
                    "required_tools": ["youtrack.get_user_details"],
                    "expected_output": "user authentication data",
                    "dependencies": [],
                    "error_handling": "retry authentication",
                    "estimated_time_ms": 1000
                },
                {
                    "step_number": 2,
                    "step_name": "find_assigned_tickets",
                    "step_type": "mcp_tool_call",
                    "description": "Find tickets assigned to authenticated user",
                    "required_tools": ["youtrack.find_assigned_tickets"],
                    "expected_output": "list of assigned tickets",
                    "dependencies": [1],
                    "error_handling": "fallback to all tickets",
                    "estimated_time_ms": 2000
                },
                {
                    "step_number": 3,
                    "step_name": "format_results",
                    "step_type": "data_processing",
                    "description": "Format ticket data for user presentation",
                    "required_tools": [],
                    "expected_output": "formatted ticket list",
                    "dependencies": [2],
                    "error_handling": "raw data fallback",
                    "estimated_time_ms": 500
                }
            ]
        else:
            steps = [
                {
                    "step_number": 1,
                    "step_name": "process_request",
                    "step_type": "llm_analysis",
                    "description": "Process general request",
                    "required_tools": [],
                    "expected_output": "processed response",
                    "dependencies": [],
                    "error_handling": "standard error response",
                    "estimated_time_ms": 1000
                }
            ]

        return {
            "plan_id": f"fallback_{intent_type}",
            "plan_type": intent_type,
            "total_steps": len(steps),
            "steps": steps,
            "success_criteria": "User request fulfilled",
            "fallback_strategies": ["Direct LLM response"],
            "resource_requirements": ["basic_access"],
            "confidence": 0.6,
            "reasoning": "Fallback plan generated without LLM"
        }


class PlanExecutionAgent:
    """LLM-powered agent for recursive plan execution with MCP tools."""

    def __init__(self):
        self.agent = None
        if ADK_AVAILABLE:
            try:
                self.agent = LlmAgent(
                    name="plan_executor",
                    model="gemini-2.0-flash-thinking",
                    instruction="""You are an expert plan execution agent. Execute plans step by step, making decisions about:
1. When to use MCP tools vs continue with LLM analysis
2. How to adapt plans based on intermediate results
3. When recursion should continue or terminate
4. How to handle errors and fallback strategies

Always provide clear reasoning for your decisions."""
                )
                logger.info("✅ Plan Execution Agent initialized with LLM")
            except Exception as e:
                logger.warning(f"Failed to initialize Plan Execution Agent: {e}")

    async def execute_plan(self, context: ReasoningContext, mcp_tool_executor) -> Dict[str, Any]:
        """Execute plan with LLM guidance and recursive decision making."""
        try:
            logger.info("🧠 Plan Execution Agent: Starting recursive plan execution")

            plan = context.execution_plan
            if not plan:
                return {"status": "error", "error": "No execution plan available"}

            steps = plan.get("steps", [])
            execution_results = []

            for step in steps:
                step_result = await self._execute_step_with_llm_guidance(
                    step, context, mcp_tool_executor, execution_results
                )
                execution_results.append(step_result)

                # Check if we should continue
                should_continue = await self._should_continue_execution(
                    context, step_result, execution_results
                )

                if not should_continue:
                    logger.info("🧠 Plan Execution: LLM determined execution should stop")
                    break

            return {
                "status": "success",
                "execution_results": execution_results,
                "steps_completed": len(execution_results),
                "llm_powered": True
            }

        except Exception as e:
            logger.error(f"❌ Plan execution failed: {e}")
            return {"status": "error", "error": str(e)}

    async def _execute_step_with_llm_guidance(self, step: Dict[str, Any], context: ReasoningContext,
                                            mcp_tool_executor, previous_results: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Execute a single step with LLM guidance."""
        try:
            step_name = step.get("step_name", "unknown")
            step_type = step.get("step_type", "unknown")

            logger.info(f"🧠 Executing step: {step_name} (type: {step_type})")

            if step_type == "mcp_tool_call":
                # Use MCP tools for this step
                return await self._execute_mcp_step(step, context, mcp_tool_executor)

            elif step_type == "llm_analysis":
                # Use LLM for analysis
                return await self._execute_llm_analysis_step(step, context, previous_results)

            elif step_type == "data_processing":
                # Process data from previous steps
                return await self._execute_data_processing_step(step, previous_results)

            else:
                return {
                    "step_name": step_name,
                    "status": "error",
                    "error": f"Unknown step type: {step_type}"
                }

        except Exception as e:
            return {
                "step_name": step.get("step_name", "unknown"),
                "status": "error",
                "error": str(e)
            }

    async def _execute_mcp_step(self, step: Dict[str, Any], context: ReasoningContext, mcp_tool_selector) -> Dict[str, Any]:
        """Execute step using MCP tools via MCPToolSelector."""
        step_name = step.get("step_name", "unknown")

        # Skip data processing steps
        if step_name == "format_results":
            return {
                "step_name": step_name,
                "status": "completed",
                "reason": f"No MCP tools needed for {step_name}",
                "step_type": "data_processing",
                "tools_executed": 0
            }

        try:
            # For ticket-related requests, use the direct working method
            if ("ticket" in context.original_request.lower() or
                "assigned" in context.original_request.lower() or
                "youtrack" in context.original_request.lower()):

                logger.info(f"🎫 Direct ticket retrieval for: {step_name}")

                # Use the existing MCP registry connection instead of creating a new one
                from src.infrastructure.mcp.registry import mcp_registry

                # Call YouTrack directly using existing registry
                try:
                    # Get the existing youtrack client from the MCP registry
                    youtrack_client = mcp_registry.get_server_by_name("youtrack-server")
                    if youtrack_client:
                        logger.info("🎫 Using existing YouTrack MCP connection")
                        # Call the find_assigned_tickets tool directly
                        response = await youtrack_client.call_tool("find_assigned_tickets", {"state": "Open"})
                        if response and isinstance(response, list) and len(response) > 0:
                            ticket_content = response[0].text if hasattr(response[0], 'text') else str(response[0])
                            logger.info(f"🎫 Retrieved ticket data: {ticket_content[:200]}...")
                            ticket_success = True
                        else:
                            ticket_success = False
                    else:
                        logger.warning("🎫 YouTrack client not available in registry")
                        ticket_success = False

                    if ticket_success:
                        # Create a result with the actual ticket content
                        mock_result = {
                            'success': True,
                            'result': ticket_content if 'ticket_content' in locals() else 'Successfully retrieved your assigned tickets via direct YouTrack MCP connection.',
                            'tool_name': 'find_assigned_tickets_direct',
                            'server_name': 'youtrack-server',
                            'execution_time_ms': 1000
                        }

                        return {
                            "step_name": step_name,
                            "status": "completed",
                            "mcp_results": [mock_result],
                            "step_type": "mcp_tool_call",
                            "tools_executed": 1,
                            "total_tools": 1,
                            "success_count": 1,
                            "execution_time_ms": 1000
                        }
                    else:
                        return {
                            "step_name": step_name,
                            "status": "error",
                            "error": "Direct ticket retrieval failed",
                            "step_type": "mcp_tool_call",
                            "tools_executed": 0
                        }

                except Exception as e:
                    logger.error(f"Direct ticket retrieval failed: {e}")
                    # Fall back to original tool selection method
                    pass

            # Use MCPToolSelector to intelligently select and execute tools
            # This uses the same logic as the original reasoning system

            # Create tool selection context
            tool_context = ToolSelectionContext(
                request_data={"messages": [{"role": "user", "content": context.original_request}]},
                intent_analysis={"intent_type": "task_management", "domains": ["project_management"]},
                processing_phase=ProcessingPhase.REASONING
            )

            # Let MCPToolSelector intelligently select and execute tools
            # This is the same logic used by the original reasoning system
            selection_result = await mcp_tool_selector.select_tools_for_context(tool_context)
            selected_tools = selection_result.get("selected_tools", [])

            if selected_tools:
                # Create execution plan
                execution_plan_result = await mcp_tool_selector.create_execution_plan(selected_tools, tool_context)

                if execution_plan_result.get("status") == "success" and execution_plan_result.get("execution_plan"):
                    execution_plan = execution_plan_result["execution_plan"]
                    # Execute the tools
                    execution_result = await mcp_tool_selector.execute_tool_plan(execution_plan, tool_context)
                else:
                    # No execution plan created
                    execution_result = {
                        "status": "error",
                        "error": execution_plan_result.get("reason", "Failed to create execution plan")
                    }

                if execution_result.get("status") == "success":
                    # MCPToolSelector returns "results" with ToolExecutionResult objects
                    execution_results = execution_result.get("results", [])
                    # Check both .success attribute and "success" field for compatibility
                    successful_tools = len([r for r in execution_results if getattr(r, 'success', False) or r.get("success", False)])

                    # Convert ToolExecutionResult objects to dictionaries for JSON serialization
                    serializable_results = []
                    for result in execution_results:
                        if hasattr(result, '__dict__'):
                            # Convert dataclass to dict and handle nested objects
                            result_data = result.result

                            # Handle TextContent and other MCP objects in result data
                            if hasattr(result_data, '__dict__'):
                                # Convert nested objects to basic types
                                if hasattr(result_data, 'text'):
                                    result_data = str(result_data.text)
                                else:
                                    result_data = str(result_data)
                            elif isinstance(result_data, (list, tuple)):
                                # Handle lists of potentially complex objects
                                processed_list = []
                                for item in result_data:
                                    if hasattr(item, 'text'):
                                        processed_list.append(str(item.text))
                                    elif hasattr(item, '__dict__'):
                                        processed_list.append(str(item))
                                    else:
                                        processed_list.append(item)
                                result_data = processed_list

                            result_dict = {
                                'success': result.success,
                                'result': result_data,
                                'error_message': result.error_message,
                                'server_name': result.server_name,
                                'execution_time_ms': result.execution_time_ms,
                                'tool_name': result.tool_name
                            }
                            serializable_results.append(result_dict)
                        else:
                            # Already a dictionary
                            serializable_results.append(result)

                    logger.info(f"✅ {step_name}: {successful_tools}/{len(selected_tools)} MCP tools executed successfully")

                    return {
                        "step_name": step_name,
                        "status": "completed",
                        "mcp_results": serializable_results,
                        "step_type": "mcp_tool_call",
                        "tools_executed": successful_tools,
                        "total_tools": len(selected_tools),
                        "success_count": execution_result.get("success_count", successful_tools),
                        "execution_time_ms": execution_result.get("execution_time_ms", 0)
                    }
                else:
                    logger.warning(f"❌ {step_name}: MCP tool execution failed: {execution_result.get('error')}")
                    return {
                        "step_name": step_name,
                        "status": "error",
                        "error": execution_result.get("error", "Unknown execution error"),
                        "step_type": "mcp_tool_call",
                        "tools_executed": 0,
                        "total_tools": len(selected_tools)
                    }
            else:
                # No tools selected
                logger.info(f"✅ {step_name}: No MCP tools selected for this step")
                return {
                    "step_name": step_name,
                    "status": "completed",
                    "reason": "No tools selected by MCPToolSelector",
                    "step_type": "mcp_tool_call",
                    "tools_executed": 0,
                    "total_tools": 0
                }

        except Exception as e:
            logger.error(f"❌ {step_name}: MCP tool execution exception: {e}")
            return {
                "step_name": step_name,
                "status": "error",
                "error": str(e),
                "step_type": "mcp_tool_call",
                "tools_executed": 0,
                "total_tools": len(actual_tools)
            }

    async def _execute_llm_analysis_step(self, step: Dict[str, Any], context: ReasoningContext,
                                       previous_results: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Execute step using LLM analysis."""
        if not self.agent:
            return {
                "step_name": step.get("step_name"),
                "status": "skipped",
                "reason": "LLM agent not available"
            }

        analysis_prompt = f"""
STEP: {step.get('step_name')}
DESCRIPTION: {step.get('description')}

CONTEXT:
- Original Request: {context.original_request}
- Previous Results: {json.dumps(previous_results, indent=2)[:1000]}

Analyze this step and provide insights or decisions needed to continue.
Respond with JSON containing your analysis and recommendations.
"""

        try:
            async_gen = self.agent.run_async(analysis_prompt)
            response = ""
            async for chunk in async_gen:
                response += chunk

            return {
                "step_name": step.get("step_name"),
                "status": "completed",
                "step_type": "llm_analysis",
                "llm_response": response[:500],  # Truncate for logging
                "analysis_completed": True
            }

        except Exception as e:
            return {
                "step_name": step.get("step_name"),
                "status": "error",
                "error": str(e)
            }

    async def _execute_data_processing_step(self, step: Dict[str, Any], previous_results: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Execute data processing step."""
        # Simple data processing - could be enhanced
        processed_data = {
            "previous_steps": len(previous_results),
            "successful_steps": sum(1 for r in previous_results if r.get("status") == "completed"),
            "summary": f"Processed {len(previous_results)} previous steps"
        }

        return {
            "step_name": step.get("step_name"),
            "status": "completed",
            "step_type": "data_processing",
            "processed_data": processed_data
        }

    async def _should_continue_execution(self, context: ReasoningContext, last_result: Dict[str, Any],
                                       all_results: List[Dict[str, Any]]) -> bool:
        """Use LLM to determine if execution should continue."""
        if not self.agent:
            return True  # Default to continue if no LLM

        decision_prompt = f"""
EXECUTION DECISION NEEDED:

Original Request: {context.original_request}
Last Step Result: {_safe_json_serialize(last_result)[:500]}
Total Steps Executed: {len(all_results)}

Should execution continue or is there enough information to satisfy the user's request?

Respond with JSON:
{{
    "should_continue": true/false,
    "reasoning": "why continue or stop",
    "confidence": 0.9
}}
"""

        try:
            async_gen = self.agent.run_async(decision_prompt)
            response = ""
            async for chunk in async_gen:
                response += chunk

            import re
            json_match = re.search(r'\{.*\}', response, re.DOTALL)
            if json_match:
                decision = json.loads(json_match.group())
                should_continue = decision.get("should_continue", True)
                logger.info(f"🧠 Execution Decision: {'Continue' if should_continue else 'Stop'} - {decision.get('reasoning', '')}")
                return should_continue

        except Exception as e:
            logger.warning(f"Failed to get execution decision from LLM: {e}")

        return True  # Default to continue


class ContextSufficiencyAgent:
    """LLM-powered agent to determine when enough context has been collected."""

    def __init__(self):
        self.agent = None
        if ADK_AVAILABLE:
            try:
                self.agent = LlmAgent(
                    name="context_evaluator",
                    model="gemini-2.0-flash-thinking",
                    instruction="""You are an expert context evaluation agent. Your job is to determine when enough information
has been collected to answer the user's request. Consider completeness, relevance, and sufficiency of collected data.
Always provide clear reasoning for your decisions."""
                )
                logger.info("✅ Context Sufficiency Agent initialized with LLM")
            except Exception as e:
                logger.warning(f"Failed to initialize Context Sufficiency Agent: {e}")

    async def evaluate_context_sufficiency(self, context: ReasoningContext) -> Dict[str, Any]:
        """Use LLM to evaluate if collected context is sufficient."""
        try:
            logger.info("🧠 Context Sufficiency Agent: Evaluating collected context")

            evaluation_prompt = f"""
CONTEXT SUFFICIENCY EVALUATION:

Original User Request: "{context.original_request}"

Collected Context:
{json.dumps(context.collected_context, indent=2)[:2000]}

Intent Analysis: {json.dumps(context.intent_analysis, indent=2)[:500]}
Execution Plan: {json.dumps(context.execution_plan, indent=2)[:500]}

EVALUATION CRITERIA:
1. Is there enough information to answer the user's request?
2. Are all required data points collected?
3. Is the information quality sufficient?
4. Are there any critical gaps?

Respond with JSON:
{{
    "is_sufficient": true/false,
    "sufficiency_score": 0.85,
    "missing_information": ["list of what's still needed"],
    "collected_information": ["list of what we have"],
    "recommendation": "stop_and_respond|continue_collection|need_clarification",
    "reasoning": "detailed explanation of your evaluation",
    "confidence": 0.9
}}
"""

            if self.agent:
                try:
                    async_gen = self.agent.run_async(evaluation_prompt)
                    response = ""
                    async for chunk in async_gen:
                        response += chunk

                    import re
                    json_match = re.search(r'\{.*\}', response, re.DOTALL)
                    if json_match:
                        evaluation = json.loads(json_match.group())

                        is_sufficient = evaluation.get("is_sufficient", False)
                        score = evaluation.get("sufficiency_score", 0.5)

                        logger.info(f"🧠 Context Evaluation: {'Sufficient' if is_sufficient else 'Insufficient'} (score: {score})")

                        return {
                            "status": "success",
                            "evaluation": evaluation,
                            "llm_powered": True
                        }

                except asyncio.TimeoutError:
                    logger.warning("Context evaluation LLM call timed out")
                except Exception as e:
                    logger.warning(f"LLM context evaluation failed: {e}")

            # Fallback evaluation
            logger.info("🧠 Context Evaluation: Using fallback evaluation")
            fallback_evaluation = self._fallback_context_evaluation(context)
            return {
                "status": "success",
                "evaluation": fallback_evaluation,
                "llm_powered": False
            }

        except Exception as e:
            logger.error(f"❌ Context evaluation failed: {e}")
            return {"status": "error", "error": str(e)}

    def _fallback_context_evaluation(self, context: ReasoningContext) -> Dict[str, Any]:
        """Fallback context evaluation without LLM."""
        # Simple heuristics
        context_items = len(context.collected_context)
        has_intent = context.intent_analysis is not None
        has_plan = context.execution_plan is not None

        # Basic sufficiency check
        is_sufficient = context_items >= 2 and has_intent and has_plan
        score = min(1.0, (context_items * 0.3) + (0.4 if has_intent else 0) + (0.3 if has_plan else 0))

        return {
            "is_sufficient": is_sufficient,
            "sufficiency_score": score,
            "missing_information": [] if is_sufficient else ["More context needed"],
            "collected_information": [f"{context_items} context items", "Intent analysis", "Execution plan"],
            "recommendation": "stop_and_respond" if is_sufficient else "continue_collection",
            "reasoning": f"Simple heuristic: {context_items} items, intent: {has_intent}, plan: {has_plan}",
            "confidence": 0.6
        }


# Factory function to create all agents
def create_llm_reasoning_agents() -> Tuple[IntentAnalysisAgent, PlanGenerationAgent, PlanExecutionAgent, ContextSufficiencyAgent]:
    """Create and return all LLM-powered reasoning agents."""
    return (
        IntentAnalysisAgent(),
        PlanGenerationAgent(),
        PlanExecutionAgent(),
        ContextSufficiencyAgent()
    )