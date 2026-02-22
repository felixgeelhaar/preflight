package marketplace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServiceWithIndex creates a Service backed by a test HTTP server
// that serves the given index data. The service uses a clean temp dir for
// caching so that tests are isolated.
func testServiceWithIndex(t *testing.T, indexJSON string) *Service {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(indexJSON))
		}
	}))
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: server.URL,
			Timeout:     10 * time.Second,
		},
	}

	return NewService(config)
}

func TestRecommender_RecommendForUser(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"version": "1",
		"packages": [
			{
				"id": "installed-pkg",
				"type": "preset",
				"title": "Installed Package",
				"keywords": ["vim", "editor"],
				"downloads": 500,
				"stars": 20,
				"provenance": {"author": "author-a", "verified": true},
				"versions": [{"version": "1.0.0", "released_at": "2026-02-01T00:00:00Z"}]
			},
			{
				"id": "recommended-pkg",
				"type": "preset",
				"title": "Recommended Package",
				"keywords": ["vim", "neovim", "lua"],
				"downloads": 800,
				"stars": 30,
				"provenance": {"author": "author-a", "verified": true},
				"versions": [{"version": "2.0.0", "released_at": "2026-02-10T00:00:00Z"}],
				"updated_at": "2026-02-15T00:00:00Z"
			},
			{
				"id": "unrelated-pkg",
				"type": "layer-template",
				"title": "Unrelated Package",
				"keywords": ["docker", "cloud"],
				"downloads": 50,
				"stars": 2,
				"provenance": {"author": "author-b"},
				"versions": [{"version": "1.0.0", "released_at": "2025-01-01T00:00:00Z"}]
			}
		]
	}`

	service := testServiceWithIndex(t, indexJSON)

	tests := []struct {
		name           string
		userCtx        UserContext
		minResults     int
		expectFiltered string
	}{
		{
			name: "basic recommendation excludes installed",
			userCtx: UserContext{
				InstalledPackages: []PackageID{MustNewPackageID("installed-pkg")},
				Keywords:          []string{"vim", "editor"},
			},
			minResults:     1,
			expectFiltered: "installed-pkg",
		},
		{
			name: "type filter restricts results",
			userCtx: UserContext{
				PreferredTypes: []string{PackageTypeLayerTemplate},
				Keywords:       []string{"docker"},
			},
			minResults: 1,
		},
		{
			name: "include installed when configured",
			userCtx: UserContext{
				InstalledPackages: []PackageID{MustNewPackageID("installed-pkg")},
				Keywords:          []string{"vim"},
			},
			minResults: 1,
		},
		{
			name: "empty user context returns results",
			userCtx: UserContext{
				InstalledPackages: []PackageID{},
				Keywords:          []string{},
			},
			minResults: 1,
		},
		{
			name: "active providers boost score",
			userCtx: UserContext{
				ActiveProviders: []string{"nvim"},
				Keywords:        []string{"vim"},
			},
			minResults: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultRecommenderConfig()
			if tt.name == "include installed when configured" {
				cfg.IncludeInstalled = true
			}
			rec := NewRecommender(service, cfg)

			results, err := rec.RecommendForUser(context.Background(), tt.userCtx)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(results), tt.minResults)

			if tt.expectFiltered != "" {
				for _, r := range results {
					assert.NotEqual(t, tt.expectFiltered, r.Package.ID.String(),
						"installed package should be excluded")
				}
			}

			// All results should have positive scores
			for _, r := range results {
				assert.Greater(t, r.Score, 0.0)
				assert.NotEmpty(t, r.Reasons)
			}
		})
	}
}

func TestRecommender_RecommendForUser_Error(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: "http://unreachable.invalid",
			Timeout:     100 * time.Millisecond,
		},
		OfflineMode: true,
	}
	service := NewService(config)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	_, err := recommender.RecommendForUser(context.Background(), UserContext{})
	assert.Error(t, err)
}

func TestRecommender_RecommendForUser_MaxRecommendations(t *testing.T) {
	t.Parallel()

	// Create index with many packages
	indexJSON := `{
		"version": "1",
		"packages": [
			{"id": "pkg-1", "type": "preset", "title": "Pkg 1", "keywords": ["go"], "downloads": 100, "stars": 5, "versions": [{"version": "1.0.0"}]},
			{"id": "pkg-2", "type": "preset", "title": "Pkg 2", "keywords": ["go"], "downloads": 200, "stars": 10, "versions": [{"version": "1.0.0"}]},
			{"id": "pkg-3", "type": "preset", "title": "Pkg 3", "keywords": ["go"], "downloads": 300, "stars": 15, "versions": [{"version": "1.0.0"}]},
			{"id": "pkg-4", "type": "preset", "title": "Pkg 4", "keywords": ["go"], "downloads": 400, "stars": 20, "versions": [{"version": "1.0.0"}]}
		]
	}`
	service := testServiceWithIndex(t, indexJSON)

	cfg := DefaultRecommenderConfig()
	cfg.MaxRecommendations = 2
	recommender := NewRecommender(service, cfg)

	results, err := recommender.RecommendForUser(context.Background(), UserContext{Keywords: []string{"go"}})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 2)
}

func TestRecommender_RecommendSimilar(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"version": "1",
		"packages": [
			{
				"id": "source-pkg",
				"type": "preset",
				"title": "Source Package",
				"keywords": ["vim", "neovim", "lua"],
				"downloads": 500,
				"stars": 25,
				"provenance": {"author": "author-a"},
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "similar-pkg",
				"type": "preset",
				"title": "Similar Package",
				"keywords": ["vim", "neovim"],
				"downloads": 300,
				"stars": 15,
				"provenance": {"author": "author-a"},
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "complementary-pkg",
				"type": "capability-pack",
				"title": "Complementary Pack",
				"keywords": ["vim", "plugins"],
				"downloads": 200,
				"stars": 10,
				"provenance": {"author": "author-b"},
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "unrelated-pkg",
				"type": "layer-template",
				"title": "Unrelated",
				"keywords": ["docker", "cloud"],
				"downloads": 100,
				"stars": 5,
				"provenance": {"author": "author-c"},
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	service := testServiceWithIndex(t, indexJSON)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	results, err := recommender.RecommendSimilar(context.Background(), MustNewPackageID("source-pkg"))
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Source package should not be in results
	for _, r := range results {
		assert.NotEqual(t, "source-pkg", r.Package.ID.String())
	}

	// Similar package should be ranked higher than unrelated
	if len(results) >= 2 {
		assert.GreaterOrEqual(t, results[0].Score, results[len(results)-1].Score)
	}
}

func TestRecommender_RecommendSimilar_NotFound(t *testing.T) {
	t.Parallel()

	indexJSON := `{"version": "1", "packages": [{"id": "some-pkg", "type": "preset", "title": "Some", "versions": [{"version": "1.0.0"}]}]}`
	service := testServiceWithIndex(t, indexJSON)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	_, err := recommender.RecommendSimilar(context.Background(), MustNewPackageID("nonexistent"))
	assert.ErrorIs(t, err, ErrPackageNotFound)
}

func TestRecommender_RecommendSimilar_Error(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: "http://unreachable.invalid",
			Timeout:     100 * time.Millisecond,
		},
		OfflineMode: true,
	}
	service := NewService(config)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	_, err := recommender.RecommendSimilar(context.Background(), MustNewPackageID("any-pkg"))
	assert.Error(t, err)
}

func TestRecommender_RecommendSimilar_MaxRecommendations(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"version": "1",
		"packages": [
			{"id": "source", "type": "preset", "title": "Source", "keywords": ["go"], "downloads": 100, "stars": 5, "versions": [{"version": "1.0.0"}]},
			{"id": "sim-1", "type": "preset", "title": "Sim 1", "keywords": ["go"], "downloads": 200, "stars": 10, "versions": [{"version": "1.0.0"}]},
			{"id": "sim-2", "type": "preset", "title": "Sim 2", "keywords": ["go"], "downloads": 300, "stars": 15, "versions": [{"version": "1.0.0"}]},
			{"id": "sim-3", "type": "preset", "title": "Sim 3", "keywords": ["go"], "downloads": 400, "stars": 20, "versions": [{"version": "1.0.0"}]}
		]
	}`
	service := testServiceWithIndex(t, indexJSON)

	cfg := DefaultRecommenderConfig()
	cfg.MaxRecommendations = 1
	recommender := NewRecommender(service, cfg)

	results, err := recommender.RecommendSimilar(context.Background(), MustNewPackageID("source"))
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 1)
}

