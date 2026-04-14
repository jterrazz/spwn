package mind

import "testing"

func TestAddPlugin_Idempotent(t *testing.T) {
	initAgent(t, "neo")

	if err := AddPlugin("neo", "@spwn/mempalace"); err != nil {
		t.Fatalf("AddPlugin: %v", err)
	}
	if err := AddPlugin("neo", "@spwn/mempalace"); err != nil {
		t.Fatalf("AddPlugin (second): %v", err)
	}

	m, err := LoadManifest("neo")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Plugins) != 1 || m.Plugins[0] != "@spwn/mempalace" {
		t.Errorf("Plugins = %v, want [@spwn/mempalace]", m.Plugins)
	}
}

func TestRemovePlugin_Idempotent(t *testing.T) {
	initAgent(t, "neo")

	if err := AddPlugin("neo", "@spwn/mempalace"); err != nil {
		t.Fatalf("AddPlugin: %v", err)
	}
	if err := AddPlugin("neo", "@spwn/other"); err != nil {
		t.Fatalf("AddPlugin: %v", err)
	}

	if err := RemovePlugin("neo", "@spwn/mempalace"); err != nil {
		t.Fatalf("RemovePlugin: %v", err)
	}
	if err := RemovePlugin("neo", "@spwn/mempalace"); err != nil {
		t.Fatalf("RemovePlugin (second): %v", err)
	}

	m, err := LoadManifest("neo")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Plugins) != 1 || m.Plugins[0] != "@spwn/other" {
		t.Errorf("Plugins after remove = %v, want [@spwn/other]", m.Plugins)
	}
}
