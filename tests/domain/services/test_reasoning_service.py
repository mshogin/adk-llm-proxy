#!/usr/bin/env python3
"""
Test the intelligent reasoning system with a YouTrack ticket request.
"""
import requests
import json
import time

def test_reasoning_system():
    """Test the intelligent reasoning system."""
    url = "http://localhost:8001/v1/chat/completions"

    headers = {
        "Content-Type": "application/json",
        "Authorization": "Bearer test-key"
    }

    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "I would like to see all tickets assigned to me"}
        ],
        "stream": True
    }

    print("🧠 Testing intelligent reasoning system...")
    print("📝 Request: 'I would like to see all tickets assigned to me'")
    print("🎯 Expected: Should detect intent as task_management and use YouTrack MCP tools")
    print("=" * 70)

    try:
        response = requests.post(url, headers=headers, json=payload, stream=True)
        response.raise_for_status()

        print("📡 Streaming response:")
        for line in response.iter_lines():
            if line:
                line_str = line.decode('utf-8')
                if line_str.startswith('data: '):
                    data_str = line_str[6:]  # Remove 'data: ' prefix
                    if data_str.strip() == '[DONE]':
                        break
                    try:
                        data = json.loads(data_str)
                        if 'choices' in data and len(data['choices']) > 0:
                            delta = data['choices'][0].get('delta', {})
                            content = delta.get('content', '')
                            if content:
                                print(content, end='', flush=True)
                    except json.JSONDecodeError:
                        pass

        print("\n" + "=" * 70)
        print("✅ Test completed")

    except requests.exceptions.RequestException as e:
        print(f"❌ Request failed: {e}")
    except Exception as e:
        print(f"❌ Test failed: {e}")

if __name__ == "__main__":
    test_reasoning_system()