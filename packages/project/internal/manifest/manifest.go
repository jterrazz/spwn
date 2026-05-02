// Package manifest is the internal parser and schema for spwn.yaml.
// The public surface is re-exported from packages/manifest.
//
// Schema model (v1):
//
//   - Agents are the primary runtime unit. Their on-disk presence at
//     spwn/agents/<name>/ is the source of truth for the project's
//     roster.
//   - Worlds are inline entries in spwn.yaml under `worlds:`. Each one
//     declares which agents it deploys, where the workspace mounts
//     come from, and optional tool overrides.
package manifest

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// CurrentVersion is the schema version Load emits for new manifests
// and the only version LoadPath accepts without upgrade.
const CurrentVersion = 1

// Manifest is the parsed content of spwn.yaml.
type Manifest struct {
	// Version is the schema version. Must be CurrentVersion.
	Version int `yaml:"version"`

	// Name is the project name. Used in world IDs, UI, and logs.
	Name string `yaml:"name"`

	// Runtime is the project-wide runtime default. Agents that omit
	// runtime.backend in their agent.yaml inherit this value.
	// Mirrors the agent.yaml#runtime shape so authors recognise it.
	Runtime Runtime `yaml:"runtime,omitempty"`

	// Worlds is the deployable world map keyed by world name. Each
	// entry declares which agents it spawns and what workspaces are
	// mounted into the resulting container.
	Worlds map[string]World `yaml:"worlds"`

	// Deps is the project-wide dependency pool. Every agent in
	// every world inherits these. Agent-level agent.yaml can add
	// more but cannot remove project-level dependency.
	Deps []string `yaml:"dependencies,omitempty"`
}

// Runtime is the project-wide runtime block. Exactly one field today
// (Backend), but authored as a sub-map for parity with agent.yaml so
// future additions (provider, flags…) don't reshape existing files.
type Runtime struct {
	// Backend is the runtime adapter ref agents inherit when their
	// agent.yaml#runtime.backend is empty. Accepts the same forms as
	// the agent-level field — scheme ("spwn:codex") or short name
	// ("codex"). Empty means "no project-wide preference".
	Backend string `yaml:"backend,omitempty"`
}

// World is one inline world entry in spwn.yaml.
type World struct {
	// Agents is the ordered list of agent names this world deploys.
	// Each name must match a directory under spwn/agents/.
	Agents []string `yaml:"agents"`

	// Workspaces is the list of host paths to mount inside the
	// container under /workspace. The first entry may be a bare host
	// path; subsequent entries must use explicit `host:/workspace/...`
	// form.
	Workspaces []string `yaml:"workspaces"`

	// Knowledge, when set, is a project-relative (or absolute) path to
	// a directory that will be bind-mounted into the container at
	// /world/knowledge/. When empty, no bind mount is performed and
	// the agent's system prompt omits every reference to the knowledge
	// base — the agent is never told a knowledge base exists.
	Knowledge string `yaml:"knowledge,omitempty"`

	// Automations is the per-world map of trigger-driven agent wakeups.
	// Each entry binds an event (cron or filesystem) to one of the
	// world's agents and renders a prompt. The architect daemon owns
	// the engine — automations only fire while architect is running.
	// See packages/automation/ for the runtime.
	Automations map[string]Automation `yaml:"automations,omitempty"`
}

// Automation is one trigger → agent wakeup binding inside a world.
//
// Schema (v1):
//
//	on:           — exactly one of cron / fs
//	  cron: "0 6 * * *"
//	  fs:
//	    path: ./inbox
//	    events: [create]      # default: [create]
//	    recursive: false      # default: false
//	    debounce: 1s          # default: 1s
//	    patterns: ["*.md"]    # default: [] (matches all)
//	agent: <agent-name>      — must be one of the world's agents
//	prompt: <inline body>    — exactly one of prompt / command
//	command: command/<name>  — references spwn/commands/<name>.md
//	catchup: collapse|skip   — cron only; default collapse
//
// Catchup semantics mirror Apple Reminders: when the architect was
// down across one or more cron slots, `collapse` fires once on resume
// (with the missed-count exposed to the prompt template), `skip`
// drops missed slots entirely. `stack` (one fire per missed slot) is
// reserved for v2 and will require a max-replay cap to avoid blast.
type Automation struct {
	// On is the trigger source. Exactly one field must be set.
	On Trigger `yaml:"on"`

	// Agent is the target agent name. Must be present in the
	// enclosing world's `agents:` list.
	Agent string `yaml:"agent"`

	// Prompt is the inline prompt body. Mutually exclusive with
	// Command. Templating: {{ .Now }} for cron, {{ .Event.Path }}
	// {{ .Event.Name }} for fs, {{ .Missed }} {{ .LastFired }} for
	// catchup-mode cron fires.
	Prompt string `yaml:"prompt,omitempty"`

	// Command is a `command/<name>` ref resolving to
	// spwn/commands/<name>.md. The file's body is the prompt template.
	// Mutually exclusive with Prompt.
	Command string `yaml:"command,omitempty"`

	// Catchup controls behaviour when the architect resumes after
	// missing one or more cron slots. Cron-only.
	//
	// Values:
	//   "collapse" (default) — single fire on resume regardless of
	//                          missed count; .Missed exposed to
	//                          template
	//   "skip"               — no fire on resume; schedule continues
	//
	// Empty string defaults to "collapse" for cron triggers and is
	// ignored for fs triggers.
	Catchup string `yaml:"catchup,omitempty"`
}

