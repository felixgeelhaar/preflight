package main

import (
	"fmt"

	"github.com/felixgeelhaar/preflight/internal/domain/config"
	"github.com/spf13/cobra"
)

func resolveModeOverride(cmd *cobra.Command) (*config.ReproducibilityMode, error) {
	if cmd == nil {
		return nil, nil
	}
	flag := cmd.Flags().Lookup("mode")
	if flag == nil {
		flag = cmd.InheritedFlags().Lookup("mode")
	}
	if flag == nil || !flag.Changed {
		return nil, nil
	}
	mode, err := parseModeValue(flag.Value.String())
	if err != nil {
		return nil, err
	}
	return &mode, nil
}

func parseModeValue(value string) (config.ReproducibilityMode, error) {
	switch config.ReproducibilityMode(value) {
	case config.ModeIntent, config.ModeLocked, config.ModeFrozen:
		return config.ReproducibilityMode(value), nil
	default:
		return "", fmt.Errorf("invalid mode: %s (use intent, locked, or frozen)", value)
	}
}
