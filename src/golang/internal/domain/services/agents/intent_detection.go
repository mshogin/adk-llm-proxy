package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services"
)

// IntentDetectionAgent detects user intent and extracts entities from input messages.
//
// Design Principles:
// - Rule-based classification for speed and determinism
// - Entity extraction for structured data (projects, dates, providers, statuses)
// - Confidence scoring for reliability tracking
// - Optional LLM fallback for low-confidence cases (< 0.8)
//
// Supported Intent Types:
// - query_commits: Ask about commits, code changes, or version history
// - query_issues: Ask about tasks, bugs, or issue tracking
// - query_analytics: Request statistics, metrics, or trends
// - query_status: Check project status, health, or progress
// - command_action: Execute an action (deploy, restart, update)
// - request_help: Ask for help or documentation
// - conversation: General conversation or unclear intent
//
// Extracted Entities:
// - projects: Project names (gitlab-repo, youtrack-project)
// - dates: Time references (today, last week, 2024-01-15)
// - providers: Service providers (gitlab, youtrack, openai)
// - statuses: Status indicators (open, closed, in-progress, failed)
type IntentDetectionAgent struct {
	// Agent configuration
	id                  string
	confidenceThreshold float64 // Threshold for LLM fallback (default 0.8)

	// Intent patterns (compiled at initialization)
	intentPatterns map[string]*intentPattern
}

// intentPattern defines a pattern for detecting an intent.
type intentPattern struct {
	intentType string
	keywords   []string         // Keywords that indicate this intent
	patterns   []*regexp.Regexp // Regex patterns for matching
	weight     float64          // Weight for confidence calculation (0.0-1.0)
}

// NewIntentDetectionAgent creates a new intent detection agent.
func NewIntentDetectionAgent() *IntentDetectionAgent {
	agent := &IntentDetectionAgent{
		id:                  "intent_detection",
		confidenceThreshold: 0.8,
		intentPatterns:      make(map[string]*intentPattern),
	}

	agent.initializePatterns()
	return agent
}

// AgentID returns the unique identifier for this agent.
func (a *IntentDetectionAgent) AgentID() string {
	return a.id
}

// Preconditions returns the list of context keys required before execution.
// Intent detection is the first agent, so it has no preconditions.
func (a *IntentDetectionAgent) Preconditions() []string {
	return []string{} // First agent - no preconditions
}

// Postconditions returns the list of context keys guaranteed after execution.
func (a *IntentDetectionAgent) Postconditions() []string {
	return []string{
		"reasoning.intents",
		"reasoning.entities",
		"reasoning.confidence_scores",
	}
}

// Execute runs intent detection on the input context.
func (a *IntentDetectionAgent) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	startTime := time.Now()

	// Clone context to avoid modifying input
	newContext, err := agentContext.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone context: %w", err)
	}

	// Initialize reasoning context if needed
	if newContext.Reasoning == nil {
		newContext.Reasoning = &models.ReasoningContext{}
	}

	// Extract input text from metadata or use placeholder
	inputText := a.extractInputText(newContext)
	if inputText == "" {
		return nil, fmt.Errorf("no input text found in context")
	}

	// Detect intents using rule-based classification
	intents := a.detectIntents(inputText)

	// Extract entities from input text
	entities := a.extractEntities(inputText)

	// Calculate confidence scores
	confidenceScores := a.calculateConfidenceScores(intents)

	// TODO: Implement LLM fallback for low-confidence cases
	// This will be implemented in a later task when integrating with LLM orchestrator
	// For now, we rely solely on rule-based classification

	// Write results to context
	newContext.Reasoning.Intents = intents
	newContext.Reasoning.Entities = entities
	newContext.Reasoning.ConfidenceScores = confidenceScores

	// Track agent execution in audit
	duration := time.Since(startTime)
	a.recordAgentRun(newContext, duration, "success", nil)

	return newContext, nil
}

