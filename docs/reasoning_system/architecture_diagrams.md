# Architecture Diagrams

## Overview

This document provides visual representations of the ADK LLM Proxy reasoning system architecture. Diagrams use ASCII art for universal compatibility.

---

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          ADK LLM Proxy System                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐      ┌──────────────────┐      ┌─────────────────┐   │
│  │   Client    │      │  Presentation    │      │  Infrastructure │   │
│  │  (Emacs,    │─────▶│     Layer        │─────▶│     Layer       │   │
│  │   curl)     │      │  (HTTP API)      │      │  (Providers)    │   │
│  └─────────────┘      └──────────────────┘      └─────────────────┘   │
│                               │                           │            │
│                               ▼                           ▼            │
│                      ┌──────────────────┐      ┌─────────────────┐    │
│                      │  Application     │      │  LLM Providers  │    │
│                      │     Layer        │◀─────│  - OpenAI       │    │
│                      │ (Orchestration)  │      │  - Anthropic    │    │
│                      └──────────────────┘      │  - DeepSeek     │    │
│                               │                │  - Ollama       │    │
│                               ▼                └─────────────────┘    │
│                      ┌──────────────────┐                             │
│                      │   Domain Layer   │                             │
│                      │  (Agents, Models)│                             │
│                      └──────────────────┘                             │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

Flow: Client → API → Orchestrator → Agents → LLM Providers → Response
```

---

## Agent Pipeline Flow

### Sequential Execution

```
┌─────────────┐
│ User Input  │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Intent Detection    │  Detect intents, extract entities
└──────┬──────────────┘
       │  reasoning.intents, reasoning.entities
       ▼
┌─────────────────────┐
│ Reasoning Structure │  Build goal hierarchy, hypotheses
└──────┬──────────────┘
       │  reasoning.hypotheses
       ▼
┌─────────────────────┐
│ Retrieval Planner   │  Generate retrieval plans
└──────┬──────────────┘
       │  retrieval.plans, retrieval.queries
       ▼
┌─────────────────────┐
│ Retrieval Executor  │  Fetch data from sources
└──────┬──────────────┘
       │  retrieval.artifacts
       ▼
┌─────────────────────┐
│ Context Synthesizer │  Normalize and merge facts
└──────┬──────────────┘
       │  enrichment.facts, enrichment.derived_knowledge
       ▼
┌─────────────────────┐
│ Inference Agent     │  Make conclusions
└──────┬──────────────┘
       │  reasoning.conclusions
       ▼
┌─────────────────────┐
│ Validation Agent    │  Check completeness/consistency
└──────┬──────────────┘
       │  diagnostics.validation_reports
       ▼
┌─────────────────────┐
│ Summarization Agent │  Generate final output
└──────┬──────────────┘
       │  reasoning.summary
       ▼
┌─────────────┐
│   Output    │
└─────────────┘
```

---

### Parallel Execution

```
                              ┌─────────────────────┐
                              │ Intent Detection    │
                              └──────────┬──────────┘
                                         │
                                         ▼
                              ┌─────────────────────┐
                              │ Reasoning Structure │
                              └──────────┬──────────┘
                                         │
                                         ▼
                              ┌─────────────────────┐
                              │ Retrieval Planner   │
                              └──────────┬──────────┘
                                         │
                    ┌────────────────────┼────────────────────┐
                    │                    │                    │
                    ▼                    ▼                    ▼
         ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
         │ Retrieval       │  │ Retrieval       │  │ Retrieval       │
         │ (GitLab)        │  │ (YouTrack)      │  │ (Database)      │
         └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
                  │                    │                    │
                  └────────────────────┼────────────────────┘
                                       │
                                       ▼
                            ┌─────────────────────┐
                            │ Context Synthesizer │
                            └──────────┬──────────┘
                                       │
                                       ▼
                            ┌─────────────────────┐
                            │ Inference Agent     │
                            └──────────┬──────────┘
                                       │
                                       ▼
                            ┌─────────────────────┐
                            │ Validation Agent    │
                            └──────────┬──────────┘
                                       │
                                       ▼
                            ┌─────────────────────┐
                            │ Summarization Agent │
                            └──────────┬──────────┘
                                       │
                                       ▼
                                   ┌───────┐
                                   │Output │
                                   └───────┘

