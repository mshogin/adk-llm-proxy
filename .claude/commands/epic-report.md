---
description: Generate a comprehensive YouTrack epic report
---

Generate a comprehensive report for a YouTrack epic including tasks, progress, and insights.

Ask the user for:
1. Epic ID (e.g., "PROJ-123")
2. Include code analysis from GitLab? (yes/no)

Then create and run a Python script that:
1. Connects to YouTrack via the MCP server
2. Retrieves epic details and all related tasks
3. Analyzes:
   - Current status and progress
   - Task distribution by state
   - Assignee workload
   - Timeline and estimated completion
4. If GitLab analysis requested:
   - Find commits linked to epic tasks
   - Calculate code metrics (lines changed, files affected)
   - Identify active developers
5. Present findings in a comprehensive markdown report

Use the existing YouTrack and GitLab MCP client patterns from test files.
