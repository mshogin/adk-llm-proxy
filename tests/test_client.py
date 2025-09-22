#!/usr/bin/env python3
"""
Test client for the ADK-based LLM Reverse Proxy Server
Demonstrates both streaming and non-streaming functionality
"""

import asyncio
import json
import time
import requests
from typing import Dict, Any, List

SERVER_URL = "http://localhost:8000"

def test_health_check():
    """Test the health check endpoint."""
    print("🏥 Testing Health Check...")
    try:
        response = requests.get(f"{SERVER_URL}/health")
        if response.status_code == 200:
            health_data = response.json()
            print(f"✅ Server is healthy")
            print(f"   OpenAI configured: {health_data['config']['openai_configured']}")
            print(f"   Context injection: {health_data['config']['context_injection']}")
            print(f"   Analytics: {health_data['config']['analytics']}")
            return True
        else:
            print(f"❌ Health check failed: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ Could not connect to server: {e}")
        return False

def test_non_streaming_chat():
    """Test non-streaming chat completion."""
    print("\n💬 Testing Non-Streaming Chat...")

    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "Tell me a short joke about programming."}
        ],
        "stream": False,
        "temperature": 0.7,
        "max_tokens": 150
    }

    try:
        start_time = time.time()
        response = requests.post(
            f"{SERVER_URL}/v1/chat/completions",
            json=payload,
            headers={"Content-Type": "application/json"}
        )

        if response.status_code == 200:
            data = response.json()
            elapsed = time.time() - start_time

            message = data['choices'][0]['message']['content']
            usage = data.get('usage', {})

            print(f"✅ Non-streaming response received in {elapsed:.2f}s")
            print(f"   Model: {data.get('model', 'unknown')}")
            print(f"   Tokens: {usage.get('total_tokens', 'unknown')}")
            print(f"   Response: {message[:100]}...")
            return True
        else:
            print(f"❌ Request failed: {response.status_code}")
            print(f"   Error: {response.text}")
            return False

    except requests.exceptions.RequestException as e:
        print(f"❌ Request failed: {e}")
        return False

def test_streaming_chat():
    """Test streaming chat completion."""
    print("\n🌊 Testing Streaming Chat...")

    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "Count from 1 to 5, explaining each number."}
        ],
        "stream": True,
        "temperature": 0.7,
        "max_tokens": 200
    }

    try:
        start_time = time.time()
        chunks_received = 0
        content_length = 0

        with requests.post(
            f"{SERVER_URL}/v1/chat/completions",
            json=payload,
            headers={"Content-Type": "application/json"},
            stream=True
        ) as response:

            if response.status_code != 200:
                print(f"❌ Streaming request failed: {response.status_code}")
                return False

            print("✅ Streaming started...")
            print("   Content: ", end="", flush=True)

            for line in response.iter_lines():
                if line:
                    line_text = line.decode('utf-8')

                    if line_text.startswith('data: '):
                        data_part = line_text[6:]

                        if data_part == '[DONE]':
                            break

                        try:
                            chunk_data = json.loads(data_part)
                            chunks_received += 1

                            if 'choices' in chunk_data and len(chunk_data['choices']) > 0:
                                choice = chunk_data['choices'][0]
                                if 'delta' in choice and 'content' in choice['delta']:
                                    content = choice['delta']['content']
                                    print(content, end="", flush=True)
                                    content_length += len(content)

                        except json.JSONDecodeError:
                            pass  # Skip malformed chunks

            elapsed = time.time() - start_time
            print(f"\n✅ 3Streaming completed in {elapsed:.2f}s")
            print(f"   Chunks received: {chunks_received}")
            print(f"   Total content length: {content_length} characters")
            return True

    except requests.exceptions.RequestException as e:
        print(f"❌ Streaming request failed: {e}")
        return False

def test_context_injection():
    """Test context injection functionality."""
    print("\n🧠 Testing Context Injection...")

    # Test without system message (should add one)
    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "user", "content": "What are you?"}
        ],
        "stream": False,
        "max_tokens": 100
    }

    try:
        response = requests.post(
            f"{SERVER_URL}/v1/chat/completions",
            json=payload,
            headers={"Content-Type": "application/json"}
        )

        if response.status_code == 200:
            data = response.json()
            message = data['choices'][0]['message']['content'].lower()

            # Check if response indicates awareness of being an AI assistant
            if any(word in message for word in ['assistant', 'ai', 'help', 'helpful']):
                print("✅ Context injection appears to be working")
                print(f"   Response contains assistant-related keywords")
                return True
            else:
                print("⚠️  Context injection may not be working as expected")
                print(f"   Response: {message[:100]}...")
                return False
        else:
            print(f"❌ Context injection test failed: {response.status_code}")
            return False

    except requests.exceptions.RequestException as e:
        print(f"❌ Context injection test failed: {e}")
        return False

