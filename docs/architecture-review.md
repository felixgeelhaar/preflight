# Preflight Architecture Review

**Date:** 2025-12-23
**Reviewer:** Technical Architect Agent
**Scope:** Domain-Driven Design, Hexagonal Architecture, SOLID Principles, Package Cohesion

## Executive Summary

The Preflight codebase demonstrates a **strong foundation** in domain-driven design and hexagonal architecture. The implementation shows clear separation of concerns, well-defined bounded contexts, and proper dependency management. However, there are opportunities for improvement in aggregate boundary enforcement, value object immutability, and cross-cutting concerns.

**Overall Grade: B+**

**Key Strengths:**
- Clear bounded context separation
- Well-defined ports and adapters pattern
- Strong use of value objects
- Clean dependency flow (domain → ports → adapters)
- Proper aggregate roots in lock domain

**Key Improvement Areas:**
- Aggregate boundary enforcement in config domain
- Repository pattern consistency
- Cross-domain event handling
- Value object encapsulation
- Domain service extraction

---

## 1. Domain-Driven Design Assessment

### 1.1 Bounded Contexts

The codebase has **well-defined bounded contexts** with clear responsibilities:

| Domain | Status | Assessment |
|--------|--------|------------|
| **config** | ✅ Good | Clear responsibility for manifest/layer management, merge semantics |
| **compiler** | ✅ Good | Well-defined transformation from config → steps |
| **execution** | ✅ Good | Focused on runtime orchestration |
| **lock** | ✅ Excellent | Strong aggregate design with Lockfile as root |
| **advisor** | ⚠️ Minimal | Placeholder implementation, needs development |
| **catalog** | ⚠️ Minimal | Not yet implemented |

**Strengths:**
- Each bounded context has its own package with clear boundaries
- No circular dependencies between domains
- Ubiquitous language is evident (Manifest, Layer, Target, Step, etc.)

**Concerns:**
- No explicit anti-corruption layers between domains
- Missing domain events for cross-context communication
- Some domains lack clear aggregate roots

### 1.2 Aggregates and Entities

#### ✅ Strong Aggregate Design (Lock Domain)

```go
// Lockfile is a well-designed aggregate root
type Lockfile struct {
    version     int
    mode        config.ReproducibilityMode
    machineInfo MachineInfo
    packages    map[string]PackageLock  // Encapsulated
}
```

**Strengths:**
- Private fields with controlled access
- Invariant enforcement (AddPackage, UpdatePackage)
- No direct modification of internal state
- Clear consistency boundary

#### ⚠️ Weak Aggregate Design (Config Domain)

```go
// Layer is not a true aggregate - fields are public
type Layer struct {
    Name       LayerName
    Provenance string
    Packages   PackageSet  // Public, mutable
    Files      []FileDeclaration
    Git        GitConfig
    // ...
}
```

**Issues:**
- Public fields allow direct mutation outside domain
- No invariant enforcement
- Unclear if Layer or Manifest is the aggregate root
- Missing domain methods for business operations

**Recommendation:**
```go
// Refactor Layer to be a proper aggregate
type Layer struct {
    name       LayerName
    provenance string
    packages   PackageSet
    files      []FileDeclaration
    git        GitConfig
}

// Add domain methods
func (l *Layer) AddPackage(pkg string) error {
    // Validate invariants
    // Emit domain event
    return nil
}

func (l *Layer) Packages() PackageSet {
    // Return defensive copy
    return l.packages.Copy()
}
```

### 1.3 Value Objects

#### ✅ Excellent Value Object Implementation

```go
// LayerName is an immutable value object with validation
type LayerName struct {
    value string
}

func NewLayerName(s string) (LayerName, error) {
    // Validation logic
    return LayerName{value: trimmed}, nil
}
```

**Strengths:**
- Immutable by design (private field)
- Factory method with validation
- Self-validating
- Clear semantic meaning

**Examples:**
- `LayerName` ✅
- `TargetName` ✅
- `StepID` ✅
- `ReproducibilityMode` ✅
- `Integrity` (in lock domain) ✅

