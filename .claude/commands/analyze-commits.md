---
description: Analyze recent GitLab commits for a project
---

Analyze recent commits in a GitLab project to understand development activity.

Ask the user for:
1. Project ID or project name (optional, will search if name provided)
2. Number of days to analyze (default: 7)
3. Branch name (optional, default: main/master)

Then create and run a Python script that:
1. Connects to GitLab via the MCP server
2. Retrieves recent commits
3. Analyzes commit patterns:
   - Number of commits
   - Active contributors
   - File change statistics
   - Commit message patterns
   - Linked tasks (if any)
4. Presents findings in a clear, formatted report

Use the existing GitLab MCP client code patterns from test_gitlab_*.py files.
