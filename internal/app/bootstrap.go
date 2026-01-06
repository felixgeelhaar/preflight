package app

import (
	"sort"
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/execution"
)

var bootstrapStepIDs = map[string]struct{}{
	"brew:install":       {},
	"chocolatey:install": {},
	"scoop:install":      {},
	"apt:update":         {},
	"winget:ready":       {},
}

// BootstrapSteps returns bootstrap step IDs that need to apply.
func BootstrapSteps(plan *execution.Plan) []string {
	if plan == nil {
		return nil
	}

	steps := make([]string, 0)
	for _, entry := range plan.Entries() {
		if entry.Status() != compiler.StatusNeedsApply {
			continue
		}
		id := entry.Step().ID().String()
		if IsBootstrapStep(id) {
			steps = append(steps, id)
		}
	}
	sort.Strings(steps)
	return steps
}

// RequiresBootstrapConfirmation reports whether plan includes bootstrap steps needing apply.
func RequiresBootstrapConfirmation(plan *execution.Plan) bool {
	return len(BootstrapSteps(plan)) > 0
}

// IsBootstrapStep reports whether a step ID is considered a bootstrap operation.
func IsBootstrapStep(id string) bool {
	if _, ok := bootstrapStepIDs[id]; ok {
		return true
	}
	return strings.HasPrefix(id, "bootstrap:")
}
