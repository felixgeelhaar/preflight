package compiler

import (
	"testing"
)

func TestExplanation_Creation(t *testing.T) {
	exp := NewExplanation(
		"Installing git for version control",
		"Git is the standard version control system used in software development.",
		[]string{"https://git-scm.com/doc"},
	)

	if exp.Summary() != "Installing git for version control" {
		t.Errorf("Summary() = %q, want %q", exp.Summary(), "Installing git for version control")
	}
	if exp.Detail() != "Git is the standard version control system used in software development." {
		t.Errorf("Detail() = %q, want %q", exp.Detail(), "Git is the standard version control system used in software development.")
	}
	if len(exp.DocLinks()) != 1 {
		t.Fatalf("DocLinks() len = %d, want 1", len(exp.DocLinks()))
	}
	if exp.DocLinks()[0] != "https://git-scm.com/doc" {
		t.Errorf("DocLinks()[0] = %q, want %q", exp.DocLinks()[0], "https://git-scm.com/doc")
	}
}

func TestExplanation_WithTradeoffs(t *testing.T) {
	exp := NewExplanation("Summary", "Detail", nil).
		WithTradeoffs([]string{"Pro: widely adopted", "Con: steep learning curve"})

	tradeoffs := exp.Tradeoffs()
	if len(tradeoffs) != 2 {
		t.Fatalf("Tradeoffs() len = %d, want 2", len(tradeoffs))
	}
	if tradeoffs[0] != "Pro: widely adopted" {
		t.Errorf("Tradeoffs()[0] = %q, want %q", tradeoffs[0], "Pro: widely adopted")
	}
}

func TestExplanation_WithProvenance(t *testing.T) {
	exp := NewExplanation("Summary", "Detail", nil).
		WithProvenance("layers/base.yaml")

	if exp.Provenance() != "layers/base.yaml" {
		t.Errorf("Provenance() = %q, want %q", exp.Provenance(), "layers/base.yaml")
	}
}

func TestExplanation_Empty(t *testing.T) {
	exp := NewExplanation("", "", nil)
	if !exp.IsEmpty() {
		t.Error("expected empty explanation to return true for IsEmpty()")
	}

	exp2 := NewExplanation("Has summary", "", nil)
	if exp2.IsEmpty() {
		t.Error("expected non-empty explanation to return false for IsEmpty()")
	}
}

func TestExplanation_Immutability(t *testing.T) {
	// Verify that WithTradeoffs and WithProvenance return new instances
	original := NewExplanation("Summary", "Detail", nil)
	modified := original.WithProvenance("layer.yaml")

	if original.Provenance() != "" {
		t.Error("original should not be modified")
	}
	if modified.Provenance() != "layer.yaml" {
		t.Error("modified should have provenance set")
	}
}
