---
title: Contributing
description: Guidelines for contributing to Preflight.
---

Thank you for your interest in contributing to Preflight! This guide will help you get started.

## Getting Started

### Prerequisites

- **Go 1.23+**
- **golangci-lint**
- **coverctl** (for coverage checks)

### Clone and Build

```bash
git clone https://github.com/felixgeelhaar/preflight.git
cd preflight
make build
./bin/preflight version
```

### Run Tests

```bash
# All tests
make test

# With race detection
make test-race

# Coverage check
make coverage-check
```

### Run Linter

```bash
make lint
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/my-feature
```

### 2. Follow TDD

Write tests before implementation. See [TDD Workflow](/preflight/development/tdd/).

### 3. Make Atomic Commits

Each commit should be a complete, working change:

```bash
git commit -m "feat(provider): add brew tap support"
git commit -m "test(provider): add brew tap tests"
git commit -m "docs(provider): document brew tap configuration"
```

### 4. Ensure Quality

```bash
# Run all checks
make lint
make test
make coverage-check
```

### 5. Submit PR

```bash
git push origin feature/my-feature
```

Create a pull request on GitHub.

## Code Style

### Go Conventions

- Follow standard Go conventions
- Use `gofmt` and `goimports`
- Exported functions must have documentation

### Naming

```go
// Good
func (m *Merger) MergeLayers(base, overlay Layer) (*MergedLayer, error)

// Bad
func (m *Merger) DoMerge(l1, l2 Layer) (*Layer, error)
```

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("merge layers: %w", err)
}

// Bad
if err != nil {
    return err
}
```

### Comments

```go
// MergeLayers combines two layers with deep merge semantics.
// Scalars use last-wins, maps are deep merged, lists are unioned.
func (m *Merger) MergeLayers(base, overlay Layer) (*MergedLayer, error)
```

## Project Structure

```
cmd/preflight/           # CLI entry point
internal/
  app/                   # Application services
  domain/                # Domain logic (DDD)
    config/              # Configuration domain
    compiler/            # Compilation domain
    execution/           # Execution domain
  provider/              # System adapters
  tui/                   # Terminal UI
  ports/                 # Interface definitions
  adapters/              # Port implementations
```

### Adding a New Domain

1. Create domain directory: `internal/domain/mydomain/`
2. Define entities and value objects
3. Implement domain services
4. Add tests with >80% coverage
5. Register in `.coverctl.yaml`

### Adding a New Provider

1. Create provider directory: `internal/provider/myprovider/`
2. Implement `Provider` interface
3. Define steps with `Check()`, `Plan()`, `Apply()`
4. Add doctor checks
5. Register in compiler

## Testing Guidelines

### Coverage Requirements

All domains must maintain >80% test coverage.

```yaml
# .coverctl.yaml
thresholds:
  internal/domain/config: 80
  internal/domain/compiler: 80
  internal/provider/*: 80
  internal/tui: 70
```

### Test Organization

```go
func TestFeature_Scenario(t *testing.T) {
    t.Parallel()

    // Arrange
    input := NewTestInput()

    // Act
    result := DoSomething(input)

    // Assert
    assert.Equal(t, expected, result)
}
```

### Use Table-Driven Tests

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    Input
        expected Output
    }{
        {"case 1", input1, output1},
        {"case 2", input2, output2},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            result := Function(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### End-to-End Tests

Preflight includes 6 E2E test suites (281 total assertions) that run in Docker containers:

```bash
# Run all E2E suites
docker compose -f docker-compose.test.yml up --build

# Run individual suites
docker compose -f docker-compose.test.yml run e2e-cli-smoke
docker compose -f docker-compose.test.yml run e2e-config-evolution
docker compose -f docker-compose.test.yml run e2e-multi-target
docker compose -f docker-compose.test.yml run e2e-operations
```

See [TDD Workflow](/preflight/development/tdd/) for details on E2E test coverage and how to write new tests.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes nor adds |
| `test` | Adding or updating tests |
| `docs` | Documentation only |
| `chore` | Build, CI, or tooling |

### Examples

```
feat(provider): add vscode extension management
fix(config): handle empty layer content
refactor(compiler): extract step sorting logic
test(merge): add three-way merge edge cases
docs(readme): update installation instructions
```

## Pull Request Process

### Before Submitting

- [ ] All tests pass
- [ ] Lint passes
- [ ] Coverage thresholds met
- [ ] Documentation updated
- [ ] Conventional commit messages

### PR Description

Include:
- What changes were made
- Why the changes were necessary
- How to test the changes
- Any breaking changes

### Review Process

1. Automated checks run (CI)
2. Code review by maintainers
3. Address feedback
4. Merge when approved

## Issue Reporting

### Bug Reports

Include:
- Preflight version
- OS and version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or output

### Feature Requests

Include:
- Use case description
- Proposed solution
- Alternatives considered
- Willingness to implement

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow
- Maintain a welcoming environment

## Getting Help

- [GitHub Issues](https://github.com/felixgeelhaar/preflight/issues)
- [GitHub Discussions](https://github.com/felixgeelhaar/preflight/discussions)

Thank you for contributing!
