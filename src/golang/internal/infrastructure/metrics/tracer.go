package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Tracer provides distributed tracing capabilities for tracking
// requests across multiple services and agent executions.
//
// Design Principles:
// - Trace ID propagation through context
// - Span hierarchy for nested operations
// - Thread-safe span collection
// - OpenTelemetry-compatible span format
type Tracer struct {
	traceID string

	// Spans by span ID
	spans map[string]*Span
	mu    sync.RWMutex

	// Root span
	rootSpan *Span

	// Session metadata
	sessionID string
	startTime time.Time
}

// Span represents a single unit of work within a trace.
type Span struct {
	SpanID     string
	TraceID    string
	ParentID   string // Parent span ID, empty for root span
	Name       string // Operation name (e.g., "agent.intent_detection")
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Status     SpanStatus
	Attributes map[string]interface{}
	Events     []SpanEvent
	Tags       map[string]string
}

// SpanStatus represents the completion status of a span.
type SpanStatus string

const (
	SpanStatusUnset SpanStatus = "unset" // Not set
	SpanStatusOK    SpanStatus = "ok"    // Success
	SpanStatusError SpanStatus = "error" // Error occurred
)

// SpanEvent represents a significant point in time within a span.
type SpanEvent struct {
	Timestamp  time.Time
	Name       string
	Attributes map[string]interface{}
}

// Trace represents a complete trace with all spans.
type Trace struct {
	TraceID   string
	SessionID string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	RootSpan  *Span
	Spans     []*Span
	SpanCount int
}

// NewTracer creates a new tracer for a session.
func NewTracer(sessionID, traceID string) *Tracer {
	return &Tracer{
		traceID:   traceID,
		sessionID: sessionID,
		spans:     make(map[string]*Span),
		startTime: time.Now(),
	}
}

// StartSpan creates and starts a new span.
func (t *Tracer) StartSpan(name string, parentSpanID string) *Span {
	t.mu.Lock()
	defer t.mu.Unlock()

	spanID := fmt.Sprintf("%s-%d", t.traceID, len(t.spans)+1)

	span := &Span{
		SpanID:     spanID,
		TraceID:    t.traceID,
		ParentID:   parentSpanID,
		Name:       name,
		StartTime:  time.Now(),
		Status:     SpanStatusUnset,
		Attributes: make(map[string]interface{}),
		Events:     []SpanEvent{},
		Tags:       make(map[string]string),
	}

	t.spans[spanID] = span

	// Set root span if not set
	if t.rootSpan == nil && parentSpanID == "" {
		t.rootSpan = span
	}

	return span
}

// EndSpan marks a span as complete.
func (t *Tracer) EndSpan(spanID string, status SpanStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = status
}

// AddSpanAttribute adds an attribute to a span.
func (t *Tracer) AddSpanAttribute(spanID string, key string, value interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	span.Attributes[key] = value
}

// AddSpanEvent adds an event to a span.
func (t *Tracer) AddSpanEvent(spanID string, name string, attributes map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	event := SpanEvent{
		Timestamp:  time.Now(),
		Name:       name,
		Attributes: attributes,
	}

	span.Events = append(span.Events, event)
}

// AddSpanTag adds a tag to a span.
func (t *Tracer) AddSpanTag(spanID string, key string, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	span, exists := t.spans[spanID]
	if !exists {
		return
	}

	span.Tags[key] = value
}

// GetSpan retrieves a span by ID.
func (t *Tracer) GetSpan(spanID string) *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	span, exists := t.spans[spanID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	copy := *span

	// Deep copy attributes
	copy.Attributes = make(map[string]interface{}, len(span.Attributes))
	for k, v := range span.Attributes {
		copy.Attributes[k] = v
	}

	// Deep copy tags
	copy.Tags = make(map[string]string, len(span.Tags))
	for k, v := range span.Tags {
		copy.Tags[k] = v
	}

	// Deep copy events
	copy.Events = make([]SpanEvent, len(span.Events))
	for i, event := range span.Events {
		copy.Events[i] = event
		// Deep copy event attributes
		copy.Events[i].Attributes = make(map[string]interface{}, len(event.Attributes))
		for k, v := range event.Attributes {
			copy.Events[i].Attributes[k] = v
		}
	}

	return &copy
}

// GetAllSpans returns all spans in the trace.
func (t *Tracer) GetAllSpans() []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*Span, 0, len(t.spans))
	for _, span := range t.spans {
		copy := t.copySpanUnsafe(span)
		spans = append(spans, &copy)
	}

	return spans
}

// GetTrace returns the complete trace with all spans.
func (t *Tracer) GetTrace() *Trace {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*Span, 0, len(t.spans))
	var minStartTime, maxEndTime time.Time

	for _, span := range t.spans {
		copy := t.copySpanUnsafe(span)
		spans = append(spans, &copy)

		if minStartTime.IsZero() || span.StartTime.Before(minStartTime) {
			minStartTime = span.StartTime
		}

		if !span.EndTime.IsZero() && (maxEndTime.IsZero() || span.EndTime.After(maxEndTime)) {
			maxEndTime = span.EndTime
		}
	}

	duration := time.Duration(0)
	if !maxEndTime.IsZero() {
		duration = maxEndTime.Sub(minStartTime)
	}

	var rootSpanCopy *Span
	if t.rootSpan != nil {
		copy := t.copySpanUnsafe(t.rootSpan)
		rootSpanCopy = &copy
	}

	return &Trace{
		TraceID:   t.traceID,
		SessionID: t.sessionID,
		StartTime: minStartTime,
		EndTime:   maxEndTime,
		Duration:  duration,
		RootSpan:  rootSpanCopy,
		Spans:     spans,
		SpanCount: len(spans),
	}
}

// copySpanUnsafe creates a deep copy of a span without locking (internal use only).
func (t *Tracer) copySpanUnsafe(span *Span) Span {
	copy := *span

	// Deep copy attributes
	copy.Attributes = make(map[string]interface{}, len(span.Attributes))
	for k, v := range span.Attributes {
		copy.Attributes[k] = v
	}

	// Deep copy tags
	copy.Tags = make(map[string]string, len(span.Tags))
	for k, v := range span.Tags {
		copy.Tags[k] = v
	}

	// Deep copy events
	copy.Events = make([]SpanEvent, len(span.Events))
	for i, event := range span.Events {
		copy.Events[i] = event
		// Deep copy event attributes
		copy.Events[i].Attributes = make(map[string]interface{}, len(event.Attributes))
		for k, v := range event.Attributes {
			copy.Events[i].Attributes[k] = v
		}
	}

	return copy
}

// Reset resets the tracer (useful for testing).
func (t *Tracer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.spans = make(map[string]*Span)
	t.rootSpan = nil
	t.startTime = time.Now()
}

// TraceIDFromContext extracts trace ID from context.
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}

// SpanIDFromContext extracts span ID from context.
func SpanIDFromContext(ctx context.Context) string {
	if spanID, ok := ctx.Value("span_id").(string); ok {
		return spanID
	}
	return ""
}

// ContextWithTraceID adds trace ID to context.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, "trace_id", traceID)
}

// ContextWithSpanID adds span ID to context.
func ContextWithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, "span_id", spanID)
}

// ContextWithTrace adds trace and span IDs to context.
func ContextWithTrace(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, "trace_id", traceID)
	ctx = context.WithValue(ctx, "span_id", spanID)
	return ctx
}
