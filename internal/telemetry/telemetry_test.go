package telemetry

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecorder_DefaultDisabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r := NewRecorder(dir)
	if r.Enabled() {
		t.Fatal("Recorder must be disabled by default (no consent)")
	}
	r.Record(EventInitCompleted)

	// Log file must not exist when disabled.
	if _, err := os.Stat(filepath.Join(dir, "telemetry.jsonl")); !os.IsNotExist(err) {
		t.Errorf("disabled recorder must not create log file, stat err = %v", err)
	}
}

func TestRecorder_RecordsAfterConsent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if _, err := GrantConsent(dir); err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}

	r := NewRecorder(dir)
	if !r.Enabled() {
		t.Fatal("Recorder must be enabled after consent granted")
	}
	r.Record(EventInitCompleted)
	r.Record(EventApplyFirstOK)

	f, err := os.Open(filepath.Join(dir, "telemetry.jsonl"))
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	var events []Event
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Name != EventInitCompleted || events[1].Name != EventApplyFirstOK {
		t.Errorf("events = %+v, want [init.completed, apply.first_success]", events)
	}
	if events[0].MachineID == "" || events[0].MachineID != events[1].MachineID {
		t.Errorf("machine ID must be stable across events: %q vs %q", events[0].MachineID, events[1].MachineID)
	}
}

func TestRecorder_EnvOverrideDisables(t *testing.T) {
	dir := t.TempDir()
	if _, err := GrantConsent(dir); err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}
	t.Setenv("PREFLIGHT_TELEMETRY", "off")

	r := NewRecorder(dir)
	if r.Enabled() {
		t.Fatal("PREFLIGHT_TELEMETRY=off must override consent")
	}
}

func TestRecorder_EnvAcceptsMultipleDisabledValues(t *testing.T) {
	dir := t.TempDir()
	if _, err := GrantConsent(dir); err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}

	for _, v := range []string{"off", "OFF", "0", "false", "False", "no", "DISABLED"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("PREFLIGHT_TELEMETRY", v)
			if NewRecorder(dir).Enabled() {
				t.Errorf("PREFLIGHT_TELEMETRY=%q must disable telemetry", v)
			}
		})
	}
}

func TestRevokeConsent_RemovesEverything(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := GrantConsent(dir); err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}
	NewRecorder(dir).Record(EventInitCompleted)

	if err := RevokeConsent(dir); err != nil {
		t.Fatalf("RevokeConsent: %v", err)
	}

	for _, name := range []string{"telemetry.yaml", "telemetry.jsonl", "machine_id"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s should be removed by RevokeConsent, stat err = %v", name, err)
		}
	}

	// New recorder after revoke must be disabled.
	if NewRecorder(dir).Enabled() {
		t.Error("recorder must be disabled after revoke")
	}
}

func TestRecord_NeverLeaksPathOrPackageName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := GrantConsent(dir); err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}
	r := NewRecorder(dir)

	// Even if a caller passes a path-shaped string, the recorder must reject it
	// (event names are constants from this package). This test is a guard
	// against a future refactor that accepts arbitrary metadata.
	r.Record("/etc/passwd")
	r.Record(EventInitCompleted)

	body, err := os.ReadFile(filepath.Join(dir, "telemetry.jsonl"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if strings.Contains(string(body), "/etc/passwd") {
		t.Errorf("recorder leaked path-shaped event name into log:\n%s", body)
	}
	if !strings.Contains(string(body), EventInitCompleted) {
		t.Errorf("expected %q in log, got:\n%s", EventInitCompleted, body)
	}
}
