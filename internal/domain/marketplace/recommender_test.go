package marketplace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRecommenderConfig(t *testing.T) {
	t.Parallel()

	config := DefaultRecommenderConfig()

	assert.Equal(t, 10, config.MaxRecommendations)
	assert.InDelta(t, 0.3, config.PopularityWeight, 0.001)
	assert.InDelta(t, 0.2, config.RecencyWeight, 0.001)
	assert.InDelta(t, 0.5, config.SimilarityWeight, 0.001)
	assert.False(t, config.IncludeInstalled)
}

func TestRecommender_PopularityScore(t *testing.T) {
	t.Parallel()

	r := &Recommender{config: DefaultRecommenderConfig()}

	tests := []struct {
		name      string
		downloads int
		stars     int
		minScore  float64
		maxScore  float64
	}{
		{"zero engagement", 0, 0, 0, 0.01},
		{"low engagement", 100, 5, 0.1, 0.2},
		{"medium engagement", 500, 25, 0.4, 0.6},
		{"high engagement", 1000, 50, 0.9, 1.0},
		{"very high", 5000, 100, 0.9, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkg := Package{
				Downloads: tt.downloads,
				Stars:     tt.stars,
			}
			score := r.popularityScore(pkg)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestRecommender_RecencyScore(t *testing.T) {
	t.Parallel()

	r := &Recommender{config: DefaultRecommenderConfig()}

	tests := []struct {
		name     string
		daysAgo  int
		minScore float64
		maxScore float64
	}{
		{"just released", 0, 0.95, 1.0},
		{"week old", 7, 0.95, 1.0},
		{"month old", 30, 0.95, 1.0},
		{"two months", 60, 0.7, 0.95},
		{"six months", 180, 0.4, 0.7},
		{"year old", 365, 0.05, 0.15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			releaseDate := time.Now().Add(-time.Duration(tt.daysAgo) * 24 * time.Hour)
			pkg := Package{
				UpdatedAt: time.Now(),
				Versions: []PackageVersion{
					{
						Version:    "1.0.0",
						ReleasedAt: releaseDate,
					},
				},
			}
			score := r.recencyScore(pkg)
			assert.GreaterOrEqual(t, score, tt.minScore, "score should be >= %f", tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore, "score should be <= %f", tt.maxScore)
		})
	}
}

func TestRecommender_RecencyScore_NoVersions(t *testing.T) {
	t.Parallel()

	r := &Recommender{config: DefaultRecommenderConfig()}
	pkg := Package{Versions: []PackageVersion{}}

	score := r.recencyScore(pkg)
	assert.InDelta(t, 0.0, score, 0.001)
}

func TestRecommender_KeywordSimilarity(t *testing.T) {
	t.Parallel()

	r := &Recommender{config: DefaultRecommenderConfig()}

	tests := []struct {
		name         string
		pkgKeywords  []string
		userKeywords map[string]int
		minScore     float64
		maxScore     float64
	}{
		{
			name:         "no overlap",
			pkgKeywords:  []string{"vim", "editor"},
			userKeywords: map[string]int{"docker": 1, "kubernetes": 1},
			minScore:     0,
			maxScore:     0.01,
		},
		{
			name:         "full overlap",
			pkgKeywords:  []string{"vim", "neovim"},
			userKeywords: map[string]int{"vim": 1, "neovim": 1},
			minScore:     0.9,
			maxScore:     1.0,
		},
		{
			name:         "partial overlap",
			pkgKeywords:  []string{"vim", "editor", "lua"},
			userKeywords: map[string]int{"vim": 1, "neovim": 1},
			minScore:     0.2,
			maxScore:     0.4,
		},
		{
			name:         "empty pkg keywords",
			pkgKeywords:  []string{},
			userKeywords: map[string]int{"vim": 1},
			minScore:     0,
			maxScore:     0,
		},
		{
			name:         "empty user keywords",
			pkgKeywords:  []string{"vim"},
			userKeywords: map[string]int{},
			minScore:     0,
			maxScore:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			score := r.keywordSimilarity(tt.pkgKeywords, tt.userKeywords)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestRecommender_AreComplementary(t *testing.T) {
	t.Parallel()

	r := &Recommender{config: DefaultRecommenderConfig()}

	tests := []struct {
		name           string
		pkgType        string
		pkgKeywords    []string
		sourceType     string
		sourceKeywords []string
		expected       bool
	}{
		{
			name:           "complementary preset and pack",
			pkgType:        PackageTypeCapabilityPack,
			pkgKeywords:    []string{"vim", "editor"},
			sourceType:     PackageTypePreset,
			sourceKeywords: []string{"vim", "neovim"},
			expected:       true,
		},
		{
			name:           "same type not complementary",
			pkgType:        PackageTypePreset,
			pkgKeywords:    []string{"vim"},
			sourceType:     PackageTypePreset,
			sourceKeywords: []string{"vim"},
			expected:       false,
		},
		{
			name:           "different types no keyword overlap",
			pkgType:        PackageTypeCapabilityPack,
			pkgKeywords:    []string{"docker"},
			sourceType:     PackageTypePreset,
			sourceKeywords: []string{"vim"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pkg := Package{Type: tt.pkgType, Keywords: tt.pkgKeywords}
			source := Package{Type: tt.sourceType, Keywords: tt.sourceKeywords}
			result := r.areComplementary(pkg, source)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    float64
		expected float64
	}{
		{-0.5, 0},
		{0, 0},
		{0.5, 0.5},
		{1.0, 1.0},
		{1.5, 1.0},
	}

	for _, tt := range tests {
		assert.InDelta(t, tt.expected, normalizeScore(tt.input), 0.001)
	}
}

func TestUniqueReasons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []RecommendationReason
		expected []RecommendationReason
	}{
		{
			name:     "empty",
			input:    []RecommendationReason{},
			expected: []RecommendationReason{},
		},
		{
			name:     "no duplicates",
			input:    []RecommendationReason{ReasonPopular, ReasonTrending},
			expected: []RecommendationReason{ReasonPopular, ReasonTrending},
		},
		{
			name:     "with duplicates",
			input:    []RecommendationReason{ReasonPopular, ReasonTrending, ReasonPopular},
			expected: []RecommendationReason{ReasonPopular, ReasonTrending},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := uniqueReasons(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	slice := []string{"preset", "capability-pack", "layer-template"}

	assert.True(t, contains(slice, "preset"))
	assert.True(t, contains(slice, "capability-pack"))
	assert.False(t, contains(slice, "unknown"))
	assert.False(t, contains([]string{}, "preset"))
}

func TestRecommender_ScorePackage(t *testing.T) {
	t.Parallel()

	r := &Recommender{config: DefaultRecommenderConfig()}

	// Create a mock index with test packages
	testPkg := Package{
		ID:        MustNewPackageID("test-pkg"),
		Type:      PackageTypePreset,
		Title:     "Test Package",
		Keywords:  []string{"vim", "editor"},
		Downloads: 500,
		Stars:     25,
		UpdatedAt: time.Now(),
		Provenance: Provenance{
			Author:   "test-author",
			Verified: true,
		},
		Versions: []PackageVersion{
			{Version: "1.0.0", ReleasedAt: time.Now().Add(-7 * 24 * time.Hour)},
		},
	}
	idx := NewIndex()
	_ = idx.Add(testPkg)

	userKeywords := map[string]int{"vim": 1, "neovim": 1}
	activeProviders := []string{"nvim"}
	installedSet := map[string]bool{}

	pkg := testPkg
	rec := r.scorePackage(pkg, userKeywords, activeProviders, installedSet, idx)

	assert.Greater(t, rec.Score, 0.0)
	assert.NotEmpty(t, rec.Reasons)
	assert.Equal(t, pkg.ID, rec.Package.ID)
}

func TestRecommender_ScoreSimilarity(t *testing.T) {
	t.Parallel()

	r := &Recommender{config: DefaultRecommenderConfig()}

	source := Package{
		ID:       MustNewPackageID("source-pkg"),
		Type:     PackageTypePreset,
		Title:    "Source Package",
		Keywords: []string{"vim", "neovim", "lua"},
		Provenance: Provenance{
			Author: "test-author",
		},
	}

	tests := []struct {
		name          string
		pkg           Package
		expectReasons []RecommendationReason
		minScore      float64
	}{
		{
			name: "same type same author",
			pkg: Package{
				ID:         MustNewPackageID("similar-pkg"),
				Type:       PackageTypePreset,
				Keywords:   []string{"vim", "neovim"},
				Provenance: Provenance{Author: "test-author"},
			},
			expectReasons: []RecommendationReason{ReasonSameType, ReasonSameAuthor, ReasonSimilarKeywords},
			minScore:      0.5,
		},
		{
			name: "different type complementary",
			pkg: Package{
				ID:       MustNewPackageID("pack-pkg"),
				Type:     PackageTypeCapabilityPack,
				Keywords: []string{"vim", "lua"},
			},
			expectReasons: []RecommendationReason{ReasonSimilarKeywords, ReasonComplementary},
			minScore:      0.2,
		},
	}

	sourceKeywords := map[string]int{"vim": 1, "neovim": 1, "lua": 1}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := r.scoreSimilarity(tt.pkg, source, sourceKeywords)
			assert.GreaterOrEqual(t, rec.Score, tt.minScore)
			for _, expected := range tt.expectReasons {
				assert.Contains(t, rec.Reasons, expected)
			}
		})
	}
}

func TestUserContext(t *testing.T) {
	t.Parallel()

	ctx := UserContext{
		InstalledPackages: []PackageID{MustNewPackageID("pkg-1"), MustNewPackageID("pkg-2")},
		PreferredTypes:    []string{PackageTypePreset},
		ActiveProviders:   []string{"nvim", "brew"},
		Keywords:          []string{"vim", "dotfiles"},
	}

	assert.Len(t, ctx.InstalledPackages, 2)
	assert.Len(t, ctx.PreferredTypes, 1)
	assert.Len(t, ctx.ActiveProviders, 2)
	assert.Len(t, ctx.Keywords, 2)
}

func TestRecommendationReason_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason   RecommendationReason
		expected string
	}{
		{ReasonPopular, "popular"},
		{ReasonTrending, "trending"},
		{ReasonSimilarKeywords, "similar_keywords"},
		{ReasonSameType, "same_type"},
		{ReasonSameAuthor, "same_author"},
		{ReasonComplementary, "complementary"},
		{ReasonRecentlyUpdated, "recently_updated"},
		{ReasonHighlyRated, "highly_rated"},
		{ReasonProviderMatch, "provider_match"},
		{ReasonFeatured, "featured"},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, string(tt.reason))
		})
	}
}
