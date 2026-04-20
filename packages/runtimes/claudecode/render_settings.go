package claudecode

import (
	"encoding/json"
	"sort"

	"spwn.sh/packages/transpile"
)

// SettingsInput carries the per-agent knobs the renderer folds into
// `.claude/settings.json`. Growing this struct (instead of adding
// arguments to GenerateAgentSettingsJSON) keeps the render API stable
// as more Claude Code settings become first-class spwn primitives.
type SettingsInput struct {
	// Hooks from Input.Hooks — rendered into the JSON `hooks` map
	// under each event name. Unsupported events are passed through;
	// Claude Code tolerates unknown events silently.
	Hooks []transpile.HookEntry

	// Model mirrors agent.yaml#runtime.model. Written verbatim into
	// `.claude/settings.json#model` so Claude Code pins the agent to
	// the requested model at startup. Empty string → key omitted.
	Model string
}

// GenerateAgentSettingsJSON emits the canonical `.claude/settings.json`
// body for one agent. Contract:
//
//   - `skipDangerousModePermissionPrompt: true` is always on (spwn
//     containers are sandboxed; the prompt would block every
//     one-shot invocation).
//   - `hooks` is a map keyed by event name; each value is an array of
//     matcher+hooks entries matching Claude Code's documented schema
//     (https://code.claude.com/docs/en/hooks.md).
//
// Determinism: events are emitted in alphabetical order; hooks within
// each event preserve SettingsInput.Hooks' slice order (callers sort
// upstream if needed).
func GenerateAgentSettingsJSON(in SettingsInput) []byte {
	payload := map[string]any{
		"skipDangerousModePermissionPrompt": true,
	}
	if in.Model != "" {
		payload["model"] = in.Model
	}
	if hooks := buildHooksMap(in.Hooks); hooks != nil {
		payload["hooks"] = hooks
	}
	out, _ := json.MarshalIndent(payload, "", "  ")
	return append(out, '\n')
}

// buildHooksMap translates a flat []HookEntry into Claude Code's
// nested {event: [{matcher, hooks: [{type, command}]}]} shape. Returns
// nil when there are no hooks so the JSON doesn't carry an empty key.
func buildHooksMap(entries []transpile.HookEntry) map[string]any {
	if len(entries) == 0 {
		return nil
	}
	byEvent := map[string][]transpile.HookEntry{}
	for _, h := range entries {
		byEvent[h.Event] = append(byEvent[h.Event], h)
	}
	events := make([]string, 0, len(byEvent))
	for ev := range byEvent {
		events = append(events, ev)
	}
	sort.Strings(events)

	out := map[string]any{}
	for _, ev := range events {
		group := byEvent[ev]
		var rendered []any
		for _, h := range group {
			matcher := h.Matcher
			if matcher == "" {
				matcher = "*"
			}
			rendered = append(rendered, map[string]any{
				"matcher": matcher,
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": h.Command,
					},
				},
			})
		}
		out[ev] = rendered
	}
	return out
}
