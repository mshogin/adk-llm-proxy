#!/usr/bin/env python3
"""
Test direct MCP tool execution in the reasoning pipeline.
"""

import asyncio
import sys
import json
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.domain.services.reasoning_service_impl import apply_reasoning_to_request

async def test_direct_mcp_reasoning():
    """Test the original reasoning system to see if MCP tools work there."""
    print("🧪 Testing Direct MCP Reasoning (Original System)")
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
    print("🔄 Running original reasoning pipeline...")
    print()

    try:
        # Run the original reasoning pipeline directly
        result = await apply_reasoning_to_request(test_request)

        print(f"📊 Reasoning result status: {result.get('status')}")

        if result.get('status') == 'success':
            reasoning_metadata = result.get('reasoning_metadata', {})
            reasoning_insights = reasoning_metadata.get('reasoning_insights', {})

            print(f"🔍 Found {len(reasoning_insights)} reasoning insights")
            print(f"🛠️  Used {reasoning_metadata.get('mcp_tools_used', 0)} MCP tools")

            # Check for ticket data in insights
            ticket_data_found = False
            for insight_key, insight_value in reasoning_insights.items():
                if isinstance(insight_value, dict):
                    result_content = str(insight_value.get('result', ''))
                    if any(keyword in result_content.lower() for keyword in ['ticket', 'st-', 'assigned', 'сквозной']):
                        ticket_data_found = True
                        print(f"🎫 Found ticket data in {insight_key}:")
                        print(f"   {result_content[:300]}...")
                        break

            if ticket_data_found:
                print("✅ SUCCESS: Original reasoning system can retrieve tickets!")
                print("The issue is in the enhanced reasoning orchestrator display logic.")
            else:
                print("⚠️  No ticket data found in original reasoning insights")
                print("🔍 All insights:")
                for key, value in reasoning_insights.items():
                    print(f"   {key}: {str(value)[:100]}...")
        else:
            print(f"❌ Reasoning failed: {result.get('error')}")

    except Exception as e:
        print(f"❌ Error testing direct MCP reasoning: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    try:
        asyncio.run(test_direct_mcp_reasoning())
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted by user")