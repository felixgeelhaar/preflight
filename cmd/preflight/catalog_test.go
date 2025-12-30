package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/stretchr/testify/assert"
)

func TestDeriveCatalogName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location string
		want     string
	}{
		{"url with path", "https://example.com/my-catalog", "my-catalog"},
		{"url with trailing slash", "https://example.com/catalog/", "catalog"},
		{"local path", "/path/to/presets", "presets"},
		{"relative path", "./local-catalog", "local-catalog"},
		{"just name", "catalog", "catalog"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := deriveCatalogName(tt.location)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCatalogCmd_Exists(t *testing.T) {
	t.Parallel()

	// Verify catalog command exists
	assert.NotNil(t, catalogCmd)
	assert.Equal(t, "catalog", catalogCmd.Use)

	// Verify subcommands exist
	subcommands := catalogCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "add")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "verify")
	assert.Contains(t, names, "audit")
}

func TestFilterBySeverity(t *testing.T) {
	t.Parallel()

	findings := []catalog.AuditFinding{
		{Severity: catalog.AuditSeverityCritical, Message: "critical 1"},
		{Severity: catalog.AuditSeverityCritical, Message: "critical 2"},
		{Severity: catalog.AuditSeverityHigh, Message: "high 1"},
		{Severity: catalog.AuditSeverityMedium, Message: "medium 1"},
		{Severity: catalog.AuditSeverityLow, Message: "low 1"},
		{Severity: catalog.AuditSeverityLow, Message: "low 2"},
		{Severity: catalog.AuditSeverityLow, Message: "low 3"},
	}

	tests := []struct {
		name     string
		severity catalog.AuditSeverity
		expected int
	}{
		{"filter critical", catalog.AuditSeverityCritical, 2},
		{"filter high", catalog.AuditSeverityHigh, 1},
		{"filter medium", catalog.AuditSeverityMedium, 1},
		{"filter low", catalog.AuditSeverityLow, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := filterBySeverity(findings, tt.severity)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestFilterBySeverity_Empty(t *testing.T) {
	t.Parallel()

	result := filterBySeverity(nil, catalog.AuditSeverityCritical)
	assert.Empty(t, result)
}

func TestVerifyCatalogSignatures_NoSignature(t *testing.T) {
	t.Parallel()

	// verifyCatalogSignatures currently returns no signature for all catalogs
	// as the feature is a placeholder until catalog authors add signing
	result := verifyCatalogSignatures(nil, nil)
	assert.False(t, result.hasSignature)
	assert.Empty(t, result.signer)
	assert.Empty(t, result.issuer)
	assert.NoError(t, result.err)
}

func TestSignatureVerifyResult_Fields(t *testing.T) {
	t.Parallel()

	result := signatureVerifyResult{
		hasSignature: true,
		signer:       "test@example.com",
		issuer:       "https://accounts.google.com",
		err:          nil,
	}

	assert.True(t, result.hasSignature)
	assert.Equal(t, "test@example.com", result.signer)
	assert.Equal(t, "https://accounts.google.com", result.issuer)
	assert.NoError(t, result.err)
}
