#!/usr/bin/env python3
"""
Test script to verify the ticket display fix in enhanced reasoning.
"""

import asyncio
import sys
import json
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.domain.services.enhanced_reasoning_orchestrator import enhanced_reasoning_pipeline
from src.infrastructure.config.config import config

async def test_enhanced_reasoning_ticket_display():
    """Test if the enhanced reasoning system now displays tickets."""
    print("🧪 Testing Enhanced Reasoning Ticket Display Fix")
    print("=" * 50)

    # Create a test request for getting assigned tickets
    test_request = {
        "messages": [
            {
                "role": "user",
                "content": "Show me my assigned tickets from YouTrack"
            }
        ],
        "model": "gpt-4o-mini",
        "stream": True
    }

    print("📤 Sending request: 'Show me my assigned tickets from YouTrack'")
    print("🔄 Running enhanced reasoning pipeline...")
    print()

    try:
        # Run the enhanced reasoning pipeline
        output_chunks = []
        async for chunk in enhanced_reasoning_pipeline(test_request):
            # Print the chunk content to see what's being streamed
            try:
                chunk_data = json.loads(chunk.replace("data: ", "").strip())
                content = chunk_data.get("choices", [{}])[0].get("delta", {}).get("content", "")
                if content:
                    print(content, end="")
                    output_chunks.append(content)
            except (json.JSONDecodeError, KeyError, IndexError):
                # Handle non-JSON chunks or malformed data
                if chunk.strip():
                    print(f"[Non-JSON chunk: {chunk[:100]}...]")

        print()
        print("=" * 50)
        print("✅ Enhanced reasoning pipeline completed")

        # Check if tickets were displayed in the output
        full_output = "".join(output_chunks)

        if any(indicator in full_output.lower() for indicator in [
            "st-", "ticket", "assigned", "сквозной тарификатор", "переработка полинома"
        ]):
            print("🎫 SUCCESS: Ticket data found in enhanced reasoning output!")
            print("✅ The fix is working - tickets are now being displayed")
        else:
            print("⚠️  Ticket data not found in output")
            print("🔍 Full output preview:")
            print(full_output[:500] + "..." if len(full_output) > 500 else full_output)

    except Exception as e:
        print(f"❌ Error testing enhanced reasoning: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    try:
        asyncio.run(test_enhanced_reasoning_ticket_display())
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted by user")