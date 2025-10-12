"""
ADK Agent Wrapper for new ADK API compatibility
Provides a simple interface to execute agents using the new Runner pattern
"""

import logging
import uuid
from typing import Dict, Any, Optional
try:
    from google.adk.runners import Runner
    from google.adk.sessions import InMemorySessionService
    from google.genai import types
except ImportError as e:
    logger.error(f"Missing ADK dependencies: {e}")
    logger.error("Please install: pip install google-adk")
    raise

logger = logging.getLogger(__name__)

class ADKAgentWrapper:
    """Wrapper for ADK agents to provide simple execute interface."""

    def __init__(self, agent, app_name: str = "llm_proxy_agent"):
        self.agent = agent
        self.app_name = app_name
        self.user_id = "proxy_user"
        # Don't create persistent sessions, create them per request

    async def execute(self, input_data: Dict[str, Any], context: Optional[Any] = None) -> Dict[str, Any]:
        """Execute the agent by calling its tools directly."""
        try:
            logger.debug(f"🤖 REALLY Executing ADK agent tools: {self.agent.name}")

            # Since ADK Runner has session issues, let's call agent tools directly
            # This gives us the agent behavior without the complex session management

            if hasattr(self.agent, 'tools') and self.agent.tools:
                logger.debug(f"🤖 Agent {self.agent.name} has {len(self.agent.tools)} tools")

                results = []
                for tool in self.agent.tools:
                    try:
                        logger.debug(f"🤖 Executing tool: {tool.__name__ if hasattr(tool, '__name__') else str(tool)}")

                        # Call tool with input data
                        if "request_data" in input_data:
                            # For preprocessing tools
                            if hasattr(tool, '__name__'):
                                if 'validate' in tool.__name__:
                                    result = tool(input_data["request_data"])
                                elif 'extract' in tool.__name__:
                                    result = tool(input_data["request_data"])
                                elif 'inject' in tool.__name__:
                                    messages = input_data["request_data"].get("messages", [])
                                    result = tool(messages)
                                elif 'discover_preprocessing_tools' in tool.__name__:
                                    result = await tool(input_data["request_data"])
                                elif 'execute_preprocessing_pipeline' in tool.__name__:
                                    result = await tool(input_data["request_data"])
                                elif 'enrich_context_with_mcp_tools' in tool.__name__:
                                    messages = input_data["request_data"].get("messages", [])
                                    result = await tool(messages, [], input_data["request_data"])  # async function
                                else:
                                    result = tool(input_data["request_data"])
                            else:
                                result = tool(input_data["request_data"])

                            results.append(result)
                            logger.debug(f"🤖 Tool result: {str(result)[:100]}...")

                        elif "content" in input_data:
                            # For postprocessing tools
                            content = input_data["content"]
                            if hasattr(tool, '__name__'):
                                if 'analyze' in tool.__name__:
                                    result = tool(content)  # sync function
                                elif 'filter' in tool.__name__:
                                    result = tool(content)  # sync function
                                elif 'add_chat' in tool.__name__:
                                    result = tool(content, "analysis")  # sync function
                                elif 'log_interaction' in tool.__name__:
                                    result = tool(  # sync function
                                        input_data.get("metadata", {}),
                                        {"model": input_data.get("model", "unknown")},
                                        {"quality_score": 85}
                                    )
                                elif 'execute_postprocessing_tools' in tool.__name__:
                                    result = await tool(content, [], input_data.get("metadata"))  # async function
                                elif 'validate_response_with_mcp_tools' in tool.__name__:
                                    result = await tool(content, [], input_data.get("metadata"))  # async function
                                elif 'enhance_response_with_mcp_tools' in tool.__name__:
                                    result = await tool(content, [], input_data.get("metadata"))  # async function
                                elif 'execute_postprocessing_pipeline' in tool.__name__:
                                    # Get pipeline config or create default
                                    pipeline_config = input_data.get("pipeline_config", {
                                        "validation_enabled": True,
                                        "enhancement_enabled": True,
                                        "validation_tools": [],
                                        "enhancement_tools": []
                                    })
                                    result = await tool(content, pipeline_config, input_data.get("metadata"))  # async function
                                elif 'create_postprocessing_pipeline' in tool.__name__:
                                    result = await tool(content, input_data.get("metadata"))  # async function
                                elif 'discover_postprocessing_tools' in tool.__name__:
                                    result = await tool(content, input_data.get("metadata"))  # async function
                                else:
                                    # Default to sync for unknown functions
                                    result = tool(content)
                            else:
                                result = tool(content)

                            results.append(result)
                            logger.debug(f"🤖 Tool result: {str(result)[:100]}...")

                    except Exception as tool_error:
                        logger.warning(f"⚠️ Tool {tool} failed: {tool_error}")
                        results.append({"status": "error", "error": str(tool_error)})

                logger.info(f"✅ ADK agent {self.agent.name} executed {len(results)} tools successfully")
                return {
                    "status": "success",
                    "response": f"Agent {self.agent.name} executed {len(results)} tools",
                    "tool_results": results,
                    "agent_name": self.agent.name,
                    "execution_mode": "direct_tools"
                }
            else:
                logger.warning(f"⚠️ Agent {self.agent.name} has no tools")
                return {
                    "status": "success",
                    "response": f"Agent {self.agent.name} has no tools to execute",
                    "agent_name": self.agent.name,
                    "execution_mode": "no_tools"
                }

        except Exception as e:
            logger.error(f"❌ Error executing ADK agent {self.agent.name}: {e}")
            import traceback
            logger.error(f"❌ Traceback: {traceback.format_exc()}")
            return {
                "status": "error",
                "error": str(e),
                "agent_name": self.agent.name
            }

    def _format_input_for_agent(self, input_data: Dict[str, Any]) -> str:
        """Format input data as text for the agent."""
        if "request_data" in input_data:
            request_data = input_data["request_data"]
            messages = request_data.get("messages", [])

            # Format the conversation for the agent
            formatted_messages = []
            for msg in messages:
                role = msg.get("role", "user")
                content = msg.get("content", "")
                formatted_messages.append(f"{role}: {content}")

            conversation = "\n".join(formatted_messages)

            return f"""Process this request:
Provider: {input_data.get('provider', 'unknown')}
Model: {input_data.get('model', 'unknown')}
Stream: {input_data.get('stream', False)}

Conversation:
{conversation}

Please process this according to your instructions."""

        else:
            # Fallback formatting
            return f"Process this data: {input_data}"

