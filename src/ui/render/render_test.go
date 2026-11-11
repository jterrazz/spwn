package render

import (
	"bytes"
	"errors"
	"flag"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/jterrazz/studio-station/src/page"
	pagejson "github.com/jterrazz/studio-station/src/ui/render/json"
	"github.com/jterrazz/studio-station/src/ui/render/pipe"
	"github.com/jterrazz/studio-station/src/ui/render/tty"
	"github.com/jterrazz/studio-station/src/ui/token"
)

var update = flag.Bool("update", false, "rewrite the goldens from the renderers")

const (
	pagesDir  = "../testdata/pages"
	goldenDir = "../testdata/__expected__"
)

type fixture struct {
	name  string
	state string
	page  page.Page
}

func fixtures(t *testing.T) []fixture {
	t.Helper()
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]fixture, 0, len(entries))
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(pagesDir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		p, err := page.Decode(body)
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		out = append(out, fixture{name: name, state: name[strings.LastIndexByte(name, '.')+1:], page: p})
	}
	return out
}

// invariant: the renderers each fixture is proved by, named as its golden files
// are; specs/architecture's rule 18 reads the same list.
func renderings(f fixture) map[string]func() (string, error) {
	out := map[string]func() (string, error){
		"tty100": func() (string, error) {
			return ansi.Strip(string(tty.Render(f.page, tty.Options{Width: 100, Dark: true}))), nil
		},
		"tty80": func() (string, error) {
			return ansi.Strip(string(tty.Render(f.page, tty.Options{Width: 80, Dark: true}))), nil
		},
		"json": func() (string, error) { return string(pagejson.Render(f.page)), nil },
		"pipe": func() (string, error) {
			body, err := pipe.Render(f.page)
			if errors.Is(err, pipe.ErrPending) {
				return "ErrPending\n", nil
			}
			return string(body), err
		},
	}
	if f.state == "full" {
		out["ttylight100"] = func() (string, error) { return string(tty.Render(f.page, tty.Options{Width: 100, Dark: false})), nil }
	}
	if strings.HasPrefix(f.name, "kitchen-sink.") {
		out["tty40"] = func() (string, error) {
			return ansi.Strip(string(tty.Render(f.page, tty.Options{Width: 40, Dark: true}))), nil
		}
	}
	return out
}

func TestEveryFixtureRendersAsItsGoldenInEveryMedium(t *testing.T) {
	for _, f := range fixtures(t) {
		for renderer, render := range renderings(f) {
			t.Run(f.name+"."+renderer, func(t *testing.T) {
				got, err := render()
				if err != nil {
					t.Fatal(err)
				}
				golden(t, f.name+"."+renderer, got)
			})
		}
	}
}

func TestNoLineIsWiderThanItsTerminal(t *testing.T) {
	for _, f := range fixtures(t) {
		for renderer, render := range renderings(f) {
			width, _ := strconv.Atoi(strings.TrimLeft(renderer, "abcdefghijklmnopqrstuvwxyz"))
			if width < token.NarrowWidth || !strings.HasPrefix(renderer, "t") {
				continue
			}
			got, err := render()
			if err != nil {
				t.Fatal(err)
			}
			for i, line := range strings.Split(ansi.Strip(got), "\n") {
				if w := ansi.StringWidth(line); w > width-1 {
					t.Errorf("%s.%s:%d is %d wide at %d: %q", f.name, renderer, i+1, w, width, line)
				}
			}
		}
	}
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join(goldenDir, name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing (go test ./src/ui/... -update writes it)", name)
	}
	if got != string(want) {
		t.Errorf("%s differs (go test ./src/ui/... -update rewrites it):\n%s\nwant:\n%s", name, got, want)
	}
}

func TestTheJSONFormRoundTrips(t *testing.T) {
	for _, f := range fixtures(t) {
		again, err := page.Decode(pagejson.Render(f.page))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(pagejson.Render(again), pagejson.Render(f.page)) {
			t.Errorf("%s does not survive a decode", f.name)
		}
	}
}

func TestThePipeRefusesAPendingPage(t *testing.T) {
	for _, f := range fixtures(t) {
		_, err := pipe.Render(f.page)
		if pending := f.page.State == page.Loading; pending != errors.Is(err, pipe.ErrPending) {
			t.Errorf("%s (state %s): pipe answered %v", f.name, f.page.State, err)
		}
	}
}

func TestALoanSaysHowLongItHasLeftInEveryMedium(t *testing.T) {
	for _, f := range fixtures(t) {
		for _, s := range f.page.Sections {
			for _, it := range s.Items {
				if it.Row == nil || it.Row.Left == nil {
					continue
				}
				left := " left"
				for medium, render := range map[string]func() (string, error){
					"tty100": func() (string, error) {
						return ansi.Strip(string(tty.Render(f.page, tty.Options{Width: 100, Dark: true}))), nil
					},
					"pipe": func() (string, error) { b, err := pipe.Render(f.page); return string(b), err },
				} {
					out, err := render()
					if err != nil {
						t.Fatal(err)
					}
					if !slices.ContainsFunc(strings.Split(out, "\n"), func(l string) bool { return strings.Contains(l, it.Row.Label) && strings.Contains(l, left) }) {
						t.Errorf("%s %s: %s does not say %q", f.name, medium, it.Row.Label, left)
					}
				}
			}
		}
	}
}

