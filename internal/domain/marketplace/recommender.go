package marketplace

import (
	"context"
	"sort"
	"strings"
)

// RecommendationReason explains why a package was recommended.
type RecommendationReason string

// Recommendation reason constants.
const (
	ReasonPopular         RecommendationReason = "popular"
	ReasonTrending        RecommendationReason = "trending"
	ReasonSimilarKeywords RecommendationReason = "similar_keywords"
	ReasonSameType        RecommendationReason = "same_type"
	ReasonSameAuthor      RecommendationReason = "same_author"
	ReasonComplementary   RecommendationReason = "complementary"
	ReasonRecentlyUpdated RecommendationReason = "recently_updated"
	ReasonHighlyRated     RecommendationReason = "highly_rated"
	ReasonProviderMatch   RecommendationReason = "provider_match"
	ReasonFeatured        RecommendationReason = "featured"
)

// Recommendation represents a recommended package with explanation.
type Recommendation struct {
	Package Package                `json:"package"`
	Score   float64                `json:"score"`
	Reasons []RecommendationReason `json:"reasons"`
}

// RecommenderConfig configures the recommendation engine.
type RecommenderConfig struct {
	// MaxRecommendations limits the number of recommendations returned
	MaxRecommendations int
	// PopularityWeight controls how much popularity affects score (0-1)
	PopularityWeight float64
	// RecencyWeight controls how much recency affects score (0-1)
	RecencyWeight float64
	// SimilarityWeight controls how much keyword similarity affects score (0-1)
	SimilarityWeight float64
	// IncludeInstalled includes already installed packages in results
	IncludeInstalled bool
}

// DefaultRecommenderConfig returns sensible defaults.
func DefaultRecommenderConfig() RecommenderConfig {
	return RecommenderConfig{
		MaxRecommendations: 10,
		PopularityWeight:   0.3,
		RecencyWeight:      0.2,
		SimilarityWeight:   0.5,
		IncludeInstalled:   false,
	}
}

// Recommender provides package recommendations based on user context.
type Recommender struct {
	config  RecommenderConfig
	service *Service
}

// NewRecommender creates a new recommendation engine.
func NewRecommender(service *Service, config RecommenderConfig) *Recommender {
	return &Recommender{
		config:  config,
		service: service,
	}
}

// UserContext provides information about the user's setup for personalized recommendations.
type UserContext struct {
	// InstalledPackages is the list of installed package IDs
	InstalledPackages []PackageID
	// PreferredTypes filters recommendations to specific types
	PreferredTypes []string
	// ActiveProviders lists providers the user has configured
	ActiveProviders []string
	// Keywords are topics the user is interested in
	Keywords []string
}

