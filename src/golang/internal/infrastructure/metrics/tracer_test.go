package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracer(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	assert.Equal(t, "trace456", tracer.traceID)
	assert.Equal(t, "session123", tracer.sessionID)
	assert.NotNil(t, tracer.spans)
	assert.Nil(t, tracer.rootSpan)
}

func TestTracer_StartSpan_SingleSpan(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.intent_detection", "")

	assert.NotEmpty(t, span.SpanID)
	assert.Equal(t, "trace456", span.TraceID)
	assert.Empty(t, span.ParentID) // Root span
	assert.Equal(t, "agent.intent_detection", span.Name)
	assert.Equal(t, SpanStatusUnset, span.Status)
	assert.NotZero(t, span.StartTime)
	assert.Zero(t, span.EndTime)
	assert.NotNil(t, span.Attributes)
	assert.NotNil(t, span.Tags)
}

func TestTracer_StartSpan_NestedSpans(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	rootSpan := tracer.StartSpan("agent.pipeline", "")
	childSpan1 := tracer.StartSpan("agent.intent_detection", rootSpan.SpanID)
	childSpan2 := tracer.StartSpan("agent.inference", rootSpan.SpanID)

	assert.Equal(t, "", rootSpan.ParentID)
	assert.Equal(t, rootSpan.SpanID, childSpan1.ParentID)
	assert.Equal(t, rootSpan.SpanID, childSpan2.ParentID)
	assert.NotEqual(t, childSpan1.SpanID, childSpan2.SpanID)
}

func TestTracer_EndSpan(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.intent_detection", "")
	startTime := span.StartTime

	time.Sleep(10 * time.Millisecond)

	tracer.EndSpan(span.SpanID, SpanStatusOK)

	retrievedSpan := tracer.GetSpan(span.SpanID)
	assert.Equal(t, SpanStatusOK, retrievedSpan.Status)
	assert.NotZero(t, retrievedSpan.EndTime)
	assert.Greater(t, retrievedSpan.EndTime, startTime)
	assert.Greater(t, retrievedSpan.Duration, time.Duration(0))
	assert.GreaterOrEqual(t, retrievedSpan.Duration.Milliseconds(), int64(10))
}

func TestTracer_EndSpan_WithError(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.inference", "")
	tracer.EndSpan(span.SpanID, SpanStatusError)

	retrievedSpan := tracer.GetSpan(span.SpanID)
	assert.Equal(t, SpanStatusError, retrievedSpan.Status)
}

func TestTracer_AddSpanAttribute(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.intent_detection", "")
	tracer.AddSpanAttribute(span.SpanID, "agent_id", "intent_detection")
	tracer.AddSpanAttribute(span.SpanID, "execution_count", 1)
	tracer.AddSpanAttribute(span.SpanID, "confidence", 0.95)

	retrievedSpan := tracer.GetSpan(span.SpanID)
	assert.Equal(t, "intent_detection", retrievedSpan.Attributes["agent_id"])
	assert.Equal(t, 1, retrievedSpan.Attributes["execution_count"])
	assert.Equal(t, 0.95, retrievedSpan.Attributes["confidence"])
}

func TestTracer_AddSpanEvent(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.inference", "")
	tracer.AddSpanEvent(span.SpanID, "conclusion_generated", map[string]interface{}{
		"conclusion_id": "c1",
		"confidence":    0.90,
	})

	retrievedSpan := tracer.GetSpan(span.SpanID)
	require.Len(t, retrievedSpan.Events, 1)

	event := retrievedSpan.Events[0]
	assert.Equal(t, "conclusion_generated", event.Name)
	assert.Equal(t, "c1", event.Attributes["conclusion_id"])
	assert.Equal(t, 0.90, event.Attributes["confidence"])
	assert.NotZero(t, event.Timestamp)
}

func TestTracer_AddSpanTag(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.inference", "")
	tracer.AddSpanTag(span.SpanID, "environment", "production")
	tracer.AddSpanTag(span.SpanID, "version", "1.0.0")

	retrievedSpan := tracer.GetSpan(span.SpanID)
	assert.Equal(t, "production", retrievedSpan.Tags["environment"])
	assert.Equal(t, "1.0.0", retrievedSpan.Tags["version"])
}

func TestTracer_GetSpan_NonExistent(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.GetSpan("nonexistent")

	assert.Nil(t, span)
}

