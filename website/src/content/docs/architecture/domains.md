---
title: Domains
description: Deep dive into Preflight's bounded contexts and domain model.
---

Preflight is organized around bounded contexts following Domain-Driven Design principles. Each domain has its own entities, value objects, and services.

## Domain Map

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Config    │  │  Compiler   │  │  Execution  │
│   Domain    │──│   Domain    │──│   Domain    │
└─────────────┘  └─────────────┘  └─────────────┘
       │                │                │
       ▼                ▼                ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│    Lock     │  │   Advisor   │  │   Catalog   │
│   Domain    │  │   Domain    │  │   Domain    │
└─────────────┘  └─────────────┘  └─────────────┘
                        │
       ┌────────────────┼────────────────┐
       ▼                ▼                ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│    Drift    │  │  Snapshot   │  │    Merge    │
│   Domain    │  │   Domain    │  │   Domain    │
└─────────────┘  └─────────────┘  └─────────────┘
```

## Config Domain

**Responsibility:** Configuration loading, validation, and merging.

### Aggregates

**Manifest** — Root configuration entity

```go
type Manifest struct {
    Name     string
    Version  string
    Target   string
    Layers   []string
    Defaults map[string]any
}
```

**Layer** — Configuration overlay

```go
type Layer struct {
    Name       string
    Path       string
    Content    map[string]any
    Provenance Provenance
}
```

**Target** — Ordered layer composition

```go
type Target struct {
    Name   string
    Layers []string
}
```

### Services

**Loader** — Parse YAML files into domain objects

```go
func (l *Loader) Load(path string) (*Manifest, error)
func (l *Loader) LoadLayers(dir string) ([]Layer, error)
```

**Merger** — Combine layers with provenance tracking

```go
func (m *Merger) Merge(manifest *Manifest, layers []Layer) (*MergedConfig, error)
```

### Merge Rules

| Type | Strategy |
|------|----------|
| Scalar | Last wins |
| Map | Deep merge |
| List | Set union |

---

## Compiler Domain

**Responsibility:** Transform configuration into executable step graph.

### Aggregates

**StepGraph** — Directed Acyclic Graph of steps

```go
type StepGraph struct {
    steps map[string]Step
    edges map[string][]string
}

func (g *StepGraph) Add(step Step)
func (g *StepGraph) TopologicalOrder() ([]Step, error)
```

**CompileContext** — Compilation context

```go
type CompileContext struct {
    Config     *MergedConfig
    Target     string
    Mode       Mode
    Providers  []Provider
}
```

### Provider Interface

```go
type Provider interface {
    Name() string
    Compile(ctx CompileContext) ([]Step, error)
}
```

---

## Execution Domain

**Responsibility:** Step execution and orchestration.

### Step Interface

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

### Status Values

| Status | Meaning |
|--------|---------|
| `Satisfied` | No action needed |
| `NeedsApply` | Action required |
| `Unknown` | Cannot determine |
| `Failed` | Check failed |

### Scheduler

Orchestrates step execution with dependency resolution:

```go
type Scheduler struct {
    graph   *StepGraph
    runner  *Runner
}

func (s *Scheduler) Execute(ctx context.Context) error
```

---

## Lock Domain

**Responsibility:** Version resolution and integrity verification.

### Entities

**Lockfile** — Resolved state

```go
type Lockfile struct {
    Version  string
    Machine  MachineInfo
    Layers   []LayerLock
    Packages []PackageLock
    Files    []FileLock
}
```

### Resolution Modes

| Mode | Behavior |
|------|----------|
| Intent | Latest compatible |
| Locked | Prefer lockfile |
| Frozen | Fail on mismatch |

### Integrity

- SHA256/SHA512 hashes for files
- Version pinning for packages
- Commit SHAs for git sources

---

## Advisor Domain

**Responsibility:** AI-powered guidance (optional).

### Interface

```go
type Advisor interface {
    Suggest(ctx SuggestContext) ([]Recommendation, error)
    Explain(ctx ExplainContext) (Explanation, error)
}
```

### Providers

- OpenAI
- Anthropic
- Ollama
- None (disabled)

### Constraints

- **Advisory only** — Never executes
- **BYOK** — Bring your own key
- **Explainable** — Maps to known presets

---

## Catalog Domain

**Responsibility:** Presets, capability packs, and metadata.

### Entities

**Preset** — Pre-configured setup

```go
type Preset struct {
    Name        string
    Description string
    Difficulty  Difficulty
    Config      map[string]any
}
```

**CapabilityPack** — Tool collection

```go
type CapabilityPack struct {
    Name         string
    Capabilities []Capability
    Explanation  Explanation
}
```

---

## Drift Domain

**Responsibility:** Detect external file changes.

### Entities

```go
type AppliedState struct {
    Files map[string]FileState
}

type FileState struct {
    Path        string
    AppliedHash string
    AppliedAt   time.Time
    SourceLayer string
}
```

### Detector

```go
type Detector struct {
    state *AppliedState
    fs    ports.FileSystem
}

func (d *Detector) Detect(path string) (*Drift, error)
```

---

## Snapshot Domain

**Responsibility:** Automatic backups before modifications.

### Entities

```go
type Snapshot struct {
    ID        string
    Path      string
    Hash      string
    CreatedAt time.Time
}

type SnapshotSet struct {
    ID        string
    Snapshots []Snapshot
    Reason    string
}
```

### Manager

```go
type Manager struct {
    store Store
}

func (m *Manager) BeforeApply(paths []string) (*SnapshotSet, error)
func (m *Manager) Restore(setID string) error
```

---

## Merge Domain

**Responsibility:** Three-way merge for conflict resolution.

### Functions

```go
func ThreeWayMerge(base, ours, theirs string, style ConflictStyle) Result
func DetectChangeType(base, ours, theirs string) ChangeType
func HasConflictMarkers(content string) bool
func ResolveAllConflicts(content string, resolution Resolution) string
```

### Change Types

| Type | Description |
|------|-------------|
| None | No changes |
| Ours | Config changed |
| Theirs | File changed |
| Both | Conflict |
| Same | Identical changes |

---

## Domain Interactions

### Config → Compiler

```go
config := merger.Merge(manifest, layers)
graph := compiler.Compile(config, providers)
```

### Compiler → Execution

```go
for _, step := range graph.TopologicalOrder() {
    status := step.Check(ctx)
    if status == NeedsApply {
        step.Apply(ctx)
    }
}
```

### Execution → Lock

```go
if opts.UpdateLock {
    lock.Record(executedSteps)
}
```

### Drift → Merge

```go
drifts := detector.Detect(paths)
for _, drift := range drifts {
    result := merge.ThreeWayMerge(drift.Base, drift.Ours, drift.Theirs)
}
```

## What's Next?

- [Design Principles](/preflight/architecture/principles/) — Core guarantees
- [Providers](/preflight/guides/providers/) — Provider implementations
- [TDD Workflow](/preflight/development/tdd/) — Development practices
