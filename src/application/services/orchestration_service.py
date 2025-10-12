import logging
from typing import Dict, List, Any
# Import functions directly instead of agents since new ADK agents don't have execute method
from src.application.services.preprocessing_service import inject_context, validate_request, extract_request_metadata
from src.domain.services.reasoning_service_impl import apply_reasoning_to_request
from src.infrastructure.repositories.llm_proxy_repository import forward_to_openai, prepare_openai_request, extract_response_metadata
from src.application.services.postprocessing_service import analyze_response_content, filter_response, log_interaction, add_chat_content

logger = logging.getLogger(__name__)

# Create a streaming-specific orchestrator for preprocessing and reasoning
async def llm_proxy_orchestrator_preprocessing_and_reasoning(input_data: Dict[str, Any], context=None) -> Dict[str, Any]:
    """
    Execute preprocessing and reasoning phases for streaming:
    1. Preprocessing: Validates, analyzes, and enhances incoming requests
    2. Reasoning: Adds intelligent context and reasoning to the request
    """
    try:
        logger.debug("🤖 Orchestrator starting preprocessing + reasoning pipeline")

        request_data = input_data.get("request_data", {})

        # Step 1: Preprocessing
        logger.debug("🤖 Step 1: Preprocessing (streaming mode)")

        # Validate request
        validation_result = validate_request(request_data)
        if not validation_result.get("is_valid", False):
            return {"status": "error", "error": f"Invalid request: {validation_result.get('errors', [])}"}

        # Extract metadata
        metadata_result = extract_request_metadata(request_data)

        # Inject context
        messages = request_data.get("messages", [])
        context_result = inject_context(messages)

        if context_result.get("status") == "success":
            processed_messages = context_result.get("messages", messages)
        else:
            processed_messages = messages

        # Build preprocessed request
        preprocessed_request = request_data.copy()
        preprocessed_request["messages"] = processed_messages

        # Step 2: Reasoning
        logger.debug("🤖 Step 2: Reasoning (streaming mode)")

        reasoning_result = await apply_reasoning_to_request(preprocessed_request)
        if reasoning_result.get("status") != "success":
            return {"status": "error", "error": f"Reasoning failed: {reasoning_result.get('error')}"}

        enhanced_request = reasoning_result["enhanced_request"]
        reasoning_metadata = reasoning_result["reasoning_metadata"]

        logger.debug("🤖 Orchestrator preprocessing + reasoning pipeline completed")
        return {
            "status": "success",
            "processed_request": enhanced_request,
            "metadata": metadata_result.get("metadata", {}),
            "reasoning_metadata": reasoning_metadata,
            "mode": "preprocessing_and_reasoning"
        }

    except Exception as e:
        logger.error(f"❌ Orchestrator preprocessing + reasoning error: {e}")
        return {
            "status": "error",
            "error": str(e)
        }

# Create a streaming-specific orchestrator for preprocessing only
async def llm_proxy_orchestrator_preprocessing_only(input_data: Dict[str, Any], context=None) -> Dict[str, Any]:
    """
    Execute only the preprocessing phase for streaming:
    1. Preprocessing: Validates, analyzes, and enhances incoming requests
    """
    try:
        logger.debug("🤖 Orchestrator starting preprocessing-only pipeline")

        request_data = input_data.get("request_data", {})

        # Step 1: Preprocessing only
        logger.debug("🤖 Step 1: Preprocessing (streaming mode)")

        # Validate request
        validation_result = validate_request(request_data)
        if not validation_result.get("is_valid", False):
            return {"status": "error", "error": f"Invalid request: {validation_result.get('errors', [])}"}

        # Extract metadata
        metadata_result = extract_request_metadata(request_data)

        # Inject context
        messages = request_data.get("messages", [])
        context_result = inject_context(messages)

        if context_result.get("status") == "success":
            processed_messages = context_result.get("messages", messages)
        else:
            processed_messages = messages

        # Build processed request
        processed_request = request_data.copy()
        processed_request["messages"] = processed_messages

        logger.debug("🤖 Orchestrator preprocessing-only pipeline completed")
        return {
            "status": "success",
            "processed_request": processed_request,
            "metadata": metadata_result.get("metadata", {}),
            "mode": "preprocessing_only"
        }

    except Exception as e:
        logger.error(f"❌ Orchestrator preprocessing-only error: {e}")
        return {
            "status": "error",
            "error": str(e)
        }

