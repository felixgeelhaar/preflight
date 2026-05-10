package advisor

import "testing"

func mustRec(t *testing.T, presetID string) Recommendation {
	t.Helper()
	r, err := NewRecommendation(presetID, "rationale", ConfidenceMedium)
	if err != nil {
		t.Fatalf("NewRecommendation(%q): %v", presetID, err)
	}
	return r
}

func TestFilterRecommendations_PartitionsKnownAndUnknown(t *testing.T) {
	t.Parallel()

	catalog := []string{"shell:minimal", "nvim:kickstart", "git:minimal"}
	recs := []Recommendation{
		mustRec(t, "shell:minimal"),
		mustRec(t, "made-up:thing"),
		mustRec(t, "NVIM:KICKSTART"), // case-insensitive match
	}

	known, unknown := FilterRecommendations(recs, catalog)
	if len(known) != 2 {
		t.Fatalf("known len = %d, want 2", len(known))
	}
	if len(unknown) != 1 {
		t.Fatalf("unknown len = %d, want 1", len(unknown))
	}
	if unknown[0].PresetID() != "made-up:thing" {
		t.Errorf("unexpected unknown preset: %q", unknown[0].PresetID())
	}
}

func TestFilterRecommendations_EmptyInputs(t *testing.T) {
	t.Parallel()
	known, unknown := FilterRecommendations(nil, []string{"shell:minimal"})
	if known != nil || unknown != nil {
		t.Errorf("expected nil partitions for empty input, got known=%v unknown=%v", known, unknown)
	}
}

func TestFilterRecommendations_EmptyCatalogDropsAll(t *testing.T) {
	t.Parallel()
	recs := []Recommendation{mustRec(t, "shell:minimal")}
	known, unknown := FilterRecommendations(recs, nil)
	if len(known) != 0 {
		t.Errorf("known should be empty when catalog is empty, got %d", len(known))
	}
	if len(unknown) != 1 {
		t.Errorf("unknown should be 1 when catalog is empty, got %d", len(unknown))
	}
}
