package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSecurityFilter tests security filter initialization
func TestNewSecurityFilter(t *testing.T) {
	filter := NewSecurityFilter()

	assert.NotNil(t, filter)
	assert.NotNil(t, filter.emailPattern)
	assert.NotNil(t, filter.phonePattern)
	assert.NotNil(t, filter.ssnPattern)
	assert.NotNil(t, filter.creditCardPattern)
	assert.NotNil(t, filter.ipAddressPattern)
	assert.NotNil(t, filter.apiKeyPattern)
	assert.Equal(t, 10000, filter.maxFieldLength)
	assert.True(t, filter.maskingEnabled)
	assert.True(t, filter.truncationEnabled)
}

// TestNewSecurityFilterWithConfig tests custom configuration
func TestNewSecurityFilterWithConfig(t *testing.T) {
	config := SecurityFilterConfig{
		MaxFieldLength:    5000,
		MaskingEnabled:    false,
		TruncationEnabled: false,
	}

	filter := NewSecurityFilterWithConfig(config)

	assert.Equal(t, 5000, filter.maxFieldLength)
	assert.False(t, filter.maskingEnabled)
	assert.False(t, filter.truncationEnabled)
}

// TestFilterPrompt_EmailMasking tests email address masking
func TestFilterPrompt_EmailMasking(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple email",
			input:    "Contact me at john.doe@example.com for details",
			expected: "Contact me at j***e@example.com for details",
		},
		{
			name:     "short email",
			input:    "Email: ab@test.com",
			expected: "Email: ***@test.com",
		},
		{
			name:     "multiple emails",
			input:    "Emails: alice@foo.com and bob@bar.com",
			expected: "Emails: a***e@foo.com and b***b@bar.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterPrompt(tt.input)
			assert.Equal(t, tt.expected, result.FilteredContent)
			assert.Contains(t, result.Masked, "email")
			assert.False(t, result.Truncated)
		})
	}
}

// TestFilterPrompt_PhoneMasking tests phone number masking
func TestFilterPrompt_PhoneMasking(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "US phone with dashes",
			input: "Call me at 555-123-4567",
		},
		{
			name:  "US phone with parentheses",
			input: "Phone: (555) 123-4567",
		},
		{
			name:  "International phone",
			input: "Contact: +1-555-123-4567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterPrompt(tt.input)
			assert.Contains(t, result.FilteredContent, "[PHONE]")
			assert.Contains(t, result.Masked, "phone")
		})
	}
}

// TestFilterPrompt_SSNMasking tests SSN masking
func TestFilterPrompt_SSNMasking(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "SSN with dashes",
			input: "SSN: 123-45-6789",
		},
		{
			name:  "SSN with dashes in sentence",
			input: "My SSN is 987-65-4321 for reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterPrompt(tt.input)
			assert.Contains(t, result.FilteredContent, "[SSN]")
			assert.Contains(t, result.Masked, "ssn")
		})
	}
}

// TestFilterPrompt_CreditCardMasking tests credit card masking
func TestFilterPrompt_CreditCardMasking(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "credit card with dashes",
			input:    "Card: 1234-5678-9012-3456",
			expected: "Card: [CC-XXXX-3456]",
		},
		{
			name:     "credit card with spaces",
			input:    "Card: 1234 5678 9012 3456",
			expected: "Card: [CC-XXXX-3456]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterPrompt(tt.input)
			assert.Equal(t, tt.expected, result.FilteredContent)
			assert.Contains(t, result.Masked, "credit_card")
		})
	}
}

// TestFilterPrompt_APIKeyMasking tests API key masking
func TestFilterPrompt_APIKeyMasking(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "api_key parameter",
			input:    "Config: api_key=sk_test_1234567890abcdef",
			expected: "Config: api_key=[REDACTED]",
		},
		{
			name:     "token parameter",
			input:    "Auth: token=ghp_1234567890abcdefghijklmnop",
			expected: "Auth: token=[REDACTED]",
		},
		{
			name:     "basic token",
			input:    "token=ghp_abcdefghijklmnopqrstuvwxyz123456",
			expected: "token=[REDACTED]",
		},
		{
			name:     "password parameter",
			input:    "Credentials: password=MySecretP@ssw0rd",
			expected: "Credentials: password=[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterPrompt(tt.input)
			assert.Equal(t, tt.expected, result.FilteredContent)
			assert.Contains(t, result.Masked, "api_key")
		})
	}
}

// TestFilterPrompt_Truncation tests field truncation
func TestFilterPrompt_Truncation(t *testing.T) {
	config := SecurityFilterConfig{
		MaxFieldLength:    100,
		MaskingEnabled:    false,
		TruncationEnabled: true,
	}
	filter := NewSecurityFilterWithConfig(config)

	// Create long content
	longContent := strings.Repeat("This is a test sentence. ", 10) // ~250 chars

	result := filter.FilterPrompt(longContent)

	assert.True(t, result.Truncated)
	assert.Less(t, result.FilteredLength, result.OriginalLength)
	assert.Contains(t, result.FilteredContent, "[TRUNCATED]")
	assert.LessOrEqual(t, len(result.FilteredContent), 120) // Some margin for truncation message
}

