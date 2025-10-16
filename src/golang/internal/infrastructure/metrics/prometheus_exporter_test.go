package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusExporter(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	assert.Equal(t, "adk_llm_proxy", exporter.namespace)
	assert.NotNil(t, exporter.collectors)
	assert.NotNil(t, exporter.metrics)
}

func TestPrometheusExporter_RegisterCollector(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name:  "test_metric",
		Type:  PrometheusGauge,
		Help:  "Test metric",
		Value: 42.0,
	})

	exporter.RegisterCollector(adapter)

	assert.Len(t, exporter.collectors, 1)
}

func TestPrometheusExporter_Export_Empty(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	output := exporter.Export()

	assert.Contains(t, output, "# Generated at")
	assert.Contains(t, output, "# Metrics from ADK LLM Proxy")
}

func TestPrometheusExporter_Export_SingleGauge(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name:  "test_gauge",
		Type:  PrometheusGauge,
		Help:  "Test gauge metric",
		Value: 42.5,
	})

	exporter.RegisterCollector(adapter)

	output := exporter.Export()

	assert.Contains(t, output, "# HELP adk_llm_proxy_test_gauge Test gauge metric")
	assert.Contains(t, output, "# TYPE adk_llm_proxy_test_gauge gauge")
	assert.Contains(t, output, "adk_llm_proxy_test_gauge 42.5")
}

func TestPrometheusExporter_Export_GaugeWithLabels(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name: "agent_duration_ms",
		Type: PrometheusGauge,
		Help: "Agent execution duration in milliseconds",
		Labels: map[string]string{
			"agent_id": "intent_detection",
			"status":   "success",
		},
		Value: 150.5,
	})

	exporter.RegisterCollector(adapter)

	output := exporter.Export()

	assert.Contains(t, output, "# HELP adk_llm_proxy_agent_duration_ms Agent execution duration in milliseconds")
	assert.Contains(t, output, "# TYPE adk_llm_proxy_agent_duration_ms gauge")
	assert.Contains(t, output, "agent_duration_ms{")
	assert.Contains(t, output, "agent_id=\"intent_detection\"")
	assert.Contains(t, output, "status=\"success\"")
	assert.Contains(t, output, "} 150.5")
}

func TestPrometheusExporter_Export_Counter(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name:  "agent_executions_total",
		Type:  PrometheusCounter,
		Help:  "Total number of agent executions",
		Value: 1000,
		Labels: map[string]string{
			"agent_id": "inference",
		},
	})

	exporter.RegisterCollector(adapter)

	output := exporter.Export()

	assert.Contains(t, output, "# HELP adk_llm_proxy_agent_executions_total Total number of agent executions")
	assert.Contains(t, output, "# TYPE adk_llm_proxy_agent_executions_total counter")
	assert.Contains(t, output, "agent_executions_total{agent_id=\"inference\"} 1000")
}

func TestPrometheusExporter_Export_Histogram(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name: "agent_duration_histogram",
		Type: PrometheusHistogram,
		Help: "Histogram of agent execution durations",
		Labels: map[string]string{
			"agent_id": "inference",
		},
		Value: []HistogramBucket{
			{UpperBound: 50, Count: 10},
			{UpperBound: 100, Count: 25},
			{UpperBound: 200, Count: 50},
		},
	})

	exporter.RegisterCollector(adapter)

	output := exporter.Export()

	assert.Contains(t, output, "# HELP adk_llm_proxy_agent_duration_histogram Histogram of agent execution durations")
	assert.Contains(t, output, "# TYPE adk_llm_proxy_agent_duration_histogram histogram")

	// Check buckets
	assert.Contains(t, output, "agent_duration_histogram_bucket{agent_id=\"inference\",le=\"50.000000\"} 10")
	assert.Contains(t, output, "agent_duration_histogram_bucket{agent_id=\"inference\",le=\"100.000000\"} 25")
	assert.Contains(t, output, "agent_duration_histogram_bucket{agent_id=\"inference\",le=\"200.000000\"} 50")
	assert.Contains(t, output, "agent_duration_histogram_bucket{agent_id=\"inference\",le=\"+Inf\"} 50")

	// Check sum and count
	assert.Contains(t, output, "agent_duration_histogram_sum{agent_id=\"inference\"}")
	assert.Contains(t, output, "agent_duration_histogram_count{agent_id=\"inference\"} 50")
}

