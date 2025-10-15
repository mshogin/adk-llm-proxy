# Intent Detection Agent Documentation

## Overview

The Intent Detection Agent is the first agent in the reasoning pipeline, responsible for detecting user intent and extracting structured entities from natural language input. It uses rule-based classification with keyword matching and regex patterns to achieve deterministic, high-performance intent detection without LLM calls.

**Location**: `src/golang/internal/domain/services/agents/intent_detection.go`

**Key Characteristics**:
- **Zero LLM calls** - fully rule-based for speed (<50ms)
- **Deterministic** - same input always produces same output
- **High confidence** - pattern-based scoring with weighted matches
- **Entity extraction** - structured data extraction (projects, dates, providers, statuses)
- **First agent** - no preconditions, initializes reasoning pipeline

## Supported Intent Types

The agent detects 7 distinct intent types, each with specific keywords and patterns:

### 1. query_commits
**Description**: User wants information about code commits, changes, or version history

**Keywords**: commit, commits, changes, changelog, version history, git, repository, code change, pushed, merged

**Regex Patterns**:
- `(?i)\b(latest|recent|last)\s+(commit|change)s?\b` - "latest commits", "recent changes"
- `(?i)\bwhat('s| is| are)?\s+(changed|new|updated)\b` - "what changed?", "what's new?"
- `(?i)\b(show|list|get)\s+commit` - "show commits", "list commits"

**Confidence Weight**: 0.9

**Examples**:
```
✓ "show me the latest commits"
✓ "what changed recently?"
✓ "list commits from last week"
✓ "get recent changes from gitlab-mcp"
```

### 2. query_issues
**Description**: User wants information about issues, tasks, bugs, or tickets

**Keywords**: issue, issues, task, tasks, bug, bugs, ticket, tickets, story, stories, epic

**Regex Patterns**:
- `(?i)\b(open|active|pending|closed)\s+(issue|task|bug)s?\b` - "open issues", "active tasks"
- `(?i)\b(show|list|get)\s+(issue|task|bug)s?\b` - "show issues", "list tasks"
- `(?i)\bwhat('s| is| are)?\s+(my|our)?\s+(task|issue|bug)s?\b` - "what are my tasks?"

**Confidence Weight**: 0.9

**Examples**:
```
✓ "show me open issues"
✓ "what are my tasks?"
✓ "list all bugs in the project"
✓ "check active tickets in youtrack"
```

### 3. query_analytics
**Description**: User wants statistics, metrics, trends, or analytics

**Keywords**: statistics, stats, metrics, analytics, report, trend, trends, performance, count, total

**Regex Patterns**:
- `(?i)\b(how many|count|total|number of)\b` - "how many", "total count"
- `(?i)\b(statistics|stats|metrics|analytics)\b` - direct analytics keywords
- `(?i)\b(trend|pattern|over time)\b` - trend analysis

**Confidence Weight**: 0.85

**Examples**:
```
✓ "show me statistics for this project"
✓ "how many items were processed last week?"
✓ "what are the trends over time?"
✓ "get performance metrics"
```

### 4. query_status
**Description**: User wants to check system status, health, or progress

**Keywords**: status, health, progress, state, condition, running, stopped, failed, ok, ready

**Regex Patterns**:
- `(?i)\bwhat('s| is)?\s+the\s+status\b` - "what's the status?"
- `(?i)\bis\s+(it|everything|system)\s+(ok|running|working|healthy)\b` - "is it running?"
- `(?i)\bcheck\s+status\b` - "check status"

**Confidence Weight**: 0.85

**Examples**:
```
✓ "what's the status of the system?"
✓ "is everything running ok?"
✓ "check status of the deployment"
✓ "show me the health check"
```

### 5. command_action
**Description**: User wants to execute an action or command

**Keywords**: deploy, restart, start, stop, update, upgrade, create, delete, modify, execute, run

**Regex Patterns**:
- `(?i)\b(please|can you|could you)?\s*(deploy|restart|start|stop|update)\b` - polite commands
- `(?i)\b(create|delete|modify|execute|run)\s+` - direct actions

**Confidence Weight**: 0.9

**Examples**:
```
✓ "please deploy the latest version"
✓ "restart the server"
✓ "create a new issue for this bug"
✓ "execute the test suite"
```

### 6. request_help
**Description**: User needs help, documentation, or explanation

**Keywords**: help, how, what, explain, documentation, guide, tutorial, instructions, support

**Regex Patterns**:
- `(?i)\b(help|how do|how to|can you help)\b` - help requests
- `(?i)\b(what is|what are|explain|tell me about)\b` - explanation requests
- `(?i)\b(documentation|guide|tutorial|instructions)\b` - documentation requests

