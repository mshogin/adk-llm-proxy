"""
MCP Tool Capability Introspection and Availability Tracking.

This module provides advanced introspection capabilities for MCP tools,
including schema analysis, compatibility checking, and availability monitoring.
"""
import asyncio
import logging
from typing import Dict, List, Optional, Any, Set, Tuple, Union
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass, field
import json
import re
from collections import defaultdict

from .discovery import MCPToolDiscovery, MCPToolInfo, MCPResourceInfo, MCPPromptInfo, ToolAvailabilityStatus
from .registry import MCPServerRegistry
from .tool_registry import MCPUnifiedToolRegistry


class SchemaComplexity(Enum):
    """Tool schema complexity levels."""
    SIMPLE = "simple"        # No parameters or simple types only
    MODERATE = "moderate"    # Some complex types, few nested objects
    COMPLEX = "complex"      # Many nested objects, arrays, complex validation
    ADVANCED = "advanced"    # Very complex schemas with deep nesting


class ToolCategory(Enum):
    """Tool category classifications."""
    FILE_SYSTEM = "file_system"
    DATABASE = "database"
    WEB_API = "web_api"
    SEARCH = "search"
    ANALYSIS = "analysis"
    GENERATION = "generation"
    COMMUNICATION = "communication"
    DEVELOPMENT = "development"
    SYSTEM = "system"
    UTILITY = "utility"
    CUSTOM = "custom"


@dataclass
class ToolCompatibilityInfo:
    """Information about tool compatibility and requirements."""
    min_parameters: int = 0
    max_parameters: int = 0
    required_parameters: Set[str] = field(default_factory=set)
    optional_parameters: Set[str] = field(default_factory=set)
    parameter_types: Dict[str, str] = field(default_factory=dict)
    supports_streaming: bool = False
    requires_auth: bool = False
    rate_limited: bool = False
    complexity: SchemaComplexity = SchemaComplexity.SIMPLE
    estimated_runtime_ms: Optional[int] = None


@dataclass
class ToolIntrospectionResult:
    """Comprehensive tool introspection result."""
    tool_name: str
    server_name: str
    category: ToolCategory
    compatibility: ToolCompatibilityInfo
    schema_analysis: Dict[str, Any]
    usage_patterns: Dict[str, Any]
    performance_metrics: Dict[str, Any]
    availability_history: List[Dict[str, Any]]
    recommendations: List[str]
    last_analyzed: datetime


