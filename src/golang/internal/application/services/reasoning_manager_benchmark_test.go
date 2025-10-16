package services

import (
	"context"
	"testing"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
	"github.com/mshogin/agents/internal/domain/services/agents"
	domainservices "github.com/mshogin/agents/internal/domain/services"
)

// Performance benchmark tests for latency, LLM call frequency, and cost per session

// BenchmarkPipeline_Sequential_SingleAgent benchmarks a single agent execution
func BenchmarkPipeline_Sequential_SingleAgent(b *testing.B) {
	agent := &MockAgent{
		id:             "test_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "test_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentContext := models.NewAgentContext("bench-session", "bench-trace")
		_, err := manager.Execute(ctx, agentContext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPipeline_Sequential_ThreeAgents benchmarks a 3-agent pipeline
func BenchmarkPipeline_Sequential_ThreeAgents(b *testing.B) {
	agent1 := &MockAgent{
		id:             "agent1",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "agent2",
		preconditions:  []string{"reasoning.intents"},
		postconditions: []string{"reasoning.hypotheses"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Hypotheses = []models.Hypothesis{{ID: "h1", Description: "test"}}
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id:             "agent3",
		preconditions:  []string{"reasoning.hypotheses"},
		postconditions: []string{"reasoning.conclusions"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Description: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "agent1", Enabled: true, Timeout: 5000},
			{ID: "agent2", Enabled: true, Timeout: 5000},
			{ID: "agent3", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: true,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)
	manager.RegisterAgent(agent3)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentContext := models.NewAgentContext("bench-session", "bench-trace")
		_, err := manager.Execute(ctx, agentContext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPipeline_Sequential_SevenAgents benchmarks full 7-agent pipeline
func BenchmarkPipeline_Sequential_SevenAgents(b *testing.B) {
	agents := []struct {
		id             string
		preconditions  []string
		postconditions []string
	}{
		{"agent1", []string{}, []string{"out1"}},
		{"agent2", []string{"out1"}, []string{"out2"}},
		{"agent3", []string{"out2"}, []string{"out3"}},
		{"agent4", []string{"out3"}, []string{"out4"}},
		{"agent5", []string{"out4"}, []string{"out5"}},
		{"agent6", []string{"out5"}, []string{"out6"}},
		{"agent7", []string{"out6"}, []string{"out7"}},
	}

	config := PipelineConfig{
		Mode:   SequentialMode,
		Agents: []AgentConfig{},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)

	for _, a := range agents {
		agent := &MockAgent{
			id:             a.id,
			preconditions:  a.preconditions,
			postconditions: a.postconditions,
			executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
				time.Sleep(100 * time.Microsecond) // Simulate some work
				return agentContext, nil
			},
		}
		manager.RegisterAgent(agent)
		config.Agents = append(config.Agents, AgentConfig{ID: a.id, Enabled: true, Timeout: 5000})
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentContext := models.NewAgentContext("bench-session", "bench-trace")
		_, err := manager.Execute(ctx, agentContext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPipeline_Parallel_MultiPath benchmarks parallel execution
func BenchmarkPipeline_Parallel_MultiPath(b *testing.B) {
	b.Skip("TODO: ReasoningManager has race condition in parallel mode (concurrent map writes in audit trail)")

	agent1 := &MockAgent{
		id:             "root",
		preconditions:  []string{},
		postconditions: []string{"root_out"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(50 * time.Microsecond)
			return agentContext, nil
		},
	}

	agent2 := &MockAgent{
		id:             "branch1",
		preconditions:  []string{"root_out"},
		postconditions: []string{"branch1_out"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(100 * time.Microsecond)
			return agentContext, nil
		},
	}

	agent3 := &MockAgent{
		id:             "branch2",
		preconditions:  []string{"root_out"},
		postconditions: []string{"branch2_out"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(100 * time.Microsecond)
			return agentContext, nil
		},
	}

	agent4 := &MockAgent{
		id:             "merge",
		preconditions:  []string{"branch1_out", "branch2_out"},
		postconditions: []string{"merge_out"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			time.Sleep(50 * time.Microsecond)
			return agentContext, nil
		},
	}

	config := PipelineConfig{
		Mode: ParallelMode,
		Agents: []AgentConfig{
			{ID: "root", Enabled: true, DependsOn: []string{}, Timeout: 5000},
			{ID: "branch1", Enabled: true, DependsOn: []string{"root"}, Timeout: 5000},
			{ID: "branch2", Enabled: true, DependsOn: []string{"root"}, Timeout: 5000},
			{ID: "merge", Enabled: true, DependsOn: []string{"branch1", "branch2"}, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent1)
	manager.RegisterAgent(agent2)
	manager.RegisterAgent(agent3)
	manager.RegisterAgent(agent4)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentContext := models.NewAgentContext("bench-session", "bench-trace")
		_, err := manager.Execute(ctx, agentContext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRealAgents_IntentDetection benchmarks real intent detection agent
func BenchmarkRealAgents_IntentDetection(b *testing.B) {
	b.Skip("TODO: Intent detection requires input text parsing setup")

	agent := agents.NewIntentDetectionAgent()

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "intent_detection", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(agent)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentContext := models.NewAgentContext("bench-session", "bench-trace")
		// Pre-populate with intents since intent detection needs input text
		agentContext.Reasoning.Intents = []models.Intent{{Type: "query_commits", Confidence: 0.9}}
		_, err := manager.Execute(ctx, agentContext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRealAgents_FullPipeline benchmarks complete real agent pipeline
func BenchmarkRealAgents_FullPipeline(b *testing.B) {
	b.Skip("TODO: Real agents require proper context setup")

	intentAgent := agents.NewIntentDetectionAgent()
	structureAgent := agents.NewReasoningStructureAgent()
	retrievalAgent := agents.NewRetrievalPlannerAgent()
	synthesizerAgent := agents.NewContextSynthesizerAgent()
	inferenceAgent := agents.NewInferenceAgent()
	validationAgent := agents.NewValidationAgent()
	summaryAgent := agents.NewSummarizationAgent()

	config := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "intent_detection", Enabled: true, Timeout: 5000},
			{ID: "reasoning_structure", Enabled: true, Timeout: 5000},
			{ID: "retrieval_planner", Enabled: true, Timeout: 5000},
			{ID: "context_synthesizer", Enabled: true, Timeout: 5000},
			{ID: "inference", Enabled: true, Timeout: 5000},
			{ID: "validation", Enabled: true, Timeout: 5000},
			{ID: "summarization", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	manager := NewReasoningManager(config)
	manager.RegisterAgent(intentAgent)
	manager.RegisterAgent(structureAgent)
	manager.RegisterAgent(retrievalAgent)
	manager.RegisterAgent(synthesizerAgent)
	manager.RegisterAgent(inferenceAgent)
	manager.RegisterAgent(validationAgent)
	manager.RegisterAgent(summaryAgent)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agentContext := models.NewAgentContext("bench-session", "bench-trace")
		// Pre-populate to avoid input text requirement
		agentContext.Reasoning.Intents = []models.Intent{{Type: "query_commits", Confidence: 0.9}}
		_, err := manager.Execute(ctx, agentContext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextClone benchmarks AgentContext cloning
func BenchmarkContextClone(b *testing.B) {
	// Create a complex context
	ctx := models.NewAgentContext("bench-session", "bench-trace")
	ctx.Reasoning.Intents = []models.Intent{
		{Type: "query", Confidence: 0.9},
		{Type: "command", Confidence: 0.8},
	}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{
		{ID: "h1", Description: "test1", Dependencies: []string{}},
		{ID: "h2", Description: "test2", Dependencies: []string{"h1"}},
	}
	ctx.Reasoning.Conclusions = []models.Conclusion{
		{ID: "c1", Description: "test", Confidence: 0.9},
	}
	ctx.Enrichment.Facts = []models.Fact{
		{ID: "f1", Content: "fact1", Source: "test", Confidence: 0.95},
		{ID: "f2", Content: "fact2", Source: "test", Confidence: 0.90},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ctx.Clone()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContextSerialization benchmarks AgentContext serialization
func BenchmarkContextSerialization(b *testing.B) {
	ctx := models.NewAgentContext("bench-session", "bench-trace")
	ctx.Reasoning.Intents = []models.Intent{{Type: "query", Confidence: 0.9}}
	ctx.Reasoning.Hypotheses = []models.Hypothesis{{ID: "h1", Description: "test"}}
	ctx.Reasoning.Conclusions = []models.Conclusion{{ID: "c1", Description: "test", Confidence: 0.9}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ctx.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMetricsCollection benchmarks metrics collection overhead
func BenchmarkMetricsCollection(b *testing.B) {
	agent := &MockAgent{
		id:             "test_agent",
		preconditions:  []string{},
		postconditions: []string{"reasoning.intents"},
		executeFn: func(ctx context.Context, agentContext *models.AgentContext) (*models.AgentContext, error) {
			agentContext.Reasoning.Intents = []models.Intent{{Type: "test", Confidence: 0.9}}
			return agentContext, nil
		},
	}

	configWithMetrics := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "test_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: true,
		},
	}

	managerWithMetrics := NewReasoningManager(configWithMetrics)
	managerWithMetrics.RegisterAgent(agent)

	configWithoutMetrics := PipelineConfig{
		Mode: SequentialMode,
		Agents: []AgentConfig{
			{ID: "test_agent", Enabled: true, Timeout: 5000},
		},
		Options: domainservices.AgentExecutionOptions{
			ValidateContract: false,
			TrackPerformance: false,
		},
	}

	managerWithoutMetrics := NewReasoningManager(configWithoutMetrics)
	managerWithoutMetrics.RegisterAgent(agent)

	ctx := context.Background()

	b.Run("WithMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			agentContext := models.NewAgentContext("bench-session", "bench-trace")
			managerWithMetrics.Execute(ctx, agentContext)
		}
	})

	b.Run("WithoutMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			agentContext := models.NewAgentContext("bench-session", "bench-trace")
			managerWithoutMetrics.Execute(ctx, agentContext)
		}
	})
}
