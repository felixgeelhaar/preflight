// Package merge provides three-way merge functionality for file content.
package merge

import (
	"fmt"
	"strings"
)

// ConflictStyle defines the style of conflict markers to use.
type ConflictStyle string

const (
	// StyleGit uses standard git-style conflict markers.
	StyleGit ConflictStyle = "git"
	// StyleDiff3 uses diff3-style with base content included.
	StyleDiff3 ConflictStyle = "diff3"
)

// Conflict represents a merge conflict.
type Conflict struct {
	// Start is the line number where the conflict starts (0-indexed).
	Start int
	// End is the line number where the conflict ends (0-indexed).
	End int
	// Base is the original content (what was last applied).
	Base []string
	// Ours is our content (from config).
	Ours []string
	// Theirs is their content (current file with user modifications).
	Theirs []string
}

// Result represents the result of a three-way merge.
type Result struct {
	// Content is the merged content (with conflict markers if needed).
	Content string
	// HasConflicts indicates whether conflicts were detected.
	HasConflicts bool
	// Conflicts lists all detected conflicts.
	Conflicts []Conflict
	// CleanMerge indicates the merge completed without conflicts.
	CleanMerge bool
}

// NewCleanResult creates a result for a clean merge.
func NewCleanResult(content string) Result {
	return Result{
		Content:      content,
		HasConflicts: false,
		Conflicts:    nil,
		CleanMerge:   true,
	}
}

// NewConflictResult creates a result with conflicts.
func NewConflictResult(content string, conflicts []Conflict) Result {
	return Result{
		Content:      content,
		HasConflicts: len(conflicts) > 0,
		Conflicts:    conflicts,
		CleanMerge:   false,
	}
}

// ThreeWayMerge performs a three-way merge.
// - base: The original content (what was last applied)
// - ours: Our content (from config)
// - theirs: Their content (current file with user modifications)
//
// For v1.3 MVP, this uses a simple strategy:
// - If base == ours: file changed only, use theirs
// - If base == theirs: config changed only, use ours
// - If ours == theirs: both made same change, use either
// - Otherwise: generate conflict markers for the entire file
func ThreeWayMerge(base, ours, theirs string, style ConflictStyle) Result {
	// Fast path: if base equals ours, user made all changes - use theirs
	if base == ours {
		return NewCleanResult(theirs)
	}

	// Fast path: if base equals theirs, config made all changes - use ours
	if base == theirs {
		return NewCleanResult(ours)
	}

	// Fast path: if ours equals theirs, both made same changes
	if ours == theirs {
		return NewCleanResult(ours)
	}

	// Both sides diverged from base differently - create conflict
	conflict := Conflict{
		Start:  0,
		Base:   splitLines(base),
		Ours:   splitLines(ours),
		Theirs: splitLines(theirs),
	}

	content := formatWholeFileConflict(conflict, style)
	conflict.End = len(splitLines(content)) - 1

	return NewConflictResult(content, []Conflict{conflict})
}

// formatWholeFileConflict formats a whole-file conflict with markers.
func formatWholeFileConflict(c Conflict, style ConflictStyle) string {
	var lines []string

	lines = append(lines, "<<<<<<< ours (config)")
	lines = append(lines, c.Ours...)

	if style == StyleDiff3 {
		lines = append(lines, "||||||| base")
		lines = append(lines, c.Base...)
	}

	lines = append(lines, "=======")
	lines = append(lines, c.Theirs...)
	lines = append(lines, ">>>>>>> theirs (file)")

	return strings.Join(lines, "\n") + "\n"
}

// splitLines splits content into lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

// sliceEqual checks if two string slices are equal.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ConflictMarkers contains the standard conflict marker strings.
var ConflictMarkers = struct {
	Start  string
	Middle string
	End    string
	Base   string
}{
	Start:  "<<<<<<<",
	Middle: "=======",
	End:    ">>>>>>>",
	Base:   "|||||||",
}

// HasConflictMarkers checks if content contains conflict markers.
func HasConflictMarkers(content string) bool {
	return strings.Contains(content, ConflictMarkers.Start) &&
		strings.Contains(content, ConflictMarkers.Middle) &&
		strings.Contains(content, ConflictMarkers.End)
}

