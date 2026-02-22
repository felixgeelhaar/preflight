package ports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandResult_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exitCode int
		want     bool
	}{
		{
			name:     "exit code 0 is success",
			exitCode: 0,
			want:     true,
		},
		{
			name:     "exit code 1 is failure",
			exitCode: 1,
			want:     false,
		},
		{
			name:     "exit code 127 is failure",
			exitCode: 127,
			want:     false,
		},
		{
			name:     "negative exit code is failure",
			exitCode: -1,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CommandResult{ExitCode: tt.exitCode}
			assert.Equal(t, tt.want, result.Success())
		})
	}
}

func TestCommandCall_Fields(t *testing.T) {
	t.Parallel()

	call := CommandCall{
		Command: "brew",
		Args:    []string{"install", "git"},
	}

	assert.Equal(t, "brew", call.Command)
	assert.Equal(t, []string{"install", "git"}, call.Args)
}

func TestCommandResult_Fields(t *testing.T) {
	t.Parallel()

	result := CommandResult{
		ExitCode: 0,
		Stdout:   "installed",
		Stderr:   "warning: already linked",
	}

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "installed", result.Stdout)
	assert.Equal(t, "warning: already linked", result.Stderr)
}