// Trigger holds the event source. Exactly one of Cron / FS is set.
type Trigger struct {
	// Cron is a 5-field cron expression in the host's local timezone
	// (matches launchd's StartCalendarInterval, which is what the
	// jterrazz-os prior art used). Standard "min hour dom month dow".
	Cron string `yaml:"cron,omitempty"`

	// FS is the filesystem watcher trigger.
	FS *FSTrigger `yaml:"fs,omitempty"`
}

// FSTrigger is a filesystem-watch event source.
type FSTrigger struct {
	// Path is the host path to watch. Project-relative paths resolve
	// against the project root. Must exist when the engine starts —
	// validation warns if the path is missing on disk.
	Path string `yaml:"path"`

	// Events filters which fsnotify ops fire the automation.
	// Allowed: "create" | "write" | "rename". Default: ["create"].
	// "remove" is intentionally unsupported in v1 — agents reacting
	// to deletions tend to want "the file used to be there" context
	// the watcher can't provide.
	Events []string `yaml:"events,omitempty"`

	// Recursive, when true, watches every subdirectory of Path.
	// Default false. New subdirs created after engine start are
	// auto-watched.
	Recursive bool `yaml:"recursive,omitempty"`

	// Debounce coalesces bursts. A burst is a sequence of events
	// where each is within Debounce of the previous; the engine fires
	// once at the trailing edge with all paths in the rendered
	// prompt's .Event.Paths field. Default: 1s. Min 100ms, max 1h.
	Debounce Duration `yaml:"debounce,omitempty"`

	// Patterns filters by filename. Doublestar globs against the
	// filename (not the full path). Empty = match all.
	// Examples: ["*.md"], ["*.{md,txt}"].
	Patterns []string `yaml:"patterns,omitempty"`

	// IncludeHidden enables fires for files inside dot-prefixed
	// directories (.git/, .cache/, etc) when Recursive is true. By
	// default the recursive watcher silently excludes them — most
	// users don't want their inbox watcher woken by every git
	// operation. Set true for cases where the hidden tree is
	// intentional content (e.g. dotfile editors).
	IncludeHidden bool `yaml:"include_hidden,omitempty"`
}

// Duration wraps time.Duration so spwn.yaml can use the natural
// "10s" / "1m" / "1h30m" string form. The standard yaml.v3 lib does
// not parse durations natively — without this wrapper authors would
// have to write nanosecond integers.
type Duration time.Duration

// UnmarshalYAML decodes a YAML scalar string into a Duration via
// time.ParseDuration. Empty / missing values produce zero, callers
// (parser defaults / validation) substitute the appropriate fallback.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("duration must be a string like \"10s\" or \"1m\": %w", err)
	}
	if s == "" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// MarshalYAML re-emits the duration in its canonical short form so a
// round-trip leaves spwn.yaml readable instead of dumping a raw
// integer of nanoseconds.
func (d Duration) MarshalYAML() (any, error) {
	if d == 0 {
		return "", nil
	}
	return time.Duration(d).String(), nil
}

// AsDuration is a typed conversion helper for engines that operate on
// the standard time.Duration type. Returns 0 when the field was
// omitted.
func (d Duration) AsDuration() time.Duration {
	return time.Duration(d)
}

// LoadPath reads and parses spwn.yaml from an explicit file path.
// Applies defaults but does NOT run validation rules.
func LoadPath(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	ApplyDefaults(&m)
	return &m, nil
}

// ApplyDefaults fills in optional fields that were left blank.
func ApplyDefaults(m *Manifest) {
	if m.Version == 0 {
		m.Version = CurrentVersion
	}
	if m.Worlds == nil {
		m.Worlds = map[string]World{}
	}
	for wname, w := range m.Worlds {
		applyAutomationDefaults(w.Automations)
		m.Worlds[wname] = w
	}
}

// applyAutomationDefaults fills in per-automation defaults so the
// engine + validator can read a single canonical shape regardless of
// how terse the author was. Only fills the fields the author left
// blank — explicit values are never overridden.
//
// Modifies the map values in place.
func applyAutomationDefaults(autos map[string]Automation) {
	for name, a := range autos {
		// Catchup default — collapse for cron, blank for fs (the
		// concept doesn't apply to filesystem watchers, which always
		// stack-on-diff via the seen-list).
		if a.On.Cron != "" && a.Catchup == "" {
			a.Catchup = "collapse"
		}
		// FS-trigger defaults.
		if a.On.FS != nil {
			fs := a.On.FS
			if len(fs.Events) == 0 {
				fs.Events = []string{"create"}
			}
			if fs.Debounce == 0 {
				fs.Debounce = Duration(1 * time.Second)
			}
			a.On.FS = fs
		}
		autos[name] = a
	}
}

// AllAgentNames returns the deduplicated set of agent names referenced
// by any world entry in the manifest, in stable sorted order.
func (m *Manifest) AllAgentNames() []string {
	if m == nil {
		return nil
	}
	seen := map[string]struct{}{}
	for _, w := range m.Worlds {
		for _, a := range w.Agents {
			seen[a] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	// stable order without importing sort here would be ugly; use
	// sort to keep callers predictable.
	sortStrings(out)
	return out
}

// sortStrings is a tiny insertion sort kept local so this file doesn't
// pull in "sort" just for AllAgentNames.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
