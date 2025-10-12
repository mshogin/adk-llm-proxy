"""
Calculator MCP Server - Example Implementation

A simple MCP server demonstrating basic arithmetic operations.
Perfect for learning MCP server development.
"""

import asyncio
import logging
import sys
from typing import List

from mcp.server import Server
from mcp.types import Tool, TextContent

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stderr)]
)

logger = logging.getLogger("calculator-mcp")


class CalculatorMCPServer:
    """
    Example MCP server providing basic arithmetic operations.

    This server demonstrates:
    - Tool registration
    - Input validation
    - Error handling
    - Structured responses
    """

    def __init__(self):
        """Initialize the calculator server."""
        self.server = Server("calculator-server")
        self.operations_count = 0
        self._register_tools()

        logger.info("Calculator MCP Server initialized")

    def _register_tools(self):
        """Register all available calculator tools."""

        # Addition tool
        self.server.add_tool(
            Tool(
                name="add",
                description="Add two numbers together",
                inputSchema={
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
            ),
            self.add
        )

        # Subtraction tool
        self.server.add_tool(
            Tool(
                name="subtract",
                description="Subtract second number from first number",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "a": {
                            "type": "number",
                            "description": "Number to subtract from"
                        },
                        "b": {
                            "type": "number",
                            "description": "Number to subtract"
                        }
                    },
                    "required": ["a", "b"]
                }
            ),
            self.subtract
        )

        # Multiplication tool
        self.server.add_tool(
            Tool(
                name="multiply",
                description="Multiply two numbers together",
                inputSchema={
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
            ),
            self.multiply
        )

        # Division tool
        self.server.add_tool(
            Tool(
                name="divide",
                description="Divide first number by second number",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "a": {
                            "type": "number",
                            "description": "Numerator"
                        },
                        "b": {
                            "type": "number",
                            "description": "Denominator"
                        }
                    },
                    "required": ["a", "b"]
                }
            ),
            self.divide
        )

        # Power tool
        self.server.add_tool(
            Tool(
                name="power",
                description="Raise first number to the power of second number",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "base": {
                            "type": "number",
                            "description": "Base number"
                        },
                        "exponent": {
                            "type": "number",
                            "description": "Exponent"
                        }
                    },
                    "required": ["base", "exponent"]
                }
            ),
            self.power
        )

        # Statistics tool
        self.server.add_tool(
            Tool(
                name="stats",
                description="Get server statistics (operations performed)",
                inputSchema={
                    "type": "object",
                    "properties": {}
                }
            ),
            self.get_stats
        )

        logger.info("Registered 6 calculator tools")

    async def add(self, a: float, b: float) -> List[TextContent]:
        """Add two numbers."""
        try:
            result = a + b
            self.operations_count += 1

            logger.info(f"Addition: {a} + {b} = {result}")

            return [TextContent(
                type="text",
                text=f"{a} + {b} = {result}"
            )]

        except Exception as e:
            logger.error(f"Error in addition: {e}")
            return [TextContent(
                type="text",
                text=f"Error: {str(e)}"
            )]

    async def subtract(self, a: float, b: float) -> List[TextContent]:
        """Subtract second number from first."""
        try:
            result = a - b
            self.operations_count += 1

            logger.info(f"Subtraction: {a} - {b} = {result}")

            return [TextContent(
                type="text",
                text=f"{a} - {b} = {result}"
            )]

        except Exception as e:
            logger.error(f"Error in subtraction: {e}")
            return [TextContent(
                type="text",
                text=f"Error: {str(e)}"
            )]

    async def multiply(self, a: float, b: float) -> List[TextContent]:
        """Multiply two numbers."""
        try:
            result = a * b
            self.operations_count += 1

            logger.info(f"Multiplication: {a} × {b} = {result}")

            return [TextContent(
                type="text",
                text=f"{a} × {b} = {result}"
            )]

        except Exception as e:
            logger.error(f"Error in multiplication: {e}")
            return [TextContent(
                type="text",
                text=f"Error: {str(e)}"
            )]

    async def divide(self, a: float, b: float) -> List[TextContent]:
        """Divide first number by second."""
        try:
            if b == 0:
                logger.warning(f"Division by zero attempted: {a} / {b}")
                return [TextContent(
                    type="text",
                    text="Error: Division by zero is not allowed"
                )]

            result = a / b
            self.operations_count += 1

            logger.info(f"Division: {a} ÷ {b} = {result}")

            return [TextContent(
                type="text",
                text=f"{a} ÷ {b} = {result}"
            )]

        except Exception as e:
            logger.error(f"Error in division: {e}")
            return [TextContent(
                type="text",
                text=f"Error: {str(e)}"
            )]

    async def power(self, base: float, exponent: float) -> List[TextContent]:
        """Raise base to the power of exponent."""
        try:
            result = base ** exponent
            self.operations_count += 1

            logger.info(f"Power: {base}^{exponent} = {result}")

            return [TextContent(
                type="text",
                text=f"{base}^{exponent} = {result}"
            )]

        except Exception as e:
            logger.error(f"Error in power operation: {e}")
            return [TextContent(
                type="text",
                text=f"Error: {str(e)}"
            )]

    async def get_stats(self) -> List[TextContent]:
        """Get server statistics."""
        stats = f"""Calculator Server Statistics:
Operations performed: {self.operations_count}
Server status: Running
Available operations: add, subtract, multiply, divide, power"""

        logger.info("Statistics requested")

        return [TextContent(
            type="text",
            text=stats
        )]

    async def run(self):
        """Start the MCP server."""
        logger.info("Starting Calculator MCP Server...")

        try:
            async with self.server.stdio_server() as streams:
                await self.server.run(
                    streams[0],
                    streams[1],
                    self.server.create_initialization_options()
                )
        except KeyboardInterrupt:
            logger.info("Server stopped by user")
        except Exception as e:
            logger.error(f"Server error: {e}")
            raise


async def main():
    """Main entry point."""
    server = CalculatorMCPServer()
    await server.run()


if __name__ == "__main__":
    asyncio.run(main())