**Confidence Weight**: 0.8

**Examples**:
```
✓ "how do I configure the application?"
✓ "what is MCP?"
✓ "can you help me with this error?"
✓ "show me the documentation"
```

### 7. conversation
**Description**: General conversation, greetings, or unclear intent (fallback)

**Keywords**: hello, hi, thanks, thank you, bye, goodbye

**Regex Patterns**:
- `(?i)\b(hello|hi|hey|greetings)\b` - greetings
- `(?i)\b(thanks|thank you|bye|goodbye)\b` - pleasantries

**Confidence Weight**: 0.6

**Examples**:
```
✓ "hello there"
✓ "thanks!"
✓ "bye, see you later"
```

**Note**: This is the default fallback intent when no other intents match strongly.

## Confidence Scoring

### Confidence Calculation Algorithm

The agent uses a weighted scoring system that combines keyword matches and regex pattern matches:

```go
score = (keyword_matches * 0.3) + (pattern_matches * 0.5)
score = score * intent_weight
score = min(score, 1.0)  // Normalize to 0.0-1.0 range
```

**Components**:
1. **Keyword Matching** (weight: 0.3 per keyword)
   - Each matched keyword adds 0.3 to the score
   - Case-insensitive matching
   - Multiple keyword matches accumulate

2. **Pattern Matching** (weight: 0.5 per pattern)
   - Each matched regex pattern adds 0.5 to the score
   - More specific patterns (longer, more complex) are more valuable
   - Multiple pattern matches accumulate

3. **Intent Weight** (0.6-0.9)
   - Applied to final score based on intent type
   - Higher weight for more confident intents
   - `command_action`, `query_commits`, `query_issues`: 0.9
   - `query_analytics`, `query_status`: 0.85
   - `request_help`: 0.8
   - `conversation`: 0.6

### Confidence Thresholds

| Confidence Range | Interpretation | Action |
|------------------|----------------|--------|
| **0.8 - 1.0** | High confidence | Use rule-based result directly |
| **0.5 - 0.8** | Medium confidence | Use rule-based result, log for review |
| **0.0 - 0.5** | Low confidence | Consider LLM fallback (future) |

**Current Behavior**:
- All intents are processed rule-based only (no LLM fallback implemented yet)
- Low confidence intents are still returned but marked with lower scores
- Multiple intents can be detected simultaneously (sorted by confidence)

### Confidence Scores in Output

The agent outputs confidence scores in `reasoning.confidence_scores`:

```json
{
  "primary_intent": 0.9,      // Confidence of top intent
  "overall": 0.85             // Average of top 2 intents (or top 1 if only one)
}
```

**Typical Confidence Ranges by Intent Type**:
- `query_commits`: 0.7-0.9 (explicit keywords like "commits", "changes")
- `query_issues`: 0.7-0.9 (explicit keywords like "issues", "tasks", "bugs")
- `query_analytics`: 0.6-0.85 (matches on "how many", "statistics")
- `query_status`: 0.6-0.85 (matches on "status", "health")
- `command_action`: 0.7-0.9 (action verbs are strong signals)
- `request_help`: 0.6-0.8 (overlaps with other intents)
- `conversation`: 0.3-0.6 (fallback with weak signals)

## Entity Extraction

The agent extracts 4 types of structured entities from input text:

### 1. Projects
**Pattern**: Project names with common prefixes or GitLab/YouTrack patterns

**Extraction Methods**:
- Explicit patterns: `project: <name>`, `repo: <name>`, `repository: <name>`
- GitLab/YouTrack patterns: any word containing "gitlab", "youtrack", "mcp"

**Examples**:
```
Input: "show commits from project gitlab-mcp"
Output: entities.projects = ["gitlab-mcp"]

Input: "compare project foo-service and bar-service"
Output: entities.projects = ["foo-service", "bar-service"]
```

### 2. Dates
**Types**: Relative dates and absolute dates

**Relative Date Keywords**:
- today, yesterday, tomorrow
- last week, this week, next week
- last month, this month, next month
- last year, this year

**Absolute Date Patterns**:
- ISO format: `2024-01-15`, `2024-12-31`
- US format: `01/15/2024`, `12/31/2024`
- Month-day-year: `Jan 15, 2024`, `December 31, 2024`

**Examples**:
```
Input: "commits from last week"
Output: entities.dates = ["last week"]

Input: "issues created on 2024-01-15"
Output: entities.dates = ["2024-01-15"]

Input: "statistics for this month"
Output: entities.dates = ["this month"]
```