# Create a simple orchestrator function that uses functions directly
async def llm_proxy_orchestrator_execute(input_data: Dict[str, Any], context=None) -> Dict[str, Any]:
    """
    Execute the complete request/response pipeline using direct function calls:
    1. Preprocessing: Validates, analyzes, and enhances incoming requests
    2. Proxy: Forwards requests to OpenAI API and handles responses
    3. Postprocessing: Analyzes, filters, and enhances responses before delivery
    """
    try:
        logger.debug("🤖 Orchestrator starting pipeline")

        request_data = input_data.get("request_data", {})

        # Step 1: Preprocessing
        logger.debug("🤖 Step 1: Preprocessing")

        # Validate request
        validation_result = validate_request(request_data)
        if not validation_result.get("is_valid", False):
            return {"status": "error", "error": f"Invalid request: {validation_result.get('errors', [])}"}

        # Extract metadata
        metadata_result = extract_request_metadata(request_data)

        # Inject context
        messages = request_data.get("messages", [])
        context_result = inject_context(messages)

        if context_result.get("status") == "success":
            processed_messages = context_result.get("messages", messages)
        else:
            processed_messages = messages

        # Build processed request
        processed_request = request_data.copy()
        processed_request["messages"] = processed_messages

        # Step 2: Proxy
        logger.debug("🤖 Step 2: Proxy")

        # Prepare OpenAI request
        openai_prep_result = prepare_openai_request(processed_request)
        if openai_prep_result.get("status") != "success":
            return {"status": "error", "error": openai_prep_result.get("error", "Failed to prepare OpenAI request")}

        openai_request = openai_prep_result["openai_request"]

        # Forward to OpenAI
        forward_result = await forward_to_openai(openai_request)
        if forward_result.get("status") != "success":
            return {"status": "error", "error": forward_result.get("error", "Failed to forward to OpenAI")}

        openai_response = forward_result["response"]

        # Step 3: Postprocessing
        logger.debug("🤖 Step 3: Postprocessing")

        if not openai_response.get("choices"):
            return {"status": "success", "response": openai_response}

        # Get the response content
        choice = openai_response["choices"][0]
        content = choice.get("message", {}).get("content", "")

        if content:
            # Apply postprocessing
            analysis_result = analyze_response_content(content)
            filter_result = filter_response(content)

            # Add chat content (including "mshogin")
            chat_content_result = add_chat_content(content, "analysis")

            # Log the interaction if enabled
            if hasattr(input_data, 'ENABLE_RESPONSE_ANALYTICS') and input_data.ENABLE_RESPONSE_ANALYTICS:
                response_metadata = {
                    "response_id": openai_response.get("id"),
                    "model_used": openai_response.get("model"),
                    "finish_reason": choice.get("finish_reason"),
                    "total_tokens": openai_response.get("usage", {}).get("total_tokens", 0),
                    "prompt_tokens": openai_response.get("usage", {}).get("prompt_tokens", 0),
                    "completion_tokens": openai_response.get("usage", {}).get("completion_tokens", 0)
                }

                log_interaction(
                    metadata_result.get("metadata", {}),
                    response_metadata,
                    analysis_result.get("analysis", {})
                )

            # Prepare the modified content
            final_content = content

            # Apply content filtering if needed
            if filter_result.get("content_modified", False):
                final_content = filter_result["filtered_content"]

            # Add chat content if available
            if chat_content_result.get("should_add", False):
                final_content += chat_content_result.get("chat_content", "")

            # Create processed response with final content
            processed_response = openai_response.copy()
            processed_response["choices"][0]["message"]["content"] = final_content

            logger.debug("🤖 Orchestrator pipeline completed")
            return {"status": "success", "response": processed_response}

        logger.debug("🤖 Orchestrator pipeline completed")
        return {"status": "success", "response": openai_response}

    except Exception as e:
        logger.error(f"❌ Orchestrator error: {e}")
        return {
            "status": "error",
            "error": str(e)
        }

# Create a simple orchestrator class
class LLMProxyOrchestrator:
    def __init__(self):
        self.name = "llm_proxy_orchestrator"
        self.description = """Main orchestrator for the LLM reverse proxy server.

        Coordinates the complete request/response pipeline:
        1. Preprocessing: Validates, analyzes, and enhances incoming requests
        2. Proxy: Forwards requests to OpenAI API and handles responses
        3. Postprocessing: Analyzes, filters, and enhances responses before delivery

        Maintains streaming capabilities while adding intelligent preprocessing and postprocessing."""

    async def execute(self, input_data: Dict[str, Any], context=None) -> Dict[str, Any]:
        return await llm_proxy_orchestrator_execute(input_data, context)

llm_proxy_orchestrator = LLMProxyOrchestrator()

# This is the root agent that the server will use
root_agent = llm_proxy_orchestrator 