#!/usr/bin/env python3
"""
Main entry point for the ADK-based LLM Reverse Proxy Server
STREAMING ONLY: This server exclusively supports streaming with intelligent ADK agent processing.
Run this script to start the agentic server with Google ADK agents and orchestrator.
"""

import os
import sys
import argparse
import warnings
import asyncio
from pathlib import Path

# Suppress asyncio warnings and errors during shutdown
warnings.filterwarnings("ignore", category=RuntimeWarning, message=".*was never awaited.*")
warnings.filterwarnings("ignore", category=RuntimeWarning, message=".*cancel scope.*")
warnings.filterwarnings("ignore", category=DeprecationWarning)

# Set environment variables to reduce asyncio noise
os.environ['PYTHONWARNINGS'] = 'ignore::DeprecationWarning,ignore::RuntimeWarning'
os.environ['PYTHONASYNCIODEBUG'] = '0'

# Reduce asyncio debug output
if hasattr(asyncio, 'set_event_loop_policy'):
    try:
        import uvloop
        asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())
    except ImportError:
        pass

# Suppress asyncio exception logging during shutdown
original_call_exception_handler = asyncio.AbstractEventLoop.call_exception_handler

def quiet_exception_handler(self, context):
    exception = context.get('exception')
    if exception:
        # Suppress common shutdown exceptions
        exception_str = str(exception).lower()
        if any(phrase in exception_str for phrase in [
            'cancel scope', 'generatorexit', 'cancelled', 'connection lost',
            'task was destroyed', 'async generator', 'stdio_client'
        ]):
            return
    original_call_exception_handler(self, context)

asyncio.AbstractEventLoop.call_exception_handler = quiet_exception_handler

# Add the current directory to the Python path so imports work
current_dir = Path(__file__).parent
sys.path.insert(0, str(current_dir))

