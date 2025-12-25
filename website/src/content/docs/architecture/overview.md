---
title: Architecture Overview
description: High-level architecture of Preflight's compiler model.
---

Preflight is built as a **deterministic workstation compiler** using Domain-Driven Design principles. This document provides an overview of the system architecture.

## Compiler Model

Preflight operates like a compiler, transforming declarative intent into executable actions:

```
Intent (YAML)
     ↓
   Load
     ↓
   Merge
     ↓
  Compile
     ↓
   Plan
     ↓
   Apply
     ↓
  Verify
```

### Pipeline Stages

| Stage | Input | Output |
|-------|-------|--------|
| Load | YAML files | Manifest + Layers |
| Merge | Manifest + Layers | Target Config |
| Compile | Target Config | Step Graph (DAG) |
| Plan | Step Graph | Diff + Explanations |
| Apply | Step Graph | System Changes |
| Verify | System State | Doctor Report |

## High-Level Architecture

```
┌─────────────────────────────────────────────────────┐
│                    CLI / TUI                        │
│              (cmd/preflight, internal/tui)          │
├─────────────────────────────────────────────────────┤
│                 Application Layer                   │
│                  (internal/app)                     │
├──────────────┬──────────────┬──────────────────────┤
│    Config    │   Compiler   │     Execution        │
│    Domain    │    Domain    │      Domain          │
├──────────────┼──────────────┼──────────────────────┤
│              │              │                      │
│   Manifest   │  StepGraph   │    Scheduler         │
│   Layer      │  Provider    │    Runner            │
│   Target     │  Context     │    Status            │
│   Merger     │              │                      │
│              │              │                      │
├──────────────┴──────────────┴──────────────────────┤
│                    Providers                        │
│    brew │ apt │ files │ git │ ssh │ shell │ ...    │
├─────────────────────────────────────────────────────┤
│                      Ports                          │
│           (FileSystem, CommandRunner, etc.)         │
├─────────────────────────────────────────────────────┤
│                    Adapters                         │
│              (OS, Git, Homebrew, etc.)              │
└─────────────────────────────────────────────────────┘
```

## Directory Structure

```
cmd/preflight/           # CLI entry point (Cobra)
internal/
  app/                   # Application services
  domain/
    config/              # Configuration domain
    compiler/            # Compilation domain
    execution/           # Execution domain
    lock/                # Lockfile domain
    advisor/             # AI advisor domain
    catalog/             # Presets and packs
    drift/               # Drift detection
    snapshot/            # File snapshots
    merge/               # Three-way merge
    capability/          # Permission system
    sandbox/             # WASM isolation
    trust/               # Signature verification
  provider/              # System adapters
    brew/
    apt/
    files/
    git/
    ssh/
    shell/
    runtime/
    nvim/
    vscode/
  tui/                   # Bubble Tea UI
  ports/                 # Interface definitions
  adapters/              # Port implementations
```

## Security Architecture

Preflight uses a defense-in-depth security model for plugins:

```
┌─────────────────────────────────────┐
│  Layer 4: Sandbox (WASM isolation)  │
├─────────────────────────────────────┤
│  Layer 3: Capabilities (permissions)│
├─────────────────────────────────────┤
│  Layer 2: Signatures (identity)     │
├─────────────────────────────────────┤
│  Layer 1: Integrity (hashes)        │
└─────────────────────────────────────┘
```

### Security Domains

| Domain | Responsibility |
|--------|----------------|
| **capability** | Fine-grained permission system |
| **sandbox** | WASM isolation with Wazero |
| **trust** | Cryptographic signature verification |

See [Plugin Security](/preflight/guides/security/) for details.

## Key Abstractions

### Provider

Compiles configuration into executable steps:

```go
type Provider interface {
    Name() string
    Compile(ctx CompileContext) ([]Step, error)
}
```

### Step

Idempotent unit of execution:

```go
type Step interface {
    ID() string
    DependsOn() []string
    Check(ctx RunContext) (Status, error)
    Plan(ctx RunContext) (Diff, error)
    Apply(ctx RunContext) error
    Explain(ctx ExplainContext) Explanation
}
```

### Step Graph

Directed Acyclic Graph of steps with topological ordering:

```go
type StepGraph struct {
    steps    []Step
    edges    map[string][]string
    compiled bool
}

func (g *StepGraph) TopologicalOrder() []Step
func (g *StepGraph) ParallelBatches() [][]Step
```

## Data Flow

### Configuration Loading

```
preflight.yaml ─┐
                ├─→ Loader ─→ Manifest
layers/*.yaml ──┘

Manifest + Target ─→ Merger ─→ MergedConfig
```

### Compilation

```
MergedConfig ─→ Providers ─→ Steps[]
Steps[] ─→ DependencyResolver ─→ StepGraph
```

### Execution

```
StepGraph ─→ Scheduler ─→ Parallel Batches
Batch ─→ Runner ─→ Check() ─→ Plan() ─→ Apply()
```

## Ports & Adapters

Preflight uses the Ports & Adapters (Hexagonal) architecture pattern.

### Ports (Interfaces)

```go
// internal/ports/filesystem.go
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, data []byte) error
    Exists(path string) bool
    Symlink(src, dest string) error
}

// internal/ports/command.go
type CommandRunner interface {
    Run(ctx context.Context, cmd string, args ...string) (Output, error)
}
```

### Adapters (Implementations)

```go
// internal/adapters/os/filesystem.go
type OSFileSystem struct{}

func (fs *OSFileSystem) Read(path string) ([]byte, error) {
    return os.ReadFile(path)
}
```

This allows:
- Easy testing with mock implementations
- Swappable implementations
- Clear system boundaries

## Error Handling

### Error Types

| Type | Description |
|------|-------------|
| `ConfigError` | Invalid configuration |
| `CompileError` | Compilation failure |
| `ApplyError` | Execution failure |
| `LockError` | Lock mismatch |

### Recovery Strategy

1. **Plan Phase** — Validate everything before execution
2. **Rollback** — Snapshot before destructive changes
3. **Idempotency** — Safe to retry on failure

## What's Next?

- [Domains](/preflight/architecture/domains/) — Deep dive into each domain
- [Design Principles](/preflight/architecture/principles/) — Core guarantees
- [TDD Workflow](/preflight/development/tdd/) — Development practices
