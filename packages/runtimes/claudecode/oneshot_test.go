package claudecode

import "testing"

// Tests for the talk-time one-shot interface that got lifted out of
// apps/cli/agent/talk.go when codex became a first-class runtime.
// The inline logic used to live there; now each adapter owns its own
// flag synthesis and output parsing.

func TestOneShotFlags_default(t *testing.T) {
	base := []string{"claude", "--dangerously-skip-permissions", "-p", "hello"}
	got := Spawner.OneShotFlags(base, "")
	want := []string{
		"claude", "--dangerously-skip-permissions", "-p", "hello",
		"--print", "--output-format", "json",
	}
	assertEqStrings(t, got, want)
}

func TestOneShotFlags_streamJSON(t *testing.T) {
	base := []string{"claude", "-p", "hi"}
	got := Spawner.OneShotFlags(base, "stream-json")
	want := []string{"claude", "-p", "hi", "--output-format", "stream-json", "--verbose"}
	assertEqStrings(t, got, want)
}

func TestOneShotFlags_unknownFormatFallsThrough(t *testing.T) {
	// Any unknown format (e.g. "text") resolves to the JSON envelope
	// path so ParseOneShotResult always has a predictable shape to
	// work against.
	base := []string{"claude"}
	got := Spawner.OneShotFlags(base, "text")
	if got[len(got)-2] != "--output-format" || got[len(got)-1] != "json" {
		t.Errorf("expected fallback to --output-format json, got %v", got)
	}
}

func TestParseOneShotResult_happy(t *testing.T) {
	raw := []byte(`{"type":"result","subtype":"success","result":"hello world","session_id":"sess-abc","total_cost_usd":0.01}`)
	text, sid, err := Spawner.ParseOneShotResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello world" {
		t.Errorf("text = %q, want %q", text, "hello world")
	}
	if sid != "sess-abc" {
		t.Errorf("sessionID = %q, want %q", sid, "sess-abc")
	}
}

func TestParseOneShotResult_emptyFieldsAccepted(t *testing.T) {
	// A valid envelope with blank fields is not an error — blank text
	// can mean "prompt produced no reply", blank session_id means the
	// runtime didn't return one. Caller decides whether that's useful.
	raw := []byte(`{"result":"","session_id":""}`)
	text, sid, err := Spawner.ParseOneShotResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" || sid != "" {
		t.Errorf("expected blanks, got text=%q sid=%q", text, sid)
	}
}

func TestParseOneShotResult_malformedIsError(t *testing.T) {
	raw := []byte(`not json at all`)
	_, _, err := Spawner.ParseOneShotResult(raw)
	if err == nil {
		t.Fatal("expected error on non-JSON input")
	}
}

func TestParseOneShotResult_unknownFieldsIgnored(t *testing.T) {
	// Future Claude Code releases can add fields without breaking us.
	raw := []byte(`{"result":"ok","session_id":"s","new_field_from_future":42,"another":{"nested":true}}`)
	text, sid, err := Spawner.ParseOneShotResult(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "ok" || sid != "s" {
		t.Errorf("text/sid = %q/%q", text, sid)
	}
}

func assertEqStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d, want %d\n  got=%v\n  want=%v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
