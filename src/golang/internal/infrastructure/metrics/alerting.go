package metrics

import (
	"fmt"
	"sync"
	"time"
)

// AlertManager manages alerts for budget overruns and SLA violations.
//
// Design Principles:
// - Real-time alert generation based on thresholds
// - Multiple severity levels (info, warning, critical)
// - Configurable thresholds per alert type
// - Alert deduplication to prevent spam
// - Alert history tracking
// - Thread-safe alert management
type AlertManager struct {
	sessionID string
	traceID   string

	// Alert configuration
	config AlertConfig

	// Alert state
	alerts        []*Alert
	alertHistory  []*Alert
	firedAlerts   map[string]time.Time // Key: alert ID, Value: last fired time
	mu            sync.RWMutex

	// Metrics sources
	costReporter   *CostReporter
	profiler       *PerformanceProfiler
}

// AlertConfig defines thresholds for different alert types.
type AlertConfig struct {
	// Budget alerts
	BudgetInfoThreshold     float64 // Percentage (0-100)
	BudgetWarningThreshold  float64 // Percentage (0-100)
	BudgetCriticalThreshold float64 // Percentage (0-100)

	// SLA alerts (duration in milliseconds)
	AgentDurationWarningThreshold  int64 // ms
	AgentDurationCriticalThreshold int64 // ms
	SessionDurationWarningThreshold int64 // ms
	SessionDurationCriticalThreshold int64 // ms

	// Error rate alerts
	ErrorRateWarningThreshold  float64 // Percentage (0-100)
	ErrorRateCriticalThreshold float64 // Percentage (0-100)

	// Memory alerts
	MemoryWarningThresholdMB  int64 // MB
	MemoryCriticalThresholdMB int64 // MB

	// Alert deduplication window
	DeduplicationWindowSeconds int
}

// Alert represents a single alert.
type Alert struct {
	ID          string
	Timestamp   time.Time
	Severity    AlertSeverity
	Type        AlertType
	Message     string
	Details     map[string]interface{}
	SessionID   string
	TraceID     string
	AgentID     string // Empty for session-level alerts
	Resolved    bool
	ResolvedAt  time.Time
}

// AlertType represents the type of alert.
type AlertType string

const (
	AlertTypeBudgetOverrun     AlertType = "budget_overrun"
	AlertTypeSLAViolation      AlertType = "sla_violation"
	AlertTypeErrorRate         AlertType = "error_rate"
	AlertTypeMemoryExhaustion  AlertType = "memory_exhaustion"
	AlertTypeAgentFailure      AlertType = "agent_failure"
)

// NewAlertManager creates a new alert manager.
func NewAlertManager(
	sessionID, traceID string,
	config AlertConfig,
	costReporter *CostReporter,
	profiler *PerformanceProfiler,
) *AlertManager {
	// Set default config if not provided
	if config.BudgetInfoThreshold == 0 {
		config.BudgetInfoThreshold = 50.0
	}
	if config.BudgetWarningThreshold == 0 {
		config.BudgetWarningThreshold = 80.0
	}
	if config.BudgetCriticalThreshold == 0 {
		config.BudgetCriticalThreshold = 100.0
	}
	if config.AgentDurationWarningThreshold == 0 {
		config.AgentDurationWarningThreshold = 1000 // 1 second
	}
	if config.AgentDurationCriticalThreshold == 0 {
		config.AgentDurationCriticalThreshold = 5000 // 5 seconds
	}
	if config.SessionDurationWarningThreshold == 0 {
		config.SessionDurationWarningThreshold = 10000 // 10 seconds
	}
	if config.SessionDurationCriticalThreshold == 0 {
		config.SessionDurationCriticalThreshold = 30000 // 30 seconds
	}
	if config.ErrorRateWarningThreshold == 0 {
		config.ErrorRateWarningThreshold = 10.0 // 10%
	}
	if config.ErrorRateCriticalThreshold == 0 {
		config.ErrorRateCriticalThreshold = 25.0 // 25%
	}
	if config.MemoryWarningThresholdMB == 0 {
		config.MemoryWarningThresholdMB = 100 // 100 MB
	}
	if config.MemoryCriticalThresholdMB == 0 {
		config.MemoryCriticalThresholdMB = 500 // 500 MB
	}
	if config.DeduplicationWindowSeconds == 0 {
		config.DeduplicationWindowSeconds = 300 // 5 minutes
	}

	return &AlertManager{
		sessionID:    sessionID,
		traceID:      traceID,
		config:       config,
		alerts:       []*Alert{},
		alertHistory: []*Alert{},
		firedAlerts:  make(map[string]time.Time),
		costReporter: costReporter,
		profiler:     profiler,
	}
}

