# Pipeline Configuration Examples

This directory contains example pipeline configurations for the ADK LLM Proxy reasoning system. Use these as templates for your own configurations.

## Quick Start

1. **Copy an example** that matches your use case
2. **Customize** agent IDs, timeouts, and budget settings
3. **Test** with your data sources
4. **Deploy** to your environment

```bash
# Copy example
cp examples/pipelines/sequential-basic.yaml config/my-pipeline.yaml

# Edit configuration
vim config/my-pipeline.yaml

# Run with custom config
./proxy --config config/my-pipeline.yaml
```

---

## Available Examples

### 1. Sequential Basic (`sequential-basic.yaml`)

**Best for:** Simple, straightforward reasoning tasks

**Features:**
- Linear agent execution (one after another)
- 4 agents: Intent → Reasoning → Inference → Summarization
- Low cost (~$0.01-0.05 per session)
- Fast execution (<1s without LLM, <5s with LLM)
- Contract validation enabled

**Use Cases:**
- Simple question answering
- Intent-based routing
- Basic reasoning without external data
- Development and testing

**Pros:**
- ✓ Simple and predictable
- ✓ Easy to debug
- ✓ Low cost
- ✓ Fast for simple tasks

**Cons:**
- ✗ No parallel execution (slower for multi-source data)
- ✗ No data retrieval (limited to reasoning only)

**Example Query:**
```
User: "What is the best approach for implementing caching?"

Pipeline:
1. Intent Detection → Detects "advice_seeking" intent
2. Reasoning Structure → Builds reasoning goal hierarchy
3. Inference → Generates recommendations
4. Summarization → Formats response

Cost: ~$0.02
Latency: ~3s
```

---

### 2. Parallel Retrieval (`parallel-retrieval.yaml`)

**Best for:** Multi-source data aggregation with performance optimization

**Features:**
- Parallel agent execution (independent agents run simultaneously)
- 9 agents: Intent → Planner → [GitLab + YouTrack + Database] → Synthesis → Inference → Validation → Summarization
- Medium cost (~$0.05-0.20 per session)
- Faster than sequential for multi-source retrieval
- Dependency graph with parallel branches

**Use Cases:**
- Project status reports (GitLab + YouTrack + metrics)
- Cross-system analytics
- Multi-source data aggregation
- Comprehensive issue analysis

**Pros:**
- ✓ Parallel execution (50% faster than sequential)
- ✓ Scales to many data sources
- ✓ Clear dependency graph
- ✓ Comprehensive validation

**Cons:**
- ✗ More complex configuration
- ✗ Higher cost (more agents)
- ✗ Single point of failure (any retrieval failure blocks synthesis)

**Example Query:**
```
User: "What critical production issues do we have in GitLab and YouTrack?"

Pipeline:
1. Intent Detection → Detects "query" intent with filters (critical, production)
2. Retrieval Planner → Creates retrieval plans for GitLab + YouTrack
3. [Parallel] Retrieval (GitLab, YouTrack, Database) → Fetch data concurrently
4. Context Synthesis → Merge and deduplicate results
5. Inference → Identify patterns and priorities
6. Validation → Check completeness
7. Summarization → Generate report

Cost: ~$0.12
Latency: ~18s (vs ~30s sequential)
```

---

### 3. Conditional Validation (`conditional-validation.yaml`)

**Best for:** Adaptive workflows with variable complexity

**Features:**
- Conditional agent execution (agents run only if needed)
- 9 agents (4-9 execute depending on conditions)
- Variable cost (~$0.01-0.15 per session)
- Variable latency (<3s best case, <20s worst case)
- Graceful degradation (optional agents can be skipped)

**Conditions:**
- `has_query_intent` - Retrieval needed?
- `has_retrieval_plan` - Plan exists?
- `high_stakes` - Validation required?
- `validation_failed` - Deep reasoning needed?

**Use Cases:**
- Mixed query types (simple lookups vs complex analytics)
- Budget-constrained workflows
- Progressive reasoning (escalate complexity as needed)
- Adaptive question answering

