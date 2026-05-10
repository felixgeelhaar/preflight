package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/felixgeelhaar/preflight/internal/ports"
)

// StatefulCommandRunner is a test double that models commands as state
// transitions. Unlike CommandRunner (which returns the same pre-registered
// result for every invocation of a given command line), this runner supports:
//
//   - Listed packages: a "package manager" backing set; commands like
//     `brew list --formula` and `apt list --installed` read from it,
//     while `brew install foo` / `apt install foo` insert into it.
//   - Programmable handlers: AddHandler lets a test register a function
//     that gets called for matching commands and may mutate state.
//
// Use this for idempotency contract tests where Apply followed by Check must
// transition the system from "needs apply" to "satisfied" and stay there.
type StatefulCommandRunner struct {
	mu       sync.Mutex
	packages map[string]map[string]struct{} // manager -> set of installed package names
	taps     map[string]struct{}            // brew taps that exist
	handlers []handler
	calls    []ports.CommandCall
}

type handler struct {
	command string
	match   func(args []string) bool
	fn      func(args []string) (ports.CommandResult, error)
}

// NewStatefulCommandRunner returns an empty stateful runner.
func NewStatefulCommandRunner() *StatefulCommandRunner {
	return &StatefulCommandRunner{
		packages: make(map[string]map[string]struct{}),
		taps:     make(map[string]struct{}),
	}
}

// SeedInstalled marks a package as already installed in the named manager.
// Subsequent `brew list` / `apt list` calls will include it without an
// install command being invoked.
func (s *StatefulCommandRunner) SeedInstalled(manager, pkg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.packages[manager] == nil {
		s.packages[manager] = make(map[string]struct{})
	}
	s.packages[manager][pkg] = struct{}{}
}

// AddHandler registers a callback for commands matching command+match. The
// most recently added handler that matches wins (LIFO).
func (s *StatefulCommandRunner) AddHandler(command string, match func(args []string) bool, fn func(args []string) (ports.CommandResult, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler{command: command, match: match, fn: fn})
}

// Calls returns a copy of recorded command invocations.
func (s *StatefulCommandRunner) Calls() []ports.CommandCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ports.CommandCall, len(s.calls))
	copy(out, s.calls)
	return out
}

