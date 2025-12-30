package audit_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).Build()

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, audit.EventCatalogInstalled, event.Type)
	assert.Equal(t, audit.SeverityInfo, event.Severity)
	assert.True(t, event.Success)
	assert.WithinDuration(t, time.Now(), event.Timestamp, time.Second)
}

func TestEventBuilder_WithSeverity(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventSandboxViolation).
		WithSeverity(audit.SeverityCritical).
		Build()

	assert.Equal(t, audit.SeverityCritical, event.Severity)
}

func TestEventBuilder_WithUser(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventPluginInstalled).
		WithUser("testuser").
		Build()

	assert.Equal(t, "testuser", event.User)
}

func TestEventBuilder_WithCatalog(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithCatalog("my-catalog").
		Build()

	assert.Equal(t, "my-catalog", event.Catalog)
}

func TestEventBuilder_WithPlugin(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventPluginInstalled).
		WithPlugin("my-plugin").
		Build()

	assert.Equal(t, "my-plugin", event.Plugin)
}

func TestEventBuilder_WithSource(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithSource("https://example.com/catalog.yaml").
		Build()

	assert.Equal(t, "https://example.com/catalog.yaml", event.Source)
}

func TestEventBuilder_WithIntegrity(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithIntegrity("sha256:abc123").
		Build()

	assert.Equal(t, "sha256:abc123", event.Integrity)
}

func TestEventBuilder_WithSignature(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithSignature(true, "user@example.com").
		Build()

	assert.True(t, event.SignatureVerified)
	assert.Equal(t, "user@example.com", event.Signer)
}

func TestEventBuilder_WithCapabilities(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCapabilityGranted).
		WithCapabilitiesGranted([]string{"files:read", "shell:execute"}).
		WithCapabilitiesDenied([]string{"network:fetch"}).
		Build()

	assert.Equal(t, []string{"files:read", "shell:execute"}, event.CapabilitiesGranted)
	assert.Equal(t, []string{"network:fetch"}, event.CapabilitiesDenied)
}

func TestEventBuilder_WithTrustLevel(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithTrustLevel("verified").
		Build()

	assert.Equal(t, "verified", event.TrustLevel)
}

func TestEventBuilder_WithSandboxMode(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventPluginExecuted).
		WithSandboxMode("restricted").
		Build()

	assert.Equal(t, "restricted", event.SandboxMode)
}

func TestEventBuilder_WithDuration(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventPluginExecuted).
		WithDuration(500 * time.Millisecond).
		Build()

	assert.Equal(t, 500*time.Millisecond, event.Duration)
}

func TestEventBuilder_WithError(t *testing.T) {
	t.Parallel()

	err := assert.AnError

	event := audit.NewEvent(audit.EventPluginExecuted).
		WithError(err).
		Build()

	assert.False(t, event.Success)
	assert.Equal(t, err.Error(), event.Error)
}

func TestEventBuilder_WithError_Nil(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventPluginExecuted).
		WithError(nil).
		Build()

	assert.True(t, event.Success)
	assert.Empty(t, event.Error)
}

func TestEventBuilder_WithDetails(t *testing.T) {
	t.Parallel()

	details := map[string]interface{}{
		"version": "1.0.0",
		"count":   42,
	}

	event := audit.NewEvent(audit.EventPluginInstalled).
		WithDetails(details).
		Build()

	assert.Equal(t, details, event.Details)
}

func TestEventBuilder_AddDetail(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventPluginInstalled).
		AddDetail("key1", "value1").
		AddDetail("key2", 123).
		Build()

	assert.Equal(t, "value1", event.Details["key1"])
	assert.Equal(t, 123, event.Details["key2"])
}

func TestEventBuilder_Chaining(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithUser("admin").
		WithCatalog("company-tools").
		WithSource("https://company.com/catalog.yaml").
		WithIntegrity("sha256:abc123").
		WithSignature(true, "devops@company.com").
		WithTrustLevel("verified").
		WithCapabilitiesGranted([]string{"files:write", "packages:brew"}).
		AddDetail("presets_count", 10).
		Build()

	assert.Equal(t, "admin", event.User)
	assert.Equal(t, "company-tools", event.Catalog)
	assert.Equal(t, "https://company.com/catalog.yaml", event.Source)
	assert.Equal(t, "sha256:abc123", event.Integrity)
	assert.True(t, event.SignatureVerified)
	assert.Equal(t, "devops@company.com", event.Signer)
	assert.Equal(t, "verified", event.TrustLevel)
	assert.Equal(t, []string{"files:write", "packages:brew"}, event.CapabilitiesGranted)
	assert.Equal(t, 10, event.Details["presets_count"])
}

func TestEvent_MarshalJSON(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventPluginExecuted).
		WithPlugin("test-plugin").
		WithDuration(1500 * time.Millisecond).
		Build()

	data, err := json.Marshal(event)
	require.NoError(t, err)

	// Check that duration_ms is present
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.InDelta(t, float64(1500), result["duration_ms"], 0.001)
}

