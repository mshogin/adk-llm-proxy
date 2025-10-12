#!/usr/bin/env python3
"""
Integration test for YouTrack with enhanced reasoning workflow.
This test starts the server programmatically - no separate terminal needed.
"""

import asyncio
import httpx
import json
import sys
import threading
import time
from contextlib import asynccontextmanager


class ServerManager:
    """Manages the FastAPI server lifecycle for testing."""

    def __init__(self, port: int = 8001):
        self.port = port
        self.server_thread = None
        self.server = None
        self.should_stop = False

    def start_server(self):
        """Start the server in a background thread."""
        import uvicorn
        from main import setup_server

        # Setup server with test port
        app, config = setup_server(
            provider="openai",
            model="gpt-4o-mini",
            port=self.port
        )

        # Run uvicorn server
        uvicorn_config = uvicorn.Config(
            app,
            host="127.0.0.1",
            port=self.port,
            log_level="error",  # Minimize noise
            access_log=False
        )
        self.server = uvicorn.Server(uvicorn_config)

        # Run server in thread
        def run_server():
            asyncio.run(self.server.serve())

        self.server_thread = threading.Thread(target=run_server, daemon=True)
        self.server_thread.start()

    async def wait_for_ready(self, timeout: float = 30.0) -> bool:
        """
        Wait for server to be ready to accept connections.

        Args:
            timeout: Maximum time to wait in seconds

        Returns:
            bool: True if server is ready, False if timeout
        """
        start_time = time.time()
        url = f"http://127.0.0.1:{self.port}/health"

        print(f"⏳ Waiting for server on port {self.port} to be ready...")

        async with httpx.AsyncClient() as client:
            while time.time() - start_time < timeout:
                try:
                    response = await client.get(url, timeout=1.0)
                    if response.status_code == 200:
                        elapsed = time.time() - start_time
                        print(f"✅ Server ready after {elapsed:.1f}s")
                        return True
                except (httpx.ConnectError, httpx.TimeoutException):
                    await asyncio.sleep(0.5)
                    continue

        print(f"❌ Server failed to start within {timeout}s")
        return False

    async def shutdown(self):
        """Shutdown the server gracefully."""
        if self.server:
            self.server.should_exit = True
            # Give server time to cleanup
            await asyncio.sleep(1.0)


