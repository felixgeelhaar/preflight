package plugin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// ResolutionMode determines how version conflicts are handled.
type ResolutionMode string

const (
	// ResolutionStrict fails on any version conflict.
	ResolutionStrict ResolutionMode = "strict"
	// ResolutionLatest uses the latest compatible version.
	ResolutionLatest ResolutionMode = "latest"
)

// DependencyConflict represents a version conflict between dependencies.
type DependencyConflict struct {
	Name       string
	Version1   string
	RequiredBy string
	Version2   string
}

// ResolutionResult contains the result of dependency resolution.
type ResolutionResult struct {
	// Resolved contains all resolved dependencies in install order.
	Resolved []Dependency
	// Missing contains dependencies that could not be found.
	Missing []Dependency
	// Conflicts contains version conflicts (only in strict mode).
	Conflicts []DependencyConflict
	// InstallOrder contains plugin names in topological order.
	InstallOrder []string
}

// HasErrors returns true if there are missing or conflicting dependencies.
func (r *ResolutionResult) HasErrors() bool {
	return len(r.Missing) > 0 || len(r.Conflicts) > 0
}

// DependencyResolver resolves plugin dependencies.
type DependencyResolver struct {
	registry *Registry
	mode     ResolutionMode
}

// NewDependencyResolver creates a new resolver with the given registry.
func NewDependencyResolver(registry *Registry, mode ResolutionMode) *DependencyResolver {
	if mode == "" {
		mode = ResolutionLatest
	}
	return &DependencyResolver{
		registry: registry,
		mode:     mode,
	}
}

// Resolve resolves all dependencies for a manifest.
func (r *DependencyResolver) Resolve(_ context.Context, manifest *Manifest) (*ResolutionResult, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest cannot be nil")
	}

	result := &ResolutionResult{
		Resolved:     make([]Dependency, 0),
		Missing:      make([]Dependency, 0),
		Conflicts:    make([]DependencyConflict, 0),
		InstallOrder: make([]string, 0),
	}

	// Build dependency graph
	graph := make(map[string][]string)
	versions := make(map[string]string)
	requiredBy := make(map[string]string)

	// Add manifest's direct dependencies
	for _, dep := range manifest.Requires {
		if err := r.resolveDependency(dep, manifest.Name, graph, versions, requiredBy, result); err != nil {
			return nil, err
		}
	}

	// If there are missing or conflicting dependencies, return early
	if result.HasErrors() {
		return result, nil
	}

	// Detect cycles and build install order
	order, cycleErr := topologicalSort(graph)
	if cycleErr != nil {
		return nil, cycleErr
	}

	result.InstallOrder = order

	// Build resolved list from install order
	for _, name := range order {
		if version, ok := versions[name]; ok {
			result.Resolved = append(result.Resolved, Dependency{
				Name:    name,
				Version: version,
			})
		}
	}

	return result, nil
}

// resolveDependency recursively resolves a single dependency.
func (r *DependencyResolver) resolveDependency(
	dep Dependency,
	parentName string,
	graph map[string][]string,
	versions map[string]string,
	requiredBy map[string]string,
	result *ResolutionResult,
) error {
	// Check if already resolved
	if existingVersion, ok := versions[dep.Name]; ok {
		// Version conflict check
		if dep.Version != "" && existingVersion != "" && dep.Version != existingVersion {
			if r.mode == ResolutionStrict {
				result.Conflicts = append(result.Conflicts, DependencyConflict{
					Name:       dep.Name,
					Version1:   existingVersion,
					RequiredBy: requiredBy[dep.Name],
					Version2:   dep.Version,
				})
				return nil
			}
			// In latest mode, use the higher version
			if compareVersionConstraints(dep.Version, existingVersion) > 0 {
				versions[dep.Name] = dep.Version
				requiredBy[dep.Name] = parentName
			}
		}
		// Already resolved, just add edge
		graph[parentName] = append(graph[parentName], dep.Name)
		return nil
	}

	// Try to find in registry
	var plugin *Plugin
	if r.registry != nil {
		plugin, _ = r.registry.Get(dep.Name)
	}

	if plugin == nil {
		// Dependency not found
		result.Missing = append(result.Missing, dep)
		return nil
	}

	// Check version constraint
	if dep.Version != "" {
		if !satisfiesVersionConstraint(plugin.Manifest.Version, dep.Version) {
			result.Missing = append(result.Missing, dep)
			return nil
		}
	}

	// Mark as resolved
	versions[dep.Name] = dep.Version
	if versions[dep.Name] == "" {
		versions[dep.Name] = plugin.Manifest.Version
	}
	requiredBy[dep.Name] = parentName

	// Add edge
	if _, ok := graph[parentName]; !ok {
		graph[parentName] = make([]string, 0)
	}
	graph[parentName] = append(graph[parentName], dep.Name)

	// Initialize node in graph
	if _, ok := graph[dep.Name]; !ok {
		graph[dep.Name] = make([]string, 0)
	}

	// Resolve transitive dependencies
	for _, transitiveDep := range plugin.Manifest.Requires {
		if err := r.resolveDependency(transitiveDep, dep.Name, graph, versions, requiredBy, result); err != nil {
			return err
		}
	}

	return nil
}