**Pros:**
- ✓ Cost-efficient (skip unnecessary agents)
- ✓ Adapts to query complexity
- ✓ Fast for simple queries
- ✓ Can handle complex queries when needed

**Cons:**
- ✗ Non-deterministic execution (harder to debug)
- ✗ Requires careful condition design
- ✗ Contract validation disabled (optional agents)

**Example Queries:**

**Simple Query:**
```
User: "What is 2+2?"

Executed Agents:
- Intent Detection ✓ (no query intent)
- Reasoning Structure ✓
- Retrieval Planner ✗ (skipped, no query intent)
- Retrieval ✗ (skipped, no plan)
- Synthesis ✗ (skipped, no data)
- Inference ✓
- Validation ✗ (skipped, not high stakes)
- Summarization ✓

Cost: ~$0.01
Latency: ~3s
```

**Complex Query:**
```
User: "What are the critical production issues in GitLab?"

Executed Agents:
- Intent Detection ✓ (query intent + high_stakes flag)
- Reasoning Structure ✓
- Retrieval Planner ✓ (has query intent)
- Retrieval ✓ (has plan)
- Synthesis ✓ (has data)
- Inference ✓
- Validation ✓ (high stakes)
- Deep Reasoning ? (only if validation fails)
- Summarization ✓

Cost: ~$0.12
Latency: ~18s
```

---

### 4. Production Full (`production-full.yaml`)

**Best for:** Production deployments with comprehensive features

**Features:**
- Complete agent pipeline (10 agents)
- Parallel execution with dependency graph
- Full provider support (Ollama, DeepSeek, OpenAI, Anthropic)
- Budget controls with graceful degradation
- Aggressive caching (10K entries, 500MB)
- Performance monitoring (Prometheus metrics)
- Distributed tracing
- Alerting (Slack, PagerDuty)
- Security (PII masking, field truncation)
- Health checks and graceful shutdown

**Use Cases:**
- Production deployments
- High-throughput services
- Mission-critical applications
- Multi-tenant platforms

**Pros:**
- ✓ Production-ready (all features enabled)
- ✓ Comprehensive monitoring
- ✓ Graceful degradation
- ✓ Multi-provider fallback chains
- ✓ Cost controls

**Cons:**
- ✗ Complex configuration (many settings)
- ✗ Requires all infrastructure (Prometheus, alerting)
- ✗ Higher maintenance burden

**Configuration Sections:**
- **Pipeline:** 10-agent parallel execution
- **LLM:** 4 providers with fallback chains
- **Budget:** $2.00/session, $0.50/agent
- **Cache:** 10K entries, 500MB, aggressive TTLs
- **Observability:** Prometheus, tracing, alerting
- **Performance:** SLA targets (p50: 5s, p99: 30s)
- **Deployment:** Server config, health checks, shutdown

**Performance Targets:**
- p50 latency: ~5s
- p99 latency: ~25s
- Cache hit rate: 40%+
- Success rate: 99%+
- Cost: $0.10-0.50 per session

**Scaling:**
- Horizontal: Deploy multiple instances (stateless agents)
- Vertical: 4 CPU cores, 8GB RAM per instance
- Cache: Use Redis for shared cache
- Logs: Ship to ELK stack

---

## Choosing the Right Example

Use this decision tree:

```
Do you need external data sources?
├─ NO → sequential-basic.yaml
│        (Simple reasoning only)
│
└─ YES → Do you need multiple data sources?
         ├─ NO → sequential-basic.yaml
         │        (Add retrieval agents)
         │
         └─ YES → Is your workload variable (simple + complex queries)?
                  ├─ YES → conditional-validation.yaml
                  │        (Adaptive workflow)
                  │
                  └─ NO → Is this for production?
                           ├─ YES → production-full.yaml
                           │        (Full features)
                           │
                           └─ NO → parallel-retrieval.yaml
                                    (Development/staging)
```

---

## Customization Guide

### 1. Adjust Timeouts

