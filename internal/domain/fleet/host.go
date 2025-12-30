package fleet

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// HostID is a unique identifier for a host within the fleet.
type HostID string

// hostIDPattern validates host IDs: alphanumeric with hyphens and dots.
var hostIDPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]{0,62}[a-zA-Z0-9]?$`)

// NewHostID creates a new host ID, validating the format.
func NewHostID(id string) (HostID, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("host ID cannot be empty")
	}
	if !hostIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid host ID %q: must be alphanumeric with hyphens/dots, 1-64 chars", id)
	}
	return HostID(id), nil
}

// String returns the host ID as a string.
func (h HostID) String() string {
	return string(h)
}

// HostStatus represents the current status of a host.
type HostStatus string

const (
	// HostStatusUnknown indicates the host status is not known.
	HostStatusUnknown HostStatus = "unknown"
	// HostStatusOnline indicates the host is reachable.
	HostStatusOnline HostStatus = "online"
	// HostStatusOffline indicates the host is not reachable.
	HostStatusOffline HostStatus = "offline"
	// HostStatusError indicates an error connecting to the host.
	HostStatusError HostStatus = "error"
)

// SSHConfig holds SSH connection configuration for a host.
type SSHConfig struct {
	// Hostname is the SSH hostname or IP address.
	Hostname string `yaml:"hostname" json:"hostname"`
	// User is the SSH username.
	User string `yaml:"user" json:"user"`
	// Port is the SSH port (default 22).
	Port int `yaml:"port" json:"port"`
	// IdentityFile is the path to the SSH private key.
	IdentityFile string `yaml:"ssh_key" json:"ssh_key"`
	// ProxyJump is an optional jump host.
	ProxyJump string `yaml:"proxy_jump,omitempty" json:"proxy_jump,omitempty"`
	// ConnectTimeout is the connection timeout.
	ConnectTimeout time.Duration `yaml:"connect_timeout,omitempty" json:"connect_timeout,omitempty"`
}

// Validate validates the SSH configuration.
func (c SSHConfig) Validate() error {
	if c.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return nil
}

// WithDefaults returns a copy with default values applied.
func (c SSHConfig) WithDefaults() SSHConfig {
	if c.Port == 0 {
		c.Port = 22
	}
	if c.User == "" {
		c.User = "root"
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 30 * time.Second
	}
	return c
}

// Host represents a machine in the fleet.
// Host is an entity with identity (HostID).
type Host struct {
	// ID is the unique identifier for this host.
	id HostID
	// SSH holds the SSH connection configuration.
	ssh SSHConfig
	// Tags are labels for targeting.
	tags Tags
	// Groups are named groups this host belongs to.
	groups []string
	// Status is the current host status.
	status HostStatus
	// LastSeen is when the host was last successfully contacted.
	lastSeen time.Time
	// LastError is the most recent error, if any.
	lastError error
	// Metadata holds arbitrary key-value data.
	metadata map[string]string
}

// NewHost creates a new host with the given ID and SSH configuration.
func NewHost(id HostID, ssh SSHConfig) (*Host, error) {
	ssh = ssh.WithDefaults()
	if err := ssh.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SSH config: %w", err)
	}
	return &Host{
		id:       id,
		ssh:      ssh,
		tags:     Tags{},
		groups:   []string{},
		status:   HostStatusUnknown,
		metadata: make(map[string]string),
	}, nil
}

// ID returns the host's unique identifier.
func (h *Host) ID() HostID {
	return h.id
}

// SSH returns the SSH configuration.
func (h *Host) SSH() SSHConfig {
	return h.ssh
}

// Tags returns the host's tags.
func (h *Host) Tags() Tags {
	return h.tags
}

// Groups returns the groups this host belongs to.
func (h *Host) Groups() []string {
	result := make([]string, len(h.groups))
	copy(result, h.groups)
	return result
}

// Status returns the current host status.
func (h *Host) Status() HostStatus {
	return h.status
}

// LastSeen returns when the host was last successfully contacted.
func (h *Host) LastSeen() time.Time {
	return h.lastSeen
}

// LastError returns the most recent error.
func (h *Host) LastError() error {
	return h.lastError
}

// Metadata returns the host metadata.
func (h *Host) Metadata() map[string]string {
	result := make(map[string]string, len(h.metadata))
	for k, v := range h.metadata {
		result[k] = v
	}
	return result
}

// SetTags sets the host's tags.
func (h *Host) SetTags(tags Tags) {
	h.tags = tags
}

// AddTag adds a tag to the host.
func (h *Host) AddTag(tag Tag) {
	if !h.tags.Contains(tag) {
		h.tags = append(h.tags, tag)
	}
}

// SetGroups sets the groups this host belongs to.
func (h *Host) SetGroups(groups []string) {
	h.groups = make([]string, len(groups))
	copy(h.groups, groups)
}

// AddGroup adds the host to a group.
func (h *Host) AddGroup(group string) {
	for _, g := range h.groups {
		if g == group {
			return
		}
	}
	h.groups = append(h.groups, group)
}

// InGroup checks if the host is in a specific group.
func (h *Host) InGroup(group string) bool {
	for _, g := range h.groups {
		if g == group {
			return true
		}
	}
	return false
}

// SetMetadata sets a metadata key-value pair.
func (h *Host) SetMetadata(key, value string) {
	h.metadata[key] = value
}

// MarkOnline marks the host as online.
func (h *Host) MarkOnline() {
	h.status = HostStatusOnline
	h.lastSeen = time.Now()
	h.lastError = nil
}

// MarkOffline marks the host as offline.
func (h *Host) MarkOffline() {
	h.status = HostStatusOffline
}

// MarkError marks the host as having an error.
func (h *Host) MarkError(err error) {
	h.status = HostStatusError
	h.lastError = err
}

// HasTag checks if the host has a specific tag.
func (h *Host) HasTag(tag Tag) bool {
	return h.tags.Contains(tag)
}

// HasAnyTag checks if the host has any of the given tags.
func (h *Host) HasAnyTag(tags Tags) bool {
	return h.tags.ContainsAny(tags)
}

// HasAllTags checks if the host has all the given tags.
func (h *Host) HasAllTags(tags Tags) bool {
	return h.tags.ContainsAll(tags)
}

// HostSummary is a read-only summary of host state.
type HostSummary struct {
	ID       HostID            `json:"id"`
	Hostname string            `json:"hostname"`
	User     string            `json:"user"`
	Port     int               `json:"port"`
	Tags     []string          `json:"tags"`
	Groups   []string          `json:"groups"`
	Status   HostStatus        `json:"status"`
	LastSeen time.Time         `json:"last_seen,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Summary returns a read-only summary of the host.
func (h *Host) Summary() HostSummary {
	return HostSummary{
		ID:       h.id,
		Hostname: h.ssh.Hostname,
		User:     h.ssh.User,
		Port:     h.ssh.Port,
		Tags:     h.tags.Strings(),
		Groups:   h.Groups(),
		Status:   h.status,
		LastSeen: h.lastSeen,
		Metadata: h.Metadata(),
	}
}
