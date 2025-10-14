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

	// Execute Python script
	// TODO: Pass input via stdin or command-line args for full ADK integration
	cmd := exec.CommandContext(ctx, a.pythonPath, a.agentPath)
	cmd.Stdin = nil

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

	// Parse JSON output from Python script
	var pythonResult map[string]interface{}
	if err := json.Unmarshal(output, &pythonResult); err != nil {
		// If JSON parsing fails, return raw output
		return &models.AgentResult{
			AgentName: "adk_agent",
			Output:    string(output),
			Success:   true,
			Duration:  duration,
		}, nil
	}

	// Extract message from Python output
	message := "ADK agent analysis completed"
	if msg, ok := pythonResult["message"].(string); ok {
		message = msg
	}
	if reasoning, ok := pythonResult["reasoning"].(string); ok {
		message += ": " + reasoning
	}

	return &models.AgentResult{
		AgentName: "adk_agent",
		Output:    message,
		Success:   true,
		Duration:  duration,
		Metadata:  pythonResult,
	}, nil
}
