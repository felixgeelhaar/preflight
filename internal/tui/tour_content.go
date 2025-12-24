package tui

// TopicContent defines the content for a tour topic.
type TopicContent struct {
	ID          string
	Title       string
	Description string
	Sections    []Section
	NextTopics  []string // Suggested follow-up topics
}

// Section represents a section within a topic.
type Section struct {
	Title   string
	Content string
	Code    string // Optional code example
}

// GetAllTopics returns all available tour topics.
func GetAllTopics() []TopicContent {
	return []TopicContent{
		getBasicsTopic(),
		getConfigTopic(),
		getLayersTopic(),
		getProvidersTopic(),
		getPresetsTopic(),
		getWorkflowTopic(),
	}
}

// GetTopic returns a specific topic by ID.
func GetTopic(id string) (TopicContent, bool) {
	for _, topic := range GetAllTopics() {
		if topic.ID == id {
			return topic, true
		}
	}
	return TopicContent{}, false
}

// GetTopicIDs returns all topic IDs.
func GetTopicIDs() []string {
	topics := GetAllTopics()
	ids := make([]string, len(topics))
	for i, t := range topics {
		ids[i] = t.ID
	}
	return ids
}

func getBasicsTopic() TopicContent {
	return TopicContent{
		ID:          "basics",
		Title:       "Preflight Fundamentals",
		Description: "Learn the core concepts behind Preflight",
		Sections: []Section{
			{
				Title: "What is Preflight?",
				Content: `Preflight is a deterministic workstation compiler. It treats your
machine setup as a compilation problem:

  Intent (config) → Merge → Plan → Apply → Verify

Unlike scripts or ad-hoc installers, Preflight guarantees:
• Reproducible results across machines
• Safe, idempotent operations
• Full explainability of every action`,
			},
			{
				Title: "The Compiler Model",
				Content: `Preflight works like a compiler:

1. LOAD    - Parse your configuration files
2. MERGE   - Combine layers into a unified config
3. COMPILE - Transform config into executable steps
4. PLAN    - Diff current state vs desired state
5. APPLY   - Execute steps idempotently
6. VERIFY  - Doctor checks for drift

This model ensures you always know what will happen before it does.`,
			},
			{
				Title: "Key Guarantees",
				Content: `Preflight provides these guarantees:

✓ No execution without a plan
  Always see what will change first

✓ Idempotent operations
  Re-running apply is always safe

✓ Explainability
  Every action has "why" and tradeoffs

✓ Secrets never leave the machine
  Capture redacts, config uses references

✓ User ownership
  Config is portable, git-native, yours`,
			},
		},
		NextTopics: []string{"config", "workflow"},
	}
}

func getConfigTopic() TopicContent {
	return TopicContent{
		ID:          "config",
		Title:       "Configuration Deep-Dive",
		Description: "Understand Preflight configuration structure",
		Sections: []Section{
			{
				Title: "Configuration Structure",
				Content: `Preflight uses a layered configuration model:

preflight.yaml    → Root manifest (targets, defaults)
layers/           → Composable overlays
  base.yaml       → Shared configuration
  identity.*.yaml → User-specific settings
  role.*.yaml     → Role-specific tools
  device.*.yaml   → Machine-specific config
dotfiles/         → Generated or managed files
preflight.lock    → Version pinning`,
			},
			{
				Title:   "The Manifest File",
				Content: `The manifest (preflight.yaml) defines targets:`,
				Code: `# preflight.yaml
targets:
  default:
    - base
    - identity.work
    - role.dev
  personal:
    - base
    - identity.personal
    - role.dev

defaults:
  mode: locked
  editor: nvim`,
			},
			{
				Title:   "Layer Files",
				Content: `Layers contain actual configuration:`,
				Code: `# layers/base.yaml
name: base

packages:
  brew:
    formulae:
      - ripgrep
      - fzf
      - bat

git:
  user:
    name: "Your Name"
  core:
    editor: nvim

shell:
  default: zsh`,
			},
			{
				Title: "Merge Semantics",
				Content: `When layers are combined:

• Scalars: Last layer wins
• Maps: Deep merge (keys combined)
• Lists: Set union with add/remove directives

Each value tracks its source layer for explainability.`,
			},
		},
		NextTopics: []string{"layers", "providers"},
	}
}

