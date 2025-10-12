---
description: Run the project test suite
---

Run the pytest test suite for the project.

Use the Bash tool to run:
```bash
pytest tests/ -v
```

If you want to run with coverage:
```bash
pytest tests/ -v --cov=src --cov-report=term-missing
```

After tests complete:
1. Report the results (passed/failed)
2. Highlight any failures with details
3. Suggest fixes for common test failures
4. If all pass, congratulate and show coverage stats
