package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// PrometheusExporter exports metrics in Prometheus text format.
//
// Design Principles:
// - Prometheus text exposition format
// - Support for counters, gauges, histograms, summaries
// - Automatic metric type detection
// - Thread-safe metric registration
// - Label support for multi-dimensional metrics
type PrometheusExporter struct {
	collectors []MetricsCollectorInterface
	mu         sync.RWMutex

	// Metric registry
	metrics map[string]*PrometheusMetric

	// Namespace for all metrics
	namespace string
}

// MetricsCollectorInterface defines the interface for metrics collectors.
type MetricsCollectorInterface interface {
	// GetName returns the collector name for metric prefixing
	GetName() string
	// Export returns prometheus-compatible metrics
	ExportPrometheusMetrics() []PrometheusMetric
}

// PrometheusMetric represents a metric in Prometheus format.
type PrometheusMetric struct {
	Name   string
	Type   PrometheusMetricType
	Help   string
	Labels map[string]string
	Value  interface{} // float64, int, or []HistogramBucket
}

// PrometheusMetricType represents Prometheus metric types.
type PrometheusMetricType string

const (
	PrometheusCounter   PrometheusMetricType = "counter"
	PrometheusGauge     PrometheusMetricType = "gauge"
	PrometheusHistogram PrometheusMetricType = "histogram"
	PrometheusSummary   PrometheusMetricType = "summary"
)

// HistogramBucket represents a single bucket in a histogram.
type HistogramBucket struct {
	UpperBound float64
	Count      uint64
}

// NewPrometheusExporter creates a new Prometheus exporter.
func NewPrometheusExporter(namespace string) *PrometheusExporter {
	return &PrometheusExporter{
		collectors: []MetricsCollectorInterface{},
		metrics:    make(map[string]*PrometheusMetric),
		namespace:  namespace,
	}
}

// RegisterCollector registers a metrics collector for export.
func (e *PrometheusExporter) RegisterCollector(collector MetricsCollectorInterface) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.collectors = append(e.collectors, collector)
}

