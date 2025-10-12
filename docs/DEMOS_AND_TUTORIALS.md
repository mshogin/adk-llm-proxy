# MCP Integration Demos and Tutorials

## Overview

This document provides scripts and guidance for creating demos and tutorials showcasing MCP integration capabilities.

## Table of Contents

1. [Quick Start Demo](#quick-start-demo)
2. [Tutorial Scripts](#tutorial-scripts)
3. [Live Demo Scenarios](#live-demo-scenarios)
4. [Video Tutorial Outlines](#video-tutorial-outlines)
5. [Interactive Examples](#interactive-examples)
6. [Presentation Materials](#presentation-materials)

---

## Quick Start Demo

### 5-Minute Demo Script

**Goal**: Show basic MCP integration in under 5 minutes

**Setup**:
```bash
# Terminal 1: Start the proxy
python main.py -provider openai -model gpt-4o-mini

# Terminal 2: Health check
make check-mcp
```

**Demo Script**:

```bash
# 1. Show MCP Server Status (30 seconds)
echo "=== MCP Server Status ==="
make check-mcp

# Expected output:
# ✅ YouTrack Server: Connected (7 tools available)
# ✅ GitLab Server: Connected (7 tools available)

# 2. Simple Query via API (1 minute)
echo "=== Basic Query ==="
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "What is the status of epic AUTH-123?"}
    ]
  }' | jq '.choices[0].message.content'

# 3. Cross-Platform Query (2 minutes)
echo "=== Cross-Platform Analysis ==="
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Show me all commits related to AUTH-123 from last week"}
    ]
  }' | jq '.choices[0].message.content'

# 4. Multi-Tool Orchestration (1.5 minutes)
echo "=== Complex Workflow ==="
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Generate a weekly progress report for the Authentication epic, including code changes and task updates"}
    ]
  }' | jq '.choices[0].message.content'
```

**Talking Points**:
1. "MCP automatically discovers tools from YouTrack and GitLab"
2. "The LLM decides which tools to use based on the query"
3. "Multiple tools can be orchestrated automatically"
4. "Results are combined into a coherent response"

---

## Tutorial Scripts

### Tutorial 1: Setting Up Your First MCP Server (15 minutes)

**Objective**: Create and configure a simple MCP server

**Steps**:

1. **Introduction** (2 min)
   - What is MCP?
   - Why use MCP servers?
   - Overview of what we'll build

2. **Create Server Structure** (3 min)
   ```bash
   mkdir -p mcps/weather
   cd mcps/weather
   touch __init__.py server.py requirements.txt
   ```

3. **Implement Basic Server** (5 min)
   - Show server class template
   - Add one tool (get_weather)
   - Explain tool registration

4. **Configure and Test** (3 min)
   - Add to config.yaml
   - Start proxy
   - Test via API call

5. **Wrap-up** (2 min)
   - What we learned
   - Next steps
   - Resources

### Tutorial 2: Building Production-Ready Tools (20 minutes)

**Objective**: Add error handling, validation, and best practices

**Steps**:

1. **Error Handling** (5 min)
   - Try/except patterns
   - User-friendly error messages
   - Logging

2. **Input Validation** (5 min)
   - JSON schema validation
   - Custom validation logic
   - Sanitization

3. **Performance** (5 min)
   - Caching strategies
   - Connection pooling
   - Timeout handling

4. **Testing** (5 min)
   - Unit tests
   - Integration tests
   - Mocking external APIs

### Tutorial 3: Multi-Server Orchestration (25 minutes)

**Objective**: Build workflows using multiple MCP servers

**Steps**:

1. **Planning Workflow** (5 min)
   - Identify required tools
   - Map data dependencies
   - Design orchestration flow

2. **Implement Coordination** (10 min)
   - Sequential execution
   - Parallel execution
   - Data passing between tools

3. **Error Recovery** (5 min)
   - Fallback strategies
   - Retry logic
   - Graceful degradation

4. **Testing & Debugging** (5 min)
   - Debug logging
   - Tracing tool execution
   - Performance profiling

---

## Live Demo Scenarios

### Scenario 1: Project Management Intelligence

**Use Case**: Automated epic analysis and reporting

**Demo Flow**:
```
1. User: "What's the status of the Authentication epic?"
   → System discovers epic via YouTrack
   → Returns epic details, task breakdown, progress

2. User: "Show me related code changes"
   → System links tasks to commits via GitLab
   → Analyzes commit messages and diffs
   → Returns code activity summary

3. User: "Generate a weekly progress report"
   → System combines YouTrack + GitLab data
   → Analyzes trends and blockers
   → Creates comprehensive report
```

**Key Points to Highlight**:
- Automatic tool discovery
- Cross-platform data correlation
- Intelligent orchestration
- Natural language interface

### Scenario 2: Developer Productivity

**Use Case**: Quickly find information without leaving terminal

**Demo Flow**:
```bash
# Find my assigned tasks
curl -X POST http://localhost:8000/v1/chat/completions \
  -d '{"messages": [{"role": "user", "content": "Show my assigned tasks"}]}'

# Check CI/CD status
curl -X POST http://localhost:8000/v1/chat/completions \
  -d '{"messages": [{"role": "user", "content": "Are there any failing pipelines?"}]}'

# Code review assistant
curl -X POST http://localhost:8000/v1/chat/completions \
  -d '{"messages": [{"role": "user", "content": "Summarize MR #123"}]}'
```

### Scenario 3: Knowledge Discovery

**Use Case**: Semantic search across documentation

**Demo Flow**:
```
1. Index documentation folder
   → RAG server scans files
   → Builds vector index
   → Creates knowledge graph

2. Query: "How do I implement authentication?"
   → Semantic search finds relevant docs
   → Extracts key information
   → Returns structured answer with sources

3. Query: "What are related concepts?"
   → Traverses knowledge graph
   → Finds connected topics
   → Suggests related reading
```

---

## Video Tutorial Outlines

### Video 1: "MCP Integration in 10 Minutes"

**Target Audience**: Developers new to MCP

**Duration**: 10 minutes

**Outline**:
1. **Intro** (1 min)
   - What problem does MCP solve?
   - Quick preview of what we'll build

2. **Setup** (2 min)
   - Clone repo
   - Install dependencies
   - Configure environment

3. **First Query** (3 min)
   - Start server
   - Make API call
   - See MCP in action

4. **How It Works** (3 min)
   - Architecture diagram
   - Tool discovery
   - Execution flow

5. **Next Steps** (1 min)
   - Additional resources
   - Join community
   - Call to action

**Production Notes**:
- Screen recording with narration
- Split screen: code + terminal
- Highlight key commands
- Add captions for accessibility

### Video 2: "Building Your First MCP Server"

**Target Audience**: Developers ready to create custom servers

**Duration**: 20 minutes

**Outline**:
1. **Planning** (3 min)
   - Choose integration target
   - Define tools needed
   - Design API interaction

2. **Implementation** (10 min)
   - Server skeleton
   - Tool registration
   - API integration
   - Error handling

3. **Testing** (4 min)
   - Unit tests
   - Integration tests
   - Manual testing

4. **Deployment** (3 min)
   - Configuration
   - Health checks
   - Monitoring

**Production Notes**:
- Live coding session
- Show mistakes and debugging
- Pause for explanations
- Include code repository link

### Video 3: "Advanced MCP Patterns"

**Target Audience**: Experienced MCP users

**Duration**: 30 minutes

**Outline**:
1. **Multi-Tool Workflows** (10 min)
   - Orchestration patterns
   - Data transformation
   - Error recovery

2. **Performance Optimization** (10 min)
   - Caching strategies
   - Parallel execution
   - Resource management

3. **Production Patterns** (10 min)
   - Monitoring
   - Security
   - Scaling

---

## Interactive Examples

### Jupyter Notebook: MCP Integration Basics

```python
# mcp_integration_tutorial.ipynb

# Cell 1: Setup
import httpx
import json

API_URL = "http://localhost:8000/v1/chat/completions"

# Cell 2: Helper Function
async def query_mcp(message: str):
    """Send a query to the MCP-enabled proxy."""
    async with httpx.AsyncClient() as client:
        response = await client.post(
            API_URL,
            json={
                "model": "gpt-4o-mini",
                "messages": [{"role": "user", "content": message}]
            },
            timeout=60.0
        )
        return response.json()["choices"][0]["message"]["content"]

# Cell 3: Example 1 - Simple Query
result = await query_mcp("What is the status of AUTH-123?")
print(result)

# Cell 4: Example 2 - Complex Query
result = await query_mcp(
    "Generate a report for the Authentication epic including "
    "task progress and recent commits"
)
print(result)

# Cell 5: Exercise - Try Your Own Query
# TODO: Write your own query here
result = await query_mcp("YOUR_QUERY_HERE")
print(result)
```

### Command-Line Tutorial

```bash
#!/bin/bash
# interactive_mcp_tutorial.sh

echo "=== MCP Integration Tutorial ==="
echo ""

# Step 1
echo "Step 1: Check MCP Server Status"
echo "Command: make check-mcp"
read -p "Press Enter to run..."
make check-mcp
echo ""

# Step 2
echo "Step 2: Simple Query"
echo "Let's find an epic in YouTrack"
read -p "Enter epic ID (e.g., AUTH-123): " EPIC_ID
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"gpt-4o-mini\",
    \"messages\": [{\"role\": \"user\", \"content\": \"Show details of ${EPIC_ID}\"}]
  }" | jq '.choices[0].message.content'
echo ""

# Step 3
echo "Step 3: Cross-Platform Query"
echo "Let's find related commits"
read -p "Press Enter to continue..."
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"gpt-4o-mini\",
    \"messages\": [{\"role\": \"user\", \"content\": \"Show commits related to ${EPIC_ID}\"}]
  }" | jq '.choices[0].message.content'
echo ""

echo "=== Tutorial Complete! ==="
echo "Next: Try creating your own MCP server"
```

---

## Presentation Materials

### Slide Deck Outline

**Title**: "Intelligent LLM Integration with MCP"

**Slides**:

1. **Title Slide**
   - Project name
   - Your name/org
   - Date

2. **The Problem**
   - LLMs are powerful but isolated
   - Manual API integration is tedious
   - Need for standardized tool access

3. **The Solution: MCP**
   - What is Model Context Protocol?
   - Standardized server interface
   - Automatic tool discovery

4. **Architecture**
   - System diagram
   - Component overview
   - Data flow

5. **Capabilities**
   - YouTrack integration
   - GitLab integration
   - RAG server
   - Custom servers

6. **Live Demo**
   - [Live demonstration]

7. **Key Benefits**
   - Automatic orchestration
   - Natural language interface
   - Extensible architecture
   - Production-ready

8. **Getting Started**
   - Installation
   - Configuration
   - First query

9. **Use Cases**
   - Project management
   - Developer productivity
   - Knowledge discovery
   - Custom workflows

10. **Resources**
    - Documentation links
    - GitHub repository
    - Community
    - Contact

### Demo Environment Setup

```bash
# setup_demo_environment.sh

#!/bin/bash
set -e

echo "Setting up MCP demo environment..."

# 1. Install dependencies
pip install -r requirements.txt

# 2. Configure demo data
cp config.example.yaml config.yaml
# Edit with demo credentials

# 3. Start servers
python main.py -provider openai -model gpt-4o-mini &
SERVER_PID=$!

# Wait for startup
sleep 5

# 4. Verify health
make check-mcp

echo "Demo environment ready!"
echo "Server PID: $SERVER_PID"
echo ""
echo "Run demos with:"
echo "  ./demos/demo1_basic.sh"
echo "  ./demos/demo2_advanced.sh"
echo ""
echo "Stop with: kill $SERVER_PID"
```

---

## Recording Guidelines

### Screen Recording Setup

**Tools**:
- OBS Studio (free, cross-platform)
- QuickTime (macOS)
- Camtasia (paid, powerful editing)

**Settings**:
- Resolution: 1920x1080 or 2560x1440
- Frame rate: 30 fps
- Audio: Clear microphone, no background noise
- Cursor: Large, visible cursor

**Terminal Setup**:
```bash
# Use readable terminal settings
# Font: Menlo/Monaco 18pt
# Colors: High contrast theme
# Prompt: Minimal, clear

export PS1="\$ "  # Simple prompt
```

### Editing Checklist

- [ ] Remove dead air and long pauses
- [ ] Add captions/subtitles
- [ ] Include intro/outro screens
- [ ] Add chapter markers
- [ ] Highlight important commands
- [ ] Speed up slow operations (e.g., pip install)
- [ ] Add background music (optional, low volume)
- [ ] Export at 1080p minimum

### Publishing

**Platforms**:
- YouTube (primary)
- Vimeo (backup)
- Company website (embedded)
- GitHub README (linked)

**Metadata**:
- Clear, descriptive title
- Comprehensive description
- Relevant tags
- Thumbnail with text overlay
- Links to code/docs in description

---

## Community Contributions

### Contributing Your Tutorial

Have a great MCP tutorial idea? We'd love to include it!

**Process**:
1. Open GitHub issue describing your tutorial
2. Get feedback from maintainers
3. Create pull request with:
   - Tutorial script/outline
   - Code examples
   - Any supporting materials
4. Review and merge

**Format**:
- Markdown for text
- Code in repository
- Videos linked (not stored in repo)
- Images in `docs/images/`

### Tutorial Ideas Wanted

- Language-specific integrations (Go, Rust, TypeScript)
- Cloud deployment (AWS, GCP, Azure)
- Kubernetes deployment
- CI/CD integration
- Monitoring and observability
- Custom authentication patterns
- Performance tuning deep-dive

---

## Resources

- **Documentation**: [MCP Integration Guide](MCP_INTEGRATION.md)
- **Code Examples**: `examples/` directory
- **Test Servers**: `mcps/calculator/`, `mcps/template/`
- **Community**: GitHub Discussions
- **Support**: GitHub Issues

---

**Last Updated**: October 2025
**Version**: 1.0.0
**Contributors Welcome!**
