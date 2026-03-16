package fleet

import "context"

// InventorySource discovers hosts from external sources (cloud APIs, etc).
type InventorySource interface {
	// Name returns the source name (e.g., "aws", "azure", "gcp").
	Name() string
	// Discover queries the source and returns discovered hosts.
	Discover(ctx context.Context) ([]*Host, error)
	// Available returns true if the source is properly configured and accessible.
	Available() bool
}
