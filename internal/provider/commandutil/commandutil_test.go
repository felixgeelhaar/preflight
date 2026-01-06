package commandutil

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCommandNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"exec ErrNotFound", exec.ErrNotFound, true},
		{"exec error wrapper", &exec.Error{Err: exec.ErrNotFound}, true},
		{"path error", &os.PathError{Err: os.ErrNotExist}, true},
		{"other error", errors.New("nope"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsCommandNotFound(tt.err))
		})
	}
}
