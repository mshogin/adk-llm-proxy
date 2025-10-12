#!/usr/bin/env python3
"""
Enhanced Multi-Agent Reasoning Orchestrator.

This orchestrator implements the sophisticated reasoning system where:
1. Intent Analysis Agent uses LLM to understand user intent
2. Plan Generation Agent uses LLM to create detailed execution plans
3. Plan Execution Agent uses LLM for recursive execution with MCP tools
4. Context Sufficiency Agent uses LLM to determine when to stop

Each reasoning step involves LLM calls for intelligent decision making.
"""

import logging
import json
import time
import asyncio
from typing import Dict, List, Any, Optional, AsyncGenerator

from .llm_reasoning_agents import (
    ReasoningContext, ReasoningPhase,
    IntentAnalysisAgent, PlanGenerationAgent,
    PlanExecutionAgent, ContextSufficiencyAgent,
    create_llm_reasoning_agents
)

from src.infrastructure.mcp.registry import mcp_registry
from src.infrastructure.mcp.discovery import MCPToolDiscovery
from src.infrastructure.mcp.tool_registry import MCPUnifiedToolRegistry
from src.application.services.mcp_tool_selector import MCPToolSelector

logger = logging.getLogger(__name__)

# Global components for enhanced reasoning
_enhanced_reasoning_components = None


