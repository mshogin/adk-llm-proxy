package services

import (
	"regexp"
	"strings"
)

// SecurityFilter provides PII masking and sensitive field truncation for LLM requests.
type SecurityFilter struct {
	// PII patterns
	emailPattern       *regexp.Regexp
	phonePattern       *regexp.Regexp
	ssnPattern         *regexp.Regexp
	creditCardPattern  *regexp.Regexp
	ipAddressPattern   *regexp.Regexp
	apiKeyPattern      *regexp.Regexp

	// Configuration
	maxFieldLength     int
	maskingEnabled     bool
	truncationEnabled  bool
}

// SecurityFilterConfig configures the security filter.
type SecurityFilterConfig struct {
	MaxFieldLength    int  `json:"max_field_length"`    // Maximum allowed field length before truncation
	MaskingEnabled    bool `json:"masking_enabled"`     // Enable PII masking
	TruncationEnabled bool `json:"truncation_enabled"`  // Enable field truncation
}

// FilterResult contains the filtered content and detected issues.
type FilterResult struct {
	FilteredContent string
	Masked          []string // List of PII types that were masked
	Truncated       bool     // Whether content was truncated
	OriginalLength  int      // Original content length
	FilteredLength  int      // Filtered content length
}

// NewSecurityFilter creates a new security filter with default configuration.
func NewSecurityFilter() *SecurityFilter {
	return NewSecurityFilterWithConfig(DefaultSecurityFilterConfig())
}

// NewSecurityFilterWithConfig creates a security filter with custom configuration.
func NewSecurityFilterWithConfig(config SecurityFilterConfig) *SecurityFilter {
	return &SecurityFilter{
		// Email pattern: basic email validation
		emailPattern: regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),

		// Phone patterns: US and international formats
		phonePattern: regexp.MustCompile(`(\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`),

		// SSN pattern: XXX-XX-XXXX (with dashes only, to avoid false positives)
		ssnPattern: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),

		// Credit card pattern: 13-19 digits with optional spaces/dashes (only with separators)
		creditCardPattern: regexp.MustCompile(`\b(?:\d{4}[-\s]){3}\d{4}\b`),

		// IP address pattern: IPv4
		ipAddressPattern: regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),

		// API key patterns: common formats (hex, base64-like)
		apiKeyPattern: regexp.MustCompile(`\b[A-Za-z0-9_-]{32,}\b`),

		maxFieldLength:    config.MaxFieldLength,
		maskingEnabled:    config.MaskingEnabled,
		truncationEnabled: config.TruncationEnabled,
	}
}

// DefaultSecurityFilterConfig returns default security filter configuration.
func DefaultSecurityFilterConfig() SecurityFilterConfig {
	return SecurityFilterConfig{
		MaxFieldLength:    10000, // 10K characters max
		MaskingEnabled:    true,
		TruncationEnabled: true,
	}
}

// FilterPrompt filters a prompt for PII and truncates if necessary.
func (f *SecurityFilter) FilterPrompt(prompt string) FilterResult {
	result := FilterResult{
		FilteredContent: prompt,
		Masked:          []string{},
		Truncated:       false,
		OriginalLength:  len(prompt),
	}

	// Apply PII masking if enabled
	if f.maskingEnabled {
		result.FilteredContent, result.Masked = f.maskPII(result.FilteredContent)
	}

	// Apply truncation if enabled
	if f.truncationEnabled && len(result.FilteredContent) > f.maxFieldLength {
		result.FilteredContent = f.truncate(result.FilteredContent, f.maxFieldLength)
		result.Truncated = true
	}

	result.FilteredLength = len(result.FilteredContent)
	return result
}

// maskPII masks personally identifiable information in the content.
func (f *SecurityFilter) maskPII(content string) (string, []string) {
	masked := []string{}

	// Mask emails
	if f.emailPattern.MatchString(content) {
		content = f.emailPattern.ReplaceAllStringFunc(content, func(email string) string {
			return f.maskEmail(email)
		})
		masked = append(masked, "email")
	}

	// Mask phone numbers
	if f.phonePattern.MatchString(content) {
		content = f.phonePattern.ReplaceAllString(content, "[PHONE]")
		masked = append(masked, "phone")
	}

	// Mask SSN
	if f.ssnPattern.MatchString(content) {
		content = f.ssnPattern.ReplaceAllString(content, "[SSN]")
		masked = append(masked, "ssn")
	}

	// Mask credit cards
	if f.creditCardPattern.MatchString(content) {
		content = f.creditCardPattern.ReplaceAllStringFunc(content, func(cc string) string {
			// Keep last 4 digits for reference
			digits := strings.ReplaceAll(strings.ReplaceAll(cc, "-", ""), " ", "")
			if len(digits) >= 4 {
				return "[CC-XXXX-" + digits[len(digits)-4:] + "]"
			}
			return "[CC]"
		})
		masked = append(masked, "credit_card")
	}

	// Mask IP addresses (optional - might be needed for debugging)
	// Uncomment if needed:
	// if f.ipAddressPattern.MatchString(content) {
	//     content = f.ipAddressPattern.ReplaceAllString(content, "[IP]")
	//     masked = append(masked, "ip_address")
	// }

	// Mask potential API keys (be conservative to avoid false positives)
	content = f.maskPotentialAPIKeys(content, &masked)

	return content, masked
}