func TestTracer_GetAllSpans(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span1 := tracer.StartSpan("agent.intent_detection", "")
	span2 := tracer.StartSpan("agent.inference", "")
	span3 := tracer.StartSpan("agent.validation", "")

	tracer.EndSpan(span1.SpanID, SpanStatusOK)
	tracer.EndSpan(span2.SpanID, SpanStatusOK)
	tracer.EndSpan(span3.SpanID, SpanStatusError)

	spans := tracer.GetAllSpans()

	assert.Len(t, spans, 3)

	// Verify spans are present
	spanIDs := make(map[string]bool)
	for _, span := range spans {
		spanIDs[span.SpanID] = true
	}

	assert.True(t, spanIDs[span1.SpanID])
	assert.True(t, spanIDs[span2.SpanID])
	assert.True(t, spanIDs[span3.SpanID])
}

func TestTracer_GetTrace_Empty(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	trace := tracer.GetTrace()

	assert.Equal(t, "trace456", trace.TraceID)
	assert.Equal(t, "session123", trace.SessionID)
	assert.Equal(t, 0, trace.SpanCount)
	assert.Nil(t, trace.RootSpan)
	assert.Empty(t, trace.Spans)
}

func TestTracer_GetTrace_WithSpans(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	rootSpan := tracer.StartSpan("agent.pipeline", "")
	childSpan1 := tracer.StartSpan("agent.intent_detection", rootSpan.SpanID)
	childSpan2 := tracer.StartSpan("agent.inference", rootSpan.SpanID)

	time.Sleep(10 * time.Millisecond)

	tracer.EndSpan(childSpan1.SpanID, SpanStatusOK)
	tracer.EndSpan(childSpan2.SpanID, SpanStatusOK)
	tracer.EndSpan(rootSpan.SpanID, SpanStatusOK)

	trace := tracer.GetTrace()

	assert.Equal(t, "trace456", trace.TraceID)
	assert.Equal(t, "session123", trace.SessionID)
	assert.Equal(t, 3, trace.SpanCount)
	assert.NotNil(t, trace.RootSpan)
	assert.Equal(t, "agent.pipeline", trace.RootSpan.Name)
	assert.Len(t, trace.Spans, 3)
	assert.Greater(t, trace.Duration, time.Duration(0))
	assert.GreaterOrEqual(t, trace.Duration.Milliseconds(), int64(10))
}

func TestTracer_RootSpan(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	// First span with empty parent becomes root
	span1 := tracer.StartSpan("agent.pipeline", "")
	assert.Equal(t, span1, tracer.rootSpan)

	// Second span with empty parent does not replace root
	span2 := tracer.StartSpan("agent.another_root", "")
	assert.Equal(t, span1, tracer.rootSpan)
	assert.NotEqual(t, span2, tracer.rootSpan)
}

func TestTracer_Reset(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.intent_detection", "")
	tracer.AddSpanAttribute(span.SpanID, "agent_id", "intent_detection")

	assert.NotNil(t, tracer.GetSpan(span.SpanID))
	assert.NotNil(t, tracer.rootSpan)

	tracer.Reset()

	assert.Nil(t, tracer.GetSpan(span.SpanID))
	assert.Nil(t, tracer.rootSpan)
	assert.Empty(t, tracer.spans)
}

