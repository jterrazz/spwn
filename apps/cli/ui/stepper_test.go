package ui

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// newTestStepper creates a Stepper that writes to a buffer instead of stderr.
func newTestStepper(mode Mode) (*Stepper, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	s := &Stepper{
		mode:  mode,
		w:     buf,
		isTTY: false, // non-TTY so spinners don't run
	}
	return s, buf
}

func TestStepper_Done_WithDetail(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Done("Built image", "sha256:abc")

	out := buf.String()
	if !strings.Contains(out, "Built image") {
		t.Errorf("Done() output missing label, got %q", out)
	}
	if !strings.Contains(out, "sha256:abc") {
		t.Errorf("Done() output missing detail, got %q", out)
	}
}

func TestStepper_Done_WithoutDetail(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Done("Complete", "")

	out := buf.String()
	if !strings.Contains(out, "Complete") {
		t.Errorf("Done() output missing label, got %q", out)
	}
}

func TestStepper_Fail_WithError(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Fail("Spawn failed", errors.New("container timeout"))

	out := buf.String()
	if !strings.Contains(out, "Spawn failed") {
		t.Errorf("Fail() output missing label, got %q", out)
	}
	if !strings.Contains(out, "container timeout") {
		t.Errorf("Fail() output missing error message, got %q", out)
	}
}

func TestStepper_Fail_WithoutError(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Fail("Something broke", nil)

	out := buf.String()
	if !strings.Contains(out, "Something broke") {
		t.Errorf("Fail() output missing label, got %q", out)
	}
}

func TestStepper_Log_VerboseMode(t *testing.T) {
	s, buf := newTestStepper(ModeVerbose)
	s.Log("pulling layer %d", 3)

	out := buf.String()
	if !strings.Contains(out, "pulling layer 3") {
		t.Errorf("Log() in verbose mode should output message, got %q", out)
	}
}

func TestStepper_Log_NormalMode_Suppressed(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Log("should not appear")

	if buf.Len() != 0 {
		t.Errorf("Log() in normal mode should be suppressed, got %q", buf.String())
	}
}

func TestStepper_QuietMode_SuppressesAll(t *testing.T) {
	s, buf := newTestStepper(ModeQuiet)

	s.Done("label", "detail")
	s.Fail("err", errors.New("oops"))
	s.Blank()
	s.Success("yay")
	s.Warn("caution", "something")
	s.Info("key", "val")

	if buf.Len() != 0 {
		t.Errorf("quiet mode should suppress all output, got %q", buf.String())
	}
}

func TestStepper_JSONMode_SuppressesAll(t *testing.T) {
	s, buf := newTestStepper(ModeJSON)

	s.Done("label", "detail")
	s.Fail("err", errors.New("oops"))
	s.Blank()
	s.Success("yay")
	s.Warn("caution", "something")
	s.Info("key", "val")

	if buf.Len() != 0 {
		t.Errorf("JSON mode should suppress all output, got %q", buf.String())
	}
}

func TestStepper_Success(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Success("World spawned.")

	out := buf.String()
	if !strings.Contains(out, "World spawned.") {
		t.Errorf("Success() output missing message, got %q", out)
	}
}

func TestStepper_Warn_WithDetail(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Warn("Deprecated", "use --new-flag")

	out := buf.String()
	if !strings.Contains(out, "Deprecated") {
		t.Errorf("Warn() output missing label, got %q", out)
	}
	if !strings.Contains(out, "use --new-flag") {
		t.Errorf("Warn() output missing detail, got %q", out)
	}
}

func TestStepper_Info(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Info("World:", "w-nebula-12345")

	out := buf.String()
	if !strings.Contains(out, "World:") {
		t.Errorf("Info() output missing label, got %q", out)
	}
	if !strings.Contains(out, "w-nebula-12345") {
		t.Errorf("Info() output missing value, got %q", out)
	}
}

func TestStepper_Blank(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.Blank()

	if buf.String() != "\n" {
		t.Errorf("Blank() should output a single newline, got %q", buf.String())
	}
}

func TestStepper_Writer_VerboseMode(t *testing.T) {
	s, _ := newTestStepper(ModeVerbose)
	w := s.Writer()
	if w == io.Discard {
		t.Error("Writer() in verbose mode should not return io.Discard")
	}
}

func TestStepper_Writer_NormalMode(t *testing.T) {
	s, _ := newTestStepper(ModeNormal)
	w := s.Writer()
	if w != io.Discard {
		t.Error("Writer() in normal mode should return io.Discard")
	}
}

func TestStepper_Start_VerboseMode(t *testing.T) {
	s, buf := newTestStepper(ModeVerbose)
	s.Start("Building image...")

	out := buf.String()
	if !strings.Contains(out, "Building image...") {
		t.Errorf("Start() in verbose mode should print message, got %q", out)
	}
}

func TestStepper_Start_QuietMode_Suppressed(t *testing.T) {
	s, buf := newTestStepper(ModeQuiet)
	s.Start("Building image...")

	if buf.Len() != 0 {
		t.Errorf("Start() in quiet mode should be suppressed, got %q", buf.String())
	}
}

func TestStepper_FailHint_ShowsHint(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	err := s.FailHint("Agent failed", errors.New("not found"), `Run "spwn agent new neo"`)

	out := buf.String()
	if !strings.Contains(out, "Agent failed") {
		t.Errorf("FailHint() missing label, got %q", out)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("FailHint() missing error, got %q", out)
	}
	if !strings.Contains(out, "spwn agent new neo") {
		t.Errorf("FailHint() missing hint, got %q", out)
	}

	// Should return a DisplayedError
	if _, ok := err.(*DisplayedError); !ok {
		t.Errorf("FailHint() should return *DisplayedError, got %T", err)
	}
}

func TestStepper_FailHint_QuietMode_Suppressed(t *testing.T) {
	s, buf := newTestStepper(ModeQuiet)
	err := s.FailHint("Agent failed", errors.New("not found"), "some hint")

	if buf.Len() != 0 {
		t.Errorf("FailHint() in quiet mode should suppress output, got %q", buf.String())
	}
	if _, ok := err.(*DisplayedError); !ok {
		t.Error("FailHint() should still return DisplayedError in quiet mode")
	}
}

func TestStepper_FailHint_EmptyHint(t *testing.T) {
	s, buf := newTestStepper(ModeNormal)
	s.FailHint("Failed", errors.New("oops"), "")

	out := buf.String()
	if !strings.Contains(out, "Failed") {
		t.Error("FailHint with empty hint should still show label")
	}
}

func TestDisplayedError(t *testing.T) {
	inner := errors.New("inner error")
	de := &DisplayedError{Err: inner}

	if de.Error() != "inner error" {
		t.Errorf("expected 'inner error', got %q", de.Error())
	}
	if de.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestIndentWriter(t *testing.T) {
	var buf bytes.Buffer
	iw := &indentWriter{w: &buf, prefix: "    "}

	_, err := iw.Write([]byte("line1\nline2\nline3\n"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "    ") {
			t.Errorf("line %d missing indent prefix: %q", i, line)
		}
	}
}