Performance: 50% faster than sequential for multi-source retrieval
```

---

### Conditional Execution

```
┌─────────────────────┐
│ Intent Detection    │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Reasoning Structure │
└──────────┬──────────┘
           │
           ▼
   ┌───────────────┐
   │ has_query     │ ◀── Condition Check
   │ intent?       │
   └───────┬───────┘
       NO  │  YES
           │
    ┌──────┼──────┐
    │      ▼      │
    │  ┌─────────────────────┐
    │  │ Retrieval Planner   │
    │  └──────────┬──────────┘
    │             │
    │      ┌──────┼──────┐
    │      ▼      │      │
    │  ┌─────────────────┐  │
    │  │ Retrieval       │  │
    │  └────────┬────────┘  │
    │           │           │
    │           ▼           │
    │  ┌─────────────────────┐
    │  │ Context Synthesizer │
    │  └──────────┬──────────┘
    │             │
    └─────────────┼──────────┘
                  │
                  ▼
       ┌─────────────────────┐
       │ Inference Agent     │
       └──────────┬──────────┘
                  │
                  ▼
          ┌───────────────┐
          │ high_stakes?  │ ◀── Condition Check
          └───────┬───────┘
              NO  │  YES
                  │
           ┌──────┼──────┐
           │      ▼      │
           │  ┌─────────────────┐
           │  │ Validation      │
           │  └────────┬────────┘
           │           │
           └───────────┼────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │ Summarization       │
            └──────────┬──────────┘
                       │
                       ▼
                   ┌───────┐
                   │Output │
                   └───────┘

Agents execute only if conditions are met
Cost: $0.01-0.15 (varies by path taken)
```

---

## AgentContext Structure

### Context Namespaces

```
┌─────────────────────────────────────────────────────────────────┐
│                         AgentContext                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Metadata                                                  │ │
│  │  - session_id, trace_id, created_at, version, locale     │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Reasoning                                                 │ │
│  │  - intents: []Intent                                      │ │
│  │  - entities: map[string]Entity                            │ │
│  │  - hypotheses: []Hypothesis                               │ │
│  │  - conclusions: []Conclusion                              │ │
│  │  - summary: string                                        │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Enrichment                                                │ │
│  │  - facts: []Fact                                          │ │
│  │  - derived_knowledge: []Knowledge                         │ │
│  │  - relationships: []Relationship                          │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Retrieval                                                 │ │
│  │  - plans: []RetrievalPlan                                 │ │
│  │  - queries: []Query                                       │ │
│  │  - artifacts: []Artifact                                  │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ LLM                                                       │ │
│  │  - provider: string                                       │ │
│  │  - model: string                                          │ │
│  │  - usage: LLMUsage                                        │ │
│  │  - decisions: []LLMDecision                               │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Diagnostics                                               │ │
│  │  - errors: []Error                                        │ │
│  │  - warnings: []Warning                                    │ │
│  │  - performance: []PerformanceMetric                       │ │
│  │  - validation_reports: []ValidationReport                 │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │ Audit                                                     │ │
│  │  - agent_runs: []AgentRun                                 │ │
│  │  - diffs: []ContextDiff                                   │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Each agent writes only to designated namespaces
Full audit trail captured in audit.agent_runs
```

---

### Context Flow Through Pipeline

```
Initial Context          After Intent Detection      After Inference
─────────────────        ────────────────────────    ─────────────────────
metadata: {              metadata: {                 metadata: {
  session_id: "123"        session_id: "123"           session_id: "123"
}                        }                           }
reasoning: {}            reasoning: {                reasoning: {
                           intents: [                  intents: [...]
                             {type: "query",           hypotheses: [...]
                              confidence: 0.95}        conclusions: [
                           ],                            {desc: "High activity",
                           entities: {                    confidence: 0.88}
                             project: "my-proj"         ]
                           }                          }
                         }                           enrichment: {
enrichment: {}           enrichment: {}                facts: [...]
                                                     }
retrieval: {}            retrieval: {}               retrieval: {
                                                       artifacts: [...]
                                                     }
