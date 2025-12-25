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
		// SSH key types
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
			name:   "ssh dss",
			data:   []byte("ssh-dss AAAAB3NzaC1kc3M user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "ecdsa nistp256",
			data:   []byte("ecdsa-sha2-nistp256 AAAA user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "ecdsa nistp384",
			data:   []byte("ecdsa-sha2-nistp384 AAAA user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "ecdsa nistp521",
			data:   []byte("ecdsa-sha2-nistp521 AAAA user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "sk-ssh-ed25519 (FIDO)",
			data:   []byte("sk-ssh-ed25519@openssh.com AAAA user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		{
			name:   "sk-ecdsa (FIDO)",
			data:   []byte("sk-ecdsa-sha2-nistp256@openssh.com AAAA user@host"),
			expect: catalog.SignatureTypeSSH,
		},
		// GPG armored formats
		{
			name:   "gpg armored public key",
			data:   []byte("-----BEGIN PGP PUBLIC KEY-----\nVersion: 1\n"),
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "gpg armored public key block",
			data:   []byte("-----BEGIN PGP PUBLIC KEY BLOCK-----\nVersion: 1\n"),
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "gpg armored private key",
			data:   []byte("-----BEGIN PGP PRIVATE KEY BLOCK-----\n"),
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "gpg armored signature",
			data:   []byte("-----BEGIN PGP SIGNATURE-----\n"),
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "gpg armored message",
			data:   []byte("-----BEGIN PGP MESSAGE-----\n"),
			expect: catalog.SignatureTypeGPG,
		},
		// GPG binary formats (OpenPGP packets)
		{
			name:   "gpg binary old format public key",
			data:   []byte{0x99, 0x01, 0x0d}, // Old format, tag 6 (public key)
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "gpg binary new format public key",
			data:   []byte{0xc6, 0x01, 0x0d}, // New format, tag 6 (public key)
			expect: catalog.SignatureTypeGPG,
		},
		{
			name:   "gpg binary signature packet",
			data:   []byte{0xc2, 0x01, 0x0d}, // New format, tag 2 (signature)
			expect: catalog.SignatureTypeGPG,
		},
		// Edge cases - should NOT match
		{
			name:   "empty",
			data:   []byte{},
			expect: "",
		},
		{
			name:   "unknown text",
			data:   []byte("some random data"),
			expect: "",
		},
		{
			name:   "binary with high bit but invalid tag",
			data:   []byte{0x80, 0x01, 0x0d}, // High bit set but tag 0 (reserved)
			expect: "",
		},
		{
			name:   "almost ssh prefix",
			data:   []byte("ssh_ed25519 AAAA"), // underscore instead of dash
			expect: "",
		},
		{
			name:   "almost gpg armor",
			data:   []byte("----BEGIN PGP PUBLIC KEY-----"), // 4 dashes instead of 5
			expect: "",
		},
		{
			name:   "random binary",
			data:   []byte{0x50, 0x4b, 0x03, 0x04}, // ZIP file header
			expect: "",
		},
		{
			name:   "single byte",
			data:   []byte{0x99},
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

func TestIsValidOpenPGPPacket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		data   []byte
		expect bool
	}{
		// Valid packets
		{
			name:   "old format public key (tag 6)",
			data:   []byte{0x99, 0x01}, // 10011001 -> tag = (0x18 >> 2) = 6
			expect: true,
		},
		{
			name:   "new format public key (tag 6)",
			data:   []byte{0xc6, 0x01}, // 11000110 -> tag = 6
			expect: true,
		},
		{
			name:   "new format signature (tag 2)",
			data:   []byte{0xc2, 0x01},
			expect: true,
		},
		{
			name:   "new format secret key (tag 5)",
			data:   []byte{0xc5, 0x01},
			expect: true,
		},
		{
			name:   "new format public subkey (tag 14)",
			data:   []byte{0xce, 0x01},
			expect: true,
		},
		// Invalid packets
		{
			name:   "empty",
			data:   []byte{},
			expect: false,
		},
		{
			name:   "single byte",
			data:   []byte{0x99},
			expect: false,
		},
		{
			name:   "bit 7 not set",
			data:   []byte{0x40, 0x01},
			expect: false,
		},
		{
			name:   "invalid tag 0",
			data:   []byte{0xc0, 0x01}, // New format tag 0 (reserved)
			expect: false,
		},
		{
			name:   "invalid tag 1 (PKESK)",
			data:   []byte{0xc1, 0x01}, // Tag 1 not in valid list
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isValidOpenPGPPacket(tt.data)
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
