package marketplace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client errors.
var (
	ErrFetchFailed  = errors.New("fetch failed")
	ErrNetworkError = errors.New("network error")
	ErrRateLimited  = errors.New("rate limited")
	ErrUnauthorized = errors.New("unauthorized")
	ErrServerError  = errors.New("server error")
)

// DefaultRegistryURL is the default marketplace registry.
const DefaultRegistryURL = "https://registry.preflight.dev"

// ClientConfig configures the HTTP client.
type ClientConfig struct {
	// RegistryURL is the base URL of the registry
	RegistryURL string
	// Timeout is the HTTP request timeout
	Timeout time.Duration
	// UserAgent is the User-Agent header value
	UserAgent string
	// AuthToken is an optional authentication token
	AuthToken string
}

// DefaultClientConfig returns sensible defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		RegistryURL: DefaultRegistryURL,
		Timeout:     30 * time.Second,
		UserAgent:   "preflight/2.6",
	}
}

// Client provides HTTP access to the marketplace registry.
type Client struct {
	config     ClientConfig
	httpClient *http.Client
}

// NewClient creates a new marketplace client.
func NewClient(config ClientConfig) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// FetchIndex downloads the package index from the registry.
func (c *Client) FetchIndex(ctx context.Context) (*Index, error) {
	url := c.config.RegistryURL + "/v1/index.json"

	data, err := c.fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index: %w", err)
	}

	idx, err := ParseIndex(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	return idx, nil
}

// FetchPackage downloads a specific package version.
func (c *Client) FetchPackage(ctx context.Context, id PackageID, version string) ([]byte, error) {
	url := fmt.Sprintf("%s/v1/packages/%s/%s.tar.gz", c.config.RegistryURL, id.String(), version)

	data, err := c.fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package %s@%s: %w", id, version, err)
	}

	return data, nil
}

// FetchPackageMetadata downloads metadata for a specific package.
func (c *Client) FetchPackageMetadata(ctx context.Context, id PackageID) (*Package, error) {
	url := fmt.Sprintf("%s/v1/packages/%s/metadata.json", c.config.RegistryURL, id.String())

	data, err := c.fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package metadata %s: %w", id, err)
	}

	var pkg Package
	idx, err := ParseIndex([]byte(fmt.Sprintf(`{"version":"1","packages":[%s]}`, string(data))))
	if err != nil || len(idx.Packages) == 0 {
		return nil, fmt.Errorf("failed to parse package metadata: %w", err)
	}

	pkg = idx.Packages[0]
	return &pkg, nil
}

// Search performs a server-side search query.
func (c *Client) Search(ctx context.Context, query string, opts SearchOptions) ([]Package, error) {
	url := fmt.Sprintf("%s/v1/search?q=%s", c.config.RegistryURL, query)

	if opts.Type != "" {
		url += "&type=" + opts.Type
	}
	if opts.Limit > 0 {
		url += fmt.Sprintf("&limit=%d", opts.Limit)
	}
	if opts.Offset > 0 {
		url += fmt.Sprintf("&offset=%d", opts.Offset)
	}

	data, err := c.fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	idx, err := ParseIndex(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return idx.Packages, nil
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	Type   string // Filter by package type
	Limit  int    // Maximum results
	Offset int    // Pagination offset
}

// fetch performs an HTTP GET request.
func (c *Client) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: request creation failed", ErrNetworkError)
	}

	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	if c.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.AuthToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed", ErrNetworkError)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle HTTP errors
	switch resp.StatusCode {
	case http.StatusOK:
		// Continue
	case http.StatusNotFound:
		return nil, ErrPackageNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrUnauthorized
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return nil, fmt.Errorf("%w: status %d", ErrServerError, resp.StatusCode)
	default:
		return nil, fmt.Errorf("%w: status %d", ErrFetchFailed, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response", ErrNetworkError)
	}

	return data, nil
}

// Ping checks if the registry is reachable.
func (c *Client) Ping(ctx context.Context) error {
	url := c.config.RegistryURL + "/v1/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: request creation failed", ErrNetworkError)
	}

	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: request failed", ErrNetworkError)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
