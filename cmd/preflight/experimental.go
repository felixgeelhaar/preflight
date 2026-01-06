package main

import (
	"fmt"
	"os"
)

const experimentalEnvVar = "PREFLIGHT_EXPERIMENTAL"

func requireExperimental(feature string) error {
	if os.Getenv(experimentalEnvVar) == "1" {
		return nil
	}
	return fmt.Errorf("%s is experimental; set %s=1 to enable", feature, experimentalEnvVar)
}
