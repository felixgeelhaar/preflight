package tui

import (
	"testing"
	"time"

	"github.com/felixgeelhaar/preflight/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestConvertDoctorIssue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    app.DoctorIssue
		expected DoctorIssue
	}{
		{
			name: "converts error severity",
			input: app.DoctorIssue{
				Provider:   "brew",
				StepID:     "brew:formula:git",
				Severity:   app.SeverityError,
				Message:    "Package not installed",
				Expected:   "installed",
				Actual:     "missing",
				Fixable:    true,
				FixCommand: "preflight apply",
			},
			expected: DoctorIssue{
				Severity:   IssueSeverityError,
				Category:   "brew",
				Message:    "Package not installed",
				Details:    "Expected: installed, Actual: missing",
				CanAutoFix: true,
				FixCommand: "preflight apply",
			},
		},
		{
			name: "converts warning severity",
			input: app.DoctorIssue{
				Provider:   "files",
				StepID:     "files:link:gitconfig",
				Severity:   app.SeverityWarning,
				Message:    "File drift detected",
				Expected:   "symlink",
				Actual:     "modified",
				Fixable:    true,
				FixCommand: "preflight apply",
			},
			expected: DoctorIssue{
				Severity:   IssueSeverityWarning,
				Category:   "files",
				Message:    "File drift detected",
				Details:    "Expected: symlink, Actual: modified",
				CanAutoFix: true,
				FixCommand: "preflight apply",
			},
		},
		{
			name: "converts info severity",
			input: app.DoctorIssue{
				Provider: "git",
				StepID:   "git:config:user",
				Severity: app.SeverityInfo,
				Message:  "Configuration unknown",
				Fixable:  false,
			},
			expected: DoctorIssue{
				Severity:   IssueSeverityInfo,
				Category:   "git",
				Message:    "Configuration unknown",
				Details:    "",
				CanAutoFix: false,
				FixCommand: "",
			},
		},
		{
			name: "handles empty expected/actual",
			input: app.DoctorIssue{
				Provider: "shell",
				StepID:   "shell:framework:zsh",
				Severity: app.SeverityWarning,
				Message:  "Framework needs update",
				Fixable:  true,
			},
			expected: DoctorIssue{
				Severity:   IssueSeverityWarning,
				Category:   "shell",
				Message:    "Framework needs update",
				Details:    "",
				CanAutoFix: true,
				FixCommand: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ConvertDoctorIssue(tt.input)

			assert.Equal(t, tt.expected.Severity, result.Severity)
			assert.Equal(t, tt.expected.Category, result.Category)
			assert.Equal(t, tt.expected.Message, result.Message)
			assert.Equal(t, tt.expected.Details, result.Details)
			assert.Equal(t, tt.expected.CanAutoFix, result.CanAutoFix)
			assert.Equal(t, tt.expected.FixCommand, result.FixCommand)
		})
	}
}

func TestConvertDoctorReport(t *testing.T) {
	t.Parallel()

	t.Run("converts empty report", func(t *testing.T) {
		t.Parallel()

		input := &app.DoctorReport{
			ConfigPath: "preflight.yaml",
			Target:     "default",
			Issues:     []app.DoctorIssue{},
			CheckedAt:  time.Now(),
			Duration:   100 * time.Millisecond,
		}

		result := ConvertDoctorReport(input)

		assert.NotNil(t, result)
		assert.Empty(t, result.Issues)
		assert.False(t, result.HasIssues())
	})

	t.Run("converts report with multiple issues", func(t *testing.T) {
		t.Parallel()

		input := &app.DoctorReport{
			ConfigPath: "preflight.yaml",
			Target:     "default",
			Issues: []app.DoctorIssue{
				{
					Provider:   "brew",
					StepID:     "brew:formula:git",
					Severity:   app.SeverityError,
					Message:    "Package not installed",
					Expected:   "installed",
					Actual:     "missing",
					Fixable:    true,
					FixCommand: "preflight apply",
				},
				{
					Provider: "files",
					StepID:   "files:link:bashrc",
					Severity: app.SeverityWarning,
					Message:  "File drift",
					Fixable:  false,
				},
			},
			CheckedAt: time.Now(),
			Duration:  200 * time.Millisecond,
		}

		result := ConvertDoctorReport(input)

		assert.NotNil(t, result)
		assert.Len(t, result.Issues, 2)
		assert.True(t, result.HasIssues())
		assert.Equal(t, 1, result.FixableCount())

		// Check first issue
		assert.Equal(t, IssueSeverityError, result.Issues[0].Severity)
		assert.Equal(t, "brew", result.Issues[0].Category)

		// Check second issue
		assert.Equal(t, IssueSeverityWarning, result.Issues[1].Severity)
		assert.Equal(t, "files", result.Issues[1].Category)
	})

	t.Run("handles nil report", func(t *testing.T) {
		t.Parallel()

		result := ConvertDoctorReport(nil)

		assert.NotNil(t, result)
		assert.Empty(t, result.Issues)
	})
}

func TestConvertSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    app.IssueSeverity
		expected IssueSeverity
	}{
		{app.SeverityError, IssueSeverityError},
		{app.SeverityWarning, IssueSeverityWarning},
		{app.SeverityInfo, IssueSeverityInfo},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			t.Parallel()

			result := convertSeverity(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertCapturedItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        app.CapturedItem
		expectedType CaptureType
	}{
		{
			name: "converts brew item to formula",
			input: app.CapturedItem{
				Provider:   "brew",
				Name:       "git",
				Value:      "2.40.0",
				Source:     "brew list",
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeFormula,
		},
		{
			name: "converts git item",
			input: app.CapturedItem{
				Provider:   "git",
				Name:       "user.email",
				Value:      "test@example.com",
				Source:     "~/.gitconfig",
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeGit,
		},
		{
			name: "converts ssh item",
			input: app.CapturedItem{
				Provider:   "ssh",
				Name:       "github.com",
				Source:     "~/.ssh/config",
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeSSH,
		},
		{
			name: "converts runtime item",
			input: app.CapturedItem{
				Provider:   "runtime",
				Name:       "go",
				Value:      "1.22.0",
				Source:     "rtx list",
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeRuntime,
		},
		{
			name: "handles redacted items",
			input: app.CapturedItem{
				Provider:   "ssh",
				Name:       "id_rsa",
				Source:     "~/.ssh/id_rsa",
				Redacted:   true,
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeSSH,
		},
		{
			name: "converts nvim item",
			input: app.CapturedItem{
				Provider:   "nvim",
				Name:       "lazy-lock.json",
				Value:      "~/.config/nvim/lazy-lock.json",
				Source:     "~/.config/nvim/lazy-lock.json",
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeNvim,
		},
		{
			name: "converts vscode item to extension",
			input: app.CapturedItem{
				Provider:   "vscode",
				Name:       "golang.go",
				Value:      "golang.go",
				Source:     "code --list-extensions",
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeExtension,
		},
		{
			name: "converts shell item",
			input: app.CapturedItem{
				Provider:   "shell",
				Name:       ".zshrc",
				Value:      "~/.zshrc",
				Source:     "~/.zshrc",
				CapturedAt: time.Now(),
			},
			expectedType: CaptureTypeShell,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ConvertCapturedItem(tt.input)

			assert.Equal(t, tt.input.Provider, result.Category)
			assert.Equal(t, tt.input.Name, result.Name)
			assert.Equal(t, tt.expectedType, result.Type)

			if tt.input.Redacted {
				assert.Contains(t, result.Details, "[REDACTED]")
			}
		})
	}
}

func TestConvertCapturedItems(t *testing.T) {
	t.Parallel()

	items := []app.CapturedItem{
		{
			Provider:   "brew",
			Name:       "git",
			Value:      "2.40.0",
			Source:     "brew list",
			CapturedAt: time.Now(),
		},
		{
			Provider:   "git",
			Name:       "user.name",
			Value:      "Test User",
			Source:     "~/.gitconfig",
			CapturedAt: time.Now(),
		},
	}

	result := ConvertCapturedItems(items)

	assert.Len(t, result, 2)
	assert.Equal(t, "brew", result[0].Category)
	assert.Equal(t, "git", result[0].Name)
	assert.Equal(t, CaptureTypeFormula, result[0].Type)
	assert.Equal(t, "git", result[1].Category)
	assert.Equal(t, "user.name", result[1].Name)
	assert.Equal(t, CaptureTypeGit, result[1].Type)
}
