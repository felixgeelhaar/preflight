package compiler

import (
	"errors"
	"testing"
)

func TestNewStepID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid simple ID",
			input:   "brew:install:git",
			wantErr: nil,
		},
		{
			name:    "valid with underscores",
			input:   "files:link:git_config",
			wantErr: nil,
		},
		{
			name:    "valid with hyphens",
			input:   "apt:install:build-essential",
			wantErr: nil,
		},
		{
			name:    "valid versioned package with @",
			input:   "brew:formula:go@1.24",
			wantErr: nil,
		},
		{
			name:    "valid versioned package python@3.12",
			input:   "brew:formula:python@3.12",
			wantErr: nil,
		},
		{
			name:    "valid versioned package openssl@3",
			input:   "brew:formula:openssl@3",
			wantErr: nil,
		},
		{
			name:    "valid winget package with dots",
			input:   "winget:install:Microsoft.VisualStudioCode",
			wantErr: nil,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrEmptyStepID,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: ErrEmptyStepID,
		},
		{
			name:    "contains spaces",
			input:   "brew install git",
			wantErr: ErrInvalidStepID,
		},
		{
			name:    "starts with colon",
			input:   ":install:git",
			wantErr: ErrInvalidStepID,
		},
		{
			name:    "ends with colon",
			input:   "brew:install:",
			wantErr: ErrInvalidStepID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewStepID(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewStepID(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("NewStepID(%q) unexpected error: %v", tt.input, err)
				return
			}
			if id.String() != tt.input {
				t.Errorf("StepID.String() = %q, want %q", id.String(), tt.input)
			}
		})
	}
}

func TestStepID_Equality(t *testing.T) {
	id1, _ := NewStepID("brew:install:git")
	id2, _ := NewStepID("brew:install:git")
	id3, _ := NewStepID("brew:install:curl")

	if !id1.Equals(id2) {
		t.Error("expected id1 to equal id2")
	}
	if id1.Equals(id3) {
		t.Error("expected id1 to not equal id3")
	}
}

func TestStepID_Provider(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"brew:install:git", "brew"},
		{"apt:install:build-essential", "apt"},
		{"files:link:gitconfig", "files"},
	}

	for _, tt := range tests {
		id, _ := NewStepID(tt.input)
		if id.Provider() != tt.expected {
			t.Errorf("StepID(%q).Provider() = %q, want %q", tt.input, id.Provider(), tt.expected)
		}
	}
}

func TestMustNewStepID(t *testing.T) {
	t.Run("valid ID does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustNewStepID panicked unexpectedly: %v", r)
			}
		}()

		id := MustNewStepID("brew:install:git")
		if id.String() != "brew:install:git" {
			t.Errorf("MustNewStepID returned wrong value: %q", id.String())
		}
	})

	t.Run("invalid ID panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustNewStepID should have panicked for invalid ID")
			}
		}()

		MustNewStepID("")
	})

	t.Run("invalid format panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustNewStepID should have panicked for invalid format")
			}
		}()

		MustNewStepID("invalid id with spaces")
	})
}

func TestStepID_IsZero(t *testing.T) {
	t.Run("zero value is zero", func(t *testing.T) {
		var id StepID
		if !id.IsZero() {
			t.Error("zero value StepID should return true for IsZero()")
		}
	})

	t.Run("valid ID is not zero", func(t *testing.T) {
		id, _ := NewStepID("brew:install:git")
		if id.IsZero() {
			t.Error("valid StepID should return false for IsZero()")
		}
	})
}