def test_openai_client_compatibility():
    """Test compatibility with OpenAI Python client."""
    print("\n🔗 Testing OpenAI Client Compatibility...")

    try:
        from openai import OpenAI

        # Create client pointing to local server
        client = OpenAI(
            api_key="dummy",  # Not used but required
            base_url=f"{SERVER_URL}/v1"
        )

        # Test non-streaming
        response = client.chat.completions.create(
            model="gpt-4o-mini",
            messages=[{"role": "user", "content": "Say hello!"}],
            max_tokens=50
        )

        if response.choices and response.choices[0].message:
            print("✅ OpenAI client non-streaming works")
            print(f"   Response: {response.choices[0].message.content[:50]}...")

            # Test streaming
            stream = client.chat.completions.create(
                model="gpt-4o-mini",
                messages=[{"role": "user", "content": "Count to 3"}],
                max_tokens=50,
                stream=True
            )

            chunks = 0
            for chunk in stream:
                if chunk.choices and chunk.choices[0].delta.content:
                    chunks += 1
                if chunks > 5:  # Don't process entire stream
                    break

            if chunks > 0:
                print("✅ OpenAI client streaming works")
                print(f"   Received {chunks} chunks")
                return True
            else:
                print("❌ OpenAI client streaming failed")
                return False
        else:
            print("❌ OpenAI client non-streaming failed")
            return False

    except ImportError:
        print("⚠️  OpenAI client not installed (pip install openai)")
        return True  # Not a failure, just not installed
    except Exception as e:
        print(f"❌ OpenAI client test failed: {e}")
        return False

def test_models_endpoint():
    """Test the models endpoint."""
    print("\n📋 Testing Models Endpoint...")

    try:
        response = requests.get(f"{SERVER_URL}/v1/models")
        if response.status_code == 200:
            data = response.json()
            models = data.get('data', [])
            print(f"✅ Models endpoint works")
            print(f"   Available models: {len(models)}")
            for model in models:
                print(f"   - {model.get('id', 'unknown')}")
            return True
        else:
            print(f"❌ Models endpoint failed: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ Models endpoint failed: {e}")
        return False

def main():
    """Run all tests."""
    print("🧪 ADK-based LLM Reverse Proxy Server Test Suite")
    print("=" * 60)

    tests = [
        ("Health Check", test_health_check),
        ("Models Endpoint", test_models_endpoint),
        ("Non-Streaming Chat", test_non_streaming_chat),
        ("Streaming Chat", test_streaming_chat),
        ("Context Injection", test_context_injection),
        ("OpenAI Client Compatibility", test_openai_client_compatibility),
    ]

    results = []

    for test_name, test_func in tests:
        try:
            result = test_func()
            results.append((test_name, result))
            if not result:
                print(f"⚠️  {test_name} failed, but continuing...")
        except Exception as e:
            print(f"❌ {test_name} crashed: {e}")
            results.append((test_name, False))

    # Summary
    print("\n" + "=" * 60)
    print("📊 Test Results Summary:")

    passed = sum(1 for _, result in results if result)
    total = len(results)

    for test_name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"   {status} {test_name}")

    print(f"\n🎯 Overall: {passed}/{total} tests passed")

    if passed == total:
        print("🎉 All tests passed! Your ADK-based LLM reverse proxy is working correctly.")
        print("\n📝 Next steps:")
        print("   1. Configure your emacs/gptel to use http://localhost:8000")
        print("   2. Set up your .env file with proper API keys")
        print("   3. Customize preprocessing and postprocessing as needed")
    else:
        print("⚠️  Some tests failed. Check the output above for details.")
        print("💡 Make sure:")
        print("   - The server is running on http://localhost:8000")
        print("   - Your OPENAI_API_KEY is set correctly")
        print("   - All dependencies are installed (pip install -r requirements.txt)")

if __name__ == "__main__":
    main()