def parse_arguments():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="ADK-based LLM Reverse Proxy Server (STREAMING ONLY) with Intelligent Agents",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  main.py -provider openai -model gpt-4o-mini
  main.py -provider ollama -model mistral
  main.py --provider openai --model gpt-4o-mini
        """
    )

    parser.add_argument(
        "-provider", "--provider",
        choices=["openai", "ollama", "deepseek"],
        default="openai",
        help="LLM provider to use (default: openai)"
    )

    parser.add_argument(
        "-model", "--model",
        default=None,
        help="Model to use (e.g., gpt-4o-mini, mistral, llama2)"
    )

    parser.add_argument(
        "--prompt",
        type=str,
        default=None,
        help="Run in prompt mode: execute a single prompt and exit (no server)"
    )

    return parser.parse_args()


async def run_prompt_mode(prompt: str, provider: str = "openai", model: str = None):
    """
    Run a single prompt inference without starting a server.

    Args:
        prompt: The user prompt to process
        provider: LLM provider
        model: Model name
    """
    print("🤖 Prompt Mode: Running inference...")
    print(f"🔧 Provider: {provider}")
    print(f"🤖 Model: {model or 'default'}")
    print("=" * 70)
    print()

    # Set environment variables
    os.environ["LLM_PROVIDER"] = provider
    if model:
        os.environ["LLM_MODEL"] = model

    try:
        # Import components
        from src.infrastructure.config.config import config
        from src.application.services.orchestration_service import llm_proxy_orchestrator_execute
        from src.infrastructure.mcp import mcp_registry

        # Initialize MCP servers from config
        print("🔌 Initializing MCP servers...")
        enabled_servers = config.get_enabled_mcp_servers()

        for server_config in enabled_servers:
            await mcp_registry.register_server(server_config)

        await mcp_registry.connect_all()
        connected_count = len(mcp_registry.get_connected_servers())
        print(f"✅ Connected to {connected_count} MCP server(s)")

        # Prepare request
        request_data = {
            "model": model or config.current_model,
            "messages": [
                {"role": "user", "content": prompt}
            ],
            "stream": False
        }

        print("🧠 Processing request with full pipeline (preprocessing → reasoning → LLM → postprocessing)...")
        print()

        # Step 1: Preprocessing
        print("📝 Step 1: Preprocessing...")
        from src.application.services.orchestration_service import llm_proxy_orchestrator_preprocessing_only
        input_data = {"request_data": request_data}
        preprocessing_result = await llm_proxy_orchestrator_preprocessing_only(input_data)

        if preprocessing_result.get("status") != "success":
            print(f"❌ Preprocessing failed: {preprocessing_result.get('error')}")
            return False

        preprocessed_request = preprocessing_result.get("processed_request", request_data)
        print("✅ Preprocessing complete")

        # Step 2: Reasoning (using the configured workflow)
        print("🧠 Step 2: Reasoning...")
        print("-" * 70)

        # Show input to reasoning
        print("\n📥 Input to Reasoning:")
        print(f"   Messages: {len(preprocessed_request.get('messages', []))} message(s)")
        for i, msg in enumerate(preprocessed_request.get('messages', []), 1):
            role = msg.get('role', 'unknown')
            content_preview = msg.get('content', '')[:100]
            print(f"   {i}. [{role}] {content_preview}...")

        # Use the reasoning pipeline directly to see all steps
        from src.domain.services.reasoning_service_impl import load_workflow_callback
        from src.domain.services.reasoning_service_impl import (
            analyze_request_intent,
            generate_reasoning_context,
            enhance_messages_with_reasoning,
            discover_reasoning_tools,
            execute_reasoning_tools,
            stream_reasoning_step
        )

        workflow_callback = load_workflow_callback()
        if not workflow_callback:
            print(f"\n❌ No workflow callback loaded")
            return False

        print("\n   🔄 Running reasoning workflow...")
        reasoning_steps = []

        # Run workflow and capture all steps
        try:
            async for chunk in workflow_callback(
                preprocessed_request,
                analyze_request_intent,
                generate_reasoning_context,
                enhance_messages_with_reasoning,
                discover_reasoning_tools,
                execute_reasoning_tools,
                stream_reasoning_step
            ):
                # Chunk is a dict with step information
                if isinstance(chunk, dict):
                    step = chunk.get('step', 'unknown')
                    data = chunk.get('data', {})

                    # Display the step
                    if step not in [s.get('step') for s in reasoning_steps]:
                        print(f"\n      ▶ {step}")
                        if isinstance(data, dict):
                            status = data.get('status', '')
                            if status:
                                print(f"        Status: {status[:150]}")
                            tools_executed = data.get('tools_executed', 0)
                            if tools_executed:
                                print(f"        Tools executed: {tools_executed}")

                    reasoning_steps.append(chunk)
        except Exception as e:
            print(f"\n❌ Workflow error: {e}")
            import traceback
            traceback.print_exc()

        # Get the result manually
        from src.domain.services.reasoning_service_impl import apply_reasoning_to_request
        reasoning_result = await apply_reasoning_to_request(preprocessed_request)

        if reasoning_result.get("status") != "success":
            print(f"\n❌ Reasoning failed: {reasoning_result.get('error')}")
            return False

        enhanced_request = reasoning_result.get("enhanced_request", preprocessed_request)

        # Show reasoning output
        print("\n📤 Output from Reasoning:")
        reasoning_metadata = reasoning_result.get("reasoning_metadata", {})
        if reasoning_metadata:
            print(f"   Intent: {reasoning_metadata.get('intent', 'N/A')}")
            print(f"   Plan steps: {len(reasoning_metadata.get('plan', {}).get('steps', []))}")
            print(f"   Context items: {len(reasoning_metadata.get('context', []))}")
            print(f"   Tools executed: {reasoning_metadata.get('tools_executed', 0)}")

            # Show context details if any
            context_items = reasoning_metadata.get('context', [])
            if context_items:
                print("\n   📦 Context collected:")
                for i, item in enumerate(context_items[:3], 1):  # Show first 3
                    if isinstance(item, dict):
                        source = item.get('source', 'unknown')
                        data_preview = str(item.get('data', ''))[:80]
                        print(f"      {i}. [{source}] {data_preview}...")
                if len(context_items) > 3:
                    print(f"      ... and {len(context_items) - 3} more")

        print("\n   Enhanced messages: {} message(s)".format(len(enhanced_request.get('messages', []))))
        print("✅ Reasoning complete")
        print("-" * 70)

        # Step 3: Forward to LLM
        print("\n🤖 Step 3: Forwarding to LLM...")
        print("-" * 70)
        from src.infrastructure.repositories.llm_proxy_repository import forward_to_openai, prepare_openai_request

        openai_prep_result = prepare_openai_request(enhanced_request)
        if openai_prep_result.get("status") != "success":
            print(f"❌ OpenAI prep failed: {openai_prep_result.get('error')}")
            return False

        openai_request = openai_prep_result["openai_request"]

        # Show what's being sent to LLM
        print("\n📤 Final Request to LLM:")
        print(f"   Model: {openai_request.get('model', 'N/A')}")
        print(f"   Temperature: {openai_request.get('temperature', 'N/A')}")
        print(f"   Messages: {len(openai_request.get('messages', []))} message(s)")
        print("\n   💬 Messages being sent:")
        for i, msg in enumerate(openai_request.get('messages', []), 1):
            role = msg.get('role', 'unknown')
            content = msg.get('content', '')
            print(f"\n   Message {i} [{role}]:")
            print("   " + "-" * 66)
            # Show full content with proper indentation
            for line in content.split('\n')[:20]:  # First 20 lines
                print(f"   {line}")
            if len(content.split('\n')) > 20:
                print(f"   ... ({len(content.split('\\n')) - 20} more lines)")
            print("   " + "-" * 66)

        forward_result = await forward_to_openai(openai_request)
        if forward_result.get("status") != "success":
            print(f"\n❌ LLM forward failed: {forward_result.get('error')}")
            return False

        llm_response = forward_result["response"]
        print("\n✅ LLM responded")
        print("-" * 70)

        # Step 4: Postprocessing
        print("✨ Step 4: Postprocessing...")
        from src.application.services.postprocessing_service import analyze_response_content, add_chat_content

        if llm_response.get("choices"):
            choice = llm_response["choices"][0]
            content = choice.get("message", {}).get("content", "")

            if content:
                # Apply postprocessing
                analyze_response_content(content)
                chat_content_result = add_chat_content(content, "analysis")

                # Add the chat content to the response
                if chat_content_result.get("status") == "success":
                    llm_response["choices"][0]["message"]["content"] = chat_content_result.get("content", content)

        print("✅ Postprocessing complete")
        result = {"status": "success", "response": llm_response}

        # Extract response
        if result and result.get("status") == "success" and "response" in result:
            response_data = result["response"]
            if "choices" in response_data and len(response_data["choices"]) > 0:
                content = response_data["choices"][0].get("message", {}).get("content", "")
                print("💬 Response:")
                print("=" * 70)
                print(content)
                print("=" * 70)
            else:
                print("❌ No choices in response")
        elif result and result.get("status") == "error":
            print(f"❌ Error: {result.get('error', 'Unknown error')}")
        else:
            print(f"❌ Unexpected result: {result}")

        # Cleanup
        print()
        print("🧹 Cleaning up...")
        try:
            await asyncio.wait_for(mcp_registry.shutdown(), timeout=2.0)
        except (asyncio.TimeoutError, asyncio.CancelledError):
            pass  # Expected during shutdown

        return True

    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


def main():
    """Main entry point - delegates to the existing working ADK server."""
    args = parse_arguments()

    # Check if running in prompt mode
    if args.prompt:
        result = asyncio.run(run_prompt_mode(
            prompt=args.prompt,
            provider=args.provider,
            model=args.model
        ))
        sys.exit(0 if result else 1)

    # Set environment variables for the server
    os.environ["LLM_PROVIDER"] = args.provider
    if args.model:
        os.environ["LLM_MODEL"] = args.model

    print("🚀 Starting ADK-based LLM Reverse Proxy Server with Intelligent Agents")
    print(f"🔧 Provider: {args.provider}")
    print(f"🤖 Model: {args.model or 'default'}")
    print("🤖 Using Google ADK agents for intelligent processing")
    print("📁 Agent pipeline: preprocessing → reasoning → LLM → postprocessing")
    print("✅ STREAMING ONLY: No non-streaming support")
    print("🎯 mshogin postprocessing: ENABLED")
    print("=" * 70)

    try:
        # Setup server using extracted function
        app, config = setup_server(
            provider=args.provider,
            model=args.model
        )

        print("✅ Google ADK available")

        import uvicorn

        # Start ADK server using your existing working configuration
        debug_mode = os.getenv('DEBUG', 'false').lower() in ('true', '1', 'yes', 'on')

        if config.DEBUG:
            print("🔄 Debug mode: Using auto-reload with ADK agents")
            uvicorn.run(
                "src.presentation.api.streaming_controller:app",
                host=config.HOST,
                port=config.PORT,
                reload=True,
                log_level="debug" if debug_mode else config.LOG_LEVEL.lower(),
                access_log=True
            )
        else:
            uvicorn.run(
                app,
                host=config.HOST,
                port=config.PORT,
                reload=False,
                log_level="debug" if debug_mode else config.LOG_LEVEL.lower(),
                access_log=True
            )

    except KeyboardInterrupt:
        print("\n👋 ADK Server stopped by user")
        # Let uvicorn handle the shutdown gracefully
        # The @app.on_event("shutdown") handler will clean up MCP connections
    except Exception as e:
        print(f"❌ Error starting ADK server: {e}")
        print("\n🔧 Troubleshooting:")
        print("   1. Make sure you're in the project root directory")
        print("   2. Install dependencies: pip install -r requirements.txt")
        print("   3. Install Google ADK: pip install google-adk")
        print("   4. Check your config.yaml file has the required API keys")
        if args.provider == "openai":
            print("   5. For OpenAI: Set OPENAI_API_KEY environment variable")
        elif args.provider == "ollama":
            print("   5. For Ollama: Make sure Ollama is running locally")
        sys.exit(1)


def setup_server(provider: str = "openai", model: str = None, port: int = None):
    """
    Setup and return the FastAPI app and server configuration.
    Can be used for both main() and integration tests.

    Args:
        provider: LLM provider to use
        model: Model to use (optional)
        port: Port to run server on (optional, uses config default if not provided)

    Returns:
        tuple: (app, config) - FastAPI app and config object
    """
    # Set environment variables for the server
    os.environ["LLM_PROVIDER"] = provider
    if model:
        os.environ["LLM_MODEL"] = model

    # Import your existing working server from DDD structure
    from src.presentation.api.streaming_controller import app
    from src.infrastructure.config.config import config

    # Check ADK availability
    try:
        from google.adk.agents import LlmAgent
    except ImportError:
        raise ImportError("Google ADK not found. Install with: pip install google-adk")

    # Override port if provided
    if port:
        config.PORT = port

    return app, config

if __name__ == "__main__":
    main()