// TestFilterPrompt_TruncationWordBoundary tests word boundary preservation
func TestFilterPrompt_TruncationWordBoundary(t *testing.T) {
	config := SecurityFilterConfig{
		MaxFieldLength:    50,
		MaskingEnabled:    false,
		TruncationEnabled: true,
	}
	filter := NewSecurityFilterWithConfig(config)

	input := "This is a test sentence with multiple words that should be truncated at word boundary"
	result := filter.FilterPrompt(input)

	assert.True(t, result.Truncated)
	assert.Contains(t, result.FilteredContent, "[TRUNCATED]")

	// Verify truncation happened and resulted content is shorter
	assert.Less(t, result.FilteredLength, result.OriginalLength)
}

// TestFilterPrompt_DisabledMasking tests masking disabled
func TestFilterPrompt_DisabledMasking(t *testing.T) {
	config := SecurityFilterConfig{
		MaxFieldLength:    10000,
		MaskingEnabled:    false,
		TruncationEnabled: false,
	}
	filter := NewSecurityFilterWithConfig(config)

	input := "Email: john@example.com, Phone: 555-123-4567"
	result := filter.FilterPrompt(input)

	assert.Equal(t, input, result.FilteredContent)
	assert.Empty(t, result.Masked)
	assert.False(t, result.Truncated)
}

// TestFilterPrompt_DisabledTruncation tests truncation disabled
func TestFilterPrompt_DisabledTruncation(t *testing.T) {
	config := SecurityFilterConfig{
		MaxFieldLength:    50,
		MaskingEnabled:    false,
		TruncationEnabled: false,
	}
	filter := NewSecurityFilterWithConfig(config)

	longContent := strings.Repeat("test ", 50) // Much longer than 50 chars
	result := filter.FilterPrompt(longContent)

	assert.Equal(t, longContent, result.FilteredContent)
	assert.False(t, result.Truncated)
}

// TestFilterPrompt_MultiplePIITypes tests multiple PII types in one prompt
func TestFilterPrompt_MultiplePIITypes(t *testing.T) {
	filter := NewSecurityFilter()

	input := `
		Contact Information:
		Email: john.doe@example.com
		Phone: 555-123-4567
		SSN: 123-45-6789
		Card: 1234-5678-9012-3456
		API Key: api_key=sk_test_abcdefghijklmnop
	`

	result := filter.FilterPrompt(input)

	// Check all PII types are masked
	assert.Contains(t, result.Masked, "email")
	assert.Contains(t, result.Masked, "phone")
	assert.Contains(t, result.Masked, "ssn")
	assert.Contains(t, result.Masked, "credit_card")
	assert.Contains(t, result.Masked, "api_key")

	// Check content is modified
	assert.NotEqual(t, input, result.FilteredContent)
	assert.Contains(t, result.FilteredContent, "[PHONE]")
	assert.Contains(t, result.FilteredContent, "[SSN]")
	assert.Contains(t, result.FilteredContent, "[CC-XXXX-3456]")
	assert.Contains(t, result.FilteredContent, "[REDACTED]")
}

