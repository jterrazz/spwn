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
		agentSet := map[string]struct{}{}
		for _, a := range w.Agents {
			agentSet[a] = struct{}{}
		}

		for _, aname := range sortedAutomationKeys(w.Automations) {
			a := w.Automations[aname]
			pathPrefix := "spwn.yaml#worlds." + wname + ".automations." + aname

			if !slugRe.MatchString(aname) {
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix,
					Message: fmt.Sprintf("automation name %q must match %s", aname, slugRe.String()),
					Hint:    "use a kebab-case slug like \"morning-brief\" or \"inbox-pull\"",
				})
			}

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
					Message: "trigger missing: set `on.cron` or `on.fs`",
					Hint:    "see docs/guides/automations.md for the supported triggers",
				})
			}

			if hasCron {
				schedule, err := parser.Parse(a.On.Cron)
				if err != nil {
					out = append(out, Issue{
						Level:   LevelError,
						Path:    pathPrefix + ".on.cron",
						Message: fmt.Sprintf("invalid cron expression %q: %v", a.On.Cron, err),
						Hint:    "use 5 fields (min hour dom month dow), e.g. \"0 6 * * *\" for daily 6am",
					})
				} else {
					next := schedule.Next(time.Time{})
					afterNext := schedule.Next(next)
					if afterNext.Sub(next) <= time.Minute {
						out = append(out, Issue{
							Level:   LevelWarning,
							Path:    pathPrefix + ".on.cron",
							Message: fmt.Sprintf("cron %q fires every minute: agents will be woken 1440×/day", a.On.Cron),
							Hint:    "if you meant \"once a day at midnight\", use \"0 0 * * *\". See https://crontab.guru",
						})
					}
				}
			}

			if hasFS {
				out = append(out, validateFS(in.Root, pathPrefix+".on.fs", a.On.FS)...)
			}

			hasPrompt := strings.TrimSpace(a.Prompt) != ""
			hasCommand := strings.TrimSpace(a.Command) != ""
			switch {
			case hasPrompt && hasCommand:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix,
					Message: "exactly one body required, got both `prompt` and `command`",
					Hint:    "drop one: `prompt:` is for inline bodies, `command:` references spwn/commands/<name>.md",
				})
			case !hasPrompt && !hasCommand:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix,
					Message: "body missing: set `prompt` or `command`",
					Hint:    "inline prose goes in `prompt:`, or use `command: command/<name>` to reuse a slash command",
				})
			}

			if hasCommand {
				out = append(out, validateAutomationCommandRef(in.Root, pathPrefix+".command", a.Command)...)
			}

			switch a.Catchup {
			case "", "collapse", "skip", "stack":
			default:
				out = append(out, Issue{
					Level:   LevelError,
					Path:    pathPrefix + ".catchup",
					Message: fmt.Sprintf("unknown catchup mode %q", a.Catchup),
					Hint:    "use \"collapse\" (single fire on resume; default), \"skip\" (drop missed slots), or \"stack\" (one fire per missed slot, capped at 100)",
				})
			}

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

func validateFS(root, pathPrefix string, fs *intmanifest.FSTrigger) []Issue {
	var out []Issue

	if strings.TrimSpace(fs.Path) == "" {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix + ".path",
			Message: "fs trigger requires `path:`",
			Hint:    "set the host path to watch (project-relative paths resolve against the project root)",
		})
		return out
	}

	pathHasGlob := strings.ContainsAny(fs.Path, "*?[")
	if pathHasGlob {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix + ".path",
			Message: fmt.Sprintf("watch path %q contains glob characters", fs.Path),
			Hint:    "the watcher watches directories; move the glob to `patterns:` and set `path:` to the parent dir",
		})
	}

	resolved := strings.TrimSpace(fs.Path)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(root, resolved)
	}
	if !pathHasGlob {
		if info, err := os.Stat(resolved); err != nil {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    pathPrefix + ".path",
				Message: fmt.Sprintf("watch path %q does not exist on disk", strings.TrimSpace(fs.Path)),
				Hint:    fmt.Sprintf("create the directory before `spwn up` (e.g. `mkdir -p %s`), or fix the path", strings.TrimSpace(fs.Path)),
			})
		} else if !info.IsDir() {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    pathPrefix + ".path",
				Message: fmt.Sprintf("watch path %q is a file, not a directory", fs.Path),
				Hint:    "fs triggers watch directories: use the parent directory and `patterns:` to filter",
			})
		}
	}

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

	d := fs.Debounce.AsDuration()
	if d < 100*time.Millisecond {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix + ".debounce",
			Message: fmt.Sprintf("debounce %s is below the 100ms minimum", d),
			Hint:    "set debounce ≥ 100ms; below that the engine coalesces every keystroke and never fires",
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

	for i, p := range fs.Patterns {
		field := fmt.Sprintf("%s.patterns[%d]", pathPrefix, i)
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    field,
				Message: "pattern must be a non-empty glob",
				Hint:    "remove the entry, or use \"*.md\" / \"prefix-*\" / similar",
			})
			continue
		}
		if strings.Contains(trimmed, "{") || strings.Contains(trimmed, "}") {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    field,
				Message: fmt.Sprintf("pattern %q uses brace expansion, which filepath.Match does not support", trimmed),
				Hint:    "split into multiple patterns, e.g. [\"*.md\", \"*.txt\"] instead of \"*.{md,txt}\"",
			})
			continue
		}
		if strings.Contains(trimmed, "**") {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    field,
				Message: fmt.Sprintf("pattern %q uses doublestar, which filepath.Match does not support", trimmed),
				Hint:    "set `recursive: true` on the trigger and use a basename pattern like \"*.md\"",
			})
			continue
		}
		if _, err := filepath.Match(trimmed, "x"); err != nil {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    field,
				Message: fmt.Sprintf("pattern %q is not a valid filepath.Match glob: %v", trimmed, err),
				Hint:    "supported: literal names, `*` (any chars except `/`), `?` (single char), `[abc]` character classes",
			})
		}
	}

	return out
}

func validateAutomationCommandRef(root, pathPrefix, ref string) []Issue {
	var out []Issue
	const prefix = "command/"
	if !strings.HasPrefix(ref, prefix) {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix,
			Message: fmt.Sprintf("command ref %q must use the `command/<name>` form", ref),
			Hint:    "match the slash-command scheme, e.g. `command/morning-brief` for spwn/commands/morning-brief.md",
		})
		return out
	}
	name := strings.TrimPrefix(ref, prefix)
	if !slugRe.MatchString(name) {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix,
			Message: fmt.Sprintf("command name %q must match %s", name, slugRe.String()),
			Hint:    "use a kebab-case slug: `command/morning-brief`, not `command/MorningBrief`",
		})
		return out
	}
	target := filepath.Join(root, "spwn", "commands", name+".md")
	if _, err := os.Stat(target); err != nil {
		out = append(out, Issue{
			Level:   LevelError,
			Path:    pathPrefix,
			Message: fmt.Sprintf("command file not found at spwn/commands/%s.md", name),
			Hint:    fmt.Sprintf("create the file or fix the ref: `spwn install command/%s --agent <name>` scaffolds one", name),
		})
	}
	return out
}

func sortedAutomationKeys(autos map[string]intmanifest.Automation) []string {
	out := make([]string, 0, len(autos))
	for k := range autos {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