// NewDefaultAlertManager creates an alert manager with default thresholds.
func NewDefaultAlertManager(sessionID, traceID string, costReporter *CostReporter, profiler *PerformanceProfiler) *AlertManager {
	return NewAlertManager(sessionID, traceID, AlertConfig{}, costReporter, profiler)
}

// CheckAlerts checks all metrics and generates alerts.
func (am *AlertManager) CheckAlerts() []*Alert {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Clear active alerts
	am.alerts = []*Alert{}

	// Check budget alerts
	am.checkBudgetAlerts()

	// Check SLA violations
	am.checkSLAViolations()

	// Check error rate alerts
	am.checkErrorRateAlerts()

	// Check memory alerts
	am.checkMemoryAlerts()

	return am.getActiveAlertsUnsafe()
}

// checkBudgetAlerts checks for budget overruns.
func (am *AlertManager) checkBudgetAlerts() {
	if am.costReporter == nil {
		return
	}

	report := am.costReporter.GetCostReport()

	if report.BudgetLimit == 0 {
		return // No budget limit set
	}

	budgetUsage := report.BudgetUsage

	var severity AlertSeverity
	var threshold float64

	if budgetUsage >= am.config.BudgetCriticalThreshold {
		severity = AlertCritical
		threshold = am.config.BudgetCriticalThreshold
	} else if budgetUsage >= am.config.BudgetWarningThreshold {
		severity = AlertWarning
		threshold = am.config.BudgetWarningThreshold
	} else if budgetUsage >= am.config.BudgetInfoThreshold {
		severity = AlertInfo
		threshold = am.config.BudgetInfoThreshold
	} else {
		return // No alert
	}

	alert := &Alert{
		ID:        fmt.Sprintf("budget_%s_%s", severity, am.sessionID),
		Timestamp: time.Now(),
		Severity:  severity,
		Type:      AlertTypeBudgetOverrun,
		Message:   fmt.Sprintf("Budget usage at %.1f%% (threshold: %.1f%%)", budgetUsage, threshold),
		Details: map[string]interface{}{
			"budget_usage":    budgetUsage,
			"total_cost":      report.TotalCost,
			"budget_limit":    report.BudgetLimit,
			"budget_remaining": report.BudgetRemaining,
			"threshold":       threshold,
		},
		SessionID: am.sessionID,
		TraceID:   am.traceID,
	}

	am.fireAlertUnsafe(alert)
}

// checkSLAViolations checks for SLA violations.
func (am *AlertManager) checkSLAViolations() {
	if am.profiler == nil {
		return
	}

	report := am.profiler.GetPerformanceReport()

	// Check session-level SLA
	sessionDurationMS := report.TotalDurationMS

	if sessionDurationMS >= am.config.SessionDurationCriticalThreshold {
		alert := &Alert{
			ID:        fmt.Sprintf("sla_session_critical_%s", am.sessionID),
			Timestamp: time.Now(),
			Severity:  AlertCritical,
			Type:      AlertTypeSLAViolation,
			Message:   fmt.Sprintf("Session duration %dms exceeds critical SLA threshold %dms", sessionDurationMS, am.config.SessionDurationCriticalThreshold),
			Details: map[string]interface{}{
				"session_duration_ms": sessionDurationMS,
				"threshold_ms":        am.config.SessionDurationCriticalThreshold,
			},
			SessionID: am.sessionID,
			TraceID:   am.traceID,
		}
		am.fireAlertUnsafe(alert)
	} else if sessionDurationMS >= am.config.SessionDurationWarningThreshold {
		alert := &Alert{
			ID:        fmt.Sprintf("sla_session_warning_%s", am.sessionID),
			Timestamp: time.Now(),
			Severity:  AlertWarning,
			Type:      AlertTypeSLAViolation,
			Message:   fmt.Sprintf("Session duration %dms exceeds warning SLA threshold %dms", sessionDurationMS, am.config.SessionDurationWarningThreshold),
			Details: map[string]interface{}{
				"session_duration_ms": sessionDurationMS,
				"threshold_ms":        am.config.SessionDurationWarningThreshold,
			},
			SessionID: am.sessionID,
			TraceID:   am.traceID,
		}
		am.fireAlertUnsafe(alert)
	}

	// Check agent-level SLA
	for _, profile := range report.AgentProfiles {
		if profile.AvgDurationMS >= am.config.AgentDurationCriticalThreshold {
			alert := &Alert{
				ID:        fmt.Sprintf("sla_agent_critical_%s_%s", am.sessionID, profile.AgentID),
				Timestamp: time.Now(),
				Severity:  AlertCritical,
				Type:      AlertTypeSLAViolation,
				Message:   fmt.Sprintf("Agent %s average duration %dms exceeds critical SLA threshold %dms", profile.AgentID, profile.AvgDurationMS, am.config.AgentDurationCriticalThreshold),
				Details: map[string]interface{}{
					"agent_id":         profile.AgentID,
					"avg_duration_ms":  profile.AvgDurationMS,
					"threshold_ms":     am.config.AgentDurationCriticalThreshold,
					"execution_count":  profile.ExecutionCount,
				},
				SessionID: am.sessionID,
				TraceID:   am.traceID,
				AgentID:   profile.AgentID,
			}
			am.fireAlertUnsafe(alert)
		} else if profile.AvgDurationMS >= am.config.AgentDurationWarningThreshold {
			alert := &Alert{
				ID:        fmt.Sprintf("sla_agent_warning_%s_%s", am.sessionID, profile.AgentID),
				Timestamp: time.Now(),
				Severity:  AlertWarning,
				Type:      AlertTypeSLAViolation,
				Message:   fmt.Sprintf("Agent %s average duration %dms exceeds warning SLA threshold %dms", profile.AgentID, profile.AvgDurationMS, am.config.AgentDurationWarningThreshold),
				Details: map[string]interface{}{
					"agent_id":         profile.AgentID,
					"avg_duration_ms":  profile.AvgDurationMS,
					"threshold_ms":     am.config.AgentDurationWarningThreshold,
					"execution_count":  profile.ExecutionCount,
				},
				SessionID: am.sessionID,
				TraceID:   am.traceID,
				AgentID:   profile.AgentID,
			}
			am.fireAlertUnsafe(alert)
		}
	}
}

