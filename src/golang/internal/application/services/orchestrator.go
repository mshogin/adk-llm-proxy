package services

import (
	"context"
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	domainServices "github.com/mshogin/agents/internal/domain/services"
)

// Orchestrator coordinates the entire request processing pipeline.
// It executes reasoning workflows and streams LLM completions asynchronously.
//
// Design principles:
// - Single Responsibility: Coordinates workflow + LLM streaming
// - Dependency Injection: Depends on domain interfaces
// - Async streaming: Reasoning and inference happen in parallel
type Orchestrator struct {
	providers        map[string]domainServices.LLMProvider
	workflows        map[string]domainServices.Workflow
	defaultWorkflow  string
	providerSelector *ProviderSelector
}

// NewOrchestrator creates a new Orchestrator instance.
func NewOrchestrator(
	providers map[string]domainServices.LLMProvider,
	workflows map[string]domainServices.Workflow,
	defaultWorkflow string,
) *Orchestrator {
	return &Orchestrator{
		providers:        providers,
		workflows:        workflows,
		defaultWorkflow:  defaultWorkflow,
		providerSelector: NewProviderSelector(providers),
	}
}

// ProcessRequest orchestrates the full request pipeline:
// 1. Execute reasoning workflow (async)
// 2. Stream LLM completion (async)
// 3. Return event stream to caller
//
// Returns a channel that emits StreamEvents as they occur.
func (o *Orchestrator) ProcessRequest(
	ctx context.Context,
	req *models.CompletionRequest,
	workflowName string,
) (<-chan *StreamEvent, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Select workflow
	if workflowName == "" {
		workflowName = o.defaultWorkflow
	}
	workflow, ok := o.workflows[workflowName]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", workflowName)
	}

	// Select provider
	provider, err := o.providerSelector.SelectProvider(req.Model)
	if err != nil {
		return nil, err
	}

	// Create event channel
	eventChan := make(chan *StreamEvent, 10)

	// Start async processing
	go o.processAsync(ctx, req, workflow, provider, eventChan)

	return eventChan, nil
}

// processAsync runs the full pipeline asynchronously.
func (o *Orchestrator) processAsync(
	ctx context.Context,
	req *models.CompletionRequest,
	workflow domainServices.Workflow,
	provider domainServices.LLMProvider,
	eventChan chan<- *StreamEvent,
) {
	defer close(eventChan)

	// Phase 1: Execute reasoning workflow
	reasoningInput := models.NewReasoningInput(req, workflow.Name())
	reasoningStart := time.Now()

	reasoningResult, err := workflow.Execute(ctx, reasoningInput)
	if err != nil {
		o.sendEvent(ctx, eventChan, NewErrorEvent(fmt.Sprintf("reasoning failed: %v", err)))
		return
	}

	reasoningResult.Duration = time.Since(reasoningStart).Milliseconds()

	// Send reasoning event
	o.sendEvent(ctx, eventChan, NewReasoningEvent(reasoningResult))

	// Phase 2: Stream LLM completion with enriched messages
	// Build enriched request if reasoning workflow provided enriched messages
	enrichedReq := req
	if len(reasoningResult.EnrichedMessages) > 0 {
		// Create new request with enriched messages prepended to original messages
		enrichedReq = &models.CompletionRequest{
			Model:       req.Model,
			Messages:    append(reasoningResult.EnrichedMessages, req.Messages...),
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			Stream:      req.Stream,
		}
	}

	chunkChan, err := provider.StreamCompletion(ctx, enrichedReq)
	if err != nil {
		o.sendEvent(ctx, eventChan, NewErrorEvent(fmt.Sprintf("streaming failed: %v", err)))
		return
	}

	// Forward completion chunks
	for chunk := range chunkChan {
		o.sendEvent(ctx, eventChan, NewCompletionEvent(chunk))
	}

	// Send done event
	o.sendEvent(ctx, eventChan, NewDoneEvent())
}

// sendEvent sends an event to the channel, checking for context cancellation.
func (o *Orchestrator) sendEvent(ctx context.Context, eventChan chan<- *StreamEvent, event *StreamEvent) {
	select {
	case eventChan <- event:
	case <-ctx.Done():
		return
	}
}

// GetWorkflows returns the list of available workflows.
func (o *Orchestrator) GetWorkflows() []string {
	workflows := make([]string, 0, len(o.workflows))
	for name := range o.workflows {
		workflows = append(workflows, name)
	}
	return workflows
}

// GetProviders returns the list of available providers.
func (o *Orchestrator) GetProviders() []string {
	providers := make([]string, 0, len(o.providers))
	for name := range o.providers {
		providers = append(providers, name)
	}
	return providers
}
