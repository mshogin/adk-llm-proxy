package models

import (
	"encoding/json"
	"time"
)

// AgentContext provides versioned, namespaced context storage for agent execution.
// Each namespace is isolated and agents can only write to their designated areas.
type AgentContext struct {
	// Version of the context schema (for migrations)
	Version string `json:"version"`

	// Namespaced context data
	Metadata    *MetadataContext    `json:"metadata"`
	Reasoning   *ReasoningContext   `json:"reasoning"`
	Enrichment  *EnrichmentContext  `json:"enrichment"`
	Retrieval   *RetrievalContext   `json:"retrieval"`
	LLM         *LLMContext         `json:"llm"`
	Diagnostics *DiagnosticsContext `json:"diagnostics"`
	Audit       *AuditContext       `json:"audit"`
}

// MetadataContext holds session and trace information.
type MetadataContext struct {
	SessionID string    `json:"session_id"`
	TraceID   string    `json:"trace_id"`
	CreatedAt time.Time `json:"created_at"`
	Locale    string    `json:"locale,omitempty"`
}

// ReasoningContext stores reasoning-related data.
type ReasoningContext struct {
	// Intent detection outputs
	Intents []Intent `json:"intents,omitempty"`

	// Entity extraction outputs
	Entities map[string]interface{} `json:"entities,omitempty"`

	// Reasoning structure outputs
	Hypotheses    []Hypothesis `json:"hypotheses,omitempty"`
	Conclusions   []Conclusion `json:"conclusions,omitempty"`
	DependencyMap interface{}  `json:"dependency_map,omitempty"`

	// Inference outputs
	InferenceChain []InferenceStep `json:"inference_chain,omitempty"`
	Alternatives   []Alternative   `json:"alternatives,omitempty"`

	// Confidence scores
	ConfidenceScores map[string]float64 `json:"confidence_scores,omitempty"`

	// Summary
	Summary string `json:"summary,omitempty"`
}

// Intent represents detected user intent.
type Intent struct {
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Entities   []string `json:"entities,omitempty"`
}

// Hypothesis represents a reasoning hypothesis.
type Hypothesis struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// Conclusion represents a reasoning conclusion.
type Conclusion struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Confidence  float64  `json:"confidence"`
	Evidence    []string `json:"evidence,omitempty"`
	Intent      string   `json:"intent,omitempty"`
}

// InferenceStep represents a step in the inference chain.
type InferenceStep struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Hypothesis  string   `json:"hypothesis"`
	Evidence    []string `json:"evidence,omitempty"`
	Confidence  float64  `json:"confidence"`
}

// Alternative represents an alternative interpretation.
type Alternative struct {
	ID          string  `json:"id"`
	Conclusion  string  `json:"conclusion"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
}

// EnrichmentContext stores facts and derived knowledge.
type EnrichmentContext struct {
	Facts            []Fact          `json:"facts,omitempty"`
	DerivedKnowledge []Knowledge     `json:"derived_knowledge,omitempty"`
	Relationships    []Relationship  `json:"relationships,omitempty"`
	ContextLinks     []string        `json:"context_links,omitempty"`
}

// Fact represents a factual piece of information.
type Fact struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Source     string                 `json:"source"`
	Timestamp  time.Time              `json:"timestamp"`
	Confidence float64                `json:"confidence"`
	Provenance map[string]interface{} `json:"provenance,omitempty"`
}

// Knowledge represents derived knowledge.
type Knowledge struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	DerivedFrom []string `json:"derived_from,omitempty"`
}

// Relationship represents a relationship between entities.
type Relationship struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// RetrievalContext stores retrieval plans and queries.
type RetrievalContext struct {
	Plans     []RetrievalPlan `json:"plans,omitempty"`
	Queries   []Query         `json:"queries,omitempty"`
	Artifacts []Artifact      `json:"artifacts,omitempty"`
}

// RetrievalPlan represents a plan for retrieving information.
type RetrievalPlan struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Sources     []string               `json:"sources,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Priority    int                    `json:"priority"`
}

