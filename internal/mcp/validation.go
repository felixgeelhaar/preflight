// Package mcp provides MCP (Model Context Protocol) server implementation for preflight.
package mcp

import (
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/validation"
)

// ValidatePlanInput validates PlanInput fields.
func ValidatePlanInput(in *PlanInput) error {
	if err := validation.ValidateConfigPath(in.ConfigPath); err != nil {
		return fmt.Errorf("invalid config_path: %w", err)
	}
	if err := validation.ValidateTarget(in.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	return nil
}

// ValidateApplyInput validates ApplyInput fields.
func ValidateApplyInput(in *ApplyInput) error {
	if err := validation.ValidateConfigPath(in.ConfigPath); err != nil {
		return fmt.Errorf("invalid config_path: %w", err)
	}
	if err := validation.ValidateTarget(in.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	return nil
}

// ValidateDoctorInput validates DoctorInput fields.
func ValidateDoctorInput(in *DoctorInput) error {
	if err := validation.ValidateConfigPath(in.ConfigPath); err != nil {
		return fmt.Errorf("invalid config_path: %w", err)
	}
	if err := validation.ValidateTarget(in.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	return nil
}

// ValidateValidateInput validates ValidateInput fields.
func ValidateValidateInput(in *ValidateInput) error {
	if err := validation.ValidateConfigPath(in.ConfigPath); err != nil {
		return fmt.Errorf("invalid config_path: %w", err)
	}
	if err := validation.ValidateTarget(in.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	// PolicyFile and OrgPolicyFile are optional paths
	// We don't require .yaml extension for policy files
	return nil
}

// ValidateStatusInput validates StatusInput fields.
func ValidateStatusInput(in *StatusInput) error {
	if err := validation.ValidateConfigPath(in.ConfigPath); err != nil {
		return fmt.Errorf("invalid config_path: %w", err)
	}
	if err := validation.ValidateTarget(in.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	return nil
}

// ValidateCaptureInput validates CaptureInput fields.
func ValidateCaptureInput(in *CaptureInput) error {
	// Provider is optional and only used for filtering
	if in.Provider != "" {
		if err := validation.ValidateTarget(in.Provider); err != nil {
			return fmt.Errorf("invalid provider: %w", err)
		}
	}
	return nil
}

// ValidateDiffInput validates DiffInput fields.
func ValidateDiffInput(in *DiffInput) error {
	if err := validation.ValidateConfigPath(in.ConfigPath); err != nil {
		return fmt.Errorf("invalid config_path: %w", err)
	}
	if err := validation.ValidateTarget(in.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	return nil
}

// ValidateRollbackInput validates RollbackInput fields.
func ValidateRollbackInput(in *RollbackInput) error {
	if err := validation.ValidateSnapshotID(in.SnapshotID); err != nil {
		return fmt.Errorf("invalid snapshot_id: %w", err)
	}
	return nil
}

// ValidateSyncInput validates SyncInput fields.
func ValidateSyncInput(in *SyncInput) error {
	if err := validation.ValidateConfigPath(in.ConfigPath); err != nil {
		return fmt.Errorf("invalid config_path: %w", err)
	}
	if err := validation.ValidateTarget(in.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	if in.Remote != "" {
		if err := validation.ValidateGitBranch(in.Remote); err != nil {
			return fmt.Errorf("invalid remote: %w", err)
		}
	}
	if in.Branch != "" {
		if err := validation.ValidateGitBranch(in.Branch); err != nil {
			return fmt.Errorf("invalid branch: %w", err)
		}
	}
	return nil
}
