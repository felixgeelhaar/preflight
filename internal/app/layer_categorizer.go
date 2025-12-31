package app

import (
	"regexp"
	"sort"
	"strings"
)

// LayerCategory defines a category for organizing captured items.
type LayerCategory struct {
	Name        string
	Description string
	Patterns    []string // Package name patterns (case-insensitive)
	Prefixes    []string // Package name prefixes
	Exact       []string // Exact package matches
}

// LayerCategorizer organizes captured items into logical layer groups.
type LayerCategorizer struct {
	categories []LayerCategory
}

// NewLayerCategorizer creates a categorizer with default categories.
func NewLayerCategorizer() *LayerCategorizer {
	return &LayerCategorizer{
		categories: defaultCategories(),
	}
}

// CategorizedItems represents items organized by layer.
type CategorizedItems struct {
	Layers        map[string][]CapturedItem
	LayerOrder    []string
	Uncategorized []CapturedItem
}

// Categorize organizes captured items into layer groups.
func (c *LayerCategorizer) Categorize(items []CapturedItem) *CategorizedItems {
	result := &CategorizedItems{
		Layers:     make(map[string][]CapturedItem),
		LayerOrder: make([]string, 0),
	}

	// Track which items have been categorized
	categorized := make(map[int]bool)

	// Process each category in order
	for _, cat := range c.categories {
		layerItems := make([]CapturedItem, 0)

		for i, item := range items {
			if categorized[i] {
				continue
			}

			if c.matchesCategory(item, cat) {
				layerItems = append(layerItems, item)
				categorized[i] = true
			}
		}

		if len(layerItems) > 0 {
			result.Layers[cat.Name] = layerItems
			result.LayerOrder = append(result.LayerOrder, cat.Name)
		}
	}

	// Collect uncategorized items
	for i, item := range items {
		if !categorized[i] {
			result.Uncategorized = append(result.Uncategorized, item)
		}
	}

	// Add uncategorized to "misc" layer if any
	if len(result.Uncategorized) > 0 {
		result.Layers["misc"] = result.Uncategorized
		result.LayerOrder = append(result.LayerOrder, "misc")
	}

	return result
}

// matchesCategory checks if an item matches a category's patterns.
func (c *LayerCategorizer) matchesCategory(item CapturedItem, cat LayerCategory) bool {
	name := strings.ToLower(item.Name)

	// Check exact matches first
	for _, exact := range cat.Exact {
		if strings.EqualFold(name, exact) {
			return true
		}
	}

	// Check prefixes
	for _, prefix := range cat.Prefixes {
		if strings.HasPrefix(name, strings.ToLower(prefix)) {
			return true
		}
	}

	// Check patterns (regex)
	for _, pattern := range cat.Patterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		if re.MatchString(name) {
			return true
		}
	}

	return false
}

// Summary returns a summary of categorized items.
func (ci *CategorizedItems) Summary() []LayerSummary {
	summaries := make([]LayerSummary, 0, len(ci.Layers))
	for _, name := range ci.LayerOrder {
		items := ci.Layers[name]
		summaries = append(summaries, LayerSummary{
			Name:  name,
			Count: len(items),
		})
	}
	return summaries
}

// LayerSummary provides a count summary for a layer.
type LayerSummary struct {
	Name  string
	Count int
}

