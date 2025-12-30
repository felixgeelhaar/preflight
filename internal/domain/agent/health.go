package agent

import "time"

// Health represents the health level of the agent.
type Health string

const (
	// HealthUnknown indicates health status is not yet determined.
	HealthUnknown Health = "unknown"
	// HealthHealthy indicates the agent is functioning normally.
	HealthHealthy Health = "healthy"
	// HealthDegraded indicates the agent has issues but is still running.
	HealthDegraded Health = "degraded"
	// HealthUnhealthy indicates the agent is not functioning correctly.
	HealthUnhealthy Health = "unhealthy"
)

// HealthStatus represents the current health of the agent.
type HealthStatus struct {
	// Status is the overall health level.
	Status Health `json:"status"`
	// LastCheck is when health was last evaluated.
	LastCheck time.Time `json:"last_check"`
	// Message provides additional context about the health status.
	Message string `json:"message,omitempty"`
	// Checks contains individual health check results.
	Checks []HealthCheck `json:"checks,omitempty"`
}

// HealthCheck represents a single health check result.
type HealthCheck struct {
	// Name identifies the health check.
	Name string `json:"name"`
	// Status is the result of this check.
	Status Health `json:"status"`
	// Message provides details about the check result.
	Message string `json:"message,omitempty"`
	// Duration is how long the check took.
	Duration time.Duration `json:"duration,omitempty"`
}

// IsHealthy returns true if the status is healthy.
func (h HealthStatus) IsHealthy() bool {
	return h.Status == HealthHealthy
}

// IsDegraded returns true if the status is degraded.
func (h HealthStatus) IsDegraded() bool {
	return h.Status == HealthDegraded
}

// IsUnhealthy returns true if the status is unhealthy.
func (h HealthStatus) IsUnhealthy() bool {
	return h.Status == HealthUnhealthy
}

// AddCheck adds a health check result and updates overall status.
func (h *HealthStatus) AddCheck(check HealthCheck) {
	h.Checks = append(h.Checks, check)
	h.updateOverallStatus()
}

// updateOverallStatus recalculates the overall status based on checks.
func (h *HealthStatus) updateOverallStatus() {
	if len(h.Checks) == 0 {
		return
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, check := range h.Checks {
		switch check.Status { //nolint:exhaustive // Only track problematic states
		case HealthUnhealthy:
			hasUnhealthy = true
		case HealthDegraded:
			hasDegraded = true
		}
	}

	switch {
	case hasUnhealthy:
		h.Status = HealthUnhealthy
	case hasDegraded:
		h.Status = HealthDegraded
	default:
		h.Status = HealthHealthy
	}
}

// NewHealthStatus creates a new healthy status.
func NewHealthStatus() HealthStatus {
	return HealthStatus{
		Status:    HealthHealthy,
		LastCheck: time.Now(),
	}
}

// NewHealthCheck creates a new health check result.
func NewHealthCheck(name string, status Health, message string) HealthCheck {
	return HealthCheck{
		Name:    name,
		Status:  status,
		Message: message,
	}
}