// checkErrorRateAlerts checks for high error rates.
func (am *AlertManager) checkErrorRateAlerts() {
	if am.profiler == nil {
		return
	}

	profiles := am.profiler.GetAllProfiles()

	for _, profile := range profiles {
		if profile.ExecutionCount == 0 {
			continue
		}

		errorRate := (float64(profile.ErrorCount) / float64(profile.ExecutionCount)) * 100.0

		if errorRate >= am.config.ErrorRateCriticalThreshold {
			alert := &Alert{
				ID:        fmt.Sprintf("error_rate_critical_%s_%s", am.sessionID, profile.AgentID),
				Timestamp: time.Now(),
				Severity:  AlertCritical,
				Type:      AlertTypeErrorRate,
				Message:   fmt.Sprintf("Agent %s error rate %.1f%% exceeds critical threshold %.1f%%", profile.AgentID, errorRate, am.config.ErrorRateCriticalThreshold),
				Details: map[string]interface{}{
					"agent_id":         profile.AgentID,
					"error_rate":       errorRate,
					"threshold":        am.config.ErrorRateCriticalThreshold,
					"error_count":      profile.ErrorCount,
					"execution_count":  profile.ExecutionCount,
					"last_error":       profile.LastError,
				},
				SessionID: am.sessionID,
				TraceID:   am.traceID,
				AgentID:   profile.AgentID,
			}
			am.fireAlertUnsafe(alert)
		} else if errorRate >= am.config.ErrorRateWarningThreshold {
			alert := &Alert{
				ID:        fmt.Sprintf("error_rate_warning_%s_%s", am.sessionID, profile.AgentID),
				Timestamp: time.Now(),
				Severity:  AlertWarning,
				Type:      AlertTypeErrorRate,
				Message:   fmt.Sprintf("Agent %s error rate %.1f%% exceeds warning threshold %.1f%%", profile.AgentID, errorRate, am.config.ErrorRateWarningThreshold),
				Details: map[string]interface{}{
					"agent_id":         profile.AgentID,
					"error_rate":       errorRate,
					"threshold":        am.config.ErrorRateWarningThreshold,
					"error_count":      profile.ErrorCount,
					"execution_count":  profile.ExecutionCount,
					"last_error":       profile.LastError,
				},
				SessionID: am.sessionID,
				TraceID:   am.traceID,
				AgentID:   profile.AgentID,
			}
			am.fireAlertUnsafe(alert)
		}
	}
}