class MCPAvailabilityTracker:
    """Advanced availability tracking for MCP tools and servers."""

    def __init__(self, registry: MCPServerRegistry, discovery: MCPToolDiscovery):
        self.registry = registry
        self.discovery = discovery
        self.logger = logging.getLogger("MCPAvailabilityTracker")

        # Availability tracking
        self._availability_history: Dict[str, List[Dict[str, Any]]] = defaultdict(list)
        self._max_history_entries = 1000
        self._tracking_task: Optional[asyncio.Task] = None
        self._tracking_interval = 60.0  # 1 minute

        # Performance tracking
        self._performance_metrics: Dict[str, Dict[str, Any]] = defaultdict(dict)

    async def start_tracking(self, interval: float = 60.0):
        """Start availability tracking."""
        if self._tracking_task and not self._tracking_task.done():
            return

        self._tracking_interval = interval
        self._tracking_task = asyncio.create_task(self._tracking_loop())
        self.logger.info(f"Started availability tracking (interval: {interval}s)")

    async def stop_tracking(self):
        """Stop availability tracking."""
        if self._tracking_task and not self._tracking_task.done():
            self._tracking_task.cancel()
            try:
                await self._tracking_task
            except asyncio.CancelledError:
                pass

        self.logger.info("Stopped availability tracking")

    async def _tracking_loop(self):
        """Main tracking loop."""
        while True:
            try:
                await self._update_availability_status()
                await asyncio.sleep(self._tracking_interval)
            except asyncio.CancelledError:
                break
            except Exception as e:
                self.logger.error(f"Error in availability tracking loop: {e}")
                await asyncio.sleep(10)

    async def _update_availability_status(self):
        """Update availability status for all tools."""
        all_tools = self.discovery.get_all_tools()

        for tool in all_tools:
            await self._check_tool_availability(tool)

    async def _check_tool_availability(self, tool: MCPToolInfo):
        """Check and record availability for a specific tool."""
        timestamp = datetime.now()

        try:
            # Get server info
            server_info = self.registry.get_server_info(tool.server_name)
            if not server_info or not server_info.is_healthy:
                status = ToolAvailabilityStatus.UNAVAILABLE
                error_msg = "Server not healthy"
                response_time = None
            else:
                # Test tool availability
                start_time = datetime.now()
                is_available = await self.discovery.update_tool_availability(tool.name)
                response_time = (datetime.now() - start_time).total_seconds() * 1000

                status = ToolAvailabilityStatus.AVAILABLE if is_available else ToolAvailabilityStatus.UNAVAILABLE
                error_msg = tool.error_message if not is_available else None

            # Record availability
            availability_record = {
                "timestamp": timestamp.isoformat(),
                "status": status.value,
                "response_time_ms": response_time,
                "error_message": error_msg,
                "server_status": server_info.status.value if server_info else "unknown"
            }

            self._availability_history[tool.name].append(availability_record)

            # Limit history size
            if len(self._availability_history[tool.name]) > self._max_history_entries:
                self._availability_history[tool.name].pop(0)

            # Update performance metrics
            if response_time is not None:
                self._update_performance_metrics(tool.name, response_time)

        except Exception as e:
            self.logger.error(f"Error checking availability for {tool.name}: {e}")

    def _update_performance_metrics(self, tool_name: str, response_time: float):
        """Update performance metrics for a tool."""
        metrics = self._performance_metrics[tool_name]

        # Initialize metrics if first time
        if "response_times" not in metrics:
            metrics["response_times"] = []
            metrics["avg_response_time"] = 0.0
            metrics["min_response_time"] = float('inf')
            metrics["max_response_time"] = 0.0

        # Add response time
        metrics["response_times"].append(response_time)

        # Keep only last 100 measurements
        if len(metrics["response_times"]) > 100:
            metrics["response_times"].pop(0)

        # Update statistics
        times = metrics["response_times"]
        metrics["avg_response_time"] = sum(times) / len(times)
        metrics["min_response_time"] = min(times)
        metrics["max_response_time"] = max(times)
        metrics["last_updated"] = datetime.now().isoformat()

    def get_availability_summary(self, tool_name: str) -> Optional[Dict[str, Any]]:
        """Get availability summary for a tool."""
        history = self._availability_history.get(tool_name, [])
        if not history:
            return None

        # Calculate availability percentage (last 24 hours)
        cutoff_time = datetime.now() - timedelta(hours=24)
        recent_records = [
            r for r in history[-100:]  # Last 100 records
            if datetime.fromisoformat(r["timestamp"]) > cutoff_time
        ]

        if not recent_records:
            return None

        available_count = sum(1 for r in recent_records if r["status"] == "available")
        availability_percentage = (available_count / len(recent_records)) * 100

        # Get performance metrics
        perf_metrics = self._performance_metrics.get(tool_name, {})

        return {
            "tool_name": tool_name,
            "availability_percentage_24h": availability_percentage,
            "total_checks": len(recent_records),
            "available_checks": available_count,
            "last_check": recent_records[-1] if recent_records else None,
            "performance_metrics": perf_metrics,
            "status_distribution": self._get_status_distribution(recent_records)
        }

    def _get_status_distribution(self, records: List[Dict[str, Any]]) -> Dict[str, int]:
        """Get distribution of statuses in records."""
        distribution = defaultdict(int)
        for record in records:
            distribution[record["status"]] += 1
        return dict(distribution)

    def get_server_availability_summary(self, server_name: str) -> Dict[str, Any]:
        """Get availability summary for all tools on a server."""
        server_tools = self.discovery.find_tools_by_server(server_name)

        summaries = []
        total_availability = 0.0

        for tool in server_tools:
            summary = self.get_availability_summary(tool.name)
            if summary:
                summaries.append(summary)
                total_availability += summary["availability_percentage_24h"]

        avg_availability = total_availability / len(summaries) if summaries else 0.0

        return {
            "server_name": server_name,
            "tool_count": len(server_tools),
            "monitored_tools": len(summaries),
            "average_availability_24h": avg_availability,
            "tool_summaries": summaries
        }


