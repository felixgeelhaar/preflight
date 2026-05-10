// Package telemetry records a small, opt-in set of activation events to
// estimate the North Star metric (Time-to-First-Successful-Apply).
//
// **Default: disabled.** No event leaves the machine until the user has
// explicitly opted in by writing a consent file. Setting
// PREFLIGHT_TELEMETRY=off (or "0", "false") overrides any consent file.
//
// What this package does NOT do (yet):
//   - Send events to a remote endpoint.
//   - Record any path, package name, email, IP, hostname, or config content.
//
// See docs/north-star.md for the rationale and consent UX contract.
package telemetry

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// randReader is the source of nonces for machine ID derivation. Wrapped as a
// var so tests can override it deterministically.
var randReader io.Reader = rand.Reader

// Event names. Keep this set small; growing the event list dilutes signal.
const (
	EventInitCompleted    = "init.completed"
	EventApplyFirstOK     = "apply.first_success"
	EventDoctorGreen      = "doctor.green"
	EventCaptureCompleted = "capture.completed"
)

// allowedEvents is the closed set of event names that may be recorded. Any
// other input is silently dropped. This is the contract that prevents a
// future refactor (or careless caller) from leaking paths, package names, or
// other PII through the event-name field.
var allowedEvents = map[string]struct{}{
	EventInitCompleted:    {},
	EventApplyFirstOK:     {},
	EventDoctorGreen:      {},
	EventCaptureCompleted: {},
}

// Event is a single activation record. The on-disk format is JSON Lines.
type Event struct {
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	MachineID string    `json:"machine_id"`
}

// Recorder writes events to a local log file when telemetry is opted in.
// The zero value is a no-op recorder safe to call from any code path.
type Recorder struct {
	enabled   bool
	machineID string
	logPath   string
}

// NewRecorder loads consent state and returns a Recorder. dir is the
// preflight config directory (typically ~/.preflight). If consent is missing
// or PREFLIGHT_TELEMETRY explicitly disables it, all subsequent Record calls
// are no-ops.
func NewRecorder(dir string) *Recorder {
	r := &Recorder{logPath: filepath.Join(dir, "telemetry.jsonl")}

	if isExplicitlyDisabled() {
		return r
	}
	if !hasConsent(dir) {
		return r
	}

	r.machineID = loadOrCreateMachineID(dir)
	r.enabled = r.machineID != ""
	return r
}

// Record appends an Event to the local log if telemetry is enabled. It never
// returns an error — failure to record is silent by design (telemetry must
// not break the user's workflow).
func (r *Recorder) Record(name string) {
	if r == nil || !r.enabled {
		return
	}
	if _, ok := allowedEvents[name]; !ok {
		return
	}

	ev := Event{
		Name:      name,
		Timestamp: time.Now().UTC(),
		MachineID: r.machineID,
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return
	}

	f, err := os.OpenFile(r.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(line, '\n'))
}

// Enabled reports whether the recorder will actually record events. Useful
// for callers that need to skip expensive computation when telemetry is off.
func (r *Recorder) Enabled() bool {
	return r != nil && r.enabled
}

// GrantConsent writes the consent marker to dir. Subsequent NewRecorder
// calls in dir return an enabled recorder. Returns the path of the marker.
func GrantConsent(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	path := consentPath(dir)
	contents := fmt.Sprintf("granted_at: %s\nversion: 1\n", time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return "", fmt.Errorf("write consent marker: %w", err)
	}
	return path, nil
}

// RevokeConsent removes the consent marker and the local event log. Returns
// nil if neither file exists.
func RevokeConsent(dir string) error {
	var errs []error
	for _, name := range []string{"telemetry.yaml", "telemetry.jsonl", "machine_id"} {
		p := filepath.Join(dir, name)
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func consentPath(dir string) string { return filepath.Join(dir, "telemetry.yaml") }

func hasConsent(dir string) bool {
	_, err := os.Stat(consentPath(dir))
	return err == nil
}

// isExplicitlyDisabled reads PREFLIGHT_TELEMETRY and returns true for "off",
// "0", "false", or "no" (case-insensitive).
func isExplicitlyDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PREFLIGHT_TELEMETRY")))
	switch v {
	case "off", "0", "false", "no", "disable", "disabled":
		return true
	default:
		return false
	}
}

// loadOrCreateMachineID returns a stable, anonymous identifier rooted in a
// random nonce. If reading or creating the file fails, returns "".
func loadOrCreateMachineID(dir string) string {
	path := filepath.Join(dir, "machine_id")
	if existing, err := os.ReadFile(path); err == nil {
		return strings.TrimSpace(string(existing))
	}

	nonce := make([]byte, 32)
	if _, err := io.ReadFull(randReader, nonce); err != nil {
		return ""
	}
	sum := sha256.Sum256(nonce)
	id := hex.EncodeToString(sum[:16])

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(path, []byte(id), 0o600); err != nil {
		return ""
	}
	return id
}
