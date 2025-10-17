package datasources

import (
	"context"
	"fmt"
	"strings"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// MockDataSourceClient is a mock implementation of DataSourceClient for testing.
//
// Design:
// - Returns predefined test data based on query source
// - Simulates different data sources (gitlab, youtrack)
// - Supports customizable responses for different test scenarios
type MockDataSourceClient struct {
	sourceName      string
	mockArtifacts   map[string][]models.Artifact // Keyed by query ID
	shouldFail      bool
	healthCheckPass bool
}

// NewMockDataSourceClient creates a new mock data source client.
func NewMockDataSourceClient(sourceName string) *MockDataSourceClient {
	return &MockDataSourceClient{
		sourceName:      sourceName,
		mockArtifacts:   make(map[string][]models.Artifact),
		shouldFail:      false,
		healthCheckPass: true,
	}
}

// WithMockArtifacts sets mock artifacts for a specific query ID.
func (m *MockDataSourceClient) WithMockArtifacts(queryID string, artifacts []models.Artifact) *MockDataSourceClient {
	m.mockArtifacts[queryID] = artifacts
	return m
}

// WithFailure configures the mock to fail on queries.
func (m *MockDataSourceClient) WithFailure(shouldFail bool) *MockDataSourceClient {
	m.shouldFail = shouldFail
	return m
}

// WithHealthCheck configures the mock health check result.
func (m *MockDataSourceClient) WithHealthCheck(pass bool) *MockDataSourceClient {
	m.healthCheckPass = pass
	return m
}

// ExecuteQuery executes a mock query and returns predefined artifacts.
func (m *MockDataSourceClient) ExecuteQuery(ctx context.Context, query models.Query) ([]models.Artifact, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock query failed for source: %s", m.sourceName)
	}

	// Return artifacts if configured for this query
	if artifacts, ok := m.mockArtifacts[query.ID]; ok {
		return artifacts, nil
	}

	// Generate default mock artifacts based on source
	return m.generateDefaultArtifacts(query), nil
}

// generateDefaultArtifacts creates default test artifacts based on query source.
func (m *MockDataSourceClient) generateDefaultArtifacts(query models.Query) []models.Artifact {
	switch m.sourceName {
	case "gitlab":
		return m.generateGitLabArtifacts(query)
	case "youtrack":
		return m.generateYouTrackArtifacts(query)
	default:
		return []models.Artifact{
			{
				ID:      fmt.Sprintf("artifact-%s-1", query.ID),
				Type:    "generic",
				Source:  m.sourceName,
				Content: fmt.Sprintf("Mock data for query: %s", query.QueryString),
			},
		}
	}
}

// generateGitLabArtifacts creates mock GitLab commit artifacts.
func (m *MockDataSourceClient) generateGitLabArtifacts(query models.Query) []models.Artifact {
	// Simulate GitLab commit data
	commits := []models.Artifact{
		{
			ID:     fmt.Sprintf("%s-commit-1", query.ID),
			Type:   "commit",
			Source: "gitlab",
			Content: map[string]interface{}{
				"id":      "abc123",
				"message": "feat: add new feature",
				"author":  "john.doe@example.com",
				"date":    "2025-01-15T10:30:00Z",
				"project": extractProject(query),
			},
		},
		{
			ID:     fmt.Sprintf("%s-commit-2", query.ID),
			Type:   "commit",
			Source: "gitlab",
			Content: map[string]interface{}{
				"id":      "def456",
				"message": "fix: resolve bug in authentication",
				"author":  "jane.smith@example.com",
				"date":    "2025-01-14T15:20:00Z",
				"project": extractProject(query),
			},
		},
	}

	return commits
}

// generateYouTrackArtifacts creates mock YouTrack issue artifacts.
func (m *MockDataSourceClient) generateYouTrackArtifacts(query models.Query) []models.Artifact {
	// Simulate YouTrack issue data
	issues := []models.Artifact{
		{
			ID:     fmt.Sprintf("%s-issue-1", query.ID),
			Type:   "issue",
			Source: "youtrack",
			Content: map[string]interface{}{
				"id":          "PROJECT-123",
				"title":       "Implement user authentication",
				"status":      "open",
				"priority":    "high",
				"assignee":    "john.doe",
				"created":     "2025-01-10T09:00:00Z",
				"description": "Add OAuth2 authentication for users",
			},
		},
		{
			ID:     fmt.Sprintf("%s-issue-2", query.ID),
			Type:   "issue",
			Source: "youtrack",
			Content: map[string]interface{}{
				"id":          "PROJECT-124",
				"title":       "Fix database connection pool",
				"status":      "in-progress",
				"priority":    "critical",
				"assignee":    "jane.smith",
				"created":     "2025-01-12T11:30:00Z",
				"description": "Connection pool exhaustion under load",
			},
		},
	}

	return issues
}

// extractProject extracts project name from query filters.
func extractProject(query models.Query) string {
	if query.Filters != nil {
		if projects, ok := query.Filters["projects"]; ok {
			if projectList, ok := projects.([]string); ok && len(projectList) > 0 {
				return projectList[0]
			}
			if projectStr, ok := projects.(string); ok {
				return projectStr
			}
		}
	}

	// Default project
	return "default-project"
}

// HealthCheck returns the configured health status.
func (m *MockDataSourceClient) HealthCheck(ctx context.Context) bool {
	return m.healthCheckPass
}

// SourceName returns the source name.
func (m *MockDataSourceClient) SourceName() string {
	return m.sourceName
}

// Ensure MockDataSourceClient implements DataSourceClient interface
var _ services.DataSourceClient = (*MockDataSourceClient)(nil)

// NewMockGitLabClient creates a mock GitLab client with realistic test data.
func NewMockGitLabClient() *MockDataSourceClient {
	return NewMockDataSourceClient("gitlab")
}

// NewMockYouTrackClient creates a mock YouTrack client with realistic test data.
func NewMockYouTrackClient() *MockDataSourceClient {
	return NewMockDataSourceClient("youtrack")
}

// NewMockMultiSourceClient creates a MultiSourceClient with mock clients for testing.
func NewMockMultiSourceClient() *services.MultiSourceClient {
	multiClient := services.NewMultiSourceClient()
	multiClient.RegisterClient(NewMockGitLabClient())
	multiClient.RegisterClient(NewMockYouTrackClient())
	return multiClient
}

// Helper function to check if query string contains a keyword
func containsKeyword(queryString string, keyword string) bool {
	return strings.Contains(strings.ToLower(queryString), strings.ToLower(keyword))
}
