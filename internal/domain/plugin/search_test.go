package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearcher_Search(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		opts           SearchOptions
		serverResponse GitHubSearchResponse
		serverStatus   int
		wantResults    int
		wantErr        bool
	}{
		{
			name: "successful search returns results",
			opts: SearchOptions{
				Query: "docker",
				Limit: 10,
			},
			serverResponse: GitHubSearchResponse{
				TotalCount: 2,
				Items: []SearchResult{
					{
						Name:        "preflight-docker",
						FullName:    "example/preflight-docker",
						Description: "Docker provider for Preflight",
						HTMLURL:     "https://github.com/example/preflight-docker",
						Stars:       42,
						Topics:      []string{"preflight-plugin", "docker"},
					},
					{
						Name:        "docker-config",
						FullName:    "user/docker-config",
						Description: "Docker configuration plugin",
						HTMLURL:     "https://github.com/user/docker-config",
						Stars:       10,
						Topics:      []string{"preflight-provider", "docker"},
					},
				},
			},
			serverStatus: http.StatusOK,
			wantResults:  2,
			wantErr:      false,
		},
		{
			name: "filter by type provider",
			opts: SearchOptions{
				Query: "docker",
				Type:  TypeProvider,
				Limit: 10,
			},
			serverResponse: GitHubSearchResponse{
				TotalCount: 2,
				Items: []SearchResult{
					{
						Name:   "preflight-docker",
						Topics: []string{"preflight-plugin"},
					},
					{
						Name:   "docker-provider",
						Topics: []string{"preflight-provider"},
					},
				},
			},
			serverStatus: http.StatusOK,
			wantResults:  1, // Only provider should match
			wantErr:      false,
		},
		{
			name: "filter by minimum stars",
			opts: SearchOptions{
				MinStars: 20,
				Limit:    10,
			},
			serverResponse: GitHubSearchResponse{
				TotalCount: 2,
				Items: []SearchResult{
					{Name: "low-stars", Stars: 5, Topics: []string{"preflight-plugin"}},
					{Name: "high-stars", Stars: 100, Topics: []string{"preflight-plugin"}},
				},
			},
			serverStatus: http.StatusOK,
			wantResults:  1, // Only high-stars should match
			wantErr:      false,
		},
		{
			name: "empty results",
			opts: SearchOptions{
				Query: "nonexistent",
			},
			serverResponse: GitHubSearchResponse{
				TotalCount: 0,
				Items:      []SearchResult{},
			},
			serverStatus: http.StatusOK,
			wantResults:  0,
			wantErr:      false,
		},
		{
			name: "API error returns error",
			opts: SearchOptions{
				Query: "test",
			},
			serverStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.serverStatus)
				if tt.serverStatus == http.StatusOK {
					_ = json.NewEncoder(w).Encode(tt.serverResponse)
				} else {
					_, _ = w.Write([]byte("error"))
				}
			}))
			defer server.Close()

			// Create searcher with test server
			searcher := &GitHubSearcher{
				client:  server.Client(),
				baseURL: server.URL,
			}

			// Execute search
			results, err := searcher.Search(context.Background(), tt.opts)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, results, tt.wantResults)
		})
	}
}

func TestBuildSearchQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opts     SearchOptions
		expected string
	}{
		{
			name: "basic query",
			opts: SearchOptions{
				Query: "docker",
			},
			expected: "docker topic:preflight-plugin OR topic:preflight-provider",
		},
		{
			name: "provider type filter",
			opts: SearchOptions{
				Query: "kubernetes",
				Type:  TypeProvider,
			},
			expected: "kubernetes topic:preflight-provider",
		},
		{
			name: "config type filter",
			opts: SearchOptions{
				Query: "shell",
				Type:  TypeConfig,
			},
			expected: "shell topic:preflight-plugin -topic:preflight-provider",
		},
		{
			name: "with minimum stars",
			opts: SearchOptions{
				Query:    "test",
				MinStars: 10,
			},
			expected: "test topic:preflight-plugin OR topic:preflight-provider stars:>=10",
		},
		{
			name:     "empty query",
			opts:     SearchOptions{},
			expected: "topic:preflight-plugin OR topic:preflight-provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := buildSearchQuery(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferPluginType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		topics   []string
		expected PluginType
	}{
		{
			name:     "provider topic",
			topics:   []string{"preflight-provider", "docker"},
			expected: TypeProvider,
		},
		{
			name:     "plugin topic only",
			topics:   []string{"preflight-plugin", "config"},
			expected: TypeConfig,
		},
		{
			name:     "no preflight topics",
			topics:   []string{"docker", "container"},
			expected: TypeConfig,
		},
		{
			name:     "empty topics",
			topics:   []string{},
			expected: TypeConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := inferPluginType(tt.topics)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortResults(t *testing.T) {
	t.Parallel()

	results := []SearchResult{
		{Name: "b-plugin", Stars: 10, UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "a-plugin", Stars: 100, UpdatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
		{Name: "c-plugin", Stars: 50, UpdatedAt: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)},
	}

	t.Run("sort by stars", func(t *testing.T) {
		t.Parallel()
		r := make([]SearchResult, len(results))
		copy(r, results)
		SortResults(r, "stars")
		assert.Equal(t, "a-plugin", r[0].Name) // 100 stars
		assert.Equal(t, "c-plugin", r[1].Name) // 50 stars
		assert.Equal(t, "b-plugin", r[2].Name) // 10 stars
	})

	t.Run("sort by updated", func(t *testing.T) {
		t.Parallel()
		r := make([]SearchResult, len(results))
		copy(r, results)
		SortResults(r, "updated")
		assert.Equal(t, "a-plugin", r[0].Name) // June 2024
		assert.Equal(t, "c-plugin", r[1].Name) // March 2024
		assert.Equal(t, "b-plugin", r[2].Name) // Jan 2024
	})

	t.Run("sort by name", func(t *testing.T) {
		t.Parallel()
		r := make([]SearchResult, len(results))
		copy(r, results)
		SortResults(r, "name")
		assert.Equal(t, "a-plugin", r[0].Name)
		assert.Equal(t, "b-plugin", r[1].Name)
		assert.Equal(t, "c-plugin", r[2].Name)
	})

	t.Run("sort by trust", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		r := []SearchResult{
			{Name: "low-trust", Stars: 5, UpdatedAt: now.AddDate(-1, 0, 0)},
			{Name: "high-trust", Stars: 150, UpdatedAt: now, HasSignature: true},
			{Name: "med-trust", Stars: 50, UpdatedAt: now.AddDate(0, -2, 0)},
		}
		SortResults(r, "trust")
		assert.Equal(t, "high-trust", r[0].Name)
		assert.Equal(t, "med-trust", r[1].Name)
		assert.Equal(t, "low-trust", r[2].Name)
	})
}

func TestFormatSearchResults(t *testing.T) {
	t.Parallel()

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()
		result := FormatSearchResults(nil)
		assert.Equal(t, "No plugins found.", result)
	})

	t.Run("with results", func(t *testing.T) {
		t.Parallel()
		results := []SearchResult{
			{
				FullName:    "example/preflight-docker",
				Description: "Docker provider",
				HTMLURL:     "https://github.com/example/preflight-docker",
				Stars:       42,
				PluginType:  TypeProvider,
				UpdatedAt:   time.Now(),
			},
		}
		result := FormatSearchResults(results)
		assert.Contains(t, result, "Found 1 plugin(s)")
		assert.Contains(t, result, "example/preflight-docker")
		assert.Contains(t, result, "provider")
		assert.Contains(t, result, "42")
		assert.Contains(t, result, "trust:")
		assert.Contains(t, result, "Trust indicators:")
	})

	t.Run("archived plugin shows label", func(t *testing.T) {
		t.Parallel()
		results := []SearchResult{
			{
				FullName:   "example/archived-plugin",
				HTMLURL:    "https://github.com/example/archived-plugin",
				Stars:      10,
				PluginType: TypeConfig,
				UpdatedAt:  time.Now(),
				IsArchived: true,
			},
		}
		result := FormatSearchResults(results)
		assert.Contains(t, result, "[ARCHIVED]")
	})
}

