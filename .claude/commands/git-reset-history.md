---
description: Reset repository to first commit keeping current files
---

# Reset History - Keep Files, Remove History

Reset the repository to the first commit while keeping all current files, and delete all branches except main (both local and remote).

## WARNING

This is a DESTRUCTIVE operation that will:
- Delete all local branches except main
- Delete all remote branches except main
- Reset commit history to first commit
- Keep all current files (but recommit them on top of first commit)
- Force push to remote (overwrites remote history)

**This action cannot be undone!**

## Task

1. **Check current state**:
   ```bash
   git status
   git branch -a
   git log --reverse --oneline | head -1
   ```

2. **Ensure we're on main branch**:
   ```bash
   git checkout main
   ```

3. **Stage all current changes** (if any uncommitted):
   ```bash
   git add -A
   ```

4. **Get first commit hash**:
   ```bash
   FIRST_COMMIT=$(git log --reverse --oneline | head -1 | awk '{print $1}')
   echo "First commit: $FIRST_COMMIT"
   ```

5. **Reset to first commit keeping files** (soft reset):
   ```bash
   git reset --soft $FIRST_COMMIT
   ```
   This keeps all files staged but moves HEAD to first commit

6. **Commit all current files**:
   ```bash
   git commit -m "Reset history - current state of project

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
   ```

7. **Delete local branches** (except main):
   ```bash
   git branch | grep -v "^\* main" | grep -v "^  main" | xargs -r git branch -D
   ```

8. **Delete remote branches** (except main):
   ```bash
   # Get list of remote branches to delete
   REMOTE_BRANCHES=$(git branch -r | grep -v "HEAD" | grep -v "main" | sed 's/origin\///')

   # Delete each remote branch
   for branch in $REMOTE_BRANCHES; do
     git push origin --delete $branch
   done
   ```

9. **Force push main to remote**:
   ```bash
   git push origin main --force
   ```

10. **Verify final state**:
    ```bash
    git log --oneline
    git branch -a
    git status
    ```

## Output Format

```
🔄 REPOSITORY RESET - KEEP FILES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Current state:
- Branches: [list all branches]
- First commit: [hash] [message]
- Current files: [status summary]

⚠️  WARNING: This will permanently delete all history!
Current files will be preserved and recommitted.

Proceeding...

✓ Switched to main branch
✓ Staged all changes
✓ Reset to first commit (soft)
✓ Committed current state
✓ Deleted local branches: [list]
✓ Deleted remote branches: [list]
✓ Force pushed to remote

Final state:
- Branch: main
- Commits: 2 (first commit + current state)
- All current files preserved

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Repository reset complete!

⚠️  Other developers will need to:
   git fetch origin
   git reset --hard origin/main
```

## What This Does

1. **Keeps all your files** - no file changes
2. **Removes commit history** - only 2 commits remain:
   - Original first commit
   - New commit with all current files
3. **Deletes all branches** - local and remote (except main)
4. **Force pushes** - overwrites remote repository

## Safety Notes

- Files are preserved (Option A)
- Commit history is removed
- Branches are deleted (local + remote)
- Other developers must re-sync with: `git fetch && git reset --hard origin/main`

START THE RESET NOW!
