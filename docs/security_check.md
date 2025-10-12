# Security Check System

Simple, unified approach to prevent sensitive data from going to GitHub.

## Overview

**Two tools, one workflow:**

1. **Pattern-based check** (automatic) - Fast scan for obvious secrets
2. **Claude Code analysis** (manual) - Deep, context-aware review

## Quick Start

### Automatic Check (Pre-commit Hook)

Already installed! Every commit is automatically checked for:
- API keys (AWS, OpenAI, GitHub, GitLab, Slack, YouTrack)
- Private keys
- Database credentials
- OAuth tokens

```bash
# Automatic on every commit
git commit -m "Your message"
```

### Manual Deep Check (Before GitHub Push)

Before pushing to public repository:

```
/seccheck
```

This runs intelligent analysis that understands:
- Context (documentation vs real code)
- False positives (test keys, examples)
- Company-specific patterns (internal URLs)

## Configuration

Edit `.security-check.yaml`:

```yaml
deep_check:
  # Show reminder to run /seccheck
  enabled: false  # Set to true to get reminders
```

## Files

- `.security-check.yaml` - Configuration (patterns, whitelist)
- `scripts/security_check.sh` - Pattern scanner (used by pre-commit)
- `scripts/git-hooks/pre-commit` - Git hook
- `.claude/commands/seccheck.md` - Claude Code command definition

## Workflow

### Regular Commits
```bash
git add .
git commit -m "feat: add feature"  # Automatic pattern check
git push
```

### Important Commits (config changes, new integrations)
```bash
git add .
git commit -m "feat: add feature"  # Automatic pattern check
/seccheck                          # Manual deep check
git push
```

## What Gets Checked

### Pattern-Based (Automatic)
- `AKIA[0-9A-Z]{16}` - AWS keys
- `sk-[a-zA-Z0-9]{48,}` - OpenAI keys
- `ghp_[a-zA-Z0-9]{36}` - GitHub tokens
- `glpat-[a-zA-Z0-9_-]{20}` - GitLab tokens
- `perm:[a-zA-Z0-9=.]+` - YouTrack tokens
- `-----BEGIN.*PRIVATE KEY-----` - Private keys

### Claude Code Analysis (Manual)
- **Critical**: API keys, private keys, credentials
- **High**: Personal data, internal URLs, session tokens
- **Medium**: Internal IPs, hostnames, config files

## Excluding Files

Automatically excluded:
- `*.example`, `*.template`, `*.sample`
- `*.md` (documentation)
- `.claude/` directory
- Test files (`test_*.py`, `*.test.js`)

## Bypassing (NOT Recommended)

```bash
git commit --no-verify  # Skip pre-commit hook
```

Only use if you're certain the flagged items are false positives.

## Troubleshooting

### Pre-commit hook not running
```bash
./scripts/install-hooks.sh
```

### False positive in pattern check
Add to `.security-check.yaml` whitelist:
```yaml
whitelist:
  - value: "your-safe-value"
    reason: "Why it's safe"
```

### Need to check without committing
```bash
./scripts/security_check.sh
```

## Maintenance

**One script, one command:**
- Modify patterns: Edit `scripts/security_check.sh`
- Update deep analysis: Edit `.claude/commands/seccheck.md`

No redundant scripts or commands to maintain!
