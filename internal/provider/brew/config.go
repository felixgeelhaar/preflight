// Package brew provides the Homebrew provider for package management on macOS.
package brew

import (
	"fmt"
)

// Config represents the brew section of the configuration.
type Config struct {
	Taps     []string
	Formulae []Formula
	Casks    []Cask
}

// Formula represents a Homebrew formula to install.
type Formula struct {
	Name string
	Tap  string   // Optional: specific tap (e.g., "homebrew/core")
	Args []string // Optional: install arguments (e.g., "--HEAD")
}

// FullName returns the fully qualified formula name.
func (f Formula) FullName() string {
	if f.Tap != "" {
		return fmt.Sprintf("%s/%s", f.Tap, f.Name)
	}
	return f.Name
}

// Cask represents a Homebrew cask to install.
type Cask struct {
	Name string
	Tap  string // Optional: specific tap (e.g., "homebrew/cask-fonts")
}

// FullName returns the fully qualified cask name.
func (c Cask) FullName() string {
	if c.Tap != "" {
		return fmt.Sprintf("%s/%s", c.Tap, c.Name)
	}
	return c.Name
}

// ParseConfig parses the brew configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Taps:     make([]string, 0),
		Formulae: make([]Formula, 0),
		Casks:    make([]Cask, 0),
	}

	// Parse taps
	if taps, ok := raw["taps"]; ok {
		tapList, ok := taps.([]interface{})
		if !ok {
			return nil, fmt.Errorf("taps must be a list")
		}
		for _, tap := range tapList {
			tapStr, ok := tap.(string)
			if !ok {
				return nil, fmt.Errorf("tap must be a string")
			}
			cfg.Taps = append(cfg.Taps, tapStr)
		}
	}

	// Parse formulae
	if formulae, ok := raw["formulae"]; ok {
		formulaeList, ok := formulae.([]interface{})
		if !ok {
			return nil, fmt.Errorf("formulae must be a list")
		}
		for _, f := range formulaeList {
			formula, err := parseFormula(f)
			if err != nil {
				return nil, err
			}
			cfg.Formulae = append(cfg.Formulae, formula)
		}
	}

	// Parse casks
	if casks, ok := raw["casks"]; ok {
		caskList, ok := casks.([]interface{})
		if !ok {
			return nil, fmt.Errorf("casks must be a list")
		}
		for _, c := range caskList {
			cask, err := parseCask(c)
			if err != nil {
				return nil, err
			}
			cfg.Casks = append(cfg.Casks, cask)
		}
	}

	return cfg, nil
}

// parseFormula parses a single formula from either a string or a map.
func parseFormula(raw interface{}) (Formula, error) {
	switch v := raw.(type) {
	case string:
		return Formula{Name: v}, nil
	case map[string]interface{}:
		formula := Formula{}
		if name, ok := v["name"].(string); ok {
			formula.Name = name
		} else {
			return Formula{}, fmt.Errorf("formula must have a name")
		}
		if tap, ok := v["tap"].(string); ok {
			formula.Tap = tap
		}
		if args, ok := v["args"].([]interface{}); ok {
			for _, arg := range args {
				if argStr, ok := arg.(string); ok {
					formula.Args = append(formula.Args, argStr)
				}
			}
		}
		return formula, nil
	default:
		return Formula{}, fmt.Errorf("formula must be a string or object")
	}
}

// parseCask parses a single cask from either a string or a map.
func parseCask(raw interface{}) (Cask, error) {
	switch v := raw.(type) {
	case string:
		return Cask{Name: v}, nil
	case map[string]interface{}:
		cask := Cask{}
		if name, ok := v["name"].(string); ok {
			cask.Name = name
		} else {
			return Cask{}, fmt.Errorf("cask must have a name")
		}
		if tap, ok := v["tap"].(string); ok {
			cask.Tap = tap
		}
		return cask, nil
	default:
		return Cask{}, fmt.Errorf("cask must be a string or object")
	}
}
