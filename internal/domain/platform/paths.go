package platform

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// PathTranslator handles path translation between Windows and WSL.
type PathTranslator struct {
	platform    *Platform
	windowsRoot string // e.g., /mnt/c
}

// NewPathTranslator creates a path translator for the given platform.
func NewPathTranslator(p *Platform) *PathTranslator {
	return &PathTranslator{
		platform:    p,
		windowsRoot: p.windowsPath,
	}
}

// windowsPathRegex matches Windows paths like C:\Users or C:/Users.
var windowsPathRegex = regexp.MustCompile(`^([A-Za-z]):[/\\](.*)$`)

// ToWSL converts a Windows path to its WSL equivalent.
// e.g., C:\Users\name -> /mnt/c/Users/name
func (t *PathTranslator) ToWSL(windowsPath string) (string, error) {
	if windowsPath == "" {
		return "", fmt.Errorf("empty path")
	}

	// Check if it's already a Unix path
	if strings.HasPrefix(windowsPath, "/") {
		return windowsPath, nil
	}

	// Parse Windows path
	matches := windowsPathRegex.FindStringSubmatch(windowsPath)
	if matches == nil {
		return "", fmt.Errorf("invalid Windows path: %s", windowsPath)
	}

	driveLetter := strings.ToLower(matches[1])
	relativePath := matches[2]

	// Convert backslashes to forward slashes
	relativePath = strings.ReplaceAll(relativePath, "\\", "/")

	// Construct WSL path
	wslPath := fmt.Sprintf("/mnt/%s/%s", driveLetter, relativePath)

	return filepath.Clean(wslPath), nil
}

// ToWindows converts a WSL path to its Windows equivalent.
// e.g., /mnt/c/Users/name -> C:\Users\name
func (t *PathTranslator) ToWindows(wslPath string) (string, error) {
	if wslPath == "" {
		return "", fmt.Errorf("empty path")
	}

	// Check if it's a Windows mount
	if !strings.HasPrefix(wslPath, "/mnt/") {
		// Could be a WSL-native path, return with \\wsl$ prefix
		return fmt.Sprintf("\\\\wsl$\\%s%s", t.platform.wslDistro, wslPath), nil
	}

	// Extract drive letter and path
	// /mnt/c/Users/name -> C:\Users\name
	parts := strings.SplitN(strings.TrimPrefix(wslPath, "/mnt/"), "/", 2)
	if len(parts) == 0 || len(parts[0]) != 1 {
		return "", fmt.Errorf("invalid WSL mount path: %s", wslPath)
	}

	driveLetter := strings.ToUpper(parts[0])
	var relativePath string
	if len(parts) > 1 {
		relativePath = parts[1]
	}

	// Convert to Windows path
	windowsPath := fmt.Sprintf("%s:\\%s", driveLetter, strings.ReplaceAll(relativePath, "/", "\\"))

	return windowsPath, nil
}

// wslpathCmd uses the wslpath utility for accurate translation.
func (t *PathTranslator) wslpathToUnix(windowsPath string) (string, error) {
	cmd := exec.Command("wslpath", "-u", windowsPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wslpath failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// wslpathToWindows uses the wslpath utility for accurate translation.
func (t *PathTranslator) wslpathToWindows(unixPath string) (string, error) {
	cmd := exec.Command("wslpath", "-w", unixPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wslpath failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ToWSLWithWslpath converts a path using the wslpath utility if available.
func (t *PathTranslator) ToWSLWithWslpath(windowsPath string) (string, error) {
	if t.platform.HasCommand("wslpath") {
		return t.wslpathToUnix(windowsPath)
	}
	return t.ToWSL(windowsPath)
}

// ToWindowsWithWslpath converts a path using the wslpath utility if available.
func (t *PathTranslator) ToWindowsWithWslpath(wslPath string) (string, error) {
	if t.platform.HasCommand("wslpath") {
		return t.wslpathToWindows(wslPath)
	}
	return t.ToWindows(wslPath)
}

// WindowsHome returns the Windows user home directory from WSL.
func (t *PathTranslator) WindowsHome() (string, error) {
	if !t.platform.IsWSL() {
		return "", fmt.Errorf("not running in WSL")
	}

	// Try to get from environment
	if home := getEnvFromWindows("USERPROFILE"); home != "" {
		return t.ToWSL(home)
	}

	// Fallback: try common location
	username := getEnvFromWindows("USERNAME")
	if username != "" {
		return fmt.Sprintf("/mnt/c/Users/%s", username), nil
	}

	return "", fmt.Errorf("could not determine Windows home directory")
}

// getEnvFromWindows gets an environment variable from the Windows host.
func getEnvFromWindows(name string) string {
	cmd := exec.Command("cmd.exe", "/c", "echo", "%"+name+"%")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	value := strings.TrimSpace(string(output))
	// Check if variable was not expanded (returns literal %NAME%)
	if value == "%"+name+"%" {
		return ""
	}
	return value
}

// ExpandWindowsVars expands Windows environment variables in a path.
func (t *PathTranslator) ExpandWindowsVars(path string) string {
	// Common Windows variables
	vars := map[string]string{
		"%USERPROFILE%":       getEnvFromWindows("USERPROFILE"),
		"%APPDATA%":           getEnvFromWindows("APPDATA"),
		"%LOCALAPPDATA%":      getEnvFromWindows("LOCALAPPDATA"),
		"%PROGRAMFILES%":      getEnvFromWindows("PROGRAMFILES"),
		"%PROGRAMFILES(X86)%": getEnvFromWindows("PROGRAMFILES(X86)"),
		"%SYSTEMROOT%":        getEnvFromWindows("SYSTEMROOT"),
		"%TEMP%":              getEnvFromWindows("TEMP"),
		"%TMP%":               getEnvFromWindows("TMP"),
	}

	result := path
	for varName, value := range vars {
		if value != "" {
			result = strings.ReplaceAll(result, varName, value)
		}
	}

	return result
}

// IsWindowsPath returns true if the path looks like a Windows path.
func IsWindowsPath(path string) bool {
	return windowsPathRegex.MatchString(path)
}

// IsWSLMountPath returns true if the path is a WSL Windows mount.
func IsWSLMountPath(path string) bool {
	if !strings.HasPrefix(path, "/mnt/") || len(path) < 6 {
		return false
	}
	driveLetter := path[5]
	return (driveLetter >= 'a' && driveLetter <= 'z') || (driveLetter >= 'A' && driveLetter <= 'Z')
}

// NormalizePath normalizes a path for the current platform.
func (t *PathTranslator) NormalizePath(path string) string {
	if t.platform.IsWindows() {
		return strings.ReplaceAll(path, "/", "\\")
	}
	return strings.ReplaceAll(path, "\\", "/")
}

// JoinPath joins path elements using the appropriate separator.
func (t *PathTranslator) JoinPath(elem ...string) string {
	if t.platform.IsWindows() {
		return strings.Join(elem, "\\")
	}
	return filepath.Join(elem...)
}