#### ⚠️ Missing Value Objects

Several primitive obsessions should be value objects:

```go
// Current - primitive obsession
type GitUserConfig struct {
    Name       string  // Should be PersonName value object
    Email      string  // Should be EmailAddress value object
    SigningKey string  // Should be GPGKeyID value object
}

// Recommended
type GitUserConfig struct {
    name       PersonName
    email      EmailAddress
    signingKey GPGKeyID
}
```

### 1.4 Domain Services

#### ⚠️ Missing Domain Services

The `Merger` in config domain should be a domain service with richer behavior:

```go
// Current - anemic service
type Merger struct{}

func (m *Merger) Merge(layers []Layer) (*MergedConfig, error) {
    // 365 lines of procedural code
}
```

**Issues:**
- No explicit merge strategy pattern
- Hard-coded merge semantics
- Difficult to extend or customize
- No support for conflict resolution strategies

**Recommendation:**
```go
// Domain service with strategy pattern
type MergeStrategy interface {
    MergeScalars(old, new string) string
    MergeLists(old, new []string) []string
    MergeMaps(old, new map[string]string) map[string]string
}

type Merger struct {
    strategy MergeStrategy
}

func NewMerger(strategy MergeStrategy) *Merger {
    return &Merger{strategy: strategy}
}
```

---

## 2. Hexagonal Architecture (Ports and Adapters)

### 2.1 Port Definitions

#### ✅ Well-Defined Ports

```go
// ports/command.go
type CommandRunner interface {
    Run(ctx context.Context, command string, args ...string) (CommandResult, error)
}

// ports/filesystem.go
type FileSystem interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, perm os.FileMode) error
    // ... more methods
}
```

**Strengths:**
- Clear interface definitions in `internal/ports/`
- Domain depends on abstractions, not implementations
- Testability through interface injection

#### ⚠️ Missing Ports

Several domain interactions lack port definitions:

1. **No Repository Port in Config Domain**
   ```go
   // Currently - direct file system access
   type Loader struct {
       // No dependency injection
   }

   // Should be
   type ConfigRepository interface {
       LoadManifest(path string) (*Manifest, error)
       LoadLayer(path string) (*Layer, error)
   }
   ```

2. **Missing Advisor Port in Compiler Domain**
   - Compiler domain should depend on an abstraction, not concrete advisor implementations

3. **Missing Event Publisher Port**
   - No mechanism for domain events

### 2.2 Adapter Implementations

#### ✅ Clean Adapter Pattern

```go
// adapters/command/runner.go
type RealRunner struct{}

func (r *RealRunner) Run(ctx context.Context, command string, args ...string) (ports.CommandResult, error) {
    // Implementation
}

var _ ports.CommandRunner = (*RealRunner)(nil)  // Compile-time verification
```

**Strengths:**
- Adapters in dedicated `internal/adapters/` package
- Compile-time verification of interface compliance
- Clear separation from domain logic

### 2.3 Provider Pattern Analysis

#### ⚠️ Provider Pattern Confusion

Providers sit in an ambiguous architectural layer:

```go
// internal/provider/brew/provider.go
type Provider struct {
    runner ports.CommandRunner  // Depends on port ✅
}

func (p *Provider) Compile(ctx compiler.CompileContext) ([]compiler.Step, error) {
    // Returns compiler domain types ✅
}
```

**Issue:** Providers are **anti-corruption layers** between domain and infrastructure, but they're not in the `adapters` package.

**Current Structure:**
```
internal/
  domain/
    compiler/      # Domain
  ports/           # Port definitions
  adapters/        # Infrastructure adapters
  provider/        # ??? Anti-corruption layer but not in adapters/
```

**Recommendation:**
```
internal/
  domain/
    compiler/
  ports/
  adapters/
    command/
    filesystem/
    providers/     # Move here - these are adapters!
      brew/
      apt/
      files/
```

---