func TestPrometheusExporter_Export_MultipleMetrics(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")

	// Add multiple metrics
	adapter.AddMetric(PrometheusMetric{
		Name:  "metric1",
		Type:  PrometheusGauge,
		Help:  "First metric",
		Value: 10.0,
	})

	adapter.AddMetric(PrometheusMetric{
		Name:  "metric2",
		Type:  PrometheusCounter,
		Help:  "Second metric",
		Value: 20,
	})

	adapter.AddMetric(PrometheusMetric{
		Name:  "metric3",
		Type:  PrometheusGauge,
		Help:  "Third metric",
		Value: 30.5,
	})

	exporter.RegisterCollector(adapter)

	output := exporter.Export()

	// Check all metrics present
	assert.Contains(t, output, "adk_llm_proxy_metric1")
	assert.Contains(t, output, "adk_llm_proxy_metric2")
	assert.Contains(t, output, "adk_llm_proxy_metric3")

	assert.Contains(t, output, "10")
	assert.Contains(t, output, "20")
	assert.Contains(t, output, "30.5")
}

func TestPrometheusExporter_Export_MultipleCollectors(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter1 := NewSimpleCollectorAdapter("collector1")
	adapter1.AddMetric(PrometheusMetric{
		Name:  "metric_from_collector1",
		Type:  PrometheusGauge,
		Help:  "Metric from collector 1",
		Value: 100.0,
	})

	adapter2 := NewSimpleCollectorAdapter("collector2")
	adapter2.AddMetric(PrometheusMetric{
		Name:  "metric_from_collector2",
		Type:  PrometheusCounter,
		Help:  "Metric from collector 2",
		Value: 200,
	})

	exporter.RegisterCollector(adapter1)
	exporter.RegisterCollector(adapter2)

	output := exporter.Export()

	assert.Contains(t, output, "metric_from_collector1")
	assert.Contains(t, output, "metric_from_collector2")
	assert.Contains(t, output, "100")
	assert.Contains(t, output, "200")
}

func TestPrometheusExporter_Export_NoNamespace(t *testing.T) {
	exporter := NewPrometheusExporter("") // No namespace

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name:  "test_metric",
		Type:  PrometheusGauge,
		Help:  "Test metric",
		Value: 42.0,
	})

	exporter.RegisterCollector(adapter)

	output := exporter.Export()

	// Without namespace, metric name should not have prefix
	assert.Contains(t, output, "# HELP test_metric Test metric")
	assert.Contains(t, output, "# TYPE test_metric gauge")
	assert.Contains(t, output, "test_metric 42")
}

func TestPrometheusExporter_Export_LabelEscaping(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name: "test_metric",
		Type: PrometheusGauge,
		Help: "Test metric with special characters",
		Labels: map[string]string{
			"label_with_quotes":   "value with \"quotes\"",
			"label_with_newline":  "value with\nnewline",
			"label_with_backslash": "value with \\ backslash",
		},
		Value: 42.0,
	})

	exporter.RegisterCollector(adapter)

	output := exporter.Export()

	// Check escaped values
	assert.Contains(t, output, "label_with_quotes=\"value with \\\"quotes\\\"\"")
	assert.Contains(t, output, "label_with_newline=\"value with\\nnewline\"")
	assert.Contains(t, output, "label_with_backslash=\"value with \\\\ backslash\"")
}

func TestCreateAgentMetric(t *testing.T) {
	metric := CreateAgentMetric("intent_detection", "agent_duration_ms", 150.5, "Agent execution duration")

	assert.Equal(t, "agent_duration_ms", metric.Name)
	assert.Equal(t, PrometheusGauge, metric.Type)
	assert.Equal(t, "Agent execution duration", metric.Help)
	assert.Equal(t, "intent_detection", metric.Labels["agent_id"])
	assert.Equal(t, 150.5, metric.Value)
}

func TestCreateSessionMetric(t *testing.T) {
	metric := CreateSessionMetric("session123", "session_duration_ms", 5000, "Session total duration")

	assert.Equal(t, "session_duration_ms", metric.Name)
	assert.Equal(t, PrometheusGauge, metric.Type)
	assert.Equal(t, "Session total duration", metric.Help)
	assert.Equal(t, "session123", metric.Labels["session_id"])
	assert.Equal(t, 5000, metric.Value)
}

func TestCreateLLMMetric(t *testing.T) {
	metric := CreateLLMMetric("openai", "gpt-4o-mini", "llm_tokens_total", 1500, "Total tokens used")

	assert.Equal(t, "llm_tokens_total", metric.Name)
	assert.Equal(t, PrometheusGauge, metric.Type)
	assert.Equal(t, "Total tokens used", metric.Help)
	assert.Equal(t, "openai", metric.Labels["provider"])
	assert.Equal(t, "gpt-4o-mini", metric.Labels["model"])
	assert.Equal(t, 1500, metric.Value)
}

