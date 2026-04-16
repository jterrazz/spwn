// Package validate is the rule engine for spwn projects.
//
// Rules are pure functions: they take an Input and return zero or more
// Issues. Run calls every rule and returns the collected issues so a
// single `spwn check` invocation produces the full picture of what's
// wrong, not just the first failure.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	intmanifest "spwn.sh/packages/project/internal/manifest"
	"spwn.sh/packages/project/internal/resolve"
	"spwn.sh/packages/dependency"

	"gopkg.in/yaml.v3"
)

// Level ranks the severity of an Issue.
type Level int

const (
	// LevelError means the project cannot be used as-is. spwn up
	// would fail, or worse, silently do the wrong thing.
	LevelError Level = iota

	// LevelWarning means the project works but is missing something
	// the user probably wants.
	LevelWarning

	// LevelInfo is advisory - best-practice suggestions.
	LevelInfo
)

// String returns "error" / "warning" / "info" for display purposes.
func (l Level) String() string {
	switch l {
	case LevelError:
		return "error"
	case LevelWarning:
		return "warning"
	case LevelInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Issue is one finding from a rule.
type Issue struct {
	Level   Level
	Path    string
	Message string
	Hint    string
}

// AgentRef is the validator's view of an agent reference, mirroring
// the public manifest.AgentRef but kept local so the validate package
// doesn't import the public manifest package (which would cycle).
type AgentRef struct {
	Name   string
	Path   string
	Exists bool
}

// Input is the data every rule operates on.
type Input struct {
	Root     string
	Manifest *intmanifest.Manifest

	// AgentRefs are the deployable agents (referenced by some world).
	AgentRefs []AgentRef

	// OrphanRefs are agent directories on disk not referenced by any
	// world. Surfaced as info-level issues.
	OrphanRefs []AgentRef

	// BuiltinTools is the authoritative catalog of known built-in
	// dependencies (tools, skills, runtimes). Nil → fall back to
	// @spwn/* prefix heuristic.
	BuiltinTools []string

	// SupportedRuntimes is the list of runtime backends the host can
	// actually spawn (e.g. "@spwn/claude-code"). Nil → don't check.
	SupportedRuntimes []string
}

// Run executes every rule against the input and returns all issues.
func Run(in Input) []Issue {
	var out []Issue
	rules := []func(Input) []Issue{
		ruleManifestVersion,
		ruleManifestName,
		ruleWorldsMap,
		ruleWorldNames,
		ruleReservedWorldNames,
		ruleAgentDirsExist,
		ruleAgentStructure,
		ruleAgentYAMLParses,
		ruleReservedAgentNames,
		ruleOneAgentOneWorld,
		ruleWorkspaceMounts,
		rulePackVersionConflict,
		ruleRuntimeBackendConflict,
		rulePacksExist,
		ruleLockfileConsistent,
		ruleRuntimeSupported,
		ruleMarkdownImports,
		ruleOrphanAgents,
	}
	for _, r := range rules {
		out = append(out, r(in)...)
	}
	return out
}

var (
	slugRe      = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	projectName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

// IsValidAgentName reports whether the given name is a valid agent
// name — i.e. a slug matching the same regex the manifest enforces
// for world names (`^[a-z][a-z0-9-]*$`). Empty names are invalid.
// Callers use this to reject invalid names at creation time, before
// anything is written to disk.
func IsValidAgentName(name string) bool {
	return name != "" && slugRe.MatchString(name)
}

// IsValidProjectName reports whether the given name matches the
// manifest's project-name regex (`^[a-z0-9][a-z0-9-]*$`). Empty
// names are invalid.
func IsValidProjectName(name string) bool {
	return name != "" && projectName.MatchString(name)
}

// reservedAgentSubcommands is the set of names that collide with
// `spwn agent <subcommand>` and therefore cannot be used as agent
// names.
var reservedAgentSubcommands = map[string]struct{}{
	"create": {}, "new": {}, "ls": {}, "rm": {}, "fork": {}, "inspect": {},
	"logs": {}, "add": {}, "remove": {}, "talk": {}, "send": {}, "inbox": {},
	"watch": {}, "dream": {}, "sleep": {}, "publish": {}, "get": {},
	"export": {}, "import": {}, "start": {}, "stop": {}, "delete": {},
	"deploy": {}, "compose": {}, "list": {}, "init": {},
}

// IsReservedAgentName reports whether the given name would collide
// with a `spwn agent <subcommand>` subcommand. Used by the CLI to
// reject such names at creation time, before the agent directory is
// ever written.
func IsReservedAgentName(name string) bool {
	_, ok := reservedAgentSubcommands[name]
	return ok
}

// ReservedAgentNames returns the sorted list of agent names that
// collide with subcommands of `spwn agent`. The slice is a fresh
// copy the caller can mutate.
func ReservedAgentNames() []string {
	out := make([]string, 0, len(reservedAgentSubcommands))
	for k := range reservedAgentSubcommands {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// reservedWorldSubcommands is the equivalent for `spwn world ...`.
var reservedWorldSubcommands = map[string]struct{}{
	"create": {}, "start": {}, "stop": {}, "ls": {}, "rm": {}, "inspect": {},
	"logs": {}, "enter": {}, "destroy": {}, "rename": {}, "knowledge": {},
	"new": {}, "list": {}, "remove": {}, "up": {}, "down": {}, "snap": {},
}

// ----- Rules -----

func ruleManifestVersion(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	if in.Manifest.Version != intmanifest.CurrentVersion {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#version",
			Message: fmt.Sprintf("unsupported manifest version %d", in.Manifest.Version),
			Hint:    fmt.Sprintf("set version: %d", intmanifest.CurrentVersion),
		}}
	}
	return nil
}

func ruleManifestName(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	if in.Manifest.Name == "" {
		return []Issue{{
			Level: LevelError, Path: "spwn.yaml#name",
			Message: "name is required",
			Hint:    "set name: to a slug like my-project",
		}}
	}
	if !projectName.MatchString(in.Manifest.Name) {
		return []Issue{{
			Level: LevelError, Path: "spwn.yaml#name",
			Message: "name must match ^[a-z0-9][a-z0-9-]*$",
			Hint:    "use lowercase letters, digits, and dashes only",
		}}
	}
	return nil
}

// ruleWorldsMap requires at least one world entry — unless the project
// has zero agents on disk too (a brand-new init that hasn't created
// anything yet).
func ruleWorldsMap(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	if len(in.Manifest.Worlds) > 0 {
		return nil
	}
	if len(in.AgentRefs) == 0 && len(in.OrphanRefs) == 0 {
		return nil
	}
	return []Issue{{
		Level: LevelError, Path: "spwn.yaml#worlds",
		Message: "no worlds declared but agents exist on disk",
		Hint:    "add a worlds: entry that references at least one agent",
	}}
}

func ruleWorldNames(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for _, name := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[name]
		if !slugRe.MatchString(name) {
			out = append(out, Issue{
				Level: LevelError, Path: "spwn.yaml#worlds." + name,
				Message: fmt.Sprintf("world name %q must match ^[a-z][a-z0-9-]*$", name),
			})
		}
		if len(w.Agents) == 0 {
			out = append(out, Issue{
				Level: LevelError, Path: "spwn.yaml#worlds." + name + ".agents",
				Message: "world must declare at least one agent",
			})
		}
		if len(w.Workspaces) == 0 {
			out = append(out, Issue{
				Level: LevelError, Path: "spwn.yaml#worlds." + name + ".workspaces",
				Message: "world must declare at least one workspace",
				Hint:    "add `workspaces: [.]` to mount the project root",
			})
		}
	}
	return out
}

func ruleReservedWorldNames(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for _, name := range sortedKeys(in.Manifest.Worlds) {
		if _, ok := reservedWorldSubcommands[name]; ok {
			out = append(out, Issue{
				Level: LevelError, Path: "spwn.yaml#worlds." + name,
				Message: fmt.Sprintf("world name %q collides with `spwn world %s` subcommand", name, name),
				Hint:    "rename the world to something that doesn't shadow a subcommand",
			})
		}
	}
	return out
}

func ruleAgentDirsExist(in Input) []Issue {
	var out []Issue
	for _, a := range in.AgentRefs {
		if a.Exists {
			continue
		}
		out = append(out, Issue{
			Level: LevelError, Path: relPath(in.Root, a.Path),
			Message: "agent directory not found: " + a.Name,
			Hint:    "create it with `spwn agent new " + a.Name + "`",
		})
	}
	return out
}

func ruleAgentStructure(in Input) []Issue {
	required := []struct {
		rel   string
		isDir bool
		level Level
	}{
		{"agent.yaml", false, LevelError},
		{"AGENTS.md", false, LevelError},
		{filepath.Join("identity", "profile.md"), false, LevelError},
		{"identity", true, LevelWarning},
		{"skills", true, LevelWarning},
		{"knowledge", true, LevelWarning},
		{"playbooks", true, LevelWarning},
		{"journal", true, LevelWarning},
	}
	var out []Issue
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		for _, r := range required {
			full := filepath.Join(a.Path, r.rel)
			info, err := os.Stat(full)
			if err != nil {
				out = append(out, Issue{
					Level: r.level, Path: relPath(in.Root, full),
					Message: "missing " + r.rel,
					Hint:    "regenerate with `spwn agent new " + a.Name + " --force`",
				})
				continue
			}
			if r.isDir && !info.IsDir() {
				out = append(out, Issue{
					Level: r.level, Path: relPath(in.Root, full),
					Message: r.rel + " is not a directory",
				})
			}
			if !r.isDir && info.IsDir() {
				out = append(out, Issue{
					Level: r.level, Path: relPath(in.Root, full),
					Message: r.rel + " should be a file, found directory",
				})
			}
		}
	}
	return out
}

// agentYAML is the validator's local view of agent.yaml. Just enough
// to drive rules.
//
// Deps replaces the old Tools/Dependencies/Skills trichotomy under the
// unified package model — see packages/agent/manifest.go.
type agentYAML struct {
	Name     string   `yaml:"name"`
	Deps []string `yaml:"dependencies"`
	Runtime  struct {
		Backend string `yaml:"backend"`
	} `yaml:"runtime"`
}

func loadAgentYAML(dir string) (*agentYAML, error) {
	data, err := os.ReadFile(filepath.Join(dir, "agent.yaml"))
	if err != nil {
		return nil, err
	}
	var a agentYAML
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func ruleAgentYAMLParses(in Input) []Issue {
	var out []Issue
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		yamlPath := filepath.Join(a.Path, "agent.yaml")
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // already reported by ruleAgentStructure
			}
			out = append(out, Issue{
				Level: LevelError, Path: relPath(in.Root, yamlPath),
				Message: fmt.Sprintf("cannot read agent.yaml: %v", err),
			})
			continue
		}
		var parsed agentYAML
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			out = append(out, Issue{
				Level: LevelError, Path: relPath(in.Root, yamlPath),
				Message: "agent.yaml is not valid YAML: " + err.Error(),
			})
			continue
		}
		if parsed.Name != "" && parsed.Name != a.Name {
			out = append(out, Issue{
				Level: LevelWarning, Path: relPath(in.Root, yamlPath) + "#name",
				Message: fmt.Sprintf("agent.yaml name %q does not match directory name %q", parsed.Name, a.Name),
			})
		}
	}
	return out
}

