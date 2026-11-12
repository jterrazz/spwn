package catalog

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// value: the suffix of an overlay's file, `<name>.local.yaml`.

const OverlaySuffix = ".local.yaml"

// value: a Machinefile's name, as its file is named: lower-case words
// joined by dashes, `<whose>-<what>` by convention (`jterrazz-macbook-pro`).
var machinefileName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func IsMachinefileName(name string) bool {
	// value: a name as a Machinefile is named, which a shared source's name
	// is too: no dot, no slash, no leading dash, so it is one folder of the
	// cache and never a flag.
	return machinefileName.MatchString(name)
}

var (
	sizeWord     = regexp.MustCompile(`^[1-9]\d*[MGT]$`)
	durationWord = regexp.MustCompile(`^[1-9]\d*[mhdw]$`)
	versionWord  = regexp.MustCompile(`^\d+(?:\.\d+)*$`)
	vaultName    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
)

// value: the most ports `to: public` names: the internet reaches a machine
// on three ports (Funnel's 443, 8443 and 10000).

const PublicPorts = 3

func DecodeMachinefile(file string, body []byte) (Machinefile, error) {
	var mf Machinefile
	doc, err := strict(file, body, machinefiles, &mf)
	if err != nil {
		return Machinefile{}, err
	}
	if err := checkMachinefile(file, doc, mf); err != nil {
		return Machinefile{}, err
	}
	return mf, nil
}

func MachinefileOf(file string) (name string, overlay bool) {
	// value: the name a Machinefile's file gives it, and whether the file is
	// an overlay.
	base := path.Base(file)
	if name, ok := strings.CutSuffix(base, OverlaySuffix); ok {
		return name, true
	}
	return strings.TrimSuffix(base, ".yaml"), false
}

// invariant: one check per key, in the reading order, each saying the key
// it judges. What needs the catalog or the files a Machinefile extends (an
// item, an option, a method for the system, vm on a computer) is the
// resolution's, never the file's.
func checkMachinefile(file string, doc *yaml.Node, mf Machinefile) error {
	name, overlay := MachinefileOf(file)
	if problem := checkHead(mf, name, overlay); problem != "" {
		key := "name"
		if overlay {
			key = "overrides"
		}
		return refused(file, doc, key, problem)
	}
	if overlay {
		for _, k := range []struct {
			key string
			set bool
		}{{"machine", mf.Machine != nil}, {"vm", mf.VM != nil}, {"extends", len(mf.Extends) > 0}, {"prefer", len(mf.Prefer) > 0}, {"run", mf.Run != nil}} {
			if k.set {
				return refused(file, doc, k.key, "an overlay changes apps, settings, accounts and ports: `"+k.key+"` is its Machinefile's")
			}
		}
	}
	if m := mf.Machine; m != nil {
		if problem := checkMachine(*m); problem != "" {
			return refused(file, doc, "machine", problem)
		}
	}
	if v := mf.VM; v != nil {
		if problem := checkVM(*v); problem != "" {
			return refused(file, doc, "vm", problem)
		}
	}
	for _, e := range mf.Extends {
		if !machinefileName.MatchString(e) || e == name {
			return refused(file, doc, "extends", fmt.Sprintf("extends %q, which is no other Machinefile's name", e))
		}
	}
	if problem := checkPrefer(mf.Prefer); problem != "" {
		return refused(file, doc, "prefer", problem)
	}
	if a := mf.Apps; a != nil {
		if problem := checkApps(*a, overlay); problem != "" {
			return refused(file, doc, "apps", problem)
		}
	}
	if s := mf.Settings; s != nil {
		if problem := checkSettings(*s, overlay); problem != "" {
			return refused(file, doc, "settings", problem)
		}
	}
	if a := mf.Accounts; a != nil {
		if problem := checkAccounts(*a); problem != "" {
			return refused(file, doc, "accounts", problem)
		}
	}
	if p := mf.Ports; p != nil {
		if problem := checkPorts(*p); problem != "" {
			return refused(file, doc, "ports", problem)
		}
	}
	if r := mf.Run; r != nil && strings.TrimSpace(r.Image+r.Boot+r.Upgrade) == "" {
		return refused(file, doc, "run", "run names the lines of a phase: image, boot or upgrade")
	}
	return nil
}

