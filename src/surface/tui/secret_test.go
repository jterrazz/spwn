package tui

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	pagesettings "github.com/jterrazz/studio-station/src/pages/settings"
	settingsscreen "github.com/jterrazz/studio-station/src/surface/tui/screens/settings"
)

func builtinSettings(t *testing.T) []catalog.Setting {
	t.Helper()
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	return c.Settings
}

const pasted = "sk-ant-oat01-kept-nowhere"

// invariant: TU-25: the token typed in claude-token's form reaches its
// command on stdin only, through the question it asks first, and once the
// command has run the shell holds no byte of it.
func TestSecretIsHandedNeverKept(t *testing.T) {
	var handed []map[string]string
	var words [][]string
	do := func(_ context.Context, w []string, a Answer) ([]string, error) {
		handed, words = append(handed, a.Values), append(words, w)
		if !a.Yes {
			return nil, &Asked{Message: "claude-token changed since it was applied"}
		}
		return nil, nil
	}
	s := newShell(t.Context(), Deps{ThisMachine: page.Context{Machine: "jterrazz-macbook-pro"}, Do: do})
	g, ok := settingsscreen.Gesture(builtinSettings(t), page.Action{Key: pagesettings.KeyApply}, []page.Row{{ID: "claude-token", Label: "claude-token"}})
	if !ok || g.Values == nil {
		t.Fatal("a on claude-token opens no form")
	}
	s, _ = s.perform(g)
	for _, r := range pasted {
		s = press(t, s, tea.Key{Code: r, Text: string(r)})
	}
	if drawn := s.dialogPanel(); drawn == nil || strings.Contains(strings.Join(drawn.Lines, "\n"), pasted) {
		t.Fatalf("the form echoes the token: %v", drawn)
	}
	s = press(t, s, tea.Key{Code: tea.KeyEnter})
	if s.acts.modal == nil {
		t.Fatal("the command's question is not put")
	}
	s = press(t, s, tea.Key{Code: 'y', Text: "y"})
	if len(handed) != 2 || handed[0]["token"] != pasted || handed[1]["token"] != pasted {
		t.Fatalf("the command was handed %v, want the token both times", handed)
	}
	for _, w := range words {
		if slices.ContainsFunc(w, func(x string) bool { return strings.Contains(x, pasted) }) {
			t.Fatalf("the token is in the command's words: %v", w)
		}
	}
	if holds(reflect.ValueOf(s), pasted, map[uintptr]bool{}) {
		t.Fatal("the shell still holds the token once the command has run")
	}
}

// value: the shell after a key and every message the key's commands send,
// the reads it starts left unanswered.
func press(t *testing.T, s shell, k tea.Key) shell {
	t.Helper()
	m, cmd := s.update(tea.KeyPressMsg(k))
	s, _ = m.(shell)
	if cmd == nil {
		return s
	}
	if msg, ok := cmd().(stepped); ok {
		m, _ = s.update(msg)
		s, _ = m.(shell)
	}
	return s
}

func holds(v reflect.Value, secret string, seen map[uintptr]bool) bool {
	switch v.Kind() {
	case reflect.String:
		return strings.Contains(v.String(), secret)
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return false
		}
		if v.Kind() == reflect.Pointer {
			if seen[v.Pointer()] {
				return false
			}
			seen[v.Pointer()] = true
		}
		return holds(v.Elem(), secret, seen)
	case reflect.Struct:
		for i := range v.NumField() {
			if holds(v.Field(i), secret, seen) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if holds(v.Index(i), secret, seen) {
				return true
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			if holds(k, secret, seen) || holds(v.MapIndex(k), secret, seen) {
				return true
			}
		}
	default:
	}
	return false
}
