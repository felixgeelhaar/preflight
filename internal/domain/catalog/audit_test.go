package catalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditor_Audit(t *testing.T) {
	t.Parallel()

	auditor := NewAuditor()

	t.Run("clean catalog passes", func(t *testing.T) {
		t.Parallel()

		cat := NewCatalog()
		id, _ := ParsePresetID("nvim:clean")
		meta, _ := NewMetadata("Clean Setup", "A safe preset")
		preset, _ := NewPreset(id, meta, DifficultyBeginner, map[string]interface{}{
			"plugins": []string{"treesitter", "telescope"},
		})
		_ = cat.AddPreset(preset)

		src := NewBuiltinSource()
		manifest, _ := NewManifestBuilder("test").
			WithAuthor("Test Author").
			WithRepository("https://github.com/test/catalog").
			WithLicense("MIT").
			Build()
		rc := NewRegisteredCatalog(src, manifest, cat)

		result := auditor.Audit(rc)
		assert.True(t, result.Passed)
		assert.Equal(t, 0, result.CriticalCount())
		assert.Equal(t, 0, result.HighCount())
	})

	t.Run("curl pipe shell is critical", func(t *testing.T) {
		t.Parallel()

		cat := NewCatalog()
		id, _ := ParsePresetID("shell:dangerous")
		meta, _ := NewMetadata("Dangerous Setup", "Don't use this")
		preset, _ := NewPreset(id, meta, DifficultyBeginner, map[string]interface{}{
			"install": "curl -fsSL https://example.com/script.sh | sh",
		})
		_ = cat.AddPreset(preset)

		manifest, _ := NewManifestBuilder("test").WithAuthor("Test").Build()
		rc := NewRegisteredCatalog(NewBuiltinSource(), manifest, cat)

		result := auditor.Audit(rc)
		assert.False(t, result.Passed)
		assert.Equal(t, 1, result.CriticalCount())
	})

	t.Run("sudo is high severity", func(t *testing.T) {
		t.Parallel()

		cat := NewCatalog()
		id, _ := ParsePresetID("shell:sudo")
		meta, _ := NewMetadata("Sudo Setup", "Needs privileges")
		preset, _ := NewPreset(id, meta, DifficultyBeginner, map[string]interface{}{
			"command": "sudo apt-get install vim",
		})
		_ = cat.AddPreset(preset)

		manifest, _ := NewManifestBuilder("test").WithAuthor("Test").Build()
		rc := NewRegisteredCatalog(NewBuiltinSource(), manifest, cat)

		result := auditor.Audit(rc)
		assert.False(t, result.Passed)
		assert.Equal(t, 1, result.HighCount())
	})

	t.Run("chmod 777 is high severity", func(t *testing.T) {
		t.Parallel()

		cat := NewCatalog()
		id, _ := ParsePresetID("files:insecure")
		meta, _ := NewMetadata("Insecure Permissions", "World-writable")
		preset, _ := NewPreset(id, meta, DifficultyBeginner, map[string]interface{}{
			"setup": "chmod 777 /tmp/shared",
		})
		_ = cat.AddPreset(preset)

		manifest, _ := NewManifestBuilder("test").WithAuthor("Test").Build()
		rc := NewRegisteredCatalog(NewBuiltinSource(), manifest, cat)

		result := auditor.Audit(rc)
		assert.False(t, result.Passed)
		assert.Equal(t, 1, result.HighCount())
	})

	t.Run("private key is high severity", func(t *testing.T) {
		t.Parallel()

		cat := NewCatalog()
		id, _ := ParsePresetID("ssh:leaked")
		meta, _ := NewMetadata("Leaked Key", "Contains private key")
		preset, _ := NewPreset(id, meta, DifficultyBeginner, map[string]interface{}{
			"key": "-----BEGIN RSA PRIVATE KEY-----\nMIIE....",
		})
		_ = cat.AddPreset(preset)

		manifest, _ := NewManifestBuilder("test").WithAuthor("Test").Build()
		rc := NewRegisteredCatalog(NewBuiltinSource(), manifest, cat)

		result := auditor.Audit(rc)
		assert.False(t, result.Passed)
		assert.GreaterOrEqual(t, result.HighCount(), 1)
	})

	t.Run("missing metadata generates info findings", func(t *testing.T) {
		t.Parallel()

		cat := NewCatalog()
		manifest, _ := NewManifestBuilder("test").Build()
		rc := NewRegisteredCatalog(NewBuiltinSource(), manifest, cat)

		result := auditor.Audit(rc)
		// Should have info findings for missing author, repo, license
		infoCount := 0
		for _, f := range result.Findings {
			if f.Severity == AuditSeverityInfo {
				infoCount++
			}
		}
		assert.GreaterOrEqual(t, infoCount, 1)
	})

	t.Run("hardcoded user path is low severity", func(t *testing.T) {
		t.Parallel()

		cat := NewCatalog()
		id, _ := ParsePresetID("files:hardcoded")
		meta, _ := NewMetadata("Hardcoded Paths", "Not portable")
		preset, _ := NewPreset(id, meta, DifficultyBeginner, map[string]interface{}{
			"path": "/Users/johndoe/.config",
		})
		_ = cat.AddPreset(preset)

		manifest, _ := NewManifestBuilder("test").WithAuthor("Test").Build()
		rc := NewRegisteredCatalog(NewBuiltinSource(), manifest, cat)

		result := auditor.Audit(rc)
		assert.True(t, result.Passed) // Low severity doesn't fail
		assert.GreaterOrEqual(t, result.LowCount(), 1)
	})
}

func TestAuditResult_Counts(t *testing.T) {
	t.Parallel()

	result := AuditResult{
		Findings: []AuditFinding{
			{Severity: AuditSeverityCritical},
			{Severity: AuditSeverityCritical},
			{Severity: AuditSeverityHigh},
			{Severity: AuditSeverityMedium},
			{Severity: AuditSeverityMedium},
			{Severity: AuditSeverityMedium},
			{Severity: AuditSeverityLow},
		},
	}

	assert.Equal(t, 2, result.CriticalCount())
	assert.Equal(t, 1, result.HighCount())
	assert.Equal(t, 3, result.MediumCount())
	assert.Equal(t, 1, result.LowCount())
}