// checkMemoryAlerts checks for memory exhaustion.
func (am *AlertManager) checkMemoryAlerts() {
	if am.profiler == nil {
		return
	}

	profiles := am.profiler.GetAllProfiles()

	for _, profile := range profiles {
		avgMemoryMB := profile.AvgMemoryAllocBytes / (1024 * 1024)

		if avgMemoryMB >= am.config.MemoryCriticalThresholdMB {
			alert := &Alert{
				ID:        fmt.Sprintf("memory_critical_%s_%s", am.sessionID, profile.AgentID),
				Timestamp: time.Now(),
				Severity:  AlertCritical,
				Type:      AlertTypeMemoryExhaustion,
				Message:   fmt.Sprintf("Agent %s average memory %dMB exceeds critical threshold %dMB", profile.AgentID, avgMemoryMB, am.config.MemoryCriticalThresholdMB),
				Details: map[string]interface{}{
					"agent_id":           profile.AgentID,
					"avg_memory_mb":      avgMemoryMB,
					"threshold_mb":       am.config.MemoryCriticalThresholdMB,
					"max_memory_bytes":   profile.MaxMemoryAllocBytes,
					"execution_count":    profile.ExecutionCount,
				},
				SessionID: am.sessionID,
				TraceID:   am.traceID,
				AgentID:   profile.AgentID,
			}
			am.fireAlertUnsafe(alert)
		} else if avgMemoryMB >= am.config.MemoryWarningThresholdMB {
			alert := &Alert{
				ID:        fmt.Sprintf("memory_warning_%s_%s", am.sessionID, profile.AgentID),
				Timestamp: time.Now(),
				Severity:  AlertWarning,
				Type:      AlertTypeMemoryExhaustion,
				Message:   fmt.Sprintf("Agent %s average memory %dMB exceeds warning threshold %dMB", profile.AgentID, avgMemoryMB, am.config.MemoryWarningThresholdMB),
				Details: map[string]interface{}{
					"agent_id":           profile.AgentID,
					"avg_memory_mb":      avgMemoryMB,
					"threshold_mb":       am.config.MemoryWarningThresholdMB,
					"max_memory_bytes":   profile.MaxMemoryAllocBytes,
					"execution_count":    profile.ExecutionCount,
				},
				SessionID: am.sessionID,
				TraceID:   am.traceID,
				AgentID:   profile.AgentID,
			}
			am.fireAlertUnsafe(alert)
		}
	}
}

// fireAlertUnsafe fires an alert with deduplication (assumes already locked).
func (am *AlertManager) fireAlertUnsafe(alert *Alert) {
	// Check deduplication window
	if lastFired, exists := am.firedAlerts[alert.ID]; exists {
		if time.Since(lastFired).Seconds() < float64(am.config.DeduplicationWindowSeconds) {
			return // Skip duplicate alert
		}
	}

	// Fire alert
	am.alerts = append(am.alerts, alert)
	am.alertHistory = append(am.alertHistory, alert)
	am.firedAlerts[alert.ID] = time.Now()
}

// GetActiveAlerts returns all currently active alerts.
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.getActiveAlertsUnsafe()
}

// getActiveAlertsUnsafe returns active alerts without locking.
func (am *AlertManager) getActiveAlertsUnsafe() []*Alert {
	active := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		if !alert.Resolved {
			// Deep copy
			copy := *alert
			copy.Details = make(map[string]interface{})
			for k, v := range alert.Details {
				copy.Details[k] = v
			}
			active = append(active, &copy)
		}
	}
	return active
}

// GetAlertHistory returns all alerts (active and resolved).
func (am *AlertManager) GetAlertHistory() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	history := make([]*Alert, 0, len(am.alertHistory))
	for _, alert := range am.alertHistory {
		// Deep copy
		copy := *alert
		copy.Details = make(map[string]interface{})
		for k, v := range alert.Details {
			copy.Details[k] = v
		}
		history = append(history, &copy)
	}
	return history
}

// ResolveAlert marks an alert as resolved.
func (am *AlertManager) ResolveAlert(alertID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, alert := range am.alerts {
		if alert.ID == alertID {
			alert.Resolved = true
			alert.ResolvedAt = time.Now()
			break
		}
	}
}

// ClearResolvedAlerts removes resolved alerts from active list.
func (am *AlertManager) ClearResolvedAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()

	active := []*Alert{}
	for _, alert := range am.alerts {
		if !alert.Resolved {
			active = append(active, alert)
		}
	}
	am.alerts = active
}

// Reset resets the alert manager (useful for testing).
func (am *AlertManager) Reset() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.alerts = []*Alert{}
	am.alertHistory = []*Alert{}
	am.firedAlerts = make(map[string]time.Time)
}
