#!/bin/bash
# Record terminal demos for Preflight documentation
# Outputs: SVG (for docs) and GIF (for README/social)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEMOS_DIR="$PROJECT_ROOT/website/public/demos"
RECORDINGS_DIR="$PROJECT_ROOT/.recordings"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default settings
COLS=100
ROWS=30
THEME="monokai"

mkdir -p "$DEMOS_DIR/svg" "$DEMOS_DIR/gif" "$RECORDINGS_DIR"

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

check_dependencies() {
    info "Checking dependencies..."

    if ! command -v asciinema &> /dev/null; then
        error "asciinema not found. Install with: brew install asciinema"
    fi
    success "asciinema found"

    if ! command -v agg &> /dev/null; then
        warn "agg not found. GIF output disabled. Install with: brew install agg"
        AGG_AVAILABLE=false
    else
        success "agg found"
        AGG_AVAILABLE=true
    fi

    # svg-term can be run via npx
    if npx --yes svg-term-cli --version &> /dev/null 2>&1; then
        success "svg-term available via npx"
        SVG_AVAILABLE=true
    else
        warn "svg-term not available. SVG output disabled."
        SVG_AVAILABLE=false
    fi
}

record_demo() {
    local name="$1"
    local description="$2"
    local script="$3"

    local cast_file="$RECORDINGS_DIR/${name}.cast"
    local svg_file="$DEMOS_DIR/svg/${name}.svg"
    local gif_file="$DEMOS_DIR/gif/${name}.gif"

    info "Recording: $name ($description)"

    # Create temp script
    local temp_script=$(mktemp)
    echo "$script" > "$temp_script"
    chmod +x "$temp_script"

    # Record with asciinema
    asciinema rec \
        --cols "$COLS" \
        --rows "$ROWS" \
        --command "bash $temp_script" \
        --overwrite \
        "$cast_file"

    rm "$temp_script"

    # Convert to SVG
    if [ "$SVG_AVAILABLE" = true ]; then
        info "Converting to SVG..."
        npx --yes svg-term-cli \
            --in "$cast_file" \
            --out "$svg_file" \
            --window \
            --no-cursor \
            --term iterm2 \
            --profile "$THEME"
        success "Created: $svg_file"
    fi

    # Convert to GIF
    if [ "$AGG_AVAILABLE" = true ]; then
        info "Converting to GIF..."
        agg \
            --cols "$COLS" \
            --rows "$ROWS" \
            --theme monokai \
            --speed 1.5 \
            "$cast_file" \
            "$gif_file"
        success "Created: $gif_file"
    fi

    success "Recorded: $name"
}

# Demo scripts
demo_init_wizard() {
    cat << 'SCRIPT'
#!/bin/bash
clear
echo "$ preflight init"
sleep 1

# Simulate TUI interactions
cat << 'EOF'

  ┌─────────────────────────────────────────────────────────────────┐
  │                                                                 │
  │     ██████╗ ██████╗ ███████╗███████╗██╗     ██╗ ██████╗ ██╗  ██╗████████╗   │
  │     ██╔══██╗██╔══██╗██╔════╝██╔════╝██║     ██║██╔════╝ ██║  ██║╚══██╔══╝   │
  │     ██████╔╝██████╔╝█████╗  █████╗  ██║     ██║██║  ███╗███████║   ██║      │
  │     ██╔═══╝ ██╔══██╗██╔══╝  ██╔══╝  ██║     ██║██║   ██║██╔══██║   ██║      │
  │     ██║     ██║  ██║███████╗██║     ███████╗██║╚██████╔╝██║  ██║   ██║      │
  │     ╚═╝     ╚═╝  ╚═╝╚══════╝╚═╝     ╚══════╝╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝      │
  │                                                                 │
  │                 Deterministic Workstation Compiler              │
  │                                                                 │
  │  Let's set up your workstation configuration.                   │
  │                                                                 │
  │  ❯ Get started                                                  │
  │    I already have a config                                      │
  │                                                                 │
  └─────────────────────────────────────────────────────────────────┘

EOF
sleep 2

cat << 'EOF'
  Select your editor:

    ❯ nvim (Recommended)
      vscode
      cursor
      none

EOF
sleep 1.5

cat << 'EOF'
  Select languages you use:

    [x] Go
    [x] TypeScript
    [ ] Python
    [ ] Rust
    [ ] Java

EOF
sleep 1.5

cat << 'EOF'
  ✓ Created preflight.yaml
  ✓ Created layers/base.yaml
  ✓ Created layers/identity.work.yaml
  ✓ Created layers/role.dev.yaml

  Run 'preflight plan' to see what would change.
EOF
sleep 2
SCRIPT
}

