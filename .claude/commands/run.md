# Run Custom Agent Pipeline

Execute a custom agent pipeline with specified agents and prompt (one-time execution, no server).

## Usage

```bash
/run <agents> "<prompt>"
```

**Arguments:**
- `<agents>`: Comma-separated list of agent names (no spaces)
- `<prompt>`: User prompt in quotes

**Agent Names:**
- `intent` - Intent Detection Agent
- `reasoning` - Reasoning Structure Agent
- `retrieval-planner` - Retrieval Planner Agent
- `retrieval-executor` - Retrieval Executor Agent
- `context` - Context Synthesizer Agent
- `inference` - Inference Agent
- `validation` - Validation Agent
- `summary` - Summarization Agent

## Examples

**Full pipeline:**
```bash
/run intent,reasoning,retrieval-planner,retrieval-executor,context,inference,validation,summary "Get my recent GitLab commits"
```

**Partial pipeline (intent + inference only):**
```bash
/run intent,inference "What are the recent tickets?"
```

**Single agent:**
```bash
/run intent "I would like to get the tickets statistics"
```

**Common shortcuts:**
```bash
# Quick analysis (no retrieval)
/run intent,reasoning,inference,summary "Analyze this request"

# With retrieval
/run intent,retrieval-planner,retrieval-executor,context,summary "Get latest commits"

# Validate reasoning
/run intent,reasoning,inference,validation "Complex query here"
```

## How It Works

1. Parses agent list and prompt
2. Uses SAME Orchestrator and pipeline logic as server mode
3. Creates AgentContext and executes agents sequentially
4. Shows detailed output with INPUT/OUTPUT for each agent
5. Exits after completion

## Output Format

```
=== AGENT PIPELINE EXECUTION ===

Agent: Intent Detection
📥 INPUT: "Get my recent commits"
📤 OUTPUT: Detected intents: query_commits (0.99)

Agent: Reasoning Structure
📥 INPUT: intents=[query_commits]
📤 OUTPUT: Generated 2 hypotheses

Agent: Inference
📥 INPUT: hypotheses=[...], facts=[...]
📤 OUTPUT: Drew 3 conclusions

=== FINAL RESULT ===
[Summary of reasoning + conclusions]
```

## Notes

- Agents run in the order specified
- Each agent requires preconditions from previous agents
- Invalid agent order will show clear error message
- Uses same config.yaml as server mode
- Respects LLM provider settings and budget limits
