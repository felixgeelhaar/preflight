package anthropic

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSystemBlock_MarshalsCacheControl ensures the request body emits
// `cache_control: {type: "ephemeral"}` on the system block, which is the
// signal Anthropic uses to prefix-cache the static system prompt.
func TestSystemBlock_MarshalsCacheControl(t *testing.T) {
	t.Parallel()

	req := messagesRequest{
		Model:     DefaultModel,
		MaxTokens: 100,
		System: []systemBlock{
			newCacheableSystemBlock("you are a helpful advisor"),
		},
		Messages: []message{{Role: "user", Content: "hi"}},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	body := string(raw)
	if !strings.Contains(body, `"cache_control":{"type":"ephemeral"}`) {
		t.Errorf("system block missing ephemeral cache_control marker:\n%s", body)
	}
	if !strings.Contains(body, `"system":[`) {
		t.Errorf("system field must encode as an array of blocks, not a string:\n%s", body)
	}
}

// TestNewCacheableSystemBlock_ShapeIsCorrect prevents accidental drift in the
// block shape (must be type=text + ephemeral cache_control).
func TestNewCacheableSystemBlock_ShapeIsCorrect(t *testing.T) {
	t.Parallel()

	b := newCacheableSystemBlock("hello")
	if b.Type != "text" {
		t.Errorf("Type = %q, want %q", b.Type, "text")
	}
	if b.Text != "hello" {
		t.Errorf("Text = %q, want %q", b.Text, "hello")
	}
	if b.CacheControl == nil || b.CacheControl.Type != "ephemeral" {
		t.Errorf("CacheControl = %+v, want ephemeral", b.CacheControl)
	}
}

func TestDefaultModel_NotStale(t *testing.T) {
	t.Parallel()
	// Guard against the previous 14-month-stale default sneaking back in.
	if DefaultModel == "claude-3-5-sonnet-20241022" {
		t.Errorf("DefaultModel must not be the legacy stale default")
	}
	if !strings.HasPrefix(DefaultModel, "claude-") {
		t.Errorf("DefaultModel = %q, want a claude-* identifier", DefaultModel)
	}
}
