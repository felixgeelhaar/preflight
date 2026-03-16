package fleet

// SourceRegistry manages available inventory sources.
type SourceRegistry struct {
	sources []InventorySource
}

// NewSourceRegistry creates a new source registry.
func NewSourceRegistry() *SourceRegistry {
	return &SourceRegistry{
		sources: make([]InventorySource, 0),
	}
}

// Register adds a source to the registry.
func (r *SourceRegistry) Register(s InventorySource) {
	r.sources = append(r.sources, s)
}

// Available returns all available sources.
func (r *SourceRegistry) Available() []InventorySource {
	available := make([]InventorySource, 0)
	for _, s := range r.sources {
		if s.Available() {
			available = append(available, s)
		}
	}
	return available
}

// Get returns a source by name, or nil if not found or not available.
func (r *SourceRegistry) Get(name string) InventorySource {
	for _, s := range r.sources {
		if s.Name() == name && s.Available() {
			return s
		}
	}
	return nil
}

// First returns the first available source.
func (r *SourceRegistry) First() InventorySource {
	available := r.Available()
	if len(available) == 0 {
		return nil
	}
	return available[0]
}

// Names returns the names of all registered sources.
func (r *SourceRegistry) Names() []string {
	names := make([]string, len(r.sources))
	for i, s := range r.sources {
		names[i] = s.Name()
	}
	return names
}

// AvailableNames returns the names of available sources.
func (r *SourceRegistry) AvailableNames() []string {
	available := r.Available()
	names := make([]string, len(available))
	for i, s := range available {
		names[i] = s.Name()
	}
	return names
}
