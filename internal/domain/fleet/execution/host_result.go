// Package execution provides fleet execution orchestration.
package execution

import (
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
)

// HostStatus represents the status of execution on a single host.
type HostStatus string

const (
	// HostStatusPending means execution hasn't started.
	HostStatusPending HostStatus = "pending"
	// HostStatusRunning means execution is in progress.
	HostStatusRunning HostStatus = "running"
	// HostStatusSuccess means execution completed successfully.
	HostStatusSuccess HostStatus = "success"
	// HostStatusFailed means execution failed.
	HostStatusFailed HostStatus = "failed"
	// HostStatusSkipped means execution was skipped.
	HostStatusSkipped HostStatus = "skipped"
)

// HostResult captures the result of execution on a single host.
type HostResult struct {
	// HostID identifies the host.
	HostID fleet.HostID
	// Hostname is the SSH hostname.
	Hostname string
	// Status is the execution status.
	Status HostStatus
	// StartTime is when execution started.
	StartTime time.Time
	// EndTime is when execution ended.
	EndTime time.Time
	// StepResults contains results for each step.
	StepResults []StepResult
	// Error is any error that occurred.
	Error error
}

// Duration returns how long execution took.
func (r *HostResult) Duration() time.Duration {
	if r.EndTime.IsZero() {
		return 0
	}
	return r.EndTime.Sub(r.StartTime)
}

// StepsApplied returns the number of steps that were applied.
func (r *HostResult) StepsApplied() int {
	count := 0
	for _, s := range r.StepResults {
		if s.Applied {
			count++
		}
	}
	return count
}

// StepsFailed returns the number of steps that failed.
func (r *HostResult) StepsFailed() int {
	count := 0
	for _, s := range r.StepResults {
		if s.Error != nil {
			count++
		}
	}
	return count
}

// StepResult captures the result of a single step execution.
type StepResult struct {
	// StepID identifies the step.
	StepID string
	// Status indicates whether the step was satisfied before execution.
	Status StepStatus
	// Applied indicates whether the step was applied.
	Applied bool
	// Duration is how long the step took.
	Duration time.Duration
	// Output is any output from the step.
	Output string
	// Error is any error that occurred.
	Error error
}

// StepStatus represents the check result of a step.
type StepStatus string

const (
	// StepStatusSatisfied means the step was already satisfied.
	StepStatusSatisfied StepStatus = "satisfied"
	// StepStatusNeeds means the step needs to be applied.
	StepStatusNeeds StepStatus = "needs"
	// StepStatusUnknown means the check failed.
	StepStatusUnknown StepStatus = "unknown"
)

// FleetResult aggregates results across all hosts.
type FleetResult struct {
	// StartTime is when fleet execution started.
	StartTime time.Time
	// EndTime is when fleet execution ended.
	EndTime time.Time
	// HostResults contains results for each host.
	HostResults []*HostResult
}

// NewFleetResult creates a new fleet result.
func NewFleetResult() *FleetResult {
	return &FleetResult{
		StartTime:   time.Now(),
		HostResults: make([]*HostResult, 0),
	}
}

// AddHostResult adds a host result.
func (r *FleetResult) AddHostResult(hr *HostResult) {
	r.HostResults = append(r.HostResults, hr)
}

// Complete marks the fleet result as complete.
func (r *FleetResult) Complete() {
	r.EndTime = time.Now()
}

// Duration returns total fleet execution time.
func (r *FleetResult) Duration() time.Duration {
	if r.EndTime.IsZero() {
		return time.Since(r.StartTime)
	}
	return r.EndTime.Sub(r.StartTime)
}

// TotalHosts returns the total number of hosts.
func (r *FleetResult) TotalHosts() int {
	return len(r.HostResults)
}

// SuccessfulHosts returns the number of hosts that succeeded.
func (r *FleetResult) SuccessfulHosts() int {
	count := 0
	for _, hr := range r.HostResults {
		if hr.Status == HostStatusSuccess {
			count++
		}
	}
	return count
}

// FailedHosts returns the number of hosts that failed.
func (r *FleetResult) FailedHosts() int {
	count := 0
	for _, hr := range r.HostResults {
		if hr.Status == HostStatusFailed {
			count++
		}
	}
	return count
}

// SkippedHosts returns the number of hosts that were skipped.
func (r *FleetResult) SkippedHosts() int {
	count := 0
	for _, hr := range r.HostResults {
		if hr.Status == HostStatusSkipped {
			count++
		}
	}
	return count
}

// AllSuccessful returns true if all hosts succeeded.
func (r *FleetResult) AllSuccessful() bool {
	for _, hr := range r.HostResults {
		if hr.Status != HostStatusSuccess {
			return false
		}
	}
	return len(r.HostResults) > 0
}

// Summary returns a summary of the fleet execution.
type Summary struct {
	TotalHosts      int           `json:"total_hosts"`
	SuccessfulHosts int           `json:"successful_hosts"`
	FailedHosts     int           `json:"failed_hosts"`
	SkippedHosts    int           `json:"skipped_hosts"`
	TotalDuration   time.Duration `json:"total_duration"`
}

// Summary returns a summary of the fleet result.
func (r *FleetResult) Summary() Summary {
	return Summary{
		TotalHosts:      r.TotalHosts(),
		SuccessfulHosts: r.SuccessfulHosts(),
		FailedHosts:     r.FailedHosts(),
		SkippedHosts:    r.SkippedHosts(),
		TotalDuration:   r.Duration(),
	}
}
