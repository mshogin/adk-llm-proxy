package fixtures

import (
	"time"

	"github.com/mshogin/agents/internal/domain/models"
)

// FixedTime provides a consistent timestamp for testing
var FixedTime = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

// EmptyContext creates an AgentContext with only metadata initialized
func EmptyContext() *models.AgentContext {
	return &models.AgentContext{
		Version: "1.0.0",
		Metadata: &models.MetadataContext{
			SessionID: "session-empty",
			TraceID:   "trace-empty",
			CreatedAt: FixedTime,
		},
		Reasoning:   &models.ReasoningContext{},
		Enrichment:  &models.EnrichmentContext{},
		Retrieval:   &models.RetrievalContext{},
		LLM:         &models.LLMContext{Usage: &models.LLMUsage{}},
		Diagnostics: &models.DiagnosticsContext{Performance: &models.PerformanceData{}},
		Audit:       &models.AuditContext{},
	}
}

// MinimalContext creates an AgentContext with basic metadata
func MinimalContext() *models.AgentContext {
	return &models.AgentContext{
		Version: "1.0.0",
		Metadata: &models.MetadataContext{
			SessionID: "session-minimal",
			TraceID:   "trace-minimal",
			CreatedAt: FixedTime,
			Locale:    "en-US",
		},
		Reasoning:   &models.ReasoningContext{},
		Enrichment:  &models.EnrichmentContext{},
		Retrieval:   &models.RetrievalContext{},
		LLM:         &models.LLMContext{Usage: &models.LLMUsage{}},
		Diagnostics: &models.DiagnosticsContext{Performance: &models.PerformanceData{}},
		Audit:       &models.AuditContext{},
	}
}

// ContextWithIntent creates an AgentContext with detected intents
func ContextWithIntent() *models.AgentContext {
	ctx := EmptyContext()
	ctx.Metadata.SessionID = "session-intent"
	ctx.Metadata.TraceID = "trace-intent"
	ctx.Reasoning.Intents = []models.Intent{
		{
			Type:       "query",
			Confidence: 0.95,
			Entities:   []string{"user_id", "timeframe"},
		},
	}
	ctx.Reasoning.Entities = map[string]interface{}{
		"user_id":   "user123",
		"timeframe": "last_week",
	}
	return ctx
}

// ContextWithMultipleIntents creates an AgentContext with multiple competing intents
func ContextWithMultipleIntents() *models.AgentContext {
	ctx := EmptyContext()
	ctx.Metadata.SessionID = "session-multi-intent"
	ctx.Metadata.TraceID = "trace-multi-intent"
	ctx.Reasoning.Intents = []models.Intent{
		{
			Type:       "query",
			Confidence: 0.85,
			Entities:   []string{"user_id"},
		},
		{
			Type:       "command",
			Confidence: 0.75,
			Entities:   []string{"action", "target"},
		},
		{
			Type:       "clarification",
			Confidence: 0.40,
			Entities:   []string{},
		},
	}
	return ctx
}

// ContextWithReasoningChain creates an AgentContext with a full reasoning chain
func ContextWithReasoningChain() *models.AgentContext {
	ctx := ContextWithIntent()
	ctx.Metadata.SessionID = "session-reasoning"
	ctx.Metadata.TraceID = "trace-reasoning"

	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{
			ID:           "hyp-1",
			Description:  "User wants to query recent activity",
			Dependencies: []string{},
		},
		{
			ID:           "hyp-2",
			Description:  "User wants aggregated metrics for last week",
			Dependencies: []string{"hyp-1"},
		},
	}

	ctx.Reasoning.InferenceChain = []models.InferenceStep{
		{
			ID:          "inf-1",
			Description: "Detected time-based query intent",
			Hypothesis:  "hyp-1",
			Evidence:    []string{"timeframe entity", "query verb"},
			Confidence:  0.90,
		},
		{
			ID:          "inf-2",
			Description: "Requires aggregation of user metrics",
			Hypothesis:  "hyp-2",
			Evidence:    []string{"user_id entity", "timeframe=last_week"},
			Confidence:  0.85,
		},
	}

	ctx.Reasoning.Conclusions = []models.Conclusion{
		{
			ID:          "conc-1",
			Description: "Execute user metrics query for last week",
			Confidence:  0.90,
			Evidence:    []string{"inf-1", "inf-2"},
			Intent:      "query",
		},
	}

	ctx.Reasoning.ConfidenceScores = map[string]float64{
		"intent_detection": 0.95,
		"reasoning":        0.88,
		"inference":        0.87,
	}

	ctx.Reasoning.Summary = "User requesting aggregated metrics for user123 over the last week"

	return ctx
}

