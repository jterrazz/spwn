package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	intmanifest "spwn.sh/packages/project/internal/manifest"
)

// ruleAutomations validates spwn.yaml#worlds.<name>.automations entries.
//
// Each automation describes "trigger → wake this agent → with this
// prompt body". The validator's job is to refuse files that the engine
// (packages/automation, landing in Phase 2) couldn't run sensibly.
//
// Rules per entry — references match the comments inline below:
//
//	(1) Name slug — same regex as world names (^[a-z][a-z0-9-]*$).
//	(2) Trigger XOR — exactly one of `on.cron` / `on.fs`.
//	(3) Cron parses — standard 5-field grammar (matches launchd's
//	    StartCalendarInterval semantics, the prior art in jterrazz-os).
//	(4) fs.path resolves on disk — LevelWarning when missing, since
//	    a typo is far more common than "the dir will be created later"
//	    and the engine would silently never fire otherwise.
//	(5) fs.events ⊂ {create, write, rename}. "remove" is intentionally
//	    unsupported in v1 — agents reacting to deletions almost always
//	    need "what was the file" context the watcher can't provide.
//	(6) fs.debounce ∈ [100ms, 1h]. Below 100ms the engine spends more
//	    time coalescing than firing; above 1h burst protection becomes
//	    a worse latency penalty than the burst itself.
//	(7) fs.patterns are non-empty strings. Glob syntax validation is
//	    deferred to engine-time — doublestar's grammar is permissive
//	    enough that errors here would be false positives.
//	(8) Body XOR — exactly one of `prompt` / `command`.
//	(9) command refs use the `command/<name>` shape and resolve to
//	    spwn/commands/<name>.md on disk.
//	(10) catchup ∈ {"", "collapse", "skip"}. Cron-only — flagged as
//	    LevelInfo on fs (no semantic effect, but worth telling the
//	    user we ignored it).
//	(11) agent ∈ world.agents (the engine picks the agent from inside
//	    the world's container, so any other name would dispatch to
//	    nowhere).
func ruleAutomations(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	for _, wname := range sortedKeys(in.Manifest.Worlds) {
		w := in.Manifest.Worlds[wname]
		if len(w.Automations) == 0 {
			continue
		}
		// World-local agent set powers the membership check (11).
		agentSet := map[string]struct{}{}
		for _, a := range w.Agents {
			agentSet[a] = struct{}{}
		}

		for _, aname := range sortedAutomationKeys(w.Automations) {
			a := w.Automations[aname]
			pathPrefix := "spwn.yaml#worlds." + wname + ".automations." + aname

			// (1) Slug.
			if !slugRe.MatchString(aname) {
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix,
					Message: fmt.Sprintf("automation name %q must match %s", aname, slugRe.String()),
					Hint:    "use a kebab-case slug like \"morning-brief\" or \"inbox-pull\"",
				})
			}

			// (2) Trigger XOR.
			hasCron := strings.TrimSpace(a.On.Cron) != ""
			hasFS := a.On.FS != nil
			switch {
			case hasCron && hasFS:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix + ".on",
					Message: "exactly one trigger required, got both `cron` and `fs`",
					Hint:    "split into two automations or pick the event you actually want",
				})
			case !hasCron && !hasFS:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix + ".on",
					Message: "trigger missing — set `on.cron` or `on.fs`",
					Hint:    "see docs/automations.md for the supported triggers",
				})
			}

			// (3) Cron expression parses.
			if hasCron {
				if _, err := parser.Parse(a.On.Cron); err != nil {
					out = append(out, Issue{
						Level:   LevelError,
						Path:    pathPrefix + ".on.cron",
						Message: fmt.Sprintf("invalid cron expression %q: %v", a.On.Cron, err),
						Hint:    "use 5 fields (min hour dom month dow) — e.g. \"0 6 * * *\" for daily 6am",
					})
				}
			}

			// (4)-(7) FS-trigger validation.
			if hasFS {
				out = append(out, validateFS(in.Root, pathPrefix+".on.fs", a.On.FS)...)
			}

			// (8) Body XOR.
			hasPrompt := strings.TrimSpace(a.Prompt) != ""
			hasCommand := strings.TrimSpace(a.Command) != ""
			switch {
			case hasPrompt && hasCommand:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix,
					Message: "exactly one body required, got both `prompt` and `command`",
					Hint:    "drop one — `prompt:` is for inline bodies, `command:` references spwn/commands/<name>.md",
				})
			case !hasPrompt && !hasCommand:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix,
					Message: "body missing — set `prompt` or `command`",
					Hint:    "inline prose goes in `prompt:`, or use `command: command/<name>` to reuse a slash command",
				})
			}

			// (9) Command ref shape + resolution.
			if hasCommand {
				out = append(out, validateAutomationCommandRef(in.Root, pathPrefix+".command", a.Command)...)
			}

			// (10) Catchup mode + cron-only enforcement.
			switch a.Catchup {
			case "", "collapse", "skip":
				// OK
			default:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix + ".catchup",
					Message: fmt.Sprintf("unknown catchup mode %q", a.Catchup),
					Hint:    "use \"collapse\" (default — single fire on resume) or \"skip\" (drop missed slots)",
				})
			}
			if a.Catchup != "" && hasFS && !hasCron {
				out = append(out, Issue{
					Level:   LevelInfo,
					Path:    pathPrefix + ".catchup",
					Message: "catchup is cron-only and has no effect on fs triggers",
					Hint:    "fs watchers always replay the seen-list diff on resume; remove the `catchup:` key here",
				})
			}

			// (11) Agent membership.
			if a.Agent == "" {
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix + ".agent",
					Message: "automation must declare an `agent:` to wake",
					Hint:    "set `agent:` to one of the world's agents (" + strings.Join(w.Agents, ", ") + ")",
				})
			} else if _, ok := agentSet[a.Agent]; !ok {
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix + ".agent",
					Message: fmt.Sprintf("agent %q is not in world %q", a.Agent, wname),
					Hint:    fmt.Sprintf("add %q to worlds.%s.agents, or pick one of: %s", a.Agent, wname, strings.Join(w.Agents, ", ")),
				})
			}
		}
	}
	return out
}