class MCPCapabilityIntrospector:
    """Advanced capability introspection for MCP tools."""

    def __init__(
        self,
        registry: MCPServerRegistry,
        discovery: MCPToolDiscovery,
        availability_tracker: MCPAvailabilityTracker
    ):
        self.registry = registry
        self.discovery = discovery
        self.availability_tracker = availability_tracker
        self.logger = logging.getLogger("MCPCapabilityIntrospector")

    async def introspect_tool(self, tool_name: str) -> Optional[ToolIntrospectionResult]:
        """Perform comprehensive introspection of a tool."""
        tool = self.discovery.get_tool(tool_name)
        if not tool:
            return None

        try:
            # Analyze tool schema
            compatibility = self._analyze_tool_compatibility(tool)
            schema_analysis = self._analyze_tool_schema(tool)
            category = self._classify_tool(tool)

            # Get usage patterns
            usage_patterns = self._analyze_usage_patterns(tool)

            # Get performance metrics
            perf_metrics = self.availability_tracker._performance_metrics.get(tool_name, {})

            # Get availability history
            availability_history = self.availability_tracker._availability_history.get(tool_name, [])

            # Generate recommendations
            recommendations = self._generate_recommendations(tool, compatibility, schema_analysis)

            return ToolIntrospectionResult(
                tool_name=tool.name,
                server_name=tool.server_name,
                category=category,
                compatibility=compatibility,
                schema_analysis=schema_analysis,
                usage_patterns=usage_patterns,
                performance_metrics=perf_metrics,
                availability_history=availability_history[-20:],  # Last 20 records
                recommendations=recommendations,
                last_analyzed=datetime.now()
            )

        except Exception as e:
            self.logger.error(f"Error introspecting tool {tool_name}: {e}")
            return None

    def _analyze_tool_compatibility(self, tool: MCPToolInfo) -> ToolCompatibilityInfo:
        """Analyze tool compatibility and requirements."""
        schema = tool.input_schema
        properties = schema.get("properties", {})
        required = set(schema.get("required", []))

        compatibility = ToolCompatibilityInfo()

        # Parameter analysis
        compatibility.required_parameters = required
        compatibility.optional_parameters = set(properties.keys()) - required
        compatibility.min_parameters = len(required)
        compatibility.max_parameters = len(properties)

        # Parameter types
        for param_name, param_schema in properties.items():
            param_type = param_schema.get("type", "unknown")
            compatibility.parameter_types[param_name] = param_type

        # Complexity analysis
        compatibility.complexity = self._determine_schema_complexity(schema)

        # Check for common patterns
        compatibility.supports_streaming = self._check_streaming_support(tool, schema)
        compatibility.requires_auth = self._check_auth_requirements(tool, schema)
        compatibility.rate_limited = self._check_rate_limiting(tool, schema)

        # Estimate runtime based on tool type and complexity
        compatibility.estimated_runtime_ms = self._estimate_runtime(tool, compatibility)

        return compatibility

    def _analyze_tool_schema(self, tool: MCPToolInfo) -> Dict[str, Any]:
        """Analyze tool input schema in detail."""
        schema = tool.input_schema

        analysis = {
            "has_schema": bool(schema),
            "schema_version": schema.get("$schema", "unknown"),
            "parameter_count": len(schema.get("properties", {})),
            "required_parameter_count": len(schema.get("required", [])),
            "nested_objects": 0,
            "arrays": 0,
            "validation_rules": 0,
            "enum_parameters": 0,
            "default_values": 0
        }

        # Analyze properties
        properties = schema.get("properties", {})
        for prop_name, prop_schema in properties.items():
            self._analyze_property_schema(prop_schema, analysis)

        # Check for advanced features
        analysis["has_conditionals"] = "if" in schema or "then" in schema or "else" in schema
        analysis["has_dependencies"] = "dependencies" in schema or "dependentSchemas" in schema
        analysis["has_pattern_properties"] = "patternProperties" in schema
        analysis["has_additional_properties"] = "additionalProperties" in schema

        return analysis

    def _analyze_property_schema(self, prop_schema: Dict[str, Any], analysis: Dict[str, Any]):
        """Recursively analyze property schema."""
        prop_type = prop_schema.get("type")

        if prop_type == "object":
            analysis["nested_objects"] += 1
            # Recursively analyze nested properties
            nested_props = prop_schema.get("properties", {})
            for nested_prop in nested_props.values():
                self._analyze_property_schema(nested_prop, analysis)

        elif prop_type == "array":
            analysis["arrays"] += 1
            # Analyze array items
            items = prop_schema.get("items", {})
            if items:
                self._analyze_property_schema(items, analysis)

        # Count validation rules
        validation_keywords = ["minimum", "maximum", "minLength", "maxLength", "pattern", "format"]
        analysis["validation_rules"] += sum(1 for keyword in validation_keywords if keyword in prop_schema)

        # Count enums and defaults
        if "enum" in prop_schema:
            analysis["enum_parameters"] += 1
        if "default" in prop_schema:
            analysis["default_values"] += 1

    def _determine_schema_complexity(self, schema: Dict[str, Any]) -> SchemaComplexity:
        """Determine schema complexity level."""
        properties = schema.get("properties", {})
        param_count = len(properties)
        required_count = len(schema.get("required", []))

        # Count nested structures
        nested_count = 0
        array_count = 0

        for prop_schema in properties.values():
            if prop_schema.get("type") == "object":
                nested_count += 1
            elif prop_schema.get("type") == "array":
                array_count += 1

        # Classify based on complexity metrics
        if param_count <= 2 and nested_count == 0 and array_count == 0:
            return SchemaComplexity.SIMPLE
        elif param_count <= 5 and nested_count <= 1 and array_count <= 1:
            return SchemaComplexity.MODERATE
        elif param_count <= 10 and nested_count <= 3 and array_count <= 3:
            return SchemaComplexity.COMPLEX
        else:
            return SchemaComplexity.ADVANCED

    def _classify_tool(self, tool: MCPToolInfo) -> ToolCategory:
        """Classify tool into a category based on name and description."""
        name_lower = tool.name.lower()
        desc_lower = tool.description.lower()
        text = f"{name_lower} {desc_lower}"

        # File system operations
        if any(keyword in text for keyword in ["file", "directory", "path", "read", "write", "list"]):
            return ToolCategory.FILE_SYSTEM

        # Database operations
        if any(keyword in text for keyword in ["database", "sql", "query", "table", "record"]):
            return ToolCategory.DATABASE

        # Web API operations
        if any(keyword in text for keyword in ["http", "api", "request", "web", "url", "fetch"]):
            return ToolCategory.WEB_API

        # Search operations
        if any(keyword in text for keyword in ["search", "find", "locate", "lookup"]):
            return ToolCategory.SEARCH

        # Analysis operations
        if any(keyword in text for keyword in ["analyze", "analysis", "calculate", "compute", "process"]):
            return ToolCategory.ANALYSIS

        # Generation operations
        if any(keyword in text for keyword in ["generate", "create", "build", "make", "produce"]):
            return ToolCategory.GENERATION

        # Communication operations
        if any(keyword in text for keyword in ["send", "message", "email", "notification", "communicate"]):
            return ToolCategory.COMMUNICATION

        # Development operations
        if any(keyword in text for keyword in ["code", "compile", "build", "test", "debug", "git"]):
            return ToolCategory.DEVELOPMENT

        # System operations
        if any(keyword in text for keyword in ["system", "process", "service", "monitor", "status"]):
            return ToolCategory.SYSTEM

        # Utility operations
        if any(keyword in text for keyword in ["convert", "transform", "format", "validate", "utility"]):
            return ToolCategory.UTILITY

        return ToolCategory.CUSTOM

    def _check_streaming_support(self, tool: MCPToolInfo, schema: Dict[str, Any]) -> bool:
        """Check if tool supports streaming."""
        # Check for streaming-related parameters
        properties = schema.get("properties", {})
        streaming_keywords = ["stream", "streaming", "chunk", "incremental"]

        for prop_name, prop_schema in properties.items():
            prop_name_lower = prop_name.lower()
            if any(keyword in prop_name_lower for keyword in streaming_keywords):
                return True

            # Check in description
            description = prop_schema.get("description", "").lower()
            if any(keyword in description for keyword in streaming_keywords):
                return True

        return False

    def _check_auth_requirements(self, tool: MCPToolInfo, schema: Dict[str, Any]) -> bool:
        """Check if tool requires authentication."""
        properties = schema.get("properties", {})
        auth_keywords = ["auth", "token", "key", "credential", "password", "login"]

        for prop_name, prop_schema in properties.items():
            prop_name_lower = prop_name.lower()
            if any(keyword in prop_name_lower for keyword in auth_keywords):
                return True

        return False

    def _check_rate_limiting(self, tool: MCPToolInfo, schema: Dict[str, Any]) -> bool:
        """Check if tool might be rate limited."""
        # Heuristic: web API tools are often rate limited
        category = self._classify_tool(tool)
        return category == ToolCategory.WEB_API

    def _estimate_runtime(self, tool: MCPToolInfo, compatibility: ToolCompatibilityInfo) -> int:
        """Estimate tool runtime in milliseconds."""
        base_time = 100  # Base 100ms

        # Adjust based on complexity
        complexity_multipliers = {
            SchemaComplexity.SIMPLE: 1.0,
            SchemaComplexity.MODERATE: 1.5,
            SchemaComplexity.COMPLEX: 2.0,
            SchemaComplexity.ADVANCED: 3.0
        }

        multiplier = complexity_multipliers[compatibility.complexity]

        # Adjust based on category
        category_adjustments = {
            ToolCategory.FILE_SYSTEM: 50,
            ToolCategory.DATABASE: 200,
            ToolCategory.WEB_API: 500,
            ToolCategory.SEARCH: 300,
            ToolCategory.ANALYSIS: 1000,
            ToolCategory.GENERATION: 2000,
            ToolCategory.COMMUNICATION: 800,
            ToolCategory.DEVELOPMENT: 1500,
            ToolCategory.SYSTEM: 100,
            ToolCategory.UTILITY: 200,
            ToolCategory.CUSTOM: 500
        }

        category = self._classify_tool(tool)
        category_time = category_adjustments.get(category, 500)

        return int(base_time * multiplier + category_time)

    def _analyze_usage_patterns(self, tool: MCPToolInfo) -> Dict[str, Any]:
        """Analyze tool usage patterns."""
        return {
            "usage_count": tool.usage_count,
            "last_used": tool.last_used.isoformat() if tool.last_used else None,
            "average_usage_frequency": self._calculate_usage_frequency(tool),
            "usage_trend": self._analyze_usage_trend(tool)
        }

    def _calculate_usage_frequency(self, tool: MCPToolInfo) -> Optional[float]:
        """Calculate usage frequency (uses per day)."""
        if not tool.last_used or tool.usage_count == 0:
            return None

        days_since_first_use = (datetime.now() - tool.last_used).days
        if days_since_first_use == 0:
            days_since_first_use = 1

        return tool.usage_count / days_since_first_use

    def _analyze_usage_trend(self, tool: MCPToolInfo) -> str:
        """Analyze usage trend."""
        # This is a simplified implementation
        # In a real system, you'd track usage over time
        if tool.usage_count == 0:
            return "unused"
        elif tool.usage_count < 10:
            return "low"
        elif tool.usage_count < 100:
            return "moderate"
        else:
            return "high"

    def _generate_recommendations(
        self,
        tool: MCPToolInfo,
        compatibility: ToolCompatibilityInfo,
        schema_analysis: Dict[str, Any]
    ) -> List[str]:
        """Generate recommendations for tool usage."""
        recommendations = []

        # Availability recommendations
        if tool.availability_status == ToolAvailabilityStatus.UNAVAILABLE:
            recommendations.append("Tool is currently unavailable - check server status")

        # Complexity recommendations
        if compatibility.complexity == SchemaComplexity.ADVANCED:
            recommendations.append("Complex tool - consider validating parameters carefully")

        # Performance recommendations
        if compatibility.estimated_runtime_ms and compatibility.estimated_runtime_ms > 5000:
            recommendations.append("Long-running tool - consider using with timeout")

        # Usage recommendations
        if tool.usage_count == 0:
            recommendations.append("Unused tool - consider testing before production use")

        # Schema recommendations
        if schema_analysis["parameter_count"] > 10:
            recommendations.append("Many parameters - consider using parameter objects")

        if schema_analysis["required_parameter_count"] == 0:
            recommendations.append("No required parameters - tool may have flexible usage")

        # Auth recommendations
        if compatibility.requires_auth:
            recommendations.append("Authentication required - ensure credentials are available")

        return recommendations

    def get_capability_overview(self) -> Dict[str, Any]:
        """Get comprehensive capability overview."""
        all_tools = self.discovery.get_all_tools()

        # Category distribution
        category_counts = defaultdict(int)
        complexity_counts = defaultdict(int)

        for tool in all_tools:
            category = self._classify_tool(tool)
            category_counts[category.value] += 1

        # Analyze a sample of tools for complexity
        sample_tools = all_tools[:min(50, len(all_tools))]
        for tool in sample_tools:
            compatibility = self._analyze_tool_compatibility(tool)
            complexity_counts[compatibility.complexity.value] += 1

        return {
            "total_tools": len(all_tools),
            "category_distribution": dict(category_counts),
            "complexity_distribution": dict(complexity_counts),
            "servers_with_tools": len(set(tool.server_name for tool in all_tools)),
            "tools_per_server": {
                server: len(self.discovery.find_tools_by_server(server))
                for server in set(tool.server_name for tool in all_tools)
            }
        }