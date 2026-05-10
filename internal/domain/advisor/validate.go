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
	allowed := make(map[string]struct{}, len(presetIDs))
	for _, id := range presetIDs {
		allowed[normalizePresetID(id)] = struct{}{}
	}
	for _, r := range recs {
		if _, ok := allowed[normalizePresetID(r.PresetID())]; ok {
			known = append(known, r)
		} else {
			unknown = append(unknown, r)
		}
	}
	return known, unknown
}

func normalizePresetID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}