// initializePatterns sets up all intent detection patterns.
func (a *IntentDetectionAgent) initializePatterns() {
	// Query Commits Intent
	a.intentPatterns["query_commits"] = &intentPattern{
		intentType: "query_commits",
		keywords: []string{
			"commit", "commits", "changes", "changelog", "version history",
			"git", "repository", "code change", "pushed", "merged",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(latest|recent|last)\s+(commit|change)s?\b`),
			regexp.MustCompile(`(?i)\bwhat('s| is| are)?\s+(changed|new|updated)\b`),
			regexp.MustCompile(`(?i)\b(show|list|get)\s+commit`),
		},
		weight: 0.9,
	}

	// Query Issues Intent
	a.intentPatterns["query_issues"] = &intentPattern{
		intentType: "query_issues",
		keywords: []string{
			"issue", "issues", "task", "tasks", "bug", "bugs",
			"ticket", "tickets", "story", "stories", "epic",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(open|active|pending|closed)\s+(issue|task|bug)s?\b`),
			regexp.MustCompile(`(?i)\b(show|list|get)\s+(issue|task|bug)s?\b`),
			regexp.MustCompile(`(?i)\bwhat('s| is| are)?\s+(my|our)?\s+(task|issue|bug)s?\b`),
		},
		weight: 0.9,
	}

	// Query Analytics Intent
	a.intentPatterns["query_analytics"] = &intentPattern{
		intentType: "query_analytics",
		keywords: []string{
			"statistics", "stats", "metrics", "analytics", "report",
			"trend", "trends", "performance", "count", "total",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(how many|count|total|number of)\b`),
			regexp.MustCompile(`(?i)\b(statistics|stats|metrics|analytics)\b`),
			regexp.MustCompile(`(?i)\b(trend|pattern|over time)\b`),
		},
		weight: 0.85,
	}

	// Query Status Intent
	a.intentPatterns["query_status"] = &intentPattern{
		intentType: "query_status",
		keywords: []string{
			"status", "health", "progress", "state", "condition",
			"running", "stopped", "failed", "ok", "ready",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bwhat('s| is)?\s+the\s+status\b`),
			regexp.MustCompile(`(?i)\bis\s+(it|everything|system)\s+(ok|running|working|healthy)\b`),
			regexp.MustCompile(`(?i)\bcheck\s+status\b`),
		},
		weight: 0.85,
	}

	// Command Action Intent
	a.intentPatterns["command_action"] = &intentPattern{
		intentType: "command_action",
		keywords: []string{
			"deploy", "restart", "start", "stop", "update", "upgrade",
			"create", "delete", "modify", "execute", "run",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(please|can you|could you)?\s*(deploy|restart|start|stop|update)\b`),
			regexp.MustCompile(`(?i)\b(create|delete|modify|execute|run)\s+`),
		},
		weight: 0.9,
	}

	// Request Help Intent
	a.intentPatterns["request_help"] = &intentPattern{
		intentType: "request_help",
		keywords: []string{
			"help", "how", "what", "explain", "documentation",
			"guide", "tutorial", "instructions", "support",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(help|how do|how to|can you help)\b`),
			regexp.MustCompile(`(?i)\b(what is|what are|explain|tell me about)\b`),
			regexp.MustCompile(`(?i)\b(documentation|guide|tutorial|instructions)\b`),
		},
		weight: 0.8,
	}

	// Conversation Intent (fallback)
	a.intentPatterns["conversation"] = &intentPattern{
		intentType: "conversation",
		keywords: []string{
			"hello", "hi", "thanks", "thank you", "bye", "goodbye",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(hello|hi|hey|greetings)\b`),
			regexp.MustCompile(`(?i)\b(thanks|thank you|bye|goodbye)\b`),
		},
		weight: 0.6,
	}
}

// detectIntents analyzes input text and returns detected intents with confidence scores.
func (a *IntentDetectionAgent) detectIntents(inputText string) []models.Intent {
	lowerInput := strings.ToLower(inputText)
	intentScores := make(map[string]float64)

	// Check each intent pattern
	for intentType, pattern := range a.intentPatterns {
		score := 0.0

		// Check keywords
		keywordMatches := 0
		for _, keyword := range pattern.keywords {
			if strings.Contains(lowerInput, keyword) {
				keywordMatches++
			}
		}
		if keywordMatches > 0 {
			score += float64(keywordMatches) * 0.3 // Each keyword adds 0.3
		}

		// Check regex patterns
		patternMatches := 0
		for _, regex := range pattern.patterns {
			if regex.MatchString(inputText) {
				patternMatches++
			}
		}
		if patternMatches > 0 {
			score += float64(patternMatches) * 0.5 // Each pattern match adds 0.5
		}

		// Apply pattern weight
		score *= pattern.weight

		// Normalize to 0.0-1.0 range
		if score > 1.0 {
			score = 1.0
		}

		if score > 0.0 {
			intentScores[intentType] = score
		}
	}

	// If no intents detected, default to conversation
	if len(intentScores) == 0 {
		intentScores["conversation"] = 0.5
	}

	// Convert to Intent objects, sorted by confidence
	intents := make([]models.Intent, 0, len(intentScores))
	for intentType, confidence := range intentScores {
		intents = append(intents, models.Intent{
			Type:       intentType,
			Confidence: confidence,
		})
	}

	// Sort by confidence (highest first)
	for i := 0; i < len(intents)-1; i++ {
		for j := i + 1; j < len(intents); j++ {
			if intents[j].Confidence > intents[i].Confidence {
				intents[i], intents[j] = intents[j], intents[i]
			}
		}
	}

	return intents
}

// extractEntities extracts structured entities from input text.
func (a *IntentDetectionAgent) extractEntities(inputText string) map[string]interface{} {
	entities := make(map[string]interface{})

	// Extract projects
	projects := a.extractProjects(inputText)
	if len(projects) > 0 {
		entities["projects"] = projects
	}

	// Extract dates
	dates := a.extractDates(inputText)
	if len(dates) > 0 {
		entities["dates"] = dates
	}

	// Extract providers
	providers := a.extractProviders(inputText)
	if len(providers) > 0 {
		entities["providers"] = providers
	}

	// Extract statuses
	statuses := a.extractStatuses(inputText)
	if len(statuses) > 0 {
		entities["statuses"] = statuses
	}

	return entities
}

// extractProjects extracts project names from input text.
func (a *IntentDetectionAgent) extractProjects(inputText string) []string {
	projects := []string{}
	lowerInput := strings.ToLower(inputText)

	// Common project name patterns
	projectPatterns := []string{
		"project", "repo", "repository", "module", "service",
	}

	for _, pattern := range projectPatterns {
		regex := regexp.MustCompile(fmt.Sprintf(`(?i)\b%s[:\s]+([a-zA-Z0-9\-_]+)\b`, pattern))
		matches := regex.FindAllStringSubmatch(inputText, -1)
		for _, match := range matches {
			if len(match) > 1 {
				projects = append(projects, match[1])
			}
		}
	}

	// Also check for explicit project names (gitlab-, youtrack-, etc.)
	explicitRegex := regexp.MustCompile(`\b([a-zA-Z0-9\-_]*(?:gitlab|youtrack|mcp)[a-zA-Z0-9\-_]*)\b`)
	matches := explicitRegex.FindAllString(lowerInput, -1)
	projects = append(projects, matches...)

	return a.deduplicateStrings(projects)
}

// extractDates extracts date references from input text.
func (a *IntentDetectionAgent) extractDates(inputText string) []string {
	dates := []string{}
	lowerInput := strings.ToLower(inputText)

	// Relative date keywords
	relativeDates := []string{
		"today", "yesterday", "tomorrow",
		"last week", "this week", "next week",
		"last month", "this month", "next month",
		"last year", "this year",
	}

	for _, dateRef := range relativeDates {
		if strings.Contains(lowerInput, dateRef) {
			dates = append(dates, dateRef)
		}
	}

	// Absolute date patterns (ISO format, US format)
	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),               // 2024-01-15
		regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{4}\b`),           // 01/15/2024
		regexp.MustCompile(`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2},?\s+\d{4}\b`), // Jan 15, 2024
	}

	for _, regex := range datePatterns {
		matches := regex.FindAllString(inputText, -1)
		dates = append(dates, matches...)
	}

	return a.deduplicateStrings(dates)
}

