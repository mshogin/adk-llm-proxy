#!/usr/bin/env python3
"""
ADK-based LLM Reverse Proxy Server using Google Agent Development Kit
STREAMING ONLY: This server exclusively supports streaming responses with intelligent agent-based processing.
Pipeline: preprocessing → reasoning → LLM → postprocessing with orchestrator management.
"""

import asyncio
import json
import logging
import os
import time
from typing import Dict, List, Any, Optional, AsyncGenerator

from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import StreamingResponse
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
import uvicorn

import sys
from pathlib import Path
# Add parent directory to path for imports
project_root = Path(__file__).parent.parent.parent.parent
sys.path.insert(0, str(project_root))

from src.infrastructure.config.config import config
from src.application.services.orchestration_service import root_agent, llm_proxy_orchestrator
from google.adk.tools import ToolContext
import httpx
from src.application.services.postprocessing_service import analyze_response_content, add_chat_content, create_unified_analysis
from src.application.services.preprocessing_service import inject_context, validate_request, extract_request_metadata
from src.infrastructure.agents.adk_wrapper import execute_preprocessing_agent, execute_postprocessing_agent
from src.domain.services.content_filter_service import filter_messages_for_llm, add_analysis_markers

# Global HTTP client
http_client = None

async def get_http_client():
    global http_client
    if http_client is None:
        headers = {}
        if config.current_provider == "openai" and config.current_api_key:
            headers["Authorization"] = f"Bearer {config.current_api_key}"

        http_client = httpx.AsyncClient(
            timeout=httpx.Timeout(60.0),
            headers=headers
        )
    return http_client

async def cleanup_http_client():
    global http_client
    if http_client:
        await http_client.aclose()
        http_client = None

