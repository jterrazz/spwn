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
	"spwn.sh/packages/dependency/refs"

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
	// spwn:* prefix heuristic.
	BuiltinTools []string

	// SupportedRuntimes is the list of runtime backends the host can
	// actually spawn (e.g. "spwn:claude-code"). Nil → don't check.
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
		ruleAgentYAMLLegacyKeys,
		ruleAgentDescription,
		ruleReservedAgentNames,
		ruleOneAgentOneWorld,
		ruleWorkspaceMounts,
		rulePackVersionConflict,
		ruleRuntimeBackendConflict,
		rulePacksExist,
		ruleLockfileConsistent,
		ruleRuntimeSupported,
		ruleMarkdownImports,
		ruleSkillFrontmatter,
		ruleOrphanAgents,
		ruleKnowledgePath,
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

// MaxAgentNameLen caps agent names below the tightest filesystem
// limit likely to matter in practice. Most Linux filesystems allow
// up to 255 bytes per path component; `docker cp` uses tar headers
// that have their own length rules. 63 mirrors the DNS label limit
// and is plenty for a human-memorable agent name — anything longer
// is probably a typo or pathological test input.
const MaxAgentNameLen = 63

// IsValidAgentName reports whether the given name is a valid agent
// name — a slug matching the same regex the manifest enforces for
// world names (`^[a-z][a-z0-9-]*$`) and at most MaxAgentNameLen
// bytes. Empty names are invalid. Callers use this to reject bad
// names at creation time, before anything is written to disk.
func IsValidAgentName(name string) bool {
	return name != "" && len(name) <= MaxAgentNameLen && slugRe.MatchString(name)
}

// IsValidProjectName reports whether the given name matches the
// manifest's project-name regex (`^[a-z0-9][a-z0-9-]*$`). Empty
// names are invalid.
func IsValidProjectName(name string) bool {
	return name != "" && projectName.MatchString(name)
}

// reservedAgentSubcommands is the set of names that collide with
// `spwn agent <subcommand>` OR with top-level `spwn <noun>` commands
// that the `spwn <agent-name>` shortcut form would ambiguously match.
// The former group prevents `spwn agent create agent` shadowing the
// subcommand; the latter prevents `spwn architect` meaning both "run
// the daemon" and "talk to the agent named architect".
var reservedAgentSubcommands = map[string]struct{}{
	// `spwn agent <subcommand>` collisions
	"create": {}, "new": {}, "ls": {}, "rm": {}, "fork": {}, "inspect": {},
	"logs": {}, "add": {}, "remove": {}, "talk": {}, "send": {}, "inbox": {},
	"watch": {}, "dream": {}, "sleep": {}, "publish": {}, "get": {},
	"export": {}, "import": {}, "start": {}, "stop": {}, "delete": {},
	"deploy": {}, "compose": {}, "list": {}, "init": {},

	// `spwn <top-level>` collisions — the bare-name shortcut
	// `spwn <agent-name>` would route to the top-level command
	// instead of the agent session, shadowing it permanently.
	"architect": {}, "world": {}, "agent": {}, "check": {}, "up": {},
	"down": {}, "build": {}, "install": {}, "uninstall": {}, "skill": {},
	"auth": {}, "status": {}, "web": {}, "upgrade": {}, "team": {},
	"organization": {}, "snap": {}, "help": {}, "version": {},
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
		{"SOUL.md", false, LevelError},
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
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Deps        []string `yaml:"dependencies"`
	Runtime     struct {
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

// ruleAgentYAMLLegacyKeys flags legacy top-level keys in agent.yaml
// (`tools:`, `skills:`, `hooks:`) that pre-date the unified
// `dependencies:` list. The YAML parser silently ignores unknown
// top-level keys, which meant a pre-refactor manifest upgraded
// in-place would lose every dep without any user-visible signal.
// This rule catches that case with a migration hint pointing at the
// current schema.
func ruleAgentYAMLLegacyKeys(in Input) []Issue {
	var out []Issue
	legacyKeys := []string{"tools", "skills", "hooks"}
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		yamlPath := filepath.Join(a.Path, "agent.yaml")
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			continue
		}
		// Parse into a generic map so unknown top-level keys are
		// visible. A strict-typed struct would silently drop them,
		// which is exactly the rot this rule exists to catch.
		var raw map[string]yaml.Node
		if yaml.Unmarshal(data, &raw) != nil {
			continue
		}
		for _, key := range legacyKeys {
			if _, ok := raw[key]; !ok {
				continue
			}
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, yamlPath) + "#" + key,
				Message: fmt.Sprintf("legacy top-level %q block in agent.yaml", key),
				Hint:    fmt.Sprintf("move entries into the flat `dependencies:` list (e.g. `dependencies: [\"spwn:unix\", \"%s:<name>\"]`); top-level `%s:` is no longer read", strings.TrimSuffix(key, "s"), key),
			})
		}
	}
	return out
}

