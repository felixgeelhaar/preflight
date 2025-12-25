package capability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCapability(t *testing.T) {
	t.Parallel()

	c := NewCapability(CategoryFiles, ActionRead)

	assert.Equal(t, CategoryFiles, c.Category())
	assert.Equal(t, ActionRead, c.Action())
	assert.Equal(t, "files:read", c.String())
	assert.False(t, c.IsZero())
}

func TestParseCapability(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantCat Category
		wantAct Action
		wantErr bool
	}{
		{"files:read", "files:read", CategoryFiles, ActionRead, false},
		{"packages:brew", "packages:brew", CategoryPackages, "brew", false},
		{"shell:execute", "shell:execute", CategoryShell, ActionExecute, false},
		{"with spaces", "  files:write  ", CategoryFiles, ActionWrite, false},
		{"empty", "", "", "", true},
		{"no colon", "filesread", "", "", true},
		{"unknown category", "unknown:read", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := ParseCapability(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidCapability)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCat, c.Category())
				assert.Equal(t, tt.wantAct, c.Action())
			}
		})
	}
}

func TestMustParseCapability(t *testing.T) {
	t.Parallel()

	c := MustParseCapability("files:read")
	assert.Equal(t, "files:read", c.String())

	assert.Panics(t, func() {
		MustParseCapability("invalid")
	})
}

func TestCapability_IsZero(t *testing.T) {
	t.Parallel()

	var c Capability
	assert.True(t, c.IsZero())

	c = NewCapability(CategoryFiles, ActionRead)
	assert.False(t, c.IsZero())
}

func TestCapability_IsDangerous(t *testing.T) {
	t.Parallel()

	tests := []struct {
		c         Capability
		dangerous bool
	}{
		{CapFilesRead, false},
		{CapFilesWrite, false},
		{CapPackagesBrew, false},
		{CapShellExecute, true},
		{CapSecretsRead, true},
		{CapSecretsWrite, true},
		{CapSystemModify, true},
		{CapNetworkFetch, false},
	}

	for _, tt := range tests {
		t.Run(tt.c.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.dangerous, tt.c.IsDangerous())
		})
	}
}

func TestCapability_Matches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		a      Capability
		b      Capability
		expect bool
	}{
		{"exact match", CapFilesRead, CapFilesRead, true},
		{"different action", CapFilesRead, CapFilesWrite, false},
		{"different category", CapFilesRead, CapPackagesBrew, false},
		{"wildcard a", MustParseCapability("files:*"), CapFilesRead, true},
		{"wildcard b", CapFilesRead, MustParseCapability("files:*"), true},
		{"wildcard different cat", MustParseCapability("files:*"), CapPackagesBrew, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, tt.a.Matches(tt.b))
		})
	}
}

func TestAllCapabilities(t *testing.T) {
	t.Parallel()

	caps := AllCapabilities()
	assert.NotEmpty(t, caps)

	// Check that dangerous flags are set correctly
	for _, info := range caps {
		assert.Equal(t, info.Capability.IsDangerous(), info.Dangerous)
	}
}

func TestDescribeCapability(t *testing.T) {
	t.Parallel()

	desc := DescribeCapability(CapFilesRead)
	assert.Contains(t, desc, "Read")

	// Unknown capability
	unknown := NewCapability(CategoryFiles, "unknown")
	desc = DescribeCapability(unknown)
	assert.Contains(t, desc, "files:unknown")
}

func TestWellKnownCapabilities(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "files:read", CapFilesRead.String())
	assert.Equal(t, "files:write", CapFilesWrite.String())
	assert.Equal(t, "packages:brew", CapPackagesBrew.String())
	assert.Equal(t, "packages:apt", CapPackagesApt.String())
	assert.Equal(t, "shell:execute", CapShellExecute.String())
	assert.Equal(t, "network:fetch", CapNetworkFetch.String())
	assert.Equal(t, "secrets:read", CapSecretsRead.String())
	assert.Equal(t, "secrets:write", CapSecretsWrite.String())
	assert.Equal(t, "system:modify", CapSystemModify.String())
}