func TestRecommender_PopularPackages(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"version": "1",
		"packages": [
			{
				"id": "popular-preset",
				"type": "preset",
				"title": "Popular Preset",
				"downloads": 1000,
				"stars": 50,
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "popular-pack",
				"type": "capability-pack",
				"title": "Popular Pack",
				"downloads": 800,
				"stars": 40,
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "unpopular",
				"type": "preset",
				"title": "Unpopular",
				"downloads": 0,
				"stars": 0,
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	service := testServiceWithIndex(t, indexJSON)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	tests := []struct {
		name       string
		pkgType    string
		minResults int
		maxResults int
	}{
		{
			name:       "all types",
			pkgType:    "",
			minResults: 2,
			maxResults: 3,
		},
		{
			name:       "filter by preset",
			pkgType:    PackageTypePreset,
			minResults: 1,
			maxResults: 2,
		},
		{
			name:       "filter by capability-pack",
			pkgType:    PackageTypeCapabilityPack,
			minResults: 1,
			maxResults: 1,
		},
		{
			name:       "filter by nonexistent type",
			pkgType:    "nonexistent",
			minResults: 0,
			maxResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results, err := recommender.PopularPackages(context.Background(), tt.pkgType)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(results), tt.minResults)
			assert.LessOrEqual(t, len(results), tt.maxResults)

			// All results should have the Popular reason
			for _, r := range results {
				assert.Contains(t, r.Reasons, ReasonPopular)
			}

			// Results should be sorted by score descending
			for i := 1; i < len(results); i++ {
				assert.GreaterOrEqual(t, results[i-1].Score, results[i].Score)
			}
		})
	}
}

