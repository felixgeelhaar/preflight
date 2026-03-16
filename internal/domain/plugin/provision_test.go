package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeProvisioner_Constant(t *testing.T) {
	t.Parallel()

	assert.Equal(t, TypeProvisioner, PluginType("provisioner"))
}

func TestProvisionAction_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ProvisionActionPlan, ProvisionAction("plan"))
	assert.Equal(t, ProvisionActionApply, ProvisionAction("apply"))
	assert.Equal(t, ProvisionActionDestroy, ProvisionAction("destroy"))
	assert.Equal(t, ProvisionActionState, ProvisionAction("state"))
}

func TestValidProvisionActions(t *testing.T) {
	t.Parallel()

	actions := ValidProvisionActions()
	require.Len(t, actions, 4)

	// Verify sorted order for deterministic output
	assert.Equal(t, ProvisionActionApply, actions[0])
	assert.Equal(t, ProvisionActionDestroy, actions[1])
	assert.Equal(t, ProvisionActionPlan, actions[2])
	assert.Equal(t, ProvisionActionState, actions[3])
}

func TestIsValidProvisionAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action string
		valid  bool
	}{
		{name: "plan is valid", action: "plan", valid: true},
		{name: "apply is valid", action: "apply", valid: true},
		{name: "destroy is valid", action: "destroy", valid: true},
		{name: "state is valid", action: "state", valid: true},
		{name: "empty is invalid", action: "", valid: false},
		{name: "unknown is invalid", action: "rollback", valid: false},
		{name: "uppercase is invalid", action: "Plan", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, IsValidProvisionAction(tt.action))
		})
	}
}

func TestValidateProvisionRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *ProvisionRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: &ProvisionRequest{
				PluginName: "terraform",
				Action:     ProvisionActionPlan,
				WorkDir:    "/tmp/infra",
			},
			wantErr: false,
		},
		{
			name: "valid request with variables",
			req: &ProvisionRequest{
				PluginName: "ansible",
				Action:     ProvisionActionApply,
				WorkDir:    "/tmp/playbooks",
				Variables:  map[string]string{"env": "production"},
				DryRun:     true,
			},
			wantErr: false,
		},
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errMsg:  "provision request cannot be nil",
		},
		{
			name: "missing plugin name",
			req: &ProvisionRequest{
				Action:  ProvisionActionPlan,
				WorkDir: "/tmp/infra",
			},
			wantErr: true,
			errMsg:  "plugin name is required",
		},
		{
			name: "missing action",
			req: &ProvisionRequest{
				PluginName: "terraform",
				WorkDir:    "/tmp/infra",
			},
			wantErr: true,
			errMsg:  "action is required",
		},
		{
			name: "invalid action",
			req: &ProvisionRequest{
				PluginName: "terraform",
				Action:     ProvisionAction("rollback"),
				WorkDir:    "/tmp/infra",
			},
			wantErr: true,
			errMsg:  "invalid provision action",
		},
		{
			name: "missing work dir",
			req: &ProvisionRequest{
				PluginName: "terraform",
				Action:     ProvisionActionPlan,
			},
			wantErr: true,
			errMsg:  "work directory is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateProvisionRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProvisionerCapability_Fields(t *testing.T) {
	t.Parallel()

	provCap := ProvisionerCapability{
		Name:             "terraform",
		Description:      "Terraform IaC provisioner",
		SupportedActions: []string{"plan", "apply", "destroy", "state"},
	}

	assert.Equal(t, "terraform", provCap.Name)
	assert.Equal(t, "Terraform IaC provisioner", provCap.Description)
	assert.Len(t, provCap.SupportedActions, 4)
}

func TestProvisionResult_Fields(t *testing.T) {
	t.Parallel()

	result := ProvisionResult{
		Action:  ProvisionActionApply,
		Success: true,
		Output:  "Apply complete! Resources: 2 added, 0 changed, 0 destroyed.",
		Changes: []ProvisionChange{
			{
				Resource: "aws_instance.web",
				Action:   "create",
				After:    "t2.micro",
			},
			{
				Resource: "aws_s3_bucket.data",
				Action:   "update",
				Before:   "private",
				After:    "public-read",
			},
		},
	}

	assert.Equal(t, ProvisionActionApply, result.Action)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Output)
	require.Len(t, result.Changes, 2)
	assert.Equal(t, "create", result.Changes[0].Action)
	assert.Equal(t, "update", result.Changes[1].Action)
}

func TestProvisionChange_Fields(t *testing.T) {
	t.Parallel()

	change := ProvisionChange{
		Resource: "aws_instance.web",
		Action:   "delete",
		Before:   "running",
		After:    "",
	}

	assert.Equal(t, "aws_instance.web", change.Resource)
	assert.Equal(t, "delete", change.Action)
	assert.Equal(t, "running", change.Before)
	assert.Empty(t, change.After)
}