func ruleReservedAgentNames(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	seen := map[string]bool{}
	for _, name := range in.Manifest.AllAgentNames() {
		if seen[name] {
			continue
		}
		seen[name] = true
		if _, ok := reservedAgentSubcommands[name]; ok {
			out = append(out, Issue{
				Level: LevelError, Path: "spwn.yaml#worlds",
				Message: fmt.Sprintf("agent name %q collides with `spwn agent %s` subcommand", name, name),
			})
		}
	}
	return out
}

// ruleOneAgentOneWorld enforces that the same agent name does not
// appear in more than one worlds[*].agents list.
func ruleOneAgentOneWorld(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	first := map[string]string{} // agent → first world
	var out []Issue
	for _, wname := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[wname]
		for _, a := range w.Agents {
			if prev, ok := first[a]; ok {
				out = append(out, Issue{
					Level: LevelError, Path: "spwn.yaml#worlds." + wname + ".agents",
					Message: fmt.Sprintf("agent %q already deployed by world %q", a, prev),
					Hint:    "an agent may only belong to a single world; remove the duplicate",
				})
				continue
			}
			first[a] = wname
		}
	}
	return out
}

// ruleWorkspaceMounts enforces the workspace mount form. Accepted
// entries:
//
//   - "path"           (bare host path — auto-named workspace<N>)
//   - "name=path"      (named mount; container-side becomes /workspaces/<name>)
//   - "name=path:ro"   (same, read-only)
//
// Container paths never appear in the manifest. Users don't write
// `/workspaces/...` — that's an implementation detail of where the
// mount lands inside the container.
func ruleWorkspaceMounts(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for _, wname := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[wname]
		for _, entry := range w.Workspaces {
			raw := strings.TrimSuffix(entry, ":ro")
			// If there's an `=`, the left side must be a valid name.
			if eq := strings.Index(raw, "="); eq > 0 {
				name := strings.TrimSpace(raw[:eq])
				path := strings.TrimSpace(raw[eq+1:])
				if name == "" || path == "" {
					out = append(out, Issue{
						Level: LevelError, Path: "spwn.yaml#worlds." + wname + ".workspaces",
						Message: fmt.Sprintf("workspace entry %q has empty name or path", entry),
					})
					continue
				}
				if !slugRe.MatchString(name) {
					out = append(out, Issue{
						Level: LevelError, Path: "spwn.yaml#worlds." + wname + ".workspaces",
						Message: fmt.Sprintf("workspace name %q must match ^[a-z][a-z0-9-]*$", name),
					})
				}
				continue
			}
			// Bare entry: any host path. Container-path-on-RHS form is no
			// longer accepted — colons in the entry mean the user likely
			// tried to hand-craft the container side.
			if strings.Contains(raw, ":") {
				out = append(out, Issue{
					Level: LevelError, Path: "spwn.yaml#worlds." + wname + ".workspaces",
					Message: fmt.Sprintf("workspace entry %q uses a container-path form; use name=path instead", entry),
					Hint:    "drop the container path; spwn mounts named workspaces at /workspaces/<name> automatically",
				})
			}
		}
	}
	return out
}

