// Package plugin provides plugin discovery, loading, and management.
package plugin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// maxResponseSize limits API response body to prevent memory exhaustion (2MB).
	maxResponseSize = 2 * 1024 * 1024
)

// SearchResult represents a plugin found via search.
type SearchResult struct {
	// Name is the repository name
	Name string `json:"name"`
	// FullName is the full repository name (owner/name)
	FullName string `json:"full_name"`
	// Description is the repository description
	Description string `json:"description"`
	// HTMLURL is the GitHub repository URL
	HTMLURL string `json:"html_url"`
	// CloneURL is the Git clone URL
	CloneURL string `json:"clone_url"`
	// Stars is the number of GitHub stars
	Stars int `json:"stargazers_count"`
	// UpdatedAt is when the repository was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// Topics are the repository topics
	Topics []string `json:"topics"`
	// Owner is the repository owner information
	Owner RepositoryOwner `json:"owner"`
	// License is the repository license
	License *RepositoryLicense `json:"license"`
	// PluginType is inferred from topics (config or provider)
	PluginType PluginType `json:"-"`

	// Trust signals
	// Forks is the number of forks
	Forks int `json:"forks_count"`
	// OpenIssues is the number of open issues
	OpenIssues int `json:"open_issues_count"`
	// IsArchived indicates if the repository is archived
	IsArchived bool `json:"archived"`
	// HasSignature is set after checking manifest
	HasSignature bool `json:"-"`
	// SignatureType is the type of signature (ssh, gpg, sigstore)
	SignatureType string `json:"-"`
	// TrustScore is the computed trust score (0-100)
	TrustScore int `json:"-"`
}

// TrustIndicator represents a trust level label for display.
type TrustIndicator string

// TrustIndicator constants for display levels.
const (
	TrustIndicatorHigh     TrustIndicator = "high"
	TrustIndicatorMedium   TrustIndicator = "medium"
	TrustIndicatorLow      TrustIndicator = "low"
	TrustIndicatorUnknown  TrustIndicator = "unknown"
	TrustIndicatorVerified TrustIndicator = "verified"
)

// ComputeTrustScore calculates a trust score (0-100) based on multiple signals.
// Scoring factors:
//   - Signature present: +30
//   - Stars (100+: +20, 50+: +15, 10+: +10)
//   - Activity (updated <30d: +20, <90d: +15)
//   - Forks (10+: +10, 5+: +5)
//   - Issue ratio (<0.1 issues per star: +10)
//   - License present: +5
//   - Archived: -20
func (r *SearchResult) ComputeTrustScore() int {
	score := 0

	// Signature present: +30
	if r.HasSignature {
		score += 30
	}

	// Stars scoring
	switch {
	case r.Stars >= 100:
		score += 20
	case r.Stars >= 50:
		score += 15
	case r.Stars >= 10:
		score += 10
	}

	// Activity scoring (based on UpdatedAt)
	daysSinceUpdate := time.Since(r.UpdatedAt).Hours() / 24
	switch {
	case daysSinceUpdate < 30:
		score += 20
	case daysSinceUpdate < 90:
		score += 15
	case daysSinceUpdate < 180:
		score += 10
	}

	// Forks scoring
	switch {
	case r.Forks >= 10:
		score += 10
	case r.Forks >= 5:
		score += 5
	}

	// Issue ratio scoring (low issues relative to stars is good)
	if r.Stars > 0 {
		issueRatio := float64(r.OpenIssues) / float64(r.Stars)
		if issueRatio < 0.1 {
			score += 10
		}
	}

	// License present: +5
	if r.License != nil && r.License.Key != "" {
		score += 5
	}

	// Archived: -20
	if r.IsArchived {
		score -= 20
	}

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	r.TrustScore = score
	return score
}

// GetTrustIndicator returns a trust level indicator based on the computed score.
// Thresholds: verified (signature), high (>=55), medium (>=35), low (>=15), unknown (<15)
func (r *SearchResult) GetTrustIndicator() TrustIndicator {
	if r.TrustScore == 0 {
		r.ComputeTrustScore()
	}

	// Verified signature takes precedence
	if r.HasSignature {
		return TrustIndicatorVerified
	}

	switch {
	case r.TrustScore >= 55:
		return TrustIndicatorHigh
	case r.TrustScore >= 35:
		return TrustIndicatorMedium
	case r.TrustScore >= 15:
		return TrustIndicatorLow
	default:
		return TrustIndicatorUnknown
	}
}

// Symbol returns a display symbol for the trust indicator.
func (ti TrustIndicator) Symbol() string {
	switch ti {
	case TrustIndicatorVerified:
		return "✓"
	case TrustIndicatorHigh:
		return "●"
	case TrustIndicatorMedium:
		return "◐"
	case TrustIndicatorLow:
		return "○"
	default:
		return "?"
	}
}

