package ui

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// newTestStepper returns a Stepper that writes to a buffer with spinners
// disabled (non-TTY).
func newTestStepper(mode stepperMode) (*Stepper, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	s := &Stepper{
		mode:  mode,
		w:     buf,
		isTTY: false,
	}
	return s, buf
}

// ── Human mode ──────────────────────────────────────────────────────────────

func TestStepper_HumanMode_Start(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Start("loading")
	if !strings.Contains(buf.String(), "loading") {
		t.Errorf("expected start message, got %q", buf.String())
	}
}

func TestStepper_HumanMode_Done(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Done("Loaded", "config")
	out := buf.String()
	if !strings.Contains(out, "Loaded") || !strings.Contains(out, "config") {
		t.Errorf("expected label and detail, got %q", out)
	}
}

func TestStepper_HumanMode_Fail(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Fail("Broken", errors.New("boom"))
	out := buf.String()
	if !strings.Contains(out, "Broken") || !strings.Contains(out, "boom") {
		t.Errorf("expected label and error, got %q", out)
	}
}

func TestStepper_HumanMode_Success(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Success("All good")
	if !strings.Contains(buf.String(), "All good") {
		t.Errorf("expected success message, got %q", buf.String())
	}
}

func TestStepper_HumanMode_Warn(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Warn("Careful", "detail")
	out := buf.String()
	if !strings.Contains(out, "Careful") || !strings.Contains(out, "detail") {
		t.Errorf("expected warn label and detail, got %q", out)
	}
}

func TestStepper_HumanMode_Info(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Info("World", "w-abc-123")
	if !strings.Contains(buf.String(), "w-abc-123") {
		t.Errorf("expected info value, got %q", buf.String())
	}
}

func TestStepper_HumanMode_Blank(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Blank()
	if buf.String() != "\n" {
		t.Errorf("expected single newline, got %q", buf.String())
	}
}

// ── JSON mode ────────────────────────────────────────────────────────────────
// In JSON mode the stepper should emit nothing — output goes to stdout
// as structured data by the caller.

func TestStepper_JSONMode_SuppressesAll(t *testing.T) {
	s, buf := newTestStepper(modeJSON)
	s.Start("loading")
	s.Done("Loaded", "config")
	s.Fail("Broken", errors.New("boom"))
	s.Success("All good")
	s.Warn("Careful", "detail")
	s.Info("World", "w-abc-123")
	s.Blank()
	if buf.String() != "" {
		t.Errorf("JSON mode should suppress all output, got %q", buf.String())
	}
}

// ── Constructor ──────────────────────────────────────────────────────────────

func TestNew_JSONTrue(t *testing.T) {
	s := New(true)
	if s.mode != modeJSON {
		t.Errorf("expected modeJSON, got %v", s.mode)
	}
}

func TestNew_JSONFalse(t *testing.T) {
	s := New(false)
	if s.mode != modeHuman {
		t.Errorf("expected modeHuman, got %v", s.mode)
	}
}

// ── Writer / Log ────────────────────────────────────────────────────────────

func TestStepper_Writer_AlwaysDiscards(t *testing.T) {
	s, _ := newTestStepper(modeHuman)
	w := s.Writer()
	if w != io.Discard {
		t.Errorf("expected io.Discard, got %T", w)
	}
}

func TestStepper_Log_NoOp(t *testing.T) {
	s, buf := newTestStepper(modeHuman)
	s.Log("this should not appear")
	if buf.String() != "" {
		t.Errorf("Log is a no-op, expected empty output, got %q", buf.String())
	}
}

// ── FailHint ────────────────────────────────────────────────────────────────

func TestStepper_FailHint_ReturnsDisplayedError(t *testing.T) {
	s, _ := newTestStepper(modeHuman)
	err := s.FailHint("Broken", errors.New("boom"), "try X")
	var de *DisplayedError
	if !errors.As(err, &de) {
		t.Errorf("expected *DisplayedError, got %T", err)
	}
}
