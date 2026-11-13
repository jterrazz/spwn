package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

func kinds() map[string]agents.Kind {
	return map[string]agents.Kind{
		"SessionStart":     agents.Began,
		"UserPromptSubmit": agents.Prompted,
		"PreToolUse":       agents.UseStarting,
		"PostToolUse":      agents.UseDone,
		"Notification":     agents.Notified,
		"Stop":             agents.Stopped,
		"SessionEnd":       agents.Ended,
	}
}

func act(used string) agents.Act {
	switch used {
	case "Edit", "Write", "MultiEdit", "NotebookEdit":
		return agents.ActEdit
	case "Read":
		return agents.ActRead
	case "Bash":
		return agents.ActRun
	case "WebFetch", "WebSearch":
		return agents.ActWeb
	case "Task":
		return agents.ActDelegate
	}
	return agents.ActOther
}

// value: the value at a JSON pointer (RFC 6901) of a payload, each key as
// Claude Code names it; nil where the payload has none.
func at(doc any, pointer string) any {
	for _, key := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		switch node := doc.(type) {
		case map[string]any:
			doc = node[key]
		default:
			return nil
		}
	}
	return doc
}

func text(doc any, pointer string) string {
	s, _ := at(doc, pointer).(string)
	return s
}

func lines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimSuffix(s, "\n"), "\n") + 1
}

func (h *Hooks) Event(_ context.Context, data []byte) (agents.Event, bool) {
	var doc any
	if json.Unmarshal(data, &doc) != nil {
		return agents.Event{}, false
	}
	session := text(doc, "/session_id")
	kind, ok := kinds()[text(doc, "/hook_event_name")]
	if !ok || session == "" {
		return agents.Event{}, false
	}
	e := agents.Event{
		Kind: kind, App: ID, Session: session, Folder: text(doc, "/cwd"), At: h.now(),
		Prompt: text(doc, "/prompt"), Message: text(doc, "/message"), Reason: text(doc, "/reason"),
	}
	if kind == agents.UseStarting || kind == agents.UseDone {
		e.Act, e.Path, e.Command = act(text(doc, "/tool_name")), text(doc, "/tool_input/file_path"), text(doc, "/tool_input/command")
	}
	if kind == agents.UseDone {
		e.Output = output(at(doc, "/tool_response"))
		e.Added = lines(text(doc, "/tool_input/new_string")) + lines(text(doc, "/tool_input/content"))
		e.Removed = lines(text(doc, "/tool_input/old_string"))
		edits, _ := at(doc, "/tool_input/edits").([]any)
		for _, edit := range edits {
			e.Added, e.Removed = e.Added+lines(text(edit, "/new_string")), e.Removed+lines(text(edit, "/old_string"))
		}
	}
	if kind == agents.Stopped {
		e.Message = text(doc, "/last_assistant_message")
		if e.Message == "" {
			e.Message = h.lastSaid(text(doc, "/transcript_path"))
		}
	}
	return e, true
}

func output(response any) string {
	if s, ok := response.(string); ok {
		return s
	}
	ran := []string{text(response, "/stdout"), text(response, "/stderr")}
	return strings.Join(slices.DeleteFunc(ran, func(s string) bool { return s == "" }), "\n")
}

func (h *Hooks) lastSaid(transcript string) string {
	// invariant: the last assistant entry of the transcript, its text parts
	// joined; a transcript that cannot be read says nothing.
	if transcript == "" {
		return ""
	}
	data, err := h.files.ReadFile(transcript)
	if err != nil {
		return ""
	}
	entries := bytes.Split(data, []byte("\n"))
	for i := len(entries) - 1; i >= 0; i-- {
		var entry any
		if json.Unmarshal(entries[i], &entry) != nil || text(entry, "/type") != "assistant" {
			continue
		}
		parts, _ := at(entry, "/message/content").([]any)
		var said []string
		for _, part := range parts {
			if text(part, "/type") == "text" && text(part, "/text") != "" {
				said = append(said, text(part, "/text"))
			}
		}
		if len(said) > 0 {
			return strings.Join(said, "\n")
		}
	}
	return ""
}
