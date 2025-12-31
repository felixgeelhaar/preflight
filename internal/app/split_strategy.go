package app

import (
	"context"
	"fmt"
	"strings"
)

// SplitStrategy defines how captured items should be organized into layers.
type SplitStrategy string

const (
	// SplitByCategory uses fine-grained categories (base, dev-go, security, etc.)
	SplitByCategory SplitStrategy = "category"
	// SplitByLanguage groups by programming language (go, node, python, etc.)
	SplitByLanguage SplitStrategy = "language"
	// SplitByStack groups by tech stack (frontend, backend, devops, data, security)
	SplitByStack SplitStrategy = "stack"
	// SplitByProvider groups by provider (brew, git, shell, vscode, etc.)
	SplitByProvider SplitStrategy = "provider"
)

// ValidSplitStrategies returns all valid split strategy values.
func ValidSplitStrategies() []string {
	return []string{
		string(SplitByCategory),
		string(SplitByLanguage),
		string(SplitByStack),
		string(SplitByProvider),
	}
}

// ParseSplitStrategy parses a string into a SplitStrategy.
func ParseSplitStrategy(s string) (SplitStrategy, error) {
	switch strings.ToLower(s) {
	case "category", "categories":
		return SplitByCategory, nil
	case "language", "languages", "lang":
		return SplitByLanguage, nil
	case "stack", "stacks":
		return SplitByStack, nil
	case "provider", "providers":
		return SplitByProvider, nil
	default:
		return "", fmt.Errorf("invalid split strategy: %q (valid: %s)", s, strings.Join(ValidSplitStrategies(), ", "))
	}
}

// StrategyCategorizer creates a categorizer for the given strategy.
func StrategyCategorizer(strategy SplitStrategy) *LayerCategorizer {
	switch strategy {
	case SplitByLanguage:
		return &LayerCategorizer{categories: languageCategories()}
	case SplitByStack:
		return &LayerCategorizer{categories: stackCategories()}
	case SplitByProvider:
		// Provider strategy doesn't use pattern matching
		return &LayerCategorizer{categories: []LayerCategory{}}
	default:
		return NewLayerCategorizer() // category strategy
	}
}