### 3. Providers
**Supported Providers**: gitlab, youtrack, openai, anthropic, deepseek, ollama, github, jira, confluence

**Extraction Method**: Case-insensitive keyword matching

**Examples**:
```
Input: "check open issues in gitlab"
Output: entities.providers = ["gitlab"]

Input: "compare gitlab and youtrack"
Output: entities.providers = ["gitlab", "youtrack"]
```

### 4. Statuses
**Supported Statuses**: open, closed, in-progress, in progress, pending, resolved, done, failed, error, success, active, inactive, blocked, ready, draft

**Extraction Method**: Case-insensitive keyword matching

**Examples**:
```
Input: "show open issues"
Output: entities.statuses = ["open"]

Input: "list failed and blocked tasks"
Output: entities.statuses = ["failed", "blocked"]
```

## Output Schema

The agent writes to three main keys in the `AgentContext`:

### 1. reasoning.intents[]
Array of detected intents, sorted by confidence (highest first)

```go
type Intent struct {
    Type       string   `json:"type"`        // Intent type (e.g., "query_commits")
    Confidence float64  `json:"confidence"`  // Confidence score (0.0-1.0)
    Entities   []string `json:"entities,omitempty"` // (unused currently)
}
```

**Example**:
```json
{
  "intents": [
    {
      "type": "query_commits",
      "confidence": 0.9
    },
    {
      "type": "query_analytics",
      "confidence": 0.7
    }
  ]
}
```

### 2. reasoning.entities{}
Map of entity types to extracted values

```go
entities := map[string]interface{}{
    "projects":  []string{"gitlab-mcp", "youtrack-api"},
    "dates":     []string{"last week", "2024-01-15"},
    "providers": []string{"gitlab"},
    "statuses":  []string{"open", "closed"}
}
```

**Example**:
```json
{
  "entities": {
    "projects": ["gitlab-mcp"],
    "dates": ["last week"],
    "providers": ["gitlab"],
    "statuses": ["open"]
  }
}
```

### 3. reasoning.confidence_scores{}
Map of confidence score types to values

```go
scores := map[string]float64{
    "primary_intent": 0.9,  // Confidence of primary intent
    "overall": 0.85        // Average confidence
}
```

**Example**:
```json
{
  "confidence_scores": {
    "primary_intent": 0.9,
    "overall": 0.85
  }
}
```

## Agent Contract

### Preconditions
**None** - This is the first agent in the pipeline

### Postconditions
- `reasoning.intents` - At least one intent detected
- `reasoning.entities` - Entity map (may be empty)
- `reasoning.confidence_scores` - Confidence scores

### Agent Metadata
```go
{
  ID:          "intent_detection",
  Name:        "Intent Detection Agent",
  Description: "Detects user intent and extracts entities using rule-based classification",
  Version:     "1.0.0",
  Author:      "ADK LLM Proxy",
  Tags:        ["intent", "classification", "entity-extraction", "nlp"],
  Dependencies: []  // No dependencies
}
```

### Agent Capabilities
```go
{
  SupportsParallelExecution: false,  // Must run first
  SupportsRetry:             true,   // Idempotent, can retry
  RequiresLLM:               false,  // Rule-based only
  IsDeterministic:           true,   // Same input = same output
  EstimatedDuration:         50      // ~50ms execution time
}
```

## Usage Examples

### Example 1: Commit Query
**Input**: "show me the latest commits from gitlab-mcp project last week"

**Output**:
```json
{
  "reasoning": {
    "intents": [
      {
        "type": "query_commits",
        "confidence": 0.9
      }
    ],
    "entities": {
      "projects": ["gitlab-mcp"],
      "dates": ["last week"]
    },
    "confidence_scores": {
      "primary_intent": 0.9,
      "overall": 0.9
    }
  }
}
```

### Example 2: Multi-Intent Query
**Input**: "show me recent commits and their statistics"

**Output**:
```json
{
  "reasoning": {
    "intents": [
      {
        "type": "query_commits",
        "confidence": 0.9
      },
      {
        "type": "query_analytics",
        "confidence": 0.7
      }
    ],
    "entities": {},
    "confidence_scores": {
      "primary_intent": 0.9,
      "overall": 0.8
    }
  }
}
```

### Example 3: Issue Query with Provider
**Input**: "check open issues in gitlab"

**Output**:
```json
{
  "reasoning": {
    "intents": [
      {
        "type": "query_issues",
        "confidence": 0.9
      }
    ],
    "entities": {
      "providers": ["gitlab"],
      "statuses": ["open"]
    },
    "confidence_scores": {
      "primary_intent": 0.9,
      "overall": 0.9
    }
  }
}
```

