# Preflight Core Foundation (v1)

## Goal

Define the minimum architecture for a stable core that keeps the primary user
loop reliable:

`init -> plan -> apply -> doctor`

This document sets scope boundaries, package boundaries, public contracts, and
non-goals for the v1 core foundation milestone.

## Scope (In)

- Domain contracts for config, compiler, execution, and lock.
- Deterministic planning and execution behavior for repeated runs.
- Explicit lifecycle/state semantics for step execution and rollback.
- Context-cancellation behavior that is testable and enforced.
- A shared error taxonomy suitable for CLI/TUI user-facing suggestions.
- A baseline verification harness for core flow regression checks.

## Non-goals (Out)

- New provider features or broad provider catalog expansion.
- UX redesign of command discoverability/help surfaces.
- Advanced AI advisor capabilities beyond existing guardrails.
- Enterprise-only command surface and governance workflows.
- Website/positioning and marketing content updates.

## Bounded Contexts

| Context | Responsibility | Primary Output |
|---------|----------------|----------------|
| `config` | Load + validate manifest/layers and merge to effective config | Merged config model |
| `compiler` | Convert effective config into provider-specific executable steps | `StepGraph` / step set |
| `execution` | Plan + run steps with idempotency, rollback, and cancellation semantics | `StepResult` set + run status |
| `lock` | Track reproducibility metadata and package/file integrity snapshots | Lockfile artifact |

Cross-context interaction is unidirectional in the core path:

`config -> compiler -> execution -> lock`

No context may import a higher-level orchestration package.

## Package Boundaries

Core packages and constraints:

- `internal/domain/config`: manifest, layers, merge rules, validation.
- `internal/domain/compiler`: compile merged config to domain steps.
- `internal/domain/execution`: planner + executor + rollback policy.
- `internal/domain/lock`: lockfile aggregate and mutation APIs.
- `internal/ports`: external side-effect abstractions only (filesystem,
  commands, git, network, telemetry).
- `internal/provider/*`: adapters translating provider configs to steps and
  side-effects through ports.

Rules:

- Domain packages depend only on other domain packages and `internal/ports`
  abstractions.
- Providers never bypass execution contract by mutating host state directly;
  side-effects flow through execution steps + ports.
- CLI/TUI packages (`cmd/preflight`, `internal/tui`) do not contain domain
  business rules.

## Public Core Interfaces (Contract Level)

These are contract requirements, not a commitment to exact method signatures.

1. Compile contract
   - Input: validated merged config.
   - Output: deterministic step graph for the same input.
   - Failure mode: structured domain error with actionable category.

2. Plan contract
   - Input: step graph.
   - Output: executable order preserving dependency constraints.
   - Guarantee: no dependency violations.

3. Execute contract
   - Input: execution plan + context.
   - Output: per-step result with explicit terminal status and optional error.
   - Guarantee: aggregate run returns non-nil error if any step fails.
   - Guarantee: rollback targets only steps that were actually mutated/applied.

4. Idempotency contract
   - Re-running `apply` on unchanged desired state must converge to
     no-op/satisfied outcomes with no unintended side-effects.

5. Cancellation contract
   - Cancelled context prevents new step starts immediately and halts active
     work at bounded latency (target: ~1s for cancellation-aware steps).

## Error Taxonomy (Core)

All core errors should map to one of these categories before surfacing to
CLI/TUI:

- Validation: invalid user configuration or unsupported input.
- Dependency: ordering/graph or prerequisite failure.
- Execution: command/process failure while applying steps.
- Rollback: failed recovery operation after execution failure.
- Cancellation: context deadline/cancel triggered during run.
- Internal: invariant violation or bug (requires issue/report).

Each user-facing error should provide a concrete suggestion (next command or
remediation action).

## Architecture Decisions (ADR)

### ADR-001: Execution status distinguishes satisfied vs applied

Decision:
- Keep distinct status semantics so rollback can safely target only mutated
  steps.

Rationale:
- "Satisfied" can mean pre-existing state already matched desired state.
- "Applied" means this run changed host state and may require rollback.

Consequence:
- Step result model must include sufficient data for rollback gating and tests.

### ADR-002: Idempotency is a hard contract, not best effort

Decision:
- Treat apply-idempotency as a compatibility requirement across providers.

Rationale:
- User trust in bootstrap automation depends on safe re-runs.

Consequence:
- Contract tests run Apply->Apply across provider fixtures and fail CI on
  regressions.

### ADR-003: Cancellation is part of correctness

Decision:
- Execution must be context-aware; cancellation behavior is tested.

Rationale:
- Long-running installs/config operations are common; users need bounded stop
  behavior.

Consequence:
- Execution orchestration and step adapters must respect context consistently.

### ADR-004: Domain owns invariants; interfaces own side-effects

Decision:
- Domain packages enforce invariants; side-effects are isolated behind ports.

Rationale:
- Enables deterministic testing, easier provider portability, and lower
  coupling.

Consequence:
- Any direct side-effect in domain logic is treated as architectural drift.

## Acceptance Checklist (Scope/ADR task)

- [x] Core v1 scope is explicitly documented.
- [x] Non-goals are explicitly documented.
- [x] Bounded contexts and dependency direction are defined.
- [x] Package boundaries and layering rules are defined.
- [x] Core contract expectations (compile/plan/execute/idempotency/cancel) are
      documented.
- [x] Error taxonomy for CLI/TUI mapping is documented.
- [x] ADR decisions for status semantics, idempotency, cancellation, and domain
      boundaries are documented.

## Follow-on Work (Linked Roady tasks)

- `task-core-foundation-domain-contracts`
- `task-core-foundation-adapter-alignment`
- `task-core-foundation-verification-harness`
- `task-core-foundation-docs-and-guardrails`

## Companion Docs

- `docs/core-foundation-guardrails.md` (extension guide + CI guardrails)