// extractProviders extracts service provider names from input text.
func (a *IntentDetectionAgent) extractProviders(inputText string) []string {
	providers := []string{}
	lowerInput := strings.ToLower(inputText)

	// Known providers
	knownProviders := []string{
		"gitlab", "youtrack", "openai", "anthropic", "deepseek", "ollama",
		"github", "jira", "confluence",
	}

	for _, provider := range knownProviders {
		if strings.Contains(lowerInput, provider) {
			providers = append(providers, provider)
		}
	}

	return a.deduplicateStrings(providers)
}

// extractStatuses extracts status indicators from input text.
func (a *IntentDetectionAgent) extractStatuses(inputText string) []string {
	statuses := []string{}
	lowerInput := strings.ToLower(inputText)

	// Known status keywords
	knownStatuses := []string{
		"open", "closed", "in-progress", "in progress", "pending",
		"resolved", "done", "failed", "error", "success",
		"active", "inactive", "blocked", "ready", "draft",
	}

	for _, status := range knownStatuses {
		if strings.Contains(lowerInput, status) {
			statuses = append(statuses, status)
		}
	}

	return a.deduplicateStrings(statuses)
}

// calculateConfidenceScores calculates overall confidence scores.
func (a *IntentDetectionAgent) calculateConfidenceScores(intents []models.Intent) map[string]float64 {
	scores := make(map[string]float64)

	if len(intents) > 0 {
		// Primary intent confidence
		scores["primary_intent"] = intents[0].Confidence

		// Overall confidence (average of top 2 intents)
		if len(intents) > 1 {
			scores["overall"] = (intents[0].Confidence + intents[1].Confidence) / 2.0
		} else {
			scores["overall"] = intents[0].Confidence
		}
	} else {
		scores["primary_intent"] = 0.0
		scores["overall"] = 0.0
	}

	return scores
}