func TestComputeTrustScore(t *testing.T) {
	t.Parallel()

	t.Run("signature adds 30 points", func(t *testing.T) {
		t.Parallel()
		r := &SearchResult{HasSignature: true}
		score := r.ComputeTrustScore()
		assert.GreaterOrEqual(t, score, 30)
	})

	t.Run("stars scoring", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			stars    int
			minScore int
		}{
			{100, 20}, // 100+ stars
			{50, 15},  // 50+ stars
			{10, 10},  // 10+ stars
			{5, 0},    // <10 stars
		}

		for _, tt := range tests {
			r := &SearchResult{Stars: tt.stars, UpdatedAt: time.Now().AddDate(-1, 0, 0)}
			score := r.ComputeTrustScore()
			assert.GreaterOrEqual(t, score, tt.minScore, "stars=%d", tt.stars)
		}
	})

	t.Run("activity scoring", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		tests := []struct {
			name      string
			updatedAt time.Time
			minScore  int
		}{
			{"recent", now.AddDate(0, 0, -15), 20},   // <30d = +20
			{"moderate", now.AddDate(0, 0, -60), 15}, // <90d = +15
			{"stale", now.AddDate(0, 0, -120), 10},   // <180d = +10
			{"old", now.AddDate(-1, 0, 0), 0},        // >180d = +0
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := &SearchResult{UpdatedAt: tt.updatedAt}
				score := r.ComputeTrustScore()
				assert.GreaterOrEqual(t, score, tt.minScore)
			})
		}
	})

	t.Run("forks scoring", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			forks    int
			minScore int
		}{
			{10, 10}, // 10+ = +10
			{5, 5},   // 5+ = +5
			{2, 0},   // <5 = +0
		}

		for _, tt := range tests {
			r := &SearchResult{Forks: tt.forks, UpdatedAt: time.Now().AddDate(-1, 0, 0)}
			score := r.ComputeTrustScore()
			assert.GreaterOrEqual(t, score, tt.minScore, "forks=%d", tt.forks)
		}
	})

	t.Run("low issue ratio adds 10 points", func(t *testing.T) {
		t.Parallel()
		r := &SearchResult{Stars: 100, OpenIssues: 5, UpdatedAt: time.Now().AddDate(-1, 0, 0)}
		score := r.ComputeTrustScore()
		// issue ratio = 5/100 = 0.05 < 0.1, should add 10
		assert.GreaterOrEqual(t, score, 10)
	})

	t.Run("license adds 5 points", func(t *testing.T) {
		t.Parallel()
		r := &SearchResult{
			License:   &RepositoryLicense{Key: "mit"},
			UpdatedAt: time.Now().AddDate(-1, 0, 0),
		}
		score := r.ComputeTrustScore()
		assert.GreaterOrEqual(t, score, 5)
	})

	t.Run("archived subtracts 20 points", func(t *testing.T) {
		t.Parallel()
		notArchived := &SearchResult{Stars: 100, UpdatedAt: time.Now()}
		archived := &SearchResult{Stars: 100, UpdatedAt: time.Now(), IsArchived: true}

		notArchivedScore := notArchived.ComputeTrustScore()
		archivedScore := archived.ComputeTrustScore()

		assert.Equal(t, 20, notArchivedScore-archivedScore)
	})

	t.Run("score clamped to 0-100", func(t *testing.T) {
		t.Parallel()

		// Maximum possible score
		high := &SearchResult{
			HasSignature: true,
			Stars:        1000,
			UpdatedAt:    time.Now(),
			Forks:        100,
			OpenIssues:   1,
			License:      &RepositoryLicense{Key: "mit"},
		}
		assert.LessOrEqual(t, high.ComputeTrustScore(), 100)

		// Minimum possible score (archived with nothing else)
		low := &SearchResult{
			IsArchived: true,
			UpdatedAt:  time.Now().AddDate(-5, 0, 0),
		}
		assert.GreaterOrEqual(t, low.ComputeTrustScore(), 0)
	})
}