// ContextWithAlternatives creates an AgentContext with multiple alternative interpretations
func ContextWithAlternatives() *models.AgentContext {
	ctx := ContextWithReasoningChain()
	ctx.Metadata.SessionID = "session-alternatives"
	ctx.Metadata.TraceID = "trace-alternatives"

	ctx.Reasoning.Alternatives = []models.Alternative{
		{
			ID:          "alt-1",
			Conclusion:  "conc-1",
			Description: "Query might be for current week instead of last week",
			Confidence:  0.65,
		},
		{
			ID:          "alt-2",
			Conclusion:  "conc-1",
			Description: "User might want real-time data instead of historical",
			Confidence:  0.55,
		},
	}

	return ctx
}

// ContextWithEnrichment creates an AgentContext with facts and derived knowledge
func ContextWithEnrichment() *models.AgentContext {
	ctx := ContextWithReasoningChain()
	ctx.Metadata.SessionID = "session-enrichment"
	ctx.Metadata.TraceID = "trace-enrichment"

	ctx.Enrichment.Facts = []models.Fact{
		{
			ID:         "fact-1",
			Content:    "User123 is an active premium subscriber",
			Source:     "user_database",
			Timestamp:  FixedTime,
			Confidence: 1.0,
			Provenance: map[string]interface{}{
				"table": "users",
				"query": "SELECT * FROM users WHERE id = 'user123'",
			},
		},
		{
			ID:         "fact-2",
			Content:    "Last week had 7 active days for user123",
			Source:     "activity_logs",
			Timestamp:  FixedTime.Add(1 * time.Minute),
			Confidence: 0.98,
		},
	}

	ctx.Enrichment.DerivedKnowledge = []models.Knowledge{
		{
			ID:          "know-1",
			Content:     "User is eligible for detailed analytics",
			DerivedFrom: []string{"fact-1"},
		},
		{
			ID:          "know-2",
			Content:     "Query timeframe contains complete data",
			DerivedFrom: []string{"fact-2"},
		},
	}

	ctx.Enrichment.Relationships = []models.Relationship{
		{
			From: "user123",
			To:   "premium_tier",
			Type: "member_of",
		},
		{
			From: "user123",
			To:   "activity_logs",
			Type: "has_data_in",
		},
	}

	ctx.Enrichment.ContextLinks = []string{
		"session:session-reasoning",
		"user:user123",
	}

	return ctx
}

// ContextWithRetrieval creates an AgentContext with retrieval plans and queries
func ContextWithRetrieval() *models.AgentContext {
	ctx := ContextWithEnrichment()
	ctx.Metadata.SessionID = "session-retrieval"
	ctx.Metadata.TraceID = "trace-retrieval"

	ctx.Retrieval.Plans = []models.RetrievalPlan{
		{
			ID:          "plan-1",
			Description: "Retrieve user activity logs for last week",
			Sources:     []string{"activity_database", "event_stream"},
			Filters: map[string]interface{}{
				"user_id":    "user123",
				"start_date": "2024-01-08",
				"end_date":   "2024-01-14",
			},
			Priority: 1,
		},
		{
			ID:          "plan-2",
			Description: "Retrieve user profile metadata",
			Sources:     []string{"user_database"},
			Filters: map[string]interface{}{
				"user_id": "user123",
			},
			Priority: 2,
		},
	}

	ctx.Retrieval.Queries = []models.Query{
		{
			ID:          "query-1",
			QueryString: "SELECT * FROM activity_logs WHERE user_id = ? AND date >= ? AND date <= ?",
			Source:      "activity_database",
			Filters: map[string]interface{}{
				"params": []string{"user123", "2024-01-08", "2024-01-14"},
			},
			Results: map[string]interface{}{
				"rows":  142,
				"bytes": 18432,
			},
		},
	}

	ctx.Retrieval.Artifacts = []models.Artifact{
		{
			ID:   "artifact-1",
			Type: "query_results",
			Content: map[string]interface{}{
				"total_sessions": 42,
				"total_duration": 15620,
				"avg_duration":   372,
			},
			Source: "activity_database",
		},
	}

	return ctx
}