## 3. SOLID Principles Analysis

### 3.1 Single Responsibility Principle

#### ✅ Generally Good

Most types have single, clear responsibilities:

- `Lockfile` - Manages locked package versions
- `StepGraph` - Manages step dependencies and topological sorting
- `Executor` - Orchestrates step execution

#### ⚠️ SRP Violations

**Merger Class (365 lines):**
```go
func (m *Merger) Merge(layers []Layer) (*MergedConfig, error) {
    // Merges brew formulae
    // Merges brew casks
    // Merges apt packages
    // Merges files
    // Merges git config
    // Merges SSH config
    // Merges runtime config
    // Merges shell config
    // Merges nvim config
    // Merges VSCode config
    // Tracks provenance
}
```

**Violation:** 10+ responsibilities in one method

**Recommendation:**
```go
type Merger struct {
    strategies map[string]SectionMerger
}

type SectionMerger interface {
    Merge(old, new interface{}) interface{}
}

type BrewMerger struct{}
type GitMerger struct{}
type SSHMerger struct{}
// etc.
```

### 3.2 Open/Closed Principle

#### ✅ Good Extension Points

The Step interface is open for extension:

```go
type Step interface {
    ID() StepID
    DependsOn() []StepID
    Check(ctx RunContext) (StepStatus, error)
    Plan(ctx RunContext) (Diff, error)
    Apply(ctx RunContext) error
    Explain(ctx ExplainContext) Explanation
}
```

New step types can be added without modifying existing code.

#### ⚠️ OCP Violations

**Compiler Registration:**
```go
// internal/app/preflight.go
func New(out io.Writer) *Preflight {
    comp := compiler.NewCompiler()
    comp.RegisterProvider(apt.NewProvider(cmdRunner))
    comp.RegisterProvider(brew.NewProvider(cmdRunner))
    // ... 7 more hardcoded registrations
}
```

**Issue:** Adding a new provider requires modifying `app/preflight.go`

**Recommendation:**
```go
// Use plugin discovery pattern
type ProviderRegistry interface {
    Discover() []compiler.Provider
}

// Or use configuration-based registration
func NewFromConfig(config AppConfig) *Preflight {
    // Register providers from config
}
```

### 3.3 Liskov Substitution Principle

#### ✅ Good Interface Design

All implementations are properly substitutable:

```go
var _ compiler.Provider = (*brew.Provider)(nil)
var _ compiler.Provider = (*apt.Provider)(nil)
var _ ports.CommandRunner = (*RealRunner)(nil)
```

No LSP violations detected.

### 3.4 Interface Segregation Principle

#### ✅ Focused Interfaces

Interfaces are small and focused:

```go
type CommandRunner interface {
    Run(ctx context.Context, command string, args ...string) (CommandResult, error)
}
```

#### ⚠️ ISP Concerns

The `Step` interface might be too large:

```go
type Step interface {
    ID() StepID                               // Identity
    DependsOn() []StepID                      // Graph structure
    Check(ctx RunContext) (StepStatus, error) // State inspection
    Plan(ctx RunContext) (Diff, error)        // Planning
    Apply(ctx RunContext) error               // Execution
    Explain(ctx ExplainContext) Explanation   // Documentation
}
```

**Concern:** 6 methods mixing concerns (identity, graph, lifecycle, documentation)

**Consideration:**
```go
// Split into smaller interfaces
type StepIdentifier interface {
    ID() StepID
}

type StepDependencies interface {
    DependsOn() []StepID
}

type StepExecutor interface {
    Check(ctx RunContext) (StepStatus, error)
    Plan(ctx RunContext) (Diff, error)
    Apply(ctx RunContext) error
}

type StepExplainer interface {
    Explain(ctx ExplainContext) Explanation
}

// Compose as needed
type Step interface {
    StepIdentifier
    StepDependencies
    StepExecutor
    StepExplainer
}
```

**Trade-off:** This adds complexity. Current design is acceptable given the cohesion of these operations.

### 3.5 Dependency Inversion Principle

