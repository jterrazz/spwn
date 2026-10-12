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

	"spwn.sh/packages/dependency"
	"spwn.sh/packages/dependency/refs"
	intmanifest "spwn.sh/packages/project/internal/manifest"
	"spwn.sh/packages/project/internal/resolve"

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

	// HookEventsByRuntime maps the runtime adapter's short name
	// (e.g. "claude-code", "codex") to the list of hook events that
	// runtime fires inside the container. The validator uses it to
	// warn when an agent's runtime would silently drop a hook
	// because it doesn't know the event. Nil or empty → skip the
	// check (golden tests + edge cases that don't construct a full
	// catalog stay quiet).
	HookEventsByRuntime map[string][]string
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
		ruleHookEventsSupported,
		ruleAutomations,
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

var reservedAgentSubcommands = map[string]struct{}{
	"create": {}, "new": {}, "ls": {}, "rm": {}, "fork": {}, "inspect": {},
	"logs": {}, "add": {}, "remove": {}, "talk": {}, "send": {}, "inbox": {},
	"watch": {}, "dream": {}, "sleep": {}, "publish": {}, "get": {},
	"export": {}, "import": {}, "start": {}, "stop": {}, "delete": {},
	"deploy": {}, "compose": {}, "list": {}, "init": {},

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

var reservedWorldSubcommands = map[string]struct{}{
	"create": {}, "start": {}, "stop": {}, "ls": {}, "rm": {}, "inspect": {},
	"logs": {}, "enter": {}, "destroy": {}, "rename": {}, "knowledge": {},
	"new": {}, "list": {}, "remove": {}, "up": {}, "down": {}, "snap": {},
}

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

type agentYAML struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Deps        []string
	Runtime     struct {
		Backend string `yaml:"backend"`
	} `yaml:"runtime"`
}

type rawAgentYAML struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Deps        yaml.Node `yaml:"dependencies"`
	Runtime     struct {
		Backend string `yaml:"backend"`
	} `yaml:"runtime"`
}

// UnmarshalYAML accepts each `dependencies:` entry as either a scalar
// string ("spwn:x") or a mapping with at least a "name:" key. Mapping
// entries without a name field cause a structured error rather than
// a cryptic !!map-into-string diagnostic.
func (a *agentYAML) UnmarshalYAML(node *yaml.Node) error {
	var raw rawAgentYAML
	if err := node.Decode(&raw); err != nil {
		return err
	}
	a.Name = raw.Name
	a.Description = raw.Description
	a.Runtime = raw.Runtime
	if raw.Deps.Kind == 0 {
		return nil
	}
	for i := range raw.Deps.Content {
		entry := raw.Deps.Content[i]
		switch entry.Kind {
		case yaml.ScalarNode:
			a.Deps = append(a.Deps, entry.Value)
		case yaml.MappingNode:
			var name string
			for j := 0; j < len(entry.Content)-1; j += 2 {
				k := entry.Content[j]
				v := entry.Content[j+1]
				if k.Value == "name" && v.Kind == yaml.ScalarNode {
					name = v.Value
					break
				}
			}
			if name == "" {
				return fmt.Errorf("dependencies[%d]: mapping entry missing required 'name' key", i)
			}
			a.Deps = append(a.Deps, name)
		default:
			return fmt.Errorf("dependencies[%d]: must be a string or a mapping with a 'name' key", i)
		}
	}
	return nil
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
				continue
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
				Hint:    fmt.Sprintf("move entries into the flat `dependencies:` list (e.g. `dependencies: [\"spwn:unix\", \"%s/<name>\"]`); top-level `%s:` is no longer read", strings.TrimSuffix(key, "s"), key),
			})
		}
	}
	return out
}

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

func ruleOneAgentOneWorld(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	first := map[string]string{}
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

func ruleWorkspaceMounts(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for _, wname := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[wname]
		for _, entry := range w.Workspaces {
			raw := strings.TrimSuffix(entry, ":ro")
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
		versions := map[string]map[string]string{}
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
					"skill/<name>, tool/<name>, hook/<name>, or command/<name>; remote registries " +
					"(github:<owner>/<repo>) are planned but not implemented yet",
			}}
		case refs.ResolveInvalid:
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("dependency %q is invalid", raw),
				Hint:    invalidRefHint(in.Root, raw),
			}}
		}

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
				Hint:    "create ./spwn/hooks/" + ref.Name + ".yaml for a runtime hook",
			}}
		case refs.KindLocalCommand:
			return []Issue{{
				Level: LevelError, Path: location,
				Message: fmt.Sprintf("dependency %q does not exist", raw),
				Hint:    "create ./spwn/commands/" + ref.Name + ".md for a slash-invoked command",
			}}
		}
		return []Issue{{
			Level: LevelError, Path: location,
			Message: fmt.Sprintf("dependency %q does not exist", raw),
			Hint:    suggestPackage(depRef, in.BuiltinTools),
		}}
	}

	var out []Issue
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
		return nil
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

