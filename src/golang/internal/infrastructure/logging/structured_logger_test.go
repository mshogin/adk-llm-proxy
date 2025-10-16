package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStructuredLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	assert.NotNil(t, logger)
	assert.Equal(t, InfoLevel, logger.minLevel)
	assert.NotNil(t, logger.fields)
	assert.Equal(t, "adk_llm_proxy", logger.fields["service"])
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()

	assert.NotNil(t, logger)
	assert.Equal(t, InfoLevel, logger.minLevel)
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.String())
	}
}

func TestStructuredLogger_SetMinLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.SetMinLevel(WarnLevel)
	assert.Equal(t, WarnLevel, logger.minLevel)
}

func TestStructuredLogger_WithField(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.WithField("env", "test")
	assert.Equal(t, "test", logger.fields["env"])
}

func TestStructuredLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	fields := map[string]interface{}{
		"env":     "test",
		"version": "1.0.0",
	}

	logger.WithFields(fields)
	assert.Equal(t, "test", logger.fields["env"])
	assert.Equal(t, "1.0.0", logger.fields["version"])
}

func TestStructuredLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, DebugLevel)

	logger.Debug("debug message", map[string]interface{}{
		"key": "value",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "DEBUG", entry.Level)
	assert.Equal(t, "debug message", entry.Message)
	assert.Equal(t, "value", entry.Fields["key"])
	assert.NotEmpty(t, entry.Timestamp)
}

func TestStructuredLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.Info("info message", map[string]interface{}{
		"session_id": "session123",
		"trace_id":   "trace456",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "INFO", entry.Level)
	assert.Equal(t, "info message", entry.Message)
	assert.Equal(t, "session123", entry.SessionID)
	assert.Equal(t, "trace456", entry.TraceID)
}

func TestStructuredLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.Warn("warning message", map[string]interface{}{
		"reason": "test",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "WARN", entry.Level)
	assert.Equal(t, "warning message", entry.Message)
	assert.Equal(t, "test", entry.Fields["reason"])
}

func TestStructuredLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	testErr := errors.New("test error")
	logger.Error("error occurred", testErr, map[string]interface{}{
		"operation": "test_op",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "ERROR", entry.Level)
	assert.Equal(t, "error occurred", entry.Message)
	assert.Equal(t, "test error", entry.Error)
	assert.NotEmpty(t, entry.ErrorType)
	assert.NotEmpty(t, entry.StackTrace)
	assert.Equal(t, "test_op", entry.Fields["operation"])
}

func TestStructuredLogger_MinLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, WarnLevel)

	// These should be filtered
	logger.Debug("debug message")
	logger.Info("info message")

	// This should pass
	logger.Warn("warn message")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 1) // Only one log entry should be written

	var entry LogEntry
	err := json.Unmarshal([]byte(lines[0]), &entry)
	require.NoError(t, err)

	assert.Equal(t, "WARN", entry.Level)
	assert.Equal(t, "warn message", entry.Message)
}

func TestStructuredLogger_SourceFile(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.Info("test message")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.NotEmpty(t, entry.SourceFile)
	assert.Greater(t, entry.SourceLine, 0)
	assert.Contains(t, entry.SourceFile, "structured_logger_test.go")
}

func TestStructuredLogger_MultipleFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.Info("message", map[string]interface{}{
		"session_id": "s1",
		"trace_id":   "t1",
		"span_id":    "sp1",
		"agent_id":   "a1",
		"custom":     "value",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "s1", entry.SessionID)
	assert.Equal(t, "t1", entry.TraceID)
	assert.Equal(t, "sp1", entry.SpanID)
	assert.Equal(t, "a1", entry.AgentID)
	assert.Equal(t, "value", entry.Fields["custom"])
}

func TestStructuredLogger_GlobalFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.WithField("global", "field")
	logger.Info("test")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "field", entry.Fields["global"])
}

func TestLoggerContext_Creation(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	ctx := logger.NewContext(map[string]interface{}{
		"session_id": "s1",
		"trace_id":   "t1",
	})

	assert.NotNil(t, ctx)
	assert.Equal(t, "s1", ctx.fields["session_id"])
	assert.Equal(t, "t1", ctx.fields["trace_id"])
}

