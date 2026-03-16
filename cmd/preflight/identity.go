package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/identity"
	"github.com/spf13/cobra"
)

var identityCmd = &cobra.Command{
	Use:     "identity",
	Aliases: []string{"id", "auth"},
	Short:   "Manage enterprise identity providers",
	Long: `Authenticate with enterprise identity providers (OIDC/SAML) for trust chains,
plugin signatures, and fleet access.

Examples:
  preflight identity login --provider corporate
  preflight identity status
  preflight identity whoami
  preflight identity logout`,
}

var identityLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with an identity provider",
	Long: `Authenticate using an enterprise identity provider.

Uses the device authorization grant flow for OIDC providers,
which displays a code to enter in your browser.

Examples:
  preflight identity login --provider corporate
  preflight identity login                         # uses default provider`,
	RunE: runIdentityLogin,
}

var identityLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Long: `Remove stored authentication tokens for a provider.

Examples:
  preflight identity logout --provider corporate
  preflight identity logout                        # logout from default`,
	RunE: runIdentityLogout,
}

var identityStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long: `Display the current authentication state for all configured providers.

Examples:
  preflight identity status`,
	RunE: runIdentityStatus,
}

var identityWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated identity",
	Long: `Display the identity claims from the current authentication token.

Shows subject, email, name, groups, and issuer information.

Examples:
  preflight identity whoami
  preflight identity whoami --provider corporate`,
	RunE: runIdentityWhoami,
}

var (
	idProvider string
)

func init() {
	identityLoginCmd.Flags().StringVar(&idProvider, "provider", "", "Identity provider name")
	identityLogoutCmd.Flags().StringVar(&idProvider, "provider", "", "Identity provider name")
	identityWhoamiCmd.Flags().StringVar(&idProvider, "provider", "", "Identity provider name")

	identityCmd.AddCommand(identityLoginCmd)
	identityCmd.AddCommand(identityLogoutCmd)
	identityCmd.AddCommand(identityStatusCmd)
	identityCmd.AddCommand(identityWhoamiCmd)

	rootCmd.AddCommand(identityCmd)
}

func newIdentityService() (*identity.Service, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	store := identity.NewTokenStore(homeDir + "/.preflight/identity")
	return identity.NewService(store), nil
}

func resolveProvider(svc *identity.Service) string {
	if idProvider != "" {
		return idProvider
	}
	providers := svc.ListProviders()
	if len(providers) > 0 {
		return providers[0]
	}
	return ""
}

func runIdentityLogin(_ *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	svc, err := newIdentityService()
	if err != nil {
		return err
	}

	providerName := resolveProvider(svc)
	if providerName == "" {
		return fmt.Errorf("no identity provider configured; add one to preflight.yaml under 'identity.providers'")
	}

	fmt.Printf("Authenticating with provider %q...\n", providerName)

	token, err := svc.Login(ctx, providerName)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	claims := token.Claims()
	name := claims.Name()
	if name == "" {
		name = claims.Email()
	}
	if name == "" {
		name = claims.Subject()
	}

	fmt.Printf("Authenticated as %s\n", name)
	fmt.Printf("Token expires: %s\n", token.ExpiresAt().Format(time.RFC3339))
	return nil
}

func runIdentityLogout(_ *cobra.Command, _ []string) error {
	svc, err := newIdentityService()
	if err != nil {
		return err
	}

	providerName := resolveProvider(svc)
	if providerName == "" {
		return fmt.Errorf("no identity provider specified")
	}

	if err := svc.Logout(providerName); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}

	fmt.Printf("Logged out from provider %q\n", providerName)
	return nil
}

func runIdentityStatus(_ *cobra.Command, _ []string) error {
	svc, err := newIdentityService()
	if err != nil {
		return err
	}

	providers := svc.ListProviders()
	if len(providers) == 0 {
		fmt.Println("No identity providers configured.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROVIDER\tSTATUS\tIDENTITY\tEXPIRES")

	for _, name := range providers {
		token, err := svc.Status(name)
		if err != nil {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, "not authenticated", "-", "-")
			continue
		}

		status := "authenticated"
		if token.IsExpired() {
			status = "expired"
		}

		claims := token.Claims()
		ident := claims.Email()
		if ident == "" {
			ident = claims.Subject()
		}

		expires := token.ExpiresAt().Format(time.RFC3339)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, status, ident, expires)
	}

	return w.Flush()
}

func runIdentityWhoami(_ *cobra.Command, _ []string) error {
	svc, err := newIdentityService()
	if err != nil {
		return err
	}

	providerName := resolveProvider(svc)
	if providerName == "" {
		return fmt.Errorf("no identity provider specified")
	}

	claims, err := svc.WhoAmI(providerName)
	if err != nil {
		return fmt.Errorf("not authenticated with provider %q: %w", providerName, err)
	}

	fmt.Printf("Provider: %s\n", providerName)
	fmt.Printf("Subject:  %s\n", claims.Subject())
	if claims.Email() != "" {
		fmt.Printf("Email:    %s\n", claims.Email())
	}
	if claims.Name() != "" {
		fmt.Printf("Name:     %s\n", claims.Name())
	}
	if claims.Issuer() != "" {
		fmt.Printf("Issuer:   %s\n", claims.Issuer())
	}
	if len(claims.Groups()) > 0 {
		fmt.Printf("Groups:   %s\n", strings.Join(claims.Groups(), ", "))
	}

	return nil
}
