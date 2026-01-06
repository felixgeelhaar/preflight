package deps

import (
	"strings"

	"github.com/felixgeelhaar/preflight/internal/domain/compiler"
	"github.com/felixgeelhaar/preflight/internal/domain/platform"
)

// Tool identifies a runtime dependency used by package providers.
type Tool string

const (
	ToolNode   Tool = "node"
	ToolPython Tool = "python"
	ToolRuby   Tool = "ruby"
	ToolGo     Tool = "go"
	ToolRust   Tool = "rust"
)

// ResolveToolDeps returns dependency step IDs for the given tool based on config
// and platform preference. If no matching installer is configured, returns nil.
func ResolveToolDeps(ctx compiler.CompileContext, plat *platform.Platform, tool Tool) []compiler.StepID {
	dep, ok := ResolveToolDependency(ctx, plat, tool)
	if !ok {
		return nil
	}
	return []compiler.StepID{dep.StepID}
}

type ToolDependency struct {
	StepID   compiler.StepID
	Manager  string
	Explicit bool
}

type ToolBootstrap struct {
	Tool        Tool
	Manager     string
	PackageName string
	StepID      compiler.StepID
}

func ResolveToolDependency(ctx compiler.CompileContext, plat *platform.Platform, tool Tool) (ToolDependency, bool) {
	config := ctx.Config()
	if config == nil {
		return ToolDependency{}, false
	}

	plat = resolvePlatform(plat)
	managers := preferredManagers(plat)
	for _, manager := range managers {
		if dep := findToolInstaller(ctx, manager, tool); dep != "" {
			return ToolDependency{
				StepID:   compiler.MustNewStepID(dep),
				Manager:  manager,
				Explicit: true,
			}, true
		}
	}

	if len(managers) == 0 {
		return ToolDependency{}, false
	}
	implicitManagers := preferAvailableManagers(managers, plat)
	manager := implicitManagers[0]
	if _, ok := implicitToolPackage(manager, tool); !ok {
		return ToolDependency{}, false
	}

	return ToolDependency{
		StepID:   compiler.MustNewStepID(bootstrapToolStepID(tool)),
		Manager:  manager,
		Explicit: false,
	}, true
}

func ResolveToolBootstrap(ctx compiler.CompileContext, plat *platform.Platform, tool Tool) (ToolBootstrap, bool) {
	dep, ok := ResolveToolDependency(ctx, plat, tool)
	if !ok || dep.Explicit {
		return ToolBootstrap{}, false
	}

	pkg, ok := implicitToolPackage(dep.Manager, tool)
	if !ok {
		return ToolBootstrap{}, false
	}

	return ToolBootstrap{
		Tool:        tool,
		Manager:     dep.Manager,
		PackageName: pkg,
		StepID:      dep.StepID,
	}, true
}

func resolvePlatform(plat *platform.Platform) *platform.Platform {
	if plat != nil {
		return plat
	}
	p, err := platform.Detect()
	if err != nil {
		return nil
	}
	return p
}

func preferredManagers(plat *platform.Platform) []string {
	if plat == nil {
		return []string{"brew", "apt", "winget", "chocolatey", "scoop"}
	}
	switch plat.OS() {
	case platform.OSDarwin:
		return []string{"brew"}
	case platform.OSLinux:
		return []string{"apt"}
	case platform.OSWindows:
		return []string{"winget", "chocolatey", "scoop"}
	default:
		return []string{"brew", "apt", "winget", "chocolatey", "scoop"}
	}
}

func preferAvailableManagers(managers []string, plat *platform.Platform) []string {
	if plat == nil {
		return managers
	}

	available := make([]string, 0, len(managers))
	missing := make([]string, 0, len(managers))
	for _, manager := range managers {
		if managerAvailable(plat, manager) {
			available = append(available, manager)
		} else {
			missing = append(missing, manager)
		}
	}

	if len(available) == 0 {
		return managers
	}

	return append(available, missing...)
}

func managerAvailable(plat *platform.Platform, manager string) bool {
	for _, cmd := range managerCommands(plat, manager) {
		if plat.HasCommand(cmd) {
			return true
		}
	}
	return false
}

func managerCommands(plat *platform.Platform, manager string) []string {
	switch manager {
	case "brew":
		return []string{"brew"}
	case "apt":
		return []string{"apt-get"}
	case "winget":
		if plat != nil && plat.IsWSL() {
			return []string{"winget.exe", "winget"}
		}
		return []string{"winget"}
	case "chocolatey":
		if plat != nil && plat.IsWSL() {
			return []string{"choco.exe", "choco"}
		}
		return []string{"choco"}
	case "scoop":
		if plat != nil && plat.IsWSL() {
			return []string{"scoop.cmd", "scoop"}
		}
		return []string{"scoop"}
	default:
		return nil
	}
}