func TestLoggerContext_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, DebugLevel)

	ctx := logger.NewContext(map[string]interface{}{
		"session_id": "s1",
	})

	ctx.Debug("debug message", map[string]interface{}{
		"extra": "field",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "DEBUG", entry.Level)
	assert.Equal(t, "s1", entry.SessionID)
	assert.Equal(t, "field", entry.Fields["extra"])
}

func TestLoggerContext_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	ctx := logger.NewContext(map[string]interface{}{
		"session_id": "s1",
		"trace_id":   "t1",
	})

	ctx.Info("info message")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "INFO", entry.Level)
	assert.Equal(t, "s1", entry.SessionID)
	assert.Equal(t, "t1", entry.TraceID)
}

func TestLoggerContext_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	ctx := logger.NewContext(map[string]interface{}{
		"agent_id": "a1",
	})

	ctx.Warn("warning message")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "WARN", entry.Level)
	assert.Equal(t, "a1", entry.AgentID)
}

func TestLoggerContext_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	ctx := logger.NewContext(map[string]interface{}{
		"agent_id": "a1",
	})

	testErr := errors.New("context error")
	ctx.Error("error message", testErr)

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "ERROR", entry.Level)
	assert.Equal(t, "a1", entry.AgentID)
	assert.Equal(t, "context error", entry.Error)
	assert.NotEmpty(t, entry.StackTrace)
}

func TestLoggerContext_MergeFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	ctx := logger.NewContext(map[string]interface{}{
		"context_field": "context_value",
	})

	ctx.Info("test", map[string]interface{}{
		"extra_field": "extra_value",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "context_value", entry.Fields["context_field"])
	assert.Equal(t, "extra_value", entry.Fields["extra_field"])
}

func TestStructuredLogger_ThreadSafety(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	done := make(chan bool)
	numGoroutines := 10
	logsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info("test message", map[string]interface{}{
					"goroutine": id,
					"iteration": j,
				})
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check that we have expected number of log lines
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Equal(t, numGoroutines*logsPerGoroutine, len(lines))

	// Verify each line is valid JSON
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry LogEntry
		err := json.Unmarshal([]byte(line), &entry)
		assert.NoError(t, err)
		assert.Equal(t, "INFO", entry.Level)
	}
}

func TestGlobalLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	SetDefaultLogger(NewStructuredLogger(buf, DebugLevel))

	Debug("global debug message")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "DEBUG", entry.Level)
	assert.Equal(t, "global debug message", entry.Message)
}

func TestGlobalLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	SetDefaultLogger(NewStructuredLogger(buf, InfoLevel))

	Info("global info message")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "INFO", entry.Level)
	assert.Equal(t, "global info message", entry.Message)
}

func TestGlobalLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	SetDefaultLogger(NewStructuredLogger(buf, InfoLevel))

	Warn("global warning message")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "WARN", entry.Level)
	assert.Equal(t, "global warning message", entry.Message)
}

func TestGlobalLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	SetDefaultLogger(NewStructuredLogger(buf, InfoLevel))

	testErr := errors.New("global error")
	Error("global error message", testErr)

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "ERROR", entry.Level)
	assert.Equal(t, "global error message", entry.Message)
	assert.Equal(t, "global error", entry.Error)
}

func TestStructuredLogger_ELKCompatibility(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.Info("elk test message", map[string]interface{}{
		"session_id": "s1",
		"trace_id":   "t1",
		"custom":     "value",
	})

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	// Verify ELK-standard fields
	assert.NotEmpty(t, entry.Timestamp) // @timestamp
	assert.NotEmpty(t, entry.Level)     // level
	assert.NotEmpty(t, entry.Message)   // message
	assert.NotEmpty(t, entry.Logger)    // logger
	assert.NotEmpty(t, entry.SourceFile) // source_file
	assert.Greater(t, entry.SourceLine, 0) // source_line

	// Verify application-specific fields
	assert.Equal(t, "s1", entry.SessionID)
	assert.Equal(t, "t1", entry.TraceID)
	assert.Equal(t, "value", entry.Fields["custom"])
}

func TestStructuredLogger_TimestampFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStructuredLogger(buf, InfoLevel)

	logger.Info("timestamp test")

	var entry LogEntry
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	// Verify RFC3339Nano format
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z$`, entry.Timestamp)
}
