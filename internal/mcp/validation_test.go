package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePlanInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *PlanInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid minimal",
			input:   &PlanInput{},
			wantErr: false,
		},
		{
			name:    "valid with config and target",
			input:   &PlanInput{ConfigPath: "preflight.yaml", Target: "work"},
			wantErr: false,
		},
		{
			name:    "invalid config path",
			input:   &PlanInput{ConfigPath: "config; rm -rf /"},
			wantErr: true,
			errMsg:  "invalid config_path",
		},
		{
			name:    "invalid target",
			input:   &PlanInput{Target: "work;rm"},
			wantErr: true,
			errMsg:  "invalid target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePlanInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateApplyInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *ApplyInput
		wantErr bool
	}{
		{
			name:    "valid",
			input:   &ApplyInput{ConfigPath: "preflight.yaml", Target: "work", Confirm: true},
			wantErr: false,
		},
		{
			name:    "invalid config",
			input:   &ApplyInput{ConfigPath: "config$(id)"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateApplyInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDoctorInput(t *testing.T) {
	t.Parallel()

	input := &DoctorInput{ConfigPath: "preflight.yaml", Target: "default"}
	err := ValidateDoctorInput(input)
	assert.NoError(t, err)

	input = &DoctorInput{ConfigPath: "config`id`.yaml"}
	err = ValidateDoctorInput(input)
	assert.Error(t, err)
}

func TestValidateValidateInput(t *testing.T) {
	t.Parallel()

	input := &ValidateInput{ConfigPath: "preflight.yaml", Target: "default"}
	err := ValidateValidateInput(input)
	assert.NoError(t, err)

	// Invalid config path
	input = &ValidateInput{ConfigPath: "config; rm -rf /"}
	err = ValidateValidateInput(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config_path")
}

func TestValidateStatusInput(t *testing.T) {
	t.Parallel()

	input := &StatusInput{ConfigPath: "preflight.yaml"}
	err := ValidateStatusInput(input)
	assert.NoError(t, err)

	input = &StatusInput{Target: "work|cat /etc/passwd"}
	err = ValidateStatusInput(input)
	assert.Error(t, err)
}

func TestValidateCaptureInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *CaptureInput
		wantErr bool
	}{
		{
			name:    "empty provider allowed",
			input:   &CaptureInput{},
			wantErr: false,
		},
		{
			name:    "valid provider",
			input:   &CaptureInput{Provider: "brew"},
			wantErr: false,
		},
		{
			name:    "valid provider with hyphen",
			input:   &CaptureInput{Provider: "my-provider"},
			wantErr: false,
		},
		{
			name:    "invalid provider with semicolon",
			input:   &CaptureInput{Provider: "brew;rm"},
			wantErr: true,
		},
		{
			name:    "invalid provider with space",
			input:   &CaptureInput{Provider: "brew apt"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCaptureInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDiffInput(t *testing.T) {
	t.Parallel()

	input := &DiffInput{ConfigPath: "preflight.yaml", Target: "work"}
	err := ValidateDiffInput(input)
	assert.NoError(t, err)

	input = &DiffInput{ConfigPath: "config&& cat /etc/passwd"}
	err = ValidateDiffInput(input)
	assert.Error(t, err)
}

func TestValidateRollbackInput(t *testing.T) {
	t.Parallel()

	input := &RollbackInput{SnapshotID: "snap-123"}
	err := ValidateRollbackInput(input)
	assert.NoError(t, err)

	input = &RollbackInput{SnapshotID: "snap;rm -rf /"}
	err = ValidateRollbackInput(input)
	assert.Error(t, err)
}

func TestValidateSyncInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *SyncInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid minimal",
			input:   &SyncInput{ConfigPath: "preflight.yaml", Target: "work"},
			wantErr: false,
		},
		{
			name:    "valid with remote and branch",
			input:   &SyncInput{ConfigPath: "preflight.yaml", Remote: "origin", Branch: "main"},
			wantErr: false,
		},
		{
			name:    "invalid config path",
			input:   &SyncInput{ConfigPath: "config`id`.yaml"},
			wantErr: true,
			errMsg:  "invalid config_path",
		},
		{
			name:    "invalid remote with semicolon",
			input:   &SyncInput{ConfigPath: "preflight.yaml", Remote: "origin;rm"},
			wantErr: true,
			errMsg:  "invalid remote",
		},
		{
			name:    "invalid branch with pipe",
			input:   &SyncInput{ConfigPath: "preflight.yaml", Branch: "main|cat"},
			wantErr: true,
			errMsg:  "invalid branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSyncInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateToolAnalyzeInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *ToolAnalyzeInput
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid single tool",
			input:   &ToolAnalyzeInput{Tools: []string{"trivy"}},
			wantErr: false,
		},
		{
			name:    "valid multiple tools",
			input:   &ToolAnalyzeInput{Tools: []string{"trivy", "grype", "golint"}},
			wantErr: false,
		},
		{
			name:    "empty tools list",
			input:   &ToolAnalyzeInput{Tools: []string{}},
			wantErr: true,
			errMsg:  "tools list is required",
		},
		{
			name:    "nil tools list",
			input:   &ToolAnalyzeInput{},
			wantErr: true,
			errMsg:  "tools list is required",
		},
		{
			name:    "invalid tool name with semicolon",
			input:   &ToolAnalyzeInput{Tools: []string{"trivy;rm"}},
			wantErr: true,
			errMsg:  "invalid tool name",
		},
		{
			name:    "invalid tool name with spaces",
			input:   &ToolAnalyzeInput{Tools: []string{"trivy grype"}},
			wantErr: true,
			errMsg:  "invalid tool name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateToolAnalyzeInput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
