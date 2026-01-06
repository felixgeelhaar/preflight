package main

import (
	"fmt"
	"strings"
)

func confirmBootstrap(steps []string) bool {
	if len(steps) == 0 {
		return true
	}
	if yesFlag {
		return true
	}
	if allowBootstrapFlag {
		return true
	}
	fmt.Println("Bootstrap steps require confirmation:")
	for _, step := range steps {
		fmt.Printf("  - %s\n", step)
	}
	fmt.Print("Proceed with bootstrapping? [y/N]: ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
