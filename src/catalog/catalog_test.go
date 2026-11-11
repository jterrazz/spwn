package catalog_test

import (
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
)

func builtin(t *testing.T) catalog.Catalog {
	t.Helper()
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type row struct {
	name string
	s    catalog.Disk
}

func rows(t *testing.T) []row {
	out := []row{{name: "station", s: catalog.StationDisk}}
	for _, tool := range builtin(t).Apps {
		if tool.Disk.Declared() {
			out = append(out, row{name: tool.Name, s: tool.Disk})
		}
	}
	return out
}

func inside(pattern, root string) bool {
	root = strings.TrimSuffix(root, "/*")
	pattern = strings.TrimSuffix(pattern, "/*")
	return pattern == root || strings.HasPrefix(pattern, root+"/")
}

func TestStorageDeletesOnlyWhatItsLevelAllows(t *testing.T) {
	for _, r := range rows(t) {
		levels := map[catalog.Level][]string{}
		for _, p := range r.s.Paths {
			levels[p.Level] = append(levels[p.Level], p.Path)
		}
		within := func(pattern string, allowed ...catalog.Level) bool {
			for _, level := range allowed {
				for _, root := range levels[level] {
					if inside(pattern, root) {
						return true
					}
				}
			}
			return false
		}
		check := func(what string, actions []catalog.Action, misplacedOnes []catalog.Misplaced, allowed ...catalog.Level) {
			for _, a := range actions {
				for _, d := range append(slices.Clone(a.Delete), a.Prune...) {
					if strings.HasPrefix(d, "/") {
						t.Errorf("%s: %s deletes the absolute %s", r.name, what, d)
					}
					if within(d, catalog.LevelKept, catalog.LevelSettings) {
						t.Errorf("%s: %s deletes %s, a kept or config path", r.name, what, d)
					}
					if !within(d, allowed...) {
						t.Errorf("%s: %s deletes %s, outside its %v paths", r.name, what, d, allowed)
					}
				}
			}
			for _, st := range misplacedOnes {
				for _, a := range st.Free {
					for _, d := range append(slices.Clone(a.Delete), a.Prune...) {
						if !inside(d, st.Path) {
							t.Errorf("%s: the stray %s cleans %s outside it", r.name, st.Path, d)
						}
					}
				}
			}
		}
		check("clean", r.s.Free, r.s.Misplaced, catalog.LevelCache)
		check("reset", r.s.Erase, nil, catalog.LevelCache, catalog.LevelData)
		if len(r.s.Erase) == 0 && r.s.NoErase == "" {
			t.Errorf("%s: no reset and no NoReset naming who owns its data", r.name)
		}
	}
}

func TestStoragePatternsStartAtARootAndTheHomeRootIsDeclared(t *testing.T) {
	exception := func(p string) bool {
		first, _, _ := strings.Cut(strings.TrimPrefix(p, "~/"), "/")
		switch first {
		case "Library", ".local", ".config", ".cache":
			return true
		}
		for _, e := range catalog.HomeExceptions {
			if ok, _ := path.Match(e.Path, first); ok {
				return true
			}
		}
		return false
	}
	for _, r := range rows(t) {
		misplacedOnes := make([]string, 0, len(r.s.Misplaced))
		for _, st := range r.s.Misplaced {
			misplacedOnes = append(misplacedOnes, st.Path)
		}
		patterns := slices.Clone(r.s.Installed)
		for _, p := range r.s.Paths {
			patterns = append(patterns, p.Path)
		}
		for _, p := range patterns {
			root := strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "/")
			for _, prefix := range []string{"{config}/", "{data}/", "{state}/", "{cache}/"} {
				root = root || strings.HasPrefix(p, prefix)
			}
			if !root {
				t.Errorf("%s: %s starts at no root", r.name, p)
			}
			misplaced := slices.ContainsFunc(misplacedOnes, func(s string) bool { return inside(p, s) })
			if strings.HasPrefix(p, "~/") && !misplaced && !exception(p) {
				t.Errorf("%s: %s sits at the home root, neither a stray nor a declared exception", r.name, p)
			}
		}
	}
}

func TestTheLinuxImageKernelWritesItsConsoleWhereTheHostReadsIt(t *testing.T) {
	recipe, err := catalog.Recipes.ReadFile("images/linux.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []string{
		`GRUB_CMDLINE_LINUX_DEFAULT="$GRUB_CMDLINE_LINUX_DEFAULT console=hvc0"`,
		"update-grub",
		"grep -q 'console=hvc0' /boot/grub/grub.cfg || {",
	} {
		if !strings.Contains(string(recipe), step) {
			t.Errorf("the linux recipe lacks %q: no kernel line would reach the console a cause is read from", step)
		}
	}
}

func TestARequirementLivesWhereItsAppDoes(t *testing.T) {
	// invariant: an install brings what its app requires, so a requirement
	// lives on every kind of machine its app lives on.
	c := builtin(t)
	byName := map[string]catalog.App{}
	for _, a := range c.Apps {
		byName[a.Name] = a
	}
	for _, a := range c.Apps {
		for _, r := range a.Requires {
			for _, kind := range kinds() {
				_, placed := a.InstallOn(kind)
				_, carried := byName[r].InstallOn(kind)
				if placed && !carried {
					t.Errorf("%s lives on %s, where its requirement %s does not", a.Name, kind, r)
				}
			}
		}
	}
}

func kinds() []catalog.MachineKind {
	var out []catalog.MachineKind
	for _, ks := range catalog.MethodKinds {
		for _, k := range ks {
			if !slices.Contains(out, k) {
				out = append(out, k)
			}
		}
	}
	return out
}
