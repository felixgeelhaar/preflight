package starship

import (
	"github.com/felixgeelhaar/preflight/internal/provider/pathutil"
)

// Discovery provides config path discovery for Starship.
type Discovery struct {
	finder *pathutil.ConfigFinder
}

// NewDiscovery creates a new Starship discovery.
func NewDiscovery() *Discovery {
	return &Discovery{
		finder: pathutil.NewConfigFinder(),
	}
}

// StarshipSearchOpts returns the search options for Starship config.
func StarshipSearchOpts() pathutil.ConfigSearchOpts {
	return pathutil.ConfigSearchOpts{
		// STARSHIP_CONFIG can point directly to the config file
		EnvVar:         "STARSHIP_CONFIG",
		ConfigFileName: "", // STARSHIP_CONFIG is the full path
		XDGSubpath:     "starship.toml",
		LegacyPaths:    []string{"~/.starship.toml"}, // Very old legacy location
	}
}

// FindConfig discovers the Starship configuration file location.
// Checks: 1) STARSHIP_CONFIG env var, 2) XDG_CONFIG_HOME/starship.toml, 3) legacy paths.
func (d *Discovery) FindConfig() string {
	return d.finder.FindConfig(StarshipSearchOpts())
}

// BestPracticePath returns the canonical path for Starship config.
func (d *Discovery) BestPracticePath() string {
	return d.finder.BestPracticePath(StarshipSearchOpts())
}

// GetCandidatePaths returns all candidate paths for config discovery (for capture).
func (d *Discovery) GetCandidatePaths() []string {
	return d.finder.GetCandidatePaths(StarshipSearchOpts())
}
