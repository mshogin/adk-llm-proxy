"""
MCP Tool Selector Service for Intelligent Tool Selection.

This service provides intelligent selection of MCP tools based on request analysis,
context requirements, and tool capabilities. It integrates with the existing
preprocessing, reasoning, and postprocessing pipeline.
"""
import json
import logging
import asyncio
from typing import Dict, List, Any, Optional, Set, Tuple
from dataclasses import dataclass
from enum import Enum
from datetime import datetime

from src.infrastructure.mcp.tool_registry import MCPUnifiedToolRegistry, ToolExecutionResult
from src.infrastructure.mcp.registry import MCPServerRegistry
from src.infrastructure.mcp.discovery import MCPToolDiscovery


class ProcessingPhase(Enum):
    """Processing phases where MCP tools can be utilized."""
    PREPROCESSING = "preprocessing"
    REASONING = "reasoning"
    POSTPROCESSING = "postprocessing"
    VALIDATION = "validation"


class ToolSelectionStrategy(Enum):
    """Strategies for selecting MCP tools."""
    CAPABILITY_MATCH = "capability_match"
    LLM_GUIDED = "llm_guided"
    HYBRID = "hybrid"
    RULE_BASED = "rule_based"


@dataclass
class ToolCapability:
    """Represents a tool's capabilities for matching."""
    name: str
    description: str
    input_schema: Dict[str, Any]
    domains: Set[str]
    complexity_level: str  # simple, moderate, complex
    processing_phases: Set[ProcessingPhase]
    keywords: Set[str]
    server_name: str
    confidence_score: float = 0.0


@dataclass
class ToolSelectionContext:
    """Context information for tool selection."""
    request_data: Dict[str, Any]
    intent_analysis: Optional[Dict[str, Any]] = None
    processing_phase: ProcessingPhase = ProcessingPhase.PREPROCESSING
    available_tools: List[ToolCapability] = None
    selected_tools: List[str] = None
    execution_results: List[ToolExecutionResult] = None
    metadata: Dict[str, Any] = None


@dataclass
class ToolExecutionPlan:
    """Plan for executing selected tools."""
    tools: List[str]
    execution_order: List[str]
    parallel_groups: List[List[str]]
    dependencies: Dict[str, List[str]]
    timeout_ms: int
    retry_count: int
    fallback_tools: Dict[str, List[str]]


