package config

// Target is a resolved configuration for a specific target.
// It contains the ordered list of layers with provenance tracking.
type Target struct {
	Name   TargetName
	Layers []Layer
}
