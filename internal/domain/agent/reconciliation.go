package agent

import "time"

// ReconciliationResult represents the outcome of a reconciliation cycle.
type ReconciliationResult struct {
	// StartedAt is when reconciliation started.
	StartedAt time.Time `json:"started_at"`
	// CompletedAt is when reconciliation finished.
	CompletedAt time.Time `json:"completed_at"`
	// Duration is how long reconciliation took.
	Duration time.Duration `json:"duration"`

	// DriftDetected indicates if any drift was found.
	DriftDetected bool `json:"drift_detected"`
	// DriftCount is the number of drifted items.
	DriftCount int `json:"drift_count"`
	// DriftItems lists the specific drift detections.
	DriftItems []DriftItem `json:"drift_items,omitempty"`

	// RemediationApplied indicates if fixes were applied.
	RemediationApplied bool `json:"remediation_applied"`
	// RemediationCount is the number of remediations applied.
	RemediationCount int `json:"remediation_count"`
	// RemediationItems lists the specific remediations.
	RemediationItems []RemediationItem `json:"remediation_items,omitempty"`

	// Errors contains any errors encountered.
	Errors []ReconciliationError `json:"errors,omitempty"`

	// PendingApprovals lists items awaiting approval.
	PendingApprovals []ApprovalRequest `json:"pending_approvals,omitempty"`
}

// DriftItem represents a single detected drift.
type DriftItem struct {
	// ID uniquely identifies this drift.
	ID string `json:"id"`
	// Type categorizes the drift (package, file, config).
	Type string `json:"type"`
	// Name is the name of the drifted item.
	Name string `json:"name"`
	// Expected is what the config specifies.
	Expected string `json:"expected"`
	// Actual is what was found on the system.
	Actual string `json:"actual"`
	// Severity indicates how critical this drift is.
	Severity DriftSeverity `json:"severity"`
}

// DriftSeverity indicates the severity of detected drift.
type DriftSeverity string

const (
	// DriftSeverityLow is for minor differences.
	DriftSeverityLow DriftSeverity = "low"
	// DriftSeverityMedium is for moderate differences.
	DriftSeverityMedium DriftSeverity = "medium"
	// DriftSeverityHigh is for significant differences.
	DriftSeverityHigh DriftSeverity = "high"
	// DriftSeverityCritical is for critical differences.
	DriftSeverityCritical DriftSeverity = "critical"
)

// RemediationItem represents a single remediation action.
type RemediationItem struct {
	// ID uniquely identifies this remediation.
	ID string `json:"id"`
	// DriftID links to the drift this remediates.
	DriftID string `json:"drift_id"`
	// Action describes what was done.
	Action string `json:"action"`
	// Success indicates if remediation succeeded.
	Success bool `json:"success"`
	// Message provides details about the remediation.
	Message string `json:"message,omitempty"`
}

// ReconciliationError represents an error during reconciliation.
type ReconciliationError struct {
	// Phase indicates when the error occurred.
	Phase string `json:"phase"`
	// Message describes the error.
	Message string `json:"message"`
	// Details provides additional context.
	Details string `json:"details,omitempty"`
	// Recoverable indicates if the error can be retried.
	Recoverable bool `json:"recoverable"`
}

// ApprovalRequest represents an item awaiting user approval.
type ApprovalRequest struct {
	// ID uniquely identifies this approval request.
	ID string `json:"id"`
	// DriftID links to the drift requiring approval.
	DriftID string `json:"drift_id"`
	// Action describes what will happen if approved.
	Action string `json:"action"`
	// Risk indicates the risk level of this action.
	Risk string `json:"risk"`
	// CreatedAt is when the approval was requested.
	CreatedAt time.Time `json:"created_at"`
	// ExpiresAt is when the approval request expires.
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// NewReconciliationResult creates a new result with the start time set.
func NewReconciliationResult() *ReconciliationResult {
	return &ReconciliationResult{
		StartedAt: time.Now(),
	}
}

// Complete marks the reconciliation as complete.
func (r *ReconciliationResult) Complete() {
	r.CompletedAt = time.Now()
	r.Duration = r.CompletedAt.Sub(r.StartedAt)
}

// AddDrift adds a detected drift item.
func (r *ReconciliationResult) AddDrift(item DriftItem) {
	r.DriftItems = append(r.DriftItems, item)
	r.DriftCount = len(r.DriftItems)
	r.DriftDetected = r.DriftCount > 0
}

// AddRemediation adds a remediation action.
func (r *ReconciliationResult) AddRemediation(item RemediationItem) {
	r.RemediationItems = append(r.RemediationItems, item)
	r.RemediationCount = len(r.RemediationItems)
	if item.Success {
		r.RemediationApplied = true
	}
}

// AddError adds an error to the result.
func (r *ReconciliationResult) AddError(phase, message string, recoverable bool) {
	r.Errors = append(r.Errors, ReconciliationError{
		Phase:       phase,
		Message:     message,
		Recoverable: recoverable,
	})
}

// AddPendingApproval adds an approval request.
func (r *ReconciliationResult) AddPendingApproval(req ApprovalRequest) {
	r.PendingApprovals = append(r.PendingApprovals, req)
}

// HasErrors returns true if there were any errors.
func (r *ReconciliationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasPendingApprovals returns true if there are pending approvals.
func (r *ReconciliationResult) HasPendingApprovals() bool {
	return len(r.PendingApprovals) > 0
}

// Summary returns a brief summary of the reconciliation.
func (r *ReconciliationResult) Summary() string {
	if r.DriftDetected {
		if r.RemediationApplied {
			return "drift detected and remediated"
		}
		if r.HasPendingApprovals() {
			return "drift detected, awaiting approval"
		}
		return "drift detected"
	}
	return "no drift detected"
}
