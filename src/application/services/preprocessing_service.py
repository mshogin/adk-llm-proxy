import json
import logging
from typing import Dict, List, Any, Optional
from google.adk.agents import LlmAgent
from google.adk.tools import ToolContext
from src.infrastructure.config.config import config

logger = logging.getLogger(__name__)

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

        # Add system prompt if not present and context injection is enabled
        if config.ENABLE_CONTEXT_INJECTION:
            has_system_message = any(msg.get('role') == 'system' for msg in messages)

            if not has_system_message:
                system_message = {
                    "role": "system",
                    "content": config.SYSTEM_PROMPT_PREFIX
                }

                # Add context information to system prompt if provided
                if context:
                    context_info = "\n\nAdditional Context:\n"
                    for key, value in context.items():
                        if isinstance(value, (str, int, float)):
                            context_info += f"- {key}: {value}\n"

                    system_message["content"] += context_info

                processed_messages.append(system_message)

        # Process existing messages
        for message in messages:
            processed_msg = message.copy()

            # Ensure message content doesn't exceed max length
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

# Create the preprocessing agent using new ADK API
from google.adk.agents import Agent

preprocessing_agent = Agent(
    name="preprocessing_agent",
    model="gemini-2.0-flash",
    tools=[inject_context, extract_request_metadata, validate_request],
    instruction="""You are a preprocessing agent for an LLM reverse proxy server.

    Your responsibilities:
    1. Validate incoming requests using validate_request()
    2. Extract metadata from requests using extract_request_metadata()
    3. Inject context and system prompts using inject_context()

    When processing a request:
    1. First validate the request to ensure it's properly formatted
    2. Extract metadata for analytics and routing decisions
    3. Inject appropriate context based on the conversation
    4. Return the processed request ready for forwarding to the LLM

    Always return structured data with clear status indicators and error handling.""",
    description="Handles message preprocessing, validation, and context injection for LLM requests"
)
