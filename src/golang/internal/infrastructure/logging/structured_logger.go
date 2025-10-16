package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// StructuredLogger provides ELK-compatible JSON logging.
//
// Design Principles:
// - JSON structured output for easy parsing
// - Standard fields (@timestamp, level, message, etc.)
// - Thread-safe logging
// - Context fields (session_id, trace_id, agent_id)
// - Performance-optimized with buffer pool
type StructuredLogger struct {
	mu       sync.Mutex
	writer   io.Writer
	minLevel LogLevel
	fields   map[string]interface{} // Global fields for all logs

	// Buffer pool for efficient JSON encoding
	bufferPool sync.Pool
}

// LogLevel represents logging severity levels.
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry in ELK-compatible format.
type LogEntry struct {
	// Standard ELK fields
	Timestamp  string                 `json:"@timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Logger     string                 `json:"logger,omitempty"`
	Thread     string                 `json:"thread,omitempty"`
	Host       string                 `json:"host,omitempty"`
	SourceFile string                 `json:"source_file,omitempty"`
	SourceLine int                    `json:"source_line,omitempty"`
	Fields     map[string]interface{} `json:"fields,omitempty"`

	// Application-specific fields
	SessionID string `json:"session_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	SpanID    string `json:"span_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`

	// Error tracking
	Error      string                 `json:"error,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	ErrorType  string                 `json:"error_type,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// NewStructuredLogger creates a new structured logger.
func NewStructuredLogger(writer io.Writer, minLevel LogLevel) *StructuredLogger {
	if writer == nil {
		writer = os.Stdout
	}

	hostname, _ := os.Hostname()

	return &StructuredLogger{
		writer:   writer,
		minLevel: minLevel,
		fields: map[string]interface{}{
			"service": "adk_llm_proxy",
			"host":    hostname,
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return &LogEntry{
					Fields:  make(map[string]interface{}),
					Context: make(map[string]interface{}),
				}
			},
		},
	}
}

// NewDefaultLogger creates a logger with INFO level to stdout.
func NewDefaultLogger() *StructuredLogger {
	return NewStructuredLogger(os.Stdout, InfoLevel)
}

// SetMinLevel sets the minimum log level.
func (l *StructuredLogger) SetMinLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// WithField adds a global field to all log entries.
func (l *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fields[key] = value
	return l
}

// WithFields adds multiple global fields to all log entries.
func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

// Debug logs a debug-level message.
func (l *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	l.log(DebugLevel, message, nil, fields...)
}

// Info logs an info-level message.
func (l *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	l.log(InfoLevel, message, nil, fields...)
}

// Warn logs a warning-level message.
func (l *StructuredLogger) Warn(message string, fields ...map[string]interface{}) {
	l.log(WarnLevel, message, nil, fields...)
}

// Error logs an error-level message.
func (l *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	l.log(ErrorLevel, message, err, fields...)
}

// Fatal logs a fatal-level message and exits the program.
func (l *StructuredLogger) Fatal(message string, err error, fields ...map[string]interface{}) {
	l.log(FatalLevel, message, err, fields...)
	os.Exit(1)
}

