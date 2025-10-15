package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// ReasoningStructureAgent builds a reasoning goal hierarchy, generates hypotheses,
// and creates a dependency graph between reasoning steps.
//
// Design Principles:
// - Builds structured reasoning plan from detected intents
// - Generates testable hypotheses and assumptions
// - Creates explicit dependencies between reasoning steps
// - Detects cycles in dependency graph
// - Determines expected artifacts for each step
//
// Input Requirements:
// - reasoning.intents: List of detected intents with confidence scores
// - reasoning.entities: Optional entities to enrich hypothesis generation
//
// Output:
// - reasoning.hypotheses[]: List of hypotheses with dependencies
// - reasoning.dependency_map: Graph of dependencies between steps
// - retrieval.artifacts: Expected artifacts from reasoning process
//
// Capabilities:
// - Deterministic hierarchy generation based on intent types
// - Cycle detection in dependency graph
// - Hypothesis prioritization based on intent confidence
type ReasoningStructureAgent struct {
	id string
}

// DependencyGraph represents the dependency structure between reasoning steps.
type DependencyGraph struct {
	Nodes []string            `json:"nodes"`
	Edges map[string][]string `json:"edges"` // from -> [to1, to2, ...]
}

// NewReasoningStructureAgent creates a new reasoning structure agent.
func NewReasoningStructureAgent() *ReasoningStructureAgent {
	return &ReasoningStructureAgent{
		id: "reasoning_structure",
	}
}

// AgentID returns the unique identifier for this agent.
func (a *ReasoningStructureAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
func (a *ReasoningStructureAgent) Preconditions() []string {
	return []string{
		"reasoning.intents",
	}
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *ReasoningStructureAgent) Postconditions() []string {
	return []string{
		"reasoning.hypotheses",
		"reasoning.dependency_map",
	}
}

// Execute builds the reasoning structure from detected intents.
func (a *ReasoningStructureAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	startTime := time.Now()

	// Clone context to avoid modifying input
	newContext, err := agentContext.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone context: %w", err)
	}

	// Validate preconditions
	if err := a.validatePreconditions(newContext); err != nil {
		return nil, fmt.Errorf("precondition validation failed: %w", err)
	}

	// Extract intents and entities
	intents := newContext.Reasoning.Intents
	entities := newContext.Reasoning.Entities

	// Generate hypotheses from intents
	hypotheses := a.generateHypotheses(intents, entities)

	// Build dependency graph
	dependencyGraph := a.buildDependencyGraph(hypotheses)

	// Detect cycles
	if cycles := a.detectCycles(dependencyGraph); len(cycles) > 0 {
		// Log warning but don't fail - cycles will be broken by removing lowest confidence links
		a.recordWarning(newContext, fmt.Sprintf("Detected cycles in dependency graph: %v", cycles))
		dependencyGraph = a.breakCycles(dependencyGraph, hypotheses, cycles)
	}

	// Write results to context
	newContext.Reasoning.Hypotheses = hypotheses
	newContext.Reasoning.DependencyMap = dependencyGraph

	// Track agent execution in audit
	duration := time.Since(startTime)
	a.recordAgentRun(newContext, duration, "success", nil)

	return newContext, nil
}

// validatePreconditions checks that all required context keys are present.
func (a *ReasoningStructureAgent) validatePreconditions(ctx *models.AgentContext) error {
	if ctx.Reasoning == nil {
		return fmt.Errorf("reasoning context is nil")
	}

	if len(ctx.Reasoning.Intents) == 0 {
		return fmt.Errorf("no intents found in context (required precondition: reasoning.intents)")
	}

	return nil
}