class MCPToolSelector:
    """
    Intelligent MCP Tool Selection Service.

    Provides capability-based tool selection, LLM-guided selection,
    execution planning, and integration with the processing pipeline.
    """

    def __init__(self,
                 tool_registry: MCPUnifiedToolRegistry,
                 server_registry: MCPServerRegistry,
                 tool_discovery: MCPToolDiscovery):
        self.tool_registry = tool_registry
        self.server_registry = server_registry
        self.tool_discovery = tool_discovery
        self.logger = logging.getLogger("MCPToolSelector")

        # Configuration
        self._selection_strategy = ToolSelectionStrategy.HYBRID
        self._max_tools_per_phase = 5
        self._tool_timeout_ms = 30000  # 30 seconds
        self._enable_fallback = True
        self._enable_caching = True

        # Tool capability cache
        self._capability_cache: Dict[str, ToolCapability] = {}
        self._cache_timestamp: Optional[datetime] = None
        self._cache_ttl_seconds = 300  # 5 minutes

        # Domain keywords for matching
        self._domain_keywords = {
            "programming": {"code", "function", "api", "programming", "script", "development"},
            "data_analysis": {"data", "analysis", "statistics", "metrics", "analytics"},
            "project_management": {"task", "epic", "project", "milestone", "progress"},
            "documentation": {"docs", "documentation", "readme", "guide", "tutorial"},
            "testing": {"test", "testing", "validation", "verification", "quality"},
            "deployment": {"deploy", "deployment", "production", "release", "staging"}
        }

    def set_selection_strategy(self, strategy: ToolSelectionStrategy):
        """Set the tool selection strategy."""
        self._selection_strategy = strategy
        self.logger.info(f"Tool selection strategy changed to: {strategy.value}")

    async def analyze_tool_capabilities(self) -> Dict[str, Any]:
        """
        Analyze available MCP tools and extract their capabilities.

        Returns:
            Dictionary with capability analysis results
        """
        try:
            self.logger.debug("Analyzing MCP tool capabilities...")

            # Check cache first
            if self._is_cache_valid():
                return {
                    "status": "success",
                    "capabilities": list(self._capability_cache.values()),
                    "cached": True,
                    "cache_age": (datetime.now() - self._cache_timestamp).total_seconds()
                }

            # Get all available tools
            tool_infos = self.tool_discovery.get_all_tools()
            # Convert MCPToolInfo objects to dict format expected by the rest of the code
            tools = []
            for tool_info in tool_infos:
                tools.append({
                    "name": tool_info.name,
                    "server_name": tool_info.server_name,
                    "description": tool_info.description,
                    "input_schema": tool_info.input_schema
                })

            capabilities = []

            for tool_info in tools:
                capability = await self._analyze_single_tool_capability(tool_info)
                if capability:
                    capabilities.append(capability)
                    self._capability_cache[capability.name] = capability

            # Update cache timestamp
            self._cache_timestamp = datetime.now()

            self.logger.info(f"Analyzed {len(capabilities)} tool capabilities")

            return {
                "status": "success",
                "capabilities": capabilities,
                "cached": False,
                "total_tools": len(capabilities)
            }

        except Exception as e:
            self.logger.error(f"Error analyzing tool capabilities: {e}")
            return {
                "status": "error",
                "error": str(e),
                "capabilities": []
            }

    async def _analyze_single_tool_capability(self, tool_info: Dict[str, Any]) -> Optional[ToolCapability]:
        """Analyze a single tool's capabilities."""
        try:
            tool_name = tool_info.get("name")
            description = tool_info.get("description", "")
            input_schema = tool_info.get("input_schema", {})
            server_name = tool_info.get("server_name", "")

            # Extract domains from description and name
            text_content = f"{tool_name} {description}".lower()
            domains = set()
            keywords = set()

            for domain, domain_keywords in self._domain_keywords.items():
                if any(keyword in text_content for keyword in domain_keywords):
                    domains.add(domain)
                    keywords.update(domain_keywords.intersection(set(text_content.split())))

            # Determine complexity level
            complexity = self._determine_complexity(input_schema, description)

            # Determine applicable processing phases
            processing_phases = self._determine_processing_phases(tool_name, description)

            return ToolCapability(
                name=tool_name,
                description=description,
                input_schema=input_schema,
                domains=domains,
                complexity_level=complexity,
                processing_phases=processing_phases,
                keywords=keywords,
                server_name=server_name,
                confidence_score=0.0  # Will be calculated during matching
            )

        except Exception as e:
            self.logger.warning(f"Error analyzing tool {tool_info.get('name', 'unknown')}: {e}")
            return None

    def _determine_complexity(self, input_schema: Dict[str, Any], description: str) -> str:
        """Determine tool complexity level."""
        # Simple heuristic based on input parameters and description
        properties = input_schema.get("properties", {})
        param_count = len(properties)

        if param_count <= 2:
            return "simple"
        elif param_count <= 5:
            return "moderate"
        else:
            return "complex"

    def _determine_processing_phases(self, tool_name: str, description: str) -> Set[ProcessingPhase]:
        """Determine which processing phases this tool is suitable for."""
        phases = set()
        text = f"{tool_name} {description}".lower()

        # Preprocessing indicators
        if any(word in text for word in ["validate", "preprocess", "prepare", "analyze", "extract"]):
            phases.add(ProcessingPhase.PREPROCESSING)

        # Reasoning indicators
        if any(word in text for word in ["reason", "think", "analyze", "understand", "interpret"]):
            phases.add(ProcessingPhase.REASONING)

        # Postprocessing indicators
        if any(word in text for word in ["format", "enhance", "improve", "optimize", "polish"]):
            phases.add(ProcessingPhase.POSTPROCESSING)

        # Validation indicators
        if any(word in text for word in ["validate", "verify", "check", "test", "quality"]):
            phases.add(ProcessingPhase.VALIDATION)

        # Default to preprocessing if no specific phase detected
        if not phases:
            phases.add(ProcessingPhase.PREPROCESSING)

        return phases

    async def select_tools_for_context(self, context: ToolSelectionContext) -> Dict[str, Any]:
        """
        Select appropriate MCP tools based on the given context.

        Args:
            context: Tool selection context with request data and metadata

        Returns:
            Dictionary with selected tools and selection metadata
        """
        try:
            self.logger.debug(f"Selecting tools for {context.processing_phase.value} phase")

            # Ensure capabilities are analyzed
            capability_result = await self.analyze_tool_capabilities()
            if capability_result.get("status") != "success":
                return {
                    "status": "error",
                    "error": "Failed to analyze tool capabilities",
                    "selected_tools": []
                }

            capabilities = capability_result["capabilities"]

            # Filter tools by processing phase
            phase_tools = [
                cap for cap in capabilities
                if context.processing_phase in cap.processing_phases
            ]

            if not phase_tools:
                return {
                    "status": "success",
                    "selected_tools": [],
                    "reason": f"No tools available for {context.processing_phase.value} phase"
                }

            # Apply selection strategy
            if self._selection_strategy == ToolSelectionStrategy.CAPABILITY_MATCH:
                selected = await self._select_by_capability_match(context, phase_tools)
            elif self._selection_strategy == ToolSelectionStrategy.LLM_GUIDED:
                selected = await self._select_by_llm_guidance(context, phase_tools)
            elif self._selection_strategy == ToolSelectionStrategy.HYBRID:
                selected = await self._select_by_hybrid_approach(context, phase_tools)
            else:  # RULE_BASED
                selected = await self._select_by_rules(context, phase_tools)

            # Limit number of tools
            if len(selected) > self._max_tools_per_phase:
                selected = selected[:self._max_tools_per_phase]
                self.logger.warning(f"Limited tool selection to {self._max_tools_per_phase} tools")

            self.logger.info(f"Selected {len(selected)} tools for {context.processing_phase.value}: {[t.name for t in selected]}")

            return {
                "status": "success",
                "selected_tools": [tool.name for tool in selected],
                "tool_details": selected,
                "selection_strategy": self._selection_strategy.value,
                "processing_phase": context.processing_phase.value
            }

        except Exception as e:
            self.logger.error(f"Error selecting tools: {e}")
            return {
                "status": "error",
                "error": str(e),
                "selected_tools": []
            }

    async def _select_by_capability_match(self,
                                         context: ToolSelectionContext,
                                         available_tools: List[ToolCapability]) -> List[ToolCapability]:
        """Select tools based on capability matching."""
        # Extract keywords from request
        request_text = self._extract_request_text(context.request_data)
        request_keywords = set(request_text.lower().split())

        # Score tools based on keyword overlap and domain match
        scored_tools = []

        for tool in available_tools:
            score = 0.0

            # Keyword matching
            keyword_overlap = tool.keywords.intersection(request_keywords)
            score += len(keyword_overlap) * 2.0

            # Domain matching from intent analysis
            if context.intent_analysis:
                intent_domains = set(context.intent_analysis.get("domains", []))
                domain_overlap = tool.domains.intersection(intent_domains)
                score += len(domain_overlap) * 3.0

            # Complexity matching
            if context.intent_analysis:
                request_complexity = context.intent_analysis.get("complexity", "simple")
                if tool.complexity_level == request_complexity:
                    score += 1.0

            tool.confidence_score = score
            if score > 0:
                scored_tools.append(tool)

        # Sort by score and return top tools
        scored_tools.sort(key=lambda x: x.confidence_score, reverse=True)
        return scored_tools

    async def _select_by_llm_guidance(self,
                                     context: ToolSelectionContext,
                                     available_tools: List[ToolCapability]) -> List[ToolCapability]:
        """Select tools using LLM guidance."""
        try:
            from google.adk.agents import Agent

            # Create LLM-guided tool selection prompt
            request_text = self._extract_request_text(context.request_data)
            intent_info = context.intent_analysis or {}

            # Prepare tool information for LLM
            tool_descriptions = []
            for tool in available_tools:
                tool_info = {
                    "name": tool.name,
                    "description": tool.description,
                    "domains": list(tool.domains),
                    "complexity": tool.complexity_level,
                    "keywords": list(tool.keywords)
                }
                tool_descriptions.append(tool_info)

            # Create LLM prompt for tool selection
            selection_prompt = f"""
You are an expert MCP tool selector. Given the following request and available tools, select the most appropriate tools for the {context.processing_phase.value} phase.

REQUEST: {request_text}

REQUEST ANALYSIS:
- Complexity: {intent_info.get('complexity', 'unknown')}
- Domains: {intent_info.get('domains', [])}
- Processing Phase: {context.processing_phase.value}

AVAILABLE TOOLS:
{json.dumps(tool_descriptions, indent=2)}

SELECTION CRITERIA:
1. Tools should match the request domain and complexity
2. Prioritize tools that directly address the request needs
3. Consider the processing phase requirements
4. Limit to maximum 3-5 most relevant tools
5. Explain your reasoning

Please respond with a JSON object containing:
{{
    "selected_tools": ["tool1", "tool2", ...],
    "reasoning": "explanation of selection",
    "confidence": 0.0-1.0
}}
"""

            # Create a simple LLM agent for tool selection
            tool_selection_agent = Agent(
                name="tool_selector_llm",
                model="gemini-2.0-flash-thinking",  # Use thinking model for better reasoning
                instruction="You are an expert at selecting the most appropriate MCP tools based on request context and requirements."
            )

            # Get LLM guidance
            try:
                # Use the correct ADK Agent API with async generator
                async_gen = tool_selection_agent.run_async(selection_prompt)
                llm_response = ""
                async for chunk in async_gen:
                    llm_response += chunk

                # Parse LLM response
                if llm_response and isinstance(llm_response, str):
                    # Try to extract JSON from response
                    import re
                    json_match = re.search(r'\{.*\}', llm_response, re.DOTALL)
                    if json_match:
                        selection_data = json.loads(json_match.group())
                        selected_tool_names = selection_data.get("selected_tools", [])
                        reasoning = selection_data.get("reasoning", "")
                        confidence = selection_data.get("confidence", 0.5)

                        # Filter available tools based on LLM selection
                        selected_tools = []
                        for tool in available_tools:
                            if tool.name in selected_tool_names:
                                tool.confidence_score = confidence
                                selected_tools.append(tool)

                        self.logger.info(f"LLM guided selection: {len(selected_tools)} tools selected with confidence {confidence}")
                        self.logger.debug(f"LLM reasoning: {reasoning}")

                        return selected_tools

            except asyncio.TimeoutError:
                self.logger.warning("LLM tool selection timed out, falling back to capability matching")
            except json.JSONDecodeError:
                self.logger.warning("Failed to parse LLM tool selection response, falling back to capability matching")
            except Exception as e:
                self.logger.warning(f"LLM tool selection failed: {e}, falling back to capability matching")

        except ImportError:
            self.logger.warning("LLM agent not available, falling back to capability matching")
        except Exception as e:
            self.logger.warning(f"LLM-guided selection error: {e}, falling back to capability matching")

        # Fallback to capability matching
        return await self._select_by_capability_match(context, available_tools)

    async def _select_by_hybrid_approach(self,
                                        context: ToolSelectionContext,
                                        available_tools: List[ToolCapability]) -> List[ToolCapability]:
        """Select tools using hybrid approach (capability + LLM guidance)."""
        # Start with capability matching to get initial candidates
        capability_selected = await self._select_by_capability_match(context, available_tools)

        # If we have many candidates, use LLM to refine the selection
        if len(capability_selected) > self._max_tools_per_phase:
            self.logger.debug("Using LLM guidance to refine capability-based selection")

            # Use LLM guidance on the top capability-based candidates
            top_candidates = capability_selected[:self._max_tools_per_phase * 2]  # Give LLM more options
            llm_selected = await self._select_by_llm_guidance(context, top_candidates)

            # If LLM selection was successful, use it; otherwise fall back to capability-based
            if llm_selected and len(llm_selected) <= self._max_tools_per_phase:
                self.logger.info(f"Hybrid selection: LLM refined {len(capability_selected)} candidates to {len(llm_selected)} tools")
                return llm_selected
            else:
                self.logger.debug("LLM refinement not effective, using capability-based selection")

        # Return top capability-based candidates
        return capability_selected[:self._max_tools_per_phase]

    async def _select_by_rules(self,
                              context: ToolSelectionContext,
                              available_tools: List[ToolCapability]) -> List[ToolCapability]:
        """Select tools using rule-based approach."""
        selected = []

        # Rule 1: Always include validation tools for postprocessing
        if context.processing_phase == ProcessingPhase.POSTPROCESSING:
            validation_tools = [t for t in available_tools if "validate" in t.name.lower()]
            selected.extend(validation_tools[:1])  # At most one validation tool

        # Rule 2: Include analysis tools for preprocessing
        if context.processing_phase == ProcessingPhase.PREPROCESSING:
            analysis_tools = [t for t in available_tools if any(word in t.name.lower() for word in ["analyze", "extract", "scan"])]
            selected.extend(analysis_tools[:2])  # At most two analysis tools

        # Rule 3: Include reasoning tools for reasoning phase
        if context.processing_phase == ProcessingPhase.REASONING:
            reasoning_tools = [t for t in available_tools if any(word in t.name.lower() for word in ["reason", "think", "understand"])]
            selected.extend(reasoning_tools[:1])  # At most one reasoning tool

        return selected

    async def create_execution_plan(self, selected_tools: List[str], context: ToolSelectionContext) -> Dict[str, Any]:
        """
        Create an execution plan for the selected tools.

        Args:
            selected_tools: List of selected tool names
            context: Tool selection context

        Returns:
            Dictionary with execution plan
        """
        try:
            if not selected_tools:
                return {
                    "status": "success",
                    "execution_plan": None,
                    "reason": "No tools selected"
                }

            # Create execution plan
            plan = ToolExecutionPlan(
                tools=selected_tools,
                execution_order=selected_tools.copy(),  # Simple sequential order for now
                parallel_groups=[],  # TODO: Implement parallel execution analysis
                dependencies={},     # TODO: Implement dependency analysis
                timeout_ms=self._tool_timeout_ms,
                retry_count=3,
                fallback_tools={}    # TODO: Implement fallback tool mapping
            )

            # TODO: Analyze tool dependencies and create parallel execution groups

            self.logger.info(f"Created execution plan for {len(selected_tools)} tools")

            return {
                "status": "success",
                "execution_plan": plan,
                "estimated_duration_ms": len(selected_tools) * 1000  # Rough estimate
            }

        except Exception as e:
            self.logger.error(f"Error creating execution plan: {e}")
            return {
                "status": "error",
                "error": str(e),
                "execution_plan": None
            }

    async def execute_tool_plan(self, plan: ToolExecutionPlan, context: ToolSelectionContext) -> Dict[str, Any]:
        """
        Execute the tool execution plan.

        Args:
            plan: Tool execution plan
            context: Tool selection context

        Returns:
            Dictionary with execution results
        """
        try:
            if not plan or not plan.tools:
                return {
                    "status": "success",
                    "results": [],
                    "reason": "No tools to execute"
                }

            results = []
            errors = []

            self.logger.info(f"Executing {len(plan.tools)} tools")

            # Execute tools in order (TODO: implement parallel execution)
            for tool_name in plan.execution_order:
                try:
                    # Prepare arguments based on context
                    arguments = self._prepare_tool_arguments(tool_name, context)

                    # Execute tool
                    result = await self.tool_registry.execute_tool(
                        tool_name=tool_name,
                        arguments=arguments,
                        timeout=plan.timeout_ms / 1000.0  # Convert to seconds
                    )

                    results.append(result)

                    if result.success:
                        self.logger.debug(f"Tool {tool_name} executed successfully")
                    else:
                        self.logger.warning(f"Tool {tool_name} execution failed: {result.error_message}")
                        errors.append(f"{tool_name}: {result.error_message}")

                        # TODO: Implement fallback tool execution

                except Exception as e:
                    error_msg = f"Error executing tool {tool_name}: {str(e)}"
                    self.logger.error(error_msg)
                    errors.append(error_msg)

            success_count = sum(1 for r in results if r.success)

            self.logger.info(f"Execution completed: {success_count}/{len(results)} tools succeeded")

            return {
                "status": "success",
                "results": results,
                "success_count": success_count,
                "total_count": len(results),
                "errors": errors,
                "execution_time_ms": sum(r.execution_time_ms or 0 for r in results)
            }

        except Exception as e:
            self.logger.error(f"Error executing tool plan: {e}")
            return {
                "status": "error",
                "error": str(e),
                "results": []
            }

    def _prepare_tool_arguments(self, tool_name: str, context: ToolSelectionContext) -> Dict[str, Any]:
        """Prepare arguments for tool execution based on context."""
        # Basic argument preparation - this would be enhanced based on specific tool requirements
        arguments = {}

        # Tool-specific argument preparation
        if tool_name == "find_assigned_tickets":
            # YouTrack find_assigned_tickets expects 'state' and 'project' parameters
            arguments["state"] = "Open"  # Default to open tickets
            # Don't add 'query' or 'intent' parameters as they're not expected by this tool
        elif tool_name == "get_user_details":
            # get_user_details typically doesn't need additional parameters
            pass
        else:
            # Common arguments that many tools might use
            request_text = self._extract_request_text(context.request_data)
            if request_text:
                arguments["query"] = request_text
                arguments["text"] = request_text

            # Add intent analysis for tools that might use it
            if context.intent_analysis:
                arguments["intent"] = context.intent_analysis

        return arguments

    def _extract_request_text(self, request_data: Dict[str, Any]) -> str:
        """Extract text content from request data."""
        messages = request_data.get("messages", [])
        if not messages:
            return ""

        # Get the last user message
        user_messages = [msg for msg in messages if msg.get("role") == "user"]
        if user_messages:
            return user_messages[-1].get("content", "")

        return ""

    def _is_cache_valid(self) -> bool:
        """Check if capability cache is still valid."""
        if not self._cache_timestamp or not self._capability_cache:
            return False

        age = (datetime.now() - self._cache_timestamp).total_seconds()
        return age < self._cache_ttl_seconds

    def get_selection_stats(self) -> Dict[str, Any]:
        """Get tool selection statistics."""
        return {
            "strategy": self._selection_strategy.value,
            "max_tools_per_phase": self._max_tools_per_phase,
            "tool_timeout_ms": self._tool_timeout_ms,
            "cached_capabilities": len(self._capability_cache),
            "cache_valid": self._is_cache_valid(),
            "available_domains": list(self._domain_keywords.keys()),
            "processing_phases": [phase.value for phase in ProcessingPhase]
        }

    def clear_cache(self):
        """Clear capability cache."""
        self._capability_cache.clear()
        self._cache_timestamp = None
        self.logger.info("Tool selection cache cleared")