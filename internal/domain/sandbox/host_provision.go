package sandbox

// ProvisionHostFunctions defines the provisioner host function namespace.
// These are registered as WASM host functions for provisioner plugins.
type ProvisionHostFunctions struct {
	// WorkDir is the provisioner's working directory
	WorkDir string
	// Variables available to the provisioner
	Variables map[string]string
	// Output captures provisioner output
	Output []string
}

// NewProvisionHostFunctions creates provisioner host functions.
// If variables is nil, an empty map is initialized.
func NewProvisionHostFunctions(workDir string, variables map[string]string) *ProvisionHostFunctions {
	if variables == nil {
		variables = make(map[string]string)
	}
	return &ProvisionHostFunctions{
		WorkDir:   workDir,
		Variables: variables,
		Output:    nil,
	}
}

// GetVariable returns a variable value by name.
func (h *ProvisionHostFunctions) GetVariable(name string) (string, bool) {
	val, ok := h.Variables[name]
	return val, ok
}

// SetOutput records provisioner output.
func (h *ProvisionHostFunctions) SetOutput(line string) {
	h.Output = append(h.Output, line)
}

// GetOutput returns all captured output as a copy.
func (h *ProvisionHostFunctions) GetOutput() []string {
	if len(h.Output) == 0 {
		return nil
	}
	result := make([]string, len(h.Output))
	copy(result, h.Output)
	return result
}

// GetWorkDir returns the working directory.
func (h *ProvisionHostFunctions) GetWorkDir() string {
	return h.WorkDir
}
