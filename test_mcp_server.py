#!/usr/bin/env python3
"""
Simple Test MCP Server for validation and testing.

This server provides basic tools for testing MCP integration functionality.
It implements echo tools, file system operations, and utility functions.
"""

import asyncio
import json
import logging
import sys
from typing import Dict, Any, List
from datetime import datetime
import os
from pathlib import Path

# Simple MCP server implementation for testing
class TestMCPServer:
    """Simple MCP server for testing purposes."""

    def __init__(self):
        self.logger = logging.getLogger("TestMCPServer")
        self.tools = self._register_tools()
        self.resources = self._register_resources()
        self.prompts = self._register_prompts()

    def _register_tools(self) -> Dict[str, Dict[str, Any]]:
        """Register available tools."""
        return {
            "echo": {
                "description": "Echo back the input message",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "message": {
                            "type": "string",
                            "description": "Message to echo back"
                        }
                    },
                    "required": ["message"]
                }
            },
            "add": {
                "description": "Add two numbers",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "a": {
                            "type": "number",
                            "description": "First number"
                        },
                        "b": {
                            "type": "number",
                            "description": "Second number"
                        }
                    },
                    "required": ["a", "b"]
                }
            },
            "get_time": {
                "description": "Get current time",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "format": {
                            "type": "string",
                            "description": "Time format (iso, timestamp, human)",
                            "enum": ["iso", "timestamp", "human"],
                            "default": "iso"
                        }
                    }
                }
            },
            "list_files": {
                "description": "List files in a directory",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Directory path to list",
                            "default": "."
                        },
                        "pattern": {
                            "type": "string",
                            "description": "Optional file pattern to match"
                        }
                    }
                }
            },
            "read_file": {
                "description": "Read contents of a file",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Path to the file to read"
                        },
                        "encoding": {
                            "type": "string",
                            "description": "File encoding",
                            "default": "utf-8"
                        }
                    },
                    "required": ["path"]
                }
            },
            "write_file": {
                "description": "Write content to a file",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "Path to the file to write"
                        },
                        "content": {
                            "type": "string",
                            "description": "Content to write to the file"
                        },
                        "encoding": {
                            "type": "string",
                            "description": "File encoding",
                            "default": "utf-8"
                        }
                    },
                    "required": ["path", "content"]
                }
            },
            "error_tool": {
                "description": "Tool that always throws an error (for testing error handling)",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "error_message": {
                            "type": "string",
                            "description": "Custom error message",
                            "default": "Test error"
                        }
                    }
                }
            },
            "slow_tool": {
                "description": "Tool that takes some time to execute (for testing timeouts)",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "delay": {
                            "type": "number",
                            "description": "Delay in seconds",
                            "default": 2.0,
                            "minimum": 0,
                            "maximum": 30
                        }
                    }
                }
            }
        }

    def _register_resources(self) -> Dict[str, Dict[str, Any]]:
        """Register available resources."""
        return {
            "test://config": {
                "name": "Test Configuration",
                "description": "Test server configuration",
                "mimeType": "application/json"
            },
            "test://status": {
                "name": "Server Status",
                "description": "Current server status information",
                "mimeType": "application/json"
            },
            "test://logs": {
                "name": "Server Logs",
                "description": "Recent server log entries",
                "mimeType": "text/plain"
            }
        }

    def _register_prompts(self) -> Dict[str, Dict[str, Any]]:
        """Register available prompts."""
        return {
            "greeting": {
                "description": "Generate a greeting message",
                "arguments": [
                    {
                        "name": "name",
                        "description": "Name to greet",
                        "required": False
                    },
                    {
                        "name": "style",
                        "description": "Greeting style (formal, casual, friendly)",
                        "required": False
                    }
                ]
            },
            "summarize": {
                "description": "Create a summary prompt",
                "arguments": [
                    {
                        "name": "topic",
                        "description": "Topic to summarize",
                        "required": True
                    },
                    {
                        "name": "length",
                        "description": "Summary length (short, medium, long)",
                        "required": False
                    }
                ]
            }
        }

    async def handle_message(self, message: str) -> str:
        """Handle incoming MCP message."""
        try:
            data = json.loads(message)

            if data.get("method") == "initialize":
                return self._handle_initialize(data)
            elif data.get("method") == "tools/list":
                return self._handle_list_tools(data)
            elif data.get("method") == "tools/call":
                return await self._handle_call_tool(data)
            elif data.get("method") == "resources/list":
                return self._handle_list_resources(data)
            elif data.get("method") == "resources/read":
                return self._handle_read_resource(data)
            elif data.get("method") == "prompts/list":
                return self._handle_list_prompts(data)
            elif data.get("method") == "prompts/get":
                return self._handle_get_prompt(data)
            else:
                return self._create_error_response(data.get("id"), -32601, f"Method not found: {data.get('method')}")

        except json.JSONDecodeError:
            return self._create_error_response(None, -32700, "Parse error")
        except Exception as e:
            return self._create_error_response(data.get("id"), -32603, f"Internal error: {str(e)}")

    def _handle_initialize(self, data: Dict[str, Any]) -> str:
        """Handle initialization request."""
        response = {
            "jsonrpc": "2.0",
            "id": data.get("id"),
            "result": {
                "protocolVersion": "2024-11-05",
                "capabilities": {
                    "tools": {},
                    "resources": {},
                    "prompts": {}
                },
                "serverInfo": {
                    "name": "test-mcp-server",
                    "version": "1.0.0"
                }
            }
        }
        return json.dumps(response)

    def _handle_list_tools(self, data: Dict[str, Any]) -> str:
        """Handle list tools request."""
        tools_list = []
        for name, info in self.tools.items():
            tools_list.append({
                "name": name,
                "description": info["description"],
                "inputSchema": info["inputSchema"]
            })

        response = {
            "jsonrpc": "2.0",
            "id": data.get("id"),
            "result": {
                "tools": tools_list
            }
        }
        return json.dumps(response)

    async def _handle_call_tool(self, data: Dict[str, Any]) -> str:
        """Handle tool call request."""
        try:
            params = data.get("params", {})
            tool_name = params.get("name")
            arguments = params.get("arguments", {})

            if tool_name not in self.tools:
                return self._create_error_response(data.get("id"), -32602, f"Tool not found: {tool_name}")

            # Execute the tool
            result = await self._execute_tool(tool_name, arguments)

            response = {
                "jsonrpc": "2.0",
                "id": data.get("id"),
                "result": {
                    "content": [
                        {
                            "type": "text",
                            "text": str(result)
                        }
                    ]
                }
            }
            return json.dumps(response)

        except Exception as e:
            return self._create_error_response(data.get("id"), -32603, f"Tool execution error: {str(e)}")

    async def _execute_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Any:
        """Execute a specific tool."""
        if tool_name == "echo":
            return arguments.get("message", "")

        elif tool_name == "add":
            a = arguments.get("a", 0)
            b = arguments.get("b", 0)
            return a + b

        elif tool_name == "get_time":
            now = datetime.now()
            format_type = arguments.get("format", "iso")

            if format_type == "iso":
                return now.isoformat()
            elif format_type == "timestamp":
                return now.timestamp()
            elif format_type == "human":
                return now.strftime("%Y-%m-%d %H:%M:%S")
            else:
                return now.isoformat()

        elif tool_name == "list_files":
            path = arguments.get("path", ".")
            pattern = arguments.get("pattern")

            try:
                path_obj = Path(path)
                if not path_obj.exists():
                    raise FileNotFoundError(f"Path does not exist: {path}")

                if pattern:
                    files = list(path_obj.glob(pattern))
                else:
                    files = list(path_obj.iterdir())

                return [str(f) for f in files if f.is_file()]
            except Exception as e:
                raise Exception(f"Failed to list files: {str(e)}")

        elif tool_name == "read_file":
            path = arguments.get("path")
            encoding = arguments.get("encoding", "utf-8")

            try:
                with open(path, 'r', encoding=encoding) as f:
                    return f.read()
            except Exception as e:
                raise Exception(f"Failed to read file: {str(e)}")

        elif tool_name == "write_file":
            path = arguments.get("path")
            content = arguments.get("content")
            encoding = arguments.get("encoding", "utf-8")

            try:
                with open(path, 'w', encoding=encoding) as f:
                    f.write(content)
                return f"Successfully wrote {len(content)} characters to {path}"
            except Exception as e:
                raise Exception(f"Failed to write file: {str(e)}")

        elif tool_name == "error_tool":
            error_message = arguments.get("error_message", "Test error")
            raise Exception(error_message)

        elif tool_name == "slow_tool":
            delay = arguments.get("delay", 2.0)
            await asyncio.sleep(delay)
            return f"Completed after {delay} seconds"

        else:
            raise Exception(f"Unknown tool: {tool_name}")

    def _handle_list_resources(self, data: Dict[str, Any]) -> str:
        """Handle list resources request."""
        resources_list = []
        for uri, info in self.resources.items():
            resources_list.append({
                "uri": uri,
                "name": info["name"],
                "description": info["description"],
                "mimeType": info.get("mimeType")
            })

        response = {
            "jsonrpc": "2.0",
            "id": data.get("id"),
            "result": {
                "resources": resources_list
            }
        }
        return json.dumps(response)

    def _handle_read_resource(self, data: Dict[str, Any]) -> str:
        """Handle read resource request."""
        try:
            params = data.get("params", {})
            uri = params.get("uri")

            if uri not in self.resources:
                return self._create_error_response(data.get("id"), -32602, f"Resource not found: {uri}")

            # Generate resource content
            content = self._get_resource_content(uri)

            response = {
                "jsonrpc": "2.0",
                "id": data.get("id"),
                "result": {
                    "contents": [
                        {
                            "uri": uri,
                            "mimeType": self.resources[uri].get("mimeType", "text/plain"),
                            "text": content
                        }
                    ]
                }
            }
            return json.dumps(response)

        except Exception as e:
            return self._create_error_response(data.get("id"), -32603, f"Resource read error: {str(e)}")

    def _get_resource_content(self, uri: str) -> str:
        """Get content for a resource."""
        if uri == "test://config":
            return json.dumps({
                "server_name": "test-mcp-server",
                "version": "1.0.0",
                "tools_count": len(self.tools),
                "resources_count": len(self.resources),
                "prompts_count": len(self.prompts)
            }, indent=2)

        elif uri == "test://status":
            return json.dumps({
                "status": "running",
                "uptime": "unknown",
                "timestamp": datetime.now().isoformat(),
                "memory_usage": "unknown"
            }, indent=2)

        elif uri == "test://logs":
            return f"Test MCP Server Log\nTimestamp: {datetime.now().isoformat()}\nStatus: Running\nNo errors reported."

        else:
            return f"Content for {uri}"

    def _handle_list_prompts(self, data: Dict[str, Any]) -> str:
        """Handle list prompts request."""
        prompts_list = []
        for name, info in self.prompts.items():
            prompts_list.append({
                "name": name,
                "description": info["description"],
                "arguments": info.get("arguments", [])
            })

        response = {
            "jsonrpc": "2.0",
            "id": data.get("id"),
            "result": {
                "prompts": prompts_list
            }
        }
        return json.dumps(response)

    def _handle_get_prompt(self, data: Dict[str, Any]) -> str:
        """Handle get prompt request."""
        try:
            params = data.get("params", {})
            name = params.get("name")
            arguments = params.get("arguments", {})

            if name not in self.prompts:
                return self._create_error_response(data.get("id"), -32602, f"Prompt not found: {name}")

            # Generate prompt content
            messages = self._generate_prompt_messages(name, arguments)

            response = {
                "jsonrpc": "2.0",
                "id": data.get("id"),
                "result": {
                    "messages": messages
                }
            }
            return json.dumps(response)

        except Exception as e:
            return self._create_error_response(data.get("id"), -32603, f"Prompt generation error: {str(e)}")

    def _generate_prompt_messages(self, name: str, arguments: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Generate prompt messages."""
        if name == "greeting":
            name_arg = arguments.get("name", "there")
            style = arguments.get("style", "friendly")

            if style == "formal":
                text = f"Good day, {name_arg}. I hope this message finds you well."
            elif style == "casual":
                text = f"Hey {name_arg}! How's it going?"
            else:  # friendly
                text = f"Hello {name_arg}! Nice to meet you!"

            return [
                {
                    "role": "user",
                    "content": {
                        "type": "text",
                        "text": text
                    }
                }
            ]

        elif name == "summarize":
            topic = arguments.get("topic", "general topic")
            length = arguments.get("length", "medium")

            length_instruction = {
                "short": "in 1-2 sentences",
                "medium": "in a paragraph",
                "long": "in detail with multiple paragraphs"
            }.get(length, "in a paragraph")

            text = f"Please provide a summary of {topic} {length_instruction}."

            return [
                {
                    "role": "user",
                    "content": {
                        "type": "text",
                        "text": text
                    }
                }
            ]

        else:
            return [
                {
                    "role": "user",
                    "content": {
                        "type": "text",
                        "text": f"Execute prompt: {name}"
                    }
                }
            ]

    def _create_error_response(self, request_id: Any, code: int, message: str) -> str:
        """Create an error response."""
        response = {
            "jsonrpc": "2.0",
            "id": request_id,
            "error": {
                "code": code,
                "message": message
            }
        }
        return json.dumps(response)


async def main():
    """Main function to run the test MCP server."""
    logging.basicConfig(level=logging.INFO)
    server = TestMCPServer()

    print("Test MCP Server starting...", file=sys.stderr)
    print("Ready to receive MCP requests via stdin/stdout", file=sys.stderr)

    try:
        while True:
            # Read from stdin
            line = await asyncio.get_event_loop().run_in_executor(None, sys.stdin.readline)
            if not line:
                break

            line = line.strip()
            if not line:
                continue

            # Handle the message
            response = await server.handle_message(line)

            # Write response to stdout
            print(response, flush=True)

    except KeyboardInterrupt:
        print("Test MCP Server shutting down...", file=sys.stderr)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)


if __name__ == "__main__":
    asyncio.run(main())