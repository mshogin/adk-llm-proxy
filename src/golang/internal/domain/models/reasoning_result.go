package models

// ReasoningInput represents the input to a reasoning workflow.
// This is extracted from the CompletionRequest and passed to workflows.
type ReasoningInput struct {
	// Messages is the conversation history
	Messages []Message `json:"messages"`

	// Model is the target LLM model
	Model string `json:"model"`

	// Workflow is the requested workflow name (if specified)
	Workflow string `json:"workflow,omitempty"`

	// Metadata contains additional context (user ID, session, etc.)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ReasoningResult represents the output of a reasoning workflow.
// This contains insights, transformed messages, and workflow-specific data.
type ReasoningResult struct {
	// Message is the primary reasoning output (insights, summary, etc.)
	Message string `json:"message"`

	// EnrichedMessages are the original messages with reasoning enhancements
	// (e.g., additional context, transformed prompts)
	EnrichedMessages []Message `json:"enriched_messages,omitempty"`

	// Intent is the detected user intent (for basic workflow)
	Intent string `json:"intent,omitempty"`

	// Confidence is the confidence score for intent detection (0.0 to 1.0)
	Confidence float64 `json:"confidence,omitempty"`

	// AgentResults contains results from individual agents (for advanced workflow)
	AgentResults map[string]*AgentResult `json:"agent_results,omitempty"`

	// Metadata contains workflow-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Duration is the time taken to execute the workflow (milliseconds)
	Duration int64 `json:"duration,omitempty"`

	// WorkflowName is the name of the workflow that generated this result
	WorkflowName string `json:"workflow_name"`
}

// AgentResult represents the result from a single agent in multi-agent workflows.
type AgentResult struct {
	// AgentName is the identifier of the agent (e.g., "adk_agent", "openai_agent")
	AgentName string `json:"agent_name"`

	// Output is the agent's output
	Output string `json:"output"`

	// Success indicates whether the agent executed successfully
	Success bool `json:"success"`

	// Error contains the error message if the agent failed
	Error string `json:"error,omitempty"`

	// Duration is the time taken by the agent (milliseconds)
	Duration int64 `json:"duration,omitempty"`

	// Metadata contains agent-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewReasoningInput creates a ReasoningInput from a CompletionRequest.
func NewReasoningInput(req *CompletionRequest, workflow string) *ReasoningInput {
	return &ReasoningInput{
		Messages: req.Messages,
		Model:    req.Model,
		Workflow: workflow,
		Metadata: make(map[string]interface{}),
	}
}

// NewReasoningResult creates a new ReasoningResult with defaults.
func NewReasoningResult(workflowName, message string) *ReasoningResult {
	return &ReasoningResult{
		WorkflowName:  workflowName,
		Message:       message,
		AgentResults:  make(map[string]*AgentResult),
		Metadata:      make(map[string]interface{}),
		Confidence:    0.0,
		Duration:      0,
	}
}

// AddAgentResult adds an agent result to the reasoning result.
func (r *ReasoningResult) AddAgentResult(agentName string, result *AgentResult) {
	if r.AgentResults == nil {
		r.AgentResults = make(map[string]*AgentResult)
	}
	r.AgentResults[agentName] = result
}

// HasError returns true if any agent in the result failed.
func (r *ReasoningResult) HasError() bool {
	for _, result := range r.AgentResults {
		if !result.Success {
			return true
		}
	}
	return false
}

// GetUserMessage extracts the last user message from the input.
func (i *ReasoningInput) GetUserMessage() string {
	for j := len(i.Messages) - 1; j >= 0; j-- {
		if i.Messages[j].Role == "user" {
			return i.Messages[j].Content
		}
	}
	return ""
}
