package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secret references",
	Long: `Manage secret references for secure configuration.

Preflight never stores secrets directly in configuration. Instead, it uses
references to external secret managers:

Supported backends:
  - 1password: 1Password CLI (op)
  - bitwarden: Bitwarden CLI (bw)
  - keychain: macOS Keychain
  - age: Age encryption
  - env: Environment variables

Secret references in config look like:
  git:
    signing_key: "secret://1password/GitHub/signing-key"

  ssh:
    keys:
      - name: work
        passphrase: "secret://keychain/ssh-work-passphrase"

Examples:
  preflight secrets list                     # List all secret references
  preflight secrets check                    # Verify all secrets are accessible
  preflight secrets set github-token         # Set a secret
  preflight secrets get github-token         # Get a secret value
  preflight secrets backends                 # Show available backends`,
	RunE: runSecretsList,
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List secret references in configuration",
	RunE:  runSecretsList,
}

var secretsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify all secrets are accessible",
	RunE:  runSecretsCheck,
}

var secretsSetCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Set a secret value",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsSet,
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get a secret value",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsGet,
}

var secretsBackendsCmd = &cobra.Command{
	Use:   "backends",
	Short: "Show available secret backends",
	RunE:  runSecretsBackends,
}

var (
	secretsConfigPath string
	secretsBackend    string
	secretsJSON       bool
)

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsCheckCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsBackendsCmd)

	secretsCmd.PersistentFlags().StringVarP(&secretsConfigPath, "config", "c", "preflight.yaml", "Path to preflight.yaml")
	secretsCmd.PersistentFlags().StringVar(&secretsBackend, "backend", "", "Secret backend (1password, bitwarden, keychain, age, env)")
	secretsCmd.PersistentFlags().BoolVar(&secretsJSON, "json", false, "Output as JSON")
}

// SecretRef represents a secret reference found in configuration
type SecretRef struct {
	Path     string `json:"path"`     // Config path (e.g., "git.signing_key")
	Backend  string `json:"backend"`  // Backend name
	Key      string `json:"key"`      // Key in backend
	Resolved bool   `json:"resolved"` // Whether secret was found
}

func runSecretsList(_ *cobra.Command, _ []string) error {
	refs, err := findSecretRefs(secretsConfigPath)
	if err != nil {
		return fmt.Errorf("failed to find secrets: %w", err)
	}

	if len(refs) == 0 {
		fmt.Println("No secret references found in configuration.")
		return nil
	}

	if secretsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(refs)
	}

	fmt.Printf("Found %d secret reference(s):\n\n", len(refs))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PATH\tBACKEND\tKEY")

	for _, ref := range refs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", ref.Path, ref.Backend, ref.Key)
	}

	_ = w.Flush()
	return nil
}

func runSecretsCheck(_ *cobra.Command, _ []string) error {
	refs, err := findSecretRefs(secretsConfigPath)
	if err != nil {
		return fmt.Errorf("failed to find secrets: %w", err)
	}

	if len(refs) == 0 {
		fmt.Println("No secret references to check.")
		return nil
	}

	fmt.Printf("Checking %d secret(s)...\n\n", len(refs))

	var passed, failed int
	for _, ref := range refs {
		resolved, err := resolveSecret(ref.Backend, ref.Key)
		if err != nil || resolved == "" {
			fmt.Printf("✗ %s: %s (%s)\n", ref.Path, ref.Key, ref.Backend)
			if err != nil {
				fmt.Printf("  Error: %v\n", err)
			}
			failed++
		} else {
			fmt.Printf("✓ %s: %s (%s)\n", ref.Path, ref.Key, ref.Backend)
			passed++
		}
	}

	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)

	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

func runSecretsSet(_ *cobra.Command, args []string) error {
	name := args[0]
	backend := secretsBackend

	if backend == "" {
		backend = "keychain" // Default backend
	}

	fmt.Printf("Enter secret value for '%s': ", name)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	value = strings.TrimSpace(value)

	if err := setSecret(backend, name, value); err != nil {
		return fmt.Errorf("failed to set secret: %w", err)
	}

	fmt.Printf("Secret '%s' saved to %s\n", name, backend)
	return nil
}

