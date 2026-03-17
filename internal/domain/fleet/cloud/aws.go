//go:build !noaws

// Package cloud provides cloud-native inventory sources for discovering
// fleet hosts from cloud provider APIs (AWS, Azure, GCP).
package cloud

import (
	"context"
	"fmt"
	"sort"

	"github.com/felixgeelhaar/preflight/internal/domain/fleet"
)

// AWSClient is an interface for AWS EC2 API calls (for testability).
type AWSClient interface {
	DescribeInstances(ctx context.Context, filters map[string]string) ([]EC2Instance, error)
}

// EC2Instance represents a discovered EC2 instance.
type EC2Instance struct {
	InstanceID       string
	PrivateIP        string
	PublicIP         string
	State            string // running, stopped, etc.
	Tags             map[string]string
	KeyName          string
	AvailabilityZone string
	InstanceType     string
}

// AWSOption configures an AWSSource.
type AWSOption func(*AWSSource)

// WithFilters sets the EC2 describe filters.
func WithFilters(filters map[string]string) AWSOption {
	return func(s *AWSSource) {
		s.filters = filters
	}
}

// WithSSHUser sets the default SSH user for discovered hosts.
func WithSSHUser(user string) AWSOption {
	return func(s *AWSSource) {
		s.sshUser = user
	}
}

// WithSSHKey sets the default SSH key path for discovered hosts.
func WithSSHKey(key string) AWSOption {
	return func(s *AWSSource) {
		s.sshKey = key
	}
}

// AWSSource discovers hosts from AWS EC2.
type AWSSource struct {
	client  AWSClient
	region  string
	filters map[string]string
	sshUser string
	sshKey  string
}

// Compile-time interface check.
var _ fleet.InventorySource = (*AWSSource)(nil)

// NewAWSSource creates a new AWS inventory source.
func NewAWSSource(client AWSClient, region string, opts ...AWSOption) *AWSSource {
	s := &AWSSource{
		client:  client,
		region:  region,
		filters: make(map[string]string),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the source name.
func (s *AWSSource) Name() string {
	return "aws"
}

// Available returns true if the AWS client is configured.
func (s *AWSSource) Available() bool {
	return s.client != nil
}

// Discover queries AWS EC2 and returns discovered hosts.
func (s *AWSSource) Discover(ctx context.Context) ([]*fleet.Host, error) {
	instances, err := s.client.DescribeInstances(ctx, s.filters)
	if err != nil {
		return nil, fmt.Errorf("AWS EC2 describe instances: %w", err)
	}

	hosts := make([]*fleet.Host, 0, len(instances))
	for _, inst := range instances {
		if inst.State != "running" {
			continue
		}
		if inst.PrivateIP == "" {
			continue
		}

		host, err := s.instanceToHost(inst)
		if err != nil {
			continue
		}
		hosts = append(hosts, host)
	}

	return hosts, nil
}

func (s *AWSSource) instanceToHost(inst EC2Instance) (*fleet.Host, error) {
	hostID, err := fleet.NewHostID(inst.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid instance ID %q: %w", inst.InstanceID, err)
	}

	sshCfg := fleet.SSHConfig{
		Hostname:     inst.PrivateIP,
		User:         s.sshUser,
		IdentityFile: s.sshKey,
	}

	host, err := fleet.NewHost(hostID, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create host for %q: %w", inst.InstanceID, err)
	}

	// Add to aws group.
	host.AddGroup("aws")

	// Map EC2 tags to fleet tags. Sort keys for deterministic output.
	keys := make([]string, 0, len(inst.Tags))
	for k := range inst.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// Skip the Name tag — it is not a meaningful fleet tag.
		if k == "Name" {
			continue
		}
		tag, err := fleet.NewTag(k)
		if err != nil {
			// Skip tags that do not conform to fleet naming rules.
			continue
		}
		host.AddTag(tag)
	}

	// Set metadata for cloud-specific attributes.
	host.SetMetadata("aws:region", s.region)
	if inst.AvailabilityZone != "" {
		host.SetMetadata("aws:az", inst.AvailabilityZone)
	}
	if inst.InstanceType != "" {
		host.SetMetadata("aws:instance-type", inst.InstanceType)
	}
	if inst.PublicIP != "" {
		host.SetMetadata("aws:public-ip", inst.PublicIP)
	}
	if inst.KeyName != "" {
		host.SetMetadata("aws:key-name", inst.KeyName)
	}

	return host, nil
}
