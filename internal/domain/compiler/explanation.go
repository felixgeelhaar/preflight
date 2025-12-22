package compiler

// Explanation provides context for why a step exists and what it does.
// Used in the TUI explain panel to help users understand actions.
type Explanation struct {
	summary    string
	detail     string
	docLinks   []string
	tradeoffs  []string
	provenance string
}

// NewExplanation creates a new Explanation.
func NewExplanation(summary, detail string, docLinks []string) Explanation {
	links := make([]string, len(docLinks))
	copy(links, docLinks)
	return Explanation{
		summary:  summary,
		detail:   detail,
		docLinks: links,
	}
}

// Summary returns a brief description of what the step does.
func (e Explanation) Summary() string {
	return e.summary
}

// Detail returns a longer explanation with context.
func (e Explanation) Detail() string {
	return e.detail
}

// DocLinks returns links to relevant documentation.
func (e Explanation) DocLinks() []string {
	links := make([]string, len(e.docLinks))
	copy(links, e.docLinks)
	return links
}

// Tradeoffs returns the list of pros/cons for this step.
func (e Explanation) Tradeoffs() []string {
	tradeoffs := make([]string, len(e.tradeoffs))
	copy(tradeoffs, e.tradeoffs)
	return tradeoffs
}

// Provenance returns the source layer that defined this step.
func (e Explanation) Provenance() string {
	return e.provenance
}

// WithTradeoffs returns a new Explanation with tradeoffs set.
func (e Explanation) WithTradeoffs(tradeoffs []string) Explanation {
	newExp := e
	newExp.tradeoffs = make([]string, len(tradeoffs))
	copy(newExp.tradeoffs, tradeoffs)
	return newExp
}

// WithProvenance returns a new Explanation with provenance set.
func (e Explanation) WithProvenance(provenance string) Explanation {
	newExp := e
	newExp.provenance = provenance
	return newExp
}

// IsEmpty returns true if this explanation has no content.
func (e Explanation) IsEmpty() bool {
	return e.summary == "" && e.detail == ""
}