// ContextWithLLMUsage creates an AgentContext with LLM usage and decisions
func ContextWithLLMUsage() *models.AgentContext {
	ctx := ContextWithIntent()
	ctx.Metadata.SessionID = "session-llm"
	ctx.Metadata.TraceID = "trace-llm"

	ctx.LLM.Provider = "openai"
	ctx.LLM.Model = "gpt-4"
	ctx.LLM.Usage = &models.LLMUsage{
		TotalTokens:      5430,
		PromptTokens:     3200,
		CompletionTokens: 2230,
		CostUSD:          0.1629,
		ByAgent: map[string]float64{
			"intent_detection": 0.0320,
			"reasoning":        0.0680,
			"inference":        0.0629,
		},
	}

	ctx.LLM.Decisions = []models.LLMDecision{
		{
			Timestamp:  FixedTime,
			AgentID:    "intent_detection",
			TaskType:   "classification",
			Selected:   "gpt-3.5-turbo",
			Reason:     "Simple classification task, cost optimization",
			Complexity: "low",
		},
		{
			Timestamp:  FixedTime.Add(2 * time.Second),
			AgentID:    "reasoning",
			TaskType:   "reasoning",
			Selected:   "gpt-4",
			Reason:     "Complex multi-step reasoning required",
			Complexity: "high",
		},
	}

	ctx.LLM.Cache = map[string]interface{}{
		"intent:query:user_metrics": map[string]interface{}{
			"hit":       true,
			"timestamp": FixedTime.Unix(),
		},
	}

	return ctx
}

// ContextWithErrors creates an AgentContext with errors and warnings
func ContextWithErrors() *models.AgentContext {
	ctx := EmptyContext()
	ctx.Metadata.SessionID = "session-errors"
	ctx.Metadata.TraceID = "trace-errors"

	ctx.Diagnostics.Errors = []models.ErrorReport{
		{
			Timestamp: FixedTime,
			AgentID:   "intent_detection",
			Message:   "Failed to parse user input",
			Severity:  "high",
			Details: map[string]interface{}{
				"input_length": 0,
				"error_code":   "EMPTY_INPUT",
			},
		},
		{
			Timestamp: FixedTime.Add(1 * time.Second),
			AgentID:   "reasoning",
			Message:   "Missing required entities for reasoning",
			Severity:  "critical",
			Details: map[string]interface{}{
				"required": []string{"user_id", "action"},
				"found":    []string{},
			},
		},
	}

	ctx.Diagnostics.Warnings = []models.Warning{
		{
			Timestamp: FixedTime,
			AgentID:   "intent_detection",
			Message:   "Low confidence score, consider fallback",
		},
		{
			Timestamp: FixedTime.Add(500 * time.Millisecond),
			AgentID:   "reasoning",
			Message:   "Missing contextual information",
		},
	}

	return ctx
}

// ContextWithValidationIssues creates an AgentContext with validation reports
func ContextWithValidationIssues() *models.AgentContext {
	ctx := ContextWithIntent()
	ctx.Metadata.SessionID = "session-validation"
	ctx.Metadata.TraceID = "trace-validation"

	ctx.Diagnostics.ValidationReports = []models.ValidationReport{
		{
			Timestamp: FixedTime,
			AgentID:   "intent_detection",
			Passed:    false,
			Issues: []string{
				"Confidence score below threshold (0.65 < 0.70)",
				"Missing required entity: action",
			},
			AutoFixes: []string{
				"Requested clarification from user",
			},
		},
		{
			Timestamp: FixedTime.Add(2 * time.Second),
			AgentID:   "reasoning",
			Passed:    true,
			Issues:    []string{},
			AutoFixes: []string{},
		},
	}

	return ctx
}

// ContextWithPerformanceMetrics creates an AgentContext with detailed performance data
func ContextWithPerformanceMetrics() *models.AgentContext {
	ctx := ContextWithReasoningChain()
	ctx.Metadata.SessionID = "session-perf"
	ctx.Metadata.TraceID = "trace-perf"

	ctx.Diagnostics.Performance = &models.PerformanceData{
		TotalDurationMS: 3420,
		AgentMetrics: map[string]*models.AgentMetrics{
			"intent_detection": {
				DurationMS: 245,
				LLMCalls:   1,
				Status:     "completed",
				Tokens:     856,
				Cost:       0.0086,
			},
			"reasoning": {
				DurationMS: 1850,
				LLMCalls:   3,
				Status:     "completed",
				Tokens:     3240,
				Cost:       0.0972,
			},
			"inference": {
				DurationMS: 1325,
				LLMCalls:   2,
				Status:     "completed",
				Tokens:     2156,
				Cost:       0.0647,
			},
		},
	}

	return ctx
}

