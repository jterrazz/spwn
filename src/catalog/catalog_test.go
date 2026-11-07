package catalog_test

import (
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
)

func TestLivesOnEveryInstallOnAKindItsMethodAccepts(t *testing.T) {
	for _, tool := range catalog.Apps {
		seen := map[catalog.MachineKind]bool{}
		for _, place := range tool.Installs {
			accepts := catalog.MethodKinds[place.Install.Method()]
			for _, kind := range place.Kinds {
				if seen[kind] {
					t.Errorf("%s: kind %s placed twice", tool.Name, kind)
				}
				seen[kind] = true
				if !slices.Contains(accepts, kind) {
					t.Errorf("%s: method %s does not run on %s", tool.Name, place.Install.Method(), kind)
				}
			}
		}
		if len(seen) == 0 {
			t.Errorf("%s: placed nowhere", tool.Name)
		}
	}
}

func TestUniqueNamesAcrossAppsSkillsAndConfigItems(t *testing.T) {
	owner := map[string]string{}
	claim := func(name, what string) {
		if prev, ok := owner[name]; ok {
			t.Errorf("%q names both a %s and a %s", name, prev, what)
		}
		owner[name] = what
	}
	for _, tool := range catalog.Apps {
		claim(tool.Name, "tool")
	}
	for _, item := range catalog.Settings {
		claim(item.Name, "config item")
	}
	for _, item := range catalog.CheckedSettings {
		claim(item.Name, "checked setting")
	}
}

func TestEveryRequirementAndThenNamesACatalogueEntry(t *testing.T) {
	apps := map[string]bool{}
	for _, tool := range catalog.Apps {
		apps[tool.Name] = true
	}
	items := map[string]bool{}
	for _, item := range catalog.Settings {
		items[item.Name] = true
	}
	for _, tool := range catalog.Apps {
		for _, r := range tool.Requires {
			if !apps[r] {
				t.Errorf("%s requires the unknown tool %q", tool.Name, r)
			}
		}
		for _, then := range tool.Then {
			if !items[then] {
				t.Errorf("%s runs the unknown config item %q", tool.Name, then)
			}
		}
	}
	for _, item := range catalog.Settings {
		for _, r := range item.Needs() {
			if !apps[r] {
				t.Errorf("setting %s requires the unknown tool %q", item.Name, r)
			}
		}
	}
}

type row struct {
	name string
	s    catalog.Disk
}

func rows() []row {
	out := []row{{name: "station", s: catalog.StationDisk}}
	for _, tool := range catalog.Apps {
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
	for _, r := range rows() {
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
	for _, r := range rows() {
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

func TestSystemAppsAreExactlyTheFleetsAndPlacedOnAMac(t *testing.T) {
	var system []string
	for _, tool := range catalog.Apps {
		if !tool.System {
			continue
		}
		system = append(system, tool.Name)
		if !slices.Contains(tool.LivesOn(), catalog.Mac) {
			t.Errorf("%s is a system tool not placed on a host", tool.Name)
		}
	}
	slices.Sort(system)
	if !slices.Equal(system, []string{"git", "homebrew", "mise", "tailscale", "tart"}) {
		t.Errorf("system tools = %v", system)
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