demo_capture_review() {
    cat << 'SCRIPT'
#!/bin/bash
clear
echo "$ preflight capture"
sleep 1

cat << 'EOF'

  Scanning system...

  ✓ Found 12 Homebrew formulae
  ✓ Found 5 Homebrew casks
  ✓ Found git configuration
  ✓ Found SSH hosts
  ✓ Found shell aliases

  Starting review...

EOF
sleep 1.5

cat << 'EOF'
  ┌─────────────────────────────────────────────────────────────────┐
  │ Capture Review                                        1/12     │
  ├─────────────────────────────────────────────────────────────────┤
  │                                                                 │
  │  brew formula: ripgrep                                          │
  │  ─────────────────────────────────────────────────────────      │
  │  Fast line-oriented search tool                                 │
  │                                                                 │
  │  Layer: base                                                    │
  │                                                                 │
  │  [y] Accept  [n] Reject  [l] Change Layer  [e] Edit  [?] Help  │
  │                                                                 │
  └─────────────────────────────────────────────────────────────────┘

EOF
sleep 2

cat << 'EOF'
  ✓ Accepted: ripgrep → base

  ┌─────────────────────────────────────────────────────────────────┐
  │ Capture Review                                        2/12     │
  ├─────────────────────────────────────────────────────────────────┤
  │                                                                 │
  │  git user.email: work@company.com                               │
  │  ─────────────────────────────────────────────────────────      │
  │  Git user email configuration                                   │
  │                                                                 │
  │  Layer: identity.work                                           │
  │                                                                 │
  └─────────────────────────────────────────────────────────────────┘

EOF
sleep 2

cat << 'EOF'

  ✓ Review complete
  ✓ Created layers/base.yaml (8 items)
  ✓ Created layers/identity.work.yaml (2 items)
  ✓ Created layers/role.dev.yaml (2 items)

EOF
sleep 1
SCRIPT
}

demo_plan_apply() {
    cat << 'SCRIPT'
#!/bin/bash
clear
echo "$ preflight plan"
sleep 1

cat << 'EOF'

  Loading configuration...
  Compiling plan for target: default

  ┌─────────────────────────────────────────────────────────────────┐
  │ Plan                                                12 steps   │
  ├─────────────────────────────────────────────────────────────────┤
  │                                                                 │
  │  + brew tap: homebrew/cask-fonts                                │
  │  + brew install: ripgrep, fzf, bat, fd, jq                      │
  │  + brew cask: visual-studio-code, docker                        │
  │  ∼ git config: user.name = "Your Name"                          │
  │  ∼ git config: user.email = "you@example.com"                   │
  │  + symlink: ~/.zshrc                                            │
  │  + symlink: ~/.config/nvim                                      │
  │                                                                 │
  │  + = add   ∼ = modify   - = remove                              │
  │                                                                 │
  └─────────────────────────────────────────────────────────────────┘

  Press [a] to apply, [e] to explain, [q] to quit

EOF
sleep 3

echo ""
echo '$ preflight apply'
sleep 1

cat << 'EOF'

  Applying plan...

  ✓ [1/7] brew tap: homebrew/cask-fonts
  ✓ [2/7] brew install: ripgrep
  ✓ [3/7] brew install: fzf
  ✓ [4/7] brew install: bat
  ✓ [5/7] git config: user.name
  ✓ [6/7] git config: user.email
  ✓ [7/7] symlink: ~/.zshrc

  ✓ Apply complete (7 steps in 12.3s)

EOF
sleep 2
SCRIPT
}

demo_doctor_fix() {
    cat << 'SCRIPT'
#!/bin/bash
clear
echo "$ preflight doctor"
sleep 1

cat << 'EOF'

  Running health checks...

  ┌─────────────────────────────────────────────────────────────────┐
  │ Doctor Report                                                   │
  ├─────────────────────────────────────────────────────────────────┤
  │                                                                 │
  │  ✓ packages.brew: All packages installed                        │
  │  ⚠ git.config: Drift detected                                   │
  │    └─ user.email: expected "work@company.com"                   │
  │       └─ actual "personal@gmail.com"                            │
  │  ✓ ssh.config: All hosts configured                             │
  │  ⚠ files: Drift detected                                        │
  │    └─ ~/.zshrc: Modified externally                             │
  │                                                                 │
  │  2 issues found                                                 │
  │                                                                 │
  └─────────────────────────────────────────────────────────────────┘

EOF
sleep 2

echo ""
echo '$ preflight doctor --fix'
sleep 1

cat << 'EOF'

  Fixing issues...

  ⚠ git.config: Will update user.email
    personal@gmail.com → work@company.com

  ⚠ files: Will restore ~/.zshrc from config

  Continue? [y/N] y

  ✓ Fixed: git.config.user.email
  ✓ Fixed: ~/.zshrc

  ✓ All issues fixed (2 fixed)

EOF
sleep 2
SCRIPT
}