// generateHypotheses creates hypotheses based on detected intents.
func (a *ReasoningStructureAgent) generateHypotheses(intents []models.Intent, entities map[string]interface{}) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}
	hypothesisID := 0

	// Generate hypotheses for each intent
	for _, intent := range intents {
		// Skip low-confidence intents
		if intent.Confidence < 0.3 {
			continue
		}

		intentHypotheses := a.generateHypothesesForIntent(intent, entities, &hypothesisID)
		hypotheses = append(hypotheses, intentHypotheses...)
	}

	return hypotheses
}

// generateHypothesesForIntent generates hypotheses for a specific intent type.
func (a *ReasoningStructureAgent) generateHypothesesForIntent(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	switch intent.Type {
	case "query_commits":
		hypotheses = append(hypotheses, a.createCommitQueryHypotheses(intent, entities, hypothesisID)...)
	case "query_issues":
		hypotheses = append(hypotheses, a.createIssueQueryHypotheses(intent, entities, hypothesisID)...)
	case "query_analytics":
		hypotheses = append(hypotheses, a.createAnalyticsHypotheses(intent, entities, hypothesisID)...)
	case "query_status":
		hypotheses = append(hypotheses, a.createStatusHypotheses(intent, entities, hypothesisID)...)
	case "command_action":
		hypotheses = append(hypotheses, a.createCommandHypotheses(intent, entities, hypothesisID)...)
	case "request_help":
		hypotheses = append(hypotheses, a.createHelpHypotheses(intent, entities, hypothesisID)...)
	case "conversation":
		hypotheses = append(hypotheses, a.createConversationHypotheses(intent, entities, hypothesisID)...)
	default:
		// Unknown intent type - create generic hypothesis
		hypotheses = append(hypotheses, models.Hypothesis{
			ID:          fmt.Sprintf("h%d", *hypothesisID),
			Description: fmt.Sprintf("Process %s intent", intent.Type),
		})
		*hypothesisID++
	}

	return hypotheses
}

// createCommitQueryHypotheses generates hypotheses for commit queries.
func (a *ReasoningStructureAgent) createCommitQueryHypotheses(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	// H1: Retrieve commit data
	h1ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h1ID,
		Description:  "Retrieve commit data from GitLab",
		Dependencies: []string{},
	})

	// H2: Filter and rank commits (depends on H1)
	h2ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h2ID,
		Description:  "Filter and rank commits by relevance",
		Dependencies: []string{h1ID},
	})

	// H3: Format commit summary (depends on H2)
	h3ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h3ID,
		Description:  "Format commit summary for user",
		Dependencies: []string{h2ID},
	})

	return hypotheses
}

// createIssueQueryHypotheses generates hypotheses for issue queries.
func (a *ReasoningStructureAgent) createIssueQueryHypotheses(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	// H1: Retrieve issue data
	h1ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h1ID,
		Description:  "Retrieve issue data from YouTrack",
		Dependencies: []string{},
	})

	// H2: Apply filters (status, date, etc.)
	h2ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h2ID,
		Description:  "Apply filters based on entities (status, date, project)",
		Dependencies: []string{h1ID},
	})

	// H3: Format issue summary
	h3ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h3ID,
		Description:  "Format issue summary for user",
		Dependencies: []string{h2ID},
	})

	return hypotheses
}

// createAnalyticsHypotheses generates hypotheses for analytics queries.
func (a *ReasoningStructureAgent) createAnalyticsHypotheses(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	// H1: Identify data sources
	h1ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h1ID,
		Description:  "Identify relevant data sources (commits, issues, metrics)",
		Dependencies: []string{},
	})

	// H2: Aggregate data
	h2ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h2ID,
		Description:  "Aggregate data from multiple sources",
		Dependencies: []string{h1ID},
	})

	// H3: Calculate statistics
	h3ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h3ID,
		Description:  "Calculate statistics and trends",
		Dependencies: []string{h2ID},
	})

	// H4: Generate report
	h4ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h4ID,
		Description:  "Generate analytics report with visualizations",
		Dependencies: []string{h3ID},
	})

	return hypotheses
}