// ContextWithAuditTrail creates an AgentContext with complete audit trail
func ContextWithAuditTrail() *models.AgentContext {
	ctx := ContextWithReasoningChain()
	ctx.Metadata.SessionID = "session-audit"
	ctx.Metadata.TraceID = "trace-audit"

	ctx.Audit.AgentRuns = []models.AgentRun{
		{
			Timestamp:  FixedTime,
			AgentID:    "intent_detection",
			Status:     "completed",
			DurationMS: 245,
			KeysWritten: []string{
				"reasoning.intents",
				"reasoning.entities",
			},
		},
		{
			Timestamp:  FixedTime.Add(245 * time.Millisecond),
			AgentID:    "reasoning",
			Status:     "completed",
			DurationMS: 1850,
			KeysWritten: []string{
				"reasoning.hypotheses",
				"reasoning.inference_chain",
				"reasoning.conclusions",
			},
		},
		{
			Timestamp:  FixedTime.Add(2095 * time.Millisecond),
			AgentID:    "inference",
			Status:     "completed",
			DurationMS: 1325,
			KeysWritten: []string{
				"reasoning.confidence_scores",
				"reasoning.summary",
			},
		},
	}

	ctx.Audit.Diffs = []models.ContextDiff{
		{
			Timestamp: FixedTime,
			AgentID:   "intent_detection",
			Changes: map[string]interface{}{
				"reasoning.intents": []interface{}{
					map[string]interface{}{
						"type":       "query",
						"confidence": 0.95,
					},
				},
			},
		},
		{
			Timestamp: FixedTime.Add(245 * time.Millisecond),
			AgentID:   "reasoning",
			Changes: map[string]interface{}{
				"reasoning.hypotheses": []interface{}{
					map[string]interface{}{
						"id":          "hyp-1",
						"description": "User wants to query recent activity",
					},
				},
			},
		},
	}

	return ctx
}

// ContextWithFailedAgentRun creates an AgentContext with failed agent execution
func ContextWithFailedAgentRun() *models.AgentContext {
	ctx := EmptyContext()
	ctx.Metadata.SessionID = "session-failed"
	ctx.Metadata.TraceID = "trace-failed"

	ctx.Audit.AgentRuns = []models.AgentRun{
		{
			Timestamp:   FixedTime,
			AgentID:     "intent_detection",
			Status:      "completed",
			DurationMS:  245,
			KeysWritten: []string{"reasoning.intents"},
		},
		{
			Timestamp:   FixedTime.Add(245 * time.Millisecond),
			AgentID:     "reasoning",
			Status:      "failed",
			DurationMS:  520,
			KeysWritten: []string{},
			Error:       "Missing required context: intents are below confidence threshold",
		},
	}

	ctx.Diagnostics.Errors = []models.ErrorReport{
		{
			Timestamp: FixedTime.Add(245 * time.Millisecond),
			AgentID:   "reasoning",
			Message:   "Agent execution failed",
			Severity:  "critical",
			Details: map[string]interface{}{
				"error": "Missing required context: intents are below confidence threshold",
			},
		},
	}

	return ctx
}

// ComplexContext creates a fully populated AgentContext with all fields
func ComplexContext() *models.AgentContext {
	ctx := ContextWithReasoningChain()
	ctx.Metadata.SessionID = "session-complex"
	ctx.Metadata.TraceID = "trace-complex"
	ctx.Metadata.Locale = "en-US"

	// Add enrichment
	ctx.Enrichment = ContextWithEnrichment().Enrichment

	// Add retrieval
	ctx.Retrieval = ContextWithRetrieval().Retrieval

	// Add LLM usage
	ctx.LLM = ContextWithLLMUsage().LLM

	// Add performance
	ctx.Diagnostics.Performance = ContextWithPerformanceMetrics().Diagnostics.Performance

	// Add audit
	ctx.Audit = ContextWithAuditTrail().Audit

	// Add validation
	ctx.Diagnostics.ValidationReports = []models.ValidationReport{
		{
			Timestamp: FixedTime.Add(3 * time.Second),
			AgentID:   "validation",
			Passed:    true,
			Issues:    []string{},
			AutoFixes: []string{},
		},
	}

	return ctx
}

// ContextWithArtifacts creates an AgentContext with multiple artifacts
func ContextWithArtifacts() *models.AgentContext {
	ctx := ContextWithReasoningChain()
	ctx.Metadata.SessionID = "session-artifacts"
	ctx.Metadata.TraceID = "trace-artifacts"

	ctx.Reasoning.Artifacts = []models.Artifact{
		{
			ID:   "summary-1",
			Type: "text_summary",
			Content: map[string]interface{}{
				"summary":    "User requesting aggregated metrics",
				"word_count": 42,
			},
			Source: "summarization_agent",
		},
		{
			ID:   "chart-1",
			Type: "visualization",
			Content: map[string]interface{}{
				"chart_type": "bar",
				"data_points": 7,
			},
			Source: "visualization_agent",
		},
	}

	ctx.Retrieval.Artifacts = []models.Artifact{
		{
			ID:   "data-1",
			Type: "query_results",
			Content: map[string]interface{}{
				"total_rows": 1542,
				"columns":    []string{"date", "user_id", "action", "duration"},
			},
			Source: "database",
		},
	}

	return ctx
}