func ruleRuntimeSupported(in Input) []Issue {
	if len(in.SupportedRuntimes) == 0 {
		return nil
	}
	supported := map[string]struct{}{}
	for _, r := range in.SupportedRuntimes {
		supported[refs.Canonical(r)] = struct{}{}
	}
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
				Level:   LevelError,
				Path:    relPath(in.Root, filepath.Join(a.Path, "agent.yaml")) + "#runtime.backend",
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
				Level:   LevelError,
				Path:    fmt.Sprintf("%s @%s", relPath(in.Root, ref.Source), ref.Target),
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

func skillSearchRoots(root string) []string {
	var roots []string

	roots = append(roots, filepath.Join(root, "spwn", "skills"))

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

func ruleHookEventsSupported(in Input) []Issue {
	if len(in.HookEventsByRuntime) == 0 || in.Manifest == nil {
		return nil
	}
	hooksDir := filepath.Join(in.Root, "spwn", "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return nil
	}

	defaultRuntime := strings.TrimSpace(in.Manifest.Runtime.Backend)

	agentRuntime := make(map[string]string, len(in.AgentRefs))
	agentDeps := make(map[string][]string, len(in.AgentRefs))
	for _, a := range in.AgentRefs {
		if !a.Exists {
			continue
		}
		y, ferr := loadAgentYAML(a.Path)
		if ferr != nil || y == nil {
			continue
		}
		backend := strings.TrimSpace(y.Runtime.Backend)
		if backend == "" {
			backend = defaultRuntime
		}
		agentRuntime[a.Name] = stripCatalogPrefix(backend)
		agentDeps[a.Name] = y.Deps
	}

	selectors := make(map[string][]string)
	for agentName, deps := range agentDeps {
		for _, dep := range deps {
			ref := strings.TrimSpace(dep)
			if !strings.HasPrefix(ref, "hook/") {
				continue
			}
			hookName := strings.TrimPrefix(ref, "hook/")
			if hookName == "" {
				continue
			}
			selectors[hookName] = append(selectors[hookName], agentName)
		}
	}

	var out []Issue
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		hookName := strings.TrimSuffix(e.Name(), ".yaml")
		path := filepath.Join(hooksDir, e.Name())
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			continue
		}
		var parsed struct {
			Event string `yaml:"event"`
		}
		if uerr := yaml.Unmarshal(data, &parsed); uerr != nil {
			continue
		}
		event := strings.TrimSpace(parsed.Event)
		if event == "" {
			continue
		}

		display := relPath(in.Root, path)
		picked := selectors[hookName]
		sort.Strings(picked)

		if len(picked) == 0 {
			out = append(out, Issue{
				Level:   LevelInfo,
				Path:    display,
				Message: fmt.Sprintf("hook %q is not selected by any agent's `hook/%s` dep — it will never fire", hookName, hookName),
				Hint:    fmt.Sprintf("add `- hook/%s` to the agent.yaml of every agent that should run it", hookName),
			})
			continue
		}

		seen := map[string]struct{}{}
		for _, agentName := range picked {
			runtime := agentRuntime[agentName]
			if runtime == "" {
				continue
			}
			supported, ok := in.HookEventsByRuntime[runtime]
			if !ok {
				continue
			}
			if eventInList(event, supported) {
				continue
			}
			key := runtime + "\x00" + event
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, Issue{
				Level: LevelWarning,
				Path:  display,
				Message: fmt.Sprintf(
					"hook %q targets event %q which runtime %q (used by agent %q) does not fire",
					hookName, event, runtime, agentName,
				),
				Hint: fmt.Sprintf(
					"either rename the event to one of [%s], detach this hook from %q, or add a runtime that supports %q",
					strings.Join(supported, ", "), agentName, event,
				),
			})
		}
	}
	return out
}

func eventInList(want string, set []string) bool {
	for _, s := range set {
		if s == want {
			return true
		}
	}
	return false
}

func stripCatalogPrefix(ref string) string {
	if strings.HasPrefix(ref, "spwn:") {
		return strings.TrimPrefix(ref, "spwn:")
	}
	return ref
}

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

func invalidRefHint(root, raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.TrimPrefix(name, "@")
	for _, retired := range []string{"local:", "skill:", "tool:", "hook:"} {
		if strings.HasPrefix(name, retired) {
			name = strings.TrimPrefix(name, retired)
			break
		}
	}

	if name != "" && !strings.ContainsAny(name, ":/") {
		var matches []string
		if st, err := os.Stat(filepath.Join(root, "spwn", "skills", name+".md")); err == nil && !st.IsDir() {
			matches = append(matches, "skill/"+name)
		}
		if st, err := os.Stat(filepath.Join(root, "spwn", "tools", name)); err == nil && st.IsDir() {
			matches = append(matches, "tool/"+name)
		}
		if st, err := os.Stat(filepath.Join(root, "spwn", "hooks", name+".yaml")); err == nil && !st.IsDir() {
			matches = append(matches, "hook/"+name)
		}
		if st, err := os.Stat(filepath.Join(root, "spwn", "commands", name+".md")); err == nil && !st.IsDir() {
			matches = append(matches, "command/"+name)
		}
		if len(matches) == 1 {
			return "did you mean " + matches[0] + "?"
		}
		if len(matches) > 1 {
			return "this name matches multiple local blocks — use one of: " + strings.Join(matches, ", ")
		}
	}

	return "use skill/<name> (for spwn/skills/<name>.md), tool/<name> (for spwn/tools/<name>/), " +
		"hook/<name> (for spwn/hooks/<name>.yaml), or command/<name> (for spwn/commands/<name>.md); " +
		"spwn:<name> and github:<owner>/<repo> remain the two source-prefixed schemes"
}

func suggestPackage(tool string, catalog []string) string {
	if len(catalog) == 0 {
		return "check the dependency name, or add it as a local tool under ./spwn/tools/"
	}
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
