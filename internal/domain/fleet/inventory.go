package fleet

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// Inventory is the aggregate root for fleet management.
// It manages hosts and groups, providing targeting and querying capabilities.
type Inventory struct {
	mu       sync.RWMutex
	hosts    map[HostID]*Host
	groups   map[GroupName]*Group
	defaults SSHConfig
}

// NewInventory creates a new empty inventory.
func NewInventory() *Inventory {
	return &Inventory{
		hosts:  make(map[HostID]*Host),
		groups: make(map[GroupName]*Group),
		defaults: SSHConfig{
			Port:           22,
			User:           "root",
			ConnectTimeout: 30 * time.Second,
		},
	}
}

// SetDefaults sets the default SSH configuration.
func (i *Inventory) SetDefaults(defaults SSHConfig) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.defaults = defaults
}

// Defaults returns the default SSH configuration.
func (i *Inventory) Defaults() SSHConfig {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.defaults
}

// AddHost adds a host to the inventory.
func (i *Inventory) AddHost(host *Host) error {
	if host == nil {
		return fmt.Errorf("host cannot be nil")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, exists := i.hosts[host.ID()]; exists {
		return fmt.Errorf("host %q already exists", host.ID())
	}
	i.hosts[host.ID()] = host
	return nil
}

// GetHost returns a host by ID.
func (i *Inventory) GetHost(id HostID) (*Host, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	host, ok := i.hosts[id]
	return host, ok
}

// RemoveHost removes a host from the inventory.
func (i *Inventory) RemoveHost(id HostID) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, exists := i.hosts[id]; !exists {
		return false
	}
	delete(i.hosts, id)
	return true
}

// AllHosts returns all hosts in the inventory.
func (i *Inventory) AllHosts() []*Host {
	i.mu.RLock()
	defer i.mu.RUnlock()
	hosts := make([]*Host, 0, len(i.hosts))
	for _, host := range i.hosts {
		hosts = append(hosts, host)
	}
	return hosts
}

// HostCount returns the number of hosts.
func (i *Inventory) HostCount() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return len(i.hosts)
}

// AddGroup adds a group to the inventory.
func (i *Inventory) AddGroup(group *Group) error {
	if group == nil {
		return fmt.Errorf("group cannot be nil")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, exists := i.groups[group.Name()]; exists {
		return fmt.Errorf("group %q already exists", group.Name())
	}
	i.groups[group.Name()] = group
	return nil
}

// GetGroup returns a group by name.
func (i *Inventory) GetGroup(name GroupName) (*Group, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	group, ok := i.groups[name]
	return group, ok
}

// RemoveGroup removes a group from the inventory.
func (i *Inventory) RemoveGroup(name GroupName) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, exists := i.groups[name]; !exists {
		return false
	}
	delete(i.groups, name)
	return true
}

// AllGroups returns all groups in the inventory.
func (i *Inventory) AllGroups() []*Group {
	i.mu.RLock()
	defer i.mu.RUnlock()
	groups := make([]*Group, 0, len(i.groups))
	for _, group := range i.groups {
		groups = append(groups, group)
	}
	return groups
}

// GroupCount returns the number of groups.
func (i *Inventory) GroupCount() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return len(i.groups)
}

// HostsByTag returns all hosts with a specific tag.
func (i *Inventory) HostsByTag(tag Tag) []*Host {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var result []*Host
	for _, host := range i.hosts {
		if host.HasTag(tag) {
			result = append(result, host)
		}
	}
	return result
}

// HostsByTags returns all hosts with any of the given tags.
func (i *Inventory) HostsByTags(tags Tags) []*Host {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var result []*Host
	for _, host := range i.hosts {
		if host.HasAnyTag(tags) {
			result = append(result, host)
		}
	}
	return result
}

// HostsByGroup returns all hosts in a group (by direct membership or pattern matching).
func (i *Inventory) HostsByGroup(name GroupName) []*Host {
	i.mu.RLock()
	defer i.mu.RUnlock()

	group, ok := i.groups[name]
	if !ok {
		return nil
	}

	result := make([]*Host, 0)
	seen := make(map[HostID]bool)

	// Add hosts that are directly in the group
	for _, host := range i.hosts {
		if host.InGroup(string(name)) {
			if !seen[host.ID()] {
				result = append(result, host)
				seen[host.ID()] = true
			}
		}
	}

	// Add hosts that match the group's patterns
	for _, host := range i.hosts {
		if seen[host.ID()] {
			continue
		}
		for _, pattern := range group.HostPatterns() {
			matched, _ := filepath.Match(pattern, string(host.ID()))
			if matched {
				result = append(result, host)
				seen[host.ID()] = true
				break
			}
		}
	}

	return result
}

// HostsByPattern returns hosts matching a glob pattern.
func (i *Inventory) HostsByPattern(pattern string) []*Host {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var result []*Host
	for _, host := range i.hosts {
		matched, err := filepath.Match(pattern, string(host.ID()))
		if err == nil && matched {
			result = append(result, host)
		}
	}
	return result
}

// HostsByStatus returns hosts with a specific status.
func (i *Inventory) HostsByStatus(status HostStatus) []*Host {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var result []*Host
	for _, host := range i.hosts {
		if host.Status() == status {
			result = append(result, host)
		}
	}
	return result
}

// ResolveGroupPolicies resolves all policies for a group, including inherited policies.
func (i *Inventory) ResolveGroupPolicies(name GroupName) []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.resolveGroupPoliciesLocked(name, make(map[GroupName]bool))
}

func (i *Inventory) resolveGroupPoliciesLocked(name GroupName, visited map[GroupName]bool) []string {
	if visited[name] {
		return nil // Prevent cycles
	}
	visited[name] = true

	group, ok := i.groups[name]
	if !ok {
		return nil
	}

	policies := make([]string, 0)
	seen := make(map[string]bool)

	// First, collect policies from parent groups
	for _, parent := range group.Inherit() {
		for _, policy := range i.resolveGroupPoliciesLocked(parent, visited) {
			if !seen[policy] {
				policies = append(policies, policy)
				seen[policy] = true
			}
		}
	}

	// Then add this group's policies (can override inherited)
	for _, policy := range group.Policies() {
		if !seen[policy] {
			policies = append(policies, policy)
			seen[policy] = true
		}
	}

	return policies
}

// InventorySummary is a read-only summary of the inventory.
type InventorySummary struct {
	HostCount    int            `json:"host_count"`
	GroupCount   int            `json:"group_count"`
	OnlineCount  int            `json:"online_count"`
	OfflineCount int            `json:"offline_count"`
	ErrorCount   int            `json:"error_count"`
	TagCounts    map[string]int `json:"tag_counts"`
}

// Summary returns a summary of the inventory.
func (i *Inventory) Summary() InventorySummary {
	i.mu.RLock()
	defer i.mu.RUnlock()

	summary := InventorySummary{
		HostCount:  len(i.hosts),
		GroupCount: len(i.groups),
		TagCounts:  make(map[string]int),
	}

	for _, host := range i.hosts {
		switch host.Status() {
		case HostStatusOnline:
			summary.OnlineCount++
		case HostStatusOffline:
			summary.OfflineCount++
		case HostStatusError:
			summary.ErrorCount++
		case HostStatusUnknown:
			// Unknown status hosts are not counted in any category
		}

		for _, tag := range host.Tags() {
			summary.TagCounts[tag.String()]++
		}
	}

	return summary
}