llm: {}                  llm: {                      llm: {
                           decisions: [...]            usage: {
                         }                               tokens: 1500,
                                                         cost: 0.02
audit: {                 audit: {                      }
  agent_runs: []           agent_runs: [             }
}                            {agent: "intent",      audit: {
                              status: "success"}       agent_runs: [
                           ]                             {agent: "intent"...},
                         }                               {agent: "inference"...}
                                                       ]
                                                     }

Context accumulates data as it flows through pipeline
Each agent adds to designated namespace
Full history preserved in audit trail
```

---

## LLM Selection Flow

### Model Selection Decision Tree

```
                         ┌─────────────────────┐
                         │ Incoming Request    │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │ Analyze Request     │
                         │ - Task type         │
                         │ - Context size      │
                         │ - Agent ID          │
                         └──────────┬──────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
                    ▼               ▼               ▼
         ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
         │ Simple Task     │ │ Medium Task │ │ Complex Task    │
         │ (<1K tokens)    │ │ (<8K tokens)│ │ (>8K tokens)    │
         └────────┬────────┘ └──────┬──────┘ └────────┬────────┘
                  │                 │                  │
                  ▼                 ▼                  ▼
         ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
         │ deepseek-chat   │ │ gpt-4o-mini │ │ gpt-4o          │
         │ $0.0001/1K tok  │ │ $0.00015/1K │ │ $0.0025/1K      │
         └────────┬────────┘ └──────┬──────┘ └────────┬────────┘
                  │                 │                  │
                  └─────────────────┼──────────────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │ Check Budget        │
                         └──────────┬──────────┘
                                    │
                         ┌──────────┴──────────┐
                         │                     │
                    Budget OK            Budget >80%
                         │                     │
                         ▼                     ▼
              ┌─────────────────┐   ┌─────────────────┐
              │ Use Selected    │   │ Downgrade Model │
              │ Model           │   │ (cheaper option)│
              └────────┬────────┘   └────────┬────────┘
                       │                     │
                       └──────────┬──────────┘
                                  │
                                  ▼
                       ┌─────────────────────┐
                       │ Try Provider        │
                       └──────────┬──────────┘
                                  │
                       ┌──────────┴──────────┐
                       │                     │
                   Success               Failure
                       │                     │
                       ▼                     ▼
            ┌─────────────────┐   ┌─────────────────┐
            │ Return Response │   │ Try Fallback    │
            │ + Log Decision  │   │ Provider        │
            └─────────────────┘   └────────┬────────┘
                                            │
                                            ▼
                                 ┌─────────────────┐
                                 │ Fallback Chain: │
                                 │ 1. Alternative  │
                                 │ 2. Local (free) │
                                 │ 3. Deterministic│
                                 └─────────────────┘
```

---

### Provider Fallback Chain

```
┌─────────────────────────────────────────────────────────────┐
│                    Primary Request                          │
│                    (gpt-4o-mini)                            │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
              ┌─────────────┐
              │ Available?  │
              └──────┬──────┘
               YES   │   NO
                     │
           ┌─────────┼─────────┐
           │                   │
           ▼                   ▼
    ┌──────────────┐    ┌──────────────────┐
    │ Use Primary  │    │ Try Fallback #1  │
    │ (OpenAI)     │    │ (DeepSeek)       │
    └──────┬───────┘    └──────┬───────────┘
           │                   │
           │            ┌──────┴──────┐
           │      Available?      Unavailable
           │            │                │
           │            ▼                ▼
           │     ┌──────────────┐ ┌──────────────────┐
           │     │ Use Fallback │ │ Try Fallback #2  │
           │     │ (DeepSeek)   │ │ (Ollama/Local)   │
           │     └──────┬───────┘ └──────┬───────────┘
           │            │                │
           │            │         ┌──────┴──────┐
           │            │   Available?      Unavailable
           │            │         │                │
           │            │         ▼                ▼
           │            │  ┌──────────────┐ ┌──────────────┐
           │            │  │ Use Local    │ │ Deterministic│
           │            │  │ (Ollama)     │ │ Fallback     │
           │            │  └──────┬───────┘ └──────┬───────┘
           │            │         │                │
           └────────────┴─────────┴────────────────┘
                                  │
                                  ▼
                       ┌─────────────────┐
                       │ Return Response │
                       └─────────────────┘