// languageCategories returns categories grouped by programming language.
func languageCategories() []LayerCategory {
	return []LayerCategory{
		{
			Name:        "go",
			Description: "Go programming language and tools",
			Exact: []string{
				"go", "gopls", "golangci-lint", "delve", "dlv",
				"gofumpt", "goimports", "staticcheck", "golint",
				"go-task", "goreleaser", "air", "cobra-cli",
				"mockgen", "gotests", "gomodifytags", "impl",
				"sqlc", "migrate", "goose", "sqlboiler",
			},
			Patterns: []string{"^go@", "^go-", "^golang-"},
		},
		{
			Name:        "node",
			Description: "Node.js, JavaScript, and TypeScript",
			Exact: []string{
				"node", "nodejs", "npm", "yarn", "pnpm", "bun", "deno",
				"nvm", "fnm", "n", "volta",
				"typescript", "ts-node", "tsx", "esbuild", "swc",
				"eslint", "prettier", "biome",
				"vite", "webpack", "parcel", "rollup",
				"turbo", "nx", "lerna",
			},
			Patterns: []string{"^node@", "^@"},
		},
		{
			Name:        "python",
			Description: "Python programming language and tools",
			Exact: []string{
				"python", "python3", "pip", "pipx", "pipenv",
				"poetry", "pdm", "hatch", "uv",
				"ruff", "black", "isort", "flake8", "pylint", "mypy", "pyright",
				"pytest", "tox", "nox", "coverage",
				"virtualenv", "pyenv", "conda", "mamba", "micromamba",
				"ipython", "jupyter", "jupyterlab",
			},
			Patterns: []string{"^python@", "^py"},
		},
		{
			Name:        "rust",
			Description: "Rust programming language and tools",
			Exact: []string{
				"rust", "rustup", "cargo", "rustc",
				"rust-analyzer", "rustfmt", "clippy",
				"cargo-watch", "cargo-edit", "cargo-audit", "cargo-deny",
				"cargo-nextest", "cargo-expand", "cargo-flamegraph",
				"sccache", "mold", "lld",
			},
			Patterns: []string{"^rust@", "^cargo-"},
		},
		{
			Name:        "java",
			Description: "Java, Kotlin, and JVM languages",
			Exact: []string{
				"java", "openjdk", "temurin", "graalvm",
				"maven", "mvn", "gradle", "ant",
				"kotlin", "scala", "clojure", "leiningen",
				"jenv", "sdkman", "jabba",
				"spring-boot-cli", "quarkus",
			},
			Patterns: []string{"^openjdk@", "^java@", "^temurin"},
		},
		{
			Name:        "ruby",
			Description: "Ruby programming language and tools",
			Exact: []string{
				"ruby", "rbenv", "rvm", "chruby",
				"bundler", "gem", "rake",
				"rails", "rubocop", "solargraph",
			},
			Patterns: []string{"^ruby@"},
		},
		{
			Name:        "swift",
			Description: "Swift and iOS development",
			Exact: []string{
				"swift", "swiftlint", "swiftformat",
				"cocoapods", "carthage", "fastlane",
				"xcpretty", "periphery", "xcodegen",
			},
		},
		{
			Name:        "cpp",
			Description: "C, C++, and systems programming",
			Exact: []string{
				"gcc", "g++", "clang", "llvm",
				"cmake", "make", "ninja", "meson",
				"autoconf", "automake", "libtool",
				"gdb", "lldb", "valgrind",
				"conan", "vcpkg",
			},
			Patterns: []string{"^llvm@", "^gcc@"},
		},
		{
			Name:        "dotnet",
			Description: ".NET and C# development",
			Exact: []string{
				"dotnet", "mono", "nuget",
				"omnisharp", "dotnet-sdk",
			},
			Patterns: []string{"^dotnet@"},
		},
		{
			Name:        "php",
			Description: "PHP programming language and tools",
			Exact: []string{
				"php", "composer", "phpunit",
				"phpstan", "psalm", "phan",
				"laravel", "symfony",
			},
			Patterns: []string{"^php@"},
		},
		{
			Name:        "elixir",
			Description: "Elixir and Erlang",
			Exact: []string{
				"elixir", "erlang", "mix",
				"phoenix", "hex",
			},
			Patterns: []string{"^elixir@", "^erlang@"},
		},
		{
			Name:        "tools",
			Description: "General development tools",
			Exact: []string{
				"git", "curl", "wget", "jq", "yq", "tree", "htop", "btop",
				"ripgrep", "rg", "fd", "fzf", "bat", "eza", "exa", "zoxide",
				"tmux", "screen", "stow", "direnv", "watchman", "entr",
				"neovim", "nvim", "vim", "emacs", "helix",
				"docker", "kubectl", "helm", "terraform",
			},
		},
	}
}

// stackCategories returns categories grouped by tech stack.
func stackCategories() []LayerCategory {
	return []LayerCategory{
		{
			Name:        "frontend",
			Description: "Frontend and UI development",
			Exact: []string{
				"node", "nodejs", "npm", "yarn", "pnpm", "bun",
				"typescript", "esbuild", "swc", "vite", "webpack", "parcel",
				"eslint", "prettier", "biome",
				"react", "vue", "angular", "svelte",
			},
			Patterns: []string{"^node@", "^@"},
		},
		{
			Name:        "backend",
			Description: "Backend and API development",
			Exact: []string{
				"go", "gopls", "golangci-lint", "delve",
				"python", "poetry", "ruff", "mypy",
				"rust", "cargo", "rust-analyzer",
				"java", "openjdk", "maven", "gradle",
				"ruby", "rails", "bundler",
				"php", "composer", "laravel",
				"elixir", "phoenix",
				"deno", "bun",
			},
			Patterns: []string{"^go@", "^python@", "^rust@", "^openjdk@", "^ruby@", "^php@"},
		},
		{
			Name:        "devops",
			Description: "DevOps and infrastructure",
			Exact: []string{
				"docker", "docker-compose", "docker-buildx",
				"podman", "buildah", "skopeo",
				"kubectl", "kubectx", "kubens", "k9s", "stern",
				"helm", "helmfile", "kustomize",
				"minikube", "kind", "k3d",
				"terraform", "tofu", "pulumi",
				"ansible", "vagrant", "packer",
				"argocd", "flux", "istioctl",
				"awscli", "aws-cli", "azure-cli", "gcloud",
			},
			Patterns: []string{"^kube", "^docker-", "^helm-", "^aws-", "^azure-"},
		},
		{
			Name:        "data",
			Description: "Data engineering and science",
			Exact: []string{
				"python", "jupyter", "jupyterlab", "ipython",
				"pandas", "numpy", "scikit-learn",
				"pytorch", "tensorflow", "jax",
				"duckdb", "clickhouse", "spark",
				"dbt", "airflow", "dagster",
				"postgresql", "mysql", "redis", "mongodb",
			},
			Patterns: []string{"^python@"},
		},
		{
			Name:        "security",
			Description: "Security and compliance",
			Exact: []string{
				"trivy", "grype", "syft", "cosign",
				"snyk", "semgrep", "bandit", "safety",
				"nmap", "nikto", "sqlmap", "gobuster", "nuclei",
				"age", "sops", "vault", "1password-cli",
				"checkov", "tfsec", "terrascan", "kube-bench",
				"gitleaks", "trufflehog",
			},
			Patterns: []string{".*security.*", ".*audit.*", ".*scan.*"},
		},
		{
			Name:        "mobile",
			Description: "Mobile app development",
			Exact: []string{
				"swift", "swiftlint", "swiftformat",
				"cocoapods", "carthage", "fastlane",
				"kotlin", "gradle",
				"flutter", "dart",
				"react-native", "expo",
			},
		},
		{
			Name:        "ai",
			Description: "AI and machine learning",
			Exact: []string{
				"ollama", "llm", "ttok",
				"pytorch", "tensorflow", "jax",
				"huggingface-cli", "transformers",
				"langchain", "llamaindex",
				"chromadb", "pinecone",
			},
			Patterns: []string{".*llm.*", ".*openai.*"},
		},
		{
			Name:        "tools",
			Description: "General development tools",
			Exact: []string{
				"git", "gh", "lazygit",
				"curl", "wget", "jq", "yq",
				"ripgrep", "fd", "fzf", "bat",
				"neovim", "vim", "emacs",
				"tmux", "zsh", "starship",
			},
		},
	}
}