#### ✅ Excellent DIP Compliance

The dependency flow is correct:

```
Presentation Layer (cmd/preflight)
    ↓
Application Layer (internal/app)
    ↓
Domain Layer (internal/domain/*)
    ↓ (depends on abstractions)
Ports (internal/ports)
    ↑ (implements)
Adapters (internal/adapters)
```

**Strengths:**
- Domain depends on ports (interfaces)
- Adapters implement ports
- No domain → infrastructure dependencies
- Proper dependency injection throughout

---

## 4. Dependency Flow and Coupling

### 4.1 Dependency Graph

```
cmd/preflight
  └─> internal/app
       └─> internal/domain/compiler
       └─> internal/domain/config
       └─> internal/domain/execution
       └─> internal/domain/lock
       └─> internal/ports
       └─> internal/adapters
       └─> internal/provider (⚠️ should be in adapters)
```

#### ✅ No Circular Dependencies

Clean unidirectional flow from outer layers to inner layers.

#### ⚠️ Provider Coupling

```go
// internal/app/preflight.go imports ALL providers
import (
    "github.com/felixgeelhaar/preflight/internal/provider/apt"
    "github.com/felixgeelhaar/preflight/internal/provider/brew"
    "github.com/felixgeelhaar/preflight/internal/provider/files"
    "github.com/felixgeelhaar/preflight/internal/provider/git"
    "github.com/felixgeelhaar/preflight/internal/provider/nvim"
    "github.com/felixgeelhaar/preflight/internal/provider/runtime"
    "github.com/felixgeelhaar/preflight/internal/provider/shell"
    "github.com/felixgeelhaar/preflight/internal/provider/ssh"
    "github.com/felixgeelhaar/preflight/internal/provider/vscode"
)
```

**Issue:** Application layer is coupled to all provider implementations

**Recommendation:** Use plugin discovery or registry pattern

### 4.2 Cross-Domain Communication

#### ⚠️ Missing Domain Events

There's no mechanism for domains to communicate asynchronously:

```go
// Example: When config is merged, advisor domain should be notified
// Currently - no event mechanism
merged := merger.Merge(layers)

// Should emit
events.Publish(ConfigMergedEvent{
    Target: target,
    Config: merged,
})
```

**Recommendation:**
```go
// Add domain events
type DomainEvent interface {
    EventName() string
    OccurredAt() time.Time
}

type EventPublisher interface {
    Publish(event DomainEvent)
}

// Use in aggregates
func (l *Lockfile) AddPackage(pkg PackageLock) error {
    // ... validation
    l.packages[key] = pkg
    l.events = append(l.events, PackageAddedEvent{...})
    return nil
}
```

---

## 5. Package Cohesion

### 5.1 Domain Package Organization

#### ✅ Good Cohesion

Each domain package is cohesive:

**internal/domain/config/**
- manifest.go (Manifest aggregate)
- layer.go (Layer entity)
- target.go (Target entity)
- merger.go (Merge domain service)
- layer_name.go (LayerName value object)
- target_name.go (TargetName value object)
- loader.go (Loading service)
- validator.go (Validation service)

**internal/domain/compiler/**
- compiler.go (Compiler service)
- step.go (Step interface)
- step_graph.go (StepGraph aggregate)
- step_id.go (StepID value object)
- step_status.go (StepStatus enum)
- provider.go (Provider interface)
- context.go (Context value objects)
- diff.go (Diff value object)
- explanation.go (Explanation value object)

**internal/domain/lock/**
- lockfile.go (Lockfile aggregate root)
- package_lock.go (PackageLock entity)
- machine_info.go (MachineInfo value object)
- integrity.go (Integrity value object)
- repository.go (Repository port)
- resolver.go (Resolver service)

All show **high cohesion** - related types grouped together.

### 5.2 Provider Package Organization

#### ⚠️ Inconsistent Structure

Some providers are well-organized:

**internal/provider/brew/**
- provider.go (Provider implementation)
- steps.go (Step implementations)
- config.go (Config parsing)
- *_test.go (Tests)

Others need improvement:

**internal/provider/editor/** has subdirectories **nvim/** and **vscode/** but also:
**internal/provider/nvim/** and **internal/provider/vscode/** at the root level

This creates confusion and duplication.

**Recommendation:**
```
internal/adapters/providers/
  ├── brew/
  ├── apt/
  ├── files/
  ├── git/
  ├── ssh/
  ├── runtime/
  ├── shell/
  └── editors/
      ├── nvim/
      └── vscode/
```

---

## 6. Specific Recommendations

### 6.1 High Priority

1. **Move Providers to Adapters Package**
   ```
   mv internal/provider internal/adapters/providers
   ```

2. **Implement Repository Pattern in Config Domain**
   ```go
   type ConfigRepository interface {
       LoadManifest(path string) (*Manifest, error)
       LoadLayer(path string) (*Layer, error)
       SaveLayer(path string, layer *Layer) error
   }
   ```

3. **Add Domain Events Infrastructure**
   ```go
   type EventBus interface {
       Publish(event DomainEvent)
       Subscribe(eventType string, handler EventHandler)
   }
   ```

4. **Refactor Layer to Proper Aggregate**
   - Make fields private
   - Add domain methods
   - Enforce invariants

### 6.2 Medium Priority

5. **Extract Section Mergers from Monolithic Merge Method**
   - Create `SectionMerger` interface
   - Implement per-section mergers
   - Use strategy pattern

6. **Add Value Objects for Primitive Obsessions**
   - EmailAddress
   - PersonName
   - GPGKeyID
   - FilePath
   - etc.

7. **Implement Provider Registry Pattern**
   ```go
   type ProviderRegistry interface {
       Register(provider compiler.Provider)
       GetAll() []compiler.Provider
       GetByName(name string) (compiler.Provider, bool)
   }
   ```

### 6.3 Low Priority

8. **Consider Step Interface Segregation**
   - Evaluate if 6 methods is too many
   - Consider splitting if new use cases emerge

9. **Add Anti-Corruption Layers**
   - Between config and compiler domains
   - Between execution and compiler domains

10. **Document Domain Boundaries in ADRs**
    - Create Architecture Decision Records
    - Document bounded context boundaries
    - Define integration patterns

---

## 7. Test Coverage Analysis

### ✅ Strong Test Coverage

Based on file counts:
- 209 total .go files
- Tests appear alongside implementation files (*_test.go)
- All domains have test coverage

**Evidence:**
- `internal/domain/config/manifest_test.go`
- `internal/domain/compiler/compiler_test.go`
- `internal/domain/lock/lockfile_test.go`
- `internal/provider/brew/provider_test.go`

**Recommendation:** Verify with coverage tools that >80% threshold is met per domain.

---

## 8. Anti-Patterns Detected

### 8.1 Anemic Domain Model (Config Domain)

```go
// Layer has public fields, no behavior
type Layer struct {
    Name       LayerName
    Packages   PackageSet  // Direct access
}

// Merger does all the work
type Merger struct{}
func (m *Merger) Merge(layers []Layer) (*MergedConfig, error)
```

**Fix:** Move merge logic into Layer aggregate methods

### 8.2 God Object (Merger.Merge)

365-line method doing too much.

**Fix:** Extract section mergers

### 8.3 Primitive Obsession

Excessive use of strings where value objects should exist.

**Fix:** Create domain value objects

### 8.4 Feature Envy

Providers reach into domain objects directly instead of using domain methods.

**Fix:** Add domain methods, respect encapsulation

---

## 9. Architectural Patterns Applied

### ✅ Successfully Applied

1. **Hexagonal Architecture** - Clear ports and adapters separation
2. **Repository Pattern** - Lock domain has proper repository
3. **Value Object Pattern** - LayerName, TargetName, StepID, etc.
4. **Strategy Pattern** - Step implementations
5. **Factory Pattern** - NewLockfile, NewStepGraph, etc.
6. **Dependency Injection** - Throughout the codebase

### ⚠️ Missing or Incomplete

7. **Domain Events** - No event infrastructure
8. **Aggregate Pattern** - Weak in config domain
9. **Specification Pattern** - No validation specifications
10. **Unit of Work Pattern** - No transaction boundaries
11. **CQRS** - No read/write separation (acceptable for this domain)

---

## 10. Compliance Summary

| Principle/Pattern | Grade | Notes |
|-------------------|-------|-------|
| **DDD - Bounded Contexts** | A | Clear separation |
| **DDD - Aggregates** | B | Good in lock, weak in config |
| **DDD - Value Objects** | A- | Strong usage, some missing |
| **DDD - Domain Services** | C+ | Merger needs refactoring |
| **Hexagonal - Ports** | A- | Good, some missing |
| **Hexagonal - Adapters** | B+ | Clean, provider location issue |
| **SOLID - SRP** | B | Merger violates |
| **SOLID - OCP** | B+ | Good extensibility |
| **SOLID - LSP** | A | No violations |
| **SOLID - ISP** | A- | Interfaces well-sized |
| **SOLID - DIP** | A | Excellent dependency flow |
| **Package Cohesion** | A- | High cohesion, minor issues |
| **Dependency Flow** | A | Clean unidirectional |

**Overall: B+** (85/100)

---

## 11. Critical Path Forward

### Phase 1: Foundation (Week 1)
1. Move providers to adapters package
2. Add repository port for config domain
3. Refactor Layer to proper aggregate

### Phase 2: Enrichment (Week 2)
4. Implement domain events infrastructure
5. Extract section mergers from Merger
6. Add missing value objects

### Phase 3: Polish (Week 3)
7. Add provider registry pattern
8. Document bounded contexts in ADRs
9. Implement anti-corruption layers

---

## 12. Conclusion

The Preflight architecture is **solid and well-thought-out**, demonstrating strong adherence to DDD and hexagonal architecture principles. The main areas for improvement are:

1. **Aggregate boundary enforcement** in the config domain
2. **Service extraction** to reduce the Merger god object
3. **Domain events** for cross-context communication
4. **Provider organization** to clarify architectural layers

These improvements will elevate the codebase from **good** to **excellent**, making it more maintainable, testable, and aligned with enterprise-grade standards.

The team should be commended for the strong foundation - the path forward is refinement, not rearchitecture.

---

## Appendix A: Directory Structure Recommendation

```
preflight/
├── cmd/
│   └── preflight/              # CLI entry point
├── internal/
│   ├── domain/                 # Core business logic
│   │   ├── config/            # Config bounded context
│   │   ├── compiler/          # Compiler bounded context
│   │   ├── execution/         # Execution bounded context
│   │   ├── lock/              # Lock bounded context
│   │   ├── advisor/           # Advisor bounded context
│   │   ├── catalog/           # Catalog bounded context
│   │   └── shared/            # Shared kernel (events, etc.)
│   ├── ports/                 # Interface definitions
│   │   ├── repositories/      # Repository interfaces
│   │   ├── services/          # External service interfaces
│   │   └── infrastructure/    # Infrastructure interfaces
│   ├── adapters/              # Infrastructure implementations
│   │   ├── command/           # Command execution
│   │   ├── filesystem/        # File system operations
│   │   ├── lockfile/          # Lockfile persistence
│   │   └── providers/         # Provider implementations (MOVED)
│   │       ├── brew/
│   │       ├── apt/
│   │       ├── files/
│   │       ├── git/
│   │       ├── ssh/
│   │       ├── runtime/
│   │       ├── shell/
│   │       └── editors/
│   │           ├── nvim/
│   │           └── vscode/
│   ├── app/                   # Application services
│   ├── tui/                   # Terminal UI
│   └── testutil/              # Testing utilities
└── docs/                      # Documentation
    └── adr/                   # Architecture Decision Records
```

---

**End of Architecture Review**
