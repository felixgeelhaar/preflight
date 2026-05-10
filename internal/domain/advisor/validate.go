package advisor

import "strings"

// FilterRecommendations partitions AI-produced recommendations into those
// whose PresetID is present in the supplied catalog and those that are not.
// Unknown recommendations are kept separately so the caller (TUI) can surface
// the count and require explicit user confirmation, honoring the CLAUDE.md
// guarantee: "AI outputs map to known presets or require user confirmation."
//
// presetIDs is the authoritative whitelist of preset identifiers
// (e.g. "shell:minimal", "nvim:kickstart"). Comparison is case-insensitive
// and trims surrounding whitespace.
func FilterRecommendations(recs []Recommendation, presetIDs []string) (known, unknown []Recommendation) {
	if len(recs) == 0 {
		return nil, nil
	}
	allowed := buildAllowedSet(presetIDs)
	for _, r := range recs {
		if _, ok := allowed[normalizePresetID(r.PresetID())]; ok {
			known = append(known, r)
		} else {
			unknown = append(unknown, r)
		}
	}
	return known, unknown
}

// FilterPresetIDs partitions a slice of preset/layer identifier strings (the
// shape returned in AIRecommendation.Presets / .Layers) into known and
// unknown, using the same normalization rules as FilterRecommendations. Use
// this on raw AI output before constructing domain Recommendation values so
// the catalog whitelist is enforced at the boundary.
func FilterPresetIDs(ids, allowedIDs []string) (known, unknown []string) {
	if len(ids) == 0 {
		return nil, nil
	}
	allowed := buildAllowedSet(allowedIDs)
	for _, id := range ids {
		if _, ok := allowed[normalizePresetID(id)]; ok {
			known = append(known, id)
		} else {
			unknown = append(unknown, id)
		}
	}
	return known, unknown
}

// FilterAIRecommendation partitions Presets and Layers fields of an AI
// response against the supplied catalog. Returns a filtered copy plus the
// dropped IDs. Callers should surface non-empty dropped slices to the user
// (TUI) for explicit confirmation per CLAUDE.md.
//
// If both knownPresets and knownLayers are empty, the recommendation is
// returned untouched. This makes the function safe to wire into call sites
// that have not yet connected a real catalog.
func FilterAIRecommendation(rec AIRecommendation, knownPresets, knownLayers []string) (filtered AIRecommendation, droppedPresets, droppedLayers []string) {
	if len(knownPresets) == 0 && len(knownLayers) == 0 {
		return rec, nil, nil
	}
	filtered.Explanation = rec.Explanation
	filtered.Presets, droppedPresets = FilterPresetIDs(rec.Presets, knownPresets)
	filtered.Layers, droppedLayers = FilterPresetIDs(rec.Layers, knownLayers)
	return filtered, droppedPresets, droppedLayers
}

func buildAllowedSet(ids []string) map[string]struct{} {
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		out[normalizePresetID(id)] = struct{}{}
	}
	return out
}

func normalizePresetID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}