// extractInputText extracts input text from agent context.
func (a *IntentDetectionAgent) extractInputText(ctx *models.AgentContext) string {
	// Try to get input from metadata or retrieval context
	// For now, use a placeholder - in production this would come from the request

	// TODO: Define standard location for input text in AgentContext
	// Options:
	// 1. ctx.Metadata.InputText
	// 2. ctx.Retrieval.Query
	// 3. Passed as parameter to Execute()

	// For now, check if there's a query in retrieval context
	if ctx.Retrieval != nil && len(ctx.Retrieval.Queries) > 0 {
		return ctx.Retrieval.Queries[0].QueryString
	}

	// Return empty string if no input found
	return ""
}

// recordAgentRun records the agent execution in the audit trail.
func (a *IntentDetectionAgent) recordAgentRun(ctx *models.AgentContext, duration time.Duration, status string, err error) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    a.id,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		KeysWritten: []string{
			"reasoning.intents",
			"reasoning.entities",
			"reasoning.confidence_scores",
		},
	}

	if err != nil {
		run.Error = err.Error()
	}

	if ctx.Audit == nil {
		ctx.Audit = &models.AuditContext{}
	}

	ctx.Audit.AgentRuns = append(ctx.Audit.AgentRuns, run)

	// Also update performance metrics
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
		LLMCalls:   0, // No LLM calls for rule-based detection
		Status:     status,
	}
}

// deduplicateStrings removes duplicate strings from a slice.
func (a *IntentDetectionAgent) deduplicateStrings(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range input {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// GetMetadata returns agent metadata (implements MetadataProvider).
func (a *IntentDetectionAgent) GetMetadata() services.AgentMetadata {
	return services.AgentMetadata{
		ID:          a.id,
		Name:        "Intent Detection Agent",
		Description: "Detects user intent and extracts entities from input messages using rule-based classification",
		Version:     "1.0.0",
		Author:      "ADK LLM Proxy",
		Tags:        []string{"intent", "classification", "entity-extraction", "nlp"},
		Dependencies: []string{}, // First agent - no dependencies
	}
}

// GetCapabilities returns agent capabilities (implements CapabilitiesProvider).
func (a *IntentDetectionAgent) GetCapabilities() services.AgentCapabilities {
	return services.AgentCapabilities{
		SupportsParallelExecution: false, // First agent - must run first
		SupportsRetry:             true,
		RequiresLLM:               false, // Rule-based, LLM optional for low-confidence fallback
		IsDeterministic:           true,  // Same input produces same output
		EstimatedDuration:         50,    // ~50ms for rule-based classification
	}
}
