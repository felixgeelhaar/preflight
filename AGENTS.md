# Repository Guidelines

## Project Structure & Module Organization
- `cmd/preflight/` contains the CLI entry point and command wiring.
- `internal/` holds application logic (domain, providers, adapters, TUI, MCP).
- `docs/` has design notes and specs; `website/` hosts the marketing/docs site.
- `test/` contains integration tests; unit tests live alongside packages as `*_test.go`.

## Build, Test, and Development Commands
- `make build` builds the CLI into `./bin/`.
- `make test` runs the full test suite.
- `make lint` runs `golangci-lint` with `.golangci.yml`.
- `make coverage-check` enforces per-domain coverage via `coverctl`.
- `go test ./...` is acceptable for quick full-suite runs.

## Coding Style & Naming Conventions
- Go standard formatting: run `gofmt` (or let your editor do it).
- Follow idiomatic Go naming (CamelCase for exported, lowerCamel for unexported).
- Keep packages cohesive; prefer adding new providers under `internal/provider/<name>`.
- Use Conventional Commits for messages (e.g., `feat(provider): add foo`).

## Testing Guidelines
- TDD is expected: Red → Green → Refactor per change.
- Coverage target is 80%+ per domain; validate with `make coverage-check`.
- Name tests with `TestXxx` and keep table tests in the same package.
- For focused runs: `go test ./internal/<package> -run TestName`.

## Commit & Pull Request Guidelines
- Commit format: `<type>(<scope>): <description>` (see `CONTRIBUTING.md`).
- Required PR checks: tests, lint, and coverage.
- Include a clear description and link relevant issues; note breaking changes.

## Security & Configuration Tips
- Use `preflight.yaml` plus `layers/*.yaml` for configuration.
- Avoid committing secrets; configs should reference paths or env vars.
