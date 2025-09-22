#!/usr/bin/env python3
"""
Main entry point for the ADK-based LLM Reverse Proxy Server
STREAMING ONLY: This server exclusively supports streaming with intelligent ADK agent processing.
Run this script to start the agentic server with Google ADK agents and orchestrator.
"""

import os
import sys
import argparse
from pathlib import Path

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

    return parser.parse_args()

def main():
    """Main entry point - delegates to the existing working ADK server."""
    args = parse_arguments()

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
        # Import your existing working server from DDD structure
        from src.presentation.api.streaming_controller import app
        import uvicorn
        from src.infrastructure.config.config import config

        # Check ADK availability
        try:
            from google.adk.agents import LlmAgent
            print("✅ Google ADK available")
        except ImportError:
            print("❌ Google ADK not found. Install with: pip install google-adk")
            sys.exit(1)

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
        print("\\n👋 ADK Server stopped by user")
    except Exception as e:
        print(f"❌ Error starting ADK server: {e}")
        print("\\n🔧 Troubleshooting:")
        print("   1. Make sure you're in the project root directory")
        print("   2. Install dependencies: pip install -r requirements.txt")
        print("   3. Install Google ADK: pip install google-adk")
        print("   4. Check your config.yaml file has the required API keys")
        if args.provider == "openai":
            print("   5. For OpenAI: Set OPENAI_API_KEY environment variable")
        elif args.provider == "ollama":
            print("   5. For Ollama: Make sure Ollama is running locally")
        sys.exit(1)

if __name__ == "__main__":
    main()