// maskEmail masks an email address, preserving domain.
func (f *SecurityFilter) maskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "[EMAIL]"
	}

	localPart := parts[0]
	domain := parts[1]

	// Keep first and last character of local part if long enough
	if len(localPart) > 2 {
		return string(localPart[0]) + "***" + string(localPart[len(localPart)-1]) + "@" + domain
	}

	return "***@" + domain
}

// maskPotentialAPIKeys masks strings that look like API keys.
func (f *SecurityFilter) maskPotentialAPIKeys(content string, masked *[]string) string {
	// Look for common API key prefixes (order matters - more specific first)
	apiKeyPrefixes := []string{
		"Authorization: Bearer ",
		"Authorization: ",
		"api_key=", "apikey=", "api-key=",
		"access_token=",
		"token=",
		"api_secret=",
		"secret=",
		"password=", "passwd=", "pwd=",
	}

	modified := false
	processedPrefixes := make(map[string]bool)

	for _, prefix := range apiKeyPrefixes {
		lowerPrefix := strings.ToLower(prefix)

		// Skip if a longer prefix containing this was already processed
		skip := false
		for processed := range processedPrefixes {
			if strings.Contains(lowerPrefix, processed) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, lowerPrefix) {
			// Find and mask the value after the prefix (including special chars like @)
			pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(prefix) + `[^\s]+`)
			if pattern.MatchString(content) {
				content = pattern.ReplaceAllString(content, prefix+"[REDACTED]")
				processedPrefixes[lowerPrefix] = true
				modified = true
			}
		}
	}

	if modified {
		*masked = append(*masked, "api_key")
	}

	return content
}

// truncate truncates content to maxLength, preserving word boundaries.
func (f *SecurityFilter) truncate(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// Truncate at word boundary if possible
	truncated := content[:maxLength]

	// Find last space to avoid cutting words
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLength/2 { // Only use word boundary if it's not too far back
		truncated = truncated[:lastSpace]
	}

	return truncated + "... [TRUNCATED]"
}

// SanitizeJSON removes or masks sensitive fields from JSON-like content.
func (f *SecurityFilter) SanitizeJSON(content string) string {
	// Sensitive field patterns to redact
	sensitiveFields := []string{
		`"password"\s*:\s*"[^"]*"`,
		`"api_key"\s*:\s*"[^"]*"`,
		`"apiKey"\s*:\s*"[^"]*"`,
		`"token"\s*:\s*"[^"]*"`,
		`"access_token"\s*:\s*"[^"]*"`,
		`"secret"\s*:\s*"[^"]*"`,
		`"private_key"\s*:\s*"[^"]*"`,
		`"privateKey"\s*:\s*"[^"]*"`,
	}

	result := content
	for _, pattern := range sensitiveFields {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			// Extract field name
			fieldName := strings.Split(match, ":")[0]
			return fieldName + `: "[REDACTED]"`
		})
	}

	return result
}

// IsContentSafe performs a quick check if content contains obvious sensitive data.
func (f *SecurityFilter) IsContentSafe(content string) bool {
	// Quick checks for obvious sensitive patterns
	lowerContent := strings.ToLower(content)

	// Check for explicit sensitive markers
	sensitiveMarkers := []string{
		"password", "secret", "api_key", "apikey",
		"private_key", "access_token", "ssn", "social security",
	}

	for _, marker := range sensitiveMarkers {
		if strings.Contains(lowerContent, marker) {
			// Might contain sensitive data
			return false
		}
	}

	// Check for patterns
	if f.ssnPattern.MatchString(content) ||
		f.creditCardPattern.MatchString(content) {
		return false
	}

	return true
}

// GetStats returns statistics about filtering operations.
type FilterStats struct {
	TotalFiltered   int64            `json:"total_filtered"`
	MaskedByType    map[string]int64 `json:"masked_by_type"`
	TruncatedCount  int64            `json:"truncated_count"`
	AvgReduction    float64          `json:"avg_reduction_pct"`
}
