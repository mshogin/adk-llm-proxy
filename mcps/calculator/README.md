# Calculator MCP Server

A simple example MCP server demonstrating basic arithmetic operations.

## Purpose

This server serves as a learning example for MCP server development. It demonstrates:

- Tool registration and implementation
- Input validation
- Error handling (e.g., division by zero)
- Server statistics
- Proper logging

## Available Tools

### 1. `add`
Add two numbers together.

**Parameters:**
- `a` (number): First number
- `b` (number): Second number

**Example:**
```json
{
  "name": "add",
  "arguments": {
    "a": 5,
    "b": 3
  }
}
```

**Response:** `5 + 3 = 8`

### 2. `subtract`
Subtract second number from first number.

**Parameters:**
- `a` (number): Number to subtract from
- `b` (number): Number to subtract

### 3. `multiply`
Multiply two numbers together.

**Parameters:**
- `a` (number): First number
- `b` (number): Second number

### 4. `divide`
Divide first number by second number.

**Parameters:**
- `a` (number): Numerator
- `b` (number): Denominator

**Note:** Returns error if attempting to divide by zero.

### 5. `power`
Raise first number to the power of second number.

**Parameters:**
- `base` (number): Base number
- `exponent` (number): Exponent

### 6. `stats`
Get server statistics.

**Parameters:** None

**Returns:** Operation count and server status.

## Installation

```bash
cd mcps/calculator
pip install -r requirements.txt
```

## Running Standalone

```bash
python -m mcps.calculator.server
```

## Configuration

Add to `config.yaml`:

```yaml
mcp:
  servers:
    calculator:
      command: python
      args:
        - -m
        - mcps.calculator.server
```

## Testing

```bash
# Test via proxy
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "What is 15 + 27?"}
    ]
  }'
```

## Learning Points

This example demonstrates:

1. **Clean Server Structure**: Organized class-based design
2. **Tool Registration**: Multiple tools with different schemas
3. **Error Handling**: Graceful handling of division by zero
4. **State Management**: Tracking operations count
5. **Logging**: Proper logging for debugging
6. **Type Safety**: Type hints for better code quality

## Extending This Example

Try adding:
- Square root operation
- Trigonometric functions (sin, cos, tan)
- Equation solver
- Unit converter
- Memory/history tracking
- Complex number support

## Related Documentation

- [MCP Integration Guide](../../docs/MCP_INTEGRATION.md)
- [MCP Server Development Guide](../../docs/MCP_SERVER_DEVELOPMENT.md)
