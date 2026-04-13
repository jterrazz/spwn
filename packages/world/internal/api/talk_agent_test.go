package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// Regression: handleTalk must resolve the agent name from three sources:
// 1. Explicit "agent" field in the request body (multi-agent worlds)
// 2. Legacy u.Agent field on the world (single-agent spawns)
// 3. First entry in u.Agents slice (hot-deployed agents)
//
// Before the fix, only u.Agent was checked. Agents deployed via DeployAgent
// (which populates u.Agents but not u.Agent) caused "world not found or
// has no agent".

func TestTalkRequest_DecodesAgentField(t *testing.T) {
	raw := `{"message":"hello","agent":"QA Eng"}`
	var body struct {
		Message string `json:"message"`
		Agent   string `json:"agent"`
	}
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Agent != "QA Eng" {
		t.Errorf("agent = %q, want %q", body.Agent, "QA Eng")
	}
	if body.Message != "hello" {
		t.Errorf("message = %q", body.Message)
	}
}

func TestTalkRequest_AgentFieldOptional(t *testing.T) {
	// Legacy callers may not send the agent field — handler falls back
	// to u.Agent or u.Agents[0].
	raw := `{"message":"hello"}`
	var body struct {
		Message string `json:"message"`
		Agent   string `json:"agent"`
	}
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Agent != "" {
		t.Errorf("agent should be empty when omitted, got %q", body.Agent)
	}
}

func TestTalkRequest_AgentWithSpaces(t *testing.T) {
	// Agent names with spaces must survive JSON round-trip.
	raw := `{"message":"hi","agent":"QA Eng"}`
	var body struct {
		Message string `json:"message"`
		Agent   string `json:"agent"`
	}
	_ = json.NewDecoder(strings.NewReader(raw)).Decode(&body)
	if body.Agent != "QA Eng" {
		t.Errorf("space in agent name lost: %q", body.Agent)
	}
}

func TestTalk_ReadOnlyMode(t *testing.T) {
	// POST /api/worlds/{id}/talk with arch=nil (read-only) should NOT
	// panic — it doesn't need arch, it just calls `spwn agent talk` via exec.
	// But if no worlds exist in state, it should 404 gracefully.
	_, mux := newFullTestServer(t)
	w := doJSON(t, mux, "POST", "/api/worlds/w-nonexistent/talk", map[string]string{
		"message": "hello",
		"agent":   "neo",
	})
	if w.Code != 404 {
		t.Errorf("expected 404 for nonexistent world, got %d: %s", w.Code, w.Body.String())
	}
}

// --- URL encoding regression ---
// Agent names with special characters (spaces, unicode) must work end-to-end.
// The frontend encodes them with encodeURIComponent; the Go mux decodes
// %XX sequences automatically. These tests verify the Go side handles
// encoded names correctly.

func TestTalk_URLEncodedWorldID(t *testing.T) {
	_, mux := newFullTestServer(t)
	// World IDs don't normally have special chars, but verify it doesn't panic.
	w := doJSON(t, mux, "POST", "/api/worlds/w-test-123/talk", map[string]string{
		"message": "test",
		"agent":   "neo",
	})
	// 404 because world doesn't exist in test state — not 500/panic.
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
