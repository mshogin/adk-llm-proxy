Automatically clean up ROADMAP.md by organizing completed phases.

## Task

1. **Read ROADMAP.md** and parse all phases
2. **Identify completed phases** - phases where ALL tasks are checked (✓)
3. **Identify uncompleted phases** - phases with ANY unchecked tasks (☐)
4. **Keep in ROADMAP.md**:
   - ALL uncompleted phases (any phase with ☐ tasks)
   - The 3 most recent completed phases
5. **Move to ROADMAP-DONE.md**:
   - All older completed phases (beyond the 3 most recent)
   - Append to ROADMAP-DONE.md (don't overwrite)
   - Add timestamp when moving

## Rules

- Preserve exact formatting of each phase
- Maintain phase order (newest first)
- Add separator and timestamp when moving to ROADMAP-DONE.md
- Never lose any content
- If ROADMAP-DONE.md doesn't exist, create it with a header

## Output Format

After cleanup, show:
```
✓ Roadmap cleaned up!

ROADMAP.md now contains:
- N uncompleted phases
- 3 latest completed phases

ROADMAP-DONE.md now contains:
- M archived phases
```

Execute this cleanup now.
