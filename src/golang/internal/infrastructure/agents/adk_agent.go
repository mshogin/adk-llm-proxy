package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
)

// ADKAgent executes the Python ADK agent as a subprocess.
// This allows us to leverage Google's ADK from the Go proxy.
type ADKAgent struct {
	pythonPath string
	agentPath  string
	timeout    time.Duration
}

// NewADKAgent creates a new ADK agent instance.
func NewADKAgent(pythonPath, agentPath string, timeout time.Duration) *ADKAgent {
	if pythonPath == "" {
		pythonPath = "python3"
	}
	return &ADKAgent{
		pythonPath: pythonPath,
		agentPath:  agentPath,
		timeout:    timeout,
	}
}

// Execute runs the ADK agent with the given input and returns the result.
func (a *ADKAgent) Execute(ctx context.Context, input *models.ReasoningInput) (*models.AgentResult, error) {
	start := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// Prepare input JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return &models.AgentResult{
			AgentName: "adk_agent",
			Success:   false,
			Error:     fmt.Sprintf("failed to marshal input: %v", err),
			Duration:  time.Since(start).Milliseconds(),
		}, nil
	}

	// Execute Python script
	cmd := exec.CommandContext(ctx, a.pythonPath, a.agentPath)
	cmd.Stdin = nil // We'll pass input via args or stdin in a real implementation

	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return &models.AgentResult{
			AgentName: "adk_agent",
			Success:   false,
			Error:     fmt.Sprintf("agent execution failed: %v, output: %s", err, string(output)),
			Duration:  duration,
		}, nil
	}

	// For now, return a simple result
	// In a real implementation, you would parse the JSON output from the Python script
	return &models.AgentResult{
		AgentName: "adk_agent",
		Output:    fmt.Sprintf("ADK Agent processed: %s (placeholder)", input.GetUserMessage()),
		Success:   true,
		Duration:  duration,
		Metadata: map[string]interface{}{
			"input_json": string(inputJSON),
		},
	}, nil
}
