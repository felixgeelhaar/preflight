package marketplace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-package", false},
		{"valid with numbers", "package123", false},
		{"valid hyphenated", "my-cool-package", false},
		{"empty", "", true},
		{"uppercase", "MyPackage", true},
		{"underscore", "my_package", true},
		{"starts with hyphen", "-package", true},
		{"ends with hyphen", "package-", true},
		{"special chars", "package@1.0", true},
		{"spaces", "my package", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, err := NewPackageID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input, id.String())
			}
		})
	}
}

func TestPackageID_Equals(t *testing.T) {
	t.Parallel()

	id1 := MustNewPackageID("test-package")
	id2 := MustNewPackageID("test-package")
	id3 := MustNewPackageID("other-package")

	assert.True(t, id1.Equals(id2))
	assert.False(t, id1.Equals(id3))
}

func TestPackageID_IsZero(t *testing.T) {
	t.Parallel()

	var zero PackageID
	assert.True(t, zero.IsZero())

	nonZero := MustNewPackageID("test")
	assert.False(t, nonZero.IsZero())
}

func TestPackageVersion_ValidateChecksum(t *testing.T) {
	t.Parallel()

	data := []byte("test content")
	checksum := ComputeChecksum(data)

	version := PackageVersion{
		Version:  "1.0.0",
		Checksum: checksum,
	}

	// Valid checksum
	err := version.ValidateChecksum(data)
	assert.NoError(t, err)

	// Invalid checksum
	err = version.ValidateChecksum([]byte("different content"))
	assert.ErrorIs(t, err, ErrChecksumMismatch)

	// Empty checksum
	version.Checksum = ""
	err = version.ValidateChecksum(data)
	assert.ErrorIs(t, err, ErrInvalidChecksum)
}

func TestComputeChecksum(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")
	checksum := ComputeChecksum(data)

	// SHA256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	assert.Equal(t, expected, checksum)
}

func TestPackage_LatestVersion(t *testing.T) {
	t.Parallel()

	pkg := Package{
		ID: MustNewPackageID("test"),
		Versions: []PackageVersion{
			{Version: "2.0.0"},
			{Version: "1.0.0"},
		},
	}

	latest, ok := pkg.LatestVersion()
	assert.True(t, ok)
	assert.Equal(t, "2.0.0", latest.Version)

	// Empty versions
	emptyPkg := Package{ID: MustNewPackageID("empty")}
	_, ok = emptyPkg.LatestVersion()
	assert.False(t, ok)
}

func TestPackage_GetVersion(t *testing.T) {
	t.Parallel()

	pkg := Package{
		ID: MustNewPackageID("test"),
		Versions: []PackageVersion{
			{Version: "2.0.0"},
			{Version: "1.0.0"},
		},
	}

	// Existing version
	v, ok := pkg.GetVersion("1.0.0")
	assert.True(t, ok)
	assert.Equal(t, "1.0.0", v.Version)

	// Non-existing version
	_, ok = pkg.GetVersion("3.0.0")
	assert.False(t, ok)
}

func TestPackage_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		pkg   Package
		valid bool
	}{
		{
			name: "valid preset",
			pkg: Package{
				ID:       MustNewPackageID("test"),
				Type:     PackageTypePreset,
				Title:    "Test Package",
				Versions: []PackageVersion{{Version: "1.0.0"}},
			},
			valid: true,
		},
		{
			name: "valid capability pack",
			pkg: Package{
				ID:       MustNewPackageID("test"),
				Type:     PackageTypeCapabilityPack,
				Title:    "Test Pack",
				Versions: []PackageVersion{{Version: "1.0.0"}},
			},
			valid: true,
		},
		{
			name: "missing ID",
			pkg: Package{
				Type:     PackageTypePreset,
				Title:    "Test",
				Versions: []PackageVersion{{Version: "1.0.0"}},
			},
			valid: false,
		},
		{
			name: "invalid type",
			pkg: Package{
				ID:       MustNewPackageID("test"),
				Type:     "invalid",
				Title:    "Test",
				Versions: []PackageVersion{{Version: "1.0.0"}},
			},
			valid: false,
		},
		{
			name: "missing title",
			pkg: Package{
				ID:       MustNewPackageID("test"),
				Type:     PackageTypePreset,
				Versions: []PackageVersion{{Version: "1.0.0"}},
			},
			valid: false,
		},
		{
			name: "no versions",
			pkg: Package{
				ID:    MustNewPackageID("test"),
				Type:  PackageTypePreset,
				Title: "Test",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.pkg.IsValid())
		})
	}
}

func TestPackage_MatchesQuery(t *testing.T) {
	t.Parallel()

	pkg := Package{
		ID:          MustNewPackageID("nvim-config"),
		Title:       "Neovim Configuration",
		Description: "A complete Neovim setup with LSP",
		Keywords:    []string{"editor", "vim", "neovim"},
		Provenance: Provenance{
			Author: "john-doe",
		},
	}

	tests := []struct {
		query   string
		matches bool
	}{
		{"nvim", true},    // ID match
		{"neovim", true},  // Title match
		{"LSP", true},     // Description match
		{"editor", true},  // Keyword match
		{"john", true},    // Author match
		{"vscode", false}, // No match
		{"NEOVIM", true},  // Case insensitive
		{"", true},        // Empty query matches all
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.matches, pkg.MatchesQuery(tt.query))
		})
	}
}

func TestProvenance_IsZero(t *testing.T) {
	t.Parallel()

	zero := Provenance{}
	assert.True(t, zero.IsZero())

	nonZero := Provenance{Author: "test"}
	assert.False(t, nonZero.IsZero())
}

func TestInstalledPackage_NeedsUpdate(t *testing.T) {
	t.Parallel()

	pkg := Package{
		ID: MustNewPackageID("test"),
		Versions: []PackageVersion{
			{Version: "2.0.0"},
			{Version: "1.0.0"},
		},
	}

	// Needs update
	installed := InstalledPackage{
		Package:     pkg,
		Version:     "1.0.0",
		InstalledAt: time.Now(),
	}
	assert.True(t, installed.NeedsUpdate())

	// Up to date
	installed.Version = "2.0.0"
	assert.False(t, installed.NeedsUpdate())

	// No versions available
	installed.Package.Versions = nil
	assert.False(t, installed.NeedsUpdate())
}