Fallback chain ensures high availability
Each fallback is cheaper/more available than previous
Final fallback: deterministic rules (no LLM)
```

---

## Request Flow

### Complete Request Processing

```
┌────────────┐
│   Client   │
└──────┬─────┘
       │ POST /v1/chat/completions
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Handler                             │
│  - Parse request                                            │
│  - Extract workflow header                                  │
│  - Validate input                                           │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                   Orchestrator                              │
│  1. Initialize AgentContext                                 │
│  2. Select workflow (default/basic/advanced)                │
│  3. Create event channel                                    │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                 Reasoning Manager                           │
│  1. Load pipeline configuration                             │
│  2. Execute agents (sequential/parallel/conditional)        │
│  3. Track execution in audit trail                          │
└──────┬──────────────────────────────────────────────────────┘
       │
       │ For each agent:
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Agent Execution                          │
│  1. Validate preconditions                                  │
│  2. Execute agent logic                                     │
│  3. Validate postconditions                                 │
│  4. Update AgentContext                                     │
└──────┬──────────────────────────────────────────────────────┘
       │
       │ If agent needs LLM:
       ▼
┌─────────────────────────────────────────────────────────────┐
│                  LLM Orchestrator                           │
│  1. Check cache                                             │
│  2. Select model (based on task/budget)                     │
│  3. Check budget                                            │
│  4. Call provider                                           │
│  5. Track usage/cost                                        │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                  LLM Provider                               │
│  - OpenAI / Anthropic / DeepSeek / Ollama                   │
│  - Stream or buffer response                                │
└──────┬──────────────────────────────────────────────────────┘
       │
       │ Response flows back:
       ▼
┌─────────────────────────────────────────────────────────────┐
│                Response Assembly                            │
│  1. Collect reasoning results                               │
│  2. Collect LLM completions                                 │
│  3. Format final response                                   │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                 HTTP Response                               │
│  - Streaming (SSE) or buffered                              │
│  - OpenAI-compatible format                                 │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌────────────┐
│   Client   │
└────────────┘

Total latency: <5s (p50), <25s (p99)
Cost per request: $0.01-0.50
```

---

## Caching Architecture

### Cache Layers

```
┌─────────────────────────────────────────────────────────────┐
│                      LLM Request                            │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Generate Cache Key                         │
│  SHA256(prompt + model + temperature + max_tokens)          │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                   L1: In-Memory Cache                       │
│  - Fast: <5ms lookup                                        │
│  - Limited: 10K entries, 500MB                              │
│  - TTL: 15min - 24h by task type                            │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
   ┌───────┐
   │ Hit?  │
   └───┬───┘
    YES│NO
       │
       ├─YES────▶ ┌─────────────────┐
       │          │ Return Cached   │
       │          │ Response        │
       │          └─────────────────┘
       │
       ▼ NO
┌─────────────────────────────────────────────────────────────┐
│                 L2: Redis Cache (optional)                  │
│  - Slower: 10-50ms lookup                                   │
│  - Larger: unlimited with eviction                          │
│  - Shared: across multiple instances                        │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
   ┌───────┐
   │ Hit?  │
   └───┬───┘
    YES│NO
       │
       ├─YES────▶ ┌─────────────────┐
       │          │ Return Cached   │
       │          │ Response        │
       │          │ + Update L1     │
       │          └─────────────────┘
       │
       ▼ NO
┌─────────────────────────────────────────────────────────────┐
│                   Call LLM Provider                         │
│  - Fetch fresh response                                     │
│  - Cache in L2 + L1                                         │
│  - Track cost                                               │
└─────────────────────────────────────────────────────────────┘

Cache hit rate targets:
- Classification: >60%
- Synthesis: >40%
- Inference: >20%

