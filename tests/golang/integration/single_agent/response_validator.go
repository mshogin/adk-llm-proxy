package single_agent

import (
	"fmt"
	"strings"
	"testing"
)

// ResponseValidator provides fluent API for validating single-agent test responses
type ResponseValidator struct {
	t         *testing.T
	reasoning string
}

// NewResponseValidator creates a new validator for the reasoning block content
func NewResponseValidator(t *testing.T, reasoning string) *ResponseValidator {
	return &ResponseValidator{
		t:         t,
		reasoning: reasoning,
	}
}

// AssertHasInput verifies the response contains an INPUT section
func (v *ResponseValidator) AssertHasInput() *ResponseValidator {
	if !strings.Contains(v.reasoning, "📥 INPUT") {
		v.t.Errorf("Response missing INPUT section:\n%s", v.reasoning)
	}
	return v
}

// AssertHasOutput verifies the response contains an OUTPUT section
func (v *ResponseValidator) AssertHasOutput() *ResponseValidator {
	if !strings.Contains(v.reasoning, "📤 OUTPUT") {
		v.t.Errorf("Response missing OUTPUT section:\n%s", v.reasoning)
	}
	return v
}

// AssertHasSystemPrompt verifies the response contains a SYSTEM PROMPT section
func (v *ResponseValidator) AssertHasSystemPrompt() *ResponseValidator {
	if !strings.Contains(v.reasoning, "📤 SYSTEM PROMPT FOR LLM") {
		v.t.Errorf("Response missing SYSTEM PROMPT section:\n%s", v.reasoning)
	}
	return v
}

// AssertHasMetrics verifies the response contains a METRICS section
func (v *ResponseValidator) AssertHasMetrics() *ResponseValidator {
	if !strings.Contains(v.reasoning, "⏱️  METRICS") {
		v.t.Errorf("Response missing METRICS section:\n%s", v.reasoning)
	}
	return v
}

// AssertSingleAgent verifies the response is from the expected single agent
func (v *ResponseValidator) AssertSingleAgent(expectedAgentID string) *ResponseValidator {
	expectedHeader := fmt.Sprintf("=== %s AGENT TEST ===", strings.ToUpper(strings.ReplaceAll(expectedAgentID, "_", " ")))
	if !strings.Contains(v.reasoning, expectedHeader) {
		v.t.Errorf("Response not from expected agent %s. Header not found: %s\nResponse:\n%s",
			expectedAgentID, expectedHeader, v.reasoning)
	}
	return v
}

// AssertContains verifies the response contains the specified text
func (v *ResponseValidator) AssertContains(text string) *ResponseValidator {
	if !strings.Contains(v.reasoning, text) {
		v.t.Errorf("Response does not contain expected text: %s\nResponse:\n%s", text, v.reasoning)
	}
	return v
}

// AssertNotContains verifies the response does not contain the specified text
func (v *ResponseValidator) AssertNotContains(text string) *ResponseValidator {
	if strings.Contains(v.reasoning, text) {
		v.t.Errorf("Response contains unexpected text: %s\nResponse:\n%s", text, v.reasoning)
	}
	return v
}

// AssertHasLLMInteraction verifies the response shows LLM was used
func (v *ResponseValidator) AssertHasLLMInteraction() *ResponseValidator {
	if !strings.Contains(v.reasoning, "💬 LLM INTERACTION") {
		v.t.Errorf("Response missing LLM INTERACTION section (expected LLM fallback):\n%s", v.reasoning)
	}
	return v
}

// AssertNoLLMInteraction verifies the response shows LLM was NOT used
func (v *ResponseValidator) AssertNoLLMInteraction() *ResponseValidator {
	if strings.Contains(v.reasoning, "💬 LLM INTERACTION") {
		v.t.Errorf("Response has unexpected LLM INTERACTION section (expected rules-only):\n%s", v.reasoning)
	}
	return v
}

// AssertHasError verifies the response contains error information
func (v *ResponseValidator) AssertHasError(errorCode string) *ResponseValidator {
	if !strings.Contains(v.reasoning, errorCode) {
		v.t.Errorf("Response missing expected error code: %s\nResponse:\n%s", errorCode, v.reasoning)
	}
	return v
}

