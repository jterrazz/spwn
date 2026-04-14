package labels

import (
	"testing"
	"time"

	"spwn.sh/packages/world/internal/models"
)

func TestApplyTestRun(t *testing.T) {
	t.Run("noop when env var unset", func(t *testing.T) {
		t.Setenv(TestRunEnv, "")
		m := map[string]string{"keep": "me"}
		ApplyTestRun(m)
		if _, ok := m[TestRun]; ok {
			t.Fatalf("expected no TestRun label when env var unset")
		}
		if m["keep"] != "me" {
			t.Fatalf("unrelated keys must survive: %v", m)
		}
	})

	t.Run("stamps label when env var set", func(t *testing.T) {
		t.Setenv(TestRunEnv, "abc123")
		m := map[string]string{}
		ApplyTestRun(m)
		if m[TestRun] != "abc123" {
			t.Fatalf("expected TestRun=abc123, got %q", m[TestRun])
		}
	})

	t.Run("WorldLabels picks up the env var", func(t *testing.T) {
		t.Setenv(TestRunEnv, "run-xyz")
		w := models.World{ID: "w-1", Config: "default", CreatedAt: time.Now()}
		lbls := WorldLabels(w)
		if lbls[TestRun] != "run-xyz" {
			t.Fatalf("WorldLabels must propagate SPWN_TEST_LABEL: %v", lbls)
		}
	})
}

func TestWorldLabels_RoundTrip(t *testing.T) {
	created := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	original := models.World{
		ID:           "w-default-12345",
		Name:         "default",
		Config:       "default",
		Agent:        "neo",
		AgentID:      "a-neo-99999",
		Organization: "acme",
		CreatedAt:    created,
		Workspaces: []models.Workspace{
			{Name: "default", Path: "/Users/x/proj"},
			{Name: "api", Path: "/Users/x/proj/api"},
		},
		Agents: []models.AgentRecord{
			{Name: "neo", AgentID: "a-neo-99999", Role: "worker", Status: models.StatusIdle},
			{Name: "morpheus", AgentID: "a-morph-77777", Role: "chief", Status: models.StatusIdle},
		},
	}

	lbls := WorldLabels(original)
	if lbls[KindKey] != KindWorld {
		t.Errorf("expected kind=world, got %q", lbls[KindKey])
	}

	parsed, err := ParseWorld(lbls)
	if err != nil {
		t.Fatalf("ParseWorld failed: %v", err)
	}

	// Field-by-field comparison (avoid reflect.DeepEqual on time fields).
	if parsed.ID != original.ID {
		t.Errorf("ID: got %q want %q", parsed.ID, original.ID)
	}
	if parsed.Name != original.Name {
		t.Errorf("Name: got %q want %q", parsed.Name, original.Name)
	}
	if parsed.Config != original.Config {
		t.Errorf("Config: got %q want %q", parsed.Config, original.Config)
	}
	if parsed.Agent != original.Agent {
		t.Errorf("Agent: got %q want %q", parsed.Agent, original.Agent)
	}
	if parsed.AgentID != original.AgentID {
		t.Errorf("AgentID: got %q want %q", parsed.AgentID, original.AgentID)
	}
	if parsed.Organization != original.Organization {
		t.Errorf("Organization: got %q want %q", parsed.Organization, original.Organization)
	}
	if !parsed.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt: got %v want %v", parsed.CreatedAt, created)
	}
	if len(parsed.Workspaces) != 2 || parsed.Workspaces[0].Name != "default" || parsed.Workspaces[1].Path != "/Users/x/proj/api" {
		t.Errorf("Workspaces did not round-trip: %#v", parsed.Workspaces)
	}
	if len(parsed.Agents) != 2 || parsed.Agents[0].Name != "neo" || parsed.Agents[1].Role != "chief" {
		t.Errorf("Agents did not round-trip: %#v", parsed.Agents)
	}
}

func TestWorldLabels_OmitsEmpty(t *testing.T) {
	w := models.World{
		ID:        "w-bare-00001",
		Config:    "bare",
		CreatedAt: time.Now(),
	}
	lbls := WorldLabels(w)
	if _, ok := lbls[WorldName]; ok {
		t.Error("WorldName should be omitted when empty")
	}
	if _, ok := lbls[WorldAgent]; ok {
		t.Error("WorldAgent should be omitted when empty")
	}
	if _, ok := lbls[WorldWorkspaces]; ok {
		t.Error("WorldWorkspaces should be omitted when empty")
	}
	if _, ok := lbls[WorldAgents]; ok {
		t.Error("WorldAgents should be omitted when empty")
	}
}

func TestParseWorld_RejectsNonSpwnContainers(t *testing.T) {
	if _, err := ParseWorld(nil); err == nil {
		t.Error("expected error for nil labels")
	}
	if _, err := ParseWorld(map[string]string{"app": "other"}); err == nil {
		t.Error("expected error for non-spwn container")
	}
	if _, err := ParseWorld(map[string]string{KindKey: "world"}); err == nil {
		t.Error("expected error for missing world id")
	}
}

func TestIsWorld_IsArchitect(t *testing.T) {
	world := map[string]string{KindKey: KindWorld}
	arch := map[string]string{KindKey: KindArchitect}
	other := map[string]string{KindKey: "something"}

	if !IsWorld(world) || IsWorld(arch) || IsWorld(other) || IsWorld(nil) {
		t.Error("IsWorld misclassified labels")
	}
	if !IsArchitect(arch) || IsArchitect(world) || IsArchitect(other) || IsArchitect(nil) {
		t.Error("IsArchitect misclassified labels")
	}
}