demo_rollback() {
    cat << 'SCRIPT'
#!/bin/bash
clear
echo "$ preflight rollback"
sleep 1

cat << 'EOF'

  Available snapshots:

  ID         DATE                 AGE        FILES   REASON
  ────────────────────────────────────────────────────────────────
  a1b2c3d4   2024-12-24 14:30:00  2 hours    3       pre-apply
  e5f6g7h8   2024-12-24 10:15:00  6 hours    5       doctor-fix
  i9j0k1l2   2024-12-23 16:45:00  1 day      2       pre-apply

  Use --to <id> to restore a snapshot

EOF
sleep 2

echo ""
echo '$ preflight rollback --to a1b2c3d4 --dry-run'
sleep 1

cat << 'EOF'

  Would restore from snapshot a1b2c3d4:

  ~/.zshrc
    └─ Current: 2.3 KB, modified 2024-12-24 15:00:00
    └─ Restore: 2.1 KB, from 2024-12-24 14:30:00

  ~/.gitconfig
    └─ Current: 512 B, modified 2024-12-24 16:00:00
    └─ Restore: 498 B, from 2024-12-24 14:30:00

  ~/.config/nvim/init.lua
    └─ Current: 4.2 KB, modified 2024-12-24 15:30:00
    └─ Restore: 4.0 KB, from 2024-12-24 14:30:00

  Run without --dry-run to apply

EOF
sleep 2

echo ""
echo '$ preflight rollback --to a1b2c3d4'
sleep 1

cat << 'EOF'

  Restoring from snapshot a1b2c3d4...

  ✓ Restored: ~/.zshrc
  ✓ Restored: ~/.gitconfig
  ✓ Restored: ~/.config/nvim/init.lua

  ✓ Restored 3 files from snapshot a1b2c3d4

EOF
sleep 2
SCRIPT
}

main() {
    echo ""
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║              Preflight Demo Recording Script                  ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""

    check_dependencies
    echo ""

    case "${1:-all}" in
        init)
            record_demo "init-wizard" "Interactive init flow" "$(demo_init_wizard)"
            ;;
        capture)
            record_demo "capture-review" "Capture with TUI review" "$(demo_capture_review)"
            ;;
        plan)
            record_demo "plan-apply" "Plan preview and apply" "$(demo_plan_apply)"
            ;;
        doctor)
            record_demo "doctor-fix" "Doctor detection and fix" "$(demo_doctor_fix)"
            ;;
        rollback)
            record_demo "rollback" "Snapshot rollback" "$(demo_rollback)"
            ;;
        all)
            info "Recording all demos..."
            record_demo "init-wizard" "Interactive init flow" "$(demo_init_wizard)"
            record_demo "capture-review" "Capture with TUI review" "$(demo_capture_review)"
            record_demo "plan-apply" "Plan preview and apply" "$(demo_plan_apply)"
            record_demo "doctor-fix" "Doctor detection and fix" "$(demo_doctor_fix)"
            record_demo "rollback" "Snapshot rollback" "$(demo_rollback)"
            ;;
        help|--help|-h)
            echo "Usage: $0 [demo]"
            echo ""
            echo "Demos:"
            echo "  init      Record init wizard demo"
            echo "  capture   Record capture review demo"
            echo "  plan      Record plan and apply demo"
            echo "  doctor    Record doctor fix demo"
            echo "  rollback  Record rollback demo"
            echo "  all       Record all demos (default)"
            echo ""
            echo "Output:"
            echo "  SVG: website/public/demos/svg/*.svg"
            echo "  GIF: website/public/demos/gif/*.gif"
            ;;
        *)
            error "Unknown demo: $1. Use --help for usage."
            ;;
    esac

    echo ""
    success "Done! Check website/public/demos/ for output."
}

main "$@"
