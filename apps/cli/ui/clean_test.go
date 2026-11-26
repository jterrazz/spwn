package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const (
	rlo  = "\xe2\x80\xae"
	lri  = "\xe2\x81\xa6"
	zwsp = "\xe2\x80\x8b"
	bom  = "\xef\xbb\xbf"
	lsep = "\xe2\x80\xa8"
	csi  = "\xc2\x9b"
)

func esc(code string) string { return `\` + code }

func TestCleanEscapesWhatCanRewriteOrHideAText(t *testing.T) {
	in := "a\x1b[2Jb\rc\nd\te" + rlo + "f" + lri + "g" + zwsp + "h" + bom + "i" + lsep + "j" + csi + "k\xffl"
	want := "a" + esc("x1b") + "[2Jb" + esc("r") + "c" + esc("n") + "d" + esc("t") + "e" + esc("u202e") + "f" + esc("u2066") +
		"g" + esc("u200b") + "h" + esc("ufeff") + "i" + esc("u2028") + "j" + esc("u009b") + "k" + esc("xff") + "l"
	if got := Clean(in); got != want {
		t.Fatalf("Clean = %s\nwant    %s", got, want)
	}
	printable := "~/a b/\xc3\xa9 \xe6\x9d\xb1\xe4\xba\xac \xcd\x8f"
	if got := Clean(printable); got != printable {
		t.Errorf("Clean changed printable text: %q", got)
	}
	if got := CleanLines("one \x1b[31m\nTwo"); got != "one "+esc("x1b")+"[31m\nTwo" {
		t.Errorf("CleanLines = %q", got)
	}
}

const hostile = "skill\x1b[2J\x1b[1;5r" + rlo + "name" + zwsp

func rawProjectBytes(out string) bool {
	return strings.Contains(out, "\x1b") || strings.Contains(out, rlo) || strings.Contains(out, zwsp)
}

func TestEveryWriterOfTheStepperAndTheTableCleansWhatItIsGiven(t *testing.T) {
	writers := map[string]func(s *Stepper){
		"Start":       func(s *Stepper) { s.Start(hostile) },
		"Done":        func(s *Stepper) { s.Done(hostile, hostile) },
		"Fail":        func(s *Stepper) { s.Fail(hostile, errors.New(hostile)) },
		"FailHint":    func(s *Stepper) { _ = s.FailHint(hostile, errors.New(hostile), hostile) },
		"Success":     func(s *Stepper) { s.Success(hostile) },
		"Warn":        func(s *Stepper) { s.Warn(hostile, hostile) },
		"WarnLabel":   func(s *Stepper) { s.Warn(hostile, "") },
		"Info":        func(s *Stepper) { s.Info(hostile, hostile) },
		"Hero":        func(s *Stepper) { s.Hero(hostile, hostile) },
		"Phase":       func(s *Stepper) { s.Phase(hostile) },
		"SubStart":    func(s *Stepper) { s.SubStart(hostile) },
		"SubDone":     func(s *Stepper) { s.SubDone(hostile, hostile) },
		"SubFail":     func(s *Stepper) { s.SubFail(hostile, errors.New(hostile)) },
		"SubFailHint": func(s *Stepper) { _ = s.SubFailHint(hostile, errors.New(hostile), hostile) },
		"TreeLine":    func(s *Stepper) { s.TreeLine(hostile, hostile) },
		"TreeSpin":    func(s *Stepper) { s.TreeSpin(hostile) },
		"Table": func(s *Stepper) {
			tb := NewTable("NAME", "STATUS")
			tb.w = s.w
			tb.AddRow(hostile, hostile)
			tb.Render()
		},
	}
	helpers := map[string]func(string) string{"Faint": Faint, "Strong": Strong, "Green": Green, "Yellow": Yellow, "Red": Red, "Cyan": Cyan, "faint": faint, "strong": strong}
	for name, helper := range helpers {
		writers[name] = func(s *Stepper) { _, _ = s.w.Write([]byte(helper(hostile))) }
	}
	writers["StrongIn"] = func(s *Stepper) { _, _ = s.w.Write([]byte(StrongIn(0, hostile))) }
	for name, write := range writers {
		var out bytes.Buffer
		s := &Stepper{w: &out}
		write(s)
		if rawProjectBytes(out.String()) {
			t.Errorf("%s wrote the project's bytes raw: %q", name, out.String())
		}
		if !strings.Contains(out.String(), esc("x1b")+"[2J") {
			t.Errorf("%s did not write the escaped text: %q", name, out.String())
		}
	}
}
