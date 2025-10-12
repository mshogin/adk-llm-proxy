#!/usr/bin/env python3
"""
Test context injection to LLM after enhanced reasoning.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from src.domain.services.reasoning_service_impl import apply_reasoning_to_request

async def test_context_injection():
    """Test that enhanced reasoning injects ticket context into LLM request."""
    print("🧪 Testing Enhanced Reasoning Context Injection")
    print("=" * 60)

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
    print("🔄 Running apply_reasoning_to_request (non-streaming)...")
    print()

    try:
        # Run the reasoning service
        result = await apply_reasoning_to_request(test_request)

        print(f"📊 Reasoning result status: {result.get('status')}")

        if result.get('status') == 'success':
            enhanced_request = result.get('enhanced_request', {})
            reasoning_metadata = result.get('reasoning_metadata', {})

            print(f"🔍 Reasoning type: {reasoning_metadata.get('reasoning_type')}")
            print(f"🎫 Context items collected: {reasoning_metadata.get('context_items', 0)}")
            print(f"📝 Messages: {reasoning_metadata.get('original_message_count')} → {reasoning_metadata.get('enhanced_message_count')}")

            if enhanced_request.get('messages'):
                enhanced_messages = enhanced_request['messages']
                print(f"\n📋 Enhanced messages structure:")
                for i, msg in enumerate(enhanced_messages):
                    role = msg.get('role', 'unknown')
                    content_length = len(msg.get('content', ''))
                    print(f"   Message {i+1}: {role} ({content_length} chars)")

                # Check if system message contains ticket data
                system_messages = [msg for msg in enhanced_messages if msg.get('role') == 'system']
                if system_messages:
                    system_content = system_messages[0].get('content', '')

                    # Look for ticket indicators in system message
                    ticket_indicators = ['st-', 'ticket', 'assigned', 'youtrack', 'сквозной тарификатор']
                    found_indicators = []
                    for indicator in ticket_indicators:
                        if indicator.lower() in system_content.lower():
                            found_indicators.append(indicator)

                    if found_indicators:
                        print(f"\n✅ SUCCESS: Ticket context found in system message!")
                        print(f"🎫 Found indicators: {found_indicators}")
                        print(f"\n📋 System message preview (first 300 chars):")
                        print(f"   {system_content[:300]}...")

                        if len(system_content) > 1000:
                            print(f"\n🎫 Full system message contains {len(system_content)} characters of context")
                        else:
                            print(f"\n📄 Full system message:")
                            print(system_content)
                    else:
                        print(f"\n⚠️  No obvious ticket data found in system message")
                        print(f"📋 System message preview:")
                        print(f"   {system_content[:200]}...")
                else:
                    print(f"\n❌ No system message found in enhanced request")
            else:
                print(f"\n❌ No enhanced messages found")

        else:
            print(f"❌ Reasoning failed: {result.get('error')}")

    except Exception as e:
        print(f"❌ Error testing context injection: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    try:
        asyncio.run(test_context_injection())
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted by user")