package single_agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mshogin/agents/internal/domain/models"
)

// CreateTestRequest creates a standard test request with the given user message
func CreateTestRequest(userMessage string, stream bool) *models.CompletionRequest {
	return &models.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []models.Message{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		Stream: stream,
	}
}

// SendRequest sends a POST request to the test server and returns the response
func SendRequest(t *testing.T, server *TestServer, req *models.CompletionRequest) *http.Response {
	t.Helper()

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", server.URL()+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Workflow", server.Workflow())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	return resp
}

// ExtractReasoningBlock extracts the reasoning block from a streaming SSE response
func ExtractReasoningBlock(t *testing.T, resp *http.Response) string {
	t.Helper()

	defer resp.Body.Close()

	var reasoningBlock string
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE data line
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Check for [DONE] marker
			if data == "[DONE]" {
				break
			}

			// Try to parse as JSON
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			// Extract reasoning event
			if eventType, ok := event["type"].(string); ok && eventType == "reasoning" {
				if eventData, ok := event["data"].(map[string]interface{}); ok {
					if message, ok := eventData["message"].(string); ok {
						reasoningBlock = message
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading response: %v", err)
	}

	if reasoningBlock == "" {
		t.Fatal("No reasoning block found in response")
	}

	return reasoningBlock
}

// ExtractCompletionChunks extracts all LLM completion chunks from a streaming response
func ExtractCompletionChunks(t *testing.T, resp *http.Response) []string {
	t.Helper()

	defer resp.Body.Close()

	var chunks []string
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				break
			}

			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			// Extract completion chunks
			if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok && content != "" {
							chunks = append(chunks, content)
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading completion chunks: %v", err)
	}

	return chunks
}

// ReadFullResponse reads the entire response body as a string
func ReadFullResponse(t *testing.T, resp *http.Response) string {
	t.Helper()

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return string(body)
}

// ParseJSONResponse parses a JSON response into the provided struct
func ParseJSONResponse(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("Failed to unmarshal response: %v\nBody: %s", err, string(body))
	}
}

// AssertStatusCode verifies the HTTP response status code
func AssertStatusCode(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()

	if resp.StatusCode != expectedStatus {
		body := ReadFullResponse(t, resp)
		t.Fatalf("Expected status %d, got %d\nResponse body: %s", expectedStatus, resp.StatusCode, body)
	}
}

// AssertContentType verifies the Content-Type header
func AssertContentType(t *testing.T, resp *http.Response, expectedType string) {
	t.Helper()

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, expectedType) {
		t.Fatalf("Expected Content-Type %s, got %s", expectedType, contentType)
	}
}

// CreateMockIntent creates a mock intent for testing
func CreateMockIntent(intentType string, confidence float64) models.Intent {
	return models.Intent{
		Type:       intentType,
		Confidence: confidence,
		Slots: map[string]interface{}{
			"source": "test",
		},
	}
}

// CreateMockHypothesis creates a mock hypothesis for testing
func CreateMockHypothesis(description string, confidence float64, evidenceType string) models.Hypothesis {
	return models.Hypothesis{
		Description: description,
		Confidence:  confidence,
		Type:        evidenceType,
		Evidence:    []string{"test-evidence"},
	}
}

// CreateMockRetrievalPlan creates a mock retrieval plan for testing
func CreateMockRetrievalPlan(source string, description string, priority int) models.RetrievalPlan {
	return models.RetrievalPlan{
		Source:      source,
		Description: description,
		Priority:    priority,
	}
}

// CreateMockArtifact creates a mock artifact for testing
func CreateMockArtifact(id string, artifactType string, source string, title string, content string) models.Artifact {
	return models.Artifact{
		ID:         id,
		Type:       artifactType,
		Source:     source,
		Title:      title,
		Content:    content,
		Confidence: 0.95,
		Metadata: map[string]interface{}{
			"test": true,
		},
	}
}

// CreateMockFact creates a mock fact for testing
func CreateMockFact(statement string, source string, confidence float64) models.Fact {
	return models.Fact{
		Statement:  statement,
		Source:     source,
		Confidence: confidence,
		Timestamp:  "2024-01-15T10:30:00Z",
	}
}

// CreateMockConclusion creates a mock conclusion for testing
func CreateMockConclusion(statement string, confidence float64, conclusionType string) models.Conclusion {
	return models.Conclusion{
		Statement:  statement,
		Confidence: confidence,
		Type:       conclusionType,
		Evidence:   []string{"fact-1", "fact-2"},
	}
}

// PrintReasoningBlock is a helper to print the reasoning block for debugging
func PrintReasoningBlock(t *testing.T, reasoning string) {
	t.Helper()
	t.Logf("\n=== REASONING BLOCK ===\n%s\n======================\n", reasoning)
}

// CountLinesInSection counts the number of lines in a specific section of the reasoning block
func CountLinesInSection(reasoning string, sectionMarker string) int {
	lines := strings.Split(reasoning, "\n")
	inSection := false
	count := 0

	for _, line := range lines {
		if strings.Contains(line, sectionMarker) {
			inSection = true
			continue
		}

		if inSection && strings.HasPrefix(line, "===") {
			break
		}

		if inSection && strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count
}

// ExtractMetricValue extracts a metric value from the METRICS section
func ExtractMetricValue(reasoning string, metricName string) string {
	lines := strings.Split(reasoning, "\n")
	inMetrics := false

	for _, line := range lines {
		if strings.Contains(line, "⏱️  METRICS") {
			inMetrics = true
			continue
		}

		if inMetrics && strings.Contains(line, metricName) {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}

		if inMetrics && strings.HasPrefix(line, "===") {
			break
		}
	}

	return ""
}

// ValidateReasoningStructure performs basic structural validation of reasoning block
func ValidateReasoningStructure(t *testing.T, reasoning string, expectedAgent string) {
	t.Helper()

	// Check for required sections
	requiredSections := []string{
		fmt.Sprintf("=== %s AGENT TEST ===", strings.ToUpper(strings.ReplaceAll(expectedAgent, "_", " "))),
		"📥 INPUT",
		"📤 OUTPUT",
		"📤 SYSTEM PROMPT FOR LLM",
		"⏱️  METRICS",
	}

	for _, section := range requiredSections {
		if !strings.Contains(reasoning, section) {
			t.Errorf("Missing required section: %s", section)
		}
	}
}
