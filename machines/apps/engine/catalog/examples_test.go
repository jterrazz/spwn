package catalog_test

import (
	"io/fs"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheCatalogHoldsTheSixExamples(t *testing.T) {
	want := []string{"macrohard", "matrix", "paperclip-factory", "research-lab", "severance", "startup"}
	if got := catalog.Examples(); !slices.Equal(got, want) {
		t.Errorf("Examples() = %v; want %v", got, want)
	}
}

func TestAnExampleIsTheFilesOfAProject(t *testing.T) {
	for _, name := range catalog.Examples() {
		t.Run(name, func(t *testing.T) {
			files, ok := catalog.Example(name)
			if !ok {
				t.Fatal("Example is not found")
			}
			for _, p := range []string{"spwn.yaml", "spwn.lock"} {
				if _, err := fs.Stat(files, p); err != nil {
					t.Errorf("missing %s: %v", p, err)
				}
			}
			agents, err := fs.ReadDir(files, "spwn/agents")
			if err != nil || len(agents) == 0 {
				t.Fatalf("spwn/agents holds %v, %v; want an agent", agents, err)
			}
			for _, a := range agents {
				for _, f := range []string{"SOUL.md", "AGENTS.md", "agent.yaml"} {
					if _, err := fs.Stat(files, "spwn/agents/"+a.Name()+"/"+f); err != nil {
						t.Errorf("agent %s lacks %s: %v", a.Name(), f, err)
					}
				}
			}
		})
	}
}

func TestTheSeveranceExampleShipsItsKnowledge(t *testing.T) {
	files, _ := catalog.Example("severance")
	for _, p := range []string{"spwn/knowledge/handbook.md", "spwn/knowledge/raw/note-001.md", "spwn/knowledge/raw/.gitkeep"} {
		if _, err := fs.Stat(files, p); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
}

func TestAnExampleTheCatalogDoesNotHoldIsNotFound(t *testing.T) {
	if files, ok := catalog.Example("nope"); ok || files != nil {
		t.Errorf("Example(nope) = %v, %v; want none", files, ok)
	}
}