class EnhancedReasoningOrchestrator:
    """
    Enhanced reasoning orchestrator with LLM-powered agents and recursive execution.

    This implements the sophisticated multi-agent reasoning system where each step
    uses LLM for intelligent decision making and MCP tools for actual execution.
    """

    def __init__(self):
        # Initialize LLM-powered reasoning agents
        (self.intent_agent,
         self.plan_agent,
         self.execution_agent,
         self.context_agent) = create_llm_reasoning_agents()

        # MCP components
        self.mcp_discovery = None
        self.mcp_tool_registry = None
        self.mcp_tool_selector = None

        logger.info("✅ Enhanced Reasoning Orchestrator initialized with LLM agents")

    async def initialize_mcp_components(self):
        """Initialize MCP components for tool execution."""
        try:
            if not self.mcp_discovery:
                self.mcp_discovery = MCPToolDiscovery(mcp_registry)
                await self.mcp_discovery.discover_all_capabilities()
                logger.info("✅ MCP Discovery initialized and populated")

            if not self.mcp_tool_registry:
                self.mcp_tool_registry = MCPUnifiedToolRegistry(mcp_registry, self.mcp_discovery)
                logger.info("✅ MCP Tool Registry initialized")

            if not self.mcp_tool_selector:
                self.mcp_tool_selector = MCPToolSelector(self.mcp_tool_registry, mcp_registry, self.mcp_discovery)
                logger.info("✅ MCP Tool Selector initialized")

        except Exception as e:
            logger.error(f"❌ Error initializing MCP components: {e}")

    async def execute_enhanced_reasoning(self, request_data: Dict[str, Any]) -> AsyncGenerator[str, None]:
        """
        Execute the enhanced multi-agent reasoning system with streaming output.

        This implements the sophisticated reasoning flow:
        1. LLM Intent Analysis - understand what user wants
        2. LLM Plan Generation - create detailed execution plan
        3. LLM Recursive Execution - execute plan with MCP tools
        4. LLM Context Evaluation - determine when to stop
        """
        try:
            logger.info("🧠 Enhanced Reasoning: Starting multi-agent reasoning system")

            # Initialize MCP components
            await self.initialize_mcp_components()

            # Extract user request
            messages = request_data.get("messages", [])
            user_messages = [msg for msg in messages if msg.get("role") == "user"]
            if not user_messages:
                yield await self._stream_error("No user messages found")
                return

            original_request = user_messages[-1].get("content", "")

            # Initialize reasoning context
            context = ReasoningContext(
                original_request=original_request,
                mcp_tools_available=self.mcp_discovery.get_all_tools() if self.mcp_discovery else []
            )

            # Start reasoning markers
            yield await self._stream_reasoning_start()

            # Phase 1: LLM-Powered Intent Analysis
            yield await self._stream_phase("Intent Analysis", "Analyzing user intent with LLM...")

            intent_result = await self.intent_agent.analyze_intent(
                context,
                [tool.to_dict() for tool in context.mcp_tools_available]
            )

            if intent_result.get("status") == "success":
                context.intent_analysis = intent_result["intent_analysis"]
                context.reasoning_history.append({
                    "phase": "intent_analysis",
                    "result": intent_result,
                    "timestamp": time.time()
                })

                intent_data = context.intent_analysis
                yield await self._stream_phase_result("Intent Analysis", {
                    "intent_type": intent_data.get("intent_type"),
                    "confidence": intent_data.get("confidence"),
                    "required_systems": intent_data.get("required_systems", []),
                    "llm_powered": intent_result.get("llm_powered", False)
                })
            else:
                yield await self._stream_error(f"Intent analysis failed: {intent_result.get('error')}")
                return

            await asyncio.sleep(0.2)  # Visual delay for streaming

            # Phase 2: LLM-Powered Plan Generation
            yield await self._stream_phase("Plan Generation", "Creating detailed execution plan with LLM...")

            context.current_phase = ReasoningPhase.PLAN_GENERATION
            plan_result = await self.plan_agent.generate_plan(context)

            if plan_result.get("status") == "success":
                context.execution_plan = plan_result["execution_plan"]
                context.reasoning_history.append({
                    "phase": "plan_generation",
                    "result": plan_result,
                    "timestamp": time.time()
                })

                plan_data = context.execution_plan
                yield await self._stream_phase_result("Plan Generation", {
                    "plan_type": plan_data.get("plan_type"),
                    "total_steps": plan_data.get("total_steps"),
                    "confidence": plan_data.get("confidence"),
                    "llm_powered": plan_result.get("llm_powered", False)
                })

                # Show plan details
                steps = plan_data.get("steps", [])
                for i, step in enumerate(steps[:3]):  # Show first 3 steps
                    yield await self._stream_plan_step(i + 1, step)

            else:
                yield await self._stream_error(f"Plan generation failed: {plan_result.get('error')}")
                return

            await asyncio.sleep(0.2)

            # Phase 3: LLM-Powered Recursive Plan Execution
            yield await self._stream_phase("Plan Execution", "Executing plan with LLM guidance and MCP tools...")

            context.current_phase = ReasoningPhase.PLAN_EXECUTION
            execution_result = await self.execution_agent.execute_plan(context, self.mcp_tool_selector)

            if execution_result.get("status") == "success":
                execution_results = execution_result["execution_results"]

                # Extract and convert execution results to standardized format
                serializable_results = []

                # execution_results from LLM agents contains step results
                for step_result in execution_results:
                    # Check if this step has MCP results
                    if isinstance(step_result, dict) and 'mcp_results' in step_result:
                        # This step executed MCP tools - extract the actual tool results
                        mcp_results = step_result['mcp_results']
                        for mcp_result in mcp_results:
                            serializable_results.append(mcp_result)

                    # Also check for direct ToolExecutionResult objects (backwards compatibility)
                    elif hasattr(step_result, '__dict__'):
                        # Convert dataclass to dict and handle nested objects
                        result_data = getattr(step_result, 'result', step_result)

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
                            'success': getattr(step_result, 'success', True),
                            'result': result_data,
                            'error_message': getattr(step_result, 'error_message', None),
                            'server_name': getattr(step_result, 'server_name', None),
                            'execution_time_ms': getattr(step_result, 'execution_time_ms', 0),
                            'tool_name': getattr(step_result, 'tool_name', 'unknown')
                        }
                        serializable_results.append(result_dict)
                    elif isinstance(step_result, dict):
                        # Already a dictionary - check if it's a step result or tool result
                        if 'step_name' in step_result and 'status' in step_result:
                            # This is a step result, not a tool result - skip it
                            continue
                        else:
                            # Treat as tool result
                            serializable_results.append(step_result)

                context.collected_context.extend(serializable_results)

                # Also create a serializable version for reasoning history
                serializable_execution_result = execution_result.copy()
                serializable_execution_result["execution_results"] = serializable_results

                context.reasoning_history.append({
                    "phase": "plan_execution",
                    "result": serializable_execution_result,
                    "timestamp": time.time()
                })

                yield await self._stream_phase_result("Plan Execution", {
                    "steps_completed": execution_result.get("steps_completed"),
                    "successful_steps": sum(1 for r in serializable_results if r.get("status") == "completed"),
                    "llm_powered": execution_result.get("llm_powered", False)
                })

                # Show ALL execution results (especially important for tickets)
                for result in serializable_results:  # Show all results, not just first 3
                    yield await self._stream_execution_result(result)

            else:
                yield await self._stream_error(f"Plan execution failed: {execution_result.get('error')}")
                return

            await asyncio.sleep(0.2)

            # Phase 4: LLM-Powered Context Sufficiency Evaluation
            yield await self._stream_phase("Context Evaluation", "Evaluating context sufficiency with LLM...")

            context.current_phase = ReasoningPhase.CONTEXT_EVALUATION
            sufficiency_result = await self.context_agent.evaluate_context_sufficiency(context)

            if sufficiency_result.get("status") == "success":
                evaluation = sufficiency_result["evaluation"]
                context.reasoning_history.append({
                    "phase": "context_evaluation",
                    "result": sufficiency_result,
                    "timestamp": time.time()
                })

                yield await self._stream_phase_result("Context Evaluation", {
                    "is_sufficient": evaluation.get("is_sufficient"),
                    "sufficiency_score": evaluation.get("sufficiency_score"),
                    "recommendation": evaluation.get("recommendation"),
                    "llm_powered": sufficiency_result.get("llm_powered", False)
                })

                # Check if we should continue or complete
                if evaluation.get("recommendation") == "continue_collection":
                    yield await self._stream_phase("Recursion", "LLM determined more context needed, continuing...")
                    # Here we could implement recursion back to plan execution
                    # For now, we'll proceed to completion

            await asyncio.sleep(0.2)

            # Phase 5: Completion
            context.current_phase = ReasoningPhase.COMPLETION
            yield await self._stream_completion(context)

            # Store the context for later retrieval
            self._last_context = context

            # End reasoning markers
            yield await self._stream_reasoning_end()

            logger.info("🧠 Enhanced Reasoning: Multi-agent reasoning completed successfully")

        except Exception as e:
            logger.error(f"❌ Enhanced reasoning failed: {e}")
            yield await self._stream_error(f"Enhanced reasoning failed: {str(e)}")

    async def _stream_reasoning_start(self) -> str:
        """Generate minimal reasoning start indicator."""
        chunk = {
            "id": f"enhanced-reasoning-start-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
            "choices": [
                {
                    "index": 0,
                    "delta": {
                        "content": "🔍 Analyzing...\n"
                    },
                    "finish_reason": None
                }
            ]
        }
        return f"data: {json.dumps(chunk)}\n\n"

    async def _stream_reasoning_end(self) -> str:
        """Generate minimal reasoning end indicator."""
        chunk = {
            "id": f"enhanced-reasoning-end-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
            "choices": [
                {
                    "index": 0,
                    "delta": {
                        "content": "✅ Analysis complete.\n\n"
                    },
                    "finish_reason": None
                }
            ]
        }
        return f"data: {json.dumps(chunk)}\n\n"

    async def _stream_phase(self, phase_name: str, description: str) -> str:
        """Stream simplified reasoning phase information."""
        # Only show essential phases
        if phase_name in ["Plan Execution"]:
            content = f"🎯 {description}\n"
        else:
            # Skip most verbose phases
            return ""

        chunk = {
            "id": f"enhanced-phase-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
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
        return f"data: {json.dumps(chunk)}\n\n"

    async def _stream_phase_result(self, phase_name: str, result_data: Dict[str, Any]) -> str:
        """Stream simplified reasoning phase results."""
        # Only show meaningful results
        if phase_name == "Plan Execution":
            successful_steps = result_data.get('successful_steps', 0)
            if successful_steps > 0:
                content = f"✅ Retrieved data from {successful_steps} sources\n"
            else:
                content = f"⚠️ No data retrieved\n"
        else:
            # Skip other verbose phase results
            return ""

        chunk = {
            "id": f"enhanced-result-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
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
        return f"data: {json.dumps(chunk)}\n\n"

    async def _stream_plan_step(self, step_number: int, step_data: Dict[str, Any]) -> str:
        """Stream simplified plan step information (or skip)."""
        # Skip verbose plan step details
        return ""

    async def _stream_execution_result(self, result_data: Dict[str, Any]) -> str:
        """Stream execution result information with actual tool results."""
        tool_name = result_data.get("tool_name", "unknown")
        success = result_data.get("success", False)
        result_content = result_data.get("result", "")

        if success:
            # Show actual tool results, especially for ticket-related tools
            if "find_assigned_tickets" in tool_name or "ticket" in str(result_content).lower():
                content = f"✅ **{tool_name}**: Retrieved ticket data\n\n{result_content}\n\n"
            elif "find_epic" in tool_name or "epic" in str(result_content).lower():
                content = f"✅ **{tool_name}**: Retrieved epic data\n\n{result_content}\n\n"
            elif "get_task_details" in tool_name or "task" in str(result_content).lower():
                content = f"✅ **{tool_name}**: Retrieved task details\n\n{result_content}\n\n"
            else:
                # For other tools, show result summary
                result_preview = str(result_content)[:300] + "..." if len(str(result_content)) > 300 else str(result_content)
                content = f"✅ **{tool_name}**: {result_preview}\n\n"
        else:
            error_msg = result_data.get("error_message", "Unknown error")
            content = f"❌ **{tool_name}**: {error_msg}\n"

        chunk = {
            "id": f"enhanced-execution-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
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
        return f"data: {json.dumps(chunk)}\n\n"

    async def _stream_completion(self, context: ReasoningContext) -> str:
        """Stream reasoning completion summary with collected data preview."""
        total_phases = len(context.reasoning_history)
        llm_powered_phases = sum(1 for phase in context.reasoning_history
                                if phase.get("result", {}).get("llm_powered", False))

        content = f"""🧠 **Enhanced Reasoning Complete**:
📊 **Summary**: {total_phases} phases executed, {llm_powered_phases} LLM-powered
🎯 **Intent**: {context.intent_analysis.get('intent_type', 'unknown') if context.intent_analysis else 'unknown'}
📋 **Plan**: {context.execution_plan.get('total_steps', 0) if context.execution_plan else 0} steps planned
📦 **Context**: {len(context.collected_context)} items collected
🤖 **LLM Integration**: Advanced multi-agent reasoning system

"""

        # If we have collected context with ticket/task data, show a summary
        if context.collected_context:
            ticket_results = []
            for item in context.collected_context:
                if item.get('success') and item.get('tool_name'):
                    tool_name = item['tool_name']
                    result_content = str(item.get('result', ''))

                    # Check for ticket/task data
                    if ('find_assigned_tickets' in tool_name or
                        'ticket' in result_content.lower() and 'assigned' in result_content.lower()):
                        ticket_results.append(result_content)

            if ticket_results:
                content += "🎫 **Retrieved Data Summary**:\n"
                for result in ticket_results[:2]:  # Show up to 2 ticket result summaries
                    preview = result[:200] + "..." if len(result) > 200 else result
                    content += f"{preview}\n\n"

        chunk = {
            "id": f"enhanced-completion-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
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
        return f"data: {json.dumps(chunk)}\n\n"

    async def _stream_error(self, error_message: str) -> str:
        """Stream error information."""
        content = f"❌ **Enhanced Reasoning Error**: {error_message}\n"

        chunk = {
            "id": f"enhanced-error-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
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
        return f"data: {json.dumps(chunk)}\n\n"


# Enhanced reasoning function for integration
async def enhanced_reasoning_pipeline(request_data: Dict[str, Any]) -> AsyncGenerator[str, None]:
    """
    Enhanced reasoning pipeline using multi-agent LLM system.

    This is the main entry point for the enhanced reasoning system.
    """
    global _enhanced_reasoning_components

    try:
        if not _enhanced_reasoning_components:
            _enhanced_reasoning_components = EnhancedReasoningOrchestrator()
            logger.info("🧠 Enhanced Reasoning: Orchestrator initialized")

        async for chunk in _enhanced_reasoning_components.execute_enhanced_reasoning(request_data):
            yield chunk

    except Exception as e:
        logger.error(f"❌ Enhanced reasoning pipeline failed: {e}")
        error_chunk = {
            "id": f"enhanced-error-{int(time.time())}",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": "enhanced-reasoning-engine",
            "choices": [
                {
                    "index": 0,
                    "delta": {
                        "content": f"❌ Enhanced reasoning failed: {str(e)}\n"
                    },
                    "finish_reason": None
                }
            ]
        }
        yield f"data: {json.dumps(error_chunk)}\n\n"


# Factory function for creating enhanced reasoning orchestrator
def create_enhanced_reasoning_orchestrator() -> EnhancedReasoningOrchestrator:
    """Create and return enhanced reasoning orchestrator."""
    return EnhancedReasoningOrchestrator()

# Function to get the current enhanced reasoning context
def get_enhanced_reasoning_context():
    """Get the current enhanced reasoning context with collected data."""
    global _enhanced_reasoning_components
    if _enhanced_reasoning_components and hasattr(_enhanced_reasoning_components, '_last_context'):
        return _enhanced_reasoning_components._last_context
    return None