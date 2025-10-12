---
description: Test GitLab MCP integration
---

Test the GitLab MCP server integration to verify it can:
- Connect to the configured GitLab instance
- Retrieve projects
- Fetch recent commits
- Analyze commit messages

Use the Bash tool to run:
```bash
python test_gitlab_integration.py
```

If the test file doesn't exist, use:
```bash
python test_gitlab_direct.py
```

Review the output for any connection or authentication errors.
