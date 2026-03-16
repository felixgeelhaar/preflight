package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvisionHostFunctions(t *testing.T) {
	t.Parallel()

	workDir := "/tmp/infra"
	variables := map[string]string{
		"env":    "production",
		"region": "us-east-1",
	}

	h := NewProvisionHostFunctions(workDir, variables)
	require.NotNil(t, h)
	assert.Equal(t, workDir, h.WorkDir)
	assert.Equal(t, variables, h.Variables)
	assert.Empty(t, h.Output)
}

func TestNewProvisionHostFunctions_NilVariables(t *testing.T) {
	t.Parallel()

	h := NewProvisionHostFunctions("/tmp", nil)
	require.NotNil(t, h)
	assert.NotNil(t, h.Variables, "Variables map should be initialized even when nil is passed")
	assert.Empty(t, h.Variables)
}

func TestProvisionHostFunctions_GetVariable(t *testing.T) {
	t.Parallel()

	variables := map[string]string{
		"env":    "production",
		"region": "us-east-1",
	}
	h := NewProvisionHostFunctions("/tmp", variables)

	t.Run("existing variable", func(t *testing.T) {
		t.Parallel()

		val, ok := h.GetVariable("env")
		assert.True(t, ok)
		assert.Equal(t, "production", val)
	})

	t.Run("missing variable", func(t *testing.T) {
		t.Parallel()

		val, ok := h.GetVariable("nonexistent")
		assert.False(t, ok)
		assert.Empty(t, val)
	})
}

func TestProvisionHostFunctions_SetOutput(t *testing.T) {
	t.Parallel()

	h := NewProvisionHostFunctions("/tmp", nil)

	h.SetOutput("line 1")
	h.SetOutput("line 2")
	h.SetOutput("line 3")

	output := h.GetOutput()
	require.Len(t, output, 3)
	assert.Equal(t, "line 1", output[0])
	assert.Equal(t, "line 2", output[1])
	assert.Equal(t, "line 3", output[2])
}

func TestProvisionHostFunctions_GetOutput_Empty(t *testing.T) {
	t.Parallel()

	h := NewProvisionHostFunctions("/tmp", nil)

	output := h.GetOutput()
	assert.Empty(t, output)
}

func TestProvisionHostFunctions_GetWorkDir(t *testing.T) {
	t.Parallel()

	h := NewProvisionHostFunctions("/home/user/infra", nil)
	assert.Equal(t, "/home/user/infra", h.GetWorkDir())
}

func TestProvisionHostFunctions_GetOutput_ReturnsCopy(t *testing.T) {
	t.Parallel()

	h := NewProvisionHostFunctions("/tmp", nil)
	h.SetOutput("line 1")

	output1 := h.GetOutput()
	output1[0] = "mutated"

	output2 := h.GetOutput()
	assert.Equal(t, "line 1", output2[0], "GetOutput should return a copy, not the underlying slice")
}
