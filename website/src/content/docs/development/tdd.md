---
title: TDD Workflow
description: Test-Driven Development practices for Preflight.
---

Preflight follows strict Test-Driven Development (TDD) practices. All code is written test-first using the Red-Green-Refactor cycle.

## Red-Green-Refactor Cycle

### 1. Red — Write a Failing Test

Write a test that defines the expected behavior before implementing the code.

```go
func TestMerger_MergesScalarsLastWins(t *testing.T) {
    t.Parallel()

    base := Layer{Content: map[string]any{"shell": "bash"}}
    overlay := Layer{Content: map[string]any{"shell": "zsh"}}

    merger := NewMerger()
    result, err := merger.Merge(base, overlay)

    require.NoError(t, err)
    assert.Equal(t, "zsh", result.Content["shell"])
}
```

### 2. Green — Make It Pass

Write the minimal code necessary to make the test pass.

```go
func (m *Merger) Merge(base, overlay Layer) (*MergedLayer, error) {
    result := &MergedLayer{Content: make(map[string]any)}

    // Copy base
    for k, v := range base.Content {
        result.Content[k] = v
    }

    // Overlay wins
    for k, v := range overlay.Content {
        result.Content[k] = v
    }

    return result, nil
}
```

### 3. Refactor — Clean Up

Improve the code while keeping tests green.

```go
func (m *Merger) Merge(base, overlay Layer) (*MergedLayer, error) {
    result := base.Copy()
    return result.ApplyOverlay(overlay), nil
}
```

## Commit Strategy

Each commit should represent one complete TDD cycle.

### Atomic Commits

```bash
# After each red-green-refactor cycle
git add .
git commit -m "feat(config): add scalar merge with last-wins semantics"
```

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat` — New feature
- `fix` — Bug fix
- `refactor` — Code change that neither fixes nor adds
- `test` — Adding or updating tests
- `docs` — Documentation only

## Coverage Requirements

All domains must maintain >80% test coverage.

### Check Coverage

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Check per-domain coverage
coverctl check --threshold 80

# View detailed report
coverctl report
```

### Coverage Thresholds

| Domain | Threshold |
|--------|-----------|
| config | 80% |
| compiler | 80% |
| execution | 80% |
| lock | 80% |
| provider/* | 80% |
| tui | 70% |

## Test Organization

### File Structure

```
internal/domain/config/
  merger.go
  merger_test.go      # Unit tests
  merger_integration_test.go  # Integration tests
```

### Test Naming

```go
// Function: Merge
// Test: TestMerge_<scenario>
func TestMerge_ScalarsLastWins(t *testing.T) {}
func TestMerge_MapsDeepMerge(t *testing.T) {}
func TestMerge_ListsUnion(t *testing.T) {}
```

### Table-Driven Tests

```go
func TestDetectChangeType(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        base     string
        ours     string
        theirs   string
        expected ChangeType
    }{
        {"no changes", "a", "a", "a", ChangeNone},
        {"ours changed", "a", "b", "a", ChangeOurs},
        {"theirs changed", "a", "a", "b", ChangeTheirs},
        {"both changed same", "a", "b", "b", ChangeSame},
        {"both changed different", "a", "b", "c", ChangeBoth},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            result := DetectChangeType(tt.base, tt.ours, tt.theirs)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Test Utilities

### Builders

Create test data with builder patterns:

```go
func NewTestLayer() *LayerBuilder {
    return &LayerBuilder{layer: Layer{Content: make(map[string]any)}}
}

func (b *LayerBuilder) WithPackages(pkgs ...string) *LayerBuilder {
    b.layer.Content["packages"] = pkgs
    return b
}

func (b *LayerBuilder) Build() Layer {
    return b.layer
}

// Usage
layer := NewTestLayer().
    WithPackages("git", "gh").
    Build()
```

### Fixtures

Load test data from files:

```go
//go:embed testdata
var testdata embed.FS

func LoadFixture(t *testing.T, name string) []byte {
    t.Helper()
    data, err := testdata.ReadFile("testdata/" + name)
    require.NoError(t, err)
    return data
}
```

### Mocks

Use interfaces for testability:

```go
// Port
type FileSystem interface {
    Read(path string) ([]byte, error)
    Write(path string, data []byte) error
}

// Mock
type MockFileSystem struct {
    Files map[string][]byte
}

func (m *MockFileSystem) Read(path string) ([]byte, error) {
    if data, ok := m.Files[path]; ok {
        return data, nil
    }
    return nil, os.ErrNotExist
}
```

## Running Tests

### All Tests

```bash
# Run all tests
go test ./...

# With race detection
go test -race ./...

# Verbose output
go test -v ./...
```

### Specific Domain

```bash
# Test config domain
go test -v ./internal/domain/config/...

# Test specific test
go test -v -run TestMerge_ScalarsLastWins ./internal/domain/config/...
```

### Watch Mode

Use `watchexec` or similar for continuous testing:

```bash
watchexec -e go "go test ./..."
```

## Best Practices

### 1. Test Behavior, Not Implementation

```go
// Good: Tests behavior
func TestMerger_OverlayWins(t *testing.T) {
    result := merger.Merge(base, overlay)
    assert.Equal(t, expected, result)
}

// Bad: Tests implementation details
func TestMerger_CallsDeepMerge(t *testing.T) {
    // Testing internal method calls
}
```

### 2. One Assert Per Test (When Possible)

```go
// Good: Clear what failed
func TestMerge_PreservesProvenance(t *testing.T) {
    result := merger.Merge(base, overlay)
    assert.Equal(t, "overlay.yaml", result.Provenance.Source)
}

// Less ideal: Multiple assertions
func TestMerge(t *testing.T) {
    result := merger.Merge(base, overlay)
    assert.NotNil(t, result)
    assert.Equal(t, "zsh", result.Content["shell"])
    assert.Equal(t, "overlay.yaml", result.Provenance.Source)
}
```

### 3. Use t.Parallel()

```go
func TestFeature(t *testing.T) {
    t.Parallel()  // Run tests concurrently

    t.Run("case 1", func(t *testing.T) {
        t.Parallel()
        // ...
    })
}
```

### 4. Test Edge Cases

```go
func TestMerge_EmptyBase(t *testing.T) {}
func TestMerge_EmptyOverlay(t *testing.T) {}
func TestMerge_NilContent(t *testing.T) {}
func TestMerge_DeepNesting(t *testing.T) {}
```

## Integration Tests

For testing across domain boundaries:

```go
//go:build integration

func TestFullPipeline(t *testing.T) {
    // Load → Merge → Compile → Plan
    manifest := loader.Load("testdata/preflight.yaml")
    merged := merger.Merge(manifest, layers)
    graph := compiler.Compile(merged)
    plan := planner.Plan(graph)

    assert.Len(t, plan.Steps, 5)
}
```

Run with:

```bash
go test -tags=integration ./...
```

## What's Next?

- [Contributing](/preflight/development/contributing/) — How to contribute
- [Architecture](/preflight/architecture/overview/) — System design