# Configure logging
logging.basicConfig(
    level=getattr(logging, config.LOG_LEVEL.upper()),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Check for DEBUG environment variable
DEBUG_MODE = os.getenv('DEBUG', 'false').lower() in ('true', '1', 'yes', 'on')
if DEBUG_MODE:
    logger.setLevel(logging.DEBUG)
    logger.debug("ADK DEBUG mode enabled via environment variable")

# Pydantic models
class Message(BaseModel):
    role: str = Field(..., description="Role of the message sender")
    content: str = Field(..., description="Content of the message")

class ChatCompletionRequest(BaseModel):
    model: Optional[str] = Field(default=config.current_model, description="Model to use")
    messages: List[Message] = Field(..., description="List of messages")
    max_tokens: Optional[int] = Field(default=None, description="Maximum tokens to generate")
    temperature: Optional[float] = Field(default=0.7, description="Sampling temperature")
    top_p: Optional[float] = Field(default=1.0, description="Nucleus sampling parameter")
    stream: Optional[bool] = Field(default=False, description="Enable streaming")
    frequency_penalty: Optional[float] = Field(default=0.0, description="Frequency penalty")
    presence_penalty: Optional[float] = Field(default=0.0, description="Presence penalty")

class ChatCompletionChoice(BaseModel):
    index: int
    message: Optional[Message] = None
    delta: Optional[Dict[str, Any]] = None
    finish_reason: Optional[str] = None

class ChatCompletionUsage(BaseModel):
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int

class ChatCompletionResponse(BaseModel):
    id: str
    object: str = "chat.completion"
    created: int
    model: str
    choices: List[ChatCompletionChoice]
    usage: Optional[ChatCompletionUsage] = None

# Create FastAPI app
app = FastAPI(
    title="ADK-Based LLM Streaming Proxy",
    description="Streaming-focused OpenAI-compatible API powered by Google ADK orchestrated agents pipeline",
    version="2.0.0-streaming"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add request logging middleware
@app.middleware("http")
async def log_requests(request: Request, call_next):
    """Log all requests for debugging."""
    logger.debug("ADK log_requests")

    try:
        logger.info(f"🤖 ADK {request.method} {request.url.path} from {request.client}")

        if DEBUG_MODE:
            logger.debug("=" * 60)
            logger.debug(f"🔍 ADK DETAILED REQUEST DEBUG:")
            logger.debug(f"   Method: {request.method}")
            logger.debug(f"   URL: {request.url}")
            logger.debug(f"   Path: {request.url.path}")
            logger.debug(f"   Query: {request.url.query}")
            logger.debug(f"   Client: {request.client}")
            logger.debug(f"   Headers:")
            for key, value in request.headers.items():
                logger.debug(f"     {key}: {value}")
            logger.debug("=" * 60)

        response = await call_next(request)

        if DEBUG_MODE:
            logger.debug(f"🤖 ADK Response: {response.status_code}")
        else:
            logger.info(f"🤖 ADK Response: {response.status_code}")

        return response
    except Exception as e:
        logger.error(f"❌ ADK Request failed: {e}")
        raise

@app.on_event("startup")
async def startup():
    logger.info("🚀 Starting ADK-Based LLM Reverse Proxy Server")
    if DEBUG_MODE:
        logger.debug("🐛 ADK DEBUG mode is ENABLED - detailed request logging active")
    try:
        config.validate()
        logger.info("✅ Configuration validated successfully")
        logger.info(f"🔧 Provider: {config.current_provider}")
        logger.info(f"🤖 Model: {config.current_model}")
        logger.info(f"🌐 Base URL: {config.current_base_url}")
        logger.info(f"🤖 Using ADK Orchestrator with intelligent processing pipeline")
        if config.current_provider == "openai":
            logger.info(f"🔑 OpenAI API key: {config.OPENAI_API_KEY[:10]}...")
        logger.info(f"🌐 Server: {config.HOST}:{config.PORT}")
    except ValueError as e:
        logger.error(f"❌ Configuration error: {e}")
        raise

@app.on_event("shutdown")
async def shutdown():
    await cleanup_http_client()
    logger.info("ADK Server shutdown complete")

@app.get("/")
async def root():
    return {
        "message": "ADK-Based LLM Streaming Proxy",
        "version": "2.0.0-streaming",
        "provider": config.current_provider,
        "model": config.current_model,
        "mode": "streaming_only",
        "endpoints": ["/v1/chat/completions", "/health", "/v1/models"],
        "features": ["OpenAI Compatible", "Streaming Only", "ADK Orchestrator", "Agent Pipeline", "Intelligent Processing"],
        "pipeline": "preprocessing_agent → streaming_provider → postprocessing_agent",
        "orchestrator": "llm_proxy_orchestrator"
    }

@app.get("/health")
async def health_check():
    return {
        "status": "healthy",
        "timestamp": time.time(),
        "provider": config.current_provider,
        "model": config.current_model,
        "server_type": "adk_based",
        "agents_status": "active",
        "config": {
            "openai_configured": bool(config.OPENAI_API_KEY) if config.current_provider == "openai" else None,
            "ollama_configured": config.current_provider == "ollama",
            "context_injection": config.ENABLE_CONTEXT_INJECTION,
            "analytics": config.ENABLE_RESPONSE_ANALYTICS
        }
    }

# Non-streaming processing removed - ADK server is streaming-focused only

async def stream_chat_completion_adk(request_data: Dict[str, Any]) -> AsyncGenerator[str, None]:
    """Handle streaming chat completion orchestrated by ADK orchestrator."""
    try:
        logger.debug("🤖 ADK Streaming with Orchestrator")

        # Step 1: Orchestrator manages the entire pipeline
        logger.debug("🤖 Starting ADK Orchestrator pipeline")
        orchestrator_input = {
            "request_data": request_data,
            "provider": config.current_provider,
            "model": config.current_model,
            "stream": True
        }

        # Execute orchestrator for the complete pipeline - but we'll extract steps manually for streaming
        logger.debug("🤖 Orchestrator managing complete pipeline")

        # Since orchestrator is designed for complete non-streaming pipeline,
        # we'll execute it step by step for streaming context

        # Step 1: Orchestrator handles preprocessing phase
        logger.debug("🤖 Orchestrator Step 1: Preprocessing through orchestrator")
        from src.application.services.orchestration_service import llm_proxy_orchestrator_preprocessing_only
        orchestrator_preprocessing_result = await llm_proxy_orchestrator_preprocessing_only(orchestrator_input, None)

        if orchestrator_preprocessing_result.get("status") != "success":
            error_chunk = {
                "error": {
                    "message": f"ADK Orchestrator preprocessing failed: {orchestrator_preprocessing_result.get('error', 'Unknown error')}",
                    "type": "adk_orchestrator_error"
                }
            }
            yield f"data: {json.dumps(error_chunk)}\n\n"
            yield "data: [DONE]\n\n"
            return

        logger.info("🤖 ADK Orchestrator preprocessing completed")

        # Step 2: Reasoning phase with streaming updates
        logger.debug("🤖 Orchestrator Step 2: Reasoning with streaming updates")
        from src.domain.services.reasoning_service_impl import reasoning_pipeline

        # Stream reasoning steps to the caller
        reasoning_request = orchestrator_preprocessing_result.get("processed_request", request_data.copy())
        async for reasoning_step in reasoning_pipeline(reasoning_request):
            yield reasoning_step

        # Apply reasoning to get the enhanced request
        from src.domain.services.reasoning_service_impl import apply_reasoning_to_request
        reasoning_result = apply_reasoning_to_request(reasoning_request)

        if reasoning_result.get("status") != "success":
            error_chunk = {
                "error": {
                    "message": f"ADK Reasoning failed: {reasoning_result.get('error', 'Unknown error')}",
                    "type": "adk_reasoning_error"
                }
            }
            yield f"data: {json.dumps(error_chunk)}\n\n"
            yield "data: [DONE]\n\n"
            return

        logger.info("🤖 ADK Orchestrator reasoning completed - request enhanced with intelligent context")

        # Extract processed data from reasoning result
        processed_request = reasoning_result.get("enhanced_request", request_data.copy())
        processed_request["stream"] = True

        # Get metadata from orchestrator and reasoning results
        metadata_result = {"metadata": orchestrator_preprocessing_result.get("metadata", {})}
        reasoning_metadata = reasoning_result.get("reasoning_metadata", {})

        logger.info("🤖 ADK Orchestrator guided streaming request preparation")

        # Filter reasoning and analysis content from messages before sending to LLM
        filtered_request = processed_request.copy()
        if "messages" in filtered_request:
            original_messages = filtered_request["messages"]
            filtered_messages = filter_messages_for_llm(original_messages)
            filtered_request["messages"] = filtered_messages
            logger.debug(f"🔧 Filtered messages for LLM: {len(original_messages)} → {len(filtered_messages)}")

        client = await get_http_client()
        full_content = ""

        logger.debug("🤖 ADK Streaming to provider")

        if config.current_provider == "openai":
            # Stream from OpenAI with ADK preprocessing (filtered request)
            async with client.stream(
                "POST",
                f"{config.current_base_url}/chat/completions",
                json=filtered_request
            ) as response:

                if response.status_code != 200:
                    error_chunk = {
                        "error": {
                            "message": f"ADK OpenAI API error: {response.status_code}",
                            "type": "adk_api_error"
                        }
                    }
                    yield f"data: {json.dumps(error_chunk)}\n\n"
                    yield "data: [DONE]\n\n"
                    return

                async for line in response.aiter_lines():
                    if line.startswith("data: "):
                        data = line[6:].strip()

                        if data == "[DONE]":
                            # Step 4: ADK Orchestrator Postprocessing Phase
                            if full_content:
                                logger.debug("🤖 Orchestrator Step 4: Postprocessing through orchestrator")

                                # Create postprocessing input for orchestrator
                                postprocessing_orchestrator_input = {
                                    "content": full_content,
                                    "request_metadata": metadata_result.get("metadata", {}),
                                    "provider": config.current_provider,
                                    "model": config.current_model,
                                    "orchestrator_context": "streaming_postprocessing"
                                }

                                # Execute postprocessing through orchestrator
                                # Since orchestrator already completed full pipeline, this is additional postprocessing
                                logger.debug("🤖 Orchestrator executing postprocessing phase")
                                postprocessing_orchestrator_result = await execute_postprocessing_agent(postprocessing_orchestrator_input)

                                logger.info(f"🤖 ADK Orchestrator postprocessing phase completed: {postprocessing_orchestrator_result.get('agent_name', 'unknown')}")

                                # Execute unified analysis combining request and response analysis
                                logger.debug("🤖 Orchestrator guided unified analysis")
                                unified_analysis_result = create_unified_analysis(reasoning_metadata, full_content)

                                # If we have unified analysis to add, send it as a final chunk with analysis markers
                                if unified_analysis_result.get("should_display", False):
                                    additional_content = unified_analysis_result.get("analysis_content", "")
                                    # Add clear analysis markers to separate from LLM conversation
                                    marked_content = add_analysis_markers(additional_content)
                                    final_chunk = {
                                        "id": f"adk-stream-{int(time.time())}",
                                        "object": "chat.completion.chunk",
                                        "created": int(time.time()),
                                        "model": processed_request.get("model", config.current_model),
                                        "choices": [
                                            {
                                                "index": 0,
                                                "delta": {
                                                    "content": marked_content
                                                },
                                                "finish_reason": None
                                            }
                                        ]
                                    }
                                    yield f"data: {json.dumps(final_chunk)}\n\n"

                                logger.info(f"🤖 ADK Orchestrator completed full streaming pipeline - Content length: {len(full_content)}, Unified analysis: {unified_analysis_result.get('status', 'unknown')}")

                            yield "data: [DONE]\n\n"
                            break

                        try:
                            chunk_data = json.loads(data)

                            # Extract content from chunk
                            if "choices" in chunk_data and len(chunk_data["choices"]) > 0:
                                choice = chunk_data["choices"][0]
                                if "delta" in choice and "content" in choice["delta"]:
                                    content = choice["delta"]["content"]
                                    full_content += content

                            yield f"data: {json.dumps(chunk_data)}\n\n"

                        except json.JSONDecodeError:
                            continue

        else:
            error_chunk = {
                "error": {
                    "message": f"ADK Unsupported provider for streaming: {config.current_provider}",
                    "type": "adk_provider_error"
                }
            }
            yield f"data: {json.dumps(error_chunk)}\n\n"
            yield "data: [DONE]\n\n"

    except Exception as e:
        logger.error(f"❌ Error in ADK streaming: {e}")
        error_chunk = {
            "error": {
                "message": f"ADK Streaming error: {str(e)}",
                "type": "adk_server_error"
            }
        }
        yield f"data: {json.dumps(error_chunk)}\n\n"
        yield "data: [DONE]\n\n"

@app.post("/v1/chat/completions")
async def chat_completions(request: ChatCompletionRequest):
    """OpenAI-compatible chat completions endpoint using ADK agents."""
    logger.debug("🤖 ADK chat_completions")
    try:
        request_dict = request.dict(exclude_none=True)
        logger.debug("🤖 ADK request_dict {}".format(request_dict))

        # ADK server is STREAMING ONLY - no non-streaming support
        logger.info("🤖 ADK STREAMING ONLY: Orchestrated intelligent processing")
        return StreamingResponse(
            stream_chat_completion_adk(request_dict),
            media_type="text/event-stream",
            headers={
                "Cache-Control": "no-cache",
                "Connection": "keep-alive",
                "Access-Control-Allow-Origin": "*"
            }
        )

    except Exception as e:
        logger.error(f"❌ Error in ADK chat completions: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/v1/models")
async def list_models():
    """List available models for the current provider."""
    if config.current_provider == "openai":
        return {
            "object": "list",
            "data": [
                {
                    "id": config.current_model,
                    "object": "model",
                    "created": int(time.time()),
                    "owned_by": "openai",
                    "permission": [],
                    "root": config.current_model,
                    "parent": None
                }
            ]
        }
    elif config.current_provider == "ollama":
        return {
            "object": "list",
            "data": [
                {
                    "id": config.current_model,
                    "object": "model",
                    "created": int(time.time()),
                    "owned_by": "ollama",
                    "permission": [],
                    "root": config.current_model,
                    "parent": None
                }
            ]
        }
    else:
        return {
            "object": "list",
            "data": []
        }

# CORS preflight handlers
@app.options("/v1/chat/completions")
async def chat_completions_options():
    """Handle CORS preflight requests for chat completions."""
    logger.debug("🤖 ADK chat_completions_options")
    return {}

@app.options("/v1/models")
async def models_options():
    """Handle CORS preflight requests for models."""
    return {}

@app.options("/")
async def root_options():
    """Handle CORS preflight requests for root."""
    return {}

@app.options("/health")
async def health_options():
    """Handle CORS preflight requests for health."""
    return {}

# Catch-all for debugging unknown requests
@app.api_route("/{full_path:path}", methods=["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"])
async def catch_all(request: Request, full_path: str):
    """Catch all unhandled requests for debugging."""
    logger.warning(f"🚫 ADK Unhandled request: {request.method} /{full_path}")
    if DEBUG_MODE:
        logger.debug(f"   Client: {request.client}")
        logger.debug(f"   Headers: {dict(request.headers)}")
        logger.debug(f"   Full URL: {request.url}")
    return {
        "error": "Not found",
        "path": full_path,
        "method": request.method,
        "server_type": "adk_based",
        "message": "This endpoint is not implemented. Available endpoints: /health, /v1/chat/completions, /v1/models"
    }

if __name__ == "__main__":
    uvicorn.run(
        "adk_server:app",
        host=config.HOST,
        port=config.PORT,
        reload=config.DEBUG,
        log_level="debug" if DEBUG_MODE else config.LOG_LEVEL.lower(),
        access_log=True
    )