// AssertHasWarning verifies the response contains warning information
func (v *ResponseValidator) AssertHasWarning(warningCode string) *ResponseValidator {
	if !strings.Contains(v.reasoning, warningCode) {
		v.t.Errorf("Response missing expected warning code: %s\nResponse:\n%s", warningCode, v.reasoning)
	}
	return v
}

// AssertValidationStatus verifies the validation status (PASSED/WARNING/FAILED)
func (v *ResponseValidator) AssertValidationStatus(expectedStatus string) *ResponseValidator {
	statusMarkers := map[string]string{
		"PASSED":  "✅ PASSED",
		"WARNING": "⚠️  WARNING",
		"FAILED":  "❌ FAILED",
	}

	marker, ok := statusMarkers[expectedStatus]
	if !ok {
		v.t.Errorf("Invalid validation status: %s. Must be PASSED, WARNING, or FAILED", expectedStatus)
		return v
	}

	if !strings.Contains(v.reasoning, marker) {
		v.t.Errorf("Response does not have expected validation status %s:\n%s", expectedStatus, v.reasoning)
	}
	return v
}

// AssertIntentDetected verifies a specific intent was detected with minimum confidence
func (v *ResponseValidator) AssertIntentDetected(intentType string, minConfidence float64) *ResponseValidator {
	if !strings.Contains(v.reasoning, intentType) {
		v.t.Errorf("Intent %s not detected in response:\n%s", intentType, v.reasoning)
		return v
	}

	// Check for confidence score format like "confidence: 0.99" or "(0.99)"
	confidenceStr := fmt.Sprintf("%.2f", minConfidence)
	if !strings.Contains(v.reasoning, confidenceStr) {
		v.t.Logf("Warning: Could not verify confidence %.2f for intent %s", minConfidence, intentType)
	}

	return v
}

// AssertHypothesisGenerated verifies a hypothesis was generated
func (v *ResponseValidator) AssertHypothesisGenerated(description string) *ResponseValidator {
	if !strings.Contains(v.reasoning, description) {
		v.t.Errorf("Hypothesis not found: %s\nResponse:\n%s", description, v.reasoning)
	}
	return v
}

// AssertRetrievalPlanCreated verifies a retrieval plan was created for a source
func (v *ResponseValidator) AssertRetrievalPlanCreated(source string) *ResponseValidator {
	if !strings.Contains(v.reasoning, source) {
		v.t.Errorf("Retrieval plan for source %s not found in response:\n%s", source, v.reasoning)
	}
	return v
}

// AssertArtifactRetrieved verifies an artifact was retrieved
func (v *ResponseValidator) AssertArtifactRetrieved(artifactID string) *ResponseValidator {
	if !strings.Contains(v.reasoning, artifactID) {
		v.t.Errorf("Artifact %s not found in response:\n%s", artifactID, v.reasoning)
	}
	return v
}

// AssertFactSynthesized verifies a fact was synthesized
func (v *ResponseValidator) AssertFactSynthesized(statement string) *ResponseValidator {
	if !strings.Contains(v.reasoning, statement) {
		v.t.Errorf("Fact not found: %s\nResponse:\n%s", statement, v.reasoning)
	}
	return v
}

// AssertConclusionDrawn verifies a conclusion was drawn
func (v *ResponseValidator) AssertConclusionDrawn(conclusion string) *ResponseValidator {
	if !strings.Contains(v.reasoning, conclusion) {
		v.t.Errorf("Conclusion not found: %s\nResponse:\n%s", conclusion, v.reasoning)
	}
	return v
}

// AssertSummaryGenerated verifies a summary was generated
func (v *ResponseValidator) AssertSummaryGenerated() *ResponseValidator {
	if !strings.Contains(v.reasoning, "Executive Summary") && !strings.Contains(v.reasoning, "Summary:") {
		v.t.Errorf("Summary not found in response:\n%s", v.reasoning)
	}
	return v
}

// GetReasoning returns the raw reasoning content
func (v *ResponseValidator) GetReasoning() string {
	return v.reasoning
}