// log is the internal logging function.
func (l *StructuredLogger) log(level LogLevel, message string, err error, fields ...map[string]interface{}) {
	// Check log level
	if level < l.minLevel {
		return
	}

	// Get entry from pool
	entry := l.bufferPool.Get().(*LogEntry)
	defer func() {
		// Clear and return to pool
		entry.Fields = make(map[string]interface{})
		entry.Context = make(map[string]interface{})
		entry.SessionID = ""
		entry.TraceID = ""
		entry.SpanID = ""
		entry.AgentID = ""
		entry.Error = ""
		entry.StackTrace = ""
		entry.ErrorType = ""
		entry.SourceFile = ""
		entry.SourceLine = 0
		l.bufferPool.Put(entry)
	}()

	// Fill entry
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	entry.Level = level.String()
	entry.Message = message
	entry.Logger = "adk_llm_proxy"

	// Add source file and line (skip 2 frames: log() and public method)
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.SourceFile = file
		entry.SourceLine = line
	}

	// Copy global fields
	l.mu.Lock()
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	l.mu.Unlock()

	// Merge provided fields
	if len(fields) > 0 {
		for _, fieldMap := range fields {
			for k, v := range fieldMap {
				// Special handling for known fields
				switch k {
				case "session_id":
					if sessionID, ok := v.(string); ok {
						entry.SessionID = sessionID
					}
				case "trace_id":
					if traceID, ok := v.(string); ok {
						entry.TraceID = traceID
					}
				case "span_id":
					if spanID, ok := v.(string); ok {
						entry.SpanID = spanID
					}
				case "agent_id":
					if agentID, ok := v.(string); ok {
						entry.AgentID = agentID
					}
				default:
					entry.Fields[k] = v
				}
			}
		}
	}

	// Handle error
	if err != nil {
		entry.Error = err.Error()
		entry.ErrorType = fmt.Sprintf("%T", err)

		// Capture stack trace for errors and fatal logs
		if level >= ErrorLevel {
			entry.StackTrace = captureStackTrace(3) // Skip 3 frames
		}
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON encoding fails
		fmt.Fprintf(l.writer, "{\"error\":\"failed to encode log entry\",\"original_message\":%q}\n", message)
		return
	}

	// Write log entry
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer.Write(data)
	l.writer.Write([]byte("\n"))
}

// captureStackTrace captures the current stack trace.
func captureStackTrace(skip int) string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// LoggerContext provides contextual logging with pre-set fields.
type LoggerContext struct {
	logger *StructuredLogger
	fields map[string]interface{}
}

// NewContext creates a new logger context with pre-set fields.
func (l *StructuredLogger) NewContext(fields map[string]interface{}) *LoggerContext {
	return &LoggerContext{
		logger: l,
		fields: fields,
	}
}

// Debug logs a debug-level message with context fields.
func (lc *LoggerContext) Debug(message string, fields ...map[string]interface{}) {
	allFields := lc.mergeFields(fields...)
	lc.logger.Debug(message, allFields)
}

// Info logs an info-level message with context fields.
func (lc *LoggerContext) Info(message string, fields ...map[string]interface{}) {
	allFields := lc.mergeFields(fields...)
	lc.logger.Info(message, allFields)
}

// Warn logs a warning-level message with context fields.
func (lc *LoggerContext) Warn(message string, fields ...map[string]interface{}) {
	allFields := lc.mergeFields(fields...)
	lc.logger.Warn(message, allFields)
}

// Error logs an error-level message with context fields.
func (lc *LoggerContext) Error(message string, err error, fields ...map[string]interface{}) {
	allFields := lc.mergeFields(fields...)
	lc.logger.Error(message, err, allFields)
}

// Fatal logs a fatal-level message with context fields and exits.
func (lc *LoggerContext) Fatal(message string, err error, fields ...map[string]interface{}) {
	allFields := lc.mergeFields(fields...)
	lc.logger.Fatal(message, err, allFields)
}

// mergeFields merges context fields with additional fields.
func (lc *LoggerContext) mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy context fields
	for k, v := range lc.fields {
		merged[k] = v
	}

	// Merge additional fields
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			merged[k] = v
		}
	}

	return merged
}

// Global default logger
var defaultLogger = NewDefaultLogger()

// SetDefaultLogger sets the global default logger.
func SetDefaultLogger(logger *StructuredLogger) {
	defaultLogger = logger
}

// GetDefaultLogger returns the global default logger.
func GetDefaultLogger() *StructuredLogger {
	return defaultLogger
}

// Debug logs to the default logger.
func Debug(message string, fields ...map[string]interface{}) {
	defaultLogger.Debug(message, fields...)
}

// Info logs to the default logger.
func Info(message string, fields ...map[string]interface{}) {
	defaultLogger.Info(message, fields...)
}

// Warn logs to the default logger.
func Warn(message string, fields ...map[string]interface{}) {
	defaultLogger.Warn(message, fields...)
}

// Error logs to the default logger.
func Error(message string, err error, fields ...map[string]interface{}) {
	defaultLogger.Error(message, err, fields...)
}

// Fatal logs to the default logger and exits.
func Fatal(message string, err error, fields ...map[string]interface{}) {
	defaultLogger.Fatal(message, err, fields...)
}
