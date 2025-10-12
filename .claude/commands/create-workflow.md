---
description: Create a new custom reasoning workflow
---

Create a new custom reasoning workflow for the ADK LLM Proxy.

Ask the user for:
1. Workflow name (e.g., "simple", "advanced", "domain-specific")
2. Workflow description and purpose
3. Which steps to include (preprocessing, tool discovery, tool execution, etc.)

Then:
1. Create directory: workflows/{name}/
2. Create __init__.py that exports reasoning_workflow
3. Create reasoning_callback.py with the workflow implementation
4. Add comprehensive docstrings and comments
5. Show the user how to activate it in config.yaml

Base the implementation on the default workflow in workflows/default/ but customize according to user requirements.