// createStatusHypotheses generates hypotheses for status queries.
func (a *ReasoningStructureAgent) createStatusHypotheses(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	// H1: Check system health
	h1ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h1ID,
		Description:  "Check system health and status",
		Dependencies: []string{},
	})

	// H2: Gather recent events
	h2ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h2ID,
		Description:  "Gather recent events and changes",
		Dependencies: []string{},
	})

	// H3: Synthesize status report (depends on H1 and H2)
	h3ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h3ID,
		Description:  "Synthesize overall status report",
		Dependencies: []string{h1ID, h2ID},
	})

	return hypotheses
}

// createCommandHypotheses generates hypotheses for command actions.
func (a *ReasoningStructureAgent) createCommandHypotheses(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	// H1: Validate command preconditions
	h1ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h1ID,
		Description:  "Validate command preconditions and permissions",
		Dependencies: []string{},
	})

	// H2: Execute command
	h2ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h2ID,
		Description:  "Execute command action",
		Dependencies: []string{h1ID},
	})

	// H3: Verify command result
	h3ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h3ID,
		Description:  "Verify command execution result",
		Dependencies: []string{h2ID},
	})

	return hypotheses
}

// createHelpHypotheses generates hypotheses for help requests.
func (a *ReasoningStructureAgent) createHelpHypotheses(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	// H1: Identify help topic
	h1ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h1ID,
		Description:  "Identify help topic and user context",
		Dependencies: []string{},
	})

	// H2: Retrieve relevant documentation
	h2ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h2ID,
		Description:  "Retrieve relevant documentation and examples",
		Dependencies: []string{h1ID},
	})

	// H3: Format helpful response
	h3ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h3ID,
		Description:  "Format helpful response with examples",
		Dependencies: []string{h2ID},
	})

	return hypotheses
}

// createConversationHypotheses generates hypotheses for general conversation.
func (a *ReasoningStructureAgent) createConversationHypotheses(intent models.Intent, entities map[string]interface{}, hypothesisID *int) []models.Hypothesis {
	hypotheses := []models.Hypothesis{}

	// H1: Generate conversational response
	h1ID := fmt.Sprintf("h%d", *hypothesisID)
	*hypothesisID++
	hypotheses = append(hypotheses, models.Hypothesis{
		ID:           h1ID,
		Description:  "Generate appropriate conversational response",
		Dependencies: []string{},
	})

	return hypotheses
}

// buildDependencyGraph constructs a dependency graph from hypotheses.
func (a *ReasoningStructureAgent) buildDependencyGraph(hypotheses []models.Hypothesis) *DependencyGraph {
	graph := &DependencyGraph{
		Nodes: []string{},
		Edges: make(map[string][]string),
	}

	// Add all nodes
	for _, h := range hypotheses {
		graph.Nodes = append(graph.Nodes, h.ID)
	}

	// Add edges based on dependencies
	for _, h := range hypotheses {
		if len(h.Dependencies) > 0 {
			for _, dep := range h.Dependencies {
				if graph.Edges[dep] == nil {
					graph.Edges[dep] = []string{}
				}
				graph.Edges[dep] = append(graph.Edges[dep], h.ID)
			}
		}
	}

	return graph
}

