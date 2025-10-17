package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// RetrievalPlannerAgent creates structured retrieval plans for information gathering.
//
// Design Principles:
// - Prioritizes structured data sources (GitLab, YouTrack) over unstructured
// - Creates normalized queries with filters (project, date, status, provider)
// - Adds time and volume constraints per source
// - Determines optimal retrieval strategy based on intent and entities
//
// Input Requirements:
// - reasoning.intents: List of detected intents
// - reasoning.hypotheses: Reasoning structure with dependencies
// - reasoning.entities: Optional entities to refine queries
//
// Output:
// - retrieval.plans[]: Prioritized retrieval plans per source
// - retrieval.queries[]: Normalized queries with filters
//
// Capabilities:
// - Deterministic plan generation based on intents
// - Source prioritization (structured > unstructured)
// - Query normalization with entity-based filters
// - Time/volume constraints per source
type RetrievalPlannerAgent struct {
	id string
}

// NewRetrievalPlannerAgent creates a new retrieval planner agent.
func NewRetrievalPlannerAgent() *RetrievalPlannerAgent {
	return &RetrievalPlannerAgent{
		id: "retrieval_planner",
	}
}

// AgentID returns the unique identifier for this agent.
func (a *RetrievalPlannerAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
func (a *RetrievalPlannerAgent) Preconditions() []string {
	return []string{
		"reasoning.intents",
		"reasoning.hypotheses",
	}
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *RetrievalPlannerAgent) Postconditions() []string {
	return []string{
		"retrieval.plans",
		"retrieval.queries",
	}
}

// Execute creates retrieval plans based on reasoning structure.
func (a *RetrievalPlannerAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
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

	// Extract intents, hypotheses, and entities
	intents := newContext.Reasoning.Intents
	hypotheses := newContext.Reasoning.Hypotheses
	entities := newContext.Reasoning.Entities

	// Store detailed agent trace
	agentTrace := map[string]interface{}{
		"agent_id":         a.id,
		"input_intents":    intents,
		"input_hypotheses": hypotheses,
		"input_entities":   entities,
		"intents_count":    len(intents),
		"hypotheses_count": len(hypotheses),
	}

	// Generate retrieval plans based on intents
	plans := a.generateRetrievalPlans(intents, hypotheses, entities)
	agentTrace["generated_plans"] = plans
	agentTrace["plans_count"] = len(plans)

	// Generate normalized queries from plans
	queries := a.generateQueries(plans, entities)
	agentTrace["generated_queries"] = queries
	agentTrace["queries_count"] = len(queries)

	// Write results to context
	newContext.Retrieval.Plans = plans
	newContext.Retrieval.Queries = queries

	// Store final output in trace
	agentTrace["output_plans"] = plans
	agentTrace["output_queries"] = queries

	// Store agent trace in LLM cache
	if newContext.LLM == nil {
		newContext.LLM = &models.LLMContext{
			Cache: make(map[string]interface{}),
		}
	}
	if newContext.LLM.Cache == nil {
		newContext.LLM.Cache = make(map[string]interface{})
	}
	if traces, ok := newContext.LLM.Cache["agent_traces"].([]interface{}); ok {
		newContext.LLM.Cache["agent_traces"] = append(traces, agentTrace)
	} else {
		newContext.LLM.Cache["agent_traces"] = []interface{}{agentTrace}
	}

	// Track agent execution in audit
	duration := time.Since(startTime)
	a.recordAgentRun(newContext, duration, "success", nil)

	return newContext, nil
}

// validatePreconditions checks that all required context keys are present.
func (a *RetrievalPlannerAgent) validatePreconditions(ctx *models.AgentContext) error {
	if ctx.Reasoning == nil {
		return fmt.Errorf("reasoning context is nil")
	}

	if len(ctx.Reasoning.Intents) == 0 {
		return fmt.Errorf("no intents found in context (required precondition: reasoning.intents)")
	}

	if len(ctx.Reasoning.Hypotheses) == 0 {
		return fmt.Errorf("no hypotheses found in context (required precondition: reasoning.hypotheses)")
	}

	return nil
}

// generateRetrievalPlans creates retrieval plans based on intents.
func (a *RetrievalPlannerAgent) generateRetrievalPlans(intents []models.Intent, hypotheses []models.Hypothesis, entities map[string]interface{}) []models.RetrievalPlan {
	plans := []models.RetrievalPlan{}
	planID := 0

	// Generate plans for each high-confidence intent
	for _, intent := range intents {
		if intent.Confidence < 0.3 {
			continue
		}

		intentPlans := a.generatePlansForIntent(intent, hypotheses, entities, &planID)
		plans = append(plans, intentPlans...)
	}

	// Sort plans by priority (higher first)
	for i := 0; i < len(plans)-1; i++ {
		for j := i + 1; j < len(plans); j++ {
			if plans[j].Priority > plans[i].Priority {
				plans[i], plans[j] = plans[j], plans[i]
			}
		}
	}

	return plans
}

// generatePlansForIntent creates retrieval plans for a specific intent type.
func (a *RetrievalPlannerAgent) generatePlansForIntent(intent models.Intent, hypotheses []models.Hypothesis, entities map[string]interface{}, planID *int) []models.RetrievalPlan {
	plans := []models.RetrievalPlan{}

	switch intent.Type {
	case "query_commits":
		plans = append(plans, a.createCommitQueryPlans(intent, entities, planID)...)
	case "query_issues":
		plans = append(plans, a.createIssueQueryPlans(intent, entities, planID)...)
	case "query_analytics":
		plans = append(plans, a.createAnalyticsPlans(intent, entities, planID)...)
	case "query_status":
		plans = append(plans, a.createStatusPlans(intent, entities, planID)...)
	default:
		// For other intents, create generic plan
		plans = append(plans, models.RetrievalPlan{
			ID:          fmt.Sprintf("plan%d", *planID),
			Description: fmt.Sprintf("Retrieve data for %s intent", intent.Type),
			Sources:     []string{"generic"},
			Priority:    5,
		})
		*planID++
	}

	return plans
}

// createCommitQueryPlans creates retrieval plans for commit queries.
func (a *RetrievalPlannerAgent) createCommitQueryPlans(intent models.Intent, entities map[string]interface{}, planID *int) []models.RetrievalPlan {
	plans := []models.RetrievalPlan{}

	// Plan 1: GitLab commits (structured, high priority)
	filters := make(map[string]interface{})
	if projects, ok := entities["projects"]; ok {
		filters["projects"] = projects
	}
	if dates, ok := entities["dates"]; ok {
		filters["dates"] = dates
	}

	plans = append(plans, models.RetrievalPlan{
		ID:          fmt.Sprintf("plan%d", *planID),
		Description: "Retrieve commit data from GitLab",
		Sources:     []string{"gitlab"},
		Filters:     filters,
		Priority:    10, // High priority - structured source
	})
	*planID++

	return plans
}

// createIssueQueryPlans creates retrieval plans for issue queries.
func (a *RetrievalPlannerAgent) createIssueQueryPlans(intent models.Intent, entities map[string]interface{}, planID *int) []models.RetrievalPlan {
	plans := []models.RetrievalPlan{}

	// Plan 1: YouTrack issues (structured, high priority)
	filters := make(map[string]interface{})
	if projects, ok := entities["projects"]; ok {
		filters["projects"] = projects
	}
	if dates, ok := entities["dates"]; ok {
		filters["dates"] = dates
	}
	if statuses, ok := entities["statuses"]; ok {
		filters["statuses"] = statuses
	}

	plans = append(plans, models.RetrievalPlan{
		ID:          fmt.Sprintf("plan%d", *planID),
		Description: "Retrieve issue data from YouTrack",
		Sources:     []string{"youtrack"},
		Filters:     filters,
		Priority:    10, // High priority - structured source
	})
	*planID++

	return plans
}

// createAnalyticsPlans creates retrieval plans for analytics queries.
func (a *RetrievalPlannerAgent) createAnalyticsPlans(intent models.Intent, entities map[string]interface{}, planID *int) []models.RetrievalPlan {
	plans := []models.RetrievalPlan{}

	// Plan 1: GitLab for commit metrics (structured)
	filters1 := make(map[string]interface{})
	if projects, ok := entities["projects"]; ok {
		filters1["projects"] = projects
	}
	if dates, ok := entities["dates"]; ok {
		filters1["dates"] = dates
	}

	plans = append(plans, models.RetrievalPlan{
		ID:          fmt.Sprintf("plan%d", *planID),
		Description: "Retrieve commit metrics from GitLab",
		Sources:     []string{"gitlab"},
		Filters:     filters1,
		Priority:    9, // High priority for structured data
	})
	*planID++

	// Plan 2: YouTrack for issue metrics (structured)
	filters2 := make(map[string]interface{})
	if projects, ok := entities["projects"]; ok {
		filters2["projects"] = projects
	}
	if dates, ok := entities["dates"]; ok {
		filters2["dates"] = dates
	}

	plans = append(plans, models.RetrievalPlan{
		ID:          fmt.Sprintf("plan%d", *planID),
		Description: "Retrieve issue metrics from YouTrack",
		Sources:     []string{"youtrack"},
		Filters:     filters2,
		Priority:    9, // High priority for structured data
	})
	*planID++

	return plans
}

// createStatusPlans creates retrieval plans for status queries.
func (a *RetrievalPlannerAgent) createStatusPlans(intent models.Intent, entities map[string]interface{}, planID *int) []models.RetrievalPlan {
	plans := []models.RetrievalPlan{}

	// Plan 1: Recent commits (for activity status)
	plans = append(plans, models.RetrievalPlan{
		ID:          fmt.Sprintf("plan%d", *planID),
		Description: "Retrieve recent commits for activity status",
		Sources:     []string{"gitlab"},
		Filters: map[string]interface{}{
			"dates": []string{"last week"},
		},
		Priority: 8,
	})
	*planID++

	// Plan 2: Open issues (for health status)
	plans = append(plans, models.RetrievalPlan{
		ID:          fmt.Sprintf("plan%d", *planID),
		Description: "Retrieve open issues for health status",
		Sources:     []string{"youtrack"},
		Filters: map[string]interface{}{
			"statuses": []string{"open", "in-progress"},
		},
		Priority: 8,
	})
	*planID++

	return plans
}

// generateQueries creates normalized queries from retrieval plans.
func (a *RetrievalPlannerAgent) generateQueries(plans []models.RetrievalPlan, entities map[string]interface{}) []models.Query {
	queries := []models.Query{}

	for _, plan := range plans {
		query := a.createQueryFromPlan(plan)
		queries = append(queries, query)
	}

	return queries
}

// createQueryFromPlan converts a retrieval plan into a normalized query.
func (a *RetrievalPlannerAgent) createQueryFromPlan(plan models.RetrievalPlan) models.Query {
	// Build query string from description and filters
	queryParts := []string{plan.Description}

	// Add filter details to query string
	if projects, ok := plan.Filters["projects"]; ok {
		// Try to convert to string slice (could be []interface{} from JSON unmarshaling)
		projectList := a.convertToStringSlice(projects)
		if len(projectList) > 0 {
			queryParts = append(queryParts, fmt.Sprintf("projects:%s", strings.Join(projectList, ",")))
		}
	}

	if dates, ok := plan.Filters["dates"]; ok {
		dateList := a.convertToStringSlice(dates)
		if len(dateList) > 0 {
			queryParts = append(queryParts, fmt.Sprintf("dates:%s", strings.Join(dateList, ",")))
		}
	}

	if statuses, ok := plan.Filters["statuses"]; ok {
		statusList := a.convertToStringSlice(statuses)
		if len(statusList) > 0 {
			queryParts = append(queryParts, fmt.Sprintf("statuses:%s", strings.Join(statusList, ",")))
		}
	}

	queryString := strings.Join(queryParts, " ")

	// Determine source (use first source from plan)
	source := "unknown"
	if len(plan.Sources) > 0 {
		source = plan.Sources[0]
	}

	return models.Query{
		ID:          plan.ID + "_query",
		QueryString: queryString,
		Source:      source,
		Filters:     plan.Filters,
	}
}

// convertToStringSlice converts interface{} to []string (handles []interface{} and []string).
func (a *RetrievalPlannerAgent) convertToStringSlice(data interface{}) []string {
	result := []string{}

	// Try direct conversion to []string
	if strSlice, ok := data.([]string); ok {
		return strSlice
	}

	// Try conversion from []interface{}
	if ifaceSlice, ok := data.([]interface{}); ok {
		for _, item := range ifaceSlice {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}

	return result
}

// recordAgentRun records the agent execution in the audit trail.
func (a *RetrievalPlannerAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"retrieval.plans",
			"retrieval.queries",
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
		LLMCalls:   0, // No LLM calls for rule-based planning
		Status:     status,
	}
}

// GetMetadata returns agent metadata (implements MetadataProvider).
func (a *RetrievalPlannerAgent) GetMetadata() services.AgentMetadata {
	return services.AgentMetadata{
		ID:          a.id,
		Name:        "Retrieval Planner Agent",
		Description: "Creates structured retrieval plans prioritizing structured data sources with normalized queries and filters",
		Version:     "1.0.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"retrieval", "planning", "queries", "structured-data", "rag"},
		Dependencies: []string{"intent_detection", "reasoning_structure"},
	}
}

// GetCapabilities returns agent capabilities (implements CapabilitiesProvider).
func (a *RetrievalPlannerAgent) GetCapabilities() services.AgentCapabilities {
	return services.AgentCapabilities{
		SupportsParallelExecution: false, // Must run after reasoning structure
		SupportsRetry:             true,
		RequiresLLM:               false, // Rule-based planning
		IsDeterministic:           true,  // Same intents produce same plans
		EstimatedDuration:         80,    // ~80ms for plan generation
	}
}
