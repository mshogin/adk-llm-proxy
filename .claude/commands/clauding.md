# Clauding - Automated Roadmap Implementation

Automatically implement the next unimplemented task from the project roadmap.

## Process

1. **Read the roadmap** from `ROADMAP.md`
2. **Find the next unimplemented task** (first `- [ ]` checkbox)
3. **Analyze the task** - understand what needs to be done
4. **Implement the task** - execute the necessary changes
5. **Mark as complete** - update `ROADMAP.md` to mark task as `- [x]`
6. **Report completion** - provide summary of what was done

## Task Implementation Guidelines

### For Test File Movement Tasks
- Create target directory if it doesn't exist
- Move the file using `git mv` (preserves history)
- Update imports in the moved file
- Find and update all references to the moved file
- Run the test to ensure it still works
- Mark task as complete

### For Directory Creation Tasks
- Create the directory structure with `mkdir -p`
- Add `__init__.py` files to make them Python packages
- Verify structure matches DDD guidelines
- Mark task as complete

### For File Creation Tasks
- Analyze what the file should contain based on:
  - Similar existing files
  - The module it's testing (if it's a test file)
  - Project conventions in `.claude/CLAUDE.md`
- Create the file with proper structure
- Add necessary imports and boilerplate
- Implement basic functionality
- Run tests to verify
- Mark task as complete

### For Configuration Update Tasks
- Read the current configuration
- Update as specified in the task
- Verify the configuration is valid
- Test that the changes work
- Mark task as complete

### For Source Code Organization Tasks
- Analyze current structure
- Plan the changes
- Execute the refactoring
- Update all imports
- Run tests to ensure nothing broke
- Mark task as complete

### For Documentation Tasks
- Review related code and context
- Write clear, comprehensive documentation
- Follow project documentation style
- Add examples where appropriate
- Mark task as complete

## Git Commit Guidelines

After completing each task, create a git commit following these rules:

### Commit Message Format
Use conventional commit format:
```
<type>: <short description>

<optional detailed description>

🤖 Generated with Claude Code
```

### Commit Types
- `feat:` - New feature or capability (e.g., "feat: create tests directory structure")
- `refactor:` - Code restructuring without behavior change (e.g., "refactor: move test files to proper directories")
- `test:` - Adding or updating tests (e.g., "test: add integration tests for GitLab")
- `docs:` - Documentation changes (e.g., "docs: create MCP integration guide")
- `chore:` - Maintenance tasks (e.g., "chore: organize utility scripts")
- `fix:` - Bug fixes (e.g., "fix: update imports after file reorganization")

### Commit Process
1. Stage all changes related to the task: `git add <files>`
2. Create commit with descriptive message
3. Verify commit was created successfully
4. Display commit hash in task summary

## Execution

**START CONTINUOUS IMPLEMENTATION NOW:**

**LOOP until no more unchecked tasks exist:**

1. Read `/Users/mshogin/my/agents/ROADMAP.md`
2. Search for the first unchecked task: `- [ ]`
3. **If no unchecked tasks found** - display completion summary and STOP
4. **If unchecked task found:**
   - Display the task being implemented
   - Implement the task following the guidelines above
   - Update ROADMAP.md to mark the task as complete: `- [x]`
   - **Create git commit** with descriptive message about what was done
   - Provide a brief summary of what was done
   - **CONTINUE to next task** (go back to step 1)

## Important Notes

- **Continuous execution** - implement tasks continuously until all are complete
- **Test after implementation** - always verify the change works
- **Update imports** - fix any broken imports caused by moves
- **Follow DDD** - maintain layer separation and project structure
- **Git commit after each task** - create a commit immediately after completing each task
- **Commit message format** - use clear, descriptive messages (e.g., "feat: create tests directory structure mirroring src")
- **Progress tracking** - keep count of tasks completed in the session
- **Brief summaries** - keep per-task output concise, detailed summary at end

## Output Format

**Per Task (Brief):**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Task [N]: [Task description]
Phase: [Phase number and name]

🔨 [Brief description of what was done]
✅ Complete
📝 Committed: [commit hash]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Final Summary (When all tasks complete):**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎉 CLAUDING - ALL TASKS COMPLETED!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 Session Summary:
- Total tasks completed: [count]
- Phases completed: [list]
- Time taken: [if available]

✅ All roadmap tasks have been successfully implemented!

🔍 Summary of Changes:
[Brief list of major accomplishments]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Roadmap 100% complete - ready for commit!
```

BEGIN CONTINUOUS IMPLEMENTATION!