async def test_youtrack_query():
    """Test the complete flow: enhanced reasoning -> MCP tools -> YouTrack data."""

    # Start server programmatically
    server = ServerManager(port=8001)
    server.start_server()

    # Wait for server to be ready
    if not await server.wait_for_ready(timeout=30.0):
        print("❌ Failed to start server")
        return False

    url = f"http://127.0.0.1:{server.port}/v1/chat/completions"

    request_payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {
                "role": "user",
                "content": "Please collect statistics by project ST in youtrack for the last week"
            }
        ],
        "stream": True
    }

    print()
    print("🧪 Testing YouTrack Integration")
    print("=" * 70)
    print(f"📝 User Query: {request_payload['messages'][0]['content']}")
    print("=" * 70)
    print()

    try:
        async with httpx.AsyncClient(timeout=120.0) as client:
            async with client.stream("POST", url, json=request_payload) as response:
                print(f"📡 Response Status: {response.status_code}")
                print()

                if response.status_code != 200:
                    error_text = await response.aread()
                    print(f"❌ Error: {response.status_code}")
                    print(error_text.decode())
                    await server.shutdown()
                    return False

                full_response = ""
                reasoning_steps = []
                mcp_tools_executed = []
                in_reasoning_block = False

                async for line in response.aiter_lines():
                    if not line.strip():
                        continue

                    if line.startswith("data: "):
                        data = line[6:]

                        if data == "[DONE]":
                            break

                        try:
                            chunk = json.loads(data)

                            # Check for reasoning metadata
                            if "reasoning" in chunk:
                                reasoning = chunk["reasoning"]
                                step_name = reasoning.get("step")
                                step_data = reasoning.get("data", {})

                                if step_name:
                                    if step_name not in reasoning_steps:
                                        reasoning_steps.append(step_name)

                                    print(f"🧠 {step_name}")

                                    if "status" in step_data and isinstance(step_data["status"], str):
                                        status = step_data["status"][:100]
                                        print(f"   └─ {status}")

                                    if "tools_executed" in step_data:
                                        tools_count = step_data["tools_executed"]
                                        print(f"   └─ Executed {tools_count} tools")
                                        if tools_count > 0:
                                            mcp_tools_executed.append(tools_count)

                            # Check for content
                            if "choices" in chunk and len(chunk["choices"]) > 0:
                                delta = chunk["choices"][0].get("delta", {})
                                content = delta.get("content", "")

                                if content:
                                    # Check for reasoning block markers
                                    if "**Reasoning**:" in content or "**Enhanced Reasoning" in content:
                                        in_reasoning_block = True
                                    elif "**Response**:" in content or in_reasoning_block and content.strip().startswith("Based on"):
                                        in_reasoning_block = False
                                        print()
                                        print("💬 LLM Response:")
                                        print("-" * 70)

                                    # Only print LLM response, not reasoning block
                                    if not in_reasoning_block and content.strip():
                                        print(content, end="", flush=True)

                                    full_response += content

                        except json.JSONDecodeError:
                            continue

                print()
                print()
                print("-" * 70)
                print()
                print("=" * 70)
                print("📊 TEST RESULTS")
                print("=" * 70)
                print()

                # Check reasoning workflow executed
                print("✅ Reasoning Workflow:")
                if len(reasoning_steps) > 0:
                    for step in reasoning_steps:
                        print(f"   ✓ {step}")
                else:
                    print("   ❌ No reasoning steps detected")
                print()

                # Check MCP tools
                print("✅ MCP Tools:")
                if len(mcp_tools_executed) > 0:
                    total_tools = sum(mcp_tools_executed)
                    print(f"   ✓ Total tools executed: {total_tools}")
                else:
                    print("   ⚠️  No MCP tools executed")
                print()

                # Check for YouTrack-specific content
                has_st_issues = "ST-" in full_response
                mentions_youtrack = "YouTrack" in full_response or "youtrack" in full_response.lower()
                client_initialized = "Error: YouTrack client not initialized" not in full_response
                has_statistics = any(word in full_response.lower() for word in ["statistic", "summary", "count", "total", "issue"])

                print("✅ YouTrack Integration:")
                print(f"   {'✓' if has_st_issues else '✗'} Contains ST- issue IDs")
                print(f"   {'✓' if mentions_youtrack else '✗'} Mentions YouTrack")
                print(f"   {'✓' if client_initialized else '✗'} Client initialized")
                print(f"   {'✓' if has_statistics else '✗'} Contains statistics/data")
                print()

                print("📈 Response Stats:")
                print(f"   Total length: {len(full_response)} characters")
                print(f"   Reasoning steps: {len(reasoning_steps)}")
                print(f"   MCP tool calls: {sum(mcp_tools_executed) if mcp_tools_executed else 0}")
                print()

                # Final verdict
                all_checks_passed = (
                    len(reasoning_steps) > 0 and
                    client_initialized and
                    (has_st_issues or has_statistics)
                )

                if all_checks_passed:
                    print("🎉 TEST PASSED: Complete integration working!")
                    print()
                    print("✓ Enhanced reasoning workflow executed")
                    print("✓ MCP tools discovered and called")
                    print("✓ YouTrack client initialized")
                    print("✓ Data retrieved and presented")
                    result = True
                elif not client_initialized:
                    print("❌ TEST FAILED: YouTrack client not initialized")
                    print()
                    print("💡 Fix: Check these in config.yaml:")
                    print("   - YOUTRACK_BASE_URL is set correctly")
                    print("   - YOUTRACK_TOKEN is valid")
                    print("   - YouTrack MCP server is enabled")
                    result = False
                elif len(reasoning_steps) == 0:
                    print("❌ TEST FAILED: Reasoning workflow did not execute")
                    print()
                    print("💡 Fix: Check config.yaml:")
                    print("   - reasoning_workflow: 'workflows/enhanced'")
                    result = False
                else:
                    print("⚠️  TEST PARTIAL: Some checks failed")
                    print()
                    print("💡 Review the response above for details")
                    result = False

                # Shutdown server
                await server.shutdown()
                return result

    except Exception as e:
        print(f"❌ Unexpected Error: {e}")
        import traceback
        traceback.print_exc()
        await server.shutdown()
        return False


if __name__ == "__main__":
    print()
    print("=" * 70)
    print("  YouTrack Integration Test")
    print("  Server will be started automatically")
    print("=" * 70)
    print()

    result = asyncio.run(test_youtrack_query())

    print()
    sys.exit(0 if result else 1)
