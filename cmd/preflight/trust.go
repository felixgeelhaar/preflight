package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/preflight/internal/domain/catalog"
	"github.com/spf13/cobra"
)

var trustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Manage trusted catalog publishers",
	Long: `Manage trusted public keys for catalog signature verification.

Trust levels determine how catalogs are verified:
  builtin    - Embedded in the preflight binary
  verified   - Signed by a trusted publisher (GPG/SSH/Sigstore)
  community  - Hash-verified with user approval
  untrusted  - No verification (requires --allow-untrusted)

Examples:
  preflight trust list                    # List trusted keys
  preflight trust add <keyfile>           # Add a trusted key
  preflight trust remove <keyid>          # Remove a trusted key
  preflight trust show <keyid>            # Show key details`,
}

var trustListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List trusted keys",
	Long: `Display all trusted public keys.

Shows key ID, type, publisher, trust level, and status.`,
	RunE: runTrustList,
}

var trustAddCmd = &cobra.Command{
	Use:   "add <keyfile>",
	Short: "Add a trusted key",
	Long: `Add a public key to the trust store.

Supported key formats:
  - SSH public keys (id_ed25519.pub, id_rsa.pub)
  - GPG public keys (armored or binary)

Examples:
  preflight trust add ~/.ssh/id_ed25519.pub
  preflight trust add publisher.gpg --name "Publisher Name"
  preflight trust add key.pub --level verified`,
	Args: cobra.ExactArgs(1),
	RunE: runTrustAdd,
}

var trustRemoveCmd = &cobra.Command{
	Use:     "remove <keyid>",
	Aliases: []string{"rm"},
	Short:   "Remove a trusted key",
	Long: `Remove a key from the trust store.

Examples:
  preflight trust remove SHA256:abc123...
  preflight trust remove key-name`,
	Args: cobra.ExactArgs(1),
	RunE: runTrustRemove,
}

var trustShowCmd = &cobra.Command{
	Use:   "show <keyid>",
	Short: "Show key details",
	Long: `Display detailed information about a trusted key.

Examples:
  preflight trust show SHA256:abc123...`,
	Args: cobra.ExactArgs(1),
	RunE: runTrustShow,
}

// Flags
var (
	trustKeyName  string
	trustKeyLevel string
	trustKeyType  string
	trustEmail    string
	trustForce    bool
)

func init() {
	// Add flags
	trustAddCmd.Flags().StringVar(&trustKeyName, "name", "", "Publisher name")
	trustAddCmd.Flags().StringVar(&trustEmail, "email", "", "Publisher email")
	trustAddCmd.Flags().StringVar(&trustKeyLevel, "level", "community", "Trust level (builtin, verified, community)")
	trustAddCmd.Flags().StringVar(&trustKeyType, "type", "", "Key type (ssh, gpg) - auto-detected if not specified")

	trustRemoveCmd.Flags().BoolVar(&trustForce, "force", false, "Skip confirmation")

	// Add subcommands
	trustCmd.AddCommand(trustListCmd)
	trustCmd.AddCommand(trustAddCmd)
	trustCmd.AddCommand(trustRemoveCmd)
	trustCmd.AddCommand(trustShowCmd)

	rootCmd.AddCommand(trustCmd)
}

// getTrustStore returns the trust store with keys loaded.
func getTrustStore() (*catalog.TrustStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	storePath := filepath.Join(homeDir, ".preflight", "trust.json")
	store := catalog.NewTrustStore(storePath)

	if err := store.Load(); err != nil {
		return nil, fmt.Errorf("failed to load trust store: %w", err)
	}

	return store, nil
}

