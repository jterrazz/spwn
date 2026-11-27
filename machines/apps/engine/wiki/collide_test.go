package wiki_test

import (
	"fmt"
	"hash/fnv"
	"maps"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

func TestTwoFilesThatAreOnePageIdAreBothServedTheSecondUnderItsPathMark(t *testing.T) {
	files := corpusFiles(t)
	bell, override := "blueprint/a\ab", "blueprint/a"+string(rune(0x202e))+"b"
	files["blueprint/ab.md"] = []byte("# The honest one\n\nIt is first.\n")
	files[bell+".md"] = []byte("# The bell\n\nIt rings.\n")
	files[override+".md"] = []byte("# The override\n\nIt turns.\n")
	c, err := readAll(files)
	if err != nil {
		t.Fatal(err)
	}
	mark := func(file string) string {
		h := fnv.New32a()
		_, _ = h.Write([]byte(file))
		return fmt.Sprintf("blueprint/ab~%08x", h.Sum32())
	}
	served := map[string]string{"blueprint/ab": "The honest one", mark(bell): "The bell", mark(override): "The override"}
	t.Logf("served as %q", slices.Sorted(maps.Keys(served)))
	for id, title := range served {
		s, refused := c.Serve(id)
		if refused != nil {
			t.Fatalf("%s: %v", id, refused)
		}
		hard := 1
		if id == "blueprint/ab" {
			hard = 0
		}
		if s.Page.ID != id || s.Page.Word != title || s.Page.Hard != hard {
			t.Errorf("%s served as %+v; want its title and %d hard", id, s.Page, hard)
		}
		if bad := holdsSilent(t, s); bad != "" {
			t.Errorf("%s serves %q", id, bad)
		}
	}
	want := []string{
		fmt.Sprintf(`%s the files "blueprint/ab.md" and "blueprint/a\ab.md" are both the page "blueprint/ab": the second is served as %q`, mark(bell), mark(bell)),
		fmt.Sprintf(`%s the files "blueprint/ab.md" and "blueprint/a\`+`u202eb.md" are both the page "blueprint/ab": the second is served as %q`, mark(override), mark(override)),
	}
	slices.Sort(want)
	if got := ruled(c.Report, "H3"); !slices.Equal(got, want) {
		t.Errorf("H3 findings\n%q\nwant\n%q", got, want)
	}
	m, err := c.Map("")
	if err != nil {
		t.Fatal(err)
	}
	for id := range served {
		if !slices.ContainsFunc(m.Nodes, func(n wiki.MapNode) bool { return n.ID == "P:"+id }) {
			t.Errorf("wiki.map holds no node P:%s", id)
		}
	}
}