func TestManifest_IsProvisionerPlugin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest Manifest
		expected bool
	}{
		{
			name:     "empty type is not provisioner",
			manifest: Manifest{},
			expected: false,
		},
		{
			name:     "config type is not provisioner",
			manifest: Manifest{Type: TypeConfig},
			expected: false,
		},
		{
			name:     "provider type is not provisioner",
			manifest: Manifest{Type: TypeProvider},
			expected: false,
		},
		{
			name:     "provisioner type is provisioner",
			manifest: Manifest{Type: TypeProvisioner},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.manifest.IsProvisionerPlugin())
		})
	}
}

func TestValidateManifest_Provisioner(t *testing.T) {
	t.Parallel()

	t.Run("valid provisioner manifest", func(t *testing.T) {
		t.Parallel()

		m := &Manifest{
			APIVersion: "v1",
			Type:       TypeProvisioner,
			Name:       "terraform",
			Version:    "1.0.0",
			Provides: Capabilities{
				Provisioners: []ProvisionerCapability{
					{
						Name:             "terraform",
						SupportedActions: []string{"plan", "apply"},
					},
				},
			},
			WASM: &WASMConfig{
				Module:   "plugin.wasm",
				Checksum: "sha256:abc123",
			},
		}

		err := ValidateManifest(m)
		require.NoError(t, err)
	})

	t.Run("provisioner requires wasm config", func(t *testing.T) {
		t.Parallel()

		m := &Manifest{
			APIVersion: "v1",
			Type:       TypeProvisioner,
			Name:       "terraform",
			Version:    "1.0.0",
			Provides: Capabilities{
				Provisioners: []ProvisionerCapability{
					{Name: "terraform", SupportedActions: []string{"plan"}},
				},
			},
		}

		err := ValidateManifest(m)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "wasm")
	})

	t.Run("provisioner requires at least one provisioner capability", func(t *testing.T) {
		t.Parallel()

		m := &Manifest{
			APIVersion: "v1",
			Type:       TypeProvisioner,
			Name:       "terraform",
			Version:    "1.0.0",
			Provides:   Capabilities{},
			WASM: &WASMConfig{
				Module:   "plugin.wasm",
				Checksum: "sha256:abc123",
			},
		}

		err := ValidateManifest(m)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provisioner")
	})

	t.Run("provisioner capability requires name", func(t *testing.T) {
		t.Parallel()

		m := &Manifest{
			APIVersion: "v1",
			Type:       TypeProvisioner,
			Name:       "terraform",
			Version:    "1.0.0",
			Provides: Capabilities{
				Provisioners: []ProvisionerCapability{
					{Name: "", SupportedActions: []string{"plan"}},
				},
			},
			WASM: &WASMConfig{
				Module:   "plugin.wasm",
				Checksum: "sha256:abc123",
			},
		}

		err := ValidateManifest(m)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("provisioner capability requires supported actions", func(t *testing.T) {
		t.Parallel()

		m := &Manifest{
			APIVersion: "v1",
			Type:       TypeProvisioner,
			Name:       "terraform",
			Version:    "1.0.0",
			Provides: Capabilities{
				Provisioners: []ProvisionerCapability{
					{Name: "terraform", SupportedActions: nil},
				},
			},
			WASM: &WASMConfig{
				Module:   "plugin.wasm",
				Checksum: "sha256:abc123",
			},
		}

		err := ValidateManifest(m)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "supportedActions")
	})
}

func TestCapabilities_Clone_WithProvisioners(t *testing.T) {
	t.Parallel()

	original := Capabilities{
		Providers: []ProviderSpec{
			{Name: "docker", ConfigKey: "docker"},
		},
		Provisioners: []ProvisionerCapability{
			{
				Name:             "terraform",
				Description:      "Terraform provisioner",
				SupportedActions: []string{"plan", "apply"},
			},
		},
		Presets:         []string{"preset1"},
		CapabilityPacks: []string{"pack1"},
	}

	cloned := original.Clone()

	// Verify deep copy
	assert.Equal(t, original, cloned)

	// Mutate original and verify clone is independent
	original.Provisioners[0].Name = "mutated"
	original.Provisioners[0].SupportedActions[0] = "mutated"
	assert.Equal(t, "terraform", cloned.Provisioners[0].Name)
	assert.Equal(t, "plan", cloned.Provisioners[0].SupportedActions[0])
}

func TestCapabilities_Clone_NilProvisioners(t *testing.T) {
	t.Parallel()

	original := Capabilities{
		Provisioners: nil,
	}

	cloned := original.Clone()
	assert.Nil(t, cloned.Provisioners)
}
