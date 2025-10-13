package workflows

// Package workflows provides workflow implementations for reasoning.
// Workflows are public (in pkg/) so they can be extended by external packages.
//
// Available workflows:
// - DefaultWorkflow: Simple pass-through (returns "Hello World")
// - BasicWorkflow: Intent detection via regex/keywords
// - AdvancedWorkflow: Multi-agent orchestration (ADK Python + OpenAI native)
