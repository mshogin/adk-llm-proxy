---
description: Deep security check of staged and unstaged changes
---

# Security Check - Deep Analysis with Claude Code

Analyze both staged and unstaged git changes for sensitive data leaks.

## Task

1. **Get staged changes**:
   ```bash
   git diff --cached
   ```

2. **Get unstaged changes**:
   ```bash
   git diff
   ```

3. **Analyze changes** for:

   **CHECK CATEGORIES:**

   a) **Secrets and tokens** (CRITICAL/HIGH):
      - API keys: AWS (AKIA*), OpenAI (sk-*), GitHub (ghp_*), GitLab (glpat-*), Slack (xoxb-*), etc.
      - OAuth tokens, Bearer tokens
      - JWT tokens (eyJ.*)
      - Plaintext passwords
      - YouTrack tokens (perm:*)

   b) **Cryptographic keys** (CRITICAL):
      - SSH private keys (-----BEGIN PRIVATE KEY-----)
      - PGP keys
      - SSL certificates with private keys

   c) **Credentials** (CRITICAL/HIGH):
      - Database connection strings with passwords
      - Hardcoded credentials
      - .env files with real secrets
      - config.yaml with real API keys

   d) **Personal data** (HIGH/MEDIUM):
      - Email addresses (not example.com, not noreply@*)
      - Phone numbers
      - Personal identification data

   e) **Corporate data** (MEDIUM):
      - Internal IP addresses (10.*, 172.16-31.*, 192.168.*)
      - Internal hostnames (*.wildberries.*)
      - Server names

4. **IMPORTANT RULES:**
   - ✅ Ignore: example.com, test@*, localhost, sk-test-*, REPLACE_WITH_*, "your-api-key-here"
   - ✅ Ignore: files *.example, *.template, *.sample, *.test.*, *.md (documentation)
   - ✅ Pay attention to context (documentation vs real code)
   - ⚠️ Flag: real API keys, production credentials, real email addresses
   - 🔴 MUST flag: private keys, database URLs with passwords, real tokens in config.yaml

5. **Output format:**

   ```
   🔒 SECURITY CHECK RESULTS
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   [If no issues]
   ✓ No security issues detected!
   Checked N lines of changes.

   [If issues found]
   ✗ SECURITY ISSUES DETECTED!

   Found: X issues

   [CRITICAL] category
   📁 file.py:42
   ❌ AWS API key detected
   💡 Remove key, use environment variables
   📝 API_KEY = "AKIA1234567890ABCDEF"

   [HIGH] category
   📁 config.yaml:6
   ❌ OpenAI API key in config file
   💡 This file should be in .gitignore (it already is, use git rm --cached)
   📝 api_key: "sk-svcacct-..."

   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   [If issues found]
   ❌ DO NOT COMMIT these changes

   What to do:
   1. Fix the issues found
   2. Use environment variables for secrets
   3. Add sensitive files to .gitignore
   4. Create .example files with placeholders
   5. For already tracked files: git rm --cached <file>

   For false positives:
   - Add value to whitelist: .security-check.yaml
   - Or use: git commit --no-verify (NOT RECOMMENDED)
   ```

6. **Exit code**:
   - If CRITICAL/HIGH issues - recommend blocking commit
   - If only MEDIUM/LOW - warn but don't block
   - If no issues - approve commit

## Additional

- Be careful with context: documentation vs real code
- Give specific recommendations for each issue
- If unsure - better to warn
- Remember false positives: test keys, examples in docs

START THE CHECK NOW!
