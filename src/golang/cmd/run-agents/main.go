package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mshogin/agents/internal/application/services"
	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/infrastructure/config"
	"github.com/mshogin/agents/pkg/workflows"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	// Parse arguments
	agentList := os.Args[1]
	prompt := os.Args[2]

	agentNames := parseAgentList(agentList)
	if len(agentNames) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No valid agents specified\n")
		printUsage()
		os.Exit(1)
	}

	// Load config (same as server)
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create LLM orchestrator (same as server)
	llmOrchestrator := services.NewLLMOrchestrator()

	// Note: For CLI mode, we skip provider registration
	// since agents can work with rule-based logic + LLM fallback

	// Create dynamic workflow with specified agents
	workflow := workflows.NewDynamicWorkflow(llmOrchestrator, agentNames)

	// Create reasoning input
	input := &models.ReasoningInput{
		Messages: []models.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Execute workflow (same pipeline logic as server)
	fmt.Println("=== AGENT PIPELINE EXECUTION ===\n")
	fmt.Printf("Agents: %s\n", strings.Join(agentNames, " → "))
	fmt.Printf("Prompt: \"%s\"\n\n", prompt)

	ctx := context.Background()
	startTime := time.Now()

	result, err := workflow.Execute(ctx, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError executing pipeline: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)

	// Print results
	fmt.Println(result.Message)
	fmt.Printf("\n⏱️  Total Duration: %dms\n", duration.Milliseconds())
}

func parseAgentList(agentList string) []string {
	// Agent name mapping
	agentMap := map[string]string{
		"intent":             "intent_detection",
		"reasoning":          "reasoning_structure",
		"retrieval-planner":  "retrieval_planner",
		"retrieval-executor": "retrieval_executor",
		"context":            "context_synthesizer",
		"inference":          "inference",
		"validation":         "validation",
		"summary":            "summarization",
	}

	parts := strings.Split(agentList, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if agentID, ok := agentMap[part]; ok {
			result = append(result, agentID)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Unknown agent '%s', skipping\n", part)
		}
	}

	return result
}

func printUsage() {
	fmt.Println(`Usage: run-agents <agents> "<prompt>"

Arguments:
  <agents>  Comma-separated list of agent names (no spaces)
  <prompt>  User prompt in quotes

Agent Names:
  intent             - Intent Detection Agent
  reasoning          - Reasoning Structure Agent
  retrieval-planner  - Retrieval Planner Agent
  retrieval-executor - Retrieval Executor Agent
  context            - Context Synthesizer Agent
  inference          - Inference Agent
  validation         - Validation Agent
  summary            - Summarization Agent

Examples:
  # Full pipeline
  run-agents intent,reasoning,retrieval-planner,retrieval-executor,context,inference,validation,summary "Get my commits"

  # Partial pipeline
  run-agents intent,inference "What are the tickets?"

  # Single agent
  run-agents intent "Analyze this request"

  # Quick analysis
  run-agents intent,reasoning,inference "Complex query"
`)
}
