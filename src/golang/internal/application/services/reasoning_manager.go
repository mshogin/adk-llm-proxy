package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	domainservices "github.com/mshogin/agents/internal/domain/services"
)

// ExecutionMode defines how agents are executed in the pipeline.
type ExecutionMode string

const (
	// SequentialMode executes agents one after another
	SequentialMode ExecutionMode = "sequential"

	// ParallelMode executes independent agents concurrently
	ParallelMode ExecutionMode = "parallel"

	// ConditionalMode executes agents based on context conditions
	ConditionalMode ExecutionMode = "conditional"
)

// PipelineConfig defines the pipeline configuration.
type PipelineConfig struct {
	Mode    ExecutionMode          // Execution mode
	Agents  []AgentConfig          // Agent configurations
	Options domainservices.AgentExecutionOptions // Global execution options
}

// AgentConfig defines configuration for a single agent in the pipeline.
type AgentConfig struct {
	ID         string   // Agent identifier
	Enabled    bool     // Whether agent is enabled
	DependsOn  []string // Agent dependencies
	Timeout    int      // Execution timeout in milliseconds
	Retry      int      // Number of retries on failure
	Conditions []string // Execution conditions (for conditional mode)
}

// DefaultExecutionOptions returns default execution options for the pipeline.
func DefaultExecutionOptions() domainservices.AgentExecutionOptions {
	return domainservices.DefaultExecutionOptions()
}

// ReasoningManager orchestrates the execution of reasoning agents.
type ReasoningManager struct {
	agents    map[string]domainservices.ReasoningAgent
	config    PipelineConfig
	validator *models.ContextValidator
	mu        sync.RWMutex
}

// NewReasoningManager creates a new reasoning manager.
func NewReasoningManager(config PipelineConfig) *ReasoningManager {
	return &ReasoningManager{
		agents:    make(map[string]domainservices.ReasoningAgent),
		config:    config,
		validator: models.NewContextValidator(),
	}
}

// RegisterAgent registers an agent with the manager.
func (m *ReasoningManager) RegisterAgent(agent domainservices.ReasoningAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agentID := agent.AgentID()
	if _, exists := m.agents[agentID]; exists {
		return fmt.Errorf("agent %s already registered", agentID)
	}

	m.agents[agentID] = agent

	// Register agent permissions based on default permissions
	permissions := models.DefaultAgentPermissions()
	if agentPerms, ok := permissions[agentID]; ok {
		m.validator.RegisterAgent(agentID, agentPerms)
	} else {
		// Default to diagnostics and audit only
		m.validator.RegisterAgent(agentID, []string{"diagnostics", "audit"})
	}

	return nil
}