### Example 4: Low Confidence (Conversation)
**Input**: "hello there"

**Output**:
```json
{
  "reasoning": {
    "intents": [
      {
        "type": "conversation",
        "confidence": 0.6
      }
    ],
    "entities": {},
    "confidence_scores": {
      "primary_intent": 0.6,
      "overall": 0.6
    }
  }
}
```

## Performance Characteristics

| Metric | Value |
|--------|-------|
| **Execution Time** | <50ms (typical) |
| **LLM Calls** | 0 (rule-based only) |
| **Memory Usage** | <1MB |
| **Determinism** | 100% (same input = same output) |
| **Parallelization** | Not supported (must run first) |
| **Retry Safety** | Yes (idempotent) |

## Future Enhancements

### Planned Improvements

1. **Ollama Local Model Integration** (Task: line 840)
   - Add lightweight local LLM for intent detection
   - Use for ambiguous cases or to improve confidence
   - Fallback to rule-based if Ollama unavailable

2. **Cloud LLM Fallback** (Task: line 841)
   - Use cloud LLM for low-confidence cases (< 0.8)
   - Improve intent classification accuracy
   - Cost-aware selection (DeepSeek → GPT-4o-mini)

3. **Clarification Questions** (Task: line 844)
   - Generate clarification questions for ambiguous intents
   - Example: "Did you mean commits or issues?"
   - Interactive disambiguation flow

4. **Enhanced Entity Extraction**
   - Named Entity Recognition (NER) using ML models
   - Temporal expression parsing (relative dates → absolute dates)
   - Entity linking to knowledge base

5. **Multi-Language Support**
   - Support non-English inputs
   - Language detection
   - Localized intent patterns

## Testing

### Test Coverage
- **Unit Tests**: 20+ test functions
- **Test File**: `src/golang/internal/domain/services/agents/intent_detection_test.go`
- **Coverage**: 100% of public methods

### Test Categories
1. **Intent Detection Tests**: Each intent type with multiple examples
2. **Entity Extraction Tests**: All entity types with various patterns
3. **Confidence Scoring Tests**: High, medium, low confidence scenarios
4. **Multi-Intent Tests**: Detecting multiple intents in one query
5. **Edge Cases**: Empty input, ambiguous input, long input
6. **Audit Trail Tests**: Verification of agent run tracking
7. **Idempotency Tests**: Same input produces same output
8. **Context Isolation Tests**: Input context unchanged after execution
9. **Real-World Scenarios**: Complex multi-entity queries

### Running Tests
```bash
cd src/golang
go test -v github.com/mshogin/agents/internal/domain/services/agents
go test -cover github.com/mshogin/agents/internal/domain/services/agents
```

## Troubleshooting

### Common Issues

**Issue**: Intent not detected
- **Cause**: Input doesn't match any patterns strongly
- **Solution**: Check if keywords match intent type, add more context to input
- **Example**: Instead of "show me", use "show me commits"

**Issue**: Wrong intent detected
- **Cause**: Keyword overlap between intents
- **Solution**: Use more specific keywords for the desired intent
- **Example**: "how many commits" detects query_commits instead of query_analytics
  - Use "show me statistics" instead

**Issue**: Low confidence scores
- **Cause**: Weak or ambiguous input
- **Solution**: Provide more explicit keywords or patterns
- **Example**: Instead of "what's happening?", use "what's the status of the system?"

**Issue**: Entities not extracted
- **Cause**: Entity pattern not recognized
- **Solution**: Use explicit format like "project: <name>" or known keywords
- **Example**: "show commits from gitlab-mcp" extracts "gitlab-mcp" as project

### Debug Mode

To debug intent detection:
1. Check `audit.agent_runs` for execution details
2. Examine `reasoning.confidence_scores` for score breakdown
3. Review `reasoning.intents` for all detected intents (not just primary)
4. Inspect `diagnostics.performance` for timing information

## Related Documentation

- [Agent Context Schema](../agent_context_schema.md) - AgentContext structure
- [Agent Contracts](../agent_contracts.md) - Agent interface specifications
- [LLM Selection Policies](../llm_selection_policies.md) - Model selection for LLM fallback
- [Pipeline Configuration](../pipeline_configuration.md) - Agent pipeline setup

## Changelog

### Version 1.0.0 (2025-01-15)
- Initial implementation with rule-based classification
- 7 intent types supported
- 4 entity types extracted
- 20+ unit tests with 100% pass rate
- Confidence scoring algorithm
- Complete audit trail and performance metrics
