---
description: Add a new task to the project roadmap with analysis and planning
---

Add a new task to the ROADMAP.md by analyzing it, decomposing it into smaller subtasks, and creating implementation plans.

Ask the user for:
1. Task description (what needs to be implemented)
2. Phase number or name (which phase to add it to, or "new" for a new phase)
3. Priority level (optional: high/medium/low)

Then:
1. **Analyze the task**:
   - Understand the scope and requirements
   - Identify dependencies on existing components
   - Determine which layers/files will be affected (domain/application/infrastructure/presentation)
   - Assess complexity and effort

2. **Decompose into subtasks**:
   - Break down the task into 3-8 smaller, concrete subtasks
   - Each subtask should be a discrete, implementable unit
   - Order subtasks logically (dependencies first)
   - Follow DDD architecture principles (proper layer placement)

3. **Create implementation plan for each subtask**:
   - Identify specific files to create or modify
   - List required dependencies or libraries
   - Note configuration changes needed
   - Specify test files to create/update
   - Include validation steps

4. **Add to ROADMAP.md**:
   - If phase exists: add to appropriate section
   - If new phase: create new phase section with goal
   - Use checkbox format: `- [ ] Subtask description`
   - Add implementation notes if needed
   - Maintain consistent formatting with existing roadmap

5. **Present summary**:
   - Show the decomposed task structure
   - Highlight key implementation steps
   - Note any risks or blockers identified
   - Suggest next steps

Follow these guidelines:
- Use proper DDD layer structure (see .claude/CLAUDE.md)
- Place tests in tests/ directory mirroring src/ structure
- Prefer Python for AI/ML/MCP features, consider Golang for performance-critical services
- Include configuration updates in config.yaml when needed
- Add MCP server tasks under mcps/ when creating new integrations
- Ensure async/await patterns for I/O operations
- Include proper error handling and type hints in implementation notes

Example task decomposition:
```
Task: "Add caching support for MCP tool results"

Analysis:
- Affects: infrastructure layer (caching), application layer (orchestration)
- Dependencies: Redis or in-memory cache library
- Complexity: Medium

Subtasks:
1. Add cache configuration to config.yaml
2. Create cache client in src/infrastructure/cache/cache_client.py
3. Update orchestration service to use cache before tool calls
4. Add cache invalidation logic
5. Create tests in tests/infrastructure/cache/
6. Update documentation with caching behavior

Implementation notes:
- Use Redis for distributed caching or cachetools for local
- Add TTL configuration per tool type
- Implement cache key generation based on tool + params
- Add cache metrics to monitoring
```