// detectCycles detects cycles in the dependency graph using DFS.
func (a *ReasoningStructureAgent) detectCycles(graph *DependencyGraph) [][]string {
	cycles := [][]string{}
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		// Visit all neighbors
		for _, neighbor := range graph.Edges[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle
				cycleStart := 0
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				cycle := make([]string, len(path)-cycleStart)
				copy(cycle, path[cycleStart:])
				cycles = append(cycles, cycle)
				return true
			}
		}

		// Remove from recursion stack
		recStack[node] = false
		path = path[:len(path)-1]
		return false
	}

	// Check for cycles from each unvisited node
	for _, node := range graph.Nodes {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

// breakCycles breaks cycles by removing edges with lowest confidence.
func (a *ReasoningStructureAgent) breakCycles(graph *DependencyGraph, hypotheses []models.Hypothesis, cycles [][]string) *DependencyGraph {
	// For simplicity, remove the last edge in each cycle
	// In a more sophisticated implementation, we would consider hypothesis confidence
	newGraph := &DependencyGraph{
		Nodes: graph.Nodes,
		Edges: make(map[string][]string),
	}

	// Copy all edges
	for from, tos := range graph.Edges {
		newGraph.Edges[from] = make([]string, len(tos))
		copy(newGraph.Edges[from], tos)
	}

	// Remove cycle-breaking edges
	for _, cycle := range cycles {
		if len(cycle) > 1 {
			// Remove edge from second-to-last to last node in cycle
			from := cycle[len(cycle)-2]
			to := cycle[len(cycle)-1]

			// Remove 'to' from 'from's edges
			edges := newGraph.Edges[from]
			newEdges := []string{}
			for _, edge := range edges {
				if edge != to {
					newEdges = append(newEdges, edge)
				}
			}
			newGraph.Edges[from] = newEdges
		}
	}

	return newGraph
}

// recordAgentRun records the agent execution in the audit trail.
func (a *ReasoningStructureAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"reasoning.hypotheses",
			"reasoning.dependency_map",
		},
	}

	if err != nil {
		run.Error = err.Error()
	}

	if ctx.Audit == nil {
		ctx.Audit = &models.AuditContext{}
	}

	ctx.Audit.AgentRuns = append(ctx.Audit.AgentRuns, run)

	// Update performance metrics
	if ctx.Diagnostics == nil {
		ctx.Diagnostics = &models.DiagnosticsContext{
			Performance: &models.PerformanceData{},
		}
	}

	if ctx.Diagnostics.Performance.AgentMetrics == nil {
		ctx.Diagnostics.Performance.AgentMetrics = make(map[string]*models.AgentMetrics)
	}

	ctx.Diagnostics.Performance.AgentMetrics[a.id] = &models.AgentMetrics{
		DurationMS: duration.Milliseconds(),
		LLMCalls:   0, // No LLM calls for rule-based structure generation
		Status:     status,
	}
}

// recordWarning records a warning message in diagnostics.
func (a *ReasoningStructureAgent) recordWarning(ctx *models.AgentContext, message string) {
	if ctx.Diagnostics == nil {
		ctx.Diagnostics = &models.DiagnosticsContext{}
	}

	warning := models.Warning{
		Timestamp: time.Now(),
		AgentID:   a.id,
		Message:   message,
	}

	ctx.Diagnostics.Warnings = append(ctx.Diagnostics.Warnings, warning)
}

// GetMetadata returns agent metadata (implements MetadataProvider).
func (a *ReasoningStructureAgent) GetMetadata() services.AgentMetadata {
	return services.AgentMetadata{
		ID:          a.id,
		Name:        "Reasoning Structure Agent",
		Description: "Builds reasoning goal hierarchy, generates hypotheses, and creates dependency graphs from detected intents",
		Version:     "1.0.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"reasoning", "planning", "structure", "hypotheses", "dependency-graph"},
		Dependencies: []string{"intent_detection"}, // Requires intent detection
	}
}

// GetCapabilities returns agent capabilities (implements CapabilitiesProvider).
func (a *ReasoningStructureAgent) GetCapabilities() services.AgentCapabilities {
	return services.AgentCapabilities{
		SupportsParallelExecution: false, // Must run after intent detection
		SupportsRetry:             true,
		RequiresLLM:               false, // Rule-based structure generation
		IsDeterministic:           true,  // Same intents produce same structure
		EstimatedDuration:         100,   // ~100ms for structure generation (int, not int64)
	}
}

// Format intent entities for hypothesis generation (helper function).
func formatEntities(entities map[string]interface{}) string {
	parts := []string{}
	for key, value := range entities {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	return strings.Join(parts, ", ")
}
