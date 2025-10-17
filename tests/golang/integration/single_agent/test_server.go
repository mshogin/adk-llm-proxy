package single_agent

import (
	"net/http/httptest"

	"github.com/mshogin/agents/internal/application/services"
	"github.com/mshogin/agents/internal/infrastructure/config"
	"github.com/mshogin/agents/internal/infrastructure/providers"
	"github.com/mshogin/agents/internal/presentation/api"
	"github.com/mshogin/agents/pkg/workflows"
)

// TestServer wraps httptest.Server with pre-configured single-agent workflow
type TestServer struct {
	httpServer *httptest.Server
	config     *config.Config
	workflow   string
}

// NewTestServer creates a test server with specified single-agent workflow
func NewTestServer(workflowName string) *TestServer {
	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8000,
		},
		Providers: map[string]config.ProviderConfig{
			"openai": {
				APIKey:  "test-key",
				BaseURL: "https://api.openai.com/v1",
				Enabled: true,
			},
		},
		Workflows: config.WorkflowsConfig{
			Default: workflowName,
			Enabled: []string{workflowName},
		},
	}

	// Create LLM orchestrator
	llmOrchestrator := services.NewLLMOrchestrator(cfg)

	// Create workflow registry
	workflowRegistry := make(map[string]workflows.Workflow)

	// Register single-agent workflows
	workflowRegistry["intent_detection_only"] = workflows.NewIntentDetectionOnlyWorkflow(llmOrchestrator)
	workflowRegistry["reasoning_structure_only"] = workflows.NewReasoningStructureOnlyWorkflow(llmOrchestrator)
	workflowRegistry["retrieval_planner_only"] = workflows.NewRetrievalPlannerOnlyWorkflow(llmOrchestrator)
	workflowRegistry["retrieval_executor_only"] = workflows.NewRetrievalExecutorOnlyWorkflow(llmOrchestrator)
	workflowRegistry["context_synthesizer_only"] = workflows.NewContextSynthesizerOnlyWorkflow(llmOrchestrator)
	workflowRegistry["inference_only"] = workflows.NewInferenceOnlyWorkflow(llmOrchestrator)
	workflowRegistry["summarization_only"] = workflows.NewSummarizationOnlyWorkflow(llmOrchestrator)
	workflowRegistry["validation_only"] = workflows.NewValidationOnlyWorkflow(llmOrchestrator)

	// Create provider registry
	providerRegistry := providers.NewProviderRegistry()
	providerRegistry.RegisterProvider("openai", providers.NewOpenAIProvider(cfg.Providers["openai"]))

	// Create orchestrator
	orchestrator := services.NewOrchestrator(
		workflowRegistry,
		providerRegistry,
		cfg,
	)

	// Create HTTP handler
	handler := api.NewHandler(orchestrator, cfg)

	// Create test server
	server := httptest.NewServer(handler.Router())

	return &TestServer{
		httpServer: server,
		config:     cfg,
		workflow:   workflowName,
	}
}

// URL returns the test server URL
func (ts *TestServer) URL() string {
	return ts.httpServer.URL
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	ts.httpServer.Close()
}

// Workflow returns the configured workflow name
func (ts *TestServer) Workflow() string {
	return ts.workflow
}

// Config returns the server configuration
func (ts *TestServer) Config() *config.Config {
	return ts.config
}
