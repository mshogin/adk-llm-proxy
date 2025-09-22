import json
import logging
import asyncio
from typing import Dict, List, Any, AsyncGenerator, Optional
import httpx
from google.adk.agents import LlmAgent
from google.adk.tools import ToolContext
from src.infrastructure.config.config import config

logger = logging.getLogger(__name__)

class StreamingProxy:
    """Handles streaming proxy operations to OpenAI API."""

    def __init__(self):
        self.client = None

    async def get_client(self) -> httpx.AsyncClient:
        """Get or create HTTP client."""
        if self.client is None:
            self.client = httpx.AsyncClient(
                timeout=httpx.Timeout(60.0),
                headers={"Authorization": f"Bearer {config.OPENAI_API_KEY}"}
            )
        return self.client

    async def close(self):
        """Close the HTTP client."""
        if self.client:
            await self.client.aclose()
            self.client = None

# Global proxy instance
streaming_proxy = StreamingProxy()

async def forward_to_openai(request_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Forward the request to OpenAI API and handle the response.

    Args:
        request_data: Processed request data to send to OpenAI

    Returns:
        Dictionary with response or streaming information
    """
    try:
        # Ensure model is set
        if "model" not in request_data or not request_data["model"]:
            request_data["model"] = config.OPENAI_DEFAULT_MODEL

        client = await streaming_proxy.get_client()
        stream_enabled = request_data.get("stream", False)

        if not stream_enabled:
            # Non-streaming request
            response = await client.post(
                f"{config.OPENAI_BASE_URL}/chat/completions",
                json=request_data
            )

            if response.status_code == 200:
                return {
                    "status": "success",
                    "response": response.json(),
                    "streaming": False
                }
            else:
                return {
                    "status": "error",
                    "error": f"OpenAI API error: {response.status_code} - {response.text}",
                    "streaming": False
                }
        else:
            # For streaming, we need to return connection info
            return {
                "status": "success",
                "streaming": True,
                "message": "Streaming response initiated"
            }

    except Exception as e:
        logger.error(f"Error forwarding to OpenAI: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "streaming": False
        }

async def stream_from_openai(request_data: Dict[str, Any]) -> AsyncGenerator[Dict[str, Any], None]:
    """
    Stream response from OpenAI API.

    Args:
        request_data: Processed request data to send to OpenAI

    Yields:
        Dictionary with streaming chunks
    """
    try:
        client = await streaming_proxy.get_client()

        async with client.stream(
            "POST",
            f"{config.OPENAI_BASE_URL}/chat/completions",
            json=request_data
        ) as response:

            if response.status_code != 200:
                yield {
                    "status": "error",
                    "error": f"OpenAI API error: {response.status_code}",
                    "chunk": None
                }
                return

            full_content = ""

            async for line in response.aiter_lines():
                if line.startswith("data: "):
                    data = line[6:].strip()

                    if data == "[DONE]":
                        # End of stream - yield final processing info
                        yield {
                            "status": "stream_complete",
                            "chunk": None,
                            "full_content": full_content,
                            "final": True
                        }
                        break

                    try:
                        chunk_data = json.loads(data)

                        # Extract content from chunk
                        content = ""
                        if "choices" in chunk_data and len(chunk_data["choices"]) > 0:
                            choice = chunk_data["choices"][0]
                            if "delta" in choice and "content" in choice["delta"]:
                                content = choice["delta"]["content"]
                                full_content += content

                        yield {
                            "status": "streaming",
                            "chunk": chunk_data,
                            "content": content,
                            "full_content_so_far": full_content,
                            "final": False
                        }

                    except json.JSONDecodeError as e:
                        logger.warning(f"Failed to parse streaming chunk: {e}")
                        continue

    except Exception as e:
        logger.error(f"Error in streaming from OpenAI: {str(e)}")
        yield {
            "status": "error",
            "error": str(e),
            "chunk": None,
            "final": True
        }

def prepare_openai_request(processed_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Prepare the final request data for OpenAI API.

    Args:
        processed_data: Data that has been preprocessed

    Returns:
        Dictionary with OpenAI-compatible request
    """
    try:
        # Extract the processed messages
        messages = processed_data.get("messages", [])

        # Build OpenAI request
        openai_request = {
            "model": processed_data.get("model", config.OPENAI_DEFAULT_MODEL),
            "messages": messages,
            "stream": processed_data.get("stream", False)
        }

        # Add optional parameters if present
        optional_params = ["temperature", "max_tokens", "top_p", "frequency_penalty", "presence_penalty"]
        for param in optional_params:
            if param in processed_data:
                openai_request[param] = processed_data[param]

        return {
            "status": "success",
            "openai_request": openai_request,
            "message_count": len(messages),
            "estimated_tokens": sum(len(msg.get("content", "")) for msg in messages) // 4
        }

    except Exception as e:
        logger.error(f"Error preparing OpenAI request: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "openai_request": None
        }

def extract_response_metadata(response_data: Dict[str, Any]) -> Dict[str, Any]:
    """
    Extract metadata from OpenAI response for analytics.

    Args:
        response_data: Response from OpenAI API

    Returns:
        Dictionary with extracted metadata
    """
    try:
        metadata = {
            "response_id": response_data.get("id"),
            "model_used": response_data.get("model"),
            "created": response_data.get("created"),
            "finish_reason": None,
            "total_tokens": 0,
            "prompt_tokens": 0,
            "completion_tokens": 0
        }

        # Extract usage information
        if "usage" in response_data:
            usage = response_data["usage"]
            metadata.update({
                "total_tokens": usage.get("total_tokens", 0),
                "prompt_tokens": usage.get("prompt_tokens", 0),
                "completion_tokens": usage.get("completion_tokens", 0)
            })

        # Extract finish reason
        if "choices" in response_data and len(response_data["choices"]) > 0:
            choice = response_data["choices"][0]
            metadata["finish_reason"] = choice.get("finish_reason")

        return {
            "status": "success",
            "metadata": metadata
        }

    except Exception as e:
        logger.error(f"Error extracting response metadata: {str(e)}")
        return {
            "status": "error",
            "error": str(e),
            "metadata": {}
        }

# Create the proxy agent using new ADK API
from google.adk.agents import Agent

proxy_agent = Agent(
    name="proxy_agent",
    model="gemini-2.0-flash",
    tools=[forward_to_openai, prepare_openai_request, extract_response_metadata],
    instruction="""You are a proxy agent for forwarding requests to OpenAI API.

    Your responsibilities:
    1. Prepare OpenAI-compatible requests using prepare_openai_request()
    2. Forward requests to OpenAI using forward_to_openai()
    3. Extract response metadata using extract_response_metadata()

    When processing a request:
    1. Prepare the request in OpenAI format
    2. Forward it to the OpenAI API
    3. Extract metadata from responses for analytics
    4. Handle both streaming and non-streaming responses appropriately

    Always maintain the integrity of the original request while ensuring proper formatting.""",
    description="Handles proxying requests to OpenAI API with streaming support"
)
