---
description: Validate configuration and check for issues
---

Validate the config.yaml configuration file and check for common issues:

1. Read and analyze the config.yaml file using the Read tool
2. Check for:
   - Valid YAML syntax
   - Required fields present (providers, mcp.servers, server settings)
   - API keys configured (warn if defaults detected)
   - MCP server paths and commands are valid
   - Port number is valid (1024-65535)

3. Report any issues or warnings found
4. Suggest fixes for any problems detected

Be thorough and check all critical configuration sections.
