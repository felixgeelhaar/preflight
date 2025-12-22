package compiler

import (
	"testing"
)

func TestDiff_Creation(t *testing.T) {
	diff := NewDiff(DiffTypeAdd, "package", "git", "", "2.43.0")

	if diff.Type() != DiffTypeAdd {
		t.Errorf("Diff.Type() = %v, want %v", diff.Type(), DiffTypeAdd)
	}
	if diff.Resource() != "package" {
		t.Errorf("Diff.Resource() = %q, want %q", diff.Resource(), "package")
	}
	if diff.Name() != "git" {
		t.Errorf("Diff.Name() = %q, want %q", diff.Name(), "git")
	}
	if diff.OldValue() != "" {
		t.Errorf("Diff.OldValue() = %q, want %q", diff.OldValue(), "")
	}
	if diff.NewValue() != "2.43.0" {
		t.Errorf("Diff.NewValue() = %q, want %q", diff.NewValue(), "2.43.0")
	}
}

func TestDiff_Types(t *testing.T) {
	tests := []struct {
		diffType DiffType
		expected string
	}{
		{DiffTypeAdd, "add"},
		{DiffTypeRemove, "remove"},
		{DiffTypeModify, "modify"},
		{DiffTypeNone, "none"},
	}

	for _, tt := range tests {
		if tt.diffType.String() != tt.expected {
			t.Errorf("DiffType.String() = %q, want %q", tt.diffType.String(), tt.expected)
		}
	}
}

func TestDiff_Summary(t *testing.T) {
	tests := []struct {
		name     string
		diff     Diff
		expected string
	}{
		{
			name:     "add package",
			diff:     NewDiff(DiffTypeAdd, "package", "git", "", "2.43.0"),
			expected: "+ package git (2.43.0)",
		},
		{
			name:     "remove package",
			diff:     NewDiff(DiffTypeRemove, "package", "vim", "8.2", ""),
			expected: "- package vim (8.2)",
		},
		{
			name:     "modify file",
			diff:     NewDiff(DiffTypeModify, "file", "~/.gitconfig", "old", "new"),
			expected: "~ file ~/.gitconfig",
		},
		{
			name:     "no change",
			diff:     NewDiff(DiffTypeNone, "package", "curl", "7.88", "7.88"),
			expected: "  package curl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.Summary(); got != tt.expected {
				t.Errorf("Diff.Summary() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDiff_IsEmpty(t *testing.T) {
	emptyDiff := NewDiff(DiffTypeNone, "", "", "", "")
	nonEmptyDiff := NewDiff(DiffTypeAdd, "package", "git", "", "2.43.0")

	if !emptyDiff.IsEmpty() {
		t.Error("expected empty diff to return true for IsEmpty()")
	}
	if nonEmptyDiff.IsEmpty() {
		t.Error("expected non-empty diff to return false for IsEmpty()")
	}
}