// CyclicDependencyError indicates a cyclic dependency was detected.
type CyclicDependencyError struct {
	Cycle []string
}

func (e *CyclicDependencyError) Error() string {
	return fmt.Sprintf("cyclic dependency detected: %s", strings.Join(e.Cycle, " -> "))
}

// IsCyclicDependency returns true if the error is a cyclic dependency error.
func IsCyclicDependency(err error) bool {
	var cyclicErr *CyclicDependencyError
	return errors.As(err, &cyclicErr)
}

// topologicalSort performs a topological sort on the dependency graph.
// Returns an error if a cycle is detected.
func topologicalSort(graph map[string][]string) ([]string, error) {
	// State: 0 = unvisited, 1 = visiting, 2 = visited
	state := make(map[string]int)
	result := make([]string, 0, len(graph))
	var currentPath []string

	var visit func(node string) error
	visit = func(node string) error {
		switch state[node] {
		case 1: // Visiting - cycle detected
			// Find cycle start
			cycleStart := -1
			for i, n := range currentPath {
				if n == node {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := make([]string, len(currentPath[cycleStart:])+1)
				copy(cycle, currentPath[cycleStart:])
				cycle[len(cycle)-1] = node
				return &CyclicDependencyError{Cycle: cycle}
			}
			return &CyclicDependencyError{Cycle: []string{node}}
		case 2: // Already visited
			return nil
		}

		state[node] = 1 // Mark as visiting
		currentPath = append(currentPath, node)

		for _, neighbor := range graph[node] {
			if err := visit(neighbor); err != nil {
				return err
			}
		}

		state[node] = 2 // Mark as visited
		currentPath = currentPath[:len(currentPath)-1]
		result = append(result, node)
		return nil
	}

	// Visit all nodes
	for node := range graph {
		if state[node] == 0 {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	// Post-order DFS naturally gives dependency order (dependencies first)
	// No reversal needed as nodes are added after their dependencies
	return result, nil
}

// satisfiesVersionConstraint checks if a version satisfies a constraint.
// Supports: =, >=, <=, >, <, ^, ~ prefixes.
func satisfiesVersionConstraint(version, constraint string) bool {
	if constraint == "" || version == "" {
		return true
	}

	// Normalize version to semver format
	v := version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}

	// Parse constraint
	var op string
	var cv string

	switch {
	case strings.HasPrefix(constraint, ">="):
		op = ">="
		cv = strings.TrimPrefix(constraint, ">=")
	case strings.HasPrefix(constraint, "<="):
		op = "<="
		cv = strings.TrimPrefix(constraint, "<=")
	case strings.HasPrefix(constraint, ">"):
		op = ">"
		cv = strings.TrimPrefix(constraint, ">")
	case strings.HasPrefix(constraint, "<"):
		op = "<"
		cv = strings.TrimPrefix(constraint, "<")
	case strings.HasPrefix(constraint, "^"):
		op = "^"
		cv = strings.TrimPrefix(constraint, "^")
	case strings.HasPrefix(constraint, "~"):
		op = "~"
		cv = strings.TrimPrefix(constraint, "~")
	case strings.HasPrefix(constraint, "="):
		op = "="
		cv = strings.TrimPrefix(constraint, "=")
	default:
		op = "="
		cv = constraint
	}

	// Normalize constraint version
	cv = strings.TrimSpace(cv)
	if !strings.HasPrefix(cv, "v") {
		cv = "v" + cv
	}

	if !semver.IsValid(v) || !semver.IsValid(cv) {
		return false
	}

	cmp := semver.Compare(v, cv)

	switch op {
	case "=":
		return cmp == 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case "^":
		// Compatible with major version
		return cmp >= 0 && semver.Major(v) == semver.Major(cv)
	case "~":
		// Compatible with minor version
		return cmp >= 0 && semver.Major(v) == semver.Major(cv) && semverMinor(v) == semverMinor(cv)
	default:
		return cmp == 0
	}
}

// semverMinor extracts the minor version component.
func semverMinor(v string) string {
	// v1.2.3 -> 1.2
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return parts[0]
}

// compareVersionConstraints compares two version constraints.
// Returns >0 if c1 > c2, <0 if c1 < c2, 0 if equal.
func compareVersionConstraints(c1, c2 string) int {
	// Extract version numbers from constraints
	v1 := extractVersion(c1)
	v2 := extractVersion(c2)

	// Normalize to semver
	if !strings.HasPrefix(v1, "v") {
		v1 = "v" + v1
	}
	if !strings.HasPrefix(v2, "v") {
		v2 = "v" + v2
	}

	return semver.Compare(v1, v2)
}

// extractVersion extracts the version number from a constraint.
func extractVersion(constraint string) string {
	for _, prefix := range []string{">=", "<=", ">", "<", "^", "~", "="} {
		if strings.HasPrefix(constraint, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(constraint, prefix))
		}
	}
	return strings.TrimSpace(constraint)
}
