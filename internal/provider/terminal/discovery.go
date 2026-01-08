package terminal

import (
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Paths holds discovered paths for a terminal emulator.
type Paths struct {
	ConfigFile string // Main config file path
	ConfigDir  string // Config directory (for additional files)
	DataDir    string // Data directory (themes, plugins, etc.)
}

// Discovery provides terminal-specific config path resolution.
type Discovery struct {
	finder *pathutil.ConfigFinder
}

// NewDiscovery creates a new Discovery instance.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
	}
}

// NewDiscoveryWithFinder creates a Discovery with a custom ConfigFinder (for testing).
func NewDiscoveryWithFinder(finder *pathutil.ConfigFinder) *Discovery {
	return &Discovery{finder: finder}
}

// --- Alacritty ---

// AlacrittySearchOpts returns the search options for Alacritty.
func AlacrittySearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		EnvVar:         "ALACRITTY_CONFIG_DIR",
		ConfigFileName: "alacritty.toml",
		XDGSubpath:     "alacritty/alacritty.toml",
		MacOSPaths:     []string{"~/.config/alacritty/alacritty.toml"},
		LinuxPaths:     []string{},
		WindowsPaths:   []string{"%APPDATA%/alacritty/alacritty.toml"},
		LegacyPaths:    []string{"~/.alacritty.toml", "~/.alacritty.yml"},
	}
}

// FindAlacrittyConfig finds the Alacritty config file.
func (d *Discovery) FindAlacrittyConfig() string {
	return d.finder.FindConfig(AlacrittySearchOpts())
}

// AlacrittyBestPracticePath returns the best-practice path for Alacritty config.
func (d *Discovery) AlacrittyBestPracticePath() string {
	return d.finder.BestPracticePath(AlacrittySearchOpts())
}

// --- Kitty ---

// KittySearchOpts returns the search options for Kitty.
func KittySearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		EnvVar:         "KITTY_CONFIG_DIRECTORY",
		ConfigFileName: "kitty.conf",
		XDGSubpath:     "kitty/kitty.conf",
		MacOSPaths:     []string{"~/.config/kitty/kitty.conf"},
		LinuxPaths:     []string{},
		WindowsPaths:   []string{}, // Kitty not available on Windows
		LegacyPaths:    []string{},
	}
}

// FindKittyConfig finds the Kitty config file.
func (d *Discovery) FindKittyConfig() string {
	return d.finder.FindConfig(KittySearchOpts())
}

// KittyBestPracticePath returns the best-practice path for Kitty config.
func (d *Discovery) KittyBestPracticePath() string {
	return d.finder.BestPracticePath(KittySearchOpts())
}

// --- WezTerm ---

// WezTermSearchOpts returns the search options for WezTerm.
func WezTermSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		EnvVar:         "WEZTERM_CONFIG_DIR",
		ConfigFileName: "wezterm.lua",
		XDGSubpath:     "wezterm/wezterm.lua",
		MacOSPaths:     []string{"~/.config/wezterm/wezterm.lua"},
		LinuxPaths:     []string{},
		WindowsPaths:   []string{"%USERPROFILE%/.config/wezterm/wezterm.lua"},
		LegacyPaths:    []string{"~/.wezterm.lua"},
	}
}

// FindWezTermConfig finds the WezTerm config file.
func (d *Discovery) FindWezTermConfig() string {
	return d.finder.FindConfig(WezTermSearchOpts())
}

// WezTermBestPracticePath returns the best-practice path for WezTerm config.
func (d *Discovery) WezTermBestPracticePath() string {
	return d.finder.BestPracticePath(WezTermSearchOpts())
}

// --- Ghostty ---

// GhosttySearchOpts returns the search options for Ghostty.
func GhosttySearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		EnvVar:         "", // Ghostty uses XDG_CONFIG_HOME only
		ConfigFileName: "",
		XDGSubpath:     "ghostty/config",
		MacOSPaths:     []string{"~/.config/ghostty/config"},
		LinuxPaths:     []string{},
		WindowsPaths:   []string{}, // Ghostty not available on Windows
		LegacyPaths:    []string{},
	}
}

// FindGhosttyConfig finds the Ghostty config file.
func (d *Discovery) FindGhosttyConfig() string {
	return d.finder.FindConfig(GhosttySearchOpts())
}

// GhosttyBestPracticePath returns the best-practice path for Ghostty config.
func (d *Discovery) GhosttyBestPracticePath() string {
	return d.finder.BestPracticePath(GhosttySearchOpts())
}

// --- iTerm2 (macOS only) ---

