package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatInstallAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 2 * 24 * time.Hour, "2d ago"},
		{"weeks", 2 * 7 * 24 * time.Hour, "2w ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatInstallAge(time.Now().Add(-tt.duration))
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFormatInstallAge_OldDate(t *testing.T) {
	t.Parallel()

	oldDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	result := formatInstallAge(oldDate)
	assert.Equal(t, "2024-01-15", result)
}

func TestMarketplaceCmd_Exists(t *testing.T) {
	t.Parallel()

	// Verify marketplace command exists
	assert.NotNil(t, marketplaceCmd)
	assert.Equal(t, "marketplace", marketplaceCmd.Use)
	assert.Contains(t, marketplaceCmd.Aliases, "mp")
	assert.Contains(t, marketplaceCmd.Aliases, "market")

	// Verify subcommands exist
	subcommands := marketplaceCmd.Commands()
	names := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "search")
	assert.Contains(t, names, "install")
	assert.Contains(t, names, "uninstall")
	assert.Contains(t, names, "update")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "info")
	assert.Contains(t, names, "recommend")
	assert.Contains(t, names, "featured")
	assert.Contains(t, names, "popular")
}

func TestMarketplaceCmd_IsRegisteredOnRoot(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "marketplace" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestMarketplaceSearchCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "search [query]", marketplaceSearchCmd.Use)

	typeFlag := marketplaceSearchCmd.Flags().Lookup("type")
	require.NotNil(t, typeFlag)
	assert.Equal(t, "", typeFlag.DefValue)

	limitFlag := marketplaceSearchCmd.Flags().Lookup("limit")
	require.NotNil(t, limitFlag)
	assert.Equal(t, "20", limitFlag.DefValue)
}

func TestMarketplaceInstallCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "install <package> [version]", marketplaceInstallCmd.Use)

	versionFlag := marketplaceInstallCmd.Flags().Lookup("version")
	require.NotNil(t, versionFlag)
	assert.Equal(t, "", versionFlag.DefValue)
}

func TestMarketplaceUninstallCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "uninstall <package>", marketplaceUninstallCmd.Use)
	assert.Contains(t, marketplaceUninstallCmd.Aliases, "remove")
	assert.Contains(t, marketplaceUninstallCmd.Aliases, "rm")
}

func TestMarketplaceListCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "list", marketplaceListCmd.Use)
	assert.Contains(t, marketplaceListCmd.Aliases, "ls")

	checkFlag := marketplaceListCmd.Flags().Lookup("check-updates")
	require.NotNil(t, checkFlag)
	assert.Equal(t, "false", checkFlag.DefValue)
}

func TestMarketplaceRecommendCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "recommend", marketplaceRecommendCmd.Use)
	assert.Contains(t, marketplaceRecommendCmd.Aliases, "rec")

	typeFlag := marketplaceRecommendCmd.Flags().Lookup("type")
	require.NotNil(t, typeFlag)

	keywordsFlag := marketplaceRecommendCmd.Flags().Lookup("keywords")
	require.NotNil(t, keywordsFlag)

	similarFlag := marketplaceRecommendCmd.Flags().Lookup("similar")
	require.NotNil(t, similarFlag)

	maxFlag := marketplaceRecommendCmd.Flags().Lookup("max")
	require.NotNil(t, maxFlag)
	assert.Equal(t, "10", maxFlag.DefValue)
}

func TestMarketplaceCmd_PersistentFlags(t *testing.T) {
	t.Parallel()

	offlineFlag := marketplaceCmd.PersistentFlags().Lookup("offline")
	require.NotNil(t, offlineFlag)
	assert.Equal(t, "false", offlineFlag.DefValue)

	refreshFlag := marketplaceCmd.PersistentFlags().Lookup("refresh")
	require.NotNil(t, refreshFlag)
	assert.Equal(t, "false", refreshFlag.DefValue)
}

// ---------------------------------------------------------------------------
// runMarketplaceInstall tests
// ---------------------------------------------------------------------------

