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
				// Version control
				"git",
				// Text processing
				"jq", "yq", "sed", "awk", "gawk", "grep",
				// File utilities
				"tree", "fd", "fzf", "ripgrep", "rg", "bat", "eza", "exa",
				"zoxide", "trash", "trash-cli", "lsd",
				// System monitoring
				"htop", "btop", "top", "glances", "procs",
				// Networking basics
				"curl", "wget", "httpie", "aria2",
				// Multiplexers
				"tmux", "screen", "zellij",
				// Core utilities
				"coreutils", "findutils", "diffutils", "binutils",
				"gnu-sed", "gnu-tar", "gnu-time", "gnu-which", "gnu-units",
				"moreutils", "util-linux", "procps",
				// Build essentials
				"make", "cmake", "autoconf", "automake", "libtool",
				"pkg-config", "ninja", "meson", "scons",
				// Environment
				"stow", "direnv", "watchman", "entr", "fswatch",
				// Text editors (basic)
				"nano", "less", "most",
				// Misc essentials
				"ncurses", "readline", "gettext", "libiconv",
				"pcre", "pcre2", "oniguruma",
				// Benchmarking
				"hyperfine", "bench", "time",
				// Self-reference
				"preflight",
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
				"gofumpt", "goimports", "staticcheck", "golint", "revive",
				"go-task", "task", "goreleaser", "air", "cobra-cli",
				"mockgen", "gotests", "gomodifytags", "impl", "gotestsum",
				"sqlc", "migrate", "goose", "sqlboiler", "ent",
				"buf", "protoc-gen-go", "protoc-gen-go-grpc",
				"ko", "tinygo",
				// Testing tools
				"gremlins", "ginkgo", "gomega",
				// Release tools
				"relicta", "changie", "svu",
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
				"nvm", "fnm", "n", "volta", "nodenv",
				"typescript", "ts-node", "tsx", "esbuild", "swc",
				"eslint", "prettier", "biome", "oxlint",
				"vite", "webpack", "parcel", "rollup", "rspack",
				"turbo", "nx", "lerna", "rush",
				"jest", "vitest", "playwright", "cypress", "puppeteer",
				"sass", "postcss", "tailwindcss",
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
				"poetry", "pdm", "hatch", "uv", "flit",
				"ruff", "black", "isort", "flake8", "pylint", "mypy", "pyright",
				"pytest", "tox", "nox", "coverage", "hypothesis",
				"virtualenv", "pyenv", "conda", "mamba", "micromamba",
				"miniconda", "anaconda", "miniforge",
				"ipython", "jupyter", "jupyterlab", "notebook",
				"numpy", "pandas", "scipy", "matplotlib",
				"django", "flask", "fastapi", "uvicorn", "gunicorn",
				"celery", "dramatiq",
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
				"cargo-outdated", "cargo-udeps", "cargo-bloat", "cargo-machete",
				"sccache", "mold", "lld",
				"wasm-pack", "wasm-bindgen", "trunk",
				"cross", "cargo-cross", "cargo-zigbuild",
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
				"java", "openjdk", "temurin", "graalvm", "adoptopenjdk",
				"maven", "mvn", "gradle", "ant", "bazel",
				"kotlin", "scala", "clojure", "leiningen", "sbt",
				"jenv", "sdkman", "jabba",
				"spring-boot-cli", "quarkus", "micronaut",
				"groovy", "jruby",
			},
			Patterns: []string{
				"^openjdk@", // versioned java
				"^java@",
				"^temurin",
				"^adoptopenjdk",
			},
		},
		{
			Name:        "dev-ruby",
			Description: "Ruby development ecosystem",
			Exact: []string{
				"ruby", "rbenv", "ruby-build", "chruby", "rvm",
				"bundler", "gem", "rubygems",
				"rails", "rake", "rspec", "rubocop", "solargraph",
				"puma", "unicorn", "passenger",
				"jekyll", "middleman", "sinatra",
				"cocoapods", "fastlane",
			},
			Patterns: []string{
				"^ruby@",
			},
		},
		{
			Name:        "dev-php",
			Description: "PHP development ecosystem",
			Exact: []string{
				"php", "composer", "phpunit", "phpstan", "psalm",
				"laravel", "symfony-cli",
				"phpcs", "php-cs-fixer", "phpcbf",
				"pecl", "pear",
				"xdebug", "blackfire",
			},
			Patterns: []string{
				"^php@",
			},
		},
		{
			Name:        "dev-lua",
			Description: "Lua development ecosystem (including Neovim tooling)",
			Exact: []string{
				"lua", "luajit", "luarocks",
				"stylua", "selene", "lua-language-server",
				"lpeg", "luv", "luasocket", "luafilesystem",
				"fennel", "moonscript",
				// Game frameworks
				"love", "corona",
			},
			Patterns: []string{
				"^lua@",
				"^luajit@",
			},
		},
		{
			Name:        "dev-cpp",
			Description: "C/C++ development ecosystem",
			Exact: []string{
				"gcc", "g++", "clang", "llvm",
				"ccache", "distcc",
				"gdb", "lldb", "valgrind",
				"clang-format", "clang-tidy", "cppcheck", "cpplint",
				"conan", "vcpkg", "hunter",
				"boost", "abseil", "fmt",
				"doxygen", "sphinx-doc",
			},
			Patterns: []string{
				"^gcc@",
				"^llvm@",
				"^clang@",
			},
		},
		{
			Name:        "security",
			Description: "Security scanning and analysis tools",
			Exact: []string{
				"trivy", "grype", "syft", "cosign", "sigstore",
				"snyk", "semgrep", "bandit", "safety", "bearer",
				"nmap", "masscan", "nikto", "sqlmap", "gobuster", "ffuf", "nuclei",
				"age", "sops", "vault", "1password-cli", "op", "bitwarden-cli",
				"gnupg", "gpg", "pass", "gopass", "passage",
				"checkov", "tfsec", "terrascan", "kube-bench", "kubesec",
				"gitleaks", "trufflehog", "detect-secrets", "git-secrets",
				"openssl", "libressl", "gnutls",
				"pinentry", "pinentry-mac",
				// Password managers (GUI)
				"bitwarden", "1password", "lastpass", "dashlane", "keepassxc",
			},
			Patterns: []string{
				".*security.*",
				".*audit.*",
			},
		},
		{
			Name:        "crypto",
			Description: "Cryptographic libraries and tools",
			Exact: []string{
				"libsodium", "libgcrypt", "nettle", "mbedtls",
				"liboqs", "openssl", "libressl",
				"libgpg-error", "gpgme", "libassuan", "libksba", "npth",
				"age", "minisign", "signify-osx",
				"step", "cfssl", "certbot",
			},
			Patterns: []string{
				"^libgcrypt",
				"^openssl@",
			},
		},
		{
			Name:        "containers",
			Description: "Container and Kubernetes tools",
			Exact: []string{
				"docker", "docker-compose", "docker-buildx", "docker-credential-helper",
				"podman", "buildah", "skopeo", "crun", "conmon",
				"kubectl", "kubectx", "kubens", "k9s", "stern", "kubetail",
				"helm", "helmfile", "kustomize", "kompose",
				"minikube", "kind", "k3d", "k3s", "microk8s",
				"argocd", "flux", "fluxctl", "tekton",
				"istioctl", "linkerd", "cilium-cli",
				"terraform", "tofu", "opentofu", "pulumi", "cdktf",
				"ansible", "vagrant", "packer", "nomad", "consul",
				"colima", "lima", "multipass", "orbstack",
				"dive", "ctop", "lazydocker",
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
				"awscli", "aws-cli", "aws-vault", "aws-sso-util",
				"azure-cli", "az",
				"google-cloud-sdk", "gcloud",
				"doctl", "linode-cli", "hcloud", "scaleway-cli",
				"flyctl", "railway", "vercel", "netlify-cli", "wrangler",
				"heroku", "render", "neon",
				"cloudflare-wrangler", "cloudflared",
				"minio-mc", "rclone", "s3cmd",
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
				"postgresql", "postgres", "psql", "pgcli", "libpq", "pgbouncer",
				"mysql", "mysql-client", "mycli", "mariadb",
				"sqlite", "sqlite3", "litecli", "sqlcipher",
				"redis", "redis-cli", "keydb",
				"mongodb-community", "mongosh", "mongodb-database-tools",
				"duckdb", "clickhouse", "cassandra", "scylla",
				"prisma", "drizzle-kit", "atlas",
				"dbeaver", "tableplus", "datagrip", "beekeeper-studio",
				"flyway", "liquibase", "dbmate",
				"usql", "sq", "litefs",
			},
			Patterns: []string{
				"^mysql@",
				"^postgresql@",
				"^redis@",
				"^mongo",
				"^mariadb@",
			},
		},
		{
			Name:        "editor",
			Description: "Editor and IDE configurations",
			Exact: []string{
				// Terminal editors
				"neovim", "nvim", "vim", "emacs", "spacemacs", "doom-emacs",
				"helix", "kakoune", "micro", "amp", "zee",
				// GUI editors
				"visual-studio-code", "vscode", "code", "cursor", "zed",
				"sublime-text", "atom",
				// IDEs
				"jetbrains-toolbox", "intellij-idea", "goland", "webstorm",
				"pycharm", "clion", "rider", "phpstorm", "rubymine", "datagrip",
				"android-studio", "xcode",
				// Neovim ecosystem (tree-sitter, etc.)
				"tree-sitter", "universal-ctags", "ctags",
				// LSP tools
				"efm-langserver", "diagnostic-languageserver",
			},
			Patterns: []string{
				"^tree-sitter@", // versioned tree-sitter
			},
		},
		{
			Name:        "shell",
			Description: "Shell and terminal configuration",
			Exact: []string{
				// Shells
				"zsh", "bash", "fish", "nushell", "elvish", "xonsh", "ion", "dash",
				// Shell frameworks and prompts
				"oh-my-zsh", "oh-my-posh", "starship", "powerlevel10k", "pure",
				"antigen", "antibody", "zinit", "zplug", "sheldon",
				"fisher", "oh-my-fish",
				// Shell plugins
				"zsh-autosuggestions", "zsh-syntax-highlighting", "zsh-completions",
				"zsh-history-substring-search", "zsh-you-should-use",
				// Terminal emulators
				"alacritty", "kitty", "wezterm", "iterm2", "hyper",
				"ghostty", "warp", "tabby", "rio",
				// Terminal utilities
				"terminal-notifier", "reattach-to-user-namespace",
				// Web-based terminal
				"ttyd", "gotty", "wetty",
			},
			Patterns: []string{
				"^zsh-",
				"^fish-",
				"^bash-",
				"^oh-my-",
			},
		},
		{
			Name:        "git",
			Description: "Git and version control tools",
			Exact: []string{
				// Git hosting CLIs
				"gh", "hub", "glab", "bitbucket-cli",
				// Git extensions
				"git-lfs", "git-filter-repo", "git-absorb", "git-branchless",
				"git-extras", "git-flow", "git-flow-avh",
				"git-secret", "git-crypt",
				"git-open", "git-recent", "git-standup", "git-town",
				// Diff tools
				"git-delta", "delta", "diff-so-fancy", "difftastic",
				// Git TUIs
				"lazygit", "tig", "gitui", "grv",
				// Commit tools
				"pre-commit", "husky", "commitizen", "commitlint",
				"conventional-changelog", "semantic-release",
				// Merge tools
				"meld", "kdiff3", "p4merge",
				// Other VCS
				"svn", "subversion", "mercurial", "hg", "fossil",
				"bfg", "git-sizer",
			},
			Prefixes: []string{
				"git-",
			},
		},
		{
			Name:        "media",
			Description: "Media processing and creative tools",
			Exact: []string{
				// Video
				"ffmpeg", "ffprobe", "ffplay",
				"handbrake", "handbrake-cli",
				"yt-dlp", "youtube-dl",
				// Image
				"imagemagick", "graphicsmagick", "vips", "libvips",
				"gifsicle", "gifski", "pngquant", "jpegoptim", "oxipng",
				"exiftool", "dcraw",
				// OCR
				"tesseract", "tesseract-lang", "ocrmypdf",
				// Audio
				"sox", "lame", "flac", "opus", "vorbis-tools",
				"mpv", "vlc",
				// Recording
				"asciinema", "vhs", "agg", "terminalizer",
				"obs", "obs-studio",
				// Streaming hardware
				"elgato-camera-hub", "elgato-stream-deck", "elgato-control-center",
				"elgato-wave-link",
				// Creative
				"inkscape", "gimp", "blender", "krita",
				"darktable", "rawtherapee",
				// Design tools
				"figma", "sketch", "affinity-designer", "affinity-photo",
				"canva", "adobe-creative-cloud",
				// Diagrams
				"graphviz", "plantuml", "mermaid-cli", "d2",
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
				"^sf-",
				"^ttf-",
				"^otf-",
			},
			Exact: []string{
				"fontconfig", "fontforge", "fonttools",
				"freetype", "harfbuzz", "pango", "cairo",
			},
		},
		{
			Name:        "ai",
			Description: "AI and machine learning tools",
			Exact: []string{
				// Local LLMs
				"ollama", "ollama-app", "llama-cpp", "llamafile",
				"llm", "ttok", "strip-tags",
				// AI assistants
				"claude", "chatgpt", "github-copilot",
				// Cloud APIs
				"openai", "anthropic", "cohere",
				// ML frameworks
				"pytorch", "tensorflow", "jax", "onnx",
				"scikit-learn", "xgboost", "lightgbm",
				// ML tools
				"huggingface-cli", "transformers", "sentence-transformers",
				"langchain", "llamaindex", "chromadb", "pgvector",
				// Image generation
				"stable-diffusion", "comfyui", "automatic1111",
				// Speech
				"whisper", "whisper-cpp",
			},
			Patterns: []string{
				".*llm.*",
				".*openai.*",
				".*whisper.*",
			},
		},
		{
			Name:        "networking",
			Description: "Networking and HTTP tools",
			Exact: []string{
				// HTTP clients
				"httpie", "xh", "curlie", "hurl",
				"grpcurl", "evans", "postman",
				// Network utilities
				"netcat", "nc", "socat", "ncat",
				"tcpdump", "wireshark", "tshark",
				"iperf3", "mtr", "traceroute", "dig", "dog",
				"bandwhich", "gping", "speedtest-cli",
				// DNS
				"bind", "dnsmasq", "dnscrypt-proxy", "stubby",
				// Proxy/tunnel
				"ngrok", "localtunnel", "frp",
				"mitmproxy", "charles", "proxyman",
				"sshuttle", "wireguard-tools", "tailscale",
				// SSH
				"openssh", "libssh", "libssh2", "ssh-copy-id",
				"mosh", "autossh", "sshpass", "sshfs",
			},
		},
		{
			Name:        "compression",
			Description: "Compression and archiving tools",
			Exact: []string{
				// Modern compression
				"zstd", "lz4", "brotli", "snappy", "lzo",
				// Classic compression
				"xz", "gzip", "bzip2", "lzip", "lzop",
				"zlib", "pigz", "pbzip2", "pixz",
				// Archivers
				"p7zip", "unrar", "unar", "atool",
				"zip", "unzip", "libarchive",
				"gtar", "gnu-tar", "pax",
			},
			Patterns: []string{
				"^lib.*z$",
			},
		},
		{
			Name:        "libs",
			Description: "Shared libraries and dependencies",
			Exact: []string{
				// Image libraries
				"libpng", "libjpeg", "libjpeg-turbo", "libtiff", "libwebp",
				"giflib", "libheif", "libavif", "libraw", "libopenraw",
				// XML/JSON
				"libxml2", "libxslt", "libxmlsec1",
				"jansson", "jemalloc",
				// UI libraries
				"gtk+3", "gtk4", "qt", "qt5", "qt6",
				"wxwidgets", "fltk",
				// System libraries
				"libevent", "libuv", "libev", "c-ares",
				"libyaml", "libunistring", "libffi",
				"glib", "gmp", "mpfr", "mpc",
				// Apple/macOS
				"libiconv", "icu4c", "gettext",
			},
			Patterns: []string{
				"^lib",
			},
			Prefixes: []string{
				"lib",
			},
		},
		{
			Name:        "productivity",
			Description: "Productivity and utility applications",
			Exact: []string{
				// Launchers
				"raycast", "alfred", "launchbar",
				// Automation
				"hammerspoon", "keyboard-maestro", "bettertouchtool",
				// Window management
				"rectangle", "spectacle", "amethyst", "yabai", "skhd",
				"magnet", "moom", "divvy",
				// Input
				"karabiner-elements",
				// Menu bar
				"bartender", "dozer", "hidden-bar", "vanilla",
				// Display control
				"monitorcontrol", "lunar", "displaylink",
				// Notes/PKM
				"obsidian", "notion", "logseq", "roam", "craft",
				"bear", "apple-notes", "joplin", "zettlr",
				// Tasks
				"todoist", "things", "omnifocus", "ticktick", "reminders",
				// Clipboard
				"maccy", "copyq", "pasta", "clipy",
			},
		},
		{
			Name:        "communication",
			Description: "Communication and collaboration apps",
			Exact: []string{
				// Chat
				"slack", "discord", "teams", "mattermost",
				"element", "matrix", "irc",
				// Video
				"zoom", "webex", "google-meet",
				// Messaging
				"telegram", "signal", "whatsapp", "messenger",
				// Email
				"mailspring", "thunderbird", "spark", "mimestream", "canary",
				"postbox", "airmail",
				// Calendar
				"fantastical", "itsycal", "calendly",
			},
		},
		{
			Name:        "browsers",
			Description: "Web browsers",
			Exact: []string{
				"google-chrome", "firefox", "brave-browser",
				"arc", "safari", "microsoft-edge", "edge", "vivaldi", "opera",
				"chromium", "librewolf", "tor-browser", "mullvad-browser",
				"orion", "min", "qutebrowser",
				// Development browsers
				"firefox-developer-edition", "google-chrome-canary",
				"safari-technology-preview",
			},
		},
		{
			Name:        "docs",
			Description: "Documentation and writing tools",
			Exact: []string{
				// Markdown
				"pandoc", "mdbook", "markdownlint-cli", "markdown-link-check",
				"glow", "mdcat", "grip",
				// Documentation generators
				"mkdocs", "docusaurus", "gitbook", "vuepress", "docsify",
				"typedoc", "jsdoc", "godoc",
				// Writing tools
				"vale", "proselint", "alex", "write-good",
				"languagetool", "grammarly",
				// Office
				"libreoffice", "onlyoffice",
				// PDF
				"poppler", "pdfgrep", "pdftk", "qpdf",
				"pandoc-pdf", "wkhtmltopdf", "weasyprint",
				// LaTeX
				"texlive", "mactex", "tectonic", "latexmk",
			},
		},
		{
			Name:        "testing",
			Description: "Testing and quality assurance tools",
			Exact: []string{
				// Load testing
				"k6", "locust", "vegeta", "hey", "ab", "wrk", "bombardier",
				// API testing
				"postman", "insomnia", "hoppscotch", "bruno",
				"httpie", "restclient",
				// Browser testing
				"playwright", "selenium", "cypress", "puppeteer",
				"chromedriver", "geckodriver",
				// Fuzzing
				"afl", "afl++", "honggfuzz",
				// Mocking
				"mockoon", "wiremock", "prism",
				// Contract testing
				"pact", "dredd", "schemathesis",
			},
		},
		{
			Name:        "data",
			Description: "Data processing and analysis tools",
			Exact: []string{
				// CLI data tools
				"csvkit", "xsv", "mlr", "visidata",
				"q", "textql", "trdsql",
				// Data formats
				"protobuf", "protoc", "flatbuffers", "capnproto", "avro",
				"parquet-tools", "arrow",
				// ETL
				"dbt", "dagster", "airflow", "prefect",
				"singer", "meltano", "airbyte",
				// Queues
				"kafka", "rabbitmq", "nats", "pulsar", "zeromq",
				// Caching
				"memcached", "hazelcast", "varnish",
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
