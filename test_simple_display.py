#!/usr/bin/env python3
"""
Simple test to get tickets and display them directly.
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from get_my_assigned_tickets import get_my_assigned_tickets

async def test_simple_ticket_display():
    """Test simple ticket retrieval and display."""
    print("🎫 Simple Ticket Display Test")
    print("=" * 50)

    try:
        print("🔄 Retrieving tickets directly...")
        result = await get_my_assigned_tickets()

        if result:
            print("✅ SUCCESS: Tickets retrieved successfully!")
            print("🎫 This proves the MCP connection and ticket retrieval works")
        else:
            print("❌ Failed to retrieve tickets")

    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()

if __name__ == "__main__":
    try:
        asyncio.run(test_simple_ticket_display())
    except KeyboardInterrupt:
        print("\n🛑 Test interrupted by user")