func bootstrapToolStepID(tool Tool) string {
	return "bootstrap:tool:" + string(tool)
}

func findToolInstaller(ctx compiler.CompileContext, manager string, tool Tool) string {
	switch manager {
	case "brew":
		return findBrewTool(ctx.GetSection("brew"), tool)
	case "apt":
		return findAptTool(ctx.GetSection("apt"), tool)
	case "chocolatey":
		return findChocoTool(ctx.GetSection("chocolatey"), tool)
	case "winget":
		return findWingetTool(ctx.GetSection("winget"), tool)
	case "scoop":
		return findScoopTool(ctx.GetSection("scoop"), tool)
	default:
		return ""
	}
}

func implicitToolPackage(manager string, tool Tool) (string, bool) {
	switch manager {
	case "brew":
		return mapToolPackage(tool, map[Tool]string{
			ToolNode:   "node",
			ToolPython: "python",
			ToolRuby:   "ruby",
			ToolGo:     "go",
			ToolRust:   "rust",
		})
	case "apt":
		return mapToolPackage(tool, map[Tool]string{
			ToolNode:   "nodejs",
			ToolPython: "python3",
			ToolRuby:   "ruby",
			ToolGo:     "golang-go",
			ToolRust:   "cargo",
		})
	case "winget":
		return mapToolPackage(tool, map[Tool]string{
			ToolNode:   "OpenJS.NodeJS.LTS",
			ToolPython: "Python.Python.3",
			ToolRuby:   "RubyInstallerTeam.Ruby",
			ToolGo:     "GoLang.Go",
			ToolRust:   "Rustlang.Rustup",
		})
	case "chocolatey":
		return mapToolPackage(tool, map[Tool]string{
			ToolNode:   "nodejs-lts",
			ToolPython: "python",
			ToolRuby:   "ruby",
			ToolGo:     "golang",
			ToolRust:   "rustup.install",
		})
	case "scoop":
		return mapToolPackage(tool, map[Tool]string{
			ToolNode:   "nodejs-lts",
			ToolPython: "python",
			ToolRuby:   "ruby",
			ToolGo:     "go",
			ToolRust:   "rustup",
		})
	default:
		return "", false
	}
}

func mapToolPackage(tool Tool, mapping map[Tool]string) (string, bool) {
	pkg, ok := mapping[tool]
	if !ok || pkg == "" {
		return "", false
	}
	return pkg, true
}

type toolMatch struct {
	names    []string
	prefixes []string
}

func findBrewTool(section map[string]interface{}, tool Tool) string {
	if section == nil {
		return ""
	}
	match := toolMatchForManager(tool, "brew")
	formulae, ok := section["formulae"].([]interface{})
	if !ok {
		return ""
	}
	for _, entry := range formulae {
		name, fullName := parseBrewFormula(entry)
		if name == "" || fullName == "" {
			continue
		}
		if matchName(name, match) {
			return "brew:formula:" + fullName
		}
	}
	return ""
}

func findAptTool(section map[string]interface{}, tool Tool) string {
	if section == nil {
		return ""
	}
	match := toolMatchForManager(tool, "apt")
	packages, ok := section["packages"].([]interface{})
	if !ok {
		return ""
	}
	for _, entry := range packages {
		name := parseNamedEntry(entry, "name")
		if name == "" {
			continue
		}
		if matchName(name, match) {
			return "apt:package:" + name
		}
	}
	return ""
}

func findChocoTool(section map[string]interface{}, tool Tool) string {
	if section == nil {
		return ""
	}
	match := toolMatchForManager(tool, "chocolatey")
	packages, ok := section["packages"].([]interface{})
	if !ok {
		return ""
	}
	for _, entry := range packages {
		name := parseNamedEntry(entry, "name")
		if name == "" {
			continue
		}
		if matchName(name, match) {
			return "chocolatey:package:" + name
		}
	}
	return ""
}

func findWingetTool(section map[string]interface{}, tool Tool) string {
	if section == nil {
		return ""
	}
	match := toolMatchForManager(tool, "winget")
	packages, ok := section["packages"].([]interface{})
	if !ok {
		return ""
	}
	for _, entry := range packages {
		id := parseNamedEntry(entry, "id")
		if id == "" {
			continue
		}
		if matchName(id, match) {
			return "winget:package:" + id
		}
	}
	return ""
}

func findScoopTool(section map[string]interface{}, tool Tool) string {
	if section == nil {
		return ""
	}
	match := toolMatchForManager(tool, "scoop")
	packages, ok := section["packages"].([]interface{})
	if !ok {
		return ""
	}
	for _, entry := range packages {
		name, fullName := parseScoopPackage(entry)
		if name == "" || fullName == "" {
			continue
		}
		if matchName(name, match) {
			return "scoop:package:" + fullName
		}
	}
	return ""
}

