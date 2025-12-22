// Package files provides the files provider for dotfile management.
package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Config represents the files section of the configuration.
type Config struct {
	Links     []Link
	Templates []Template
	Copies    []Copy
}

// Link represents a symbolic link to create.
type Link struct {
	Src    string // Source file path (relative to config root)
	Dest   string // Destination path (supports ~ expansion)
	Force  bool   // Overwrite existing files
	Backup bool   // Backup existing files before overwriting
}

// ID returns a unique identifier for this link.
func (l Link) ID() string {
	hash := sha256.Sum256([]byte(l.Dest))
	return hex.EncodeToString(hash[:8])
}

// Template represents a file to render from a template.
type Template struct {
	Src  string            // Template source path
	Dest string            // Destination path
	Vars map[string]string // Template variables
	Mode string            // File mode (e.g., "0644")
}

// ID returns a unique identifier for this template.
func (t Template) ID() string {
	hash := sha256.Sum256([]byte(t.Dest))
	return hex.EncodeToString(hash[:8])
}

// Copy represents a file to copy.
type Copy struct {
	Src  string // Source file path
	Dest string // Destination path
	Mode string // File mode (e.g., "0644")
}

// ID returns a unique identifier for this copy.
func (c Copy) ID() string {
	hash := sha256.Sum256([]byte(c.Dest))
	return hex.EncodeToString(hash[:8])
}

// ParseConfig parses the files configuration from a raw map.
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Links:     make([]Link, 0),
		Templates: make([]Template, 0),
		Copies:    make([]Copy, 0),
	}

	// Parse links
	if links, ok := raw["links"]; ok {
		linkList, ok := links.([]interface{})
		if !ok {
			return nil, fmt.Errorf("links must be a list")
		}
		for _, l := range linkList {
			link, err := parseLink(l)
			if err != nil {
				return nil, err
			}
			cfg.Links = append(cfg.Links, link)
		}
	}

	// Parse templates
	if templates, ok := raw["templates"]; ok {
		templateList, ok := templates.([]interface{})
		if !ok {
			return nil, fmt.Errorf("templates must be a list")
		}
		for _, t := range templateList {
			tmpl, err := parseTemplate(t)
			if err != nil {
				return nil, err
			}
			cfg.Templates = append(cfg.Templates, tmpl)
		}
	}

	// Parse copies
	if copies, ok := raw["copies"]; ok {
		copyList, ok := copies.([]interface{})
		if !ok {
			return nil, fmt.Errorf("copies must be a list")
		}
		for _, c := range copyList {
			cp, err := parseCopy(c)
			if err != nil {
				return nil, err
			}
			cfg.Copies = append(cfg.Copies, cp)
		}
	}

	return cfg, nil
}

// parseLink parses a single link from a map.
func parseLink(raw interface{}) (Link, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return Link{}, fmt.Errorf("link must be an object")
	}

	link := Link{}

	if src, ok := m["src"].(string); ok {
		link.Src = src
	} else {
		return Link{}, fmt.Errorf("link must have a src")
	}

	if dest, ok := m["dest"].(string); ok {
		link.Dest = dest
	} else {
		return Link{}, fmt.Errorf("link must have a dest")
	}

	if force, ok := m["force"].(bool); ok {
		link.Force = force
	}

	if backup, ok := m["backup"].(bool); ok {
		link.Backup = backup
	}

	return link, nil
}

// parseTemplate parses a single template from a map.
func parseTemplate(raw interface{}) (Template, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return Template{}, fmt.Errorf("template must be an object")
	}

	tmpl := Template{
		Vars: make(map[string]string),
	}

	if src, ok := m["src"].(string); ok {
		tmpl.Src = src
	} else {
		return Template{}, fmt.Errorf("template must have a src")
	}

	if dest, ok := m["dest"].(string); ok {
		tmpl.Dest = dest
	} else {
		return Template{}, fmt.Errorf("template must have a dest")
	}

	if mode, ok := m["mode"].(string); ok {
		tmpl.Mode = mode
	}

	if vars, ok := m["vars"].(map[string]interface{}); ok {
		for k, v := range vars {
			if str, ok := v.(string); ok {
				tmpl.Vars[k] = str
			}
		}
	}

	return tmpl, nil
}

// parseCopy parses a single copy from a map.
func parseCopy(raw interface{}) (Copy, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return Copy{}, fmt.Errorf("copy must be an object")
	}

	cp := Copy{}

	if src, ok := m["src"].(string); ok {
		cp.Src = src
	} else {
		return Copy{}, fmt.Errorf("copy must have a src")
	}

	if dest, ok := m["dest"].(string); ok {
		cp.Dest = dest
	} else {
		return Copy{}, fmt.Errorf("copy must have a dest")
	}

	if mode, ok := m["mode"].(string); ok {
		cp.Mode = mode
	}

	return cp, nil
}
