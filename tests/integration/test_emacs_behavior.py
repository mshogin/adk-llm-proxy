#!/usr/bin/env python3
"""
Test script to reproduce emacs/gptel HTTP behavior
This script mimics various HTTP patterns that emacs might use to trigger "Invalid HTTP request received"
"""

import asyncio
import aiohttp
import socket
import time
import json
import sys
import threading
from concurrent.futures import ThreadPoolExecutor

async def test_normal_request():
    """Test a normal HTTP request (baseline)"""
    print("🔵 Testing normal HTTP request...")
    async with aiohttp.ClientSession() as session:
        data = {
            "model": "gpt-4o-mini",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": False
        }
        headers = {
            "Content-Type": "application/json",
            "Authorization": "Bearer dummy"
        }
        
        async with session.post("http://localhost:8000/v1/chat/completions", 
                               json=data, headers=headers) as response:
            result = await response.text()
            print(f"✅ Normal request: {response.status}")
            return response.status == 200

async def test_streaming_request():
    """Test streaming request (like gptel would use)"""
    print("🔵 Testing streaming HTTP request...")
    async with aiohttp.ClientSession() as session:
        data = {
            "model": "gpt-4o-mini",
            "messages": [{"role": "user", "content": "Count to 3"}],
            "stream": True
        }
        headers = {
            "Content-Type": "application/json",
            "Authorization": "Bearer dummy",
            "Accept": "text/event-stream",
            "User-Agent": "emacs/29.1"
        }
        
        async with session.post("http://localhost:8000/v1/chat/completions", 
                               json=data, headers=headers) as response:
            print(f"✅ Streaming request: {response.status}")
            # Read a few chunks then close
            count = 0
            async for line in response.content:
                count += 1
                if count > 5:
                    break
            return response.status == 200

async def test_connection_drop():
    """Test dropping connection mid-request (common emacs behavior)"""
    print("🔵 Testing connection drop...")
    connector = aiohttp.TCPConnector()
    timeout = aiohttp.ClientTimeout(total=1.0)  # Short timeout
    
    try:
        async with aiohttp.ClientSession(connector=connector, timeout=timeout) as session:
            data = {
                "model": "gpt-4o-mini", 
                "messages": [{"role": "user", "content": "Hello"}],
                "stream": True
            }
            headers = {
                "Content-Type": "application/json",
                "Authorization": "Bearer dummy"
            }
            
            async with session.post("http://localhost:8000/v1/chat/completions", 
                                   json=data, headers=headers) as response:
                # Start reading then drop
                await asyncio.sleep(0.1)
                # Connection will drop when session closes
                pass
    except asyncio.TimeoutError:
        print("⚠️ Connection dropped (timeout)")
        return True
    except Exception as e:
        print(f"⚠️ Connection dropped ({e})")
        return True

def test_malformed_http():
    """Test malformed HTTP request (raw socket)"""
    print("🔵 Testing malformed HTTP request...")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(('localhost', 8000))
        
        # Send malformed HTTP
        malformed_request = b"POST /v1/chat/completions HTTP/1.1\r\n"
        malformed_request += b"Host: localhost:8000\r\n"
        malformed_request += b"Content-Type: application/json\r\n"
        malformed_request += b"Authorization: Bearer dummy\r\n"
        # Missing Content-Length or invalid headers
        malformed_request += b"Invalid-Header\r\n"  # No colon
        malformed_request += b"\r\n"
        
        sock.send(malformed_request)
        time.sleep(0.1)
        sock.close()
        print("⚠️ Sent malformed HTTP request")
        return True
    except Exception as e:
        print(f"❌ Malformed HTTP test failed: {e}")
        return False

def test_incomplete_request():
    """Test incomplete HTTP request"""
    print("🔵 Testing incomplete HTTP request...")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(('localhost', 8000))
        
        # Send incomplete request
        incomplete_request = b"POST /v1/chat/completions HTTP/1.1\r\n"
        incomplete_request += b"Host: localhost:8000\r\n"
        incomplete_request += b"Content-Type: application/json\r\n"
        # Don't send body or close properly
        
        sock.send(incomplete_request)
        time.sleep(0.1)
        sock.close()
        print("⚠️ Sent incomplete HTTP request")
        return True
    except Exception as e:
        print(f"❌ Incomplete HTTP test failed: {e}")
        return False