func TestTracer_ThreadSafety(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	done := make(chan bool)
	numGoroutines := 10
	spansPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < spansPerGoroutine; j++ {
				span := tracer.StartSpan("agent.test", "")
				tracer.AddSpanAttribute(span.SpanID, "goroutine_id", id)
				tracer.AddSpanTag(span.SpanID, "test", "true")
				tracer.AddSpanEvent(span.SpanID, "test_event", map[string]interface{}{
					"iteration": j,
				})
				time.Sleep(1 * time.Millisecond)
				tracer.EndSpan(span.SpanID, SpanStatusOK)
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	spans := tracer.GetAllSpans()
	assert.Equal(t, numGoroutines*spansPerGoroutine, len(spans))

	// Verify all spans have attributes, tags, and events
	for _, span := range spans {
		assert.NotEmpty(t, span.Attributes)
		assert.NotEmpty(t, span.Tags)
		assert.NotEmpty(t, span.Events)
		assert.Equal(t, SpanStatusOK, span.Status)
	}
}

func TestTracer_GetSpan_Copy(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.intent_detection", "")
	tracer.AddSpanAttribute(span.SpanID, "agent_id", "intent_detection")

	span1 := tracer.GetSpan(span.SpanID)
	span1.Attributes["agent_id"] = "modified"

	span2 := tracer.GetSpan(span.SpanID)
	assert.Equal(t, "intent_detection", span2.Attributes["agent_id"])
}

func TestTracer_GetAllSpans_Copy(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	span := tracer.StartSpan("agent.intent_detection", "")
	tracer.AddSpanAttribute(span.SpanID, "agent_id", "intent_detection")

	spans1 := tracer.GetAllSpans()
	spans1[0].Attributes["agent_id"] = "modified"

	spans2 := tracer.GetAllSpans()
	assert.Equal(t, "intent_detection", spans2[0].Attributes["agent_id"])
}

func TestContextHelpers_TraceID(t *testing.T) {
	ctx := context.Background()

	// No trace ID initially
	traceID := TraceIDFromContext(ctx)
	assert.Empty(t, traceID)

	// Add trace ID
	ctx = ContextWithTraceID(ctx, "trace123")
	traceID = TraceIDFromContext(ctx)
	assert.Equal(t, "trace123", traceID)
}

func TestContextHelpers_SpanID(t *testing.T) {
	ctx := context.Background()

	// No span ID initially
	spanID := SpanIDFromContext(ctx)
	assert.Empty(t, spanID)

	// Add span ID
	ctx = ContextWithSpanID(ctx, "span456")
	spanID = SpanIDFromContext(ctx)
	assert.Equal(t, "span456", spanID)
}

func TestContextHelpers_ContextWithTrace(t *testing.T) {
	ctx := context.Background()

	ctx = ContextWithTrace(ctx, "trace123", "span456")

	traceID := TraceIDFromContext(ctx)
	spanID := SpanIDFromContext(ctx)

	assert.Equal(t, "trace123", traceID)
	assert.Equal(t, "span456", spanID)
}

func TestTracer_ComplexTrace(t *testing.T) {
	tracer := NewTracer("session123", "trace456")

	// Create a complex span hierarchy
	rootSpan := tracer.StartSpan("agent.pipeline", "")
	tracer.AddSpanTag(rootSpan.SpanID, "pipeline_type", "reasoning")

	intentSpan := tracer.StartSpan("agent.intent_detection", rootSpan.SpanID)
	tracer.AddSpanAttribute(intentSpan.SpanID, "intents_detected", 2)
	tracer.AddSpanEvent(intentSpan.SpanID, "intent_found", map[string]interface{}{
		"intent_type": "query_commits",
		"confidence":  0.95,
	})
	time.Sleep(5 * time.Millisecond)
	tracer.EndSpan(intentSpan.SpanID, SpanStatusOK)

	inferenceSpan := tracer.StartSpan("agent.inference", rootSpan.SpanID)
	tracer.AddSpanAttribute(inferenceSpan.SpanID, "conclusions_generated", 3)
	tracer.AddSpanEvent(inferenceSpan.SpanID, "conclusion_made", map[string]interface{}{
		"conclusion_id": "c1",
		"confidence":    0.90,
	})
	time.Sleep(5 * time.Millisecond)
	tracer.EndSpan(inferenceSpan.SpanID, SpanStatusOK)

	validationSpan := tracer.StartSpan("agent.validation", rootSpan.SpanID)
	tracer.AddSpanAttribute(validationSpan.SpanID, "errors_found", 0)
	tracer.EndSpan(validationSpan.SpanID, SpanStatusOK)

	time.Sleep(5 * time.Millisecond)
	tracer.EndSpan(rootSpan.SpanID, SpanStatusOK)

	// Verify trace
	trace := tracer.GetTrace()
	assert.Equal(t, 4, trace.SpanCount)
	assert.Equal(t, "agent.pipeline", trace.RootSpan.Name)
	assert.GreaterOrEqual(t, trace.Duration.Milliseconds(), int64(15))

	// Verify spans
	spans := tracer.GetAllSpans()
	assert.Len(t, spans, 4)

	// Verify intent span
	retrievedIntentSpan := tracer.GetSpan(intentSpan.SpanID)
	assert.Equal(t, 2, retrievedIntentSpan.Attributes["intents_detected"])
	assert.Len(t, retrievedIntentSpan.Events, 1)
	assert.Equal(t, "intent_found", retrievedIntentSpan.Events[0].Name)
	assert.Equal(t, SpanStatusOK, retrievedIntentSpan.Status)

	// Verify inference span
	retrievedInferenceSpan := tracer.GetSpan(inferenceSpan.SpanID)
	assert.Equal(t, 3, retrievedInferenceSpan.Attributes["conclusions_generated"])
	assert.Len(t, retrievedInferenceSpan.Events, 1)
	assert.Equal(t, SpanStatusOK, retrievedInferenceSpan.Status)

	// Verify validation span
	retrievedValidationSpan := tracer.GetSpan(validationSpan.SpanID)
	assert.Equal(t, 0, retrievedValidationSpan.Attributes["errors_found"])
	assert.Equal(t, SpanStatusOK, retrievedValidationSpan.Status)
}
