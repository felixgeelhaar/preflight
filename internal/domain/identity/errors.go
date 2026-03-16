// Package identity provides identity and authentication management for Preflight.
package identity

import "errors"

// Sentinel errors for the identity domain.
var (
	ErrProviderNotFound     = errors.New("identity provider not found")
	ErrProviderExists       = errors.New("identity provider already registered")
	ErrNotAuthenticated     = errors.New("not authenticated")
	ErrTokenExpired         = errors.New("token expired")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrInvalidToken         = errors.New("invalid token")
	ErrInvalidConfig        = errors.New("invalid provider configuration")
	ErrDiscoveryFailed      = errors.New("OIDC discovery failed")
	ErrDeviceAuthFailed     = errors.New("device authorization failed")
	ErrTokenRefreshFailed   = errors.New("token refresh failed")
)
