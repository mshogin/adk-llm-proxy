#!/bin/bash
# Security Check Script for Git Commits - ADK LLM Proxy
# Fast pattern-based check for obvious secrets

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/.security-check.yaml"
LOG_DIR="$PROJECT_ROOT/logs/security"
LOG_FILE="$LOG_DIR/security-check.log"
TEMP_DIR=$(mktemp -d)

# Create log directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Cleanup on exit
cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Logging functions
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" >> "$LOG_FILE"
}

log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
    log "INFO: $*"
}

log_success() {
    echo -e "${GREEN}✓${NC} $*"
    log "SUCCESS: $*"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $*"
    log "WARNING: $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*"
    log "ERROR: $*"
}

# Check if security check is enabled
check_enabled() {
    if [ ! -f "$CONFIG_FILE" ]; then
        log_warning "Configuration file not found: $CONFIG_FILE"
        log_warning "Security check will run with default settings"
        return 0
    fi

    # Check if pattern check is disabled
    if grep -q "^pattern_check_enabled: false" "$CONFIG_FILE"; then
        log_info "Security check is disabled in configuration"
        exit 0
    fi
}

# Get staged changes
get_staged_diff() {
    local diff_file="$1"

    if ! git diff --cached --diff-filter=d > "$diff_file"; then
        log_error "Failed to get staged changes"
        exit 1
    fi

    if [ ! -s "$diff_file" ]; then
        log_info "No staged changes to check"
        return 1
    fi

    local lines=$(wc -l < "$diff_file" | tr -d ' ')
    log_info "Got $lines lines of staged changes to analyze"
    return 0
}

# Get unstaged changes
get_unstaged_diff() {
    local diff_file="$1"

    if ! git diff --diff-filter=d > "$diff_file"; then
        log_error "Failed to get unstaged changes"
        exit 1
    fi

    if [ ! -s "$diff_file" ]; then
        return 1
    fi

    local lines=$(wc -l < "$diff_file" | tr -d ' ')
    log_info "Got $lines lines of unstaged changes to analyze"
    return 0
}

# Analyze changes using simple pattern matching
simple_security_check() {
    local diff_file="$1"
    local issues_found=0

    log_info "Running pattern-based security check..."

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  🔒 SECURITY CHECK RESULTS"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Check for AWS keys
    if grep -E "AKIA[0-9A-Z]{16}" "$diff_file" > /dev/null 2>&1; then
        local matches=$(grep -E "AKIA[0-9A-Z]{16}" "$diff_file" | grep -vE "\.example|\.template|\.sample|\.md|\.claude/" | grep -vE "your-.*-here|REPLACE_WITH_|example\.com|AKIA1234567890" || true)
        if [ -n "$matches" ]; then
            log_error "Found potential AWS_KEY"
            echo -e "  ${RED}[CRITICAL]${NC} Potential AWS API Key detected"
            echo ""
            echo "  Sample matches (first 3 lines):"
            echo "$matches" | head -3 | sed 's/^/    /'
            echo ""
            issues_found=$((issues_found + 1))
        fi
    fi

    # Check for OpenAI keys
    if grep -E "sk-[a-zA-Z0-9]{48,}" "$diff_file" > /dev/null 2>&1; then
        local matches=$(grep -E "sk-[a-zA-Z0-9]{48,}" "$diff_file" | grep -vE "\.example|\.template|\.sample|\.md|\.claude/" | grep -vE "sk-test-|your-.*-here|REPLACE_WITH_" || true)
        if [ -n "$matches" ]; then
            log_error "Found potential OPENAI_KEY"
            echo -e "  ${RED}[CRITICAL]${NC} Potential OpenAI API Key detected"
            echo ""
            echo "  Sample matches (first 3 lines):"
            echo "$matches" | head -3 | sed 's/^/    /'
            echo ""
            issues_found=$((issues_found + 1))
        fi
    fi

    # Check for GitHub tokens
    if grep -E "ghp_[a-zA-Z0-9]{36}" "$diff_file" > /dev/null 2>&1; then
        local matches=$(grep -E "ghp_[a-zA-Z0-9]{36}" "$diff_file" | grep -vE "\.example|\.template|\.sample|\.md|\.claude/" || true)
        if [ -n "$matches" ]; then
            log_error "Found potential GITHUB_TOKEN"
            echo -e "  ${RED}[CRITICAL]${NC} Potential GitHub Token detected"
            echo ""
            echo "  Sample matches (first 3 lines):"
            echo "$matches" | head -3 | sed 's/^/    /'
            echo ""
            issues_found=$((issues_found + 1))
        fi
    fi

    # Check for GitLab tokens
    if grep -E "glpat-[a-zA-Z0-9_-]{20}" "$diff_file" > /dev/null 2>&1; then
        local matches=$(grep -E "glpat-[a-zA-Z0-9_-]{20}" "$diff_file" | grep -vE "\.example|\.template|\.sample|\.md|\.claude/" || true)
        if [ -n "$matches" ]; then
            log_error "Found potential GITLAB_TOKEN"
            echo -e "  ${RED}[CRITICAL]${NC} Potential GitLab Token detected"
            echo ""
            echo "  Sample matches (first 3 lines):"
            echo "$matches" | head -3 | sed 's/^/    /'
            echo ""
            issues_found=$((issues_found + 1))
        fi
    fi

    # Check for private keys
    if grep -E "-----BEGIN.*PRIVATE KEY-----" "$diff_file" > /dev/null 2>&1; then
        local matches=$(grep -E "-----BEGIN.*PRIVATE KEY-----" "$diff_file" | grep -vE "\.example|\.template|\.sample|\.md|\.claude/" || true)
        if [ -n "$matches" ]; then
            log_error "Found potential PRIVATE_KEY"
            echo -e "  ${RED}[CRITICAL]${NC} Potential Private Key detected"
            echo ""
            echo "  Sample matches (first 3 lines):"
            echo "$matches" | head -3 | sed 's/^/    /'
            echo ""
            issues_found=$((issues_found + 1))
        fi
    fi

    # Check for YouTrack tokens
    if grep -E "perm:[a-zA-Z0-9=.]+" "$diff_file" > /dev/null 2>&1; then
        local matches=$(grep -E "perm:[a-zA-Z0-9=.]+" "$diff_file" | grep -vE "\.example|\.template|\.sample|\.md|\.claude/" || true)
        if [ -n "$matches" ]; then
            log_error "Found potential YOUTRACK_TOKEN"
            echo -e "  ${RED}[CRITICAL]${NC} Potential YouTrack Token detected"
            echo ""
            echo "  Sample matches (first 3 lines):"
            echo "$matches" | head -3 | sed 's/^/    /'
            echo ""
            issues_found=$((issues_found + 1))
        fi
    fi

    if [ $issues_found -eq 0 ]; then
        log_success "No obvious security issues detected by pattern matching!"
        echo "  ✓ No obvious security issues detected by pattern matching"
        echo ""
        echo "  Note: This is a basic check. For deep analysis, use: /seccheck"
        echo ""
        return 0
    else
        echo "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        log_error "Found $issues_found potential security issue(s)"
        echo -e "  ${RED}❌ FOUND $issues_found POTENTIAL SECURITY ISSUE(S)${NC}"
        echo ""
        echo "  What to do:"
        echo "  1. Review the flagged items above"
        echo "  2. Use environment variables for secrets"
        echo "  3. Add sensitive files to .gitignore"
        echo "  4. For already tracked files: git rm --cached <file>"
        echo ""
        return 1
    fi
}