// ITerm2SearchOpts returns the search options for iTerm2.
func ITerm2SearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		EnvVar:         "",
		ConfigFileName: "",
		XDGSubpath:     "", // iTerm2 doesn't use XDG
		MacOSPaths:     []string{"~/Library/Preferences/com.googlecode.iterm2.plist"},
		LinuxPaths:     []string{}, // Not available
		WindowsPaths:   []string{}, // Not available
		LegacyPaths:    []string{},
	}
}

// FindITerm2Config finds the iTerm2 plist file.
func (d *Discovery) FindITerm2Config() string {
	return d.finder.FindConfig(ITerm2SearchOpts())
}

// ITerm2BestPracticePath returns the best-practice path for iTerm2 config.
func (d *Discovery) ITerm2BestPracticePath() string {
	return d.finder.BestPracticePath(ITerm2SearchOpts())
}

// ITerm2DynamicProfilesDir returns the path to iTerm2 dynamic profiles directory.
func (d *Discovery) ITerm2DynamicProfilesDir() string {
	return pathutil.ExpandPath("~/Library/Application Support/iTerm2/DynamicProfiles")
}

// --- Hyper ---

// HyperSearchOpts returns the search options for Hyper.
func HyperSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		EnvVar:         "",
		ConfigFileName: "",
		XDGSubpath:     "", // Hyper uses ~/.hyper.js directly
		MacOSPaths:     []string{"~/.hyper.js"},
		LinuxPaths:     []string{"~/.hyper.js"},
		WindowsPaths:   []string{"%USERPROFILE%/.hyper.js"},
		LegacyPaths:    []string{},
	}
}

// FindHyperConfig finds the Hyper config file.
func (d *Discovery) FindHyperConfig() string {
	return d.finder.FindConfig(HyperSearchOpts())
}

// HyperBestPracticePath returns the best-practice path for Hyper config.
func (d *Discovery) HyperBestPracticePath() string {
	return d.finder.BestPracticePath(HyperSearchOpts())
}

// --- Windows Terminal ---

// WindowsTerminalSearchOpts returns the search options for Windows Terminal.
func WindowsTerminalSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		EnvVar:         "",
		ConfigFileName: "",
		XDGSubpath:     "",         // Windows Terminal uses its own location
		MacOSPaths:     []string{}, // Not available
		LinuxPaths:     []string{}, // Not available (except WSL)
		WindowsPaths: []string{
			"%LOCALAPPDATA%/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json",
			"%LOCALAPPDATA%/Packages/Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe/LocalState/settings.json",
		},
		LegacyPaths: []string{},
	}
}

// FindWindowsTerminalConfig finds the Windows Terminal settings file.
func (d *Discovery) FindWindowsTerminalConfig() string {
	return d.finder.FindConfig(WindowsTerminalSearchOpts())
}

// WindowsTerminalBestPracticePath returns the best-practice path for Windows Terminal config.
func (d *Discovery) WindowsTerminalBestPracticePath() string {
	return d.finder.BestPracticePath(WindowsTerminalSearchOpts())
}

// --- Best Practice Paths Map ---

// BestPracticePaths returns a map of terminal names to their best-practice config paths.
func (d *Discovery) BestPracticePaths() map[string]string {
	return map[string]string{
		"alacritty":        d.AlacrittyBestPracticePath(),
		"kitty":            d.KittyBestPracticePath(),
		"wezterm":          d.WezTermBestPracticePath(),
		"ghostty":          d.GhosttyBestPracticePath(),
		"iterm2":           d.ITerm2BestPracticePath(),
		"hyper":            d.HyperBestPracticePath(),
		"windows_terminal": d.WindowsTerminalBestPracticePath(),
	}
}

// FindAllConfigs returns a map of terminal names to their found config paths.
// Only includes terminals that have a config file present.
func (d *Discovery) FindAllConfigs() map[string]string {
	result := make(map[string]string)

	if path := d.FindAlacrittyConfig(); path != "" {
		result["alacritty"] = path
	}
	if path := d.FindKittyConfig(); path != "" {
		result["kitty"] = path
	}
	if path := d.FindWezTermConfig(); path != "" {
		result["wezterm"] = path
	}
	if path := d.FindGhosttyConfig(); path != "" {
		result["ghostty"] = path
	}
	if path := d.FindITerm2Config(); path != "" {
		result["iterm2"] = path
	}
	if path := d.FindHyperConfig(); path != "" {
		result["hyper"] = path
	}
	if path := d.FindWindowsTerminalConfig(); path != "" {
		result["windows_terminal"] = path
	}

	return result
}
