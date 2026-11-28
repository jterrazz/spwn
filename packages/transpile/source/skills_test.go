package source

import (
	"os"
	"path/filepath"
	"testing"
)

func withFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		at := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestARepositorysOwnSkillsFolderRendersNoSkill(t *testing.T) {
	root := scaffoldProject(t, "version: 1\nname: site\nworlds:\n  ship:\n    agents: [neo]\n", map[string]string{"neo": "name: neo\n"})
	withFiles(t, root, map[string]string{
		"skills/notes.md":          "# Notes\n",
		"skills/vendor/readme.txt": "vendored\n",
	})
	src, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(src.Skills) != 0 {
		t.Fatalf("skills = %+v; want none", src.Skills)
	}
	in, err := ToCompileInput(src, "ship")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}
	if len(in.Skills) != 0 {
		t.Fatalf("the render's skills = %+v; want none", in.Skills)
	}

	withFiles(t, root, map[string]string{"skills/review/SKILL.md": "---\nname: review\n---\nReview.\n"})
	src, err = Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	in, err = ToCompileInput(src, "ship")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}
	if len(in.Skills) != 1 || in.Skills[0].Name != "review" {
		t.Fatalf("the render's skills = %+v; want review", in.Skills)
	}
}
