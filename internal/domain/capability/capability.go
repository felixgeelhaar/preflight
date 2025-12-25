// Package capability provides capability-based permission management for plugins.
package capability

import (
	"errors"
	"fmt"
	"strings"
)

// Capability errors.
var (
	ErrInvalidCapability    = errors.New("invalid capability")
	ErrCapabilityDenied     = errors.New("capability denied")
	ErrCapabilityNotGranted = errors.New("capability not granted")
	ErrDangerousCapability  = errors.New("dangerous capability requires approval")
)

// Category represents a capability category.
type Category string

// Category constants.
const (
	CategoryFiles    Category = "files"
	CategoryPackages Category = "packages"
	CategoryShell    Category = "shell"
	CategoryNetwork  Category = "network"
	CategorySecrets  Category = "secrets"
	CategorySystem   Category = "system"
)

// Action represents a capability action within a category.
type Action string

// Action constants.
const (
	ActionRead    Action = "read"
	ActionWrite   Action = "write"
	ActionExecute Action = "execute"
	ActionFetch   Action = "fetch"
	ActionInstall Action = "install"
	ActionRemove  Action = "remove"
)

// Capability represents a single permission capability.
// Format: "category:action" (e.g., "files:read", "packages:brew")
type Capability struct {
	category Category
	action   Action
	raw      string
}

// NewCapability creates a capability from category and action.
func NewCapability(category Category, action Action) Capability {
	return Capability{
		category: category,
		action:   action,
		raw:      string(category) + ":" + string(action),
	}
}

// ParseCapability parses a capability string.
func ParseCapability(s string) (Capability, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Capability{}, fmt.Errorf("%w: empty capability", ErrInvalidCapability)
	}

	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return Capability{}, fmt.Errorf("%w: must be category:action format", ErrInvalidCapability)
	}

	category := Category(parts[0])
	action := Action(parts[1])

	if !isValidCategory(category) {
		return Capability{}, fmt.Errorf("%w: unknown category %q", ErrInvalidCapability, category)
	}

	return Capability{
		category: category,
		action:   action,
		raw:      s,
	}, nil
}

// MustParseCapability parses a capability or panics.
func MustParseCapability(s string) Capability {
	c, err := ParseCapability(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Category returns the capability category.
func (c Capability) Category() Category {
	return c.category
}

// Action returns the capability action.
func (c Capability) Action() Action {
	return c.action
}

// String returns the string representation.
func (c Capability) String() string {
	return c.raw
}

// IsZero returns true if the capability is empty.
func (c Capability) IsZero() bool {
	return c.raw == ""
}

// IsDangerous returns true if this capability is considered dangerous.
func (c Capability) IsDangerous() bool {
	return isDangerousCapability(c)
}

// Matches checks if this capability matches another.
// Supports wildcards: "files:*" matches any files capability.
func (c Capability) Matches(other Capability) bool {
	if c.category != other.category {
		return false
	}
	if c.action == "*" || other.action == "*" {
		return true
	}
	return c.action == other.action
}

// Well-known capabilities.
var (
	CapFilesRead      = NewCapability(CategoryFiles, ActionRead)
	CapFilesWrite     = NewCapability(CategoryFiles, ActionWrite)
	CapPackagesBrew   = NewCapability(CategoryPackages, "brew")
	CapPackagesApt    = NewCapability(CategoryPackages, "apt")
	CapPackagesWinget = NewCapability(CategoryPackages, "winget")
	CapPackagesScoop  = NewCapability(CategoryPackages, "scoop")
	CapPackagesChoco  = NewCapability(CategoryPackages, "chocolatey")
	CapShellExecute   = NewCapability(CategoryShell, ActionExecute)
	CapNetworkFetch   = NewCapability(CategoryNetwork, ActionFetch)
	CapSecretsRead    = NewCapability(CategorySecrets, ActionRead)
	CapSecretsWrite   = NewCapability(CategorySecrets, ActionWrite)
	CapSystemModify   = NewCapability(CategorySystem, "modify")
)

// DangerousCapabilities lists capabilities that require explicit approval.
var DangerousCapabilities = []Capability{
	CapShellExecute,
	CapSecretsRead,
	CapSecretsWrite,
	CapSystemModify,
}

func isValidCategory(c Category) bool {
	switch c {
	case CategoryFiles, CategoryPackages, CategoryShell,
		CategoryNetwork, CategorySecrets, CategorySystem:
		return true
	default:
		return false
	}
}

func isDangerousCapability(c Capability) bool {
	for _, dangerous := range DangerousCapabilities {
		if c.category == dangerous.category && c.action == dangerous.action {
			return true
		}
	}
	return false
}

// Info provides metadata about a capability.
type Info struct {
	Capability  Capability
	Description string
	Dangerous   bool
	Examples    []string
}

// AllCapabilities returns info about all known capabilities.
func AllCapabilities() []Info {
	return []Info{
		{CapFilesRead, "Read dotfiles and configuration", false, []string{"Read ~/.gitconfig", "Read ~/.zshrc"}},
		{CapFilesWrite, "Write dotfiles and configuration", false, []string{"Write ~/.gitconfig", "Create symlinks"}},
		{CapPackagesBrew, "Install Homebrew packages", false, []string{"brew install ripgrep", "brew tap user/repo"}},
		{CapPackagesApt, "Install APT packages", false, []string{"apt install git", "add-apt-repository"}},
		{CapPackagesWinget, "Install Windows packages via winget", false, []string{"winget install Git.Git"}},
		{CapPackagesScoop, "Install Scoop packages", false, []string{"scoop install git"}},
		{CapPackagesChoco, "Install Chocolatey packages", false, []string{"choco install git"}},
		{CapShellExecute, "Execute shell commands", true, []string{"Run arbitrary scripts", "Execute setup commands"}},
		{CapNetworkFetch, "Fetch resources from network", false, []string{"Download files", "Clone repositories"}},
		{CapSecretsRead, "Read secrets (SSH keys, tokens)", true, []string{"Read ~/.ssh/id_ed25519", "Access credentials"}},
		{CapSecretsWrite, "Write secrets", true, []string{"Generate SSH keys", "Store credentials"}},
		{CapSystemModify, "Modify system configuration", true, []string{"Change system settings", "Modify /etc files"}},
	}
}

// DescribeCapability returns a human-readable description.
func DescribeCapability(c Capability) string {
	for _, info := range AllCapabilities() {
		if info.Capability.String() == c.String() {
			return info.Description
		}
	}
	return fmt.Sprintf("Access to %s:%s", c.Category(), c.Action())
}