// rulePackVersionConflict flags multi-agent worlds whose members
// declare the same package at different versions. Versions are
// detected via the `@scope/name@version` suffix convention.
func rulePackVersionConflict(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for _, wname := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[wname]
		if len(w.Agents) < 2 {
			continue
		}
		// dependency-name → seen versions
		versions := map[string]map[string]string{} // dependency → version → first-agent
		for _, agentName := range w.Agents {
			a := findAgent(in.AgentRefs, agentName)
			if a == nil || !a.Exists {
				continue
			}
			parsed, err := loadAgentYAML(a.Path)
			if err != nil {
				continue
			}
			for _, t := range parsed.Deps {
				depRef, version := dependency.SplitVersion(t)
				vmap, ok := versions[depRef]
				if !ok {
					vmap = map[string]string{}
					versions[depRef] = vmap
				}
				if existing, ok := vmap[version]; !ok {
					vmap[version] = agentName
					_ = existing
				}
			}
		}
		for depRef, vmap := range versions {
			if len(vmap) <= 1 {
				continue
			}
			out = append(out, Issue{
				Level: LevelError, Path: "spwn.yaml#worlds." + wname,
				Message: fmt.Sprintf("dependency %q has conflicting versions across agents in world %q", depRef, wname),
				Hint:    "align the dependency version across all agents that share a world",
			})
		}
	}
	return out
}