// RecommendForUser generates personalized recommendations based on user context.
func (r *Recommender) RecommendForUser(ctx context.Context, userCtx UserContext) ([]Recommendation, error) {
	idx, err := r.service.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	// Build set of installed packages for filtering
	installedSet := make(map[string]bool)
	for _, id := range userCtx.InstalledPackages {
		installedSet[id.String()] = true
	}

	// Collect keywords from installed packages for similarity matching
	userKeywords := make(map[string]int)
	for _, kw := range userCtx.Keywords {
		userKeywords[strings.ToLower(kw)]++
	}
	for _, id := range userCtx.InstalledPackages {
		if pkg, ok := idx.Get(id); ok {
			for _, kw := range pkg.Keywords {
				userKeywords[strings.ToLower(kw)]++
			}
		}
	}

	// Score all packages
	var recommendations []Recommendation
	for _, pkg := range idx.Packages {
		// Skip installed if configured
		if !r.config.IncludeInstalled && installedSet[pkg.ID.String()] {
			continue
		}

		// Skip if type filter doesn't match
		if len(userCtx.PreferredTypes) > 0 && !contains(userCtx.PreferredTypes, pkg.Type) {
			continue
		}

		rec := r.scorePackage(pkg, userKeywords, userCtx.ActiveProviders, installedSet, idx)
		if rec.Score > 0 {
			recommendations = append(recommendations, rec)
		}
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Limit results
	if len(recommendations) > r.config.MaxRecommendations {
		recommendations = recommendations[:r.config.MaxRecommendations]
	}

	return recommendations, nil
}

// RecommendSimilar finds packages similar to a given package.
func (r *Recommender) RecommendSimilar(ctx context.Context, id PackageID) ([]Recommendation, error) {
	idx, err := r.service.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	sourcePkg, ok := idx.Get(id)
	if !ok {
		return nil, ErrPackageNotFound
	}

	// Build keyword set from source package
	sourceKeywords := make(map[string]int)
	for _, kw := range sourcePkg.Keywords {
		sourceKeywords[strings.ToLower(kw)]++
	}

	var recommendations []Recommendation
	for _, pkg := range idx.Packages {
		// Skip the source package
		if pkg.ID.Equals(id) {
			continue
		}

		rec := r.scoreSimilarity(pkg, sourcePkg, sourceKeywords)
		if rec.Score > 0 {
			recommendations = append(recommendations, rec)
		}
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Limit results
	if len(recommendations) > r.config.MaxRecommendations {
		recommendations = recommendations[:r.config.MaxRecommendations]
	}

	return recommendations, nil
}

// PopularPackages returns the most popular packages.
func (r *Recommender) PopularPackages(ctx context.Context, pkgType string) ([]Recommendation, error) {
	idx, err := r.service.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	var recommendations []Recommendation
	for _, pkg := range idx.Packages {
		if pkgType != "" && pkg.Type != pkgType {
			continue
		}

		score := r.popularityScore(pkg)
		if score > 0 {
			recommendations = append(recommendations, Recommendation{
				Package: pkg,
				Score:   score,
				Reasons: []RecommendationReason{ReasonPopular},
			})
		}
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	if len(recommendations) > r.config.MaxRecommendations {
		recommendations = recommendations[:r.config.MaxRecommendations]
	}

	return recommendations, nil
}

// FeaturedPackages returns editorially selected featured packages.
func (r *Recommender) FeaturedPackages(ctx context.Context) ([]Recommendation, error) {
	idx, err := r.service.getIndex(ctx)
	if err != nil {
		return nil, err
	}

	var recommendations []Recommendation
	for _, pkg := range idx.Packages {
		// Featured packages are verified and have high engagement
		if !pkg.Provenance.Verified {
			continue
		}

		score := r.popularityScore(pkg)
		if pkg.Stars >= 10 && score > 0.5 {
			recommendations = append(recommendations, Recommendation{
				Package: pkg,
				Score:   score * 1.5, // Boost verified packages
				Reasons: []RecommendationReason{ReasonFeatured, ReasonHighlyRated},
			})
		}
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	if len(recommendations) > r.config.MaxRecommendations {
		recommendations = recommendations[:r.config.MaxRecommendations]
	}

	return recommendations, nil
}

// scorePackage computes a recommendation score for a package.
func (r *Recommender) scorePackage(
	pkg Package,
	userKeywords map[string]int,
	activeProviders []string,
	installedSet map[string]bool,
	idx *Index,
) Recommendation {
	var score float64
	var reasons []RecommendationReason

	// Popularity score
	popScore := r.popularityScore(pkg)
	if popScore > 0.7 {
		reasons = append(reasons, ReasonPopular)
	}
	score += popScore * r.config.PopularityWeight

	// Recency score
	recencyScore := r.recencyScore(pkg)
	if recencyScore > 0.7 {
		reasons = append(reasons, ReasonRecentlyUpdated)
	}
	score += recencyScore * r.config.RecencyWeight

	// Keyword similarity score
	keywordScore := r.keywordSimilarity(pkg.Keywords, userKeywords)
	if keywordScore > 0.3 {
		reasons = append(reasons, ReasonSimilarKeywords)
	}
	score += keywordScore * r.config.SimilarityWeight

	// Provider match boost
	for _, kw := range pkg.Keywords {
		for _, provider := range activeProviders {
			if strings.Contains(strings.ToLower(kw), strings.ToLower(provider)) {
				score += 0.2
				reasons = append(reasons, ReasonProviderMatch)
				break
			}
		}
	}

	// Author affinity - bonus if user has installed other packages by same author
	for installedID := range installedSet {
		if installedPkg, ok := idx.Get(MustNewPackageID(installedID)); ok {
			if installedPkg.Provenance.Author == pkg.Provenance.Author {
				score += 0.15
				reasons = append(reasons, ReasonSameAuthor)
				break
			}
		}
	}

	// Verified publisher boost
	if pkg.Provenance.Verified {
		score += 0.1
	}

	// Highly rated boost
	if pkg.Stars >= 5 {
		reasons = append(reasons, ReasonHighlyRated)
	}

	return Recommendation{
		Package: pkg,
		Score:   normalizeScore(score),
		Reasons: uniqueReasons(reasons),
	}
}

// scoreSimilarity scores how similar two packages are.
func (r *Recommender) scoreSimilarity(pkg, source Package, sourceKeywords map[string]int) Recommendation {
	var score float64
	var reasons []RecommendationReason

	// Same type bonus
	if pkg.Type == source.Type {
		score += 0.3
		reasons = append(reasons, ReasonSameType)
	}

	// Same author bonus
	if pkg.Provenance.Author == source.Provenance.Author && pkg.Provenance.Author != "" {
		score += 0.2
		reasons = append(reasons, ReasonSameAuthor)
	}

	// Keyword similarity
	keywordScore := r.keywordSimilarity(pkg.Keywords, sourceKeywords)
	if keywordScore > 0.3 {
		reasons = append(reasons, ReasonSimilarKeywords)
	}
	score += keywordScore * 0.5

	// Popularity boost for tie-breaking
	score += r.popularityScore(pkg) * 0.2

	// Complementary check - packages that commonly appear together
	// For now, this is based on keyword complementarity
	if r.areComplementary(pkg, source) {
		score += 0.25
		reasons = append(reasons, ReasonComplementary)
	}

	return Recommendation{
		Package: pkg,
		Score:   normalizeScore(score),
		Reasons: uniqueReasons(reasons),
	}
}

// popularityScore computes a 0-1 score based on downloads and stars.
func (r *Recommender) popularityScore(pkg Package) float64 {
	// Normalize downloads (assume 1000 is high)
	downloadScore := float64(pkg.Downloads) / 1000.0
	if downloadScore > 1.0 {
		downloadScore = 1.0
	}

	// Normalize stars (assume 50 is high)
	starScore := float64(pkg.Stars) / 50.0
	if starScore > 1.0 {
		starScore = 1.0
	}

	// Combine with weights
	return (downloadScore * 0.6) + (starScore * 0.4)
}

// recencyScore computes a 0-1 score based on how recently the package was updated.
func (r *Recommender) recencyScore(pkg Package) float64 {
	latest, ok := pkg.LatestVersion()
	if !ok {
		return 0
	}

	// Score based on release recency (30 days = 1.0, 365 days = 0.1)
	daysSinceRelease := int(pkg.UpdatedAt.Sub(latest.ReleasedAt).Hours() / 24)
	if daysSinceRelease < 0 {
		daysSinceRelease = 0
	}

	if daysSinceRelease <= 30 {
		return 1.0
	}
	if daysSinceRelease >= 365 {
		return 0.1
	}

	// Linear interpolation
	return 1.0 - (float64(daysSinceRelease-30) / 335.0 * 0.9)
}

// keywordSimilarity computes Jaccard similarity between package keywords and user keywords.
func (r *Recommender) keywordSimilarity(pkgKeywords []string, userKeywords map[string]int) float64 {
	if len(pkgKeywords) == 0 || len(userKeywords) == 0 {
		return 0
	}

	var intersection, union int
	pkgKeywordSet := make(map[string]bool)
	for _, kw := range pkgKeywords {
		lower := strings.ToLower(kw)
		pkgKeywordSet[lower] = true
		if userKeywords[lower] > 0 {
			intersection++
		}
	}

	union = len(userKeywords)
	for kw := range pkgKeywordSet {
		if userKeywords[kw] == 0 {
			union++
		}
	}

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// areComplementary checks if two packages are complementary.
func (r *Recommender) areComplementary(pkg, source Package) bool {
	// Different types that work well together
	complementaryPairs := map[string][]string{
		PackageTypePreset:         {PackageTypeCapabilityPack, PackageTypeLayerTemplate},
		PackageTypeCapabilityPack: {PackageTypePreset, PackageTypeLayerTemplate},
		PackageTypeLayerTemplate:  {PackageTypePreset, PackageTypeCapabilityPack},
	}

	allowedTypes, ok := complementaryPairs[source.Type]
	if !ok {
		return false
	}

	for _, t := range allowedTypes {
		if pkg.Type == t {
			// Check for keyword overlap (at least one common keyword)
			for _, pkgKw := range pkg.Keywords {
				for _, srcKw := range source.Keywords {
					if strings.EqualFold(pkgKw, srcKw) {
						return true
					}
				}
			}
		}
	}

	return false
}

// normalizeScore clamps a score to 0-1 range.
func normalizeScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// uniqueReasons removes duplicate reasons.
func uniqueReasons(reasons []RecommendationReason) []RecommendationReason {
	seen := make(map[RecommendationReason]bool)
	result := make([]RecommendationReason, 0, len(reasons))
	for _, r := range reasons {
		if !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}
	return result
}

// contains checks if a slice contains a value.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