Cost savings: 40-60% with effective caching
```

---

## Deployment Architecture

### Single Instance

```
┌────────────────────────────────────────────────────────┐
│                    Server Instance                     │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ┌────────────────────────────────────────────────┐   │
│  │          ADK LLM Proxy (Binary)                │   │
│  │  - HTTP Server (port 8001)                     │   │
│  │  - Reasoning Manager                           │   │
│  │  - LLM Orchestrator                            │   │
│  │  - In-memory cache                             │   │
│  └────────────────────────────────────────────────┘   │
│                                                        │
│  ┌────────────────────────────────────────────────┐   │
│  │          Prometheus Exporter                   │   │
│  │  - Metrics endpoint (port 9090)                │   │
│  └────────────────────────────────────────────────┘   │
│                                                        │
└────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
  ┌─────────────────┐          ┌─────────────────┐
  │ LLM Providers   │          │ Monitoring      │
  │ (Cloud APIs)    │          │ (Prometheus)    │
  └─────────────────┘          └─────────────────┘

Use for: Development, testing, small deployments
Max throughput: ~100 req/s
```

---

### Multi-Instance (Production)

```
                    ┌─────────────────┐
                    │ Load Balancer   │
                    │ (Nginx/HAProxy) │
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
            ▼                ▼                ▼
    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
    │ Instance 1   │ │ Instance 2   │ │ Instance 3   │
    │ (port 8001)  │ │ (port 8001)  │ │ (port 8001)  │
    │              │ │              │ │              │
    │ Proxy Binary │ │ Proxy Binary │ │ Proxy Binary │
    └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
           │                │                │
           └────────────────┼────────────────┘
                            │
                            ▼
                   ┌─────────────────┐
                   │  Redis Cluster  │
                   │  (Shared Cache) │
                   └────────┬────────┘
                            │
           ┌────────────────┼────────────────┐
           │                │                │
           ▼                ▼                ▼
  ┌─────────────────┐ ┌──────────────┐ ┌─────────────┐
  │ LLM Providers   │ │ Prometheus   │ │ ELK Stack   │
  │ (Cloud APIs)    │ │ (Metrics)    │ │ (Logs)      │
  └─────────────────┘ └──────────────┘ └─────────────┘

Scaling characteristics:
- Horizontal: Add instances behind load balancer
- Stateless: No shared state between instances
- Shared cache: Redis for cross-instance caching
- Max throughput: ~5000 req/s (50 instances)
```

---

## Monitoring Dashboard Layout

### Key Metrics View

```
┌─────────────────────────────────────────────────────────────┐
│                   Reasoning System Dashboard                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Latency (p50/p99)          Throughput                     │
│  ┌─────────────────┐        ┌─────────────────┐           │
│  │  4.2s / 18.7s   │        │  125 req/s      │           │
│  │  ▓▓▓▓▓▓░░░░      │        │  ▓▓▓▓▓▓▓▓░░    │           │
│  └─────────────────┘        └─────────────────┘           │
│                                                             │
│  Budget Usage               Cache Hit Rate                 │
│  ┌─────────────────┐        ┌─────────────────┐           │
│  │  $0.45 / $2.00  │        │  58%            │           │
│  │  ▓▓░░░░░░░░      │        │  ▓▓▓▓▓▓░░░░    │           │
│  └─────────────────┘        └─────────────────┘           │
│                                                             │
│  Agent Performance                                          │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ intent_detection     ████░░░░░░   150ms               │ │
│  │ reasoning_structure  ████░░░░░░   200ms               │ │
│  │ retrieval_gitlab     ██████████   3500ms              │ │
│  │ inference            █████░░░░░   1800ms              │ │
│  │ summarization        ███░░░░░░░   120ms               │ │
│  └───────────────────────────────────────────────────────┘ │
│                                                             │
│  LLM Model Usage                                            │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ deepseek-chat   ████████░░  45%  ($0.02)             │ │
│  │ gpt-4o-mini     ██████░░░░  30%  ($0.15)             │ │
│  │ gpt-4o          ███░░░░░░░  15%  ($0.28)             │ │
│  │ ollama/mistral  ██░░░░░░░░  10%  ($0.00)             │ │
│  └───────────────────────────────────────────────────────┘ │
│                                                             │
│  Error Rate: 0.8%                Recent Alerts: 0          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## References

- [AgentContext Schema](./agent_context_schema.md)
- [Agent Contracts](./agent_contracts.md)
- [Pipeline Configuration](./pipeline_configuration.md)
- [LLM Selection Policies](./llm_selection_policies.md)
- [Performance Targets](./performance_targets.md)
- [Troubleshooting Guide](./troubleshooting.md)
- Source: `src/golang/internal/`
