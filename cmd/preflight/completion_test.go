package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletionCmd_GeneratesScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		shell := shell
		t.Run(shell, func(t *testing.T) {
			devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			require.NoError(t, err)
			defer func() { _ = devNull.Close() }()

			oldStdout := os.Stdout
			os.Stdout = devNull
			defer func() {
				os.Stdout = oldStdout
			}()

			err = completionCmd.RunE(completionCmd, []string{shell})
			require.NoError(t, err)
		})
	}
}