// RepositoryOwner represents the owner of a repository.
type RepositoryOwner struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// RepositoryLicense represents a repository license.
type RepositoryLicense struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// GitHubSearchResponse represents the GitHub search API response.
type GitHubSearchResponse struct {
	TotalCount        int            `json:"total_count"`
	IncompleteResults bool           `json:"incomplete_results"`
	Items             []SearchResult `json:"items"`
}

// SearchOptions configures plugin search behavior.
type SearchOptions struct {
	// Query is the search query
	Query string
	// Type filters by plugin type (empty for all)
	Type PluginType
	// MinStars filters by minimum stars
	MinStars int
	// Limit is the maximum number of results
	Limit int
	// SortBy is the sort field: "stars", "updated", "best-match"
	SortBy string
}

// DefaultSearchOptions returns default search options.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		Limit:  20,
		SortBy: "stars",
	}
}

// GitHubSearcher searches for plugins on GitHub.
type GitHubSearcher struct {
	client  *http.Client
	baseURL string
}

// NewSearcher creates a new GitHub plugin searcher with secure defaults.
func NewSearcher() *GitHubSearcher {
	return &GitHubSearcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
				},
			},
		},
		baseURL: "https://api.github.com",
	}
}

// NewSearcherWithClient creates a searcher with a custom HTTP client.
func NewSearcherWithClient(client *http.Client) *GitHubSearcher {
	return &GitHubSearcher{
		client:  client,
		baseURL: "https://api.github.com",
	}
}

// validateBaseURL checks that the base URL is secure.
// Only allows HTTPS, or HTTP for localhost/127.0.0.1 (testing).
func validateBaseURL(baseURL string) error {
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// Allow HTTPS always
	if u.Scheme == "https" {
		return nil
	}

	// Allow HTTP only for localhost (testing)
	if u.Scheme == "http" {
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("HTTP is only allowed for localhost (testing); use HTTPS for: %s", baseURL)
	}

	return fmt.Errorf("unsupported URL scheme %q; use HTTPS", u.Scheme)
}

// Search searches for plugins matching the query.
func (s *GitHubSearcher) Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error) {
	// Validate base URL scheme (HTTPS required for non-localhost)
	if err := validateBaseURL(s.baseURL); err != nil {
		return nil, err
	}

	// Build the GitHub search query
	query := buildSearchQuery(opts)

	// Build the URL
	u, err := url.Parse(s.baseURL + "/search/repositories")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	if opts.SortBy != "" && opts.SortBy != "best-match" {
		q.Set("sort", opts.SortBy)
		q.Set("order", "desc")
	}
	if opts.Limit > 0 {
		q.Set("per_page", fmt.Sprintf("%d", min(opts.Limit, 100)))
	}
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "preflight-cli")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body with size limit to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, sanitizeAPIError(resp.StatusCode, body)
	}

	// Parse response
	var searchResp GitHubSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Post-process results
	results := make([]SearchResult, 0, len(searchResp.Items))
	for _, item := range searchResp.Items {
		// Infer plugin type from topics
		item.PluginType = inferPluginType(item.Topics)

		// Apply type filter if specified
		if opts.Type != "" && item.PluginType != opts.Type {
			continue
		}

		// Apply stars filter
		if opts.MinStars > 0 && item.Stars < opts.MinStars {
			continue
		}

		results = append(results, item)
	}

	// Apply limit
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// buildSearchQuery constructs the GitHub search query string.
func buildSearchQuery(opts SearchOptions) string {
	var parts []string

	// Add user query
	if opts.Query != "" {
		parts = append(parts, opts.Query)
	}

	// Add topic filter based on type
	switch opts.Type {
	case TypeProvider:
		parts = append(parts, "topic:preflight-provider")
	case TypeConfig:
		parts = append(parts, "topic:preflight-plugin", "-topic:preflight-provider")
	default:
		// Search for any preflight plugin
		parts = append(parts, "topic:preflight-plugin OR topic:preflight-provider")
	}

	// Add minimum stars filter
	if opts.MinStars > 0 {
		parts = append(parts, fmt.Sprintf("stars:>=%d", opts.MinStars))
	}

	return strings.Join(parts, " ")
}

// inferPluginType determines plugin type from repository topics.
func inferPluginType(topics []string) PluginType {
	for _, topic := range topics {
		if topic == "preflight-provider" {
			return TypeProvider
		}
	}
	return TypeConfig
}