// Export generates Prometheus text format output.
func (e *PrometheusExporter) Export() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var builder strings.Builder

	// Add timestamp comment
	builder.WriteString(fmt.Sprintf("# Generated at %s\n", time.Now().Format(time.RFC3339)))
	builder.WriteString("# Metrics from ADK LLM Proxy\n\n")

	// Collect metrics from all collectors
	allMetrics := e.collectMetrics()

	// Group metrics by name for proper formatting
	metricGroups := make(map[string][]*PrometheusMetric)
	for _, metric := range allMetrics {
		metricGroups[metric.Name] = append(metricGroups[metric.Name], metric)
	}

	// Export each metric group
	for name, metrics := range metricGroups {
		if len(metrics) == 0 {
			continue
		}

		// Write metric type and help
		builder.WriteString(fmt.Sprintf("# HELP %s %s\n", name, metrics[0].Help))
		builder.WriteString(fmt.Sprintf("# TYPE %s %s\n", name, metrics[0].Type))

		// Write metric values
		for _, metric := range metrics {
			e.writeMetricValue(&builder, metric)
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// collectMetrics gathers metrics from all registered collectors.
func (e *PrometheusExporter) collectMetrics() []*PrometheusMetric {
	metrics := []*PrometheusMetric{}

	for _, collector := range e.collectors {
		collectorMetrics := collector.ExportPrometheusMetrics()
		for i := range collectorMetrics {
			// Add namespace prefix if configured
			if e.namespace != "" {
				collectorMetrics[i].Name = e.namespace + "_" + collectorMetrics[i].Name
			}
			metrics = append(metrics, &collectorMetrics[i])
		}
	}

	return metrics
}

// writeMetricValue writes a single metric value in Prometheus format.
func (e *PrometheusExporter) writeMetricValue(builder *strings.Builder, metric *PrometheusMetric) {
	metricName := metric.Name

	switch metric.Type {
	case PrometheusCounter, PrometheusGauge:
		// counter{label1="value1",label2="value2"} 42
		builder.WriteString(metricName)
		e.writeLabels(builder, metric.Labels)
		builder.WriteString(fmt.Sprintf(" %v\n", metric.Value))

	case PrometheusHistogram:
		// Histogram requires _bucket, _sum, _count suffixes
		buckets, ok := metric.Value.([]HistogramBucket)
		if !ok {
			return
		}

		var sum float64
		var count uint64

		// Write buckets
		for _, bucket := range buckets {
			builder.WriteString(metricName + "_bucket")
			labels := make(map[string]string, len(metric.Labels)+1)
			for k, v := range metric.Labels {
				labels[k] = v
			}
			labels["le"] = fmt.Sprintf("%f", bucket.UpperBound)
			e.writeLabels(builder, labels)
			builder.WriteString(fmt.Sprintf(" %d\n", bucket.Count))

			count = bucket.Count // Use last bucket count as total
			sum += bucket.UpperBound * float64(bucket.Count)
		}

		// Write +Inf bucket
		builder.WriteString(metricName + "_bucket")
		infLabels := make(map[string]string, len(metric.Labels)+1)
		for k, v := range metric.Labels {
			infLabels[k] = v
		}
		infLabels["le"] = "+Inf"
		e.writeLabels(builder, infLabels)
		builder.WriteString(fmt.Sprintf(" %d\n", count))

		// Write sum
		builder.WriteString(metricName + "_sum")
		e.writeLabels(builder, metric.Labels)
		builder.WriteString(fmt.Sprintf(" %f\n", sum))

		// Write count
		builder.WriteString(metricName + "_count")
		e.writeLabels(builder, metric.Labels)
		builder.WriteString(fmt.Sprintf(" %d\n", count))

	case PrometheusSummary:
		// Similar to histogram but with quantiles
		// Simplified: treat as gauge for now
		builder.WriteString(metricName)
		e.writeLabels(builder, metric.Labels)
		builder.WriteString(fmt.Sprintf(" %v\n", metric.Value))
	}
}

// writeLabels writes labels in Prometheus format: {label1="value1",label2="value2"}
func (e *PrometheusExporter) writeLabels(builder *strings.Builder, labels map[string]string) {
	if len(labels) == 0 {
		return
	}

	builder.WriteString("{")

	first := true
	for key, value := range labels {
		if !first {
			builder.WriteString(",")
		}
		// Escape label values
		escapedValue := strings.ReplaceAll(value, "\\", "\\\\")
		escapedValue = strings.ReplaceAll(escapedValue, "\"", "\\\"")
		escapedValue = strings.ReplaceAll(escapedValue, "\n", "\\n")

		builder.WriteString(fmt.Sprintf("%s=\"%s\"", key, escapedValue))
		first = false
	}

	builder.WriteString("}")
}

// ExportToHandler generates metrics for HTTP handler (same as Export for now).
func (e *PrometheusExporter) ExportToHandler() string {
	return e.Export()
}

// CreateAgentMetric creates a Prometheus metric from agent metrics.
func CreateAgentMetric(agentID string, metricName string, value interface{}, help string) PrometheusMetric {
	return PrometheusMetric{
		Name: metricName,
		Type: PrometheusGauge,
		Help: help,
		Labels: map[string]string{
			"agent_id": agentID,
		},
		Value: value,
	}
}

// CreateSessionMetric creates a Prometheus metric from session metrics.
func CreateSessionMetric(sessionID string, metricName string, value interface{}, help string) PrometheusMetric {
	return PrometheusMetric{
		Name: metricName,
		Type: PrometheusGauge,
		Help: help,
		Labels: map[string]string{
			"session_id": sessionID,
		},
		Value: value,
	}
}

// CreateLLMMetric creates a Prometheus metric from LLM metrics.
func CreateLLMMetric(provider string, model string, metricName string, value interface{}, help string) PrometheusMetric {
	return PrometheusMetric{
		Name: metricName,
		Type: PrometheusGauge,
		Help: help,
		Labels: map[string]string{
			"provider": provider,
			"model":    model,
		},
		Value: value,
	}
}

// SimpleCollectorAdapter adapts simple metrics to PrometheusExporter interface.
type SimpleCollectorAdapter struct {
	name    string
	metrics []PrometheusMetric
}

// NewSimpleCollectorAdapter creates a new simple collector adapter.
func NewSimpleCollectorAdapter(name string) *SimpleCollectorAdapter {
	return &SimpleCollectorAdapter{
		name:    name,
		metrics: []PrometheusMetric{},
	}
}

// GetName returns the collector name.
func (a *SimpleCollectorAdapter) GetName() string {
	return a.name
}

// AddMetric adds a metric to the adapter.
func (a *SimpleCollectorAdapter) AddMetric(metric PrometheusMetric) {
	a.metrics = append(a.metrics, metric)
}

// ExportPrometheusMetrics exports all metrics.
func (a *SimpleCollectorAdapter) ExportPrometheusMetrics() []PrometheusMetric {
	return a.metrics
}

// Clear clears all metrics.
func (a *SimpleCollectorAdapter) Clear() {
	a.metrics = []PrometheusMetric{}
}
