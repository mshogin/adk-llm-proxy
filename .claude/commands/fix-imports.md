---
description: Fix common import errors in the project
---

Diagnose and fix common Python import errors in the project.

Steps:
1. Use Grep to search for common import patterns that might fail
2. Check if __init__.py files exist in all package directories
3. Verify relative vs absolute imports are correct
4. Check for circular import issues
5. Ensure all imports follow the project structure:
   - src.domain.* for domain logic
   - src.application.* for business logic
   - src.infrastructure.* for external integrations
   - src.presentation.* for web layer

Report findings and offer to fix any issues detected.