// TestSanitizeJSON tests JSON sanitization
func TestSanitizeJSON(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password field",
			input:    `{"username": "john", "password": "secret123"}`,
			expected: `{"username": "john", "password": "[REDACTED]"}`,
		},
		{
			name:     "api_key field",
			input:    `{"api_key": "sk_test_12345"}`,
			expected: `{"api_key": "[REDACTED]"}`,
		},
		{
			name:     "token field",
			input:    `{"token": "abc123xyz"}`,
			expected: `{"token": "[REDACTED]"}`,
		},
		{
			name:     "multiple sensitive fields",
			input:    `{"password": "pass", "api_key": "key", "token": "tok"}`,
			expected: `{"password": "[REDACTED]", "api_key": "[REDACTED]", "token": "[REDACTED]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.SanitizeJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsContentSafe tests content safety checks
func TestIsContentSafe(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "safe content",
			input:    "This is a normal message about weather",
			expected: true,
		},
		{
			name:     "contains password keyword",
			input:    "Please enter your password",
			expected: false,
		},
		{
			name:     "contains api_key keyword",
			input:    "Configure your api_key in settings",
			expected: false,
		},
		{
			name:     "contains SSN pattern",
			input:    "SSN: 123-45-6789",
			expected: false,
		},
		{
			name:     "contains credit card pattern",
			input:    "Card: 1234-5678-9012-3456",
			expected: false,
		},
		{
			name:     "safe technical content",
			input:    "Run the build script and check logs",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.IsContentSafe(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFilterResult_Statistics tests result statistics
func TestFilterResult_Statistics(t *testing.T) {
	filter := NewSecurityFilter()

	input := "Email me at john.doe@example.com with your phone 555-123-4567"
	result := filter.FilterPrompt(input)

	// Check statistics
	assert.Equal(t, len(input), result.OriginalLength)
	assert.Greater(t, result.FilteredLength, 0)
	assert.Len(t, result.Masked, 2) // email and phone
	assert.False(t, result.Truncated)
}

// TestMaskEmail tests email masking edge cases
func TestMaskEmail(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long local part",
			input:    "john.doe@example.com",
			expected: "j***e@example.com",
		},
		{
			name:     "short local part",
			input:    "ab@test.com",
			expected: "***@test.com",
		},
		{
			name:     "single char local",
			input:    "a@test.com",
			expected: "***@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.maskEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTruncate tests truncation edge cases
func TestTruncate(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name      string
		input     string
		maxLength int
		check     func(t *testing.T, result string)
	}{
		{
			name:      "already short",
			input:     "Short text",
			maxLength: 50,
			check: func(t *testing.T, result string) {
				assert.Equal(t, "Short text", result)
			},
		},
		{
			name:      "truncate at word boundary",
			input:     "This is a long sentence that needs truncation",
			maxLength: 20,
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "[TRUNCATED]")
				assert.LessOrEqual(t, len(result), 35) // 20 + some margin for message
			},
		},
		{
			name:      "no good word boundary",
			input:     "verylongwordwithoutanyspacesorbreaks",
			maxLength: 20,
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "[TRUNCATED]")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.truncate(tt.input, tt.maxLength)
			tt.check(t, result)
		})
	}
}

// TestCaseInsensitiveAPIKeys tests case-insensitive API key detection
func TestCaseInsensitiveAPIKeys(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "lowercase api_key",
			input: "api_key=secret123",
		},
		{
			name:  "uppercase API_KEY",
			input: "API_KEY=secret123",
		},
		{
			name:  "mixed case Api_Key",
			input: "Api_Key=secret123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterPrompt(tt.input)
			assert.Contains(t, result.FilteredContent, "[REDACTED]")
			assert.Contains(t, result.Masked, "api_key")
		})
	}
}

// TestEmptyAndNilInputs tests edge cases with empty inputs
func TestEmptyAndNilInputs(t *testing.T) {
	filter := NewSecurityFilter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "whitespace only",
			input: "   \n\t  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterPrompt(tt.input)
			assert.Equal(t, tt.input, result.FilteredContent)
			assert.Empty(t, result.Masked)
			assert.False(t, result.Truncated)
		})
	}
}

// TestRealWorldScenarios tests realistic use cases
func TestRealWorldScenarios(t *testing.T) {
	filter := NewSecurityFilter()

	t.Run("user support request", func(t *testing.T) {
		input := `
			Hi, I'm having trouble logging in.
			My email is john.doe@company.com
			Account ID: 123-45-6789
			Please help!
		`
		result := filter.FilterPrompt(input)

		assert.Contains(t, result.FilteredContent, "j***e@company.com")
		assert.Contains(t, result.FilteredContent, "[SSN]")
		assert.Contains(t, result.Masked, "email")
		assert.Contains(t, result.Masked, "ssn")
	})

	t.Run("API configuration", func(t *testing.T) {
		input := `
			Configure your application:
			api_key=sk_live_1234567890abcdef
			token=ghp_abcdefghijklmnopqrstuvwxyz
		`
		result := filter.FilterPrompt(input)

		assert.Contains(t, result.FilteredContent, "api_key=[REDACTED]")
		assert.Contains(t, result.FilteredContent, "token=[REDACTED]")
		assert.Contains(t, result.Masked, "api_key")
	})

	t.Run("payment information", func(t *testing.T) {
		input := "Please charge card 4532-1234-5678-9010 for the subscription"
		result := filter.FilterPrompt(input)

		assert.Contains(t, result.FilteredContent, "[CC-XXXX-9010]")
		assert.Contains(t, result.Masked, "credit_card")
	})
}

// TestDefaultSecurityFilterConfig tests default configuration
func TestDefaultSecurityFilterConfig(t *testing.T) {
	config := DefaultSecurityFilterConfig()

	assert.Equal(t, 10000, config.MaxFieldLength)
	assert.True(t, config.MaskingEnabled)
	assert.True(t, config.TruncationEnabled)
}

// TestConcurrentFiltering tests thread safety
func TestConcurrentFiltering(t *testing.T) {
	filter := NewSecurityFilter()

	inputs := []string{
		"Email: test1@example.com",
		"Phone: 555-123-4567",
		"SSN: 123-45-6789",
		"Card: 1234-5678-9012-3456",
		"api_key=secret123",
	}

	done := make(chan bool, len(inputs))

	// Run concurrent filtering
	for _, input := range inputs {
		go func(text string) {
			defer func() { done <- true }()
			result := filter.FilterPrompt(text)
			require.NotEmpty(t, result.FilteredContent)
		}(input)
	}

	// Wait for all goroutines
	for i := 0; i < len(inputs); i++ {
		<-done
	}
}