// FormatSearchResults formats search results for display.
func FormatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "No plugins found."
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d plugin(s):\n\n", len(results))

	for i := range results {
		r := &results[i]
		typeLabel := "config"
		if r.PluginType == TypeProvider {
			typeLabel = "provider"
		}

		// Compute trust score and get indicator
		r.ComputeTrustScore()
		trustIndicator := r.GetTrustIndicator()
		trustSymbol := trustIndicator.Symbol()

		fmt.Fprintf(&sb, "  %s %s [%s]\n", trustSymbol, r.FullName, typeLabel)
		if r.Description != "" {
			desc := r.Description
			if len(desc) > 70 {
				desc = desc[:67] + "..."
			}
			fmt.Fprintf(&sb, "    %s\n", desc)
		}

		// Enhanced stats line with trust score
		archiveLabel := ""
		if r.IsArchived {
			archiveLabel = " [ARCHIVED]"
		}
		fmt.Fprintf(&sb, "    ⭐ %d  🍴 %d  trust: %s (%d)%s\n",
			r.Stars, r.Forks, trustIndicator, r.TrustScore, archiveLabel)
		fmt.Fprintf(&sb, "    %s\n", r.HTMLURL)
		sb.WriteString("\n")
	}

	sb.WriteString("Trust indicators: ✓=verified ●=high ◐=medium ○=low ?=unknown\n")

	return sb.String()
}

// SortResults sorts search results by the specified field.
func SortResults(results []SearchResult, sortBy string) {
	switch sortBy {
	case "stars":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Stars > results[j].Stars
		})
	case "updated":
		sort.Slice(results, func(i, j int) bool {
			return results[i].UpdatedAt.After(results[j].UpdatedAt)
		})
	case "name":
		sort.Slice(results, func(i, j int) bool {
			return results[i].Name < results[j].Name
		})
	case "trust":
		// Compute trust scores if not already done
		for i := range results {
			if results[i].TrustScore == 0 {
				results[i].ComputeTrustScore()
			}
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].TrustScore > results[j].TrustScore
		})
	}
}

// sanitizeAPIError creates a user-friendly error message from API responses
// without exposing potentially sensitive information.
func sanitizeAPIError(statusCode int, _ []byte) error {
	// Map common status codes to user-friendly messages
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("GitHub API authentication required (status 401): consider setting GITHUB_TOKEN")
	case http.StatusForbidden:
		return fmt.Errorf("GitHub API rate limit exceeded or access forbidden (status 403)")
	case http.StatusNotFound:
		return fmt.Errorf("GitHub API endpoint not found (status 404)")
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("GitHub API validation failed (status 422): check search query syntax")
	case http.StatusServiceUnavailable:
		return fmt.Errorf("GitHub API temporarily unavailable (status 503): try again later")
	default:
		// For other errors, return status code only - don't expose response body
		return fmt.Errorf("GitHub API error (status %d)", statusCode)
	}
}

// SearchResultWithManifest includes manifest preview for search results.
type SearchResultWithManifest struct {
	SearchResult
	// Manifest is the plugin manifest preview (optional, fetched separately)
	Manifest *Manifest `json:"-"`
	// ManifestError is set if manifest fetch failed
	ManifestError error `json:"-"`
}

// FetchManifest fetches the plugin.yaml manifest for a search result.
// This enables preview of plugin capabilities before installation.
func (s *GitHubSearcher) FetchManifest(ctx context.Context, result SearchResult) (*Manifest, error) {
	if result.FullName == "" {
		return nil, fmt.Errorf("missing repository full name")
	}

	// Construct raw content URL for plugin.yaml
	// Format: https://raw.githubusercontent.com/{owner}/{repo}/{branch}/plugin.yaml
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/plugin.yaml", result.FullName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "preflight-cli")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		// Try master branch as fallback
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/master/plugin.yaml", result.FullName)
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating fallback request: %w", err)
		}
		req.Header.Set("User-Agent", "preflight-cli")

		resp, err = s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching manifest from master: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest not found (status %d)", resp.StatusCode)
	}

	// Read with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}

// InstallPlan represents what will happen when installing a plugin.
type InstallPlan struct {
	// Source is the installation source (URL or path)
	Source string
	// Plugin is the manifest preview
	Plugin *Manifest
	// Dependencies are required plugins that need to be installed first
	Dependencies []Dependency
	// Capabilities are the WASM capabilities that will be requested
	Capabilities []WASMCapability
	// TrustLevel is the determined trust level
	TrustLevel TrustLevel
	// Warnings are any issues found during planning
	Warnings []string
	// Actions describes what the installation will do
	Actions []string
}