def test_keep_alive_issues():
    """Test HTTP keep-alive connection issues"""
    print("🔵 Testing keep-alive connection issues...")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(('localhost', 8000))
        
        # Send request with keep-alive
        request = b"GET /health HTTP/1.1\r\n"
        request += b"Host: localhost:8000\r\n"
        request += b"Connection: keep-alive\r\n"
        request += b"\r\n"
        
        sock.send(request)
        
        # Read response
        response = sock.recv(1024)
        print(f"📥 Received: {len(response)} bytes")
        
        # Send another request on same connection
        request2 = b"GET /v1/models HTTP/1.1\r\n"
        request2 += b"Host: localhost:8000\r\n"
        request2 += b"Connection: close\r\n"
        request2 += b"\r\n"
        
        sock.send(request2)
        time.sleep(0.1)
        sock.close()
        print("✅ Keep-alive test completed")
        return True
    except Exception as e:
        print(f"❌ Keep-alive test failed: {e}")
        return False

async def test_concurrent_connections():
    """Test multiple concurrent connections (emacs might do this)"""
    print("🔵 Testing concurrent connections...")
    
    async def make_request(session, i):
        try:
            data = {
                "model": "gpt-4o-mini",
                "messages": [{"role": "user", "content": f"Request {i}"}],
                "stream": True,
                "max_tokens": 10
            }
            headers = {
                "Content-Type": "application/json",
                "Authorization": "Bearer dummy",
                "User-Agent": f"emacs-test-{i}"
            }
            
            async with session.post("http://localhost:8000/v1/chat/completions", 
                                   json=data, headers=headers) as response:
                # Read just a bit then close
                async for chunk in response.content:
                    break
                return response.status
        except Exception as e:
            print(f"⚠️ Concurrent request {i} failed: {e}")
            return 0
    
    connector = aiohttp.TCPConnector(limit=10)
    async with aiohttp.ClientSession(connector=connector) as session:
        # Make 5 concurrent requests
        tasks = [make_request(session, i) for i in range(5)]
        results = await asyncio.gather(*tasks, return_exceptions=True)
        success_count = sum(1 for r in results if r == 200)
        print(f"✅ Concurrent connections: {success_count}/5 successful")
        return success_count > 0

def test_http_1_0():
    """Test HTTP/1.0 request (emacs might use this)"""
    print("🔵 Testing HTTP/1.0 request...")
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(('localhost', 8000))
        
        # HTTP/1.0 request
        request = b"GET /health HTTP/1.0\r\n"
        request += b"Host: localhost:8000\r\n"
        request += b"\r\n"
        
        sock.send(request)
        response = sock.recv(1024)
        sock.close()
        
        print(f"✅ HTTP/1.0 request: received {len(response)} bytes")
        return len(response) > 0
    except Exception as e:
        print(f"❌ HTTP/1.0 test failed: {e}")
        return False

async def main():
    """Run all tests to reproduce the invalid HTTP request issue"""
    print("🚀 Starting emacs/gptel behavior reproduction tests")
    print("=" * 60)
    
    tests = [
        ("Normal Request", test_normal_request()),
        ("Streaming Request", test_streaming_request()),
        ("Connection Drop", test_connection_drop()),
        ("Concurrent Connections", test_concurrent_connections()),
    ]
    
    # Run async tests
    for name, test_coro in tests:
        try:
            result = await test_coro
            print(f"{'✅' if result else '❌'} {name}: {'PASSED' if result else 'FAILED'}")
        except Exception as e:
            print(f"❌ {name}: FAILED with {e}")
        print()
        await asyncio.sleep(0.5)  # Small delay between tests
    
    # Run sync tests
    sync_tests = [
        ("Malformed HTTP", test_malformed_http),
        ("Incomplete Request", test_incomplete_request), 
        ("Keep-alive Issues", test_keep_alive_issues),
        ("HTTP/1.0 Request", test_http_1_0),
    ]
    
    for name, test_func in sync_tests:
        try:
            result = test_func()
            print(f"{'✅' if result else '❌'} {name}: {'PASSED' if result else 'FAILED'}")
        except Exception as e:
            print(f"❌ {name}: FAILED with {e}")
        print()
        time.sleep(0.5)
    
    print("=" * 60)
    print("🏁 All tests completed. Check server logs for 'Invalid HTTP request received' warnings.")
    print("💡 The invalid requests are likely from the malformed/incomplete/dropped connection tests.")

if __name__ == "__main__":
    asyncio.run(main()) 