// ruleRuntimeBackendConflict flags multi-agent worlds whose members
// disagree on a non-empty runtime backend. Empty (default) is
// compatible with anything explicit.
func ruleRuntimeBackendConflict(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for _, wname := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[wname]
		if len(w.Agents) < 2 {
			continue
		}
		var seen string
		var seenAgent string
		for _, agentName := range w.Agents {
			a := findAgent(in.AgentRefs, agentName)
			if a == nil || !a.Exists {
				continue
			}
			parsed, err := loadAgentYAML(a.Path)
			if err != nil || parsed.Runtime.Backend == "" {
				continue
			}
			if seen == "" {
				seen = parsed.Runtime.Backend
				seenAgent = agentName
				continue
			}
			if seen != parsed.Runtime.Backend {
				out = append(out, Issue{
					Level: LevelError, Path: "spwn.yaml#worlds." + wname,
					Message: fmt.Sprintf("runtime backend conflict: agent %q uses %q, agent %q uses %q",
						seenAgent, seen, agentName, parsed.Runtime.Backend),
				})
				break
			}
		}
	}
	return out
}

// rulePacksExist checks every dependency referenced by any agent or
// world against the BuiltinTools catalog (for @spwn/* refs) and
// against the filesystem (for bare local refs).
//
// Local refs resolve to either:
//   - spwn/tools/<name>/ (a directory-form package, with or without
//     a spwn.yaml inside), OR
//   - spwn/skills/<name>.md (a bare-markdown skill).
func rulePacksExist(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	builtin := make(map[string]struct{}, len(in.BuiltinTools))
	for _, t := range in.BuiltinTools {
		builtin[t] = struct{}{}
	}
	haveCatalog := in.BuiltinTools != nil
	checked := map[string]bool{}
	check := func(raw, location string) []Issue {
		depRef, _ := dependency.SplitVersion(raw)
		key := depRef + "@@" + location
		if checked[key] {
			return nil
		}
		checked[key] = true
		ref := dependency.ParseRef(depRef)

		// For @spwn/* and @<owner>/* refs, resolve against the
		// catalog.
		if ref.Kind != dependency.KindLocal {
			switch dependency.ResolveTool(in.Root, ref, builtin, haveCatalog) {
			case dependency.ResolveOK:
				return nil
			case dependency.ResolveRegistryUnsupported:
				return []Issue{{
					Level: LevelError, Path: location,
					Message: fmt.Sprintf("remote registries are not yet supported (ref: %q)", raw),
					Hint: "use @spwn/<name> for built-in dependencies or drop a directory under ./spwn/tools/<name>/ for a local tool; " +
						"remote registries (@<owner>/<name>) are planned but not implemented yet",
				}}
			default: // ResolveNotFound
				return []Issue{{
					Level: LevelError, Path: location,
					Message: fmt.Sprintf("dependency %q does not exist", raw),
					Hint:    suggestPackage(depRef, in.BuiltinTools),
				}}
			}
		}

		// Local ref: delegate to dependency.ResolveSkill which already
		// handles both the directory form (full dependency) and the
		// file form (bare markdown skill) against spwn/skills/ and spwn/tools/.
		if ref.Name == "" {
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("dependency %q is malformed", raw),
			}}
		}
		if dependency.ResolveSkill(in.Root, ref, nil, false) == dependency.ResolveOK {
			return nil
		}
		return []Issue{{
			Level: LevelError, Path: location,
			Message: fmt.Sprintf("dependency %q does not exist", raw),
			Hint: "create ./spwn/tools/" + ref.Name + "/spwn.yaml for a full dependency, " +
				"or ./spwn/tools/" + ref.Name + ".md for a bare skill",
		}}
	}

	var out []Issue
	// Project-level deps (top-level deps: in spwn.yaml).
	for _, t := range in.Manifest.Deps {
		out = append(out, check(t, "spwn.yaml#deps")...)
	}
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		parsed, err := loadAgentYAML(a.Path)
		if err != nil {
			continue
		}
		loc := relPath(in.Root, filepath.Join(a.Path, "agent.yaml")) + "#deps"
		for _, t := range parsed.Deps {
			out = append(out, check(t, loc)...)
		}
	}
	return out
}

