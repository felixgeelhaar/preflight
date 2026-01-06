package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireExperimental_Enabled(t *testing.T) {
	t.Setenv(experimentalEnvVar, "1")
	assert.NoError(t, requireExperimental("feature"))
}

func TestRequireExperimental_Disabled(t *testing.T) {
	t.Setenv(experimentalEnvVar, "")
	err := requireExperimental("feature")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "experimental")
}
