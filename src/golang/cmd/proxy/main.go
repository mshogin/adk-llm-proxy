package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mshogin/agents/internal/application/services"
	domainServices "github.com/mshogin/agents/internal/domain/services"
	"github.com/mshogin/agents/internal/infrastructure/config"
	"github.com/mshogin/agents/internal/infrastructure/providers"
	"github.com/mshogin/agents/internal/presentation/api"
	"github.com/mshogin/agents/pkg/workflows"
)

func main() {
	// Parse CLI flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	host := flag.String("host", "", "Server host (overrides config)")
	port := flag.Int("port", 0, "Server port (overrides config)")
	workflow := flag.String("workflow", "", "Default workflow (overrides config)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Debug: print provider status
	log.Printf("DEBUG: Loaded %d providers", len(cfg.Providers))
	for name, provider := range cfg.Providers {
		log.Printf("DEBUG: Provider %s: enabled=%v, api_key_len=%d, base_url=%s",
			name, provider.Enabled, len(provider.APIKey), provider.BaseURL)
	}

	// Apply CLI overrides
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *workflow != "" {
		cfg.Workflows.Default = *workflow
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize LLM providers
	providerRegistry := make(map[string]domainServices.LLMProvider)
	for name, providerCfg := range cfg.Providers {
		if !providerCfg.Enabled {
			continue
		}

		var provider domainServices.LLMProvider
		switch name {
		case "openai":
			provider = providers.NewOpenAIProvider(providerCfg)
		case "anthropic":
			provider = providers.NewAnthropicProvider(providerCfg)
		case "deepseek":
			provider = providers.NewDeepSeekProvider(providerCfg)
		case "ollama":
			provider = providers.NewOllamaProvider(providerCfg)
		default:
			log.Printf("Warning: unknown provider %s", name)
			continue
		}

		providerRegistry[name] = provider
		log.Printf("Initialized provider: %s", name)
	}

	// Initialize workflows
	workflowRegistry := make(map[string]domainServices.Workflow)
	for _, wfName := range cfg.Workflows.Enabled {
		var wf domainServices.Workflow
		switch wfName {
		case "default":
			wf = workflows.NewDefaultWorkflow()
		case "basic":
			wf = workflows.NewBasicWorkflow()
		case "advanced":
			wf = workflows.NewAdvancedWorkflow(cfg.Advanced)
		default:
			log.Printf("Warning: unknown workflow %s", wfName)
			continue
		}

		workflowRegistry[wfName] = wf
		log.Printf("Initialized workflow: %s", wfName)
	}

	// Initialize orchestrator
	orchestrator := services.NewOrchestrator(
		providerRegistry,
		workflowRegistry,
		cfg.Workflows.Default,
	)

	// Initialize HTTP handler
	handler := api.NewHandler(orchestrator, cfg)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(api.CORSMiddleware())

	// Routes
	r.Post("/v1/chat/completions", handler.ChatCompletions)
	r.Get("/health", handler.Health)
	r.Get("/workflows", handler.ListWorkflows)

	// HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Starting server on %s", addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Received signal %v, shutting down gracefully...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
			if err := server.Close(); err != nil {
				log.Fatalf("Failed to close server: %v", err)
			}
		}

		log.Println("Server stopped")
	}
}
