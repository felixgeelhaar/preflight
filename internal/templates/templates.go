// Package templates provides file templates for preflight repository initialization.
package templates

import (
	"bytes"
	"text/template"
)

// GitignoreTemplate generates a .gitignore file for preflight dotfiles repositories.
const GitignoreTemplate = `# Secrets - never commit these
.env
.env.*
*.key
*.pem
credentials.json
secrets.yaml
secrets.yml
*.secret

# SSH keys - use references only
id_rsa*
id_ed25519*
id_ecdsa*
*.pub

# GPG
*.gpg
secring.*
trustdb.gpg

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
Thumbs.db
ehthumbs.db
Desktop.ini

# Editor files
*.swp
*.swo
*~
.vscode/settings.json
.idea/

# Preflight state (machine-specific)
preflight.lock
.preflight/
*.local.yaml

# Node modules (if using any scripts)
node_modules/
`

// ReadmeData contains data for the README template.
type ReadmeData struct {
	RepoName    string
	Description string
	Owner       string
}

// ReadmeTemplate is the template for the bootstrap README.
const readmeTemplateStr = `# {{.RepoName}}

{{if .Description}}{{.Description}}{{else}}Dotfiles and machine configuration managed by [preflight](https://github.com/felixgeelhaar/preflight).{{end}}

## Quick Start

### On a new machine

` + "```bash" + `
# Install preflight
brew install felixgeelhaar/tap/preflight

# Clone and apply configuration
preflight repo clone {{if .Owner}}git@github.com:{{.Owner}}/{{.RepoName}}.git{{else}}https://github.com/<username>/{{.RepoName}}.git{{end}}
` + "```" + `

### On an existing machine

` + "```bash" + `
# Review what will change
preflight plan

# Apply configuration
preflight apply

# Check for drift
preflight doctor
` + "```" + `

## Repository Structure

` + "```" + `
{{.RepoName}}/
├── preflight.yaml      # Main configuration manifest
├── layers/             # Configuration overlays
│   ├── base.yaml       # Common settings for all machines
│   ├── identity.*.yaml # Per-identity settings (work, personal)
│   └── role.*.yaml     # Per-role settings (go, python, etc.)
├── dotfiles/           # User-owned dotfiles
│   └── .config/
└── preflight.lock      # Locked versions (machine-specific)
` + "```" + `

## Targets

Targets combine layers for specific machine configurations:

- ` + "`work`" + ` - Work laptop with work identity
- ` + "`personal`" + ` - Personal machine with personal identity
- ` + "`minimal`" + ` - Minimal setup for servers/VMs

## Commands

| Command | Description |
|---------|-------------|
| ` + "`preflight plan`" + ` | Preview changes without applying |
| ` + "`preflight apply`" + ` | Apply configuration to system |
| ` + "`preflight doctor`" + ` | Check for configuration drift |
| ` + "`preflight capture`" + ` | Capture current system state |
| ` + "`preflight explain`" + ` | Explain what each setting does |

## Learn More

- [Preflight Documentation](https://github.com/felixgeelhaar/preflight#readme)
- [Configuration Reference](https://github.com/felixgeelhaar/preflight/docs/config.md)
`

// GenerateReadme generates a README.md file from the template.
func GenerateReadme(data ReadmeData) (string, error) {
	tmpl, err := template.New("readme").Parse(readmeTemplateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