// ParseConflictRegions extracts conflict regions from content with markers.
func ParseConflictRegions(content string) []Conflict {
	lines := splitLines(content)
	var conflicts []Conflict
	var current *Conflict
	var section string // "ours", "base", "theirs"

	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, ConflictMarkers.Start):
			current = &Conflict{Start: i}
			section = "ours"
		case strings.HasPrefix(line, ConflictMarkers.Base):
			section = "base"
		case line == ConflictMarkers.Middle:
			section = "theirs"
		case strings.HasPrefix(line, ConflictMarkers.End):
			if current != nil {
				current.End = i
				conflicts = append(conflicts, *current)
				current = nil
			}
			section = ""
		default:
			if current != nil {
				switch section {
				case "ours":
					current.Ours = append(current.Ours, line)
				case "base":
					current.Base = append(current.Base, line)
				case "theirs":
					current.Theirs = append(current.Theirs, line)
				}
			}
		}
	}

	return conflicts
}

// Resolution specifies how to resolve a conflict by choosing a side.
type Resolution string

const (
	// ResolveOurs uses our content (from config).
	ResolveOurs Resolution = "ours"
	// ResolveTheirs uses their content (from file).
	ResolveTheirs Resolution = "theirs"
	// ResolveBase uses the base content.
	ResolveBase Resolution = "base"
)

// ResolveAllConflicts resolves all conflicts in content with the given resolution.
func ResolveAllConflicts(content string, resolution Resolution) string {
	lines := splitLines(content)
	var result []string
	var inConflict bool
	var keeping bool

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, ConflictMarkers.Start):
			inConflict = true
			keeping = resolution == ResolveOurs
		case strings.HasPrefix(line, ConflictMarkers.Base):
			keeping = resolution == ResolveBase
		case line == ConflictMarkers.Middle:
			keeping = resolution == ResolveTheirs
		case strings.HasPrefix(line, ConflictMarkers.End):
			inConflict = false
			keeping = false
		default:
			if !inConflict || keeping {
				result = append(result, line)
			}
		}
	}

	if len(result) == 0 {
		return ""
	}
	return strings.Join(result, "\n") + "\n"
}

// ConflictSummary provides a summary of conflicts in a file.
type ConflictSummary struct {
	Path            string
	ConflictCount   int
	OursLineCount   int
	TheirsLineCount int
	BaseLineCount   int
}

// Summarize creates a summary of conflicts.
func Summarize(path string, conflicts []Conflict) ConflictSummary {
	summary := ConflictSummary{
		Path:          path,
		ConflictCount: len(conflicts),
	}

	for _, c := range conflicts {
		summary.OursLineCount += len(c.Ours)
		summary.TheirsLineCount += len(c.Theirs)
		summary.BaseLineCount += len(c.Base)
	}

	return summary
}

// Description returns a human-readable description of the summary.
func (s ConflictSummary) Description() string {
	if s.ConflictCount == 0 {
		return fmt.Sprintf("%s: no conflicts", s.Path)
	}
	return fmt.Sprintf("%s: %d conflict(s) - %d lines from config, %d lines from file",
		s.Path, s.ConflictCount, s.OursLineCount, s.TheirsLineCount)
}

// ChangeType describes what type of change occurred.
type ChangeType string

const (
	// ChangeNone means no change from base.
	ChangeNone ChangeType = "none"
	// ChangeOurs means only config (ours) changed.
	ChangeOurs ChangeType = "ours"
	// ChangeTheirs means only file (theirs) changed.
	ChangeTheirs ChangeType = "theirs"
	// ChangeBoth means both sides changed.
	ChangeBoth ChangeType = "both"
	// ChangeSame means both made identical changes.
	ChangeSame ChangeType = "same"
)

// DetectChangeType determines what type of change occurred between base, ours, and theirs.
func DetectChangeType(base, ours, theirs string) ChangeType {
	baseChanged := base != ours
	fileChanged := base != theirs

	if !baseChanged && !fileChanged {
		return ChangeNone
	}

	if baseChanged && !fileChanged {
		return ChangeOurs
	}

	if !baseChanged && fileChanged {
		return ChangeTheirs
	}

	// Both changed - check if same change
	if ours == theirs {
		return ChangeSame
	}

	return ChangeBoth
}

// NeedsManualResolution returns true if the change type requires user intervention.
func NeedsManualResolution(ct ChangeType) bool {
	return ct == ChangeBoth
}

// Description returns a human-readable description of the change type.
func (ct ChangeType) Description() string {
	switch ct {
	case ChangeNone:
		return "no changes"
	case ChangeOurs:
		return "config changed"
	case ChangeTheirs:
		return "file changed externally"
	case ChangeBoth:
		return "both config and file changed (conflict)"
	case ChangeSame:
		return "both made identical changes"
	default:
		return string(ct)
	}
}
