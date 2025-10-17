package single_agent

import (
	"fmt"
	"time"

	"github.com/mshogin/agents/internal/domain/models"
)

// MockContextBuilder provides a fluent API for building test AgentContext objects
type MockContextBuilder struct {
	ctx *models.AgentContext
}

// NewMockContextBuilder creates a new builder with default session and trace IDs
func NewMockContextBuilder() *MockContextBuilder {
	now := time.Now()
	sessionID := fmt.Sprintf("test-session-%d", now.Unix())
	traceID := fmt.Sprintf("test-trace-%d", now.UnixNano())

	return &MockContextBuilder{
		ctx: models.NewAgentContext(sessionID, traceID),
	}
}

// WithSessionID sets a custom session ID
func (b *MockContextBuilder) WithSessionID(sessionID string) *MockContextBuilder {
	b.ctx.Metadata.SessionID = sessionID
	return b
}

// WithTraceID sets a custom trace ID
func (b *MockContextBuilder) WithTraceID(traceID string) *MockContextBuilder {
	b.ctx.Metadata.TraceID = traceID
	return b
}

// WithIntents adds intents to the context
func (b *MockContextBuilder) WithIntents(intents ...models.Intent) *MockContextBuilder {
	b.ctx.Reasoning.Intents = append(b.ctx.Reasoning.Intents, intents...)
	return b
}

// WithEntities adds entities to the context
func (b *MockContextBuilder) WithEntities(entities map[string]interface{}) *MockContextBuilder {
	if b.ctx.Reasoning.Entities == nil {
		b.ctx.Reasoning.Entities = make(map[string]interface{})
	}
	for k, v := range entities {
		b.ctx.Reasoning.Entities[k] = v
	}
	return b
}

// WithHypotheses adds hypotheses to the context
func (b *MockContextBuilder) WithHypotheses(hypotheses ...models.Hypothesis) *MockContextBuilder {
	b.ctx.Reasoning.Hypotheses = append(b.ctx.Reasoning.Hypotheses, hypotheses...)
	return b
}

// WithPlans adds retrieval plans to the context
func (b *MockContextBuilder) WithPlans(plans ...models.RetrievalPlan) *MockContextBuilder {
	b.ctx.Retrieval.Plans = append(b.ctx.Retrieval.Plans, plans...)
	return b
}

// WithQueries adds normalized queries to the context
func (b *MockContextBuilder) WithQueries(queries ...models.NormalizedQuery) *MockContextBuilder {
	b.ctx.Retrieval.Queries = append(b.ctx.Retrieval.Queries, queries...)
	return b
}

// WithArtifacts adds artifacts to the context
func (b *MockContextBuilder) WithArtifacts(artifacts ...models.Artifact) *MockContextBuilder {
	b.ctx.Retrieval.Artifacts = append(b.ctx.Retrieval.Artifacts, artifacts...)
	return b
}

// WithFacts adds facts to the context
func (b *MockContextBuilder) WithFacts(facts ...models.Fact) *MockContextBuilder {
	b.ctx.Enrichment.Facts = append(b.ctx.Enrichment.Facts, facts...)
	return b
}

// WithDerivedKnowledge adds derived knowledge to the context
func (b *MockContextBuilder) WithDerivedKnowledge(knowledge ...models.DerivedKnowledge) *MockContextBuilder {
	b.ctx.Enrichment.DerivedKnowledge = append(b.ctx.Enrichment.DerivedKnowledge, knowledge...)
	return b
}

// WithRelationships adds relationships to the context
func (b *MockContextBuilder) WithRelationships(relationships ...models.Relationship) *MockContextBuilder {
	b.ctx.Enrichment.Relationships = append(b.ctx.Enrichment.Relationships, relationships...)
	return b
}

// WithConclusions adds conclusions to the context
func (b *MockContextBuilder) WithConclusions(conclusions ...models.Conclusion) *MockContextBuilder {
	b.ctx.Reasoning.Conclusions = append(b.ctx.Reasoning.Conclusions, conclusions...)
	return b
}

// WithSummary sets the reasoning summary
func (b *MockContextBuilder) WithSummary(summary string) *MockContextBuilder {
	b.ctx.Reasoning.Summary = summary
	return b
}

// WithFinalResponse sets the final response
func (b *MockContextBuilder) WithFinalResponse(response string) *MockContextBuilder {
	b.ctx.Reasoning.FinalResponse = response
	return b
}

// WithErrors adds errors to the diagnostics
func (b *MockContextBuilder) WithErrors(errors ...models.DiagnosticError) *MockContextBuilder {
	if b.ctx.Diagnostics == nil {
		b.ctx.Diagnostics = &models.DiagnosticsContext{}
	}
	b.ctx.Diagnostics.Errors = append(b.ctx.Diagnostics.Errors, errors...)
	return b
}

// WithWarnings adds warnings to the diagnostics
func (b *MockContextBuilder) WithWarnings(warnings ...models.DiagnosticWarning) *MockContextBuilder {
	if b.ctx.Diagnostics == nil {
		b.ctx.Diagnostics = &models.DiagnosticsContext{}
	}
	b.ctx.Diagnostics.Warnings = append(b.ctx.Diagnostics.Warnings, warnings...)
	return b
}

// WithValidationReports adds validation reports to the diagnostics
func (b *MockContextBuilder) WithValidationReports(reports ...models.ValidationReport) *MockContextBuilder {
	if b.ctx.Diagnostics == nil {
		b.ctx.Diagnostics = &models.DiagnosticsContext{}
	}
	b.ctx.Diagnostics.ValidationReports = append(b.ctx.Diagnostics.ValidationReports, reports...)
	return b
}

// WithUserInput stores the original user input in LLM cache
func (b *MockContextBuilder) WithUserInput(input string) *MockContextBuilder {
	if b.ctx.LLM == nil {
		b.ctx.LLM = &models.LLMContext{
			Cache: make(map[string]interface{}),
		}
	}
	if b.ctx.LLM.Cache == nil {
		b.ctx.LLM.Cache = make(map[string]interface{})
	}
	b.ctx.LLM.Cache["original_user_input"] = input
	return b
}

// Build returns the constructed AgentContext
func (b *MockContextBuilder) Build() *models.AgentContext {
	return b.ctx
}
