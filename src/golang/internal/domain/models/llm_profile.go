package models

// ModelProfile defines characteristics of an LLM model for selection.
type ModelProfile struct {
	// Provider name (e.g., "openai", "anthropic", "deepseek", "ollama")
	Provider string `json:"provider"`

	// Model name (e.g., "gpt-4o-mini", "claude-3-5-sonnet")
	Model string `json:"model"`

	// Quality tier: "basic", "standard", "premium", "advanced"
	Quality string `json:"quality"`

	// Speed tier: "fast", "medium", "slow"
	Speed string `json:"speed"`

	// Cost per 1K tokens (USD)
	CostPer1KTokens float64 `json:"cost_per_1k_tokens"`

	// Maximum context window size (tokens)
	ContextLimit int `json:"context_limit"`

	// Capabilities
	SupportsStreaming  bool `json:"supports_streaming"`
	SupportsFunctions  bool `json:"supports_functions"`
	SupportsVision     bool `json:"supports_vision"`
	SupportsChainOfThought bool `json:"supports_chain_of_thought"`

	// Performance characteristics
	AverageLatencyMS int `json:"average_latency_ms"`

	// Availability
	IsLocal     bool `json:"is_local"`      // True for Ollama models
	RequiresAuth bool `json:"requires_auth"` // True if API key required
}

// TaskComplexity defines the complexity level of a reasoning task.
type TaskComplexity string

const (
	// TaskComplexitySimple - Simple classification or extraction (<1K tokens)
	TaskComplexitySimple TaskComplexity = "simple"

	// TaskComplexityMedium - Medium synthesis or planning (1K-8K tokens)
	TaskComplexityMedium TaskComplexity = "medium"

	// TaskComplexityComplex - Complex inference or analysis (8K-32K tokens)
	TaskComplexityComplex TaskComplexity = "complex"

	// TaskComplexityAdvanced - Advanced reasoning requiring chain-of-thought (32K+ tokens)
	TaskComplexityAdvanced TaskComplexity = "advanced"
)

// TaskType defines specific task categories.
type TaskType string

const (
	TaskTypeIntentClassification  TaskType = "intent_classification"
	TaskTypeEntityExtraction       TaskType = "entity_extraction"
	TaskTypeValidation             TaskType = "validation"
	TaskTypeKeywordSearch          TaskType = "keyword_search"
	TaskTypeTextSynthesis          TaskType = "text_synthesis"
	TaskTypeQueryNormalization     TaskType = "query_normalization"
	TaskTypeFactDeduplication      TaskType = "fact_deduplication"
	TaskTypeInference              TaskType = "inference"
	TaskTypeMediumSynthesis        TaskType = "medium_synthesis"
	TaskTypeRetrievalPlanning      TaskType = "retrieval_planning"
	TaskTypeMultiSourceCorrelation TaskType = "multi_source_correlation"
	TaskTypeAdvancedInference      TaskType = "advanced_inference"
	TaskTypeLongContextAnalysis    TaskType = "long_context_analysis"
	TaskTypeDeepReasoning          TaskType = "deep_reasoning"
	TaskTypeCriticalReasoning      TaskType = "critical_reasoning"
)

// ModelSelectionStrategy defines fallback chains for different task types.
type ModelSelectionStrategy struct {
	TaskType       TaskType `json:"task_type"`
	Complexity     TaskComplexity `json:"complexity"`
	DefaultModel   string `json:"default_model"`   // Format: "provider/model"
	Fallback1      string `json:"fallback_1"`
	Fallback2      string `json:"fallback_2"`
	MaxContextSize int    `json:"max_context_size"`
}

// BudgetConstraints defines budget limits for LLM usage.
type BudgetConstraints struct {
	// Per-session budget limit (USD)
	SessionBudgetUSD float64 `json:"session_budget_usd"`

	// Per-agent budget limit (USD)
	AgentBudgetUSD float64 `json:"agent_budget_usd"`

	// Budget warning threshold (0.0-1.0, e.g., 0.8 for 80%)
	WarningThreshold float64 `json:"warning_threshold"`

	// Emergency degradation enabled
	EmergencyDegradationEnabled bool `json:"emergency_degradation_enabled"`

	// Critical agents that always get LLM access (even when budget exceeded)
	CriticalAgents []string `json:"critical_agents"`
}

// CacheConfig defines caching configuration.
type CacheConfig struct {
	// Enable caching
	Enabled bool `json:"enabled"`

	// TTL by task type (seconds)
	ClassificationTTL int `json:"classification_ttl"` // 24h default
	SynthesisTTL      int `json:"synthesis_ttl"`      // 1h default
	InferenceTTL      int `json:"inference_ttl"`      // 30min default

	// Maximum cache size (MB)
	MaxSizeMB int `json:"max_size_mb"`

	// Cache hit rate target (0.0-1.0)
	TargetHitRate float64 `json:"target_hit_rate"`
}

