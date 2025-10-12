#!/usr/bin/env python3
"""
Quick test for the new 'get_my_recent_commits' GitLab MCP tool.
"""

import asyncio
import httpx
import json
import sys
import threading
import time


class ServerManager:
    """Manages the FastAPI server lifecycle for testing."""

    def __init__(self, port: int = 8002):
        self.port = port
        self.server_thread = None
        self.server = None

    def start_server(self):
        """Start the server in a background thread."""
        import uvicorn
        from main import setup_server

        app, config = setup_server(
            provider="openai",
            model="gpt-4o-mini",
            port=self.port
        )

        uvicorn_config = uvicorn.Config(
            app,
            host="127.0.0.1",
            port=self.port,
            log_level="error",
            access_log=False
        )
        self.server = uvicorn.Server(uvicorn_config)

        def run_server():
            asyncio.run(self.server.serve())

        self.server_thread = threading.Thread(target=run_server, daemon=True)
        self.server_thread.start()

    async def wait_for_ready(self, timeout: float = 30.0) -> bool:
        """Wait for server to be ready."""
        start_time = time.time()
        url = f"http://127.0.0.1:{self.port}/health"

        print(f"⏳ Waiting for server on port {self.port}...")

        async with httpx.AsyncClient() as client:
            while time.time() - start_time < timeout:
                try:
                    response = await client.get(url, timeout=1.0)
                    if response.status_code == 200:
                        print(f"✅ Server ready after {time.time() - start_time:.1f}s")
                        return True
                except (httpx.ConnectError, httpx.TimeoutException):
                    await asyncio.sleep(0.5)
                    continue

        return False

    async def shutdown(self):
        """Shutdown the server."""
        if self.server:
            self.server.should_exit = True
            await asyncio.sleep(1.0)


async def test_gitlab_commits():
    """Test GitLab 'my commits' query."""

    server = ServerManager(port=8002)
    server.start_server()

    if not await server.wait_for_ready(timeout=30.0):
        print("❌ Failed to start server")
        return False

    url = f"http://127.0.0.1:{server.port}/v1/chat/completions"

    request_payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {
                "role": "user",
                "content": "Please collect statistics by my latest 5 commits in gitlab"
            }
        ],
        "stream": True
    }

    print()
    print("🧪 Testing GitLab My Commits")
    print("=" * 70)
    print(f"📝 Query: {request_payload['messages'][0]['content']}")
    print("=" * 70)
    print()

    try:
        async with httpx.AsyncClient(timeout=120.0) as client:
            async with client.stream("POST", url, json=request_payload) as response:
                if response.status_code != 200:
                    print(f"❌ Error: {response.status_code}")
                    await server.shutdown()
                    return False

                full_response = ""
                tool_calls = []

                async for line in response.aiter_lines():
                    if not line.strip() or not line.startswith("data: "):
                        continue

                    data = line[6:]
                    if data == "[DONE]":
                        break

                    try:
                        chunk = json.loads(data)

                        # Track tool usage
                        if "tool_call" in str(chunk):
                            tool_calls.append(chunk)

                        # Print content
                        if "choices" in chunk and len(chunk["choices"]) > 0:
                            delta = chunk["choices"][0].get("delta", {})
                            content = delta.get("content", "")
                            if content:
                                print(content, end="", flush=True)
                                full_response += content

                    except json.JSONDecodeError:
                        continue

                print()
                print()
                print("=" * 70)
                print("📊 RESULTS")
                print("=" * 70)

                # Check for success indicators
                has_commits = any(word in full_response.lower() for word in ["commit", "pushed", "branch"])
                has_stats = any(word in full_response.lower() for word in ["statistic", "total", "count"])
                no_error = "error" not in full_response.lower() or "no recent" in full_response.lower()

                print(f"✓ Response contains commit info: {has_commits}")
                print(f"✓ Response contains statistics: {has_stats}")
                print(f"✓ No errors: {no_error}")
                print()

                success = has_commits and no_error

                if success:
                    print("🎉 TEST PASSED!")
                else:
                    print("⚠️  TEST INCOMPLETE - Check response above")

                await server.shutdown()
                return success

    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        await server.shutdown()
        return False


if __name__ == "__main__":
    result = asyncio.run(test_gitlab_commits())
    sys.exit(0 if result else 1)
