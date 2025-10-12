# Claude Code Configuration for ADK LLM Proxy

This directory contains Claude Code configuration files that provide project-specific context and commands.

## Files

### CLAUDE.md
Main project context file containing:
- Project architecture and structure
- Coding standards and conventions
- Development workflow and best practices
- Common pitfalls and solutions
- Environment variables and configuration
- Code review checklist

This file is automatically loaded when Claude Code starts in this project.

### settings.json
Claude Code settings including:
- System prompt customization
- Tool permissions
- Auto-approve patterns for common commands
- Setting sources configuration

### commands/
Custom slash commands for common project tasks:

- `/start-server` - Start the ADK LLM Proxy server
- `/test-mcp` - Test MCP server connections
- `/test-gitlab` - Test GitLab integration
- `/test-youtrack` - Test YouTrack integration
- `/check-config` - Validate configuration file
- `/create-workflow` - Create a new reasoning workflow
- `/create-mcp` - Create a new MCP server
- `/analyze-commits` - Analyze recent GitLab commits
- `/epic-report` - Generate YouTrack epic report
- `/fix-imports` - Fix common import errors
- `/run-tests` - Run the test suite

## Usage

### Loading Project Context

The project context in `CLAUDE.md` is automatically loaded when you start Claude Code in this directory. It provides comprehensive information about:

- Architecture (DDD structure, request pipeline)
- Coding standards (Python style, async patterns, error handling)
- Important files and their purposes
- Development workflow and commands
- Configuration and environment variables

### Using Slash Commands

Type `/` in Claude Code to see all available commands. For example:

```
/start-server
```

This will start the ADK LLM Proxy server with default settings.

### Auto-Approved Commands

The following commands are auto-approved and won't require confirmation:

- `pytest` commands (running tests)
- `python test_*` commands (test scripts)
- `python main.py` commands (starting server)
- `black` commands (code formatting)
- `flake8` commands (linting)

## Customization

### Adding New Commands

To add a new slash command:

1. Create a new `.md` file in `commands/` directory
2. Add frontmatter with description:
   ```yaml
   ---
   description: Your command description
   ---
   ```
3. Write the command instructions
4. Save and restart Claude Code

### Modifying System Prompt

Edit the `systemPrompt.append` field in `settings.json` to add or modify custom instructions for Claude Code.

### Adding Auto-Approvals

Add new patterns to the `permissions.autoApprove` array in `settings.json`:

```json
{
  "tool": "bash",
  "pattern": "your-command-pattern.*"
}
```

## Tips

1. **Read CLAUDE.md first** when working on unfamiliar parts of the codebase
2. **Use slash commands** for common tasks to save time
3. **Check the architecture** section before making structural changes
4. **Follow the coding standards** outlined in CLAUDE.md
5. **Reference the code review checklist** before committing changes

## Troubleshooting

If Claude Code doesn't load the configuration:

1. Check that files are in `.claude/` directory
2. Verify JSON syntax in `settings.json`
3. Ensure `CLAUDE.md` is readable
4. Restart Claude Code

## Learn More

- [Claude Code Documentation](https://docs.claude.com/en/docs/claude-code)
- [Custom Slash Commands](https://docs.claude.com/en/docs/claude-code/sdk/sdk-slash-commands)
- [Settings Configuration](https://docs.claude.com/en/docs/claude-code/settings)