func checkHead(mf Machinefile, name string, overlay bool) string {
	switch {
	case !machinefileName.MatchString(name):
		return fmt.Sprintf("its file names %q: a Machinefile is named in lower-case words joined by dashes", name)
	case overlay && mf.Name != "":
		return "an overlay names the file it changes in overrides, and no name of its own"
	case overlay && mf.Overrides != name:
		return fmt.Sprintf("overrides %q, its file %q", mf.Overrides, name+OverlaySuffix)
	case !overlay && mf.Overrides != "":
		return "only an overlay, `" + name + OverlaySuffix + "`, overrides a Machinefile"
	case !overlay && mf.Name != name:
		return fmt.Sprintf("names %q, its file %q", mf.Name, name)
	}
	return ""
}

func checkMachine(m MachineSpec) string {
	if !known(m.Kind) {
		return fmt.Sprintf("kind %q is none of %s", m.Kind, strings.Join(m.Kind.Enum(), ", "))
	}
	s := m.System
	switch {
	case s == nil:
		return ""
	case s.OS == OSAny || s.OS != "" && !known(s.OS):
		return fmt.Sprintf("os %q is none of macos, ubuntu, debian", s.OS)
	case s.Arch != "" && !known(s.Arch):
		return fmt.Sprintf("arch %q is none of %s", s.Arch, strings.Join(s.Arch.Enum(), ", "))
	case s.Version != "" && !versionWord.MatchString(s.Version):
		return fmt.Sprintf("version %q is no version (`24.04`, `26`)", s.Version)
	case s.Version != "" && s.OS == "":
		return "a version is of an os: name it"
	}
	return ""
}

func checkVM(v VMSpec) string {
	switch {
	case v.Lifetime != "" && !known(v.Lifetime):
		return fmt.Sprintf("lifetime %q is none of %s", v.Lifetime, strings.Join(v.Lifetime.Enum(), ", "))
	case v.Network != "" && !known(v.Network):
		return fmt.Sprintf("network %q is none of %s", v.Network, strings.Join(v.Network.Enum(), ", "))
	}
	if r := v.Resources; r != nil {
		switch {
		case r.CPUs < 0:
			return "cpus is a count, from 1"
		case r.Memory != "" && !sizeWord.MatchString(string(r.Memory)):
			return fmt.Sprintf("memory %q is no size (`32G`, `512M`)", r.Memory)
		case r.Disk != "" && !sizeWord.MatchString(string(r.Disk)):
			return fmt.Sprintf("disk %q is no size (`80G`, `1T`)", r.Disk)
		}
	}
	if i := v.Image; i != nil && slices.Contains(i.Bake, "") {
		return "bake lists items by name"
	}
	return checkMounts(v.Mounts)
}

// invariant: a mount names both of its folders, each a path from the root
// or the home (`~/`), and no two mounts share a folder of the guest.
func checkMounts(mounts []Mount) string {
	guests := map[string]bool{}
	for _, m := range mounts {
		switch {
		case !mountFolder(m.Computer):
			return fmt.Sprintf("mount folder %q of the computer is not a path from / or ~/", m.Computer)
		case !mountFolder(m.Guest):
			return fmt.Sprintf("mount folder %q of the VM is not a path from / or ~/", m.Guest)
		case strings.ContainsAny(m.Computer+m.Guest, ": \t"):
			return fmt.Sprintf("mount %q at %q: a mount's folders hold no space and no colon", m.Computer, m.Guest)
		case guests[path.Clean(m.Guest)]:
			return fmt.Sprintf("two mounts share the guest folder %q", m.Guest)
		}
		guests[path.Clean(m.Guest)] = true
	}
	return ""
}

func mountFolder(p string) bool {
	return p == "~" || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~/")
}

func checkPrefer(prefer []ManagerKey) string {
	for i, k := range prefer {
		if !known(k) {
			return fmt.Sprintf("%q is none of %s", k, strings.Join(k.Enum(), ", "))
		}
		if slices.Contains(prefer[:i], k) {
			return fmt.Sprintf("prefers %s twice", k)
		}
	}
	return ""
}

