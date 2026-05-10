# Core Foundation Guardrails

This document defines developer guardrails for extending Preflight while
preserving the core foundation contracts.

Use it together with `docs/core-foundation.md`.

## Extension Guide: Adding or Updating a Provider

### 1) Keep provider scope narrow

- A provider should map one domain of intent (for example `git`, `runtime`,
  `terminal`) to steps.
- Keep provider compilation deterministic: identical config input should produce
  equivalent step IDs, dependencies, and explanations.

### 2) Implement stable step identity and dependencies

- Step IDs must be stable across runs for the same intent.
- `DependsOn()` should encode real prerequisites only.
- Avoid synthetic dependencies that serialize independent work.

### 3) Follow step lifecycle contract

- `Check`: return accurate status without side-effects.
- `Plan`: emit deterministic, user-readable diff output.
- `Apply`: perform one idempotent mutation unit and respect `RunContext`.
- `Explain`: provide concise rationale and docs links where available.

### 4) Respect idempotency and rollback semantics

- Re-running `Apply` on unchanged desired state must converge to satisfied/no-op.
- If your step supports rollback, implement `RollbackableStep` with safe,
  bounded rollback behavior.
- Never assume `StatusSatisfied` means this run applied changes; rollback gating
  is based on execution result semantics.

### 5) Isolate side-effects behind ports/adapters

- Avoid domain-level direct shell, filesystem, or network calls outside approved
  adapters.
- Keep command/file abstractions injectable for deterministic tests.

### 6) Return actionable errors

- Wrap low-level failures with provider/step context.
- Prefer typed/domain errors that can map to `config.UserError` with a concrete
  suggestion in CLI flows.

## Testing Guardrails

Minimum expectations for provider changes:

- Unit tests for compile/check/plan/apply behavior in provider package.
- Idempotency test path (apply twice semantics) where relevant.
- If rollback is supported, tests for rollback success and rollback failure path.
- If long-running apply logic exists, tests should confirm context cancellation
  is observed in bounded time.

Recommended commands:

```bash
go test ./internal/provider/<provider>
go test ./internal/domain/execution
go test ./test/integration -run "TestCoreFlowHarness_.*"
```

## CI Quality Gate Checklist

Every PR touching core execution, planning, compiler/provider contracts, or
command orchestration should pass this checklist:

- [ ] `make test` passes.
- [ ] `make lint` passes.
- [ ] `make coverage-check` passes with domain thresholds.
- [ ] Core flow regression tests pass (`TestCoreFlowHarness_*`).
- [ ] No new plain `fmt.Errorf` user-facing errors in `cmd/preflight` where a
      `config.UserError` with suggestion is expected.
- [ ] Any execution semantic change includes tests for at least one of:
      failure aggregation, rollback gating, idempotency, or cancellation.

## Review Prompts for Core Changes

Use these prompts in code review:

- Does this change preserve deterministic compile/plan output for identical
  inputs?
- Could this change accidentally roll back pre-existing satisfied state?
- Are cancellation and partial-failure behaviors explicit and tested?
- Can a user understand recovery steps directly from the surfaced error?
- Is the change increasing coupling between domain and adapter layers?