func TestEveryHintIsWholeInEveryMedium(t *testing.T) {
	for _, f := range fixtures(t) {
		hints, piped := map[string]bool{}, map[string]bool{}
		var walk func([]page.Item)
		walk = func(items []page.Item) {
			for _, it := range items {
				switch {
				case it.Row != nil && it.Row.Hint != nil:
					hints[it.Row.Hint.Command] = true
				case it.Group != nil:
					if it.Group.Head.Hint != nil {
						hints[it.Group.Head.Hint.Command] = true
					}
					walk(it.Group.Items)
				}
			}
		}
		for _, s := range f.page.Sections {
			if s.Fold == page.Folded {
				continue
			}
			if s.Fold == page.Brief || s.Fold == page.Calm {
				// value: a brief or calm section is one line on a terminal: its
				// rows' hints are whole in the pipe, which lists them.
				all := hints
				hints = map[string]bool{}
				walk(s.Items)
				maps.Copy(piped, hints)
				hints = all
				continue
			}
			all := hints
			hints = map[string]bool{}
			walk(s.Items)
			// value: the pipe prints no Actions, Summary or Lifted section: their
			// hints are whole on the eye's media only.
			if s.Role != page.Actions && s.Role != page.Summary && s.Role != page.Lifted {
				maps.Copy(piped, hints)
			}
			maps.Copy(all, hints)
			hints = all
		}
		mediums := map[string]string{
			"tty80":  joined(ansi.Strip(string(tty.Render(f.page, tty.Options{Width: 80, Dark: true})))),
			"tty100": joined(ansi.Strip(string(tty.Render(f.page, tty.Options{Width: 100, Dark: true})))),
		}
		if body, err := pipe.Render(f.page); err == nil {
			mediums["pipe"] = string(body)
		}
		for medium, text := range mediums {
			want := hints
			if medium == "pipe" {
				want = piped
			}
			for h := range want {
				if !strings.Contains(text, h) {
					t.Errorf("%s %s: the hint %q is cut or dropped", f.name, medium, h)
				}
			}
		}
	}
}

// invariant: G5: a hint on its own line starts at its row's label column, in
// every section, medium and width, whatever the section's widest cell.
func TestAWrappedHintSitsAtTheLabelColumn(t *testing.T) {
	marks := []rune(token.MarkOK + token.MarkAttention + token.MarkAbsent + token.MarkBroken + token.MarkPending)
	for _, f := range fixtures(t) {
		for medium, render := range renderings(f) {
			if !strings.HasPrefix(medium, "tty") || medium == "tty40" {
				continue
			}
			text, err := render()
			if err != nil {
				continue
			}
			label := -1
			for n, line := range strings.Split(text, "\n") {
				runes := []rune(line)
				trimmed := strings.TrimLeft(line, " ")
				first := len(line) - len(trimmed)
				switch {
				case strings.HasPrefix(trimmed, token.Arrow+" "):
					if label >= 0 && first != label {
						t.Errorf("%s %s line %d: the hint starts at column %d, its row's label at %d", f.name, medium, n+1, first, label)
					}
				case strings.TrimSpace(line) == "":
					label = -1
				default:
					label = -1
					for i := first; i < len(runes); i++ {
						if slices.Contains(marks, runes[i]) && i+2 < len(runes) && runes[i+1] == ' ' {
							label = i + 2
							break
						}
						if !strings.ContainsRune("├└│─▌ ", runes[i]) {
							break
						}
					}
				}
			}
		}
	}
}

// value: a chained command broken after its `&&` reads whole again (G5).
var joints = regexp.MustCompile(`&&\n +`)

func joined(text string) string { return joints.ReplaceAllString(text, "&& ") }

type span struct {
	fg   string
	text string
}

var sgr = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func spans(s string) []span {
	var out []span
	fg := ""
	at := 0
	for _, m := range sgr.FindAllStringSubmatchIndex(s, -1) {
		if m[0] > at {
			out = append(out, span{fg: fg, text: s[at:m[0]]})
		}
		params := strings.Split(s[m[2]:m[3]], ";")
		for i := 0; i < len(params); i++ {
			switch params[i] {
			case "", "0", "39":
				fg = ""
			case "38":
				if i+4 < len(params) && params[i+1] == "2" {
					fg = strings.Join(params[i+2:i+5], ";")
					i += 4
				}
			}
		}
		at = m[1]
	}
	return append(out, span{fg: fg, text: s[at:]})
}

func rgb(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return strconv.Itoa(int(r>>8)) + ";" + strconv.Itoa(int(g>>8)) + ";" + strconv.Itoa(int(b>>8))
}