func checkApps(a AppList, overlay bool) string {
	switch {
	case overlay && !a.Changes:
		return "an overlay's apps are what it adds and removes: `apps: { add: [...], remove: [...] }`"
	case !overlay && a.Changes:
		return "apps is a list; add and remove are an overlay's"
	}
	seen := map[string]bool{}
	for _, e := range slices.Concat(a.List, a.Add) {
		switch {
		case e.Name == "":
			return "an app names its item"
		case seen[e.Name]:
			return "lists " + e.Name + " twice"
		case e.Manager != "" && !known(e.Manager):
			return fmt.Sprintf("%s's manager %q is none of %s", e.Name, e.Manager, strings.Join(e.Manager.Enum(), ", "))
		}
		seen[e.Name] = true
		if problem := checkValues(e.Name, e.Options); problem != "" {
			return problem
		}
	}
	for _, r := range a.Remove {
		if seen[r] {
			return "adds and removes " + r
		}
	}
	return ""
}

func checkSettings(s SettingList, overlay bool) string {
	switch {
	case overlay && !s.Changes:
		return "an overlay's settings are what it sets and removes: `settings: { set: {...}, remove: [...] }`"
	case !overlay && s.Changes:
		return "settings is a mapping by setting; set and remove are an overlay's"
	}
	for _, name := range sortedKeys(s.Set) {
		e := s.Set[name]
		if e.From != "" && e.From != FromComputer {
			return fmt.Sprintf("%s takes its values from %q: only `from: computer` is known", name, e.From)
		}
		if slices.Contains(s.Remove, name) {
			return "sets and removes " + name
		}
		if problem := checkValues(name, e.Values); problem != "" {
			return problem
		}
	}
	return ""
}

// invariant: a `vault:` value names a secret of the vault, never holds one.
func checkValues(item string, values map[string]OptionValue) string {
	for _, k := range sortedKeys(values) {
		if ref, ok := strings.CutPrefix(string(values[k]), VaultPrefix); ok && !vaultName.MatchString(ref) {
			return fmt.Sprintf("%s's %s: %q names no secret (`vault:<name>`)", item, k, values[k])
		}
	}
	return ""
}

func checkAccounts(a AccountsSpec) string {
	seen := map[string]bool{}
	for _, s := range a.Share {
		switch {
		case s.Account == "":
			return "a share names its account"
		case seen[s.Account]:
			return "shares " + s.Account + " twice"
		case s.For != "" && !durationWord.MatchString(s.For):
			return fmt.Sprintf("%s for %q: a term is a count and m, h, d or w (`7d`)", s.Account, s.For)
		}
		seen[s.Account] = true
	}
	for i, k := range a.Keep {
		if k == "" || slices.Contains(a.Keep[:i], k) {
			return "keep lists each account once, by name"
		}
	}
	return ""
}

func checkPorts(p PortRights) string {
	switch {
	case len(p.Open) == 0 && (p.To != "" || len(p.Except) > 0):
		return "names no ports it may open (open)"
	case len(p.Open) > 0 && p.To == "":
		return "says not to whom its ports are open (to: local, private or public)"
	case p.To != "" && !known(p.To):
		return fmt.Sprintf("to %q is none of %s", p.To, strings.Join(p.To.Enum(), ", "))
	}
	for _, s := range p.Open {
		if s.From < 1 || s.To > 65535 || s.From > s.To {
			return fmt.Sprintf("%s: a port from 1 to 65535, a range lowest first", s)
		}
		if p.To == AudiencePublic && s.From != s.To {
			return fmt.Sprintf("public opens ports one by one, never a range (%s)", s)
		}
	}
	if p.To == AudiencePublic && len(p.Open) > PublicPorts {
		return fmt.Sprintf("public opens %d ports at most", PublicPorts)
	}
	for _, e := range p.Except {
		if e < 1 || e > 65535 {
			return fmt.Sprintf("except %d: a port from 1 to 65535", e)
		}
	}
	return ""
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