// CreateInstallPlan creates an installation plan for preview/confirmation.
func CreateInstallPlan(manifest *Manifest, source string) *InstallPlan {
	if manifest == nil {
		return nil
	}

	plan := &InstallPlan{
		Source:       source,
		Plugin:       manifest,
		Dependencies: manifest.Requires,
		Actions:      []string{},
		Warnings:     []string{},
	}

	// Add capabilities for provider plugins
	if manifest.IsProviderPlugin() && manifest.WASM != nil {
		plan.Capabilities = manifest.WASM.Capabilities

		// Check for dangerous capabilities
		for _, cap := range manifest.WASM.Capabilities {
			if DangerousCapabilities[cap.Name] {
				plan.Warnings = append(plan.Warnings,
					fmt.Sprintf("Plugin requests dangerous capability %q: %s", cap.Name, cap.Justification))
			}
		}
	}

	// Determine trust level based on signature
	if manifest.Signature != nil {
		plan.TrustLevel = TrustVerified
		plan.Actions = append(plan.Actions, "Verify plugin signature")
	} else {
		plan.TrustLevel = TrustCommunity
		plan.Warnings = append(plan.Warnings, "Plugin is not signed - verify source manually")
	}

	// Add standard actions
	plan.Actions = append(plan.Actions, fmt.Sprintf("Install plugin %s@%s", manifest.Name, manifest.Version))

	if len(manifest.Provides.Providers) > 0 {
		for _, p := range manifest.Provides.Providers {
			plan.Actions = append(plan.Actions, fmt.Sprintf("Register provider: %s", p.Name))
		}
	}

	if len(manifest.Provides.Presets) > 0 {
		plan.Actions = append(plan.Actions, fmt.Sprintf("Register %d preset(s)", len(manifest.Provides.Presets)))
	}

	if len(manifest.Provides.CapabilityPacks) > 0 {
		plan.Actions = append(plan.Actions, fmt.Sprintf("Register %d capability pack(s)", len(manifest.Provides.CapabilityPacks)))
	}

	if len(plan.Dependencies) > 0 {
		for _, dep := range plan.Dependencies {
			if dep.Version != "" {
				plan.Actions = append(plan.Actions, fmt.Sprintf("Install dependency: %s@%s", dep.Name, dep.Version))
			} else {
				plan.Actions = append(plan.Actions, fmt.Sprintf("Install dependency: %s", dep.Name))
			}
		}
	}

	return plan
}

// FormatInstallPlan formats an install plan for display.
func FormatInstallPlan(plan *InstallPlan) string {
	if plan == nil {
		return ""
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "📦 Install Plan: %s@%s\n", plan.Plugin.Name, plan.Plugin.Version)
	fmt.Fprintf(&sb, "   Source: %s\n", plan.Source)
	fmt.Fprintf(&sb, "   Trust: %s\n\n", plan.TrustLevel)

	if plan.Plugin.Description != "" {
		fmt.Fprintf(&sb, "   %s\n\n", plan.Plugin.Description)
	}

	if len(plan.Warnings) > 0 {
		sb.WriteString("⚠️  Warnings:\n")
		for _, w := range plan.Warnings {
			fmt.Fprintf(&sb, "   • %s\n", w)
		}
		sb.WriteString("\n")
	}

	if len(plan.Capabilities) > 0 {
		sb.WriteString("🔐 Requested Capabilities:\n")
		for _, cap := range plan.Capabilities {
			optional := ""
			if cap.Optional {
				optional = " (optional)"
			}
			fmt.Fprintf(&sb, "   • %s%s\n", cap.Name, optional)
			if cap.Justification != "" {
				fmt.Fprintf(&sb, "     → %s\n", cap.Justification)
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("📋 Actions:\n")
	for i, action := range plan.Actions {
		fmt.Fprintf(&sb, "   %d. %s\n", i+1, action)
	}

	return sb.String()
}

// Ensure GitHubSearcher implements Searcher interface.
var _ Searcher = (*GitHubSearcher)(nil)

// TestableSearcher is a variant of GitHubSearcher for testing with configurable base URL.
type TestableSearcher struct {
	*GitHubSearcher
	rawBaseURL string
}

// NewTestableSearcher creates a searcher that can be used with test servers.
func NewTestableSearcher(baseURL string, client *http.Client) *TestableSearcher {
	return &TestableSearcher{
		GitHubSearcher: &GitHubSearcher{
			client:  client,
			baseURL: baseURL,
		},
		rawBaseURL: baseURL,
	}
}

// FetchManifestTestable fetches manifest using configurable base URL for testing.
func (s *TestableSearcher) FetchManifestTestable(ctx context.Context, result SearchResult) (*Manifest, error) {
	if result.FullName == "" {
		return nil, fmt.Errorf("missing repository full name")
	}

	// Construct raw content URL using test server
	rawURL := fmt.Sprintf("%s/%s/main/plugin.yaml", s.rawBaseURL, result.FullName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "preflight-cli")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		// Try master branch as fallback
		rawURL = fmt.Sprintf("%s/%s/master/plugin.yaml", s.rawBaseURL, result.FullName)
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating fallback request: %w", err)
		}
		req.Header.Set("User-Agent", "preflight-cli")

		resp, err = s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching manifest from master: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest not found (status %d)", resp.StatusCode)
	}

	// Read with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}