// AICategorizationRequest represents a request to categorize packages with AI.
type AICategorizationRequest struct {
	Items           []CapturedItem
	AvailableLayers []string
	Strategy        SplitStrategy
}

// AICategorizationResult represents the AI's categorization suggestion.
type AICategorizationResult struct {
	Categorizations map[string]string // package name -> layer name
	Reasoning       map[string]string // package name -> why this category
}

// AICategorizer is an interface for AI-assisted categorization.
type AICategorizer interface {
	Categorize(ctx context.Context, req AICategorizationRequest) (*AICategorizationResult, error)
}

// CategorizeWithAI enhances categorization with AI for uncategorized items.
func CategorizeWithAI(ctx context.Context, categorized *CategorizedItems, ai AICategorizer, strategy SplitStrategy) error {
	if len(categorized.Uncategorized) == 0 {
		return nil
	}

	// Get available layer names
	availableLayers := make([]string, 0, len(categorized.LayerOrder))
	for _, name := range categorized.LayerOrder {
		if name != "misc" {
			availableLayers = append(availableLayers, name)
		}
	}

	// Request AI categorization
	req := AICategorizationRequest{
		Items:           categorized.Uncategorized,
		AvailableLayers: availableLayers,
		Strategy:        strategy,
	}

	result, err := ai.Categorize(ctx, req)
	if err != nil {
		return fmt.Errorf("AI categorization failed: %w", err)
	}

	// Apply AI categorizations
	newMisc := make([]CapturedItem, 0)
	for _, item := range categorized.Uncategorized {
		if layerName, ok := result.Categorizations[item.Name]; ok && layerName != "" {
			// Add to existing or new layer
			if _, exists := categorized.Layers[layerName]; !exists {
				categorized.Layers[layerName] = make([]CapturedItem, 0)
				// Insert before misc in layer order
				newOrder := make([]string, 0, len(categorized.LayerOrder)+1)
				for _, name := range categorized.LayerOrder {
					if name == "misc" {
						newOrder = append(newOrder, layerName)
					}
					newOrder = append(newOrder, name)
				}
				categorized.LayerOrder = newOrder
			}
			categorized.Layers[layerName] = append(categorized.Layers[layerName], item)
		} else {
			newMisc = append(newMisc, item)
		}
	}

	// Update misc layer
	categorized.Uncategorized = newMisc
	if len(newMisc) > 0 {
		categorized.Layers["misc"] = newMisc
	} else {
		delete(categorized.Layers, "misc")
		// Remove misc from layer order
		newOrder := make([]string, 0, len(categorized.LayerOrder))
		for _, name := range categorized.LayerOrder {
			if name != "misc" {
				newOrder = append(newOrder, name)
			}
		}
		categorized.LayerOrder = newOrder
	}

	return nil
}