func runSecretsGet(_ *cobra.Command, args []string) error {
	name := args[0]
	backend := secretsBackend

	if backend == "" {
		backend = "keychain" // Default backend
	}

	value, err := resolveSecret(backend, name)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	if value == "" {
		return fmt.Errorf("secret '%s' not found in %s", name, backend)
	}

	fmt.Println(value)
	return nil
}

func runSecretsBackends(_ *cobra.Command, _ []string) error {
	backends := []struct {
		Name        string
		Description string
		Available   bool
		Command     string
	}{
		{"1password", "1Password CLI", check1PasswordCLI(), "op"},
		{"bitwarden", "Bitwarden CLI", checkBitwardenCLI(), "bw"},
		{"keychain", "macOS Keychain", checkKeychain(), "security"},
		{"age", "Age encryption", checkAgeCLI(), "age"},
		{"env", "Environment variables", true, ""},
	}

	if secretsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(backends)
	}

	fmt.Println("Available secret backends:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "BACKEND\tDESCRIPTION\tSTATUS\tCOMMAND")

	for _, b := range backends {
		status := "✗ not available"
		if b.Available {
			status = "✓ available"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", b.Name, b.Description, status, b.Command)
	}

	_ = w.Flush()
	return nil
}

func findSecretRefs(configPath string) ([]SecretRef, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var refs []SecretRef
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if idx := strings.Index(line, "secret://"); idx >= 0 {
			// Extract the secret reference
			rest := line[idx+9:]
			endIdx := strings.IndexAny(rest, "\"' \t\n")
			if endIdx == -1 {
				endIdx = len(rest)
			}
			ref := rest[:endIdx]

			// Parse backend/key
			parts := strings.SplitN(ref, "/", 2)
			if len(parts) == 2 {
				refs = append(refs, SecretRef{
					Path:    strings.TrimSpace(strings.Split(line, ":")[0]),
					Backend: parts[0],
					Key:     parts[1],
				})
			}
		}
	}

	return refs, nil
}

func resolveSecret(backend, key string) (string, error) {
	switch backend {
	case "1password":
		return resolve1Password(key)
	case "bitwarden":
		return resolveBitwarden(key)
	case "keychain":
		return resolveKeychain(key)
	case "age":
		return resolveAge(key)
	case "env":
		return os.Getenv(key), nil
	default:
		return "", fmt.Errorf("unknown backend: %s", backend)
	}
}

func setSecret(backend, name, value string) error {
	switch backend {
	case "keychain":
		return setKeychainSecret(name, value)
	case "env":
		return fmt.Errorf("cannot set environment variables through this command")
	default:
		return fmt.Errorf("setting secrets not supported for backend: %s", backend)
	}
}

// Backend availability checks
func check1PasswordCLI() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

func checkBitwardenCLI() bool {
	_, err := exec.LookPath("bw")
	return err == nil
}

func checkKeychain() bool {
	_, err := exec.LookPath("security")
	return err == nil
}

func checkAgeCLI() bool {
	_, err := exec.LookPath("age")
	return err == nil
}

// Secret resolution implementations
func resolve1Password(key string) (string, error) {
	// key format: vault/item/field
	parts := strings.Split(key, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid 1password key format: %s", key)
	}

	cmd := exec.Command("op", "item", "get", parts[1], "--vault", parts[0], "--field", parts[len(parts)-1])
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func resolveBitwarden(key string) (string, error) {
	cmd := exec.Command("bw", "get", "password", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func resolveKeychain(key string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", "preflight", "-a", key, "-w")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func setKeychainSecret(name, value string) error {
	// Delete existing if present
	_ = exec.Command("security", "delete-generic-password", "-s", "preflight", "-a", name).Run()

	// Add new
	cmd := exec.Command("security", "add-generic-password", "-s", "preflight", "-a", name, "-w", value)
	return cmd.Run()
}

func resolveAge(key string) (string, error) {
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".preflight", "secrets", key+".age")

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("age-encrypted secret not found: %s", key)
	}

	identityPath := filepath.Join(home, ".age", "key.txt")
	cmd := exec.Command("age", "-d", "-i", identityPath, keyPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
