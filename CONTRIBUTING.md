# Contributing to Preflight

Thank you for your interest in contributing to Preflight! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.23 or later
- golangci-lint
- coverctl (`go install github.com/felixgeelhaar/coverctl@latest`)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/felixgeelhaar/preflight.git
cd preflight

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint
```

## Development Workflow

### Test-Driven Development (TDD)

We follow strict TDD practices. Every change must follow the Red-Green-Refactor cycle:

1. **Red**: Write a failing test that defines expected behavior
2. **Green**: Write minimal code to make the test pass
3. **Refactor**: Clean up while keeping tests green

Each commit should represent one complete TDD cycle.

### Coverage Requirements

- All domains must maintain **80%+ test coverage**
- Use `make coverage-check` to verify coverage
- Coverage is enforced per-domain using coverctl

### Code Style

- Follow standard Go conventions
- Run `make lint` before committing
- The linter configuration is in `.golangci.yml`

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Examples

```
feat(provider): add vscode extension management

fix(config): handle empty layer files gracefully

docs: update installation instructions

test(compiler): add step graph cycle detection tests
```

## Pull Request Process

1. **Fork** the repository
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
3. **Make your changes** following TDD
4. **Run tests and linter**:
   ```bash
   make test
   make lint
   make coverage-check
   ```
5. **Commit** with conventional commit messages
6. **Push** to your fork
7. **Open a Pull Request** against `main`

### PR Requirements

- All CI checks must pass
- Coverage must meet thresholds
- Code review approval required
- Conventional commit messages

## Architecture Guidelines

### Domain-Driven Design

Preflight uses DDD with clear bounded contexts:

- **config**: Configuration loading, merging, validation
- **compiler**: Step graph compilation
- **execution**: Step execution engine
- **provider**: System integration adapters

### Provider Pattern

Providers implement the `Provider` interface:

```go
type Provider interface {
    Name() string
    Compile(ctx CompileContext) ([]Step, error)
}
```

### Step Pattern

Steps implement the `Step` interface:

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

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones
- Use discussions for questions

## Code of Conduct

Be respectful and constructive in all interactions. We're building something together.