// ruleLockfileConsistent compares every @spwn/* or @<owner>/*
// dependency ref declared in any agent.yaml or spwn.yaml world against
// spwn.lock. Missing entries become errors so `spwn build` fails
// loudly and points the user at `spwn install`.
//
// Local (bare) refs are never lockfile-tracked.
//
// If no lockfile exists yet, the rule is silent — the project has
// never been installed against. `spwn init` seeds an initial lockfile
// so freshly-scaffolded projects pass.
func ruleLockfileConsistent(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	lock, err := dependency.LoadLockfile(in.Root)
	if err != nil {
		return []Issue{{
			Level: LevelError, Path: dependency.LockFileName,
			Message: fmt.Sprintf("cannot read lockfile: %v", err),
			Hint:    "regenerate with `spwn install` for each declared dependency, or delete " + dependency.LockFileName + " to start fresh",
		}}
	}
	if lock == nil {
		return nil // no lockfile yet, nothing to compare against
	}

	type refRec struct {
		raw      string
		location string
	}
	var all []refRec

	collect := func(list []string, location string) {
		for _, r := range list {
			all = append(all, refRec{raw: r, location: location})
		}
	}

	// Project-level dependency.
	collect(in.Manifest.Deps, "spwn.yaml#deps")
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		parsed, err := loadAgentYAML(a.Path)
		if err != nil {
			continue
		}
		rel := relPath(in.Root, filepath.Join(a.Path, "agent.yaml"))
		collect(parsed.Deps, rel+"#deps")
	}

	seen := map[string]bool{}
	var out []Issue
	for _, rec := range all {
		depRef, _ := dependency.SplitVersion(rec.raw)
		ref := dependency.ParseRef(depRef)
		// Local refs are never lockfile entries.
		if ref.Kind == dependency.KindLocal {
			continue
		}
		if seen[depRef] {
			continue
		}
		seen[depRef] = true
		if lock.Has(depRef) {
			continue
		}
		out = append(out, Issue{
			Level: LevelError, Path: rec.location,
			Message: fmt.Sprintf("%q is not recorded in %s", depRef, dependency.LockFileName),
			Hint:    "run `spwn install " + depRef + "` to sync the lockfile",
		})
	}
	return out
}

