package jetbrains

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// IDE represents a JetBrains IDE product.
type IDE string

// IDE constants for supported JetBrains products.
const (
	IDEIntelliJ  IDE = "IntelliJIdea"
	IDEPyCharm   IDE = "PyCharm"
	IDEWebStorm  IDE = "WebStorm"
	IDEGoLand    IDE = "GoLand"
	IDEPhpStorm  IDE = "PhpStorm"
	IDERubyMine  IDE = "RubyMine"
	IDECLion     IDE = "CLion"
	IDEDataGrip  IDE = "DataGrip"
	IDERider     IDE = "Rider"
	IDEAndroid   IDE = "AndroidStudio"
	IDEFleet     IDE = "Fleet"
	IDERustRover IDE = "RustRover"
	IDEAquaDB    IDE = "Aqua"
)

// AllIDEs returns all supported JetBrains IDEs.
func AllIDEs() []IDE {
	return []IDE{
		IDEIntelliJ, IDEPyCharm, IDEWebStorm, IDEGoLand, IDEPhpStorm,
		IDERubyMine, IDECLion, IDEDataGrip, IDERider, IDEAndroid,
		IDEFleet, IDERustRover, IDEAquaDB,
	}
}

// Discovery provides config path discovery for JetBrains IDEs.
type Discovery struct {
	finder *pathutil.ConfigFinder
	goos   string
}

// NewDiscovery creates a new JetBrains discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   runtime.GOOS,
	}
}

// NewDiscoveryWithOS creates a JetBrains discovery with a specific OS (for testing).
func NewDiscoveryWithOS(goos string) *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
		goos:   goos,
	}
}

// SearchOpts returns the search options for JetBrains IDE config.
func SearchOpts(ide IDE) pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// No specific env var override for JetBrains
		EnvVar:         "",
		ConfigFileName: "",
		MacOSPaths: []string{
			"~/Library/Application Support/JetBrains/" + string(ide) + "*",
		},
		LinuxPaths: []string{
			"~/.config/JetBrains/" + string(ide) + "*",
		},
		WindowsPaths: []string{
			"$APPDATA/JetBrains/" + string(ide) + "*",
		},
	}
}

// FindConfigDir discovers the JetBrains IDE configuration directory.
// Returns the most recent version's config directory.
func (d *Discovery) FindConfigDir(ide IDE) string {
	home, _ := os.UserHomeDir()
	var baseDir string

	switch d.goos {
	case "darwin":
		baseDir = filepath.Join(home, "Library", "Application Support", "JetBrains")
	case "linux":
		baseDir = filepath.Join(home, ".config", "JetBrains")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		baseDir = filepath.Join(appData, "JetBrains")
	default:
		baseDir = filepath.Join(home, ".config", "JetBrains")
	}

	// Find the most recent version directory for the IDE
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		// Return best-practice path if directory doesn't exist
		return filepath.Join(baseDir, string(ide)+"2024.1")
	}

	var matchingDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), string(ide)) {
			matchingDirs = append(matchingDirs, entry.Name())
		}
	}

	if len(matchingDirs) == 0 {
		// Return best-practice path for new installations
		return filepath.Join(baseDir, string(ide)+"2024.1")
	}

	// Sort descending to get the most recent version
	sort.Sort(sort.Reverse(sort.StringSlice(matchingDirs)))
	return filepath.Join(baseDir, matchingDirs[0])
}

// BestPracticePath returns the canonical path for JetBrains config.
func (d *Discovery) BestPracticePath(ide IDE) string {
	home, _ := os.UserHomeDir()

	switch d.goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "JetBrains", string(ide)+"2024.1")
	case "linux":
		return filepath.Join(home, ".config", "JetBrains", string(ide)+"2024.1")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "JetBrains", string(ide)+"2024.1")
	default:
		return filepath.Join(home, ".config", "JetBrains", string(ide)+"2024.1")
	}
}

// FindOptionsDir returns the path to the options directory.
func (d *Discovery) FindOptionsDir(ide IDE) string {
	return filepath.Join(d.FindConfigDir(ide), "options")
}

// FindCodeStylesDir returns the path to the codestyles directory.
func (d *Discovery) FindCodeStylesDir(ide IDE) string {
	return filepath.Join(d.FindConfigDir(ide), "codestyles")
}

// FindKeymapsDir returns the path to the keymaps directory.
func (d *Discovery) FindKeymapsDir(ide IDE) string {
	return filepath.Join(d.FindConfigDir(ide), "keymaps")
}

// FindPluginsDir returns the path to the plugins directory.
func (d *Discovery) FindPluginsDir(ide IDE) string {
	return filepath.Join(d.FindConfigDir(ide), "plugins")
}

// GetInstalledIDEs returns a list of installed JetBrains IDEs.
func (d *Discovery) GetInstalledIDEs() []IDE {
	home, _ := os.UserHomeDir()
	var baseDir string

	switch d.goos {
	case "darwin":
		baseDir = filepath.Join(home, "Library", "Application Support", "JetBrains")
	case "linux":
		baseDir = filepath.Join(home, ".config", "JetBrains")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		baseDir = filepath.Join(appData, "JetBrains")
	default:
		baseDir = filepath.Join(home, ".config", "JetBrains")
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}

	installedSet := make(map[IDE]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		for _, ide := range AllIDEs() {
			if strings.HasPrefix(entry.Name(), string(ide)) {
				installedSet[ide] = true
			}
		}
	}

	installed := make([]IDE, 0, len(installedSet))
	for ide := range installedSet {
		installed = append(installed, ide)
	}
	return installed
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths(ide IDE) []string {
	return d.finder.GetCandidatePaths(SearchOpts(ide))
}
