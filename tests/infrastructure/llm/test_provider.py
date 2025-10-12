#!/usr/bin/env python3
"""
Test script to verify provider functionality
"""

import requests
import json
import time

def test_provider(provider, model, base_url="http://localhost:8000"):
    """Test a specific provider."""
    print(f"\n🧪 Testing {provider} with model {model}")
    print("=" * 50)
    
    # Test health endpoint
    try:
        health_response = requests.get(f"{base_url}/health")
        if health_response.status_code == 200:
            health_data = health_response.json()
            print(f"✅ Health check passed")
            print(f"   Provider: {health_data.get('provider')}")
            print(f"   Model: {health_data.get('model')}")
        else:
            print(f"❌ Health check failed: {health_response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Health check error: {e}")
        return False
    
    # Test models endpoint
    try:
        models_response = requests.get(f"{base_url}/v1/models")
        if models_response.status_code == 200:
            models_data = models_response.json()
            print(f"✅ Models endpoint working")
            print(f"   Available models: {[m['id'] for m in models_data.get('data', [])]}")
        else:
            print(f"❌ Models endpoint failed: {models_response.status_code}")
    except Exception as e:
        print(f"❌ Models endpoint error: {e}")
    
    # Test chat completion
    try:
        chat_data = {
            "model": model,
            "messages": [
                {"role": "user", "content": "Hello! Please respond with a short greeting."}
            ],
            "max_tokens": 50,
            "temperature": 0.7
        }
        
        chat_response = requests.post(
            f"{base_url}/v1/chat/completions",
            json=chat_data,
            headers={"Content-Type": "application/json"}
        )
        
        if chat_response.status_code == 200:
            chat_result = chat_response.json()
            print(f"✅ Chat completion successful")
            print(f"   Model used: {chat_result.get('model')}")
            print(f"   Response: {chat_result['choices'][0]['message']['content'][:100]}...")
        else:
            print(f"❌ Chat completion failed: {chat_response.status_code}")
            print(f"   Error: {chat_response.text}")
            return False
            
    except Exception as e:
        print(f"❌ Chat completion error: {e}")
        return False
    
    print(f"✅ All tests passed for {provider}")
    return True

def main():
    """Main test function."""
    print("🚀 Testing LLM Provider Functionality")
    print("=" * 60)
    
    # Test OpenAI
    print("\n1️⃣ Testing OpenAI provider...")
    success_openai = test_provider("openai", "gpt-4o-mini")
    
    # Test Ollama
    print("\n2️⃣ Testing Ollama provider...")
    success_ollama = test_provider("ollama", "mistral")
    
    # Summary
    print("\n" + "=" * 60)
    print("📊 Test Summary:")
    print(f"   OpenAI: {'✅ PASS' if success_openai else '❌ FAIL'}")
    print(f"   Ollama: {'✅ PASS' if success_ollama else '❌ FAIL'}")
    
    if success_openai and success_ollama:
        print("\n🎉 All providers working correctly!")
    else:
        print("\n⚠️  Some providers failed. Check the logs above.")

if __name__ == "__main__":
    main() 