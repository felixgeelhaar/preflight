# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Preflight is a deterministic workstation compiler - a Go CLI/TUI tool that compiles declarative configuration into reproducible machine setups. It follows a compiler model: Intent → Merge → Plan → Apply → Verify.

## Development Commands

```bash
# Build
go build -o bin/preflight ./cmd/preflight

# Run
go run ./cmd/preflight

# Test
go test ./...
go test -v ./internal/config/...     # Test specific domain
go test -race ./...                   # With race detection

# Coverage (requires >80% per domain)
go test -coverprofile=coverage.out ./...
coverctl check --threshold 80        # Verify all domains meet threshold

# Lint
golangci-lint run

# Generate (if using go generate)
go generate ./...
```

## Test-Driven Development

Follow the Red-Green-Refactor cycle for all changes:

1. **Red** - Write a failing test that defines expected behavior
2. **Green** - Write minimal code to make the test pass
3. **Refactor** - Clean up while keeping tests green

Each commit should represent one complete TDD cycle. Use `coverctl` to ensure all domains maintain >80% test coverage:

```bash
# Check coverage per domain
coverctl check --threshold 80

# View coverage report
coverctl report
```

## Architecture

### Domain-Driven Design

The codebase is organized around bounded contexts. Each domain is self-contained with its own entities, value objects, and domain services.

**Bounded Contexts:**

| Domain | Responsibility | Key Aggregates |
|--------|----------------|----------------|
| config | Configuration loading, merging, validation | Manifest, Layer, Target |
| compiler | Transform config into executable plan | StepGraph, CompileContext |
| execution | Step orchestration and runtime | Step, RunContext, Status |
| lock | Version resolution and integrity | Lockfile, Resolution |
| provider | System integration adapters | Provider implementations |
| advisor | AI guidance (optional) | Recommendation, Explanation |
| catalog | Presets and capability metadata | Preset, CapabilityPack |

**Domain Rules:**
- Domains communicate through well-defined interfaces, not internal types
- Each domain owns its repository/persistence logic
- Value objects are immutable; entities have identity
- Domain events for cross-domain communication where needed

### Core Flow

1. **Load** - Parse `preflight.yaml` manifest and `layers/*.yaml` overlays
2. **Merge** - Deep merge layers into target config with provenance tracking
3. **Compile** - Providers emit Steps into a DAG
4. **Plan** - Diff current state vs desired, generate explanations
5. **Apply** - Execute steps idempotently
6. **Verify** - Doctor checks for drift

### Directory Structure

```
cmd/preflight/           # CLI entry point (Cobra commands)
internal/
  domain/
    config/              # Config domain: Manifest, Layer, Target, merge logic
    compiler/            # Compiler domain: StepGraph, compilation engine
    execution/           # Execution domain: Step, DAG scheduler, runtime
    lock/                # Lock domain: Lockfile, resolution, integrity
    advisor/             # Advisor domain: AI interfaces (OpenAI, Anthropic, Ollama)
    catalog/             # Catalog domain: Presets, capability packs
  provider/              # Provider adapters (anti-corruption layer)
    brew/
    apt/
    files/
    git/
    ssh/
    runtime/
    editor/
      nvim/
      vscode/
  tui/                   # Bubble Tea views (application layer)
  ports/                 # Interfaces for external dependencies
  adapters/              # Implementations of ports
```

### Key Interfaces

**Provider** - Compiles config section into executable steps:
```go
type Provider interface {
    Name() string
    Compile(ctx CompileContext) ([]Step, error)
}
```

**Step** - Idempotent unit of execution:
```go
type Step interface {
    ID() string
    DependsOn() []string
    Check(ctx RunContext) (Status, error)   // satisfied | needs-apply | unknown
    Plan(ctx RunContext) (Diff, error)
    Apply(ctx RunContext) error
    Explain(ctx ExplainContext) Explanation
}
```

**Advisor** - AI is advisory only, never executes:
```go
type Advisor interface {
    Suggest(ctx SuggestContext) ([]Recommendation, error)
    Explain(ctx ExplainContext) (Explanation, error)
}
```

### Config Model

- `preflight.yaml` - Root manifest (targets, defaults)
- `layers/*.yaml` - Composable overlays (base, identity.work, role.go, device.laptop)
- `preflight.lock` - Resolved versions, integrity hashes, machine info
- `dotfiles/` - Generated/templated/user-owned files

**Merge semantics**: Scalars last-wins, maps deep-merge, lists set-union with add/remove directives. Track layer provenance for TUI explainability.

### Reproducibility Modes

- **intent** - Install latest compatible versions
- **locked** - Prefer lockfile, update intentionally with `--update-lock`
- **frozen** - Fail if resolution differs from lock

### Providers (v1)

| Provider | Responsibility |
|----------|----------------|
| brew | Homebrew taps, formulae, casks (macOS) |
| apt | Package installation (Linux) |
| files | Dotfile rendering, linking, drift detection |
| git | .gitconfig generation, identity separation |
| ssh | ~/.ssh/config rendering, never exports keys |
| runtime | rtx/asdf tool version management |
| nvim | Neovim install, preset bootstrap, lazy-lock |
| vscode | Extension install, settings management |

### TUI (Bubble Tea)

Key screens: Init wizard, Capture review (git-add -p style), Plan review with explain panel, Apply progress, Doctor report.

## Design Principles

**Product Guarantees:**
1. **No execution without a plan** - Always show what will change first
2. **Idempotent operations** - Re-running apply is always safe
3. **Explainability** - Every action has "why", tradeoffs, and docs links
4. **Secrets never leave the machine** - Capture redacts, config uses references only
5. **AI advises, never executes** - AI outputs map to known presets or require user confirmation
6. **User ownership** - Config is portable, inspectable, git-native

**Engineering Standards:**
- **TDD** - All code written test-first using Red-Green-Refactor
- **DDD** - Clear bounded contexts with ubiquitous language per domain
- **Coverage** - All domains must maintain >80% test coverage (enforced by coverctl)