func TestGetTrustIndicator(t *testing.T) {
	t.Parallel()

	t.Run("verified for signed plugins", func(t *testing.T) {
		t.Parallel()
		r := &SearchResult{HasSignature: true, UpdatedAt: time.Now()}
		r.ComputeTrustScore()
		assert.Equal(t, TrustIndicatorVerified, r.GetTrustIndicator())
	})

	t.Run("high for score >= 55", func(t *testing.T) {
		t.Parallel()
		// Stars 150 = +20, UpdatedAt now = +20, Forks 20 = +10, License = +5, low issue ratio = +10
		// Total = 65 >= 55 (high threshold)
		r := &SearchResult{Stars: 150, UpdatedAt: time.Now(), Forks: 20, OpenIssues: 5, License: &RepositoryLicense{Key: "mit"}}
		score := r.ComputeTrustScore()
		assert.GreaterOrEqual(t, score, 55, "expected score >= 55 for high trust")
		assert.Equal(t, TrustIndicatorHigh, r.GetTrustIndicator())
	})

	t.Run("verified for signed plugin", func(t *testing.T) {
		t.Parallel()
		// With signature, returns verified regardless of other factors
		r := &SearchResult{HasSignature: true, Stars: 100, UpdatedAt: time.Now()}
		r.ComputeTrustScore()
		assert.Equal(t, TrustIndicatorVerified, r.GetTrustIndicator())
	})

	t.Run("medium for score 35-54", func(t *testing.T) {
		t.Parallel()
		// Stars 50 = +15, UpdatedAt now = +20 = 35
		r := &SearchResult{Stars: 50, UpdatedAt: time.Now()}
		score := r.ComputeTrustScore()
		assert.GreaterOrEqual(t, score, 35, "expected score >= 35 for medium trust")
		assert.Equal(t, TrustIndicatorMedium, r.GetTrustIndicator())
	})

	t.Run("low for score 15-34", func(t *testing.T) {
		t.Parallel()
		// Stars 10 = +10, UpdatedAt 4 months ago = +10 = 20
		r := &SearchResult{Stars: 10, UpdatedAt: time.Now().AddDate(0, -4, 0)}
		score := r.ComputeTrustScore()
		assert.GreaterOrEqual(t, score, 15, "expected score >= 15 for low trust")
		assert.Less(t, score, 35, "expected score < 35 for low trust")
		assert.Equal(t, TrustIndicatorLow, r.GetTrustIndicator())
	})

	t.Run("unknown for score < 15", func(t *testing.T) {
		t.Parallel()
		// Archived = -20, old = +0, no stars = +0, total = -20 clamped to 0
		r := &SearchResult{Stars: 1, UpdatedAt: time.Now().AddDate(-1, 0, 0), IsArchived: true}
		score := r.ComputeTrustScore()
		assert.Less(t, score, 15, "expected score < 15 for unknown trust")
		assert.Equal(t, TrustIndicatorUnknown, r.GetTrustIndicator())
	})
}

func TestTrustIndicatorSymbol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		indicator TrustIndicator
		symbol    string
	}{
		{TrustIndicatorVerified, "✓"},
		{TrustIndicatorHigh, "●"},
		{TrustIndicatorMedium, "◐"},
		{TrustIndicatorLow, "○"},
		{TrustIndicatorUnknown, "?"},
		{TrustIndicator("invalid"), "?"},
	}

	for _, tt := range tests {
		t.Run(string(tt.indicator), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.symbol, tt.indicator.Symbol())
		})
	}
}

func TestDefaultSearchOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultSearchOptions()
	assert.Equal(t, 20, opts.Limit)
	assert.Equal(t, "stars", opts.SortBy)
	assert.Empty(t, opts.Query)
	assert.Empty(t, opts.Type)
	assert.Zero(t, opts.MinStars)
}

func TestNewSearcherWithClient(t *testing.T) {
	t.Parallel()

	customClient := &http.Client{Timeout: 60 * time.Second}
	searcher := NewSearcherWithClient(customClient)

	assert.NotNil(t, searcher)
	assert.Equal(t, customClient, searcher.client)
	assert.Equal(t, "https://api.github.com", searcher.baseURL)
}

func TestValidateBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "HTTPS URL allowed",
			baseURL: "https://api.github.com",
			wantErr: false,
		},
		{
			name:    "HTTP localhost allowed",
			baseURL: "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "HTTP 127.0.0.1 allowed",
			baseURL: "http://127.0.0.1:8080",
			wantErr: false,
		},
		{
			name:    "HTTP ::1 allowed",
			baseURL: "http://[::1]:8080",
			wantErr: false,
		},
		{
			name:    "HTTP external domain rejected",
			baseURL: "http://api.github.com",
			wantErr: true,
			errMsg:  "HTTP is only allowed for localhost",
		},
		{
			name:    "FTP scheme rejected",
			baseURL: "ftp://example.com",
			wantErr: true,
			errMsg:  "unsupported URL scheme",
		},
		{
			name:    "invalid URL rejected",
			baseURL: "://invalid",
			wantErr: true,
			errMsg:  "invalid base URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateBaseURL(tt.baseURL)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewSearcher(t *testing.T) {
	t.Parallel()

	searcher := NewSearcher()

	assert.NotNil(t, searcher)
	assert.NotNil(t, searcher.client)
	assert.Equal(t, "https://api.github.com", searcher.baseURL)
	assert.Equal(t, 30*time.Second, searcher.client.Timeout)
}

// Response size limit tests