// ruleRuntimeSupported checks each agent's runtime backend against
// the host's SupportedRuntimes list.
func ruleRuntimeSupported(in Input) []Issue {
	if len(in.SupportedRuntimes) == 0 {
		return nil
	}
	supported := map[string]struct{}{}
	for _, r := range in.SupportedRuntimes {
		supported[r] = struct{}{}
	}
	var out []Issue
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		parsed, err := loadAgentYAML(a.Path)
		if err != nil || parsed.Runtime.Backend == "" {
			continue
		}
		if _, ok := supported[parsed.Runtime.Backend]; !ok {
			out = append(out, Issue{
				Level: LevelError,
				Path:  relPath(in.Root, filepath.Join(a.Path, "agent.yaml")) + "#runtime.backend",
				Message: fmt.Sprintf("runtime backend %q is not supported", parsed.Runtime.Backend),
				Hint:    "supported: " + strings.Join(in.SupportedRuntimes, ", "),
			})
		}
	}
	return out
}

func ruleMarkdownImports(in Input) []Issue {
	var out []Issue
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		entryPath := filepath.Join(a.Path, "AGENTS.md")
		result, err := resolve.Walk(a.Path, entryPath)
		if err != nil {
			continue
		}
		for _, ref := range result.Missing {
			out = append(out, Issue{
				Level: LevelError,
				Path:  fmt.Sprintf("%s @%s", relPath(in.Root, ref.Source), ref.Target),
				Message: "broken @-import: " + ref.Target,
				Hint:    "create " + relPath(in.Root, ref.ResolvedPath) + " or remove the reference",
			})
		}
		for _, cycle := range result.Cycles {
			rel := make([]string, len(cycle))
			for j, p := range cycle {
				rel[j] = relPath(in.Root, p)
			}
			out = append(out, Issue{
				Level: LevelWarning, Path: relPath(in.Root, entryPath),
				Message: "import cycle: " + strings.Join(rel, " -> "),
			})
		}
	}
	return out
}

func ruleOrphanAgents(in Input) []Issue {
	var out []Issue
	for _, o := range in.OrphanRefs {
		out = append(out, Issue{
			Level: LevelInfo, Path: relPath(in.Root, o.Path),
			Message: "agent " + o.Name + " is not referenced by any world",
			Hint:    "add it to a worlds: entry, or `spwn agent rm " + o.Name + "`",
		})
	}
	return out
}

// ----- helpers -----

func findAgent(refs []AgentRef, name string) *AgentRef {
	for i := range refs {
		if refs[i].Name == name {
			return &refs[i]
		}
	}
	return nil
}

func sortedKeys(m map[string]intmanifest.World) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func suggestPackage(tool string, catalog []string) string {
	if len(catalog) == 0 {
		return "check the dependency name, or add it as a local tool under ./spwn/tools/"
	}
	best := ""
	bestScore := len(tool) + 1
	for _, c := range catalog {
		if d := editDistance(tool, c); d < bestScore && d <= 3 {
			best = c
			bestScore = d
		}
	}
	if best != "" {
		return "did you mean " + best + "?"
	}
	return "available built-ins: " + strings.Join(catalog, ", ")
}

func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			d := del
			if ins < d {
				d = ins
			}
			if sub < d {
				d = sub
			}
			curr[j] = d
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