func getLayersTopic() TopicContent {
	return TopicContent{
		ID:          "layers",
		Title:       "Layer Composition",
		Description: "Master the layer system for flexible configs",
		Sections: []Section{
			{
				Title: "Layer Philosophy",
				Content: `Layers let you compose configurations:

BASE LAYER
  └── Common tools everyone needs
      └── IDENTITY LAYER
          └── User-specific settings (email, keys)
              └── ROLE LAYER
                  └── Job-specific tools (dev, writer)
                      └── DEVICE LAYER
                          └── Machine-specific tweaks`,
			},
			{
				Title:   "Identity Layers",
				Content: `Separate work and personal identities:`,
				Code: `# layers/identity.work.yaml
name: identity.work

git:
  user:
    name: "Jane Doe"
    email: "jane@company.com"
    signingkey: "~/.ssh/id_work.pub"

ssh:
  hosts:
    - host: "github.com-work"
      hostname: github.com
      identityfile: "~/.ssh/id_work"`,
			},
			{
				Title:   "Role Layers",
				Content: `Define tools for specific roles:`,
				Code: `# layers/role.go-dev.yaml
name: role.go-dev

packages:
  brew:
    formulae:
      - go
      - gopls
      - golangci-lint
      - delve

nvim:
  preset: pro
  lsp:
    - gopls`,
			},
			{
				Title:   "Device Layers",
				Content: `Machine-specific overrides:`,
				Code: `# layers/device.macbook.yaml
name: device.macbook

packages:
  brew:
    casks:
      - rectangle
      - raycast

shell:
  env:
    DISPLAY_SCALE: "2"`,
			},
		},
		NextTopics: []string{"presets", "providers"},
	}
}

func getProvidersTopic() TopicContent {
	return TopicContent{
		ID:          "providers",
		Title:       "Provider Overview",
		Description: "Learn about available configuration providers",
		Sections: []Section{
			{
				Title: "What are Providers?",
				Content: `Providers translate config into executable steps:

  Config Section → Provider → Steps → Actions

Each provider handles a specific domain:
• brew - Package management (macOS)
• apt - Package management (Linux)
• git - Git configuration
• shell - Shell setup (zsh, starship)
• nvim - Neovim configuration
• vscode - VS Code extensions & settings
• ssh - SSH config generation
• fonts - Nerd Font installation
• runtime - Version managers (mise, asdf)
• files - Dotfile management`,
			},
			{
				Title:   "Package Providers",
				Content: `Brew and APT manage system packages:`,
				Code: `packages:
  brew:
    taps:
      - homebrew/cask-fonts
    formulae:
      - ripgrep
      - neovim
    casks:
      - docker
      - wezterm`,
			},
			{
				Title:   "Editor Providers",
				Content: `Neovim and VS Code are first-class:`,
				Code: `nvim:
  preset: balanced  # minimal, balanced, pro
  plugin_manager: lazy
  colorscheme: catppuccin

vscode:
  extensions:
    - golang.go
    - esbenp.prettier-vscode
  settings:
    editor.fontSize: 14`,
			},
			{
				Title:   "Shell Provider",
				Content: `Configure your shell environment:`,
				Code: `shell:
  default: zsh
  framework: oh-my-zsh
  theme: robbyrussell
  plugins:
    - git
    - docker
  starship:
    enabled: true
    preset: plain-text
  aliases:
    ll: "ls -la"
    k: kubectl`,
			},
			{
				Title:   "Fonts Provider",
				Content: `Install Nerd Fonts for terminal icons:`,
				Code: `fonts:
  nerd_fonts:
    - JetBrainsMono
    - FiraCode
    - Hack`,
			},
		},
		NextTopics: []string{"presets", "workflow"},
	}
}

