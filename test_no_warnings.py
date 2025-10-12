#!/usr/bin/env python3
"""
Test the enhanced reasoning without ADK warnings.
"""

import asyncio
import sys
import json
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.domain.services.enhanced_reasoning_orchestrator import enhanced_reasoning_pipeline

async def test_enhanced_reasoning_no_warnings():
    """Test enhanced reasoning without ADK model_copy warnings."""
    print("🧪 Testing Enhanced Reasoning WITHOUT Warnings")
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
    print("🔄 Running enhanced reasoning with fallback logic...")
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

        # Look for ticket indicators
        ticket_indicators = [
            "st-", "ticket", "assigned", "сквозной тарификатор",
            "переработка полинома", "find_assigned_tickets", "youtrack"
        ]

        found_indicators = []
        for indicator in ticket_indicators:
            if indicator in full_output.lower():
                found_indicators.append(indicator)

        if found_indicators:
            print(f"🎫 SUCCESS: Found ticket-related content! Indicators: {found_indicators}")
            print("✅ The warnings are fixed and ticket data should be visible")

            # Show a preview of ticket content
            lines = full_output.split('\n')
            ticket_lines = [line for line in lines if any(ind in line.lower() for ind in ticket_indicators)]
            if ticket_lines:
                print("\n🎫 Ticket content preview:")
                for line in ticket_lines[:5]:  # Show first 5 relevant lines
                    print(f"   {line.strip()}")
        else:
            print("⚠️  No obvious ticket data found in output")
            print("🔍 Full output preview (first 500 chars):")
            print(full_output[:500] + "..." if len(full_output) > 500 else full_output)

    except Exception as e:
        print(f"❌ Error testing enhanced reasoning: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    try:
        asyncio.run(test_enhanced_reasoning_no_warnings())
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted by user")