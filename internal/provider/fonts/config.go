// Package fonts provides the Fonts provider for Nerd Font installation on macOS.
package fonts

import (
	"fmt"
	"regexp"
	"strings"
)

// CaskFontsTap is the Homebrew tap for cask fonts.
const CaskFontsTap = "homebrew/cask-fonts"

// Config represents the fonts section of the configuration.
type Config struct {
	NerdFonts []string
}

// ParseConfig parses the fonts configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		NerdFonts: make([]string, 0),
	}

	// Parse nerd_fonts
	if nerdFonts, ok := raw["nerd_fonts"]; ok {
		fontList, ok := nerdFonts.([]interface{})
		if !ok {
			return nil, fmt.Errorf("nerd_fonts must be a list")
		}
		for _, f := range fontList {
			fontStr, ok := f.(string)
			if !ok {
				return nil, fmt.Errorf("nerd font must be a string")
			}
			cfg.NerdFonts = append(cfg.NerdFonts, fontStr)
		}
	}

	return cfg, nil
}

// fontNameMappings maps common font names to their Nerd Font cask names.
// Some fonts have special naming conventions in the Nerd Fonts project.
var fontNameMappings = map[string]string{
	"jetbrainsmono": "jetbrains-mono",
	"firacode":      "fira-code",
	"meslo":         "meslo-lg",
	"sourcecode":    "sauce-code-pro",
	"sourcecodepro": "sauce-code-pro",
	"cascadia":      "caskaydia-cove",
	"cascadiacode":  "caskaydia-cove",
	"droidsansmono": "droid-sans-mono",
	"ubuntumono":    "ubuntu-mono",
	"robotomono":    "roboto-mono",
	"victormono":    "victor-mono",
}

// NerdFontCaskName converts a Nerd Font name to its Homebrew cask name.
// Examples:
//   - "JetBrainsMono" -> "font-jetbrains-mono-nerd-font"
//   - "FiraCode" -> "font-fira-code-nerd-font"
//   - "Meslo" -> "font-meslo-lg-nerd-font" (special case)
func NerdFontCaskName(fontName string) string {
	// Normalize input: remove NF/Nerd Font suffixes
	name := fontName
	name = strings.TrimSuffix(name, "NF")
	name = strings.TrimSuffix(name, "NerdFont")

	// Convert to lowercase for matching
	lowerName := strings.ToLower(name)

	// Check for special mappings
	if mapped, ok := fontNameMappings[lowerName]; ok {
		return fmt.Sprintf("font-%s-nerd-font", mapped)
	}

	// Convert CamelCase to kebab-case
	kebab := camelToKebab(name)

	return fmt.Sprintf("font-%s-nerd-font", kebab)
}

// camelToKebab converts CamelCase to kebab-case.
func camelToKebab(s string) string {
	// Insert hyphen before uppercase letters (except at start)
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	kebab := re.ReplaceAllString(s, "${1}-${2}")
	return strings.ToLower(kebab)
}