func getPresetsTopic() TopicContent {
	return TopicContent{
		ID:          "presets",
		Title:       "Using Presets",
		Description: "Leverage pre-built configurations",
		Sections: []Section{
			{
				Title: "What are Presets?",
				Content: `Presets are curated configuration bundles:

• Pre-configured, tested combinations
• Difficulty levels (beginner → advanced)
• Documented tradeoffs
• Links to documentation

Examples:
  nvim:minimal    → Lightweight editor setup
  nvim:balanced   → Common plugins, good defaults
  nvim:pro        → Full IDE with debugging`,
			},
			{
				Title: "Available Presets",
				Content: `Presets by category:

EDITORS
  nvim:minimal, nvim:balanced, nvim:pro
  vscode:minimal, vscode:full

SHELL
  shell:zsh, shell:oh-my-zsh, shell:starship

GIT
  git:standard, git:secure (GPG signing)

TERMINAL
  terminal:tmux, terminal:alacritty

FONTS
  fonts:nerd-essential, fonts:nerd-complete`,
			},
			{
				Title: "Capability Packs",
				Content: `Packs combine presets for specific roles:

  go-developer      → nvim:balanced + starship + go tools
  frontend-developer → nvim:balanced + node tools
  python-developer  → nvim:balanced + poetry + ruff
  rust-developer    → nvim:pro + cargo tools
  devops-engineer   → nvim:pro + terraform + k8s
  data-scientist    → vscode:full + jupyter + pandas
  full-stack        → nvim:pro + tmux + docker`,
			},
			{
				Title:   "Using Presets",
				Content: `Reference presets in your layers:`,
				Code: `# layers/role.dev.yaml
name: role.dev

nvim:
  preset: balanced

shell:
  preset: starship

git:
  preset: secure`,
			},
		},
		NextTopics: []string{"workflow", "layers"},
	}
}

func getWorkflowTopic() TopicContent {
	return TopicContent{
		ID:          "workflow",
		Title:       "Daily Workflow",
		Description: "Learn the plan-apply-doctor cycle",
		Sections: []Section{
			{
				Title: "The Core Workflow",
				Content: `Preflight follows a predictable workflow:

  ┌─────────┐     ┌───────┐     ┌────────┐
  │  plan   │ ──► │ apply │ ──► │ doctor │
  └─────────┘     └───────┘     └────────┘
       │               │              │
       ▼               ▼              ▼
   See changes    Execute plan   Verify state`,
			},
			{
				Title:   "Planning Changes",
				Content: `Always plan before applying:`,
				Code: `# See what would change
preflight plan

# With detailed explanations
preflight plan --explain

# For a specific target
preflight plan --target personal

# Output as JSON for scripting
preflight plan --json`,
			},
			{
				Title:   "Applying Changes",
				Content: `Apply your configuration:`,
				Code: `# Apply with confirmation
preflight apply

# Skip confirmation (CI/scripts)
preflight apply --yes

# Apply specific target
preflight apply --target work

# Update lockfile after apply
preflight apply --update-lock`,
			},
			{
				Title:   "Doctor & Drift Detection",
				Content: `Verify system state and detect drift:`,
				Code: `# Check for issues
preflight doctor

# Fix machine to match config
preflight doctor --fix

# Update config from machine state
preflight doctor --update-config

# Preview changes without writing
preflight doctor --update-config --dry-run`,
			},
			{
				Title:   "Rollback",
				Content: `Restore from automatic snapshots:`,
				Code: `# List available snapshots
preflight rollback

# Restore specific snapshot
preflight rollback --to abc123

# Restore most recent
preflight rollback --latest

# Preview restoration
preflight rollback --to abc123 --dry-run`,
			},
		},
		NextTopics: []string{"basics", "config"},
	}
}
