package catalog_test

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
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

func TestMiseInstallsOnlyARuntime(t *testing.T) {
	// invariant: mise carries the language runtimes of the catalog alone; a
	// command is the system manager's or a release's, so no package
	// names a mise runtime outside this list (ADR-003, ADR-040).
	runtimes := []string{"node", "bun", "go", "python", "java", "rust"}
	for _, a := range builtin(t).Apps {
		for _, p := range a.Installs {
			if m, ok := p.Install.(catalog.MiseInstall); ok && !slices.Contains(runtimes, m.App) {
				t.Errorf("%s installs by mise %q, which is no runtime", a.Name, m.App)
			}
		}
	}
}

func TestAVersionedFormulaIsLookedForInItsKeg(t *testing.T) {
	// outside: Homebrew links no versioned formula (`postgresql@18` is keg-only,
	// `versioned_formula`), so its commands are found only in its keg's bin,
	// which the app's method puts on station's search, on either prefix.
	for _, a := range builtin(t).Apps {
		for _, p := range a.Installs {
			b, ok := p.Install.(catalog.BrewInstall)
			if !ok || !strings.Contains(b.Formula, "@") || len(a.Provides) == 0 {
				continue
			}
			for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
				if keg := prefix + "/opt/" + b.Formula + "/bin"; !slices.Contains(a.Path, keg) {
					t.Errorf("%s installs brew %s, keg-only, and its search path %v lacks %s", a.Name, b.Formula, a.Path, keg)
				}
			}
		}
	}
}

func recipe(t *testing.T, name string) string {
	t.Helper()
	body, err := catalog.Recipes.ReadFile("images/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestTheLinuxImageBootsWithoutWhatAVMNeverUses(t *testing.T) {
	body := recipe(t, "linux.sh")
	for _, step := range []string{
		"touch /etc/cloud/cloud-init.disabled",
		"systemctl mask systemd-networkd-wait-online.service",
		"systemctl mask ModemManager.service multipathd.service multipathd.socket udisks2.service fwupd.service fwupd-refresh.timer motd-news.timer",
		"systemctl enable station-hostkeys.service station-intent.service station-growroot.service",
		"ExecStart=-/usr/bin/growpart /dev/vda 1",
		"ExecStart=-/usr/sbin/resize2fs /dev/vda1",
	} {
		if !strings.Contains(body, step) {
			t.Errorf("the linux recipe lacks %q", step)
		}
	}
	// invariant: unattended-upgrades and the apt timers that run it are a VM's
	// only security updates: the night upgrades a Mac's Machinefile apps. The
	// build may stop the timers for its own boot; it never turns them off.
	for line := range strings.Lines(body) {
		for _, kept := range []string{"unattended-upgrades", "apt-daily"} {
			for _, off := range []string{"disable", "mask", "purge", "remove"} {
				if strings.Contains(line, kept) && strings.Contains(line, off) {
					t.Errorf("the linux recipe turns %s off, a VM's only source of security updates: %s", kept, strings.TrimSpace(line))
				}
			}
		}
	}
}

func TestTheLinuxImageKeepsOneKernelAndWhatItsImagePhaseInstalled(t *testing.T) {
	body := recipe(t, "linux-cleanup.sh")
	for _, step := range []string{
		`if ls "$added" 2>/dev/null | grep -q '^linux-image-'; then`,
		"linux-libc-dev | linux-base | *\"$kernel\"*) ;;",
		`*) [ -e "$added/$p" ] || purge+=("$p") ;;`,
		`apt-get -s purge "${purge[@]}" | awk '$1 == "Purg" {print $2}'`,
		`apt-get purge -y "${purge[@]}"`,
		`grep -q "vmlinuz-$kernel" /boot/grub/grub.cfg`,
	} {
		if !strings.Contains(body, step) {
			t.Errorf("the linux cleanup lacks %q", step)
		}
	}
	if strings.Index(body, "apt-get purge") > strings.Index(body, "apt-get clean") {
		t.Error("the linux cleanup purges after apt-get clean, once the package lists are gone")
	}
	rebuild := `[ -z "${kernel:-}" ] || update-initramfs -u -k "$kernel"`
	if !strings.Contains(body, rebuild) || strings.Index(body, rebuild) < strings.Index(body, "apt-get purge") {
		t.Error("the linux cleanup does not build the kept kernel's initramfs again once the firmware is purged")
	}
	file, err := fs.ReadFile(catalog.BuiltinFiles(), "machines/base-ubuntu.machine.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m, err := catalog.DecodeMachinefile("base-ubuntu.machine.yaml", file)
	if err != nil {
		t.Fatal(err)
	}
	if m.VM == nil || m.VM.Image == nil || !slices.Contains(m.VM.Image.Bake, "linux-kernel") {
		t.Error("base-ubuntu bakes no linux-kernel: its image would keep both of the upstream's kernels")
	}
}