func TestStateColoursAreOnMarksAndLinesOnRules(t *testing.T) {
	for _, dark := range []bool{true, false} {
		c := token.Colors(dark)
		state := []string{rgb(c.Good), rgb(c.Warn), rgb(c.Bad)}
		marks := token.MarkOK + token.Tick + token.MarkAttention + token.MarkAbsent + token.MarkBroken + token.MarkPending + "0123456789 "
		lines := token.Rule + token.Divider + "├└ "
		notFor := token.MarkNotFor + "0123456789 "
		for _, f := range fixtures(t) {
			for _, out := range []string{string(tty.Render(f.page, tty.Options{Width: 100, Dark: dark}))} {
				for _, s := range spans(out) {
					text := strings.TrimSpace(s.text)
					if text == "" {
						continue
					}
					if slices.Contains(state, s.fg) && text != "Error:" && strings.Trim(text, marks) != "" {
						t.Errorf("%s: a state color on %q (K1)", f.name, text)
					}
					if s.fg == rgb(c.Subtle) && strings.Trim(text, lines) != "" && strings.Trim(text, notFor) != "" {
						t.Errorf("%s: Subtle on text %q (K2)", f.name, text)
					}
					if tone, ok := map[string]string{token.MarkOffHere: rgb(c.Muted), token.MarkNotFor: rgb(c.Subtle)}[text]; ok && s.fg != tone {
						t.Errorf("%s: the mark %q is not in its token (SPEC §2.3)", f.name, text)
					}
				}
			}
		}
	}
}

func TestTheKitchenSinkHoldsEveryNodeAndValueKind(t *testing.T) {
	var full page.Page
	for _, f := range fixtures(t) {
		if f.name == "kitchen-sink.full" {
			full = f.page
		}
	}
	seen := map[string]bool{}
	var walk func([]page.Item, int)
	walk = func(items []page.Item, depth int) {
		for _, it := range items {
			switch {
			case it.Row != nil:
				seen["row"], seen["mark "+string(it.Row.Mark)], seen["reason "+string(it.Row.Reason)] = true, true, true
				for _, c := range it.Row.Cells {
					seen["kind "+string(c.Kind)] = true
				}
			case it.Group != nil:
				seen["group"], seen["depth "+strconv.Itoa(depth)] = true, true
				walk(it.Group.Items, depth+1)
			case it.Fact != nil:
				seen["fact"] = true
			case it.Empty != nil:
				seen["empty"] = true
			case it.Error != nil:
				seen["error"] = true
			case it.Note != nil:
				seen["note"] = true
			case it.Link != nil:
				seen["link"] = true
			}
		}
	}
	for _, s := range full.Sections {
		walk(s.Items, 1)
		seen["role "+string(s.Role)] = true
		if len(s.Caption) > 0 {
			seen["caption"] = true
		}
	}
	want := make([]string, 0, 64)
	want = append(want, "row", "group", "depth 2", "fact", "empty", "error", "note", "link", "caption", "role actions", "role summary", "role banner", "role ")
	for _, m := range []page.Mark{page.OK, page.Attention, page.Absent, page.Broken} {
		want = append(want, "mark "+string(m))
	}
	for _, r := range []page.Reason{page.Outdated, page.Misplaced, page.Unreachable, page.Mismatch, page.Unmanaged, page.Missing, page.Partial, page.Differs, page.Unset, page.Stopped, page.Late, page.Degraded, page.Orphan, page.Waiting, page.Failing, page.Refused, page.Down, page.Outlived, page.Expiring, page.Guard} {
		want = append(want, "reason "+string(r))
	}
	for _, k := range []page.Kind{page.KindText, page.KindCount, page.KindSize, page.KindRatio, page.KindPercent, page.KindVersion, page.KindDuration, page.KindAge, page.KindUntil, page.KindURL, page.KindIdentity, page.KindPath, page.KindPort, page.KindPortMap, page.KindRange, page.KindCommand, page.KindList, page.KindLevels, page.KindOn, page.KindOff, page.KindUnknown} {
		want = append(want, "kind "+string(k))
	}
	for _, w := range want {
		if !seen[w] && (w != "kind "+string(page.KindRange) || !seen["caption"]) {
			t.Errorf("the kitchen sink holds no %s", w)
		}
	}
	if full.Strip == nil || len(full.Actions) == 0 || full.Actions[0].Confirm == nil {
		t.Error("the kitchen sink holds no strip, action or confirm")
	}
}

func TestEveryGlyphPrintedIsAToken(t *testing.T) {
	known := strings.Join(token.Glyphs(), "")
	for _, f := range fixtures(t) {
		for renderer, render := range renderings(f) {
			got, _ := render()
			for _, r := range ansi.Strip(got) {
				if r > 127 && !strings.ContainsRune(known, r) {
					t.Errorf("%s.%s prints %q, which no token names", f.name, renderer, r)
				}
			}
		}
	}
}
