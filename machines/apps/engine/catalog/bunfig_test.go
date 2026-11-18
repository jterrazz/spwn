package catalog

import (
	"strings"
	"testing"
)

const (
	bunGlobal = "/h/.local/share/bun/install/global"
	bunBin    = "/h/.local/share/bun/bin"
	bunCached = "/h/.cache/bun/install"
)

func TestTheBunfigBlockIsWrittenWholeAndAbsolute(t *testing.T) {
	got, changed, conflict := UpsertBunfig("", bunGlobal, bunBin, bunCached)
	want := "# >>> station >>>\n[install]\nglobalDir = \"" + bunGlobal + "\"\nglobalBinDir = \"" + bunBin + "\"\n\n[install.cache]\ndir = \"" + bunCached + "\"\n# <<< station <<<\n"
	if got != want || !changed || conflict != "" {
		t.Fatalf("got %q (changed %v, conflict %q)", got, changed, conflict)
	}
	again, changed, _ := UpsertBunfig("telemetry = false\n"+got, bunGlobal, bunBin, bunCached)
	if changed || again != "telemetry = false\n"+got {
		t.Fatalf("a rerun changed the file: %q", again)
	}
}

func TestAnOwnersInstallTableIsLeftAlone(t *testing.T) {
	own := "[install]\nregistry = \"https://r.example/\"\n"
	got, changed, conflict := UpsertBunfig(own, bunGlobal, bunBin, bunCached)
	if got != own || changed || !strings.Contains(conflict, "has its own [install]") {
		t.Fatalf("got %q (changed %v, conflict %q)", got, changed, conflict)
	}
}
