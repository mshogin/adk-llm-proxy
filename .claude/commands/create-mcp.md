---
description: Create a new MCP server from template
---

Create a new MCP (Model Context Protocol) server for integrating with an external system.

Ask the user for:
1. MCP server name (e.g., "jira", "github", "slack")
2. Description of what it should do
3. What tools it should provide
4. What resources it should expose

Then:
1. Create directory: mcps/{name}/
2. Copy and customize the template from mcps/template/
3. Implement server.py extending MCPServerBase
4. Create README.md with setup instructions
5. Add requirements.txt with dependencies
6. Show configuration to add to config.yaml

Provide code with:
- Proper error handling
- Async/await patterns
- Type hints
- Comprehensive docstrings