func parseNamedEntry(entry interface{}, key string) string {
	switch v := entry.(type) {
	case string:
		return v
	case map[string]interface{}:
		if name, ok := v[key].(string); ok {
			return name
		}
	}
	return ""
}

func parseBrewFormula(entry interface{}) (string, string) {
	switch v := entry.(type) {
	case string:
		full := v
		name := v
		if strings.Contains(name, "/") {
			parts := strings.Split(name, "/")
			name = parts[len(parts)-1]
		}
		return name, full
	case map[string]interface{}:
		name, ok := v["name"].(string)
		if !ok || name == "" {
			return "", ""
		}
		full := name
		if tap, ok := v["tap"].(string); ok && tap != "" {
			full = tap + "/" + name
		}
		return name, full
	default:
		return "", ""
	}
}

func parseScoopPackage(entry interface{}) (string, string) {
	switch v := entry.(type) {
	case string:
		return v, v
	case map[string]interface{}:
		name, ok := v["name"].(string)
		if !ok || name == "" {
			return "", ""
		}
		full := name
		if bucket, ok := v["bucket"].(string); ok && bucket != "" {
			full = bucket + "/" + name
		}
		return name, full
	default:
		return "", ""
	}
}

func matchName(value string, match toolMatch) bool {
	val := strings.ToLower(value)
	for _, name := range match.names {
		if val == name {
			return true
		}
	}
	for _, prefix := range match.prefixes {
		if strings.HasPrefix(val, prefix) {
			return true
		}
	}
	return false
}

func toolMatchForManager(tool Tool, manager string) toolMatch {
	switch manager {
	case "brew":
		return brewToolMatch(tool)
	case "apt":
		return aptToolMatch(tool)
	case "chocolatey":
		return chocoToolMatch(tool)
	case "winget":
		return wingetToolMatch(tool)
	case "scoop":
		return scoopToolMatch(tool)
	default:
		return toolMatch{}
	}
}

func brewToolMatch(tool Tool) toolMatch {
	switch tool {
	case ToolNode:
		return toolMatch{names: []string{"node"}}
	case ToolPython:
		return toolMatch{names: []string{"python"}, prefixes: []string{"python@"}}
	case ToolRuby:
		return toolMatch{names: []string{"ruby"}, prefixes: []string{"ruby@"}}
	case ToolGo:
		return toolMatch{names: []string{"go"}}
	case ToolRust:
		return toolMatch{names: []string{"rust", "rustup"}}
	default:
		return toolMatch{}
	}
}

func aptToolMatch(tool Tool) toolMatch {
	switch tool {
	case ToolNode:
		return toolMatch{names: []string{"nodejs", "npm"}}
	case ToolPython:
		return toolMatch{names: []string{"python3", "python", "python3-pip", "python-pip"}}
	case ToolRuby:
		return toolMatch{names: []string{"ruby", "ruby-full"}}
	case ToolGo:
		return toolMatch{names: []string{"golang-go", "golang"}}
	case ToolRust:
		return toolMatch{names: []string{"cargo", "rustc", "rust-all", "rustup"}}
	default:
		return toolMatch{}
	}
}

func chocoToolMatch(tool Tool) toolMatch {
	switch tool {
	case ToolNode:
		return toolMatch{names: []string{"nodejs", "nodejs-lts"}}
	case ToolPython:
		return toolMatch{names: []string{"python", "python3"}}
	case ToolRuby:
		return toolMatch{names: []string{"ruby"}}
	case ToolGo:
		return toolMatch{names: []string{"golang"}}
	case ToolRust:
		return toolMatch{names: []string{"rust", "rustup"}}
	default:
		return toolMatch{}
	}
}

func wingetToolMatch(tool Tool) toolMatch {
	switch tool {
	case ToolNode:
		return toolMatch{names: []string{"openjs.nodejs", "openjs.nodejs.lts"}}
	case ToolPython:
		return toolMatch{prefixes: []string{"python.python.3"}}
	case ToolRuby:
		return toolMatch{names: []string{"rubyinstallerteam.ruby"}}
	case ToolGo:
		return toolMatch{prefixes: []string{"golang.go"}}
	case ToolRust:
		return toolMatch{names: []string{"rustlang.rustup"}}
	default:
		return toolMatch{}
	}
}

func scoopToolMatch(tool Tool) toolMatch {
	switch tool {
	case ToolNode:
		return toolMatch{names: []string{"nodejs", "nodejs-lts"}}
	case ToolPython:
		return toolMatch{names: []string{"python", "python310", "python311"}}
	case ToolRuby:
		return toolMatch{names: []string{"ruby"}}
	case ToolGo:
		return toolMatch{names: []string{"go"}}
	case ToolRust:
		return toolMatch{names: []string{"rust", "rustup"}}
	default:
		return toolMatch{}
	}
}