// DefaultModelProfiles returns default model profiles for selection.
func DefaultModelProfiles() []ModelProfile {
	return []ModelProfile{
		// Ollama local models (free, fast, low quality)
		{
			Provider:        "ollama",
			Model:           "mistral",
			Quality:         "basic",
			Speed:           "fast",
			CostPer1KTokens: 0.0,
			ContextLimit:    8192,
			SupportsStreaming: true,
			AverageLatencyMS: 100,
			IsLocal:          true,
			RequiresAuth:     false,
		},
		{
			Provider:        "ollama",
			Model:           "llama3",
			Quality:         "basic",
			Speed:           "fast",
			CostPer1KTokens: 0.0,
			ContextLimit:    8192,
			SupportsStreaming: true,
			AverageLatencyMS: 120,
			IsLocal:          true,
			RequiresAuth:     false,
		},

		// DeepSeek models (very cheap, good quality)
		{
			Provider:        "deepseek",
			Model:           "deepseek-chat",
			Quality:         "standard",
			Speed:           "fast",
			CostPer1KTokens: 0.0001,
			ContextLimit:    32768,
			SupportsStreaming: true,
			AverageLatencyMS: 500,
			IsLocal:          false,
			RequiresAuth:     true,
		},
		{
			Provider:               "deepseek",
			Model:                  "deepseek-r1",
			Quality:                "premium",
			Speed:                  "medium",
			CostPer1KTokens:        0.002,
			ContextLimit:           64000,
			SupportsStreaming:      true,
			SupportsChainOfThought: true,
			AverageLatencyMS:       1500,
			IsLocal:                false,
			RequiresAuth:           true,
		},

		// OpenAI models
		{
			Provider:         "openai",
			Model:            "gpt-4o-mini",
			Quality:          "standard",
			Speed:            "fast",
			CostPer1KTokens:  0.00015,
			ContextLimit:     128000,
			SupportsStreaming: true,
			SupportsFunctions: true,
			SupportsVision:    true,
			AverageLatencyMS:  400,
			IsLocal:           false,
			RequiresAuth:      true,
		},
		{
			Provider:         "openai",
			Model:            "gpt-4o",
			Quality:          "premium",
			Speed:            "medium",
			CostPer1KTokens:  0.0025,
			ContextLimit:     128000,
			SupportsStreaming: true,
			SupportsFunctions: true,
			SupportsVision:    true,
			AverageLatencyMS:  800,
			IsLocal:           false,
			RequiresAuth:      true,
		},
		{
			Provider:               "openai",
			Model:                  "o1-mini",
			Quality:                "advanced",
			Speed:                  "slow",
			CostPer1KTokens:        0.015,
			ContextLimit:           128000,
			SupportsChainOfThought: true,
			AverageLatencyMS:       3000,
			IsLocal:                false,
			RequiresAuth:           true,
		},
		{
			Provider:               "openai",
			Model:                  "o1",
			Quality:                "advanced",
			Speed:                  "slow",
			CostPer1KTokens:        0.06,
			ContextLimit:           200000,
			SupportsChainOfThought: true,
			AverageLatencyMS:       5000,
			IsLocal:                false,
			RequiresAuth:           true,
		},

		// Anthropic models
		{
			Provider:         "anthropic",
			Model:            "claude-3-haiku",
			Quality:          "standard",
			Speed:            "fast",
			CostPer1KTokens:  0.00025,
			ContextLimit:     200000,
			SupportsStreaming: true,
			AverageLatencyMS:  500,
			IsLocal:           false,
			RequiresAuth:      true,
		},
		{
			Provider:         "anthropic",
			Model:            "claude-3-5-sonnet",
			Quality:          "premium",
			Speed:            "medium",
			CostPer1KTokens:  0.003,
			ContextLimit:     200000,
			SupportsStreaming: true,
			AverageLatencyMS:  1000,
			IsLocal:           false,
			RequiresAuth:      true,
		},
		{
			Provider:         "anthropic",
			Model:            "claude-3-opus",
			Quality:          "advanced",
			Speed:            "slow",
			CostPer1KTokens:  0.015,
			ContextLimit:     200000,
			SupportsStreaming: true,
			AverageLatencyMS:  2000,
			IsLocal:           false,
			RequiresAuth:      true,
		},
	}
}

