package tmux

import (
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for tmux.
type Discovery struct {
	finder *pathutil.ConfigFinder
}

// NewDiscovery creates a new tmux discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
	}
}

// ConfigSearchOpts returns the search options for tmux config.
func ConfigSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// TMUX_CONF can be used to specify custom config location
		EnvVar:         "TMUX_CONF",
		ConfigFileName: "", // TMUX_CONF is the full path
		XDGSubpath:     "tmux/tmux.conf",
		LegacyPaths: []string{
			"~/.tmux.conf", // Traditional location
		},
	}
}

// TPMSearchOpts returns the search options for TPM (Tmux Plugin Manager).
func TPMSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// TPM is typically installed in ~/.tmux/plugins/tpm
		// XDG location would be ~/.local/share/tmux/plugins/tpm
		XDGSubpath: "", // TPM doesn't follow XDG
		LegacyPaths: []string{
			"~/.tmux/plugins/tpm",
		},
	}
}

// FindConfig discovers the tmux configuration file location.
// Checks: 1) TMUX_CONF env var, 2) XDG_CONFIG_HOME/tmux/tmux.conf, 3) ~/.tmux.conf.
func (d *Discovery) FindConfig() string {
	return d.finder.FindConfig(ConfigSearchOpts())
}

// BestPracticePath returns the canonical path for tmux config.
// Prefers XDG location: ~/.config/tmux/tmux.conf
func (d *Discovery) BestPracticePath() string {
	return d.finder.BestPracticePath(ConfigSearchOpts())
}

// FindTPMPath discovers the TPM installation location.
func (d *Discovery) FindTPMPath() string {
	return d.finder.FindConfig(TPMSearchOpts())
}

// TPMBestPracticePath returns the canonical path for TPM installation.
func (d *Discovery) TPMBestPracticePath() string {
	return d.finder.BestPracticePath(TPMSearchOpts())
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(ConfigSearchOpts())
}
