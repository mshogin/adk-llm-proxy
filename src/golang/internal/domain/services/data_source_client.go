package services

import (
	"context"

	"github.com/mshogin/agents/internal/domain/models"
)

// DataSourceClient defines the interface for querying external data sources.
//
// Design Principles (DIP - Dependency Inversion):
// - Domain layer defines the interface
// - Infrastructure layer provides implementations
// - Allows easy testing with mocks
// - Supports multiple data source types (GitLab, YouTrack, etc.)
//
// Implementation Notes:
// - Each source (gitlab, youtrack) has its own implementation
// - Implementations may call MCP servers, REST APIs, or databases
// - All implementations must handle timeouts and errors gracefully
type DataSourceClient interface {
	// ExecuteQuery executes a query against a data source and returns artifacts.
	//
	// Parameters:
	// - ctx: Context for cancellation and timeout
	// - query: Query to execute (contains source, filters, query string)
	//
	// Returns:
	// - Artifacts: Retrieved data as structured artifacts
	// - Error: If query fails or times out
	//
	// Examples:
	// - GitLab: Fetch commits matching filters
	// - YouTrack: Fetch issues matching status/project
	ExecuteQuery(ctx context.Context, query models.Query) ([]models.Artifact, error)

	// HealthCheck verifies the data source is accessible.
	//
	// Returns:
	// - true if source is healthy and responsive
	// - false if source is unavailable or unhealthy
	HealthCheck(ctx context.Context) bool

	// SourceName returns the name of this data source.
	//
	// Examples: "gitlab", "youtrack", "database"
	SourceName() string
}

// MultiSourceClient manages multiple data source clients.
//
// Design:
// - Routes queries to appropriate client based on query.Source
// - Handles errors if source is not available
// - Can execute queries in parallel across sources
type MultiSourceClient struct {
	clients map[string]DataSourceClient
}

// NewMultiSourceClient creates a new multi-source client.
func NewMultiSourceClient() *MultiSourceClient {
	return &MultiSourceClient{
		clients: make(map[string]DataSourceClient),
	}
}

// RegisterClient registers a data source client.
func (m *MultiSourceClient) RegisterClient(client DataSourceClient) {
	m.clients[client.SourceName()] = client
}

// GetClient retrieves a client by source name.
func (m *MultiSourceClient) GetClient(source string) (DataSourceClient, bool) {
	client, ok := m.clients[source]
	return client, ok
}

// ExecuteQuery routes the query to the appropriate client.
func (m *MultiSourceClient) ExecuteQuery(ctx context.Context, query models.Query) ([]models.Artifact, error) {
	client, ok := m.GetClient(query.Source)
	if !ok {
		// Return empty artifacts if source not found (graceful degradation)
		return []models.Artifact{}, nil
	}

	return client.ExecuteQuery(ctx, query)
}

// HealthCheck checks health of all registered clients.
func (m *MultiSourceClient) HealthCheck(ctx context.Context) map[string]bool {
	health := make(map[string]bool)
	for name, client := range m.clients {
		health[name] = client.HealthCheck(ctx)
	}
	return health
}