// Run dispatches the command through built-in package-manager simulation
// first, then registered handlers, then a default success-with-empty-output
// fallback.
func (s *StatefulCommandRunner) Run(_ context.Context, command string, args ...string) (ports.CommandResult, error) {
	s.mu.Lock()
	s.calls = append(s.calls, ports.CommandCall{Command: command, Args: append([]string(nil), args...)})

	// Built-in: npm.
	if command == "npm" && len(args) > 0 {
		switch args[0] {
		case "list":
			// `npm list -g --depth=0 --json` -> JSON with dependencies map.
			s.mu.Unlock()
			return ports.CommandResult{Stdout: s.npmListJSON(), ExitCode: 0}, nil
		case "install":
			// `npm install -g <name>` or `<name>@<version>`.
			for _, a := range args[1:] {
				if strings.HasPrefix(a, "-") {
					continue
				}
				name := a
				if idx := strings.LastIndex(a, "@"); idx > 0 {
					name = a[:idx]
				}
				s.markInstalled("npm", name)
			}
			s.mu.Unlock()
			return ports.CommandResult{ExitCode: 0}, nil
		}
	}

	// Built-in: cargo.
	if command == "cargo" && len(args) > 0 && args[0] == "install" {
		// `cargo install --list` -> "<name> v<version>:" lines.
		if hasFlag(args, "--list") {
			s.mu.Unlock()
			return ports.CommandResult{Stdout: s.cargoListOutput(), ExitCode: 0}, nil
		}
		// `cargo install <name>` -> mark installed.
		for _, a := range args[1:] {
			if strings.HasPrefix(a, "-") {
				continue
			}
			s.markInstalled("cargo", a)
		}
		s.mu.Unlock()
		return ports.CommandResult{ExitCode: 0}, nil
	}

	// Built-in: gem.
	if command == "gem" && len(args) > 0 {
		switch args[0] {
		case "list":
			// `gem list -i <name>`: exit 0 if installed, 1 otherwise.
			// `gem list <name> --exact`: stdout contains name when installed.
			var pkg string
			for _, a := range args[1:] {
				if !strings.HasPrefix(a, "-") {
					pkg = a
					break
				}
			}
			if pkg != "" && s.isInstalled("gem", pkg) {
				if hasFlag(args, "-i") {
					s.mu.Unlock()
					return ports.CommandResult{Stdout: "true\n", ExitCode: 0}, nil
				}
				s.mu.Unlock()
				return ports.CommandResult{Stdout: pkg + " (1.0)\n", ExitCode: 0}, nil
			}
			if hasFlag(args, "-i") {
				s.mu.Unlock()
				return ports.CommandResult{Stdout: "false\n", ExitCode: 1}, nil
			}
			s.mu.Unlock()
			return ports.CommandResult{Stdout: "", ExitCode: 0}, nil
		case "install":
			// `gem install <name>` or `gem install --user-install <name>`.
			for _, a := range args[1:] {
				if !strings.HasPrefix(a, "-") {
					s.markInstalled("gem", a)
					break
				}
			}
			s.mu.Unlock()
			return ports.CommandResult{ExitCode: 0}, nil
		}
	}

	// Built-in: pip / pip3.
	if command == "pip" || command == "pip3" {
		if len(args) > 0 {
			switch args[0] {
			case "show":
				// `pip show <name>`: exit 0 with stdout if installed.
				if len(args) >= 2 && s.isInstalled("pip", args[1]) {
					s.mu.Unlock()
					return ports.CommandResult{Stdout: "Name: " + args[1] + "\nVersion: 1.0\n", ExitCode: 0}, nil
				}
				s.mu.Unlock()
				return ports.CommandResult{ExitCode: 1, Stderr: "Package not found"}, nil
			case "install":
				for _, a := range args[1:] {
					if strings.HasPrefix(a, "-") {
						continue
					}
					if idx := strings.IndexAny(a, "=<>"); idx > 0 {
						a = a[:idx]
					}
					s.markInstalled("pip", a)
				}
				s.mu.Unlock()
				return ports.CommandResult{ExitCode: 0}, nil
			}
		}
	}

	// Built-in: brew.
	if command == "brew" && len(args) > 0 {
		switch args[0] {
		case "list":
			result := s.listPackages("brew")
			s.mu.Unlock()
			return ports.CommandResult{Stdout: result, ExitCode: 0}, nil
		case "install":
			if len(args) >= 2 {
				s.markInstalled("brew", args[1])
			}
			s.mu.Unlock()
			return ports.CommandResult{ExitCode: 0}, nil
		case "tap":
			if len(args) == 1 {
				// `brew tap` lists configured taps
				out := s.listTaps()
				s.mu.Unlock()
				return ports.CommandResult{Stdout: out, ExitCode: 0}, nil
			}
			s.taps[args[1]] = struct{}{}
			s.mu.Unlock()
			return ports.CommandResult{ExitCode: 0}, nil
		}
	}

	// Built-in: apt-get / apt / dpkg.
	if command == "dpkg-query" {
		// apt provider asks: dpkg-query -W -f=...${db:Status-Status}\n <pkg>
		// We need stdout to include the literal "installed" token if installed.
		// Find the package argument (last positional, not starting with '-').
		var pkg string
		for _, a := range args {
			if !strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "=") {
				pkg = a
			}
		}
		if pkg == "" {
			s.mu.Unlock()
			return ports.CommandResult{ExitCode: 1}, nil
		}
		if s.isInstalled("apt", pkg) {
			s.mu.Unlock()
			return ports.CommandResult{Stdout: pkg + "\t1.0\tinstalled\n", ExitCode: 0}, nil
		}
		s.mu.Unlock()
		return ports.CommandResult{ExitCode: 1, Stderr: "no packages found"}, nil
	}
	// `sudo apt-get install -y curl` or `apt-get install curl=1.0`
	if (command == "apt-get" || command == "apt") ||
		(command == "sudo" && len(args) > 0 && (args[0] == "apt-get" || args[0] == "apt")) {
		startIdx := 0
		if command == "sudo" {
			startIdx = 1
		}
		if startIdx < len(args) && args[startIdx] == "install" {
			for i := startIdx + 1; i < len(args); i++ {
				a := args[i]
				if strings.HasPrefix(a, "-") {
					continue
				}
				// Strip optional =version suffix.
				if idx := strings.Index(a, "="); idx > 0 {
					a = a[:idx]
				}
				s.markInstalled("apt", a)
			}
			s.mu.Unlock()
			return ports.CommandResult{ExitCode: 0}, nil
		}
		// `apt-get update` etc. — succeed silently.
		s.mu.Unlock()
		return ports.CommandResult{ExitCode: 0}, nil
	}

	// Programmable handlers (LIFO).
	for i := len(s.handlers) - 1; i >= 0; i-- {
		h := s.handlers[i]
		if h.command == command && (h.match == nil || h.match(args)) {
			fn := h.fn
			s.mu.Unlock()
			return fn(args)
		}
	}

	// Default: success with empty output. Steps that require a specific
	// stderr/exitcode should register a handler.
	s.mu.Unlock()
	return ports.CommandResult{ExitCode: 0}, nil
}

func (s *StatefulCommandRunner) markInstalled(manager, pkg string) {
	if s.packages[manager] == nil {
		s.packages[manager] = make(map[string]struct{})
	}
	s.packages[manager][pkg] = struct{}{}
}

func (s *StatefulCommandRunner) isInstalled(manager, pkg string) bool {
	set, ok := s.packages[manager]
	if !ok {
		return false
	}
	_, installed := set[pkg]
	return installed
}

func (s *StatefulCommandRunner) listPackages(manager string) string {
	var b strings.Builder
	for pkg := range s.packages[manager] {
		fmt.Fprintln(&b, pkg)
	}
	return b.String()
}

func (s *StatefulCommandRunner) listTaps() string {
	var b strings.Builder
	for tap := range s.taps {
		fmt.Fprintln(&b, tap)
	}
	return b.String()
}

// npmListJSON returns a JSON document of the shape
//
//	{"dependencies":{"<name>":{"version":"1.0"}, ...}}
//
// matching what `npm list -g --depth=0 --json` produces.
func (s *StatefulCommandRunner) npmListJSON() string {
	if len(s.packages["npm"]) == 0 {
		return `{"dependencies":{}}`
	}
	var b strings.Builder
	b.WriteString(`{"dependencies":{`)
	first := true
	for pkg := range s.packages["npm"] {
		if !first {
			b.WriteString(",")
		}
		first = false
		fmt.Fprintf(&b, `%q:{"version":"1.0"}`, pkg)
	}
	b.WriteString(`}}`)
	return b.String()
}

// cargoListOutput returns the text format of `cargo install --list`:
//
//	<name> v<version>:
//	    <binary>
func (s *StatefulCommandRunner) cargoListOutput() string {
	var b strings.Builder
	for pkg := range s.packages["cargo"] {
		fmt.Fprintf(&b, "%s v1.0.0:\n    %s\n", pkg, pkg)
	}
	return b.String()
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// Compile-time interface check.
var _ ports.CommandRunner = (*StatefulCommandRunner)(nil)