// ruleAgentDescription enforces that every agent.yaml sets a
// non-empty `description:`. The description is a one-line pitch of
// the agent's purpose — what inspect, status, and the web UI render
// without opening AGENTS.md. Mirrors the skill frontmatter
// convention (name + description) applied at the agent level.
//
// Unparseable agent.yaml files are already flagged by
// ruleAgentYAMLParses; here we just skip them to avoid double-reporting.
func ruleAgentDescription(in Input) []Issue {
	var out []Issue
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		parsed, err := loadAgentYAML(a.Path)
		if err != nil {
			continue
		}
		if strings.TrimSpace(parsed.Description) == "" {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, filepath.Join(a.Path, "agent.yaml")) + "#description",
				Message: fmt.Sprintf("agent %q is missing a description", a.Name),
				Hint:    "add a one-line `description:` explaining what this agent is for",
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
//   - "path"           (bare host path — auto-named from the path basename
//                       when slug-compliant, falling back to workspace<N>)
//   - "name=path"      (named mount; container-side becomes /workspaces/<name>)
//   - "name=path:ro"   (same, read-only)
//
// Container paths never appear in the manifest. Users don't write
// `/workspaces/...` — that's an implementation detail of where the
// mount lands inside the container. See world.AutoWorkspaceName for
// the bare-path naming logic.
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
				depRef, version := refs.SplitVersion(t)
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
// world against the BuiltinTools catalog (for spwn:* refs) and
// against the filesystem (for local scheme refs: skill:/tool:/hook:).
//
// Every ref must use an explicit scheme. Bare names (including the
// retired `local:<name>` alias) are rejected with a hint pointing the
// author at the three local schemes.
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
		depRef, _ := refs.SplitVersion(raw)
		key := depRef + "@@" + location
		if checked[key] {
			return nil
		}
		checked[key] = true
		ref := refs.ParseRef(depRef)

		switch refs.ResolveTool(in.Root, ref, builtin, haveCatalog) {
		case refs.ResolveOK:
			return nil
		case refs.ResolveRegistryUnsupported:
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("remote registries are not yet supported (ref: %q)", raw),
				Hint: "use spwn:<name> for built-in dependencies, or author a local dep with " +
					"skill:<name>, tool:<name>, or hook:<name>; remote registries " +
					"(github:<owner>/<repo>) are planned but not implemented yet",
			}}
		case refs.ResolveInvalid:
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("dependency %q is invalid", raw),
				Hint:    invalidRefHint(in.Root, raw),
			}}
		}

		// ResolveNotFound falls through here.
		switch ref.Kind {
		case refs.KindLocalSkill:
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("dependency %q does not exist", raw),
				Hint:    "author ./spwn/skills/" + ref.Name + ".md (e.g. `spwn skill new " + ref.Name + "`)",
			}}
		case refs.KindLocalTool:
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("dependency %q does not exist", raw),
				Hint:    "create ./spwn/tools/" + ref.Name + "/tool.yaml for a full local tool",
			}}
		case refs.KindLocalHook:
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("dependency %q does not exist", raw),
				Hint:    "create ./spwn/hooks/" + ref.Name + ".sh for a lifecycle hook",
			}}
		}
		return []Issue{{
			Level: LevelError, Path: location,
			Message: fmt.Sprintf("dependency %q does not exist", raw),
			Hint:    suggestPackage(depRef, in.BuiltinTools),
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

// ruleLockfileConsistent compares every spwn:* or @<owner>/*
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
		depRef, _ := refs.SplitVersion(rec.raw)
		ref := refs.ParseRef(depRef)
		// Local refs are never lockfile entries. Invalid refs are
		// already surfaced by rulePacksExist with a crisper error;
		// skip them here to avoid double-reporting.
		if refs.IsLocalKind(ref.Kind) || ref.Kind == refs.KindInvalid {
			continue
		}
		if seen[depRef] {
			continue
		}
		seen[depRef] = true
		if lock.Has(depRef) || lock.Has(refs.Canonical(depRef)) {
			continue
		}
		out = append(out, Issue{
			Level: LevelError, Path: rec.location,
			Message: fmt.Sprintf("%q is not recorded in %s", refs.Canonical(depRef), dependency.LockFileName),
			Hint:    "run `spwn install " + refs.Canonical(depRef) + "` to sync the lockfile",
		})
	}
	return out
}