// Query represents a search query.
type Query struct {
	ID          string                 `json:"id"`
	QueryString string                 `json:"query_string"`
	Source      string                 `json:"source"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Results     interface{}            `json:"results,omitempty"`
}

// Artifact represents a retrieved artifact.
type Artifact struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
	Source  string      `json:"source"`
}

// LLMContext stores LLM-related information.
type LLMContext struct {
	Provider  string                 `json:"provider,omitempty"`
	Model     string                 `json:"model,omitempty"`
	Usage     *LLMUsage              `json:"usage,omitempty"`
	Decisions []LLMDecision          `json:"decisions,omitempty"`
	Cache     map[string]interface{} `json:"cache,omitempty"`
}

// LLMUsage tracks token usage and costs.
type LLMUsage struct {
	TotalTokens      int                `json:"total_tokens"`
	PromptTokens     int                `json:"prompt_tokens"`
	CompletionTokens int                `json:"completion_tokens"`
	CostUSD          float64            `json:"cost_usd"`
	ByAgent          map[string]float64 `json:"by_agent,omitempty"`
}

// LLMDecision logs why a particular model was selected.
type LLMDecision struct {
	Timestamp  time.Time `json:"timestamp"`
	AgentID    string    `json:"agent_id"`
	TaskType   string    `json:"task_type"`
	Selected   string    `json:"selected"`
	Reason     string    `json:"reason"`
	Complexity string    `json:"complexity,omitempty"`
}

// DiagnosticsContext stores errors, warnings, and performance data.
type DiagnosticsContext struct {
	Errors            []ErrorReport      `json:"errors,omitempty"`
	Warnings          []Warning          `json:"warnings,omitempty"`
	Performance       *PerformanceData   `json:"performance,omitempty"`
	ValidationReports []ValidationReport `json:"validation_reports,omitempty"`
}

// ErrorReport represents an error that occurred during processing.
type ErrorReport struct {
	Timestamp time.Time `json:"timestamp"`
	AgentID   string    `json:"agent_id"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Details   interface{} `json:"details,omitempty"`
}

// Warning represents a warning.
type Warning struct {
	Timestamp time.Time `json:"timestamp"`
	AgentID   string    `json:"agent_id"`
	Message   string    `json:"message"`
}

// PerformanceData tracks performance metrics.
type PerformanceData struct {
	TotalDurationMS int64                     `json:"total_duration_ms"`
	AgentMetrics    map[string]*AgentMetrics  `json:"agent_metrics,omitempty"`
}

// AgentMetrics tracks metrics for a single agent.
type AgentMetrics struct {
	DurationMS int64   `json:"duration_ms"`
	LLMCalls   int     `json:"llm_calls"`
	Status     string  `json:"status"`
	Tokens     int     `json:"tokens,omitempty"`
	Cost       float64 `json:"cost,omitempty"`
}

// ValidationReport represents a validation report.
type ValidationReport struct {
	Timestamp time.Time `json:"timestamp"`
	AgentID   string    `json:"agent_id"`
	Passed    bool      `json:"passed"`
	Issues    []string  `json:"issues,omitempty"`
	AutoFixes []string  `json:"auto_fixes,omitempty"`
}

// AuditContext stores audit trail of all agent runs and changes.
type AuditContext struct {
	AgentRuns []AgentRun    `json:"agent_runs,omitempty"`
	Diffs     []ContextDiff `json:"diffs,omitempty"`
}

// AgentRun represents a single agent execution.
type AgentRun struct {
	Timestamp    time.Time `json:"timestamp"`
	AgentID      string    `json:"agent_id"`
	Status       string    `json:"status"`
	DurationMS   int64     `json:"duration_ms"`
	KeysWritten  []string  `json:"keys_written,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// ContextDiff represents a diff between context states.
type ContextDiff struct {
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id"`
	Changes   map[string]interface{} `json:"changes"`
}

// NewAgentContext creates a new AgentContext with initialized namespaces.
func NewAgentContext(sessionID, traceID string) *AgentContext {
	now := time.Now()
	return &AgentContext{
		Version: "1.0.0",
		Metadata: &MetadataContext{
			SessionID: sessionID,
			TraceID:   traceID,
			CreatedAt: now,
		},
		Reasoning:   &ReasoningContext{},
		Enrichment:  &EnrichmentContext{},
		Retrieval:   &RetrievalContext{},
		LLM:         &LLMContext{Usage: &LLMUsage{}},
		Diagnostics: &DiagnosticsContext{Performance: &PerformanceData{}},
		Audit:       &AuditContext{},
	}
}

// Clone creates a deep copy of the AgentContext.
func (c *AgentContext) Clone() (*AgentContext, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	var clone AgentContext
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, err
	}

	return &clone, nil
}

// Serialize converts the AgentContext to JSON bytes.
func (c *AgentContext) Serialize() ([]byte, error) {
	return json.Marshal(c)
}

// Deserialize loads an AgentContext from JSON bytes.
func Deserialize(data []byte) (*AgentContext, error) {
	var ctx AgentContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, err
	}
	return &ctx, nil
}