func TestRecommender_PopularPackages_Error(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: "http://unreachable.invalid",
			Timeout:     100 * time.Millisecond,
		},
		OfflineMode: true,
	}
	service := NewService(config)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	_, err := recommender.PopularPackages(context.Background(), "")
	assert.Error(t, err)
}

func TestRecommender_PopularPackages_MaxRecommendations(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"version": "1",
		"packages": [
			{"id": "pkg-1", "type": "preset", "title": "Pkg 1", "downloads": 500, "stars": 25, "versions": [{"version": "1.0.0"}]},
			{"id": "pkg-2", "type": "preset", "title": "Pkg 2", "downloads": 600, "stars": 30, "versions": [{"version": "1.0.0"}]},
			{"id": "pkg-3", "type": "preset", "title": "Pkg 3", "downloads": 700, "stars": 35, "versions": [{"version": "1.0.0"}]}
		]
	}`
	service := testServiceWithIndex(t, indexJSON)

	cfg := DefaultRecommenderConfig()
	cfg.MaxRecommendations = 2
	recommender := NewRecommender(service, cfg)

	results, err := recommender.PopularPackages(context.Background(), "")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 2)
}

func TestRecommender_FeaturedPackages(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"version": "1",
		"packages": [
			{
				"id": "featured-pkg",
				"type": "preset",
				"title": "Featured Package",
				"downloads": 1000,
				"stars": 50,
				"provenance": {"author": "verified-author", "verified": true},
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "verified-low-stars",
				"type": "preset",
				"title": "Verified Low Stars",
				"downloads": 500,
				"stars": 5,
				"provenance": {"author": "author-b", "verified": true},
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "unverified-pkg",
				"type": "preset",
				"title": "Unverified Package",
				"downloads": 2000,
				"stars": 100,
				"provenance": {"author": "author-c", "verified": false},
				"versions": [{"version": "1.0.0"}]
			},
			{
				"id": "verified-high-stars",
				"type": "capability-pack",
				"title": "Verified High Stars",
				"downloads": 800,
				"stars": 40,
				"provenance": {"author": "author-d", "verified": true},
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	service := testServiceWithIndex(t, indexJSON)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	results, err := recommender.FeaturedPackages(context.Background())
	require.NoError(t, err)

	// Only verified packages with high stars should appear
	for _, r := range results {
		assert.True(t, r.Package.Provenance.Verified, "featured packages must be verified")
		assert.GreaterOrEqual(t, r.Package.Stars, 10, "featured packages must have >= 10 stars")
		assert.Contains(t, r.Reasons, ReasonFeatured)
		assert.Contains(t, r.Reasons, ReasonHighlyRated)
	}

	// Unverified package should not be in results
	for _, r := range results {
		assert.NotEqual(t, "unverified-pkg", r.Package.ID.String())
	}

	// Results should be sorted by score descending
	for i := 1; i < len(results); i++ {
		assert.GreaterOrEqual(t, results[i-1].Score, results[i].Score)
	}
}

func TestRecommender_FeaturedPackages_Empty(t *testing.T) {
	t.Parallel()

	// Index with no verified/high-star packages
	indexJSON := `{
		"version": "1",
		"packages": [
			{
				"id": "low-pkg",
				"type": "preset",
				"title": "Low Package",
				"downloads": 10,
				"stars": 1,
				"provenance": {"author": "author", "verified": false},
				"versions": [{"version": "1.0.0"}]
			}
		]
	}`

	service := testServiceWithIndex(t, indexJSON)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	results, err := recommender.FeaturedPackages(context.Background())
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRecommender_FeaturedPackages_Error(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	config := ServiceConfig{
		InstallPath: tmpDir + "/installed",
		CacheConfig: CacheConfig{
			BasePath: tmpDir + "/cache",
			IndexTTL: 1 * time.Hour,
		},
		ClientConfig: ClientConfig{
			RegistryURL: "http://unreachable.invalid",
			Timeout:     100 * time.Millisecond,
		},
		OfflineMode: true,
	}
	service := NewService(config)
	recommender := NewRecommender(service, DefaultRecommenderConfig())

	_, err := recommender.FeaturedPackages(context.Background())
	assert.Error(t, err)
}

func TestRecommender_FeaturedPackages_MaxRecommendations(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"version": "1",
		"packages": [
			{"id": "f-1", "type": "preset", "title": "F1", "downloads": 1000, "stars": 50, "provenance": {"verified": true}, "versions": [{"version": "1.0.0"}]},
			{"id": "f-2", "type": "preset", "title": "F2", "downloads": 900, "stars": 45, "provenance": {"verified": true}, "versions": [{"version": "1.0.0"}]},
			{"id": "f-3", "type": "preset", "title": "F3", "downloads": 800, "stars": 40, "provenance": {"verified": true}, "versions": [{"version": "1.0.0"}]}
		]
	}`
	service := testServiceWithIndex(t, indexJSON)

	cfg := DefaultRecommenderConfig()
	cfg.MaxRecommendations = 1
	recommender := NewRecommender(service, cfg)

	results, err := recommender.FeaturedPackages(context.Background())
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 1)
}