func TestSimpleCollectorAdapter(t *testing.T) {
	adapter := NewSimpleCollectorAdapter("test_collector")

	assert.Equal(t, "test_collector", adapter.GetName())
	assert.Empty(t, adapter.ExportPrometheusMetrics())

	metric1 := PrometheusMetric{
		Name:  "metric1",
		Type:  PrometheusGauge,
		Help:  "First metric",
		Value: 10.0,
	}

	metric2 := PrometheusMetric{
		Name:  "metric2",
		Type:  PrometheusCounter,
		Help:  "Second metric",
		Value: 20,
	}

	adapter.AddMetric(metric1)
	adapter.AddMetric(metric2)

	metrics := adapter.ExportPrometheusMetrics()
	assert.Len(t, metrics, 2)
	assert.Equal(t, "metric1", metrics[0].Name)
	assert.Equal(t, "metric2", metrics[1].Name)

	adapter.Clear()
	assert.Empty(t, adapter.ExportPrometheusMetrics())
}

func TestPrometheusExporter_Export_ComplexScenario(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	// Agent metrics collector
	agentAdapter := NewSimpleCollectorAdapter("agent_metrics")
	agentAdapter.AddMetric(CreateAgentMetric("intent_detection", "agent_duration_ms", 150.0, "Agent execution duration in milliseconds"))
	agentAdapter.AddMetric(CreateAgentMetric("intent_detection", "agent_executions_total", 100, "Total agent executions"))
	agentAdapter.AddMetric(CreateAgentMetric("inference", "agent_duration_ms", 300.0, "Agent execution duration in milliseconds"))

	// LLM metrics collector
	llmAdapter := NewSimpleCollectorAdapter("llm_metrics")
	llmAdapter.AddMetric(CreateLLMMetric("openai", "gpt-4o-mini", "llm_tokens_total", 5000, "Total tokens used"))
	llmAdapter.AddMetric(CreateLLMMetric("openai", "gpt-4o-mini", "llm_cost_usd", 0.005, "Total cost in USD"))
	llmAdapter.AddMetric(CreateLLMMetric("anthropic", "claude-3-5-sonnet", "llm_tokens_total", 8000, "Total tokens used"))

	// Session metrics collector
	sessionAdapter := NewSimpleCollectorAdapter("session_metrics")
	sessionAdapter.AddMetric(CreateSessionMetric("session123", "session_duration_ms", 10000, "Session total duration"))

	exporter.RegisterCollector(agentAdapter)
	exporter.RegisterCollector(llmAdapter)
	exporter.RegisterCollector(sessionAdapter)

	output := exporter.Export()

	// Verify structure
	assert.Contains(t, output, "# Generated at")
	assert.Contains(t, output, "# Metrics from ADK LLM Proxy")

	// Verify agent metrics
	assert.Contains(t, output, "adk_llm_proxy_agent_duration_ms")
	assert.Contains(t, output, "agent_id=\"intent_detection\"")
	assert.Contains(t, output, "agent_id=\"inference\"")
	assert.Contains(t, output, "150")
	assert.Contains(t, output, "300")

	// Verify LLM metrics
	assert.Contains(t, output, "adk_llm_proxy_llm_tokens_total")
	assert.Contains(t, output, "provider=\"openai\"")
	assert.Contains(t, output, "model=\"gpt-4o-mini\"")
	assert.Contains(t, output, "provider=\"anthropic\"")
	assert.Contains(t, output, "model=\"claude-3-5-sonnet\"")
	assert.Contains(t, output, "5000")
	assert.Contains(t, output, "8000")

	// Verify session metrics
	assert.Contains(t, output, "adk_llm_proxy_session_duration_ms")
	assert.Contains(t, output, "session_id=\"session123\"")
	assert.Contains(t, output, "10000")

	// Verify no duplicate TYPE/HELP declarations
	typeCount := strings.Count(output, "# TYPE adk_llm_proxy_agent_duration_ms")
	assert.Equal(t, 1, typeCount, "Should have exactly one TYPE declaration per metric name")
}

func TestPrometheusExporter_ExportToHandler(t *testing.T) {
	exporter := NewPrometheusExporter("adk_llm_proxy")

	adapter := NewSimpleCollectorAdapter("test_collector")
	adapter.AddMetric(PrometheusMetric{
		Name:  "test_metric",
		Type:  PrometheusGauge,
		Help:  "Test metric",
		Value: 42.0,
	})

	exporter.RegisterCollector(adapter)

	output := exporter.ExportToHandler()

	// Should be same as Export()
	assert.Contains(t, output, "# Generated at")
	assert.Contains(t, output, "adk_llm_proxy_test_metric 42")
}
