package plugin

import (
	"fmt"
	"sort"
)

// TypeProvisioner is a WASM-based provisioner plugin.
const TypeProvisioner PluginType = "provisioner"

// ProvisionerCapability represents a provisioner-specific capability.
type ProvisionerCapability struct {
	// Name is the provisioner name (e.g., "terraform", "ansible")
	Name string `yaml:"name"`
	// Description explains what this provisioner does
	Description string `yaml:"description,omitempty"`
	// SupportedActions lists supported actions (plan, apply, destroy, state)
	SupportedActions []string `yaml:"supportedActions"`
}

// ProvisionAction represents an action a provisioner can perform.
type ProvisionAction string

const (
	// ProvisionActionPlan previews changes without applying.
	ProvisionActionPlan ProvisionAction = "plan"
	// ProvisionActionApply applies infrastructure changes.
	ProvisionActionApply ProvisionAction = "apply"
	// ProvisionActionDestroy tears down infrastructure.
	ProvisionActionDestroy ProvisionAction = "destroy"
	// ProvisionActionState inspects current infrastructure state.
	ProvisionActionState ProvisionAction = "state"
)

// validProvisionActions is the set of valid provision actions for fast lookup.
var validProvisionActions = map[ProvisionAction]struct{}{
	ProvisionActionPlan:    {},
	ProvisionActionApply:   {},
	ProvisionActionDestroy: {},
	ProvisionActionState:   {},
}

// ValidProvisionActions returns the list of valid provision actions, sorted for deterministic output.
func ValidProvisionActions() []ProvisionAction {
	actions := make([]ProvisionAction, 0, len(validProvisionActions))
	for a := range validProvisionActions {
		actions = append(actions, a)
	}
	sort.Slice(actions, func(i, j int) bool {
		return actions[i] < actions[j]
	})
	return actions
}

// IsValidProvisionAction checks if an action string is valid.
func IsValidProvisionAction(action string) bool {
	_, ok := validProvisionActions[ProvisionAction(action)]
	return ok
}

// ProvisionRequest represents a request to execute a provisioner action.
type ProvisionRequest struct {
	// PluginName is the provisioner plugin name
	PluginName string
	// Action is the action to perform
	Action ProvisionAction
	// WorkDir is the working directory for the provisioner
	WorkDir string
	// Variables are key-value pairs passed to the provisioner
	Variables map[string]string
	// DryRun if true, only shows what would happen
	DryRun bool
}

// ProvisionResult represents the outcome of a provisioner action.
type ProvisionResult struct {
	// Action that was performed
	Action ProvisionAction
	// Success indicates if the action succeeded
	Success bool
	// Output is the provisioner output
	Output string
	// Changes lists what was changed (for plan/apply)
	Changes []ProvisionChange
	// Error message if the action failed
	Error string
}

// ProvisionChange describes a single infrastructure change.
type ProvisionChange struct {
	// Resource is the resource identifier
	Resource string
	// Action is create, update, delete, or no-op
	Action string
	// Before is the state before (for updates)
	Before string
	// After is the state after (for updates/creates)
	After string
}

// ValidateProvisionRequest validates a provision request.
func ValidateProvisionRequest(req *ProvisionRequest) error {
	if req == nil {
		return fmt.Errorf("provision request cannot be nil")
	}

	ve := &ValidationError{}

	if req.PluginName == "" {
		ve.Add("plugin name is required")
	}

	if req.Action == "" {
		ve.Add("action is required")
	} else if !IsValidProvisionAction(string(req.Action)) {
		ve.Addf("invalid provision action: %q (valid actions: plan, apply, destroy, state)", req.Action)
	}

	if req.WorkDir == "" {
		ve.Add("work directory is required")
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// validateProvisionerManifest validates a provisioner (WASM) plugin manifest.
func validateProvisionerManifest(m *Manifest) error {
	ve := &ValidationError{}

	// Provisioner plugins must have WASM config
	if m.WASM == nil {
		ve.Add("provisioner plugin requires 'wasm' configuration. Example:\n" +
			"  wasm:\n" +
			"    module: plugin.wasm\n" +
			"    checksum: sha256:abc123...")
		return ve
	}

	if m.WASM.Module == "" {
		ve.Add("wasm.module is required (path to WASM file). Example: module: plugin.wasm")
	}

	if m.WASM.Checksum == "" {
		ve.Add("wasm.checksum is required (SHA256 hash of WASM file). Example: checksum: sha256:abc123...")
	}

	// Validate WASM capabilities
	for i, c := range m.WASM.Capabilities {
		if c.Name == "" {
			ve.Addf("wasm.capabilities[%d].name is required", i)
		}
		if c.Justification == "" && !c.Optional {
			ve.Addf("wasm.capabilities[%d].justification is required for capability %q", i, c.Name)
		}
	}

	// Provisioner plugins must define at least one provisioner capability
	if len(m.Provides.Provisioners) == 0 {
		ve.Add("provisioner plugin must define at least one provisioner in 'provides.provisioners'. Example:\n" +
			"  provides:\n" +
			"    provisioners:\n" +
			"      - name: terraform\n" +
			"        supportedActions: [plan, apply, destroy, state]")
	}

	for i, p := range m.Provides.Provisioners {
		if p.Name == "" {
			ve.Addf("provides.provisioners[%d].name is required", i)
		}
		if len(p.SupportedActions) == 0 {
			ve.Addf("provides.provisioners[%d].supportedActions is required (at least one action)", i)
		}
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}
