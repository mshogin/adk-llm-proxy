#!/usr/bin/env python3
"""Direct test of get_my_recent_commits without statistics confusion."""

import asyncio
import httpx
import json
import sys
import threading
import time


class ServerManager:
    def __init__(self, port: int = 8003):
        self.port = port
        self.server_thread = None
        self.server = None

    def start_server(self):
        import uvicorn
        from main import setup_server
        app, config = setup_server(provider="openai", model="gpt-4o-mini", port=self.port)
        uvicorn_config = uvicorn.Config(app, host="127.0.0.1", port=self.port, log_level="error", access_log=False)
        self.server = uvicorn.Server(uvicorn_config)
        def run_server():
            asyncio.run(self.server.serve())
        self.server_thread = threading.Thread(target=run_server, daemon=True)
        self.server_thread.start()

    async def wait_for_ready(self, timeout: float = 30.0) -> bool:
        start_time = time.time()
        url = f"http://127.0.0.1:{self.port}/health"
        async with httpx.AsyncClient() as client:
            while time.time() - start_time < timeout:
                try:
                    response = await client.get(url, timeout=1.0)
                    if response.status_code == 200:
                        return True
                except:
                    await asyncio.sleep(0.5)
        return False

    async def shutdown(self):
        if self.server:
            self.server.should_exit = True
            await asyncio.sleep(1.0)


async def test_commits():
    server = ServerManager()
    server.start_server()

    if not await server.wait_for_ready():
        print("❌ Server failed to start")
        return False

    url = f"http://127.0.0.1:{server.port}/v1/chat/completions"

    # Test different queries
    queries = [
        "Show me my latest 5 commits in GitLab",
        "What are my recent commits in GitLab?",
        "List my last 5 GitLab commits"
    ]

    for i, query in enumerate(queries, 1):
        print(f"\n{'='*70}")
        print(f"Test {i}/{ len(queries)}: {query}")
        print('='*70)

        request = {
            "model": "gpt-4o-mini",
            "messages": [{"role": "user", "content": query}],
            "stream": True
        }

        try:
            async with httpx.AsyncClient(timeout=60.0) as client:
                async with client.stream("POST", url, json=request) as response:
                    if response.status_code != 200:
                        print(f"❌ Error: {response.status_code}")
                        continue

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
                            # Check for tool mentions in response
                            if "choices" in chunk and len(chunk["choices"]) > 0:
                                delta = chunk["choices"][0].get("delta", {})
                                content = delta.get("content", "")
                                if content:
                                    full_response += content
                                    if "get_my_recent_commits" in content:
                                        tool_calls.append("get_my_recent_commits")
                        except:
                            pass

                    # Check results
                    has_gitlab_data = any(word in full_response.lower() for word in ["pushed", "branch", "commit", "project"])
                    called_right_tool = "get_my_recent_commits" in str(tool_calls) or "get_my_recent_commits" in full_response
                    no_git_instructions = "git log" not in full_response.lower()

                    print(f"\n✓ Has GitLab data: {has_gitlab_data}")
                    print(f"✓ Called get_my_recent_commits: {called_right_tool}")
                    print(f"✓ Not generic Git instructions: {no_git_instructions}")

                    if has_gitlab_data and no_git_instructions:
                        print("🎉 SUCCESS!")
                    else:
                        print("⚠️  FAIL - Generic response instead of GitLab API data")
                        print(f"\nResponse preview: {full_response[:200]}...")

        except Exception as e:
            print(f"❌ Error: {e}")

    await server.shutdown()
    return True


if __name__ == "__main__":
    asyncio.run(test_commits())