```yaml
agents:
  - id: my_agent
    timeout: 5000  # 5 seconds (default)
    # Increase for slow agents:
    # - LLM-heavy: 10000-30000ms
    # - External APIs: 15000-20000ms
    # - Complex processing: 30000-60000ms
```

### 2. Configure Budget

```yaml
llm:
  budget:
    session_budget_usd: 1.00    # Max per session
    agent_budget_usd: 0.20      # Max per agent
    warning_threshold: 0.80     # Warn at 80%
    critical_agents:
      - validation              # Bypass budget limits
```

### 3. Enable/Disable Agents

```yaml
agents:
  - id: optional_agent
    enabled: false  # Disable agent
```

### 4. Add Custom Agents

```yaml
agents:
  - id: my_custom_agent
    enabled: true
    timeout: 10000
    retry: 1
    depends_on:
      - previous_agent  # Dependencies
```

### 5. Configure Caching

```yaml
llm:
  cache:
    enabled: true
    classification_ttl: 86400   # 24 hours
    synthesis_ttl: 3600         # 1 hour
    inference_ttl: 1800         # 30 minutes
```

---

## Testing Your Configuration

### 1. Validate Syntax

```bash
# Check YAML syntax
yamllint config/my-pipeline.yaml

# Validate with proxy
./proxy --config config/my-pipeline.yaml --validate
```

### 2. Test Locally

```bash
# Run proxy with custom config
./proxy --config config/my-pipeline.yaml --port 8001

# Send test request
curl -X POST http://localhost:8001/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Test query"}],
    "stream": false
  }'
```

### 3. Monitor Execution

```bash
# Check agent execution order
curl http://localhost:8001/debug/last-context | jq '.audit.agent_runs'

# Check LLM decisions
curl http://localhost:8001/debug/last-context | jq '.llm.decisions'

# Check budget usage
curl http://localhost:8001/debug/budget
```

---

## Performance Tips

### 1. Use Parallel Execution

Parallel execution can reduce latency by 50% for multi-agent pipelines:

```yaml
pipeline:
  mode: parallel  # ✓ Enable parallelism
  agents:
    - id: agent_a
      depends_on: [root]
    - id: agent_b
      depends_on: [root]  # Runs in parallel with agent_a
```

### 2. Enable Caching

Caching can reduce costs by 40-60% for repeated queries:

```yaml
llm:
  cache:
    enabled: true
    max_entries: 10000
    classification_ttl: 86400  # 24h for stable tasks
```

### 3. Choose Right Models

Use cheap models for simple tasks:

| Task | Model | Cost/1K tok |
|------|-------|-------------|
| Classification | deepseek-chat | $0.0001 |
| Simple synthesis | gpt-4o-mini | $0.00015 |
| Complex reasoning | gpt-4o | $0.0025 |

### 4. Set Budgets

Prevent cost overruns with strict budgets:

```yaml
llm:
  budget:
    session_budget_usd: 1.00
    agent_budget_usd: 0.20
```

### 5. Disable Validation in Production

Contract validation adds ~5-10% overhead:

```yaml
pipeline:
  options:
    validate_contract: false  # Disable in production
```

---

## Troubleshooting

See [docs/reasoning_system/troubleshooting.md](../../docs/reasoning_system/troubleshooting.md) for detailed troubleshooting guide.

**Common issues:**
- **Agent not executing:** Check `enabled: true` and dependencies
- **Pipeline timeout:** Increase agent timeouts
- **Budget exceeded:** Increase budget or use cheaper models
- **High latency:** Enable parallel mode, check provider health
- **Low cache hit rate:** Increase TTLs, normalize prompts

---

## References

- [AgentContext Schema](../../docs/reasoning_system/agent_context_schema.md)
- [Agent Contracts](../../docs/reasoning_system/agent_contracts.md)
- [Pipeline Configuration](../../docs/reasoning_system/pipeline_configuration.md)
- [LLM Selection Policies](../../docs/reasoning_system/llm_selection_policies.md)
- [Troubleshooting Guide](../../docs/reasoning_system/troubleshooting.md)