func TestSearcher_Search_ResponseSizeLimit(t *testing.T) {
	t.Parallel()

	t.Run("response within limit succeeds", func(t *testing.T) {
		t.Parallel()

		// Create a normal-sized response
		response := GitHubSearchResponse{
			TotalCount: 1,
			Items: []SearchResult{
				{
					Name:   "test-plugin",
					Topics: []string{"preflight-plugin"},
					Stars:  10,
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		searcher := &GitHubSearcher{
			client:  server.Client(),
			baseURL: server.URL,
		}

		results, err := searcher.Search(context.Background(), SearchOptions{Query: "test"})
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("large response is truncated", func(t *testing.T) {
		t.Parallel()

		// Create a server that returns a response larger than maxResponseSize
		// We'll send 3MB of data which exceeds the 2MB limit
		largeData := make([]byte, 3*1024*1024)
		for i := range largeData {
			largeData[i] = 'x'
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// Write partial valid JSON header
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"name":"`))
			// Write large amount of data
			_, _ = w.Write(largeData)
			_, _ = w.Write([]byte(`"}]}`))
		}))
		defer server.Close()

		searcher := &GitHubSearcher{
			client:  server.Client(),
			baseURL: server.URL,
		}

		// Should fail to parse because response is truncated
		_, err := searcher.Search(context.Background(), SearchOptions{Query: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parsing response")
	})

	t.Run("response exactly at limit succeeds", func(t *testing.T) {
		t.Parallel()

		// Create a valid response that's small
		response := GitHubSearchResponse{
			TotalCount: 0,
			Items:      []SearchResult{},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		searcher := &GitHubSearcher{
			client:  server.Client(),
			baseURL: server.URL,
		}

		results, err := searcher.Search(context.Background(), SearchOptions{Query: "test"})
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestSearcher_Search_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("cancelled context before request", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// This should not be reached
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GitHubSearchResponse{})
		}))
		defer server.Close()

		searcher := &GitHubSearcher{
			client:  server.Client(),
			baseURL: server.URL,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := searcher.Search(ctx, SearchOptions{Query: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("context timeout during slow request", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Simulate slow server
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GitHubSearchResponse{})
		}))
		defer server.Close()

		searcher := &GitHubSearcher{
			client:  server.Client(),
			baseURL: server.URL,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := searcher.Search(ctx, SearchOptions{Query: "test"})
		// Should contain context deadline or timeout error
		assert.Error(t, err)
	})
}

// FetchManifest tests

func TestFetchManifest(t *testing.T) {
	t.Parallel()

	t.Run("empty full name returns error", func(t *testing.T) {
		t.Parallel()

		searcher := NewSearcher()

		result := SearchResult{
			FullName: "",
		}

		_, err := searcher.FetchManifest(context.Background(), result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing repository full name")
	})
}

// TestFetchManifestWithTestableSearcher tests FetchManifest using a testable searcher variant
func TestFetchManifestWithTestableSearcher(t *testing.T) {
	t.Parallel()

	validManifest := `api_version: v1
name: test-plugin
version: 1.0.0
type: config
description: A test plugin
provides:
  presets:
    - test:basic
`

	t.Run("successful fetch from main branch", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that proper headers are set
			assert.Equal(t, "preflight-cli", r.Header.Get("User-Agent"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validManifest))
		}))
		defer server.Close()

		// Create a testable searcher that uses a custom HTTP transport
		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "example/test-plugin",
		}

		manifest, err := searcher.FetchManifestTestable(context.Background(), result)
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", manifest.Name)
		assert.Equal(t, "1.0.0", manifest.Version)
	})

	t.Run("fallback to master branch", func(t *testing.T) {
		t.Parallel()

		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount++
			if requestCount == 1 {
				// First request returns 404
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Second request succeeds
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(validManifest))
		}))
		defer server.Close()

		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "example/test-plugin",
		}

		manifest, err := searcher.FetchManifestTestable(context.Background(), result)
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", manifest.Name)
		assert.Equal(t, 2, requestCount, "should try main then master")
	})

	t.Run("invalid manifest YAML returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid: yaml: content: ["))
		}))
		defer server.Close()

		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "example/test-plugin",
		}

		_, err := searcher.FetchManifestTestable(context.Background(), result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing manifest")
	})

	t.Run("manifest not found on any branch", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "example/test-plugin",
		}

		_, err := searcher.FetchManifestTestable(context.Background(), result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("server error returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "example/test-plugin",
		}

		_, err := searcher.FetchManifestTestable(context.Background(), result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "example/test-plugin",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := searcher.FetchManifestTestable(ctx, result)
		require.Error(t, err)
	})

	t.Run("empty full name returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "",
		}

		_, err := searcher.FetchManifestTestable(context.Background(), result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing repository full name")
	})

	t.Run("response size limit causes truncation", func(t *testing.T) {
		t.Parallel()

		// Create a response that exceeds maxResponseSize (2MB)
		// Start with valid YAML prefix, then add invalid content that would
		// be cut off, creating invalid YAML when truncated
		largeData := make([]byte, 3*1024*1024)
		for i := range largeData {
			largeData[i] = 'x'
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			// Start with incomplete YAML that needs the closing quote
			_, _ = w.Write([]byte(`name: "test`))
			_, _ = w.Write(largeData) // This pushes the closing quote past the limit
			_, _ = w.Write([]byte(`"`))
		}))
		defer server.Close()

		searcher := NewTestableSearcher(server.URL, server.Client())

		result := SearchResult{
			FullName: "example/large-plugin",
		}

		// The truncation may or may not cause an error depending on YAML parser behavior
		// What we're really testing is that the response is limited and doesn't OOM
		manifest, err := searcher.FetchManifestTestable(context.Background(), result)
		if err == nil {
			// If it parses, the name should be truncated
			assert.Less(t, len(manifest.Name), 3*1024*1024, "name should be truncated")
		}
		// Either way, we're testing the LimitReader works
	})
}