func TestRunMarketplaceInstall_InvalidPackageName(t *testing.T) {
	err := runMarketplaceInstall(marketplaceInstallCmd, []string{"INVALID_NAME"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestRunMarketplaceInstall_EmptyPackageName(t *testing.T) {
	err := runMarketplaceInstall(marketplaceInstallCmd, []string{""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestRunMarketplaceInstall_ValidPackage_FailsOnInstall(t *testing.T) {
	oldVer := mpInstallVer
	mpInstallVer = ""
	defer func() { mpInstallVer = oldVer }()

	err := runMarketplaceInstall(marketplaceInstallCmd, []string{"nvim-pro"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "installation failed")
}

func TestRunMarketplaceInstall_AtVersionSyntax(t *testing.T) {
	oldVer := mpInstallVer
	mpInstallVer = ""
	defer func() { mpInstallVer = oldVer }()

	err := runMarketplaceInstall(marketplaceInstallCmd, []string{"nvim-pro@1.2.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "installation failed")
}

func TestRunMarketplaceInstall_VersionFromSecondArg(t *testing.T) {
	oldVer := mpInstallVer
	mpInstallVer = ""
	defer func() { mpInstallVer = oldVer }()

	err := runMarketplaceInstall(marketplaceInstallCmd, []string{"nvim-pro", "2.0.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "installation failed")
}

// ---------------------------------------------------------------------------
// runMarketplaceUninstall tests
// ---------------------------------------------------------------------------

func TestRunMarketplaceUninstall_InvalidPackageName(t *testing.T) {
	err := runMarketplaceUninstall(marketplaceUninstallCmd, []string{"INVALID"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestRunMarketplaceUninstall_ValidButNotInstalled(t *testing.T) {
	err := runMarketplaceUninstall(marketplaceUninstallCmd, []string{"nonexistent-pkg"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uninstall failed")
}

// ---------------------------------------------------------------------------
// runMarketplaceUpdate tests
// ---------------------------------------------------------------------------

func TestRunMarketplaceUpdate_InvalidPackageName(t *testing.T) {
	err := runMarketplaceUpdate(marketplaceUpdateCmd, []string{"BAD_NAME!"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestRunMarketplaceUpdate_SpecificPackage_NotInstalled(t *testing.T) {
	err := runMarketplaceUpdate(marketplaceUpdateCmd, []string{"nonexistent-pkg"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

// ---------------------------------------------------------------------------
// runMarketplaceInfo tests
// ---------------------------------------------------------------------------

func TestRunMarketplaceInfo_InvalidPackageName(t *testing.T) {
	err := runMarketplaceInfo(marketplaceInfoCmd, []string{"BAD!NAME"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestRunMarketplaceInfo_NonexistentPackage(t *testing.T) {
	err := runMarketplaceInfo(marketplaceInfoCmd, []string{"nonexistent-pkg"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package not found")
}

// ---------------------------------------------------------------------------
// runMarketplaceSearch tests
// ---------------------------------------------------------------------------

func TestRunMarketplaceSearch_WithQuery(t *testing.T) {
	err := runMarketplaceSearch(marketplaceSearchCmd, []string{"nvim"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}

func TestRunMarketplaceSearch_WithTypeFilter(t *testing.T) {
	oldType := mpSearchType
	mpSearchType = "preset"
	defer func() { mpSearchType = oldType }()

	err := runMarketplaceSearch(marketplaceSearchCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}

// ---------------------------------------------------------------------------
// runMarketplaceRecommend tests
// ---------------------------------------------------------------------------

func TestRunMarketplaceRecommend_FailsWithoutIndex(t *testing.T) {
	oldSimilar := mpSimilarTo
	oldKeywords := mpKeywords
	oldType := mpRecommendType
	mpSimilarTo = ""
	mpKeywords = ""
	mpRecommendType = ""
	defer func() {
		mpSimilarTo = oldSimilar
		mpKeywords = oldKeywords
		mpRecommendType = oldType
	}()

	err := runMarketplaceRecommend(marketplaceRecommendCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recommendation failed")
}

func TestRunMarketplaceRecommend_SimilarMode_InvalidPackage(t *testing.T) {
	oldSimilar := mpSimilarTo
	mpSimilarTo = "INVALID_PKG"
	defer func() { mpSimilarTo = oldSimilar }()

	err := runMarketplaceRecommend(marketplaceRecommendCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid package name")
}

func TestRunMarketplaceRecommend_SimilarMode(t *testing.T) {
	oldSimilar := mpSimilarTo
	mpSimilarTo = "nvim-pro"
	defer func() { mpSimilarTo = oldSimilar }()

	err := runMarketplaceRecommend(marketplaceRecommendCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recommendation failed")
}

// ---------------------------------------------------------------------------
// runMarketplaceFeatured tests
// ---------------------------------------------------------------------------

func TestRunMarketplaceFeatured_FailsWithoutIndex(t *testing.T) {
	err := runMarketplaceFeatured(marketplaceFeaturedCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get featured packages")
}

func TestRunMarketplaceFeatured_WithRefreshFlag(t *testing.T) {
	oldRefresh := mpRefreshIndex
	mpRefreshIndex = true
	defer func() { mpRefreshIndex = oldRefresh }()

	err := runMarketplaceFeatured(marketplaceFeaturedCmd, []string{})
	assert.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "failed to refresh index") ||
			strings.Contains(err.Error(), "failed to get featured packages"),
		"unexpected error: %v", err)
}

// ---------------------------------------------------------------------------
// runMarketplacePopular tests
// ---------------------------------------------------------------------------

func TestRunMarketplacePopular_FailsWithoutIndex(t *testing.T) {
	err := runMarketplacePopular(marketplacePopularCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get popular packages")
}

// ---------------------------------------------------------------------------
// formatInstallAge boundary tests
// ---------------------------------------------------------------------------

func TestFormatInstallAge_BoundaryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"exactly_1_minute", time.Minute, "1m ago"},
		{"exactly_1_hour", time.Hour, "1h ago"},
		{"exactly_1_day", 24 * time.Hour, "1d ago"},
		{"exactly_1_week", 7 * 24 * time.Hour, "1w ago"},
		{"59_seconds_just_now", 59 * time.Second, "just now"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatInstallAge(time.Now().Add(-tt.duration))
			assert.Equal(t, tt.want, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Arg validation tests
// ---------------------------------------------------------------------------

func TestMarketplaceInstallCmd_RequiresAtLeastOneArg(t *testing.T) {
	t.Parallel()
	err := marketplaceInstallCmd.Args(marketplaceInstallCmd, []string{})
	assert.Error(t, err)
}

func TestMarketplaceUninstallCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()
	err := marketplaceUninstallCmd.Args(marketplaceUninstallCmd, []string{})
	assert.Error(t, err)

	err = marketplaceUninstallCmd.Args(marketplaceUninstallCmd, []string{"pkg"})
	assert.NoError(t, err)
}

func TestMarketplaceInfoCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()
	err := marketplaceInfoCmd.Args(marketplaceInfoCmd, []string{})
	assert.Error(t, err)

	err = marketplaceInfoCmd.Args(marketplaceInfoCmd, []string{"pkg"})
	assert.NoError(t, err)
}