func runTrustList(_ *cobra.Command, _ []string) error {
	store, err := getTrustStore()
	if err != nil {
		return err
	}

	keys := store.List()
	if len(keys) == 0 {
		fmt.Println("No trusted keys.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "KEY ID\tTYPE\tPUBLISHER\tLEVEL\tSTATUS")

	for _, key := range keys {
		status := "active"
		if key.IsExpired() {
			status = "expired"
		}

		publisher := key.Publisher().Name()
		if publisher == "" {
			publisher = "-"
		}

		keyID := key.KeyID()
		if len(keyID) > 20 {
			keyID = keyID[:20] + "..."
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			keyID,
			key.KeyType(),
			publisher,
			key.TrustLevel(),
			status,
		)
	}

	_ = w.Flush()

	// Show stats
	stats := store.Stats()
	fmt.Printf("\nTotal: %d keys (%d SSH, %d GPG, %d Sigstore)\n",
		stats.TotalKeys, stats.SSHKeys, stats.GPGKeys, stats.SigstoreKeys)

	if stats.ExpiredKeys > 0 {
		fmt.Printf("Warning: %d expired keys\n", stats.ExpiredKeys)
	}

	return nil
}

func runTrustAdd(_ *cobra.Command, args []string) error {
	keyFile := args[0]

	// Read key file
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Detect key type
	keyType := detectKeyType(keyData)
	if trustKeyType != "" {
		switch trustKeyType {
		case "ssh":
			keyType = catalog.SignatureTypeSSH
		case "gpg":
			keyType = catalog.SignatureTypeGPG
		default:
			return fmt.Errorf("unknown key type: %s", trustKeyType)
		}
	}

	if keyType == "" {
		return fmt.Errorf("could not detect key type; specify with --type")
	}

	// Compute fingerprint
	fingerprint := catalog.ComputeKeyFingerprint(keyData)

	// Use fingerprint as key ID if no name provided
	keyID := fingerprint
	if trustKeyName != "" {
		keyID = trustKeyName
	}

	// Parse trust level
	level, err := catalog.TrustLevelFromString(trustKeyLevel)
	if err != nil {
		return err
	}

	// Create publisher
	publisher := catalog.NewPublisher(trustKeyName, trustEmail, keyID, keyType)

	// Create trusted key
	key := catalog.NewTrustedKey(keyID, keyType, nil, publisher)
	key.SetTrustLevel(level)
	key.SetFingerprint(fingerprint)
	key.SetComment(fmt.Sprintf("Added from %s", filepath.Base(keyFile)))

	// Add to store
	store, err := getTrustStore()
	if err != nil {
		return err
	}

	if err := store.Add(key); err != nil {
		return fmt.Errorf("failed to add key: %w", err)
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save trust store: %w", err)
	}

	fmt.Printf("Added key: %s\n", keyID)
	fmt.Printf("  Type: %s\n", keyType)
	fmt.Printf("  Fingerprint: %s\n", fingerprint)
	fmt.Printf("  Trust level: %s\n", level)

	return nil
}

func runTrustRemove(_ *cobra.Command, args []string) error {
	keyID := args[0]

	store, err := getTrustStore()
	if err != nil {
		return err
	}

	key, ok := store.Get(keyID)
	if !ok {
		return fmt.Errorf("key not found: %s", keyID)
	}

	// Confirm removal
	if !trustForce {
		fmt.Printf("Remove key '%s'", keyID)
		if key.Publisher().Name() != "" {
			fmt.Printf(" (%s)", key.Publisher().Name())
		}
		fmt.Print("? [y/N] ")

		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := store.Remove(keyID); err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save trust store: %w", err)
	}

	fmt.Printf("Removed key: %s\n", keyID)
	return nil
}

func runTrustShow(_ *cobra.Command, args []string) error {
	keyID := args[0]

	store, err := getTrustStore()
	if err != nil {
		return err
	}

	key, ok := store.Get(keyID)
	if !ok {
		return fmt.Errorf("key not found: %s", keyID)
	}

	fmt.Printf("Key ID:      %s\n", key.KeyID())
	fmt.Printf("Type:        %s\n", key.KeyType())
	fmt.Printf("Fingerprint: %s\n", key.Fingerprint())
	fmt.Printf("Trust Level: %s\n", key.TrustLevel())

	if !key.Publisher().IsZero() {
		fmt.Println("\nPublisher:")
		if key.Publisher().Name() != "" {
			fmt.Printf("  Name:  %s\n", key.Publisher().Name())
		}
		if key.Publisher().Email() != "" {
			fmt.Printf("  Email: %s\n", key.Publisher().Email())
		}
	}

	fmt.Printf("\nAdded:   %s\n", key.AddedAt().Format(time.RFC3339))
	if !key.ExpiresAt().IsZero() {
		fmt.Printf("Expires: %s\n", key.ExpiresAt().Format(time.RFC3339))
		if key.IsExpired() {
			fmt.Println("Status:  EXPIRED")
		} else {
			fmt.Println("Status:  active")
		}
	} else {
		fmt.Println("Status:  active (no expiration)")
	}

	if key.Comment() != "" {
		fmt.Printf("\nComment: %s\n", key.Comment())
	}

	return nil
}

// detectKeyType attempts to detect the key type from the data.
func detectKeyType(data []byte) catalog.SignatureType {
	content := string(data)

	// SSH key detection
	if len(data) > 4 {
		prefix := content[:4]
		if prefix == "ssh-" || prefix == "ecds" || prefix == "sk-s" {
			return catalog.SignatureTypeSSH
		}
	}

	// GPG key detection (armored)
	gpgPrefix := "-----BEGIN PGP PUBLIC KEY"
	if len(content) >= len(gpgPrefix) && content[:len(gpgPrefix)] == gpgPrefix {
		return catalog.SignatureTypeGPG
	}

	// GPG binary detection (starts with packet header)
	if len(data) > 0 && (data[0]&0x80) != 0 {
		return catalog.SignatureTypeGPG
	}

	return ""
}