// DefaultSelectionStrategies returns default model selection strategies by task type.
func DefaultSelectionStrategies() []ModelSelectionStrategy {
	return []ModelSelectionStrategy{
		{
			TaskType:       TaskTypeIntentClassification,
			Complexity:     TaskComplexitySimple,
			DefaultModel:   "deepseek/deepseek-chat",
			Fallback1:      "openai/gpt-4o-mini",
			Fallback2:      "ollama/mistral",
			MaxContextSize: 500,
		},
		{
			TaskType:       TaskTypeEntityExtraction,
			Complexity:     TaskComplexitySimple,
			DefaultModel:   "deepseek/deepseek-chat",
			Fallback1:      "openai/gpt-4o-mini",
			Fallback2:      "ollama/llama3",
			MaxContextSize: 1000,
		},
		{
			TaskType:       TaskTypeValidation,
			Complexity:     TaskComplexitySimple,
			DefaultModel:   "deepseek/deepseek-chat",
			Fallback1:      "openai/gpt-4o-mini",
			Fallback2:      "",
			MaxContextSize: 1000,
		},
		{
			TaskType:       TaskTypeKeywordSearch,
			Complexity:     TaskComplexityMedium,
			DefaultModel:   "openai/gpt-4o-mini",
			Fallback1:      "deepseek/deepseek-chat",
			Fallback2:      "ollama/mistral",
			MaxContextSize: 2000,
		},
		{
			TaskType:       TaskTypeTextSynthesis,
			Complexity:     TaskComplexityMedium,
			DefaultModel:   "openai/gpt-4o-mini",
			Fallback1:      "deepseek/deepseek-chat",
			Fallback2:      "ollama/llama3",
			MaxContextSize: 2000,
		},
		{
			TaskType:       TaskTypeQueryNormalization,
			Complexity:     TaskComplexityMedium,
			DefaultModel:   "openai/gpt-4o-mini",
			Fallback1:      "deepseek/deepseek-chat",
			Fallback2:      "",
			MaxContextSize: 2000,
		},
		{
			TaskType:       TaskTypeFactDeduplication,
			Complexity:     TaskComplexityMedium,
			DefaultModel:   "openai/gpt-4o-mini",
			Fallback1:      "deepseek/deepseek-chat",
			Fallback2:      "ollama/mistral",
			MaxContextSize: 3000,
		},
		{
			TaskType:       TaskTypeInference,
			Complexity:     TaskComplexityMedium,
			DefaultModel:   "openai/gpt-4o-mini",
			Fallback1:      "deepseek/deepseek-chat",
			Fallback2:      "",
			MaxContextSize: 3000,
		},
		{
			TaskType:       TaskTypeMediumSynthesis,
			Complexity:     TaskComplexityComplex,
			DefaultModel:   "openai/gpt-4o",
			Fallback1:      "anthropic/claude-3-haiku",
			Fallback2:      "openai/gpt-4o-mini",
			MaxContextSize: 8000,
		},
		{
			TaskType:       TaskTypeRetrievalPlanning,
			Complexity:     TaskComplexityComplex,
			DefaultModel:   "openai/gpt-4o",
			Fallback1:      "anthropic/claude-3-haiku",
			Fallback2:      "deepseek/deepseek-r1",
			MaxContextSize: 8000,
		},
		{
			TaskType:       TaskTypeMultiSourceCorrelation,
			Complexity:     TaskComplexityComplex,
			DefaultModel:   "openai/gpt-4o",
			Fallback1:      "anthropic/claude-3-5-sonnet",
			Fallback2:      "deepseek/deepseek-r1",
			MaxContextSize: 16000,
		},
		{
			TaskType:       TaskTypeAdvancedInference,
			Complexity:     TaskComplexityComplex,
			DefaultModel:   "openai/gpt-4o",
			Fallback1:      "anthropic/claude-3-5-sonnet",
			Fallback2:      "deepseek/deepseek-r1",
			MaxContextSize: 16000,
		},
		{
			TaskType:       TaskTypeLongContextAnalysis,
			Complexity:     TaskComplexityAdvanced,
			DefaultModel:   "anthropic/claude-3-5-sonnet",
			Fallback1:      "openai/gpt-4o",
			Fallback2:      "",
			MaxContextSize: 32000,
		},
		{
			TaskType:       TaskTypeDeepReasoning,
			Complexity:     TaskComplexityAdvanced,
			DefaultModel:   "openai/o1-mini",
			Fallback1:      "anthropic/claude-3-5-sonnet",
			Fallback2:      "openai/gpt-4o",
			MaxContextSize: 64000,
		},
		{
			TaskType:       TaskTypeCriticalReasoning,
			Complexity:     TaskComplexityAdvanced,
			DefaultModel:   "openai/o1",
			Fallback1:      "anthropic/claude-3-opus",
			Fallback2:      "openai/o1-mini",
			MaxContextSize: 100000,
		},
	}
}

// DefaultBudgetConstraints returns default budget constraints.
func DefaultBudgetConstraints() BudgetConstraints {
	return BudgetConstraints{
		SessionBudgetUSD:            0.10, // $0.10 per session
		AgentBudgetUSD:              0.02, // $0.02 per agent
		WarningThreshold:            0.80, // 80% of budget
		EmergencyDegradationEnabled: true,
		CriticalAgents:              []string{"intent_detection", "validation"},
	}
}

// DefaultCacheConfig returns default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:           true,
		ClassificationTTL: 86400, // 24 hours
		SynthesisTTL:      3600,  // 1 hour
		InferenceTTL:      1800,  // 30 minutes
		MaxSizeMB:         100,
		TargetHitRate:     0.40, // 40% target
	}
}
