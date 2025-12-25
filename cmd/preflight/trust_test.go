package main

import (
	"testing"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/stretchr/testify/assert"
)

func TestTrustCmd_Exists(t *testing.T) {
	t.Parallel()

	// Verify trust command exists
	assert.NotNil(t, trustCmd)
	assert.Equal(t, "trust", trustCmd.Use)

	// Verify subcommands exist
	subcommands := trustCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "list")
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "show")
}

func TestDetectKeyType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		data   []byte
		expect catalog.SignatureType
	}{
		{
			name:   "ssh ed25519",
			data:   []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5 user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "ssh rsa",
			data:   []byte("ssh-rsa AAAAB3NzaC1yc2E user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "ecdsa",
			data:   []byte("ecdsa-sha2-nistp256 AAAA user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "gpg armored",
			data:   []byte("-----BEGIN PGP PUBLIC KEY-----\nVersion: 1\n"),
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "gpg binary",
			data:   []byte{0x99, 0x01, 0x0d}, // Old format packet
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "unknown",
			data:   []byte("some random data"),
			expect: "",
		},
		{
			name:   "empty",
			data:   []byte{},
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := detectKeyType(tt.data)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestTrustListCmd_HasAliases(t *testing.T) {
	t.Parallel()

	assert.Contains(t, trustListCmd.Aliases, "ls")
}

func TestTrustRemoveCmd_HasAliases(t *testing.T) {
	t.Parallel()

	assert.Contains(t, trustRemoveCmd.Aliases, "rm")
}
