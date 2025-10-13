# Makefile for LLM Reverse Proxy with MCP Integration
# This ensures all MCP servers are restarted and changes are applied

.PHONY: run clean stop-mcp restart-mcp check-mcp test-mcp help install

# Default target
.DEFAULT_GOAL := help

# Variables
PYTHON_ENV := llm-env
PYTHON := $(shell pyenv which python)
PORT := 8000
PROVIDER := openai
MODEL := gpt-4o-mini

# Colors for output
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
PURPLE := \033[35m
CYAN := \033[36m
RESET := \033[0m

help: ## Show this help message
	@echo "$(CYAN)LLM Reverse Proxy with MCP Integration$(RESET)"
	@echo "$(YELLOW)Available commands:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'

run: check-pyenv stop-mcp restart-mcp ## Stop all MCP processes, restart servers, and run main.py
	@echo "$(BLUE)🚀 Starting LLM Reverse Proxy with fresh MCP servers...$(RESET)"
	@echo "$(YELLOW)🐍 Python Environment: $(PYTHON_ENV)$(RESET)"
	@echo "$(YELLOW)📡 Provider: $(PROVIDER), Model: $(MODEL), Port: $(PORT)$(RESET)"
	@echo "$(GREEN)✅ All MCP servers have been restarted with latest changes$(RESET)"
	@$(PYTHON) main.py -provider $(PROVIDER) -model $(MODEL)

run-debug: stop-mcp restart-mcp ## Run in debug mode with enhanced logging
	@echo "$(BLUE)🔍 Starting in DEBUG mode with fresh MCP servers...$(RESET)"
	@DEBUG=true $(PYTHON) main.py -provider $(PROVIDER) -model $(MODEL)

stop-mcp: ## Stop all running MCP server processes
	@echo "$(RED)🛑 Stopping all MCP server processes...$(RESET)"
	@pkill -f "mcps.youtrack.server" 2>/dev/null || true
	@pkill -f "mcps.gitlab.server" 2>/dev/null || true
	@pkill -f "mcp-server-filesystem" 2>/dev/null || true
	@pkill -f "youtrack-mcp-server" 2>/dev/null || true
	@pkill -f "gitlab-mcp-server" 2>/dev/null || true
	@sleep 2
	@echo "$(GREEN)✅ All MCP processes stopped$(RESET)"

restart-mcp: ## Restart MCP servers (called by run target)
	@echo "$(BLUE)🔄 Ensuring MCP servers are ready for fresh connections...$(RESET)"
	@echo "$(YELLOW)   YouTrack Server: Ready for connection$(RESET)"
	@echo "$(YELLOW)   GitLab Server: Ready for connection$(RESET)"
	@echo "$(YELLOW)   Filesystem Server: Ready for connection$(RESET)"
	@echo "$(GREEN)✅ MCP servers prepared for fresh startup$(RESET)"

check-mcp: ## Check MCP server connectivity and tools
	@echo "$(BLUE)🔍 Checking MCP server connectivity...$(RESET)"
	@$(PYTHON) scripts/debug_mcp_connection.py

test-mcp: ## Test MCP tool functionality
	@echo "$(BLUE)🧪 Testing MCP tool functionality...$(RESET)"
	@$(PYTHON) tests/domain/services/test_mcp_reasoning.py

test-tickets: ## Test ticket retrieval specifically
	@echo "$(BLUE)🎫 Testing ticket retrieval...$(RESET)"
	@$(PYTHON) examples/get_my_assigned_tickets.py

test-enhanced-reasoning: ## Test enhanced reasoning with ticket display
	@echo "$(BLUE)🧠 Testing enhanced reasoning pipeline...$(RESET)"
	@$(PYTHON) tests/integration/youtrack/test_ticket_display.py

clean: stop-mcp ## Clean up all processes and temporary files
	@echo "$(RED)🧹 Cleaning up...$(RESET)"
	@find . -name "*.pyc" -delete 2>/dev/null || true
	@find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
	@rm -f *.pid 2>/dev/null || true
	@echo "$(GREEN)✅ Cleanup completed$(RESET)"

activate-env: ## Activate the pyenv environment for this project
	@echo "$(BLUE)🐍 Activating pyenv environment...$(RESET)"
	@pyenv local $(PYTHON_ENV)
	@echo "$(GREEN)✅ pyenv environment $(PYTHON_ENV) activated for this project$(RESET)"
	@echo "$(YELLOW)💡 You can now run: make run$(RESET)"

setup-env: activate-env install ## Setup pyenv environment and install dependencies

install: ## Install dependencies
	@echo "$(BLUE)📦 Installing dependencies...$(RESET)"
	@$(PYTHON) -m pip install -r requirements.txt
	@echo "$(GREEN)✅ Dependencies installed$(RESET)"

# Development targets
dev: ## Run with development settings (auto-reload)
	@echo "$(PURPLE)👨‍💻 Starting development server with auto-reload...$(RESET)"
	@DEBUG=true $(MAKE) run

# Alternative run commands with different providers
run-ollama: PROVIDER=ollama
run-ollama: MODEL=mistral
run-ollama: run ## Run with Ollama provider

run-deepseek: PROVIDER=deepseek
run-deepseek: MODEL=deepseek-chat
run-deepseek: run ## Run with DeepSeek provider

# Port variants
run-8001: PORT=8001
run-8001: run ## Run on port 8001

run-8002: PORT=8002
run-8002: run ## Run on port 8002

run-8003: PORT=8003
run-8003: run ## Run on port 8003

# Quick status check
status: ## Show system status
	@echo "$(CYAN)📊 System Status:$(RESET)"
	@echo "$(YELLOW)Python processes:$(RESET)"
	@ps aux | grep python | grep -E "(main\.py|mcp)" || echo "  No relevant Python processes running"
	@echo "$(YELLOW)Port status:$(RESET)"
	@lsof -i :$(PORT) 2>/dev/null || echo "  Port $(PORT) is available"
	@echo "$(YELLOW)MCP servers:$(RESET)"
	@pgrep -f "mcps\." | wc -l | xargs printf "  %s MCP server processes running\n"

# Force restart everything
force-restart: ## Force kill all processes and restart
	@echo "$(RED)💥 Force restarting everything...$(RESET)"
	@pkill -9 -f "python.*main\.py" 2>/dev/null || true
	@pkill -9 -f "mcps\." 2>/dev/null || true
	@pkill -9 -f "mcp-server" 2>/dev/null || true
	@sleep 3
	@$(MAKE) run

# Logs and monitoring
logs: ## Show recent logs (if any log files exist)
	@echo "$(BLUE)📜 Recent logs:$(RESET)"
	@find . -name "*.log" -mtime -1 -exec echo "$(YELLOW){}:$(RESET)" \; -exec tail -10 {} \; 2>/dev/null || echo "No recent log files found"

# Configuration check
check-pyenv: ## Check if correct pyenv environment is active
	@echo "$(BLUE)🐍 Checking pyenv environment...$(RESET)"
	@if pyenv version | grep -q "$(PYTHON_ENV)"; then \
		echo "$(GREEN)✅ pyenv environment '$(PYTHON_ENV)' is active$(RESET)"; \
	else \
		echo "$(RED)❌ pyenv environment '$(PYTHON_ENV)' not active$(RESET)"; \
		echo "$(YELLOW)💡 Run: pyenv activate $(PYTHON_ENV)$(RESET)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)🐍 Using Python: $(PYTHON)$(RESET)"

check-config: ## Validate configuration files
	@echo "$(BLUE)⚙️  Checking configuration...$(RESET)"
	@test -f config.yaml && echo "$(GREEN)✅ config.yaml exists$(RESET)" || echo "$(RED)❌ config.yaml missing$(RESET)"
	@test -f requirements.txt && echo "$(GREEN)✅ requirements.txt exists$(RESET)" || echo "$(RED)❌ requirements.txt missing$(RESET)"
	@$(PYTHON) -c "from src.infrastructure.config.config import config; print('$(GREEN)✅ Configuration loads successfully$(RESET)')" 2>/dev/null || echo "$(RED)❌ Configuration load failed$(RESET)"

# Golang targets
.PHONY: go-build go-test go-run go-clean go-lint

go-build: ## Build Golang proxy binary
	@echo "$(BLUE)🔨 Building Golang proxy...$(RESET)"
	@go build -o bin/proxy ./src/golang/cmd/proxy
	@echo "$(GREEN)✅ Build complete: bin/proxy$(RESET)"

go-test: ## Run Golang tests
	@echo "$(BLUE)🧪 Running Golang tests...$(RESET)"
	@go test ./src/golang/... -v -cover
	@echo "$(GREEN)✅ Tests complete$(RESET)"

go-test-coverage: ## Run Golang tests with coverage report
	@echo "$(BLUE)📊 Running Golang tests with coverage...$(RESET)"
	@go test ./src/golang/... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(RESET)"

go-run: go-build ## Run Golang proxy
	@echo "$(BLUE)🚀 Starting Golang proxy...$(RESET)"
	@./bin/proxy --config config-golang.yaml --port 8001

go-clean: ## Clean Golang build artifacts
	@echo "$(RED)🧹 Cleaning Golang build artifacts...$(RESET)"
	@rm -f bin/proxy coverage.out coverage.html
	@go clean
	@echo "$(GREEN)✅ Cleanup complete$(RESET)"

go-lint: ## Run Golang linters
	@echo "$(BLUE)🔍 Running Golang linters...$(RESET)"
	@gofmt -l src/golang/ | grep . && echo "$(RED)❌ Code needs formatting$(RESET)" && exit 1 || echo "$(GREEN)✅ Code is formatted$(RESET)"
	@go vet ./src/golang/...
	@echo "$(GREEN)✅ Lint checks passed$(RESET)"

go-deps: ## Download Golang dependencies
	@echo "$(BLUE)📦 Downloading Golang dependencies...$(RESET)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(RESET)"