// ruleRuntimeSupported checks each agent's runtime backend against
// the host's SupportedRuntimes list. Accepts either ref syntax —
// `spwn:claude-code` and `spwn:claude-code` both resolve to the
// same entry in the supported-runtime set.
func ruleRuntimeSupported(in Input) []Issue {
	if len(in.SupportedRuntimes) == 0 {
		return nil
	}
	supported := map[string]struct{}{}
	for _, r := range in.SupportedRuntimes {
		supported[refs.Canonical(r)] = struct{}{}
	}
	// Display list stays in the canonical scheme form so the hint
	// matches what scaffold/docs advertise.
	display := make([]string, len(in.SupportedRuntimes))
	for i, r := range in.SupportedRuntimes {
		display[i] = refs.Canonical(r)
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
		if _, ok := supported[refs.Canonical(parsed.Runtime.Backend)]; !ok {
			out = append(out, Issue{
				Level: LevelError,
				Path:  relPath(in.Root, filepath.Join(a.Path, "agent.yaml")) + "#runtime.backend",
				Message: fmt.Sprintf("runtime backend %q is not supported", parsed.Runtime.Backend),
				Hint:    "supported: " + strings.Join(display, ", "),
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

// ruleSkillFrontmatter enforces the skill markdown convention:
// every .md under spwn/skills/ and spwn/tools/<name>/skills/ must
// start with a YAML frontmatter block declaring `name:` and
// `description:`.
//
// Shape (the "SKILL" convention, kept as generic markdown
// frontmatter so non-skill .md can opt in later):
//
//	---
//	name: paper-reading
//	description: Use when summarising academic papers …
//	---
//
//	<markdown body>
//
// Missing block, missing field, or unterminated block all surface
// as LevelError with a fix-it hint. Other fields in the
// frontmatter are accepted and ignored so authors can attach
// their own metadata without tripping the rule.
func ruleSkillFrontmatter(in Input) []Issue {
	var out []Issue
	for _, path := range collectSkillFiles(in.Root) {
		rel := relPath(in.Root, path)
		fm, err := ParseMarkdownFrontmatter(path)
		if err != nil {
			out = append(out, Issue{
				Level: LevelError, Path: rel,
				Message: "skill frontmatter is malformed: " + err.Error(),
				Hint:    "top of file must be:\n---\nname: <slug>\ndescription: <one-line hint>\n---",
			})
			continue
		}
		if !fm.Found {
			out = append(out, Issue{
				Level: LevelError, Path: rel,
				Message: "skill is missing YAML frontmatter",
				Hint:    "add this block at the top of the file:\n---\nname: <slug>\ndescription: <one-line hint>\n---",
			})
			continue
		}
		if strings.TrimSpace(fm.Keys["name"]) == "" {
			out = append(out, Issue{
				Level: LevelError, Path: rel,
				Message: "skill frontmatter is missing `name`",
				Hint:    "add `name: <slug>` to the frontmatter",
			})
		}
		if strings.TrimSpace(fm.Keys["description"]) == "" {
			out = append(out, Issue{
				Level: LevelError, Path: rel,
				Message: "skill frontmatter is missing `description`",
				Hint:    "add a one-line `description:` explaining when the agent should use this skill",
			})
		}
	}
	return out
}

// collectSkillFiles walks every place a user authors skills in a
// spwn project and returns the full list of markdown files found
// there. The two locations are:
//
//   - spwn/skills/              — project-wide bare skills
//   - spwn/tools/<name>/skills/ — skills shipped by a local tool
//
// Each location is walked recursively so nested skill directories
// (spwn/skills/reviewing/code-review.md) are covered. Missing
// locations return an empty slice, never an error.
func collectSkillFiles(root string) []string {
	var out []string
	for _, rel := range skillSearchRoots(root) {
		_ = filepath.WalkDir(rel, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				out = append(out, path)
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

// skillSearchRoots lists every directory under which skill markdown
// files may appear. Directories that don't exist are silently skipped
// by the walker (filepath.WalkDir returns os.ErrNotExist which the
// collector drops).
func skillSearchRoots(root string) []string {
	var roots []string

	// Project-wide skills.
	roots = append(roots, filepath.Join(root, "spwn", "skills"))

	// Local-tool skills. One skills/ dir per tool.
	if entries, err := os.ReadDir(filepath.Join(root, "spwn", "tools")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				roots = append(roots, filepath.Join(root, "spwn", "tools", e.Name(), "skills"))
			}
		}
	}

	return roots
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

// ruleKnowledgePath surfaces issues around the worlds.<name>.knowledge
// key. When a world declares a path, the path must exist on disk
// (LevelWarning if missing — spawn still works, it just skips the
// bind mount). When a world declares no path, an info-level hint
// reminds the user that agents in that world will never be told a
// knowledge base exists.
func ruleKnowledgePath(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for _, name := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[name]
		path := strings.TrimSpace(w.Knowledge)
		if path == "" {
			out = append(out, Issue{
				Level:   LevelInfo,
				Path:    "spwn.yaml#worlds." + name + ".knowledge",
				Message: fmt.Sprintf("world %q has no knowledge path; agents will see an empty /world/knowledge/ and won't be told one exists.", name),
				Hint:    "add `knowledge: ./knowledge` (or another path) to enable the shared knowledge base for this world",
			})
			continue
		}
		resolved := path
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(in.Root, path)
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			out = append(out, Issue{
				Level:   LevelWarning,
				Path:    "spwn.yaml#worlds." + name + ".knowledge",
				Message: fmt.Sprintf("knowledge path %q does not exist for world %q", path, name),
				Hint:    "create the directory (e.g. `mkdir -p " + path + "`) or drop the `knowledge:` key to disable the bind mount",
			})
		}
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

// invalidRefHint produces the canonical "use skill:/tool:/hook:" hint
// for a bare/malformed dependency ref. When the bare name happens to
// match exactly one on-disk target (skill file, tool dir, or hook
// script) we point at that specific scheme so the author's fix is a
// one-character edit; otherwise we list all three options.
func invalidRefHint(root, raw string) string {
	name := strings.TrimSpace(raw)
	// Drop any accidental leading `@` or `local:` so the scheme
	// suggestion still works for legacy authorings that sneak through.
	name = strings.TrimPrefix(name, "@")
	name = strings.TrimPrefix(name, "local:")

	if name != "" && !strings.ContainsAny(name, ":/") {
		var matches []string
		if st, err := os.Stat(filepath.Join(root, "spwn", "skills", name+".md")); err == nil && !st.IsDir() {
			matches = append(matches, "skill:"+name)
		}
		if st, err := os.Stat(filepath.Join(root, "spwn", "tools", name)); err == nil && st.IsDir() {
			matches = append(matches, "tool:"+name)
		}
		if st, err := os.Stat(filepath.Join(root, "spwn", "hooks", name+".sh")); err == nil && !st.IsDir() {
			matches = append(matches, "hook:"+name)
		}
		if len(matches) == 1 {
			return "did you mean " + matches[0] + "?"
		}
		if len(matches) > 1 {
			return "this name matches multiple local blocks — use one of: " + strings.Join(matches, ", ")
		}
	}

	return "use skill:<name> (for spwn/skills/<name>.md), tool:<name> (for spwn/tools/<name>/), " +
		"or hook:<name> (for spwn/hooks/<name>.sh); spwn:<name> and github:<owner>/<repo> remain the two external schemes"
}

func suggestPackage(tool string, catalog []string) string {
	if len(catalog) == 0 {
		return "check the dependency name, or add it as a local tool under ./spwn/tools/"
	}
	// Display every catalog entry in the canonical scheme form
	// (`spwn:unix`, not `spwn:unix`) so the hint matches what
	// scaffold output and docs now advertise. Legacy input like
	// `spwn:nonexistent` still matches — the check side uses
	// refs.ParseRef, not string equality.
	display := make([]string, len(catalog))
	for i, c := range catalog {
		display[i] = refs.Canonical(c)
	}
	best := ""
	bestScore := len(tool) + 1
	for i, c := range catalog {
		if d := editDistance(tool, c); d < bestScore && d <= 3 {
			best = display[i]
			bestScore = d
		}
	}
	if best != "" {
		return "did you mean " + best + "?"
	}
	return "available built-ins: " + strings.Join(display, ", ")
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
