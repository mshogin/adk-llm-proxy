package models

import "errors"

// Domain-level errors for validation and business logic.
// These errors are defined in the domain layer and can be used
// throughout the application.

var (
	// Request validation errors
	ErrMissingModel   = errors.New("model field is required")
	ErrEmptyMessages  = errors.New("messages field cannot be empty")
	ErrInvalidMessage = errors.New("invalid message format")

	// Provider errors
	ErrProviderNotFound   = errors.New("provider not found")
	ErrProviderDisabled   = errors.New("provider is disabled")
	ErrProviderUnhealthy  = errors.New("provider health check failed")
	ErrInvalidCredentials = errors.New("invalid provider credentials")

	// Workflow errors
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrWorkflowFailed   = errors.New("workflow execution failed")
	ErrWorkflowTimeout  = errors.New("workflow execution timed out")

	// Reasoning errors
	ErrReasoningFailed = errors.New("reasoning process failed")
	ErrAgentFailed     = errors.New("agent execution failed")

	// Stream errors
	ErrStreamClosed = errors.New("stream has been closed")
	ErrStreamFailed = errors.New("streaming failed")
)