// defaultCategories returns the built-in categorization rules.
func defaultCategories() []LayerCategory {
	return []LayerCategory{
		{
			Name:        "base",
			Description: "Core CLI utilities and essential tools",
			Exact: []string{
				"git", "curl", "wget", "jq", "yq", "tree", "htop", "btop",
				"ripgrep", "rg", "fd", "fzf", "bat", "eza", "exa", "zoxide",
				"tmux", "screen", "coreutils", "findutils", "gnu-sed", "gawk",
				"gnupg", "gpg", "openssl", "openssh", "ssh-copy-id",
				"make", "cmake", "autoconf", "automake", "pkg-config",
				"stow", "direnv", "watchman", "entr",
			},
			Prefixes: []string{
				"gnu-",
			},
		},
		{
			Name:        "dev-go",
			Description: "Go development ecosystem",
			Exact: []string{
				"go", "gopls", "golangci-lint", "delve", "dlv",
				"gofumpt", "goimports", "staticcheck", "golint",
				"go-task", "goreleaser", "air", "cobra-cli",
				"mockgen", "gotests", "gomodifytags", "impl",
				"sqlc", "migrate", "goose", "sqlboiler",
			},
			Patterns: []string{
				"^go@",     // versioned go
				"^go-",     // go-* tools
				"^golang-", // golang-* tools
			},
		},
		{
			Name:        "dev-node",
			Description: "Node.js and JavaScript ecosystem",
			Exact: []string{
				"node", "nodejs", "npm", "yarn", "pnpm", "bun", "deno",
				"nvm", "fnm", "n", "volta",
				"typescript", "ts-node", "tsx", "esbuild", "swc",
				"eslint", "prettier", "biome",
				"vite", "webpack", "parcel", "rollup",
				"turbo", "nx", "lerna",
			},
			Patterns: []string{
				"^node@", // versioned node
				"^@",     // scoped npm packages
			},
		},
		{
			Name:        "dev-python",
			Description: "Python development ecosystem",
			Exact: []string{
				"python", "python3", "pip", "pipx", "pipenv",
				"poetry", "pdm", "hatch", "uv",
				"ruff", "black", "isort", "flake8", "pylint", "mypy", "pyright",
				"pytest", "tox", "nox", "coverage",
				"virtualenv", "pyenv", "conda", "mamba", "micromamba",
				"ipython", "jupyter", "jupyterlab",
			},
			Patterns: []string{
				"^python@", // versioned python
				"^py",      // py* tools
			},
		},
		{
			Name:        "dev-rust",
			Description: "Rust development ecosystem",
			Exact: []string{
				"rust", "rustup", "cargo", "rustc",
				"rust-analyzer", "rustfmt", "clippy",
				"cargo-watch", "cargo-edit", "cargo-audit", "cargo-deny",
				"cargo-nextest", "cargo-expand", "cargo-flamegraph",
				"sccache", "mold", "lld",
			},
			Patterns: []string{
				"^rust@",  // versioned rust
				"^cargo-", // cargo-* tools
			},
		},
		{
			Name:        "dev-java",
			Description: "JVM and Java ecosystem",
			Exact: []string{
				"java", "openjdk", "temurin", "graalvm",
				"maven", "mvn", "gradle", "ant",
				"kotlin", "scala", "clojure", "leiningen",
				"jenv", "sdkman", "jabba",
				"spring-boot-cli", "quarkus",
			},
			Patterns: []string{
				"^openjdk@", // versioned java
				"^java@",
				"^temurin",
			},
		},
		{
			Name:        "security",
			Description: "Security scanning and analysis tools",
			Exact: []string{
				"trivy", "grype", "syft", "cosign", "sigstore",
				"snyk", "semgrep", "bandit", "safety",
				"nmap", "nikto", "sqlmap", "gobuster", "ffuf", "nuclei",
				"age", "sops", "vault", "1password-cli", "op",
				"gnupg", "gpg", "pass", "gopass",
				"checkov", "tfsec", "terrascan", "kube-bench",
				"gitleaks", "trufflehog", "detect-secrets",
			},
			Patterns: []string{
				".*security.*",
				".*audit.*",
				".*scan.*",
			},
		},
		{
			Name:        "containers",
			Description: "Container and Kubernetes tools",
			Exact: []string{
				"docker", "docker-compose", "docker-buildx",
				"podman", "buildah", "skopeo",
				"kubectl", "kubectx", "kubens", "k9s", "stern", "kubetail",
				"helm", "helmfile", "kustomize",
				"minikube", "kind", "k3d", "k3s",
				"argocd", "flux", "fluxctl",
				"istioctl", "linkerd",
				"terraform", "tofu", "opentofu", "pulumi",
				"ansible", "vagrant", "packer",
			},
			Patterns: []string{
				"^kube",
				"^docker-",
				"^helm-",
			},
		},
		{
			Name:        "cloud",
			Description: "Cloud provider CLIs and tools",
			Exact: []string{
				"awscli", "aws-cli", "aws",
				"azure-cli", "az",
				"google-cloud-sdk", "gcloud",
				"doctl", "linode-cli", "hcloud",
				"flyctl", "railway", "vercel", "netlify-cli",
				"heroku", "render",
			},
			Patterns: []string{
				"^aws-",
				"^azure-",
				"^gcloud-",
			},
		},
		{
			Name:        "database",
			Description: "Database clients and tools",
			Exact: []string{
				"postgresql", "postgres", "psql", "pgcli", "libpq",
				"mysql", "mysql-client", "mycli",
				"sqlite", "sqlite3", "litecli",
				"redis", "redis-cli",
				"mongodb-community", "mongosh", "mongodb-database-tools",
				"duckdb", "clickhouse", "cassandra",
				"prisma", "drizzle-kit",
				"dbeaver", "tableplus", "datagrip",
			},
			Patterns: []string{
				"^mysql@",
				"^postgresql@",
				"^redis@",
				"^mongo",
			},
		},
		{
			Name:        "editor",
			Description: "Editor and IDE configurations",
			Exact: []string{
				"neovim", "nvim", "vim", "emacs",
				"visual-studio-code", "vscode", "code",
				"sublime-text", "atom",
				"jetbrains-toolbox", "intellij-idea", "goland", "webstorm",
				"helix", "kakoune", "micro", "nano",
			},
		},
		{
			Name:        "shell",
			Description: "Shell and terminal configuration",
			Exact: []string{
				"zsh", "bash", "fish", "nushell", "elvish", "xonsh",
				"oh-my-zsh", "oh-my-posh", "starship", "powerlevel10k",
				"zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions",
				"alacritty", "kitty", "wezterm", "iterm2", "hyper",
				"ghostty", "warp",
			},
			Patterns: []string{
				"^zsh-",
			},
		},
		{
			Name:        "git",
			Description: "Git and version control tools",
			Exact: []string{
				"gh", "hub", "glab", "git-lfs",
				"git-delta", "delta", "diff-so-fancy",
				"lazygit", "tig", "gitui",
				"pre-commit", "husky", "commitizen",
				"git-flow", "git-extras",
			},
			Prefixes: []string{
				"git-",
			},
		},
		{
			Name:        "media",
			Description: "Media processing and creative tools",
			Exact: []string{
				"ffmpeg", "ffprobe",
				"imagemagick", "graphicsmagick", "vips",
				"gifsicle", "gifski", "asciinema", "vhs", "agg",
				"yt-dlp", "youtube-dl",
				"sox", "lame", "flac", "opus",
				"inkscape", "gimp", "blender",
				"obs", "obs-studio",
			},
		},
		{
			Name:        "fonts",
			Description: "Fonts and typography",
			Patterns: []string{
				"^font-",
				"-font$",
				"^nerd-font",
				"-nerd-font",
			},
			Exact: []string{
				"fontconfig", "fontforge",
			},
		},
		{
			Name:        "ai",
			Description: "AI and machine learning tools",
			Exact: []string{
				"ollama", "llm", "ttok",
				"openai", "anthropic",
				"pytorch", "tensorflow", "jax",
				"huggingface-cli", "transformers",
				"langchain", "llamaindex",
				"stable-diffusion", "comfyui",
			},
			Patterns: []string{
				".*llm.*",
				".*openai.*",
			},
		},
		{
			Name:        "productivity",
			Description: "Productivity and utility applications",
			Exact: []string{
				"raycast", "alfred", "hammerspoon",
				"rectangle", "spectacle", "amethyst", "yabai", "skhd",
				"karabiner-elements",
				"bartender", "dozer", "hidden-bar",
				"obsidian", "notion", "logseq", "roam",
				"todoist", "things", "omnifocus",
			},
		},
		{
			Name:        "communication",
			Description: "Communication and collaboration apps",
			Exact: []string{
				"slack", "discord", "zoom", "teams",
				"telegram", "signal", "whatsapp",
				"mailspring", "thunderbird", "spark",
			},
		},
		{
			Name:        "browsers",
			Description: "Web browsers",
			Exact: []string{
				"google-chrome", "firefox", "brave-browser",
				"arc", "safari", "edge", "vivaldi", "opera",
				"chromium", "librewolf", "tor-browser",
			},
		},
	}
}

// GetLayerDescription returns the description for a layer name.
func (c *LayerCategorizer) GetLayerDescription(name string) string {
	for _, cat := range c.categories {
		if cat.Name == name {
			return cat.Description
		}
	}
	if name == "misc" {
		return "Uncategorized items"
	}
	return ""
}

// AvailableCategories returns all category names in order.
func (c *LayerCategorizer) AvailableCategories() []string {
	names := make([]string, len(c.categories))
	for i, cat := range c.categories {
		names[i] = cat.Name
	}
	return names
}

// SortItemsAlphabetically sorts items within each layer alphabetically.
func (ci *CategorizedItems) SortItemsAlphabetically() {
	for name := range ci.Layers {
		items := ci.Layers[name]
		sort.Slice(items, func(i, j int) bool {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		})
		ci.Layers[name] = items
	}
}