# Check if deep analysis is recommended
check_deep_analysis_config() {
    # Check if deep_check.enabled is true in config
    if [ ! -f "$CONFIG_FILE" ]; then
        return 1
    fi

    # Simple check for deep_check enabled (look for the setting in the deep_check section)
    if grep -A 2 "^deep_check:" "$CONFIG_FILE" | grep -q "enabled: true"; then
        return 0
    fi
    return 1
}

# Main function
main() {
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Security Check Started"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Check if enabled
    check_enabled

    # Files for work
    local staged_diff="$TEMP_DIR/staged.diff"
    local unstaged_diff="$TEMP_DIR/unstaged.diff"
    local combined_diff="$TEMP_DIR/combined.diff"

    # Get changes
    local has_staged=0
    local has_unstaged=0

    if get_staged_diff "$staged_diff"; then
        has_staged=1
    fi

    if get_unstaged_diff "$unstaged_diff"; then
        has_unstaged=1
    fi

    # Combine diffs
    if [ $has_staged -eq 1 ] && [ $has_unstaged -eq 1 ]; then
        cat "$staged_diff" "$unstaged_diff" > "$combined_diff"
        log_info "Checking both staged and unstaged changes"
    elif [ $has_staged -eq 1 ]; then
        cp "$staged_diff" "$combined_diff"
        log_info "Checking staged changes only"
    elif [ $has_unstaged -eq 1 ]; then
        cp "$unstaged_diff" "$combined_diff"
        log_info "Checking unstaged changes only"
    else
        log_info "No changes to check"
        exit 0
    fi

    # Run pattern-based security check
    if ! simple_security_check "$combined_diff"; then
        log_error "Pattern-based security check failed"
        exit 1
    fi

    # Check if deep analysis is recommended (for important commits)
    if check_deep_analysis_config; then
        echo ""
        log_warning "⚠️  Deep security check is recommended for this project"
        echo ""
        echo "  Before pushing to GitHub, run:"
        echo "  /seccheck"
        echo ""
        echo "  This will perform Claude Code analysis for:"
        echo "  - Context-aware secret detection"
        echo "  - Personal data review"
        echo "  - Corporate information check"
        echo ""
    fi

    log_success "Security check passed"
    exit 0
}

# Run
main "$@"