// validateFS owns the inner checks for an fs trigger so the parent
// rule reads as a flat checklist. Returns one Issue per failed check;
// callers append directly to the parent's running slice.
func validateFS(root, pathPrefix string, fs *intmanifest.FSTrigger) []Issue {
	var out []Issue

	// fs.path required.
	if strings.TrimSpace(fs.Path) == "" {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix + ".path",
			Message: "fs trigger requires `path:`",
			Hint:    "set the host path to watch (project-relative paths resolve against the project root)",
		})
		// No point continuing the per-path checks below.
		return out
	}

	// fs.path resolves on disk (warn if missing — engine still wires
	// the watcher, just won't fire until the dir exists).
	resolved := fs.Path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(root, resolved)
	}
	if info, err := os.Stat(resolved); err != nil {
		out = append(out, Issue{
			Level:   LevelWarning,
			Path:    pathPrefix + ".path",
			Message: fmt.Sprintf("watch path %q does not exist on disk", fs.Path),
			Hint:    fmt.Sprintf("create the directory (e.g. `mkdir -p %s`) before `spwn up`, or fix the path", fs.Path),
		})
	} else if !info.IsDir() {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix + ".path",
			Message: fmt.Sprintf("watch path %q is a file, not a directory", fs.Path),
			Hint:    "fs triggers watch directories — use the parent directory and `patterns:` to filter",
		})
	}

	// fs.events allow-list. The default ([create]) was applied by
	// ApplyDefaults; here we only need to validate explicit values.
	allowed := map[string]struct{}{
		"create": {},
		"write":  {},
		"rename": {},
	}
	for i, ev := range fs.Events {
		if _, ok := allowed[ev]; !ok {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    fmt.Sprintf("%s.events[%d]", pathPrefix, i),
				Message: fmt.Sprintf("unknown fs event %q", ev),
				Hint:    "use one of: create, write, rename",
			})
		}
	}

	// fs.debounce range. Zero is the apply-defaults sentinel — already
	// substituted to 1s; explicit zero from the author would also have
	// landed here, which is fine: validating against the post-default
	// state matches what the engine actually runs.
	d := fs.Debounce.AsDuration()
	if d < 100*time.Millisecond {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix + ".debounce",
			Message: fmt.Sprintf("debounce %s is below the 100ms minimum", d),
			Hint:    "set debounce ≥ 100ms — below that the engine coalesces every keystroke and never fires",
		})
	}
	if d > 1*time.Hour {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix + ".debounce",
			Message: fmt.Sprintf("debounce %s exceeds the 1h maximum", d),
			Hint:    "use a cron trigger for hourly+ cadences instead of stretching debounce",
		})
	}

	// fs.patterns — string non-emptiness only. Doublestar's grammar
	// is too permissive to validate fully here without false positives.
	for i, p := range fs.Patterns {
		if strings.TrimSpace(p) == "" {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    fmt.Sprintf("%s.patterns[%d]", pathPrefix, i),
				Message: "pattern must be a non-empty glob",
				Hint:    "remove the entry, or use \"*.md\" / \"prefix-*\" / similar",
			})
		}
	}

	return out
}

// validateAutomationCommandRef checks the `command:` shape and that
// the referenced markdown file exists. The naming intentionally
// avoids `validateCommandRef` to leave room for the slash-command
// validator (commands consumed by agents directly), which lives
// elsewhere in this package.
func validateAutomationCommandRef(root, pathPrefix, ref string) []Issue {
	var out []Issue
	const prefix = "command/"
	if !strings.HasPrefix(ref, prefix) {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix,
			Message: fmt.Sprintf("command ref %q must use the `command/<name>` form", ref),
			Hint:    "match the slash-command scheme — e.g. `command/morning-brief` for spwn/commands/morning-brief.md",
		})
		return out
	}
	name := strings.TrimPrefix(ref, prefix)
	if !slugRe.MatchString(name) {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix,
			Message: fmt.Sprintf("command name %q must match %s", name, slugRe.String()),
			Hint:    "use a kebab-case slug — `command/morning-brief`, not `command/MorningBrief`",
		})
		return out
	}
	target := filepath.Join(root, "spwn", "commands", name+".md")
	if _, err := os.Stat(target); err != nil {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix,
			Message: fmt.Sprintf("command file not found at spwn/commands/%s.md", name),
			Hint:    fmt.Sprintf("create the file or fix the ref — `spwn install command/%s --agent <name>` scaffolds one", name),
		})
	}
	return out
}

// sortedAutomationKeys returns the automation map's keys in stable
// alphabetical order so issues emit deterministically (test goldens
// stay portable).
func sortedAutomationKeys(autos map[string]intmanifest.Automation) []string {
	out := make([]string, 0, len(autos))
	for k := range autos {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
