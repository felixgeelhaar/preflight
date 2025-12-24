package tui

// TopicContent defines the content for a tour topic.
type TopicContent struct {
	ID          string
	Title       string
	Description string
	Sections    []Section
	NextTopics  []string // Suggested follow-up topics
	HandsOn     bool     // True if this is a hands-on tutorial
}

// IsHandsOnTopic returns true if this topic contains hands-on exercises.
func (t TopicContent) IsHandsOnTopic() bool {
	return t.HandsOn
}

// Section represents a section within a topic.
type Section struct {
	Title   string
	Content string
	Code    string // Optional code example

	// Hands-on fields for interactive tutorials
	HandsOn       bool   // True if this is a practice section
	Command       string // Command for user to run (displayed in a copy-friendly format)
	Hint          string // Optional hint for completing the task
	VerifyCommand string // Optional command to verify completion (for display)
}

// IsHandsOn returns true if this section is a hands-on practice section.
func (s Section) IsHandsOn() bool {
	return s.HandsOn
}

// GetAllTopics returns all available tour topics.
func GetAllTopics() []TopicContent {
	return []TopicContent{
		// Conceptual topics
		getBasicsTopic(),
		getConfigTopic(),
		getLayersTopic(),
		getProvidersTopic(),
		getPresetsTopic(),
		getWorkflowTopic(),
		// Hands-on tutorials
		getNvimBasicsTopic(),
		getGitWorkflowTopic(),
		getShellCustomizationTopic(),
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

// =============================================================================
// HANDS-ON TUTORIALS
// =============================================================================

func getNvimBasicsTopic() TopicContent {
	return TopicContent{
		ID:          "nvim-basics",
		Title:       "🛠️ Neovim Basics",
		Description: "Hands-on Neovim tutorial",
		HandsOn:     true,
		Sections: []Section{
			{
				Title: "Welcome to Neovim",
				Content: `This hands-on tutorial will teach you Neovim basics.

You'll practice:
• Opening and navigating files
• Basic editing commands
• Saving and quitting
• Essential movements

Each section includes a command to try. Open a terminal alongside
this tour and follow along!`,
			},
			{
				Title:   "Opening Neovim",
				Content: `Let's start by opening Neovim. This creates a new empty buffer.`,
				HandsOn: true,
				Command: "nvim",
				Hint:    "Press 'i' to enter insert mode, type text, press Esc to exit insert mode",
			},
			{
				Title: "Understanding Modes",
				Content: `Neovim has several modes:

NORMAL (default)
  Navigate, delete, copy, paste
  Press Esc to return here

INSERT (press i)
  Type text like a normal editor
  Press Esc to exit

VISUAL (press v)
  Select text for operations
  Press Esc to exit

COMMAND (press :)
  Run commands like :w (save) or :q (quit)
  Press Enter to execute, Esc to cancel`,
			},
			{
				Title:   "Practice: Write and Save",
				Content: `Create a file, add some text, and save it.`,
				HandsOn: true,
				Command: "nvim practice.txt",
				Hint: `1. Press 'i' to enter insert mode
2. Type "Hello, Neovim!"
3. Press Esc to exit insert mode
4. Type :wq and press Enter to save and quit`,
				VerifyCommand: "cat practice.txt",
			},
			{
				Title: "Essential Movements",
				Content: `Navigate efficiently in Normal mode:

BASIC MOVEMENT
  h j k l    Left, Down, Up, Right
  w / b      Forward/backward by word
  0 / $      Start/end of line
  gg / G     Start/end of file

JUMPING
  Ctrl-d     Half page down
  Ctrl-u     Half page up
  { / }      Paragraph up/down
  /pattern   Search forward
  n / N      Next/previous match`,
			},
			{
				Title:   "Practice: Navigation",
				Content: `Open a file and practice moving around.`,
				HandsOn: true,
				Command: "nvim ~/.zshrc",
				Hint: `Try these movements:
• gg to go to top, G to go to bottom
• /alias to search for 'alias'
• n to find next match
• :q! to quit without saving`,
			},
			{
				Title: "Essential Editing",
				Content: `Common editing commands in Normal mode:

INSERTING
  i / a      Insert before/after cursor
  I / A      Insert at start/end of line
  o / O      New line below/above

DELETING
  x          Delete character
  dd         Delete line
  dw         Delete word
  d$         Delete to end of line

CHANGING
  cc         Change entire line
  cw         Change word
  r          Replace single character

UNDO/REDO
  u          Undo
  Ctrl-r     Redo`,
			},
			{
				Title:   "Practice: Editing",
				Content: `Create a file and practice editing operations.`,
				HandsOn: true,
				Command: "nvim editing-practice.txt",
				Hint: `1. Press 'i', type a few lines of text, press Esc
2. Navigate with j/k, delete a line with 'dd'
3. Press 'u' to undo
4. Press 'o' to add a new line below
5. Type :wq to save and quit`,
			},
			{
				Title: "Vim with Preflight",
				Content: `Preflight can configure Neovim for you:

PRESETS
  nvim:minimal   - Clean, fast setup
  nvim:balanced  - Common plugins, LSP ready
  nvim:pro       - Full IDE with debugging

EXAMPLE CONFIG`,
				Code: `# layers/role.dev.yaml
nvim:
  preset: balanced
  plugin_manager: lazy
  colorscheme: catppuccin
  lsp:
    - gopls
    - typescript-language-server`,
			},
			{
				Title: "Next Steps",
				Content: `You've learned the Neovim basics!

CONTINUE LEARNING
  :Tutor       Built-in Vim tutorial (30 min)
  vimtutor     Terminal command for the same

PREFLIGHT INTEGRATION
  preflight init --editor nvim
  preflight apply

RECOMMENDED
  Practice these movements daily
  Learn one new command each week

Try the git-workflow tutorial next!`,
				Code: `# Quick reference
i      Insert mode      :w     Save
Esc    Normal mode      :q     Quit
dd     Delete line      :wq    Save and quit
u      Undo            :q!    Quit without saving`,
			},
		},
		NextTopics: []string{"git-workflow", "providers"},
	}
}

func getGitWorkflowTopic() TopicContent {
	return TopicContent{
		ID:          "git-workflow",
		Title:       "🛠️ Git Workflow",
		Description: "Hands-on Git commands tutorial",
		HandsOn:     true,
		Sections: []Section{
			{
				Title: "Git Workflow Tutorial",
				Content: `This hands-on tutorial covers essential Git operations.

You'll practice:
• Initializing and configuring repositories
• Making commits with good messages
• Working with branches
• Understanding Preflight's Git config

Open a terminal and follow along!`,
			},
			{
				Title:   "Create a Practice Repository",
				Content: `Let's create a new repository to practice with.`,
				HandsOn: true,
				Command: "mkdir git-practice && cd git-practice && git init",
				Hint:    "This creates a new directory and initializes an empty Git repository",
			},
			{
				Title:   "Configure Your Identity",
				Content: `Set your name and email for commits.`,
				HandsOn: true,
				Command: `git config user.name "Your Name"
git config user.email "you@example.com"`,
				Hint: "These settings are stored in .git/config (local to this repo)",
			},
			{
				Title: "Git Identity with Preflight",
				Content: `Preflight manages Git identity across contexts:

WORK IDENTITY`,
				Code: `# layers/identity.work.yaml
git:
  user:
    name: "Jane Doe"
    email: "jane@company.com"
    signingkey: "~/.ssh/id_work.pub"`,
			},
			{
				Title:   "Create Your First Commit",
				Content: `Add a file and make your first commit.`,
				HandsOn: true,
				Command: `echo "# My Project" > README.md
git add README.md
git commit -m "docs: add initial README"`,
				Hint:          "We use conventional commits: type(scope): message",
				VerifyCommand: "git log --oneline -1",
			},
			{
				Title: "Conventional Commits",
				Content: `Write clear, consistent commit messages:

FORMAT: type(scope): description

TYPES
  feat     New feature
  fix      Bug fix
  docs     Documentation only
  style    Formatting (no code change)
  refactor Code change (no feature/fix)
  test     Adding tests
  chore    Maintenance tasks

EXAMPLES
  feat(auth): add OAuth login
  fix(api): handle null response
  docs: update installation guide`,
			},
			{
				Title:   "Practice: Multiple Commits",
				Content: `Make a series of commits with good messages.`,
				HandsOn: true,
				Command: `echo "## Installation" >> README.md
git add README.md
git commit -m "docs: add installation section"

echo "Run npm install" >> README.md
git add README.md
git commit -m "docs: add install command"`,
				Hint:          "Each commit should be atomic - one logical change",
				VerifyCommand: "git log --oneline",
			},
			{
				Title:   "Create a Feature Branch",
				Content: `Work on features in isolation using branches.`,
				HandsOn: true,
				Command: `git checkout -b feature/add-usage`,
				Hint:    "Branch names often follow: feature/*, fix/*, docs/*",
			},
			{
				Title:   "Make Changes on Branch",
				Content: `Add content to your feature branch.`,
				HandsOn: true,
				Command: `echo "## Usage" >> README.md
echo "Run npm start to begin" >> README.md
git add README.md
git commit -m "docs: add usage section"`,
				VerifyCommand: "git log --oneline main..HEAD",
			},
			{
				Title:   "Merge Your Branch",
				Content: `Merge your feature branch back to main.`,
				HandsOn: true,
				Command: `git checkout main
git merge feature/add-usage
git log --oneline`,
				Hint: "This creates a fast-forward merge since main hasn't changed",
			},
			{
				Title: "Git Configuration with Preflight",
				Content: `Preflight provides Git presets:

STANDARD - Basic configuration
SECURE   - With GPG signing`,
				Code: `# layers/base.yaml
git:
  preset: secure
  core:
    editor: nvim
    autocrlf: input
  alias:
    co: checkout
    br: branch
    st: status
    lg: "log --oneline --graph"

  # GPG signing
  commit:
    gpgsign: true
  user:
    signingkey: "~/.ssh/id_ed25519.pub"`,
			},
			{
				Title: "Next Steps",
				Content: `You've learned essential Git workflow!

KEY COMMANDS REVIEWED
  git init              Initialize repository
  git add <file>        Stage changes
  git commit -m "msg"   Create commit
  git checkout -b name  Create branch
  git merge branch      Merge branch

PREFLIGHT INTEGRATION
  preflight capture --include git
  preflight plan
  preflight apply

Try the shell-customization tutorial next!`,
			},
		},
		NextTopics: []string{"shell-customization", "config"},
	}
}

func getShellCustomizationTopic() TopicContent {
	return TopicContent{
		ID:          "shell-customization",
		Title:       "🛠️ Shell Customization",
		Description: "Hands-on shell setup tutorial",
		HandsOn:     true,
		Sections: []Section{
			{
				Title: "Shell Customization Tutorial",
				Content: `This tutorial covers shell customization with Preflight.

You'll learn:
• Understanding your current shell
• Adding aliases and functions
• Configuring your prompt
• Using Oh-My-Zsh and Starship

Open a terminal and follow along!`,
			},
			{
				Title:   "Check Your Current Shell",
				Content: `Let's see what shell you're using.`,
				HandsOn: true,
				Command: `echo $SHELL
echo $0`,
				Hint: "Most macOS users have zsh, Linux users may have bash",
			},
			{
				Title:   "View Your Shell Config",
				Content: `Examine your current shell configuration.`,
				HandsOn: true,
				Command: `cat ~/.zshrc | head -30`,
				Hint:    "Use ~/.bashrc if you're using bash",
			},
			{
				Title:   "Add a Simple Alias",
				Content: `Create an alias for a common command.`,
				HandsOn: true,
				Command: `echo 'alias ll="ls -la"' >> ~/.zshrc
source ~/.zshrc
ll`,
				Hint:          "Aliases save typing for frequently used commands",
				VerifyCommand: "alias ll",
			},
			{
				Title: "Common Useful Aliases",
				Content: `Aliases that boost productivity:

NAVIGATION
  alias ..="cd .."
  alias ...="cd ../.."
  alias ~="cd ~"

GIT SHORTCUTS
  alias gs="git status"
  alias gc="git commit"
  alias gp="git push"
  alias gl="git log --oneline"

SAFETY NETS
  alias rm="rm -i"
  alias mv="mv -i"
  alias cp="cp -i"

MODERN REPLACEMENTS
  alias cat="bat"
  alias ls="eza"
  alias find="fd"`,
			},
			{
				Title:   "Create a Shell Function",
				Content: `Functions allow more complex operations.`,
				HandsOn: true,
				Command: `cat >> ~/.zshrc << 'EOF'
# Create directory and cd into it
mkcd() {
  mkdir -p "$1" && cd "$1"
}
EOF
source ~/.zshrc
mkcd test-folder && pwd`,
				Hint: "Functions can take arguments and run multiple commands",
			},
			{
				Title:   "Shell Configuration with Preflight",
				Content: `Preflight manages shell config declaratively:`,
				Code: `# layers/base.yaml
shell:
  default: zsh

  aliases:
    ll: "ls -la"
    gs: "git status"
    k: "kubectl"

  functions:
    mkcd: |
      mkdir -p "$1" && cd "$1"

  env:
    EDITOR: nvim
    PAGER: less`,
			},
			{
				Title: "Oh-My-Zsh Framework",
				Content: `Oh-My-Zsh provides plugins and themes:

FEATURES
  • 300+ plugins for tools
  • 150+ themes
  • Auto-updates
  • Active community`,
				Code: `# layers/base.yaml
shell:
  framework: oh-my-zsh
  theme: robbyrussell
  plugins:
    - git           # Git aliases
    - docker        # Docker completion
    - kubectl       # Kubernetes completion
    - fzf           # Fuzzy finder integration
    - z             # Smart directory jumping`,
			},
			{
				Title:   "Check If Oh-My-Zsh Is Installed",
				Content: `See if you have Oh-My-Zsh.`,
				HandsOn: true,
				Command: `ls ~/.oh-my-zsh 2>/dev/null && echo "Oh-My-Zsh is installed" || echo "Not installed"`,
				Hint:    "Preflight can install it for you via the shell provider",
			},
			{
				Title: "Starship Prompt",
				Content: `Starship is a fast, customizable prompt:

FEATURES
  • Shows git branch and status
  • Shows current directory
  • Shows language versions
  • Blazingly fast (Rust)
  • Works with any shell`,
				Code: `# layers/base.yaml
shell:
  starship:
    enabled: true
    preset: plain-text   # or: pastel-powerline, tokyo-night

  # Custom starship config
  starship_config:
    add_newline: false
    character:
      success_symbol: "[›](bold green)"
      error_symbol: "[›](bold red)"`,
			},
			{
				Title:   "View Environment Variables",
				Content: `See what environment variables are set.`,
				HandsOn: true,
				Command: `env | grep -E "^(PATH|EDITOR|SHELL|HOME)" | sort`,
				Hint:    "Environment variables configure your shell environment",
			},
			{
				Title:   "Environment Variables with Preflight",
				Content: `Manage environment variables declaratively:`,
				Code: `# layers/base.yaml
shell:
  env:
    EDITOR: nvim
    VISUAL: nvim
    PAGER: "less -R"
    LESS: "-R"

  path:
    prepend:
      - "$HOME/.local/bin"
      - "$HOME/go/bin"
    append:
      - "/usr/local/bin"`,
			},
			{
				Title: "Next Steps",
				Content: `You've learned shell customization basics!

SUMMARY
  • Aliases save typing
  • Functions handle complex tasks
  • Oh-My-Zsh provides plugins
  • Starship creates a great prompt
  • Environment variables configure behavior

PREFLIGHT WORKFLOW`,
				Code: `# Capture current shell setup
preflight capture --include shell

# Review and apply
preflight plan
preflight apply

# Check for drift
preflight doctor`,
			},
		},
		NextTopics: []string{"workflow", "providers"},
	}
}