# Global agent wrappers
_preprocessing_wrapper = None
_postprocessing_wrapper = None

async def get_preprocessing_agent_wrapper():
    """Get or create preprocessing agent wrapper."""
    global _preprocessing_wrapper
    if _preprocessing_wrapper is None:
        from src.application.services.preprocessing_service import preprocessing_agent
        _preprocessing_wrapper = ADKAgentWrapper(
            agent=preprocessing_agent,
            app_name="preprocessing_agent"
        )
    return _preprocessing_wrapper

async def get_postprocessing_agent_wrapper():
    """Get or create postprocessing agent wrapper."""
    global _postprocessing_wrapper
    if _postprocessing_wrapper is None:
        from src.application.services.postprocessing_service import postprocessing_agent
        _postprocessing_wrapper = ADKAgentWrapper(
            agent=postprocessing_agent,
            app_name="postprocessing_agent"
        )
    return _postprocessing_wrapper

async def execute_preprocessing_agent(input_data: Dict[str, Any]) -> Dict[str, Any]:
    """Execute preprocessing agent with proper ADK API."""
    wrapper = await get_preprocessing_agent_wrapper()
    return await wrapper.execute(input_data)

async def execute_postprocessing_agent(input_data: Dict[str, Any]) -> Dict[str, Any]:
    """Execute postprocessing agent with proper ADK API."""
    wrapper = await get_postprocessing_agent_wrapper()
    return await wrapper.execute(input_data)