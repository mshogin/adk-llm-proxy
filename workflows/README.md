# Reasoning Workflows

This directory contains customizable reasoning workflow callbacks that control how the reasoning pipeline processes requests.

## Overview

The reasoning workflow system provides a unified way to customize the reasoning process. **There is only one reasoning implementation** - the workflow callback system. All reasoning variants (default, enhanced, empty, custom) are implemented as workflows.

## Available Workflows

- **`workflows/default`** - Standard reasoning pipeline with intent analysis, MCP tool discovery/execution, context generation, and message enhancement
- **`workflows/enhanced`** - LLM-powered multi-agent reasoning with advanced orchestration (uses `enhanced_reasoning_orchestrator`)
- **`workflows/empty`** - No-op workflow that skips all reasoning and passes through the request unchanged
- **`workflows/custom`** - Your custom workflows go here

## Configuration

Set the workflow to use in your `config.yaml`:

```yaml
processing:
  reasoning_workflow: "workflows/default"  # Path to workflow directory
```

## Creating a Custom Workflow

1. Create a new directory under `workflows/` (e.g., `workflows/custom`)
2. Create `__init__.py` that exports `reasoning_workflow`
3. Create `reasoning_callback.py` with your workflow implementation

### Example Structure

```
workflows/
├── default/
│   ├── __init__.py
│   └── reasoning_callback.py
├── custom/
│   ├── __init__.py
│   └── reasoning_callback.py
└── README.md
```

### Workflow Function Signature

```python
async def reasoning_workflow(
    request_data: Dict[str, Any],
    analyze_request_intent,
    generate_reasoning_context,
    enhance_messages_with_reasoning,
    discover_reasoning_tools,
    execute_reasoning_tools,
    stream_reasoning_step
) -> AsyncGenerator[str, None]:
    """
    Custom reasoning workflow callback.

    Args:
        request_data: The request data to process
        analyze_request_intent: Function to analyze user intent
        generate_reasoning_context: Function to generate reasoning context
        enhance_messages_with_reasoning: Function to enhance messages
        discover_reasoning_tools: Function to discover MCP tools
        execute_reasoning_tools: Function to execute MCP tools
        stream_reasoning_step: Function to stream reasoning steps

    Yields:
        Reasoning step chunks in SSE format
    """
    # Your custom workflow implementation
    pass
```

## Available Helper Functions

The workflow receives these helper functions as parameters:

### `analyze_request_intent(request_data)`
- Analyzes user intent, complexity, and domains
- Returns: `{"status": "success", "intent_analysis": {...}}`

### `generate_reasoning_context(intent_analysis, messages, reasoning_insights)`
- Generates reasoning context based on analysis
- Returns: `{"status": "success", "reasoning_context": [...], "reasoning_prompt": "..."}`

### `enhance_messages_with_reasoning(messages, reasoning_context)`
- Enhances messages with reasoning context
- Returns: `{"status": "success", "enhanced_messages": [...]}`

### `discover_reasoning_tools(request_data, intent_analysis)`
- Discovers MCP tools suitable for reasoning
- Returns: `{"status": "success", "reasoning_tools": [...], "preferred_tools": [...]}`

### `execute_reasoning_tools(request_data, reasoning_tools, intent_analysis)`
- Executes reasoning tools with intelligent orchestration
- Returns: `{"status": "success", "reasoning_insights": {...}, "execution_plan": {...}}`

### `stream_reasoning_step(step_name, step_data, enhanced_request)`
- Streams a reasoning step to the client
- Returns: SSE formatted chunk string

## Default Workflow

The default workflow (`workflows/default`) implements the standard reasoning pipeline:

1. Analyze request intent
2. Discover MCP reasoning tools
3. Execute reasoning tools
4. Generate reasoning context
5. Enhance messages with reasoning

## Example: Simple Workflow

```python
# workflows/simple/reasoning_callback.py
async def reasoning_workflow(
    request_data,
    analyze_request_intent,
    generate_reasoning_context,
    enhance_messages_with_reasoning,
    discover_reasoning_tools,
    execute_reasoning_tools,
    stream_reasoning_step
):
    # Skip tool execution for simple requests
    yield await stream_reasoning_step("simple_analysis", {"status": "analyzing..."}, None)

    intent_result = analyze_request_intent(request_data)

    yield await stream_reasoning_step("simple_analysis", {
        "status": "completed",
        "complexity": intent_result["intent_analysis"]["complexity"]
    }, None)

    # Generate minimal context
    messages = request_data.get("messages", [])
    context_result = generate_reasoning_context(
        intent_result["intent_analysis"],
        messages,
        {}  # No MCP insights
    )

    # Enhance and complete
    enhancement_result = enhance_messages_with_reasoning(
        messages,
        context_result["reasoning_prompt"]
    )

    enhanced_request = request_data.copy()
    enhanced_request["messages"] = enhancement_result["enhanced_messages"]

    yield await stream_reasoning_step("complete", {
        "status": "simple workflow completed"
    }, enhanced_request)
```

## Switching Workflows

To switch to a different workflow:

1. Update `config.yaml`:
   ```yaml
   processing:
     reasoning_workflow: "workflows/custom"
   ```

2. Restart the server

The system will automatically load the new workflow callback on startup.

## Debugging

Enable debug logging to see workflow loading:

```yaml
logging:
  level: "DEBUG"
```

Look for log messages like:
- `🔄 Loading reasoning workflow from: workflows/default`
- `✅ Successfully loaded workflow callback`
- `🔄 Using custom workflow callback`