func TestEventType_Constants(t *testing.T) {
	t.Parallel()

	// Catalog events
	assert.Equal(t, audit.EventCatalogInstalled, audit.EventType("catalog_installed"))
	assert.Equal(t, audit.EventCatalogRemoved, audit.EventType("catalog_removed"))
	assert.Equal(t, audit.EventCatalogVerified, audit.EventType("catalog_verified"))
	assert.Equal(t, audit.EventCatalogUpdated, audit.EventType("catalog_updated"))

	// Plugin events
	assert.Equal(t, audit.EventPluginInstalled, audit.EventType("plugin_installed"))
	assert.Equal(t, audit.EventPluginUninstalled, audit.EventType("plugin_uninstalled"))
	assert.Equal(t, audit.EventPluginDiscovered, audit.EventType("plugin_discovered"))
	assert.Equal(t, audit.EventPluginExecuted, audit.EventType("plugin_executed"))

	// Trust events
	assert.Equal(t, audit.EventTrustAdded, audit.EventType("trust_added"))
	assert.Equal(t, audit.EventTrustRemoved, audit.EventType("trust_removed"))
	assert.Equal(t, audit.EventSignatureVerified, audit.EventType("signature_verified"))
	assert.Equal(t, audit.EventSignatureFailed, audit.EventType("signature_failed"))

	// Security events
	assert.Equal(t, audit.EventCapabilityGranted, audit.EventType("capability_granted"))
	assert.Equal(t, audit.EventCapabilityDenied, audit.EventType("capability_denied"))
	assert.Equal(t, audit.EventSandboxViolation, audit.EventType("sandbox_violation"))
	assert.Equal(t, audit.EventSecurityAudit, audit.EventType("security_audit"))
}

func TestSeverity_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, audit.SeverityInfo, audit.Severity("info"))
	assert.Equal(t, audit.SeverityWarning, audit.Severity("warning"))
	assert.Equal(t, audit.SeverityError, audit.Severity("error"))
	assert.Equal(t, audit.SeverityCritical, audit.Severity("critical"))
}

func TestEvent_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid event", func(t *testing.T) {
		t.Parallel()
		event := audit.NewEvent(audit.EventCatalogInstalled).Build()
		assert.NoError(t, event.Validate())
	})

	t.Run("missing ID", func(t *testing.T) {
		t.Parallel()
		event := audit.Event{
			Type:      audit.EventCatalogInstalled,
			Timestamp: time.Now(),
			Severity:  audit.SeverityInfo,
		}
		assert.Error(t, event.Validate())
		assert.Contains(t, event.Validate().Error(), "ID")
	})

	t.Run("missing type", func(t *testing.T) {
		t.Parallel()
		event := audit.Event{
			ID:        "test-id",
			Timestamp: time.Now(),
			Severity:  audit.SeverityInfo,
		}
		assert.Error(t, event.Validate())
		assert.Contains(t, event.Validate().Error(), "type")
	})

	t.Run("missing timestamp", func(t *testing.T) {
		t.Parallel()
		event := audit.Event{
			ID:       "test-id",
			Type:     audit.EventCatalogInstalled,
			Severity: audit.SeverityInfo,
		}
		assert.Error(t, event.Validate())
		assert.Contains(t, event.Validate().Error(), "timestamp")
	})

	t.Run("missing severity", func(t *testing.T) {
		t.Parallel()
		event := audit.Event{
			ID:        "test-id",
			Type:      audit.EventCatalogInstalled,
			Timestamp: time.Now(),
		}
		assert.Error(t, event.Validate())
		assert.Contains(t, event.Validate().Error(), "severity")
	})
}

func TestEventBuilder_ValidatedBuild(t *testing.T) {
	t.Parallel()

	t.Run("valid event", func(t *testing.T) {
		t.Parallel()
		event, err := audit.NewEvent(audit.EventCatalogInstalled).ValidatedBuild()
		require.NoError(t, err)
		assert.NotEmpty(t, event.ID)
	})
}

func TestEvent_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	original := audit.NewEvent(audit.EventPluginExecuted).
		WithPlugin("test-plugin").
		WithDuration(1500 * time.Millisecond).
		Build()

	// Marshal
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var restored audit.Event
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err)

	// Check duration was restored correctly
	assert.Equal(t, 1500*time.Millisecond, restored.Duration)
	assert.Equal(t, original.Plugin, restored.Plugin)
	assert.Equal(t, original.Type, restored.Type)
}

func TestEvent_ComputeHash(t *testing.T) {
	t.Parallel()

	event := audit.NewEvent(audit.EventCatalogInstalled).
		WithCatalog("test-catalog").
		Build()

	hash := event.ComputeHash()
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 hex is 64 chars

	// Same event should produce same hash
	hash2 := event.ComputeHash()
	assert.Equal(t, hash, hash2)
}

func TestEvent_VerifyHash(t *testing.T) {
	t.Parallel()

	t.Run("no hash set", func(t *testing.T) {
		t.Parallel()
		event := audit.NewEvent(audit.EventCatalogInstalled).Build()
		assert.True(t, event.VerifyHash())
	})

	t.Run("valid hash", func(t *testing.T) {
		t.Parallel()
		event := audit.NewEvent(audit.EventCatalogInstalled).Build()
		event.EventHash = event.ComputeHash()
		assert.True(t, event.VerifyHash())
	})

	t.Run("tampered event", func(t *testing.T) {
		t.Parallel()
		event := audit.NewEvent(audit.EventCatalogInstalled).Build()
		event.EventHash = event.ComputeHash()

		// Tamper with the event
		event.Catalog = "tampered"
		assert.False(t, event.VerifyHash())
	})
}
