package advisor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferWorkContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		emails        []string
		wantContext   WorkContext
		wantWorkCount int
		wantPersonal  int
		minConfidence float64
	}{
		{
			name:          "empty emails",
			emails:        []string{},
			wantContext:   WorkContextUnknown,
			minConfidence: 0,
		},
		{
			name:          "only personal email",
			emails:        []string{"user@gmail.com"},
			wantContext:   WorkContextPersonal,
			wantPersonal:  1,
			minConfidence: 0.8,
		},
		{
			name:          "only work email",
			emails:        []string{"user@acme-corp.com"},
			wantContext:   WorkContextWork,
			wantWorkCount: 1,
			minConfidence: 0.8,
		},
		{
			name:          "mixed emails",
			emails:        []string{"work@company.com", "personal@gmail.com"},
			wantContext:   WorkContextMixed,
			wantWorkCount: 1,
			wantPersonal:  1,
			minConfidence: 0.5,
		},
		{
			name:          "multiple personal emails",
			emails:        []string{"user@gmail.com", "user@icloud.com", "user@protonmail.com"},
			wantContext:   WorkContextPersonal,
			wantPersonal:  3,
			minConfidence: 0.8,
		},
		{
			name:          "multiple work emails",
			emails:        []string{"user@company.com", "user@subsidiary.org"},
			wantContext:   WorkContextWork,
			wantWorkCount: 2,
			minConfidence: 0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := InferWorkContext(tt.emails)

			assert.Equal(t, tt.wantContext, result.WorkContext)
			assert.Len(t, result.WorkDomains, tt.wantWorkCount)
			assert.Len(t, result.PersonalEmails, tt.wantPersonal)
			assert.GreaterOrEqual(t, result.Confidence, tt.minConfidence)
		})
	}
}

func TestInferWorkContextFromSSHKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		keyNames    []string
		wantContext WorkContext
	}{
		{
			name:        "empty keys",
			keyNames:    []string{},
			wantContext: WorkContextUnknown,
		},
		{
			name:        "work key",
			keyNames:    []string{"id_rsa_work", "id_ed25519_corp"},
			wantContext: WorkContextWork,
		},
		{
			name:        "personal key",
			keyNames:    []string{"id_rsa_github", "id_ed25519_personal"},
			wantContext: WorkContextPersonal,
		},
		{
			name:        "mixed keys",
			keyNames:    []string{"id_rsa_work", "id_ed25519_github"},
			wantContext: WorkContextMixed,
		},
		{
			name:        "generic keys",
			keyNames:    []string{"id_rsa", "id_ed25519"},
			wantContext: WorkContextUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := InferWorkContextFromSSHKeys(tt.keyNames)
			assert.Equal(t, tt.wantContext, result)
		})
	}
}

func TestInferWorkContextFromTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tools       []string
		wantContext WorkContext
		wantSignals int
	}{
		{
			name:        "empty tools",
			tools:       []string{},
			wantContext: WorkContextUnknown,
			wantSignals: 0,
		},
		{
			name:        "work tools",
			tools:       []string{"slack", "zoom", "okta"},
			wantContext: WorkContextWork,
			wantSignals: 3,
		},
		{
			name:        "personal tools",
			tools:       []string{"steam", "discord", "spotify"},
			wantContext: WorkContextPersonal,
			wantSignals: 3,
		},
		{
			name:        "mixed tools",
			tools:       []string{"slack", "steam"},
			wantContext: WorkContextMixed,
			wantSignals: 2,
		},
		{
			name:        "unknown tools",
			tools:       []string{"git", "neovim", "ripgrep"},
			wantContext: WorkContextUnknown,
			wantSignals: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, signals := InferWorkContextFromTools(tt.tools)
			assert.Equal(t, tt.wantContext, ctx)
			assert.Len(t, signals, tt.wantSignals)
		})
	}
}

func TestSuggestLayers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ctx        InferredContext
		deviceType DeviceType
		wantLayers []string
	}{
		{
			name:       "work context on laptop",
			ctx:        InferredContext{WorkContext: WorkContextWork},
			deviceType: DeviceTypeLaptop,
			wantLayers: []string{"base", "identity.work", "device.laptop"},
		},
		{
			name:       "personal context on desktop",
			ctx:        InferredContext{WorkContext: WorkContextPersonal},
			deviceType: DeviceTypeDesktop,
			wantLayers: []string{"base", "identity.personal", "device.desktop"},
		},
		{
			name:       "mixed context on laptop",
			ctx:        InferredContext{WorkContext: WorkContextMixed},
			deviceType: DeviceTypeLaptop,
			wantLayers: []string{"base", "identity.work", "identity.personal", "device.laptop"},
		},
		{
			name:       "unknown context and device",
			ctx:        InferredContext{WorkContext: WorkContextUnknown},
			deviceType: DeviceTypeUnknown,
			wantLayers: []string{"base"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			layers := SuggestLayers(tt.ctx, tt.deviceType)
			assert.Equal(t, tt.wantLayers, layers)
		})
	}
}

func TestExtractDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		email      string
		wantDomain string
	}{
		{"user@gmail.com", "gmail.com"},
		{"user@company.co.uk", "company.co.uk"},
		{"user@subdomain.example.org", "subdomain.example.org"},
		{"invalid-email", ""},
		{"@nodomain", ""},
		{"user@", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			t.Parallel()

			domain := extractDomain(tt.email)
			assert.Equal(t, tt.wantDomain, domain)
		})
	}
}

func TestWorkContextIsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, WorkContextUnknown.IsValid())
	assert.True(t, WorkContextWork.IsValid())
	assert.True(t, WorkContextPersonal.IsValid())
	assert.True(t, WorkContextMixed.IsValid())
	assert.False(t, WorkContext("invalid").IsValid())
}

func TestDeviceTypeIsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, DeviceTypeUnknown.IsValid())
	assert.True(t, DeviceTypeLaptop.IsValid())
	assert.True(t, DeviceTypeDesktop.IsValid())
	assert.False(t, DeviceType("invalid").IsValid())
}
