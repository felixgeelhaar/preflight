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

func TestFilterPresetIDs_PartitionsAndIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	known, unknown := FilterPresetIDs(
		[]string{"shell:minimal", "NVIM:KICKSTART", "made-up:thing"},
		[]string{"shell:minimal", "nvim:kickstart"},
	)
	if len(known) != 2 || len(unknown) != 1 {
		t.Fatalf("partition wrong: known=%v unknown=%v", known, unknown)
	}
	if unknown[0] != "made-up:thing" {
		t.Errorf("unexpected unknown ID: %q", unknown[0])
	}
}

func TestFilterAIRecommendation_FiltersBothFields(t *testing.T) {
	t.Parallel()

	rec := AIRecommendation{
		Presets:     []string{"shell:minimal", "fake:preset"},
		Layers:      []string{"base", "phantom-layer"},
		Explanation: "because",
	}
	filtered, droppedP, droppedL := FilterAIRecommendation(
		rec,
		[]string{"shell:minimal"},
		[]string{"base"},
	)
	if len(filtered.Presets) != 1 || filtered.Presets[0] != "shell:minimal" {
		t.Errorf("filtered presets = %v, want [shell:minimal]", filtered.Presets)
	}
	if len(filtered.Layers) != 1 || filtered.Layers[0] != "base" {
		t.Errorf("filtered layers = %v, want [base]", filtered.Layers)
	}
	if len(droppedP) != 1 || droppedP[0] != "fake:preset" {
		t.Errorf("droppedPresets = %v, want [fake:preset]", droppedP)
	}
	if len(droppedL) != 1 || droppedL[0] != "phantom-layer" {
		t.Errorf("droppedLayers = %v, want [phantom-layer]", droppedL)
	}
	if filtered.Explanation != "because" {
		t.Errorf("explanation lost: %q", filtered.Explanation)
	}
}

// TestFilterAIRecommendation_NoCatalogReturnsAsIs documents the safety hatch:
// if no catalog is supplied, the recommendation passes through untouched so
// existing call sites are not broken before they wire in a catalog.
func TestFilterAIRecommendation_NoCatalogReturnsAsIs(t *testing.T) {
	t.Parallel()
	rec := AIRecommendation{
		Presets: []string{"anything"},
		Layers:  []string{"goes"},
	}
	filtered, dp, dl := FilterAIRecommendation(rec, nil, nil)
	if len(filtered.Presets) != 1 || len(filtered.Layers) != 1 {
		t.Errorf("expected no-op, got %+v", filtered)
	}
	if len(dp) != 0 || len(dl) != 0 {
		t.Errorf("expected no drops, got %v / %v", dp, dl)
	}
}

func TestParseAndFilterRecommendations_DropsUnknownPresets(t *testing.T) {
	t.Parallel()

	response := `{"presets":["shell:minimal","fake:preset"],"layers":["base"],"explanation":"x"}`
	rec, droppedP, droppedL, err := ParseAndFilterRecommendations(
		response,
		[]string{"shell:minimal"},
		[]string{"base"},
	)
	if err != nil {
		t.Fatalf("ParseAndFilterRecommendations: %v", err)
	}
	if len(rec.Presets) != 1 || rec.Presets[0] != "shell:minimal" {
		t.Errorf("filtered presets = %v, want [shell:minimal]", rec.Presets)
	}
	if len(droppedP) != 1 || droppedP[0] != "fake:preset" {
		t.Errorf("droppedPresets = %v, want [fake:preset]", droppedP)
	}
	if len(droppedL) != 0 {
		t.Errorf("droppedLayers should be empty, got %v", droppedL)
	}
}