// GetAgent retrieves a registered agent by ID.
func (m *ReasoningManager) GetAgent(agentID string) (domainservices.ReasoningAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// Execute runs the complete reasoning pipeline.
func (m *ReasoningManager) Execute(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	switch m.config.Mode {
	case SequentialMode:
		return m.executeSequential(ctx, agentContext)
	case ParallelMode:
		return m.executeParallel(ctx, agentContext)
	case ConditionalMode:
		return m.executeConditional(ctx, agentContext)
	default:
		return nil, fmt.Errorf("unknown execution mode: %s", m.config.Mode)
	}
}

// executeSequential executes agents sequentially.
func (m *ReasoningManager) executeSequential(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	currentContext := agentContext
	startTime := time.Now()

	for _, agentConfig := range m.config.Agents {
		if !agentConfig.Enabled {
			continue
		}

		// Execute agent
		updatedContext, err := m.executeAgent(ctx, currentContext, agentConfig)
		if err != nil {
			return currentContext, fmt.Errorf("agent %s failed: %w", agentConfig.ID, err)
		}

		currentContext = updatedContext

		// Check context cancellation
		select {
		case <-ctx.Done():
			return currentContext, ctx.Err()
		default:
		}
	}

	// Update total duration
	currentContext.Diagnostics.Performance.TotalDurationMS = time.Since(startTime).Milliseconds()

	return currentContext, nil
}

// executeParallel executes independent agents concurrently.
func (m *ReasoningManager) executeParallel(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	// Group agents by dependency levels
	levels := m.buildDependencyLevels()

	currentContext := agentContext
	startTime := time.Now()

	// Execute each level in parallel
	for _, level := range levels {
		updatedContext, err := m.executeLevel(ctx, currentContext, level)
		if err != nil {
			return currentContext, err
		}
		currentContext = updatedContext
	}

	// Update total duration
	currentContext.Diagnostics.Performance.TotalDurationMS = time.Since(startTime).Milliseconds()

	return currentContext, nil
}

// executeConditional executes agents based on conditions.
func (m *ReasoningManager) executeConditional(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
	currentContext := agentContext
	startTime := time.Now()

	for _, agentConfig := range m.config.Agents {
		if !agentConfig.Enabled {
			continue
		}

		// Check conditions
		if !m.checkConditions(currentContext, agentConfig.Conditions) {
			continue
		}

		// Execute agent
		updatedContext, err := m.executeAgent(ctx, currentContext, agentConfig)
		if err != nil {
			return currentContext, fmt.Errorf("agent %s failed: %w", agentConfig.ID, err)
		}

		currentContext = updatedContext
	}

	// Update total duration
	currentContext.Diagnostics.Performance.TotalDurationMS = time.Since(startTime).Milliseconds()

	return currentContext, nil
}

// executeAgent executes a single agent.
func (m *ReasoningManager) executeAgent(ctx context.Context, agentContext *models.AgentContext, config AgentConfig) (*models.AgentContext, error) {
	agent, err := m.GetAgent(config.ID)
	if err != nil {
		return agentContext, err
	}

	// Create diff tracker if needed
	var tracker *models.DiffTracker
	if m.config.Options.CaptureChanges {
		tracker, _ = models.NewDiffTracker(agentContext)
	}

	// Validate preconditions
	if m.config.Options.ValidateContract {
		validator := domainservices.NewContractValidator(agent)
		result := validator.ValidatePreconditions(agentContext)
		if !result.Valid {
			if m.config.Options.FailOnViolation {
				return agentContext, fmt.Errorf("precondition validation failed for %s: %s", config.ID, result.ValidationError)
			}
			// Log warning but continue
			agentContext.Diagnostics.Warnings = append(agentContext.Diagnostics.Warnings, models.Warning{
				Timestamp: time.Now(),
				AgentID:   config.ID,
				Message:   fmt.Sprintf("Precondition validation failed: %s", result.ValidationError),
			})
		}
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(config.Timeout)*time.Millisecond)
	defer cancel()

	startTime := time.Now()
	updatedContext, execErr := agent.Execute(execCtx, agentContext)
	duration := time.Since(startTime).Milliseconds()

	// Handle execution error
	if execErr != nil {
		// Record failure
		m.recordAgentRun(agentContext, config.ID, "failed", duration, execErr.Error())

		// Retry if configured
		if config.Retry > 0 {
			for i := 0; i < config.Retry; i++ {
				time.Sleep(time.Duration(100) * time.Millisecond) // Small delay between retries
				updatedContext, execErr = agent.Execute(execCtx, agentContext)
				if execErr == nil {
					break
				}
			}
		}

		if execErr != nil {
			return agentContext, execErr
		}
	}

	// Validate postconditions
	if m.config.Options.ValidateContract {
		validator := domainservices.NewContractValidator(agent)
		result := validator.ValidatePostconditions(updatedContext)
		if !result.Valid {
			if m.config.Options.FailOnViolation {
				return agentContext, fmt.Errorf("postcondition validation failed for %s: %s", config.ID, result.ValidationError)
			}
			// Log warning but continue
			updatedContext.Diagnostics.Warnings = append(updatedContext.Diagnostics.Warnings, models.Warning{
				Timestamp: time.Now(),
				AgentID:   config.ID,
				Message:   fmt.Sprintf("Postcondition validation failed: %s", result.ValidationError),
			})
		}
	}

	// Capture changes
	if m.config.Options.CaptureChanges && tracker != nil {
		diff, _ := tracker.Capture(config.ID, updatedContext)
		if diff != nil {
			updatedContext.Audit.Diffs = append(updatedContext.Audit.Diffs, *diff)
		}
	}

	// Record successful run
	m.recordAgentRun(updatedContext, config.ID, "success", duration, "")

	return updatedContext, nil
}

// executeLevel executes all agents in a dependency level concurrently.
func (m *ReasoningManager) executeLevel(ctx context.Context, agentContext *models.AgentContext, agentIDs []string) (*models.AgentContext, error) {
	var wg sync.WaitGroup
	results := make(chan *models.AgentContext, len(agentIDs))
	errors := make(chan error, len(agentIDs))

	for _, agentID := range agentIDs {
		// Find agent config
		var config AgentConfig
		found := false
		for _, ac := range m.config.Agents {
			if ac.ID == agentID {
				config = ac
				found = true
				break
			}
		}
		if !found || !config.Enabled {
			continue
		}

		wg.Add(1)
		go func(id string, cfg AgentConfig) {
			defer wg.Done()

			updatedCtx, err := m.executeAgent(ctx, agentContext, cfg)
			if err != nil {
				errors <- fmt.Errorf("agent %s: %w", id, err)
				return
			}
			results <- updatedCtx
		}(agentID, config)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	if len(errors) > 0 {
		return agentContext, <-errors
	}

	// Merge results (last write wins for now - could be more sophisticated)
	var finalContext *models.AgentContext = agentContext
	for result := range results {
		finalContext = result
	}

	return finalContext, nil
}

// buildDependencyLevels builds dependency levels for parallel execution.
func (m *ReasoningManager) buildDependencyLevels() [][]string {
	// Topological sort to determine execution order
	// Agents with no dependencies go in level 0
	// Agents that depend only on level 0 go in level 1, etc.

	levels := [][]string{}
	processed := make(map[string]bool)

	// Keep adding levels until all agents are processed
	for len(processed) < len(m.config.Agents) {
		currentLevel := []string{}

		for _, agentConfig := range m.config.Agents {
			if processed[agentConfig.ID] || !agentConfig.Enabled {
				continue
			}

			// Check if all dependencies are processed
			allDepsProcessed := true
			for _, dep := range agentConfig.DependsOn {
				if !processed[dep] {
					allDepsProcessed = false
					break
				}
			}

			if allDepsProcessed {
				currentLevel = append(currentLevel, agentConfig.ID)
				processed[agentConfig.ID] = true
			}
		}

		if len(currentLevel) > 0 {
			levels = append(levels, currentLevel)
		} else {
			// Circular dependency detected
			break
		}
	}

	return levels
}

// checkConditions checks if all conditions are satisfied.
func (m *ReasoningManager) checkConditions(ctx *models.AgentContext, conditions []string) bool {
	// If no conditions, always execute
	if len(conditions) == 0 {
		return true
	}

	// Check each condition
	for _, condition := range conditions {
		if !m.evaluateCondition(ctx, condition) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a single condition.
func (m *ReasoningManager) evaluateCondition(ctx *models.AgentContext, condition string) bool {
	// Simple condition evaluation - check if key exists
	// Could be extended to support complex expressions
	validator := domainservices.NewContractValidator(&dummyAgent{preconditions: []string{condition}})
	result := validator.ValidatePreconditions(ctx)
	return result.Valid
}

// recordAgentRun records an agent execution in the audit trail.
func (m *ReasoningManager) recordAgentRun(ctx *models.AgentContext, agentID, status string, duration int64, errMsg string) {
	run := models.AgentRun{
		Timestamp:  time.Now(),
		AgentID:    agentID,
		Status:     status,
		DurationMS: duration,
		Error:      errMsg,
	}

	ctx.Audit.AgentRuns = append(ctx.Audit.AgentRuns, run)

	// Track performance metrics
	if ctx.Diagnostics.Performance.AgentMetrics == nil {
		ctx.Diagnostics.Performance.AgentMetrics = make(map[string]*models.AgentMetrics)
	}

	ctx.Diagnostics.Performance.AgentMetrics[agentID] = &models.AgentMetrics{
		DurationMS: duration,
		Status:     status,
	}
}

// dummyAgent is a helper for condition evaluation.
type dummyAgent struct {
	preconditions []string
}

func (a *dummyAgent) AgentID() string                             { return "dummy" }
func (a *dummyAgent) Preconditions() []string                     { return a.preconditions }
func (a *dummyAgent) Postconditions() []string                    { return []string{} }
func (a *dummyAgent) Execute(context.Context, *models.AgentContext) (*models.AgentContext, error) {
	return nil, nil
}
