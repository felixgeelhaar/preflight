package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// tokenJSON is the JSON serialization format for Token.
type tokenJSON struct {
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	TokenType    string            `json:"token_type"`
	ExpiresAt    time.Time         `json:"expires_at"`
	ProviderName string            `json:"provider_name"`
	Subject      string            `json:"subject,omitempty"`
	Email        string            `json:"email,omitempty"`
	Name         string            `json:"name,omitempty"`
	Groups       []string          `json:"groups,omitempty"`
	Issuer       string            `json:"issuer,omitempty"`
	Audience     string            `json:"audience,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
}

// TokenStore persists tokens to the filesystem.
// Tokens are stored at {basePath}/identity/{providerName}.json.
type TokenStore struct {
	basePath string
}

// NewTokenStore creates a new TokenStore.
func NewTokenStore(basePath string) *TokenStore {
	return &TokenStore{basePath: basePath}
}

// identityDir returns the identity token directory path.
func (s *TokenStore) identityDir() string {
	return filepath.Join(s.basePath, "identity")
}

// tokenPath returns the path for a provider's token file.
func (s *TokenStore) tokenPath(providerName string) string {
	return filepath.Join(s.identityDir(), providerName+".json")
}

// Save persists a token to disk.
func (s *TokenStore) Save(providerName string, token *Token) error {
	dir := s.identityDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create identity directory: %w", err)
	}

	tj := tokenJSON{
		AccessToken:  token.AccessToken(),
		RefreshToken: token.RefreshToken(),
		TokenType:    token.TokenType(),
		ExpiresAt:    token.ExpiresAt(),
		ProviderName: token.ProviderName(),
		Subject:      token.Claims().Subject(),
		Email:        token.Claims().Email(),
		Name:         token.Claims().Name(),
		Groups:       token.Claims().Groups(),
		Issuer:       token.Claims().Issuer(),
		Audience:     token.Claims().Audience(),
		Extra:        token.Claims().Extra(),
	}

	data, err := json.MarshalIndent(tj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(s.tokenPath(providerName), data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// Load reads a token from disk.
func (s *TokenStore) Load(providerName string) (*Token, error) {
	data, err := os.ReadFile(s.tokenPath(providerName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: no token for provider %q", ErrNotAuthenticated, providerName)
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var tj tokenJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	claims := NewClaims(tj.Subject, tj.Email, tj.Name, tj.Groups, tj.Issuer, tj.Audience, tj.Extra)

	token, err := NewToken(tj.AccessToken, tj.TokenType, tj.ExpiresAt, claims, tj.ProviderName)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct token: %w", err)
	}

	if tj.RefreshToken != "" {
		token = token.WithRefreshToken(tj.RefreshToken)
	}

	return &token, nil
}

// Delete removes a token from disk.
func (s *TokenStore) Delete(providerName string) error {
	err := os.Remove(s.tokenPath(providerName))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}
	return nil
}

// List returns the names of all stored providers, sorted alphabetically.
func (s *TokenStore) List() ([]string, error) {
	entries, err := os.ReadDir(s.identityDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list identity directory: %w", err)
	}

	var providers []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			providers = append(providers, strings.TrimSuffix(name, ".json"))
		}
	}

	sort.Strings(providers)

	return providers, nil
}
