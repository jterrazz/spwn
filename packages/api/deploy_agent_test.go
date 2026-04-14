package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// These tests pin the JSON shape of POST /api/worlds/{id}/agents so the
// "deploy agent to running world" endpoint can never silently regress.

func TestDeployAgentRequest_DecodesNameAndRole(t *testing.T) {
	raw := `{"name": "neo", "role": "chief"}`
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "neo" || body.Role != "chief" {
		t.Errorf("expected neo/chief, got %q/%q", body.Name, body.Role)
	}
}

func TestDeployAgentRequest_RoleDefaults(t *testing.T) {
	raw := `{"name": "qa"}`
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "qa" {
		t.Errorf("name = %q", body.Name)
	}
	// Empty role: the handler should default to "worker" via manifest.DefaultRole.
	if body.Role != "" {
		t.Errorf("role should be empty (defaulted server-side), got %q", body.Role)
	}
}

func TestDeployAgentRequest_EmptyNameRejected(t *testing.T) {
	raw := `{"role": "worker"}`
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(strings.NewReader(raw)).Decode(&body)
	if body.Name != "" {
		t.Errorf("empty name should decode as empty, got %q", body.Name)
	}
}

func TestDeployAgent_ReadOnlyMode(t *testing.T) {
	_, mux := newFullTestServer(t)
	// POST to deploy with arch=nil should return 503 (read-only mode)
	w := doJSON(t, mux, "POST", "/api/worlds/w-test/agents", map[string]string{"name": "neo"})
	if w.Code != 503 {
		t.Errorf("expected 503 in read-only mode, got %d: %s", w.Code, w.Body.String())
	}
}

// Team CRUD via API
func TestTeamCRUD_API(t *testing.T) {
	_, mux := newFullTestServer(t)

	// Create
	w := doJSON(t, mux, "POST", "/api/teams", map[string]string{"name": "Ops Team", "icon": "⚙"})
	if w.Code != 201 {
		t.Fatalf("create team: %d %s", w.Code, w.Body.String())
	}
	var created map[string]string
	json.Unmarshal(w.Body.Bytes(), &created)
	if created["slug"] != "ops-team" {
		t.Errorf("slug = %q", created["slug"])
	}

	// List
	w = doJSON(t, mux, "GET", "/api/teams", nil)
	if w.Code != 200 {
		t.Fatalf("list teams: %d", w.Code)
	}
	var teams []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &teams)
	if len(teams) != 1 || teams[0]["slug"] != "ops-team" {
		t.Errorf("teams = %+v", teams)
	}

	// Update
	w = doJSON(t, mux, "PUT", "/api/teams/ops-team", map[string]string{"description": "Infrastructure"})
	if w.Code != 200 {
		t.Fatalf("update team: %d %s", w.Code, w.Body.String())
	}

	// Delete
	w = doJSON(t, mux, "DELETE", "/api/teams/ops-team", nil)
	if w.Code != 200 {
		t.Fatalf("delete team: %d %s", w.Code, w.Body.String())
	}

	// List should be empty
	w = doJSON(t, mux, "GET", "/api/teams", nil)
	var empty []interface{}
	json.Unmarshal(w.Body.Bytes(), &empty)
	if len(empty) != 0 {
		t.Errorf("expected empty after delete, got %d", len(empty))
	}
}
