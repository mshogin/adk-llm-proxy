# AgentContext Schema Documentation

## Overview

The `AgentContext` is a versioned, namespaced data structure that serves as the primary communication mechanism between agents in the reasoning pipeline. It provides isolated namespaces for different types of data, comprehensive audit trails, and validation mechanisms.

## Architecture Principles

1. **Namespace Isolation**: Each agent writes only to designated namespaces to prevent conflicts
2. **Immutable History**: All changes are tracked via diffs for complete traceability
3. **Version Control**: Schema versioning enables backward compatibility and migrations
4. **Validation**: Contract validation ensures agents fulfill their promises
5. **Performance Tracking**: Built-in performance and cost metrics

## Core Structure

```go
type AgentContext struct {
    Version   string             // Schema version (e.g., "1.0.0")
    Metadata  *MetadataContext   // Session and trace information
    Reasoning *ReasoningContext  // Intent, hypotheses, conclusions
    Enrichment *EnrichmentContext // Facts, knowledge, relationships
    Retrieval *RetrievalContext  // Retrieval plans and queries
    LLM       *LLMContext        // LLM usage and decisions
    Diagnostics *DiagnosticsContext // Errors, warnings, validation
    Audit     *AuditContext      // Agent execution history
}
```

## See Full Documentation

For the complete AgentContext schema documentation including all namespace definitions, examples, and best practices, see the source file:

`src/golang/internal/domain/models/agent_context.go`

## Key Namespaces

### 1. MetadataContext
Session identification and tracing information.

### 2. ReasoningContext
Intent detection, hypotheses generation, and conclusions.

### 3. EnrichmentContext
Facts, derived knowledge, and entity relationships.

### 4. RetrievalContext  
Retrieval plans, queries, and artifacts.

### 5. LLMContext
LLM usage tracking and cost management.

### 6. DiagnosticsContext
Errors, warnings, performance metrics, and validation reports.

### 7. AuditContext
Complete agent execution history and context diffs.

## References

- [Agent Contracts](./agent_contracts.md)
- [Pipeline Configuration](./pipeline_configuration.md)
- Source: `src/golang/internal/domain/models/agent_context.go`
