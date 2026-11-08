package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

type banned struct {
	word string
	// value: the word as written counts only in that case (`Tasks` for the
	// strip, `NAT`), not as the plain English word it also is.
	exact bool
	// value: the phrases SPEC §2.2 keeps, taken out of a text before it is
	// read, and another program's own command line (`git config`, a fix the
	// owner types).
	allowed []string
	// value: the word's other forms, each the same retired word (`lent`
	// for `lend`).
	forms []string
	// value: a word that is also plain English ("off here", "set by station
	// here") is retired only where it names something: `here` as a machine's
	// name, a value of its own or before the Mac's version.
	only string
}

// invariant: rule words, SPEC §2.2: the words v3 retired, each for the
// concept it used to name, with the phrases it keeps; the menu bar's v4
// contract keeps two gestures by name, "Lend an account" and "Clean up".
func bannedWords() []banned {
	return []banned{
		{word: "host", allowed: []string{"host:port", "host key", "known host", "host servers"}},
		{word: "fleet", forms: []string{"fleets"}},
		{word: "here", only: `^here$|\bhere v\d`},
		{word: "daemon", forms: []string{"daemons"}},
		{word: "tool"},
		{word: "tools"},
		{word: "config", allowed: []string{".config/", "git config", "ssh config", "ssh_config"}},
		{word: "storage"},
		{word: "service", forms: []string{"services"}},
		{word: "lease", forms: []string{"leases", "leased"}},
		{word: "publish", forms: []string{"published", "publishes", "publishing"}},
		{word: "lend", forms: []string{"lent", "lends", "lending"}, allowed: []string{"lend an account", "lent to"}},
		{word: "recall", forms: []string{"recalled", "recalls"}},
		{word: "wish", forms: []string{"wishes"}},
		{word: "probe", forms: []string{"probes", "probed"}},
		{word: "temporary"},
		{word: "lasting"},
		{word: "stray", forms: []string{"strays"}},
		{word: "clean", allowed: []string{"dclean", "clean up"}},
		{word: "reset", allowed: []string{"dock-reset", "dock reset", "dreset", "counter reset"}},
		{word: "power", allowed: []string{"power cut", "power-cut", "power source", "low power", "low-power", "power off", "power management", "on power"}},
		{word: "place", forms: []string{"places"}},
		{word: "Resources", exact: true},
		{word: "Tasks", exact: true},
		{word: "space", forms: []string{"spaces"}},
		{word: "sandbox", forms: []string{"sandboxes"}},
		{word: "production"},
		{word: "OrbStack", exact: true},
		{word: "a check"},
		{word: "path", forms: []string{"paths"}, exact: true},
		{word: "VM engine"},
		{word: "NAT", exact: true},
		{word: "tailnet", forms: []string{"tailnets"}},
	}
}

// value: the fields of a page that name its machinery, never read: ids,
// kinds, the calls and the inputs an action fixes.
func unread() []string {
	return []string{"id", "key", "kind", "call", "page", "row", "schema", "mark", "reason", "fold", "role", "show", "state", "on", "at", "value"}
}

// value: the fields whose whole value is machinery: the rows an action
// acts on, a call's input, a program's arguments.
func unreadTrees() []string { return []string{"rows", "in", "argv"} }

func visible(v any, skip bool, out *[]string) {
	switch x := v.(type) {
	case string:
		if !skip {
			*out = append(*out, x)
		}
	case []any:
		for _, it := range x {
			visible(it, skip, out)
		}
	case map[string]any:
		for k, it := range x {
			_, text := it.(string)
			visible(it, skip || slices.Contains(unreadTrees(), k) || text && slices.Contains(unread(), k), out)
		}
	}
}

var snakeRe = regexp.MustCompile(`\b[a-z0-9]+(?:_[a-z0-9]+)+\b`)

// invariant: a snake_case word is its words (`power_ended`, `fleet_size`);
// an environment variable's name (`DOCKER_CONFIG`) is another program's.
func snakeWords(t string) string {
	return snakeRe.ReplaceAllStringFunc(t, func(w string) string { return strings.ReplaceAll(w, "_", " ") })
}

func bannedIn(texts []string) []string { return bannedInList(bannedWords(), texts) }

func bannedInList(list []banned, texts []string) []string {
	var found []string
	for _, b := range list {
		words := []string{regexp.QuoteMeta(b.word)}
		for _, f := range b.forms {
			words = append(words, regexp.QuoteMeta(f))
		}
		pattern := `\b(?:` + strings.Join(words, "|") + `)\b`
		if b.only != "" {
			pattern = b.only
		}
		if !b.exact {
			pattern = `(?i)` + pattern
		}
		re := regexp.MustCompile(pattern)
		for _, t := range texts {
			t = snakeWords(t)
			for _, a := range b.allowed {
				t = regexp.MustCompile(`(?i)`+regexp.QuoteMeta(a)).ReplaceAllString(t, "")
			}
			if re.MatchString(t) {
				found = append(found, b.word)
				break
			}
		}
	}
	return found
}

func bannedFindings(root string) (map[string][]string, error) {
	dir := filepath.Join(root, "src", "ui", "testdata", "pages")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var doc any
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, err
		}
		var texts []string
		visible(doc, false, &texts)
		family := strings.SplitN(e.Name(), ".", 2)[0]
		for _, w := range bannedIn(texts) {
			if !slices.Contains(out[family], w) {
				out[family] = append(out[family], w)
			}
		}
	}
	for _, words := range out {
		sort.Strings(words)
	}
	return out, nil
}

func TestBannedWords(t *testing.T) {
	found, err := bannedFindings(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for family, words := range found {
		for _, w := range words {
			t.Errorf("src/ui/testdata/pages/%s.*.json says %q, a word SPEC §2.2 retired", family, w)
		}
	}
}

func TestBannedWordsBite(t *testing.T) {
	page := map[string]any{
		"id":       "fleet",
		"context":  map[string]any{"kind": "host"},
		"sections": []any{map[string]any{"title": "The fleet", "items": []any{map[string]any{"row": map[string]any{"label": "Dock reset", "cells": []any{map[string]any{"text": "storage"}, map[string]any{"text": "lent"}}}}}}},
	}
	var texts []string
	visible(page, false, &texts)
	if got := bannedIn(texts); !slices.Equal(got, []string{"fleet", "storage", "lend"}) {
		t.Errorf("a planted page says %v, want [fleet storage lend]: the id, the kind and Dock reset are not read, lent is lend", got)
	}
}

func TestBannedWordsBiteOnTheirForms(t *testing.T) {
	for _, text := range []string{"2 strays", "3 services", "the leases", "published", "two daemons", "probes", "fleet_size", "leased_port", "power_ended"} {
		if got := bannedIn([]string{text}); len(got) != 1 {
			t.Errorf("%q says %v, want one retired word: a form is the word it comes from", text, got)
		}
	}
}
