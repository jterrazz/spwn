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
func newTestStepper() (*Stepper, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	s := &Stepper{
		w:     buf,
		isTTY: false,
	}
	return s, buf
}

func TestStepper_Start(t *testing.T) {
	s, buf := newTestStepper()
	s.Start("loading")
	if !strings.Contains(buf.String(), "loading") {
		t.Errorf("expected start message, got %q", buf.String())
	}
}

func TestStepper_Done(t *testing.T) {
	s, buf := newTestStepper()
	s.Done("Loaded", "config")
	out := buf.String()
	if !strings.Contains(out, "Loaded") || !strings.Contains(out, "config") {
		t.Errorf("expected label and detail, got %q", out)
	}
}

func TestStepper_Fail(t *testing.T) {
	s, buf := newTestStepper()
	s.Fail("Broken", errors.New("boom"))
	out := buf.String()
	if !strings.Contains(out, "Broken") || !strings.Contains(out, "boom") {
		t.Errorf("expected label and error, got %q", out)
	}
}

func TestStepper_Success(t *testing.T) {
	s, buf := newTestStepper()
	s.Success("All good")
	if !strings.Contains(buf.String(), "All good") {
		t.Errorf("expected success message, got %q", buf.String())
	}
}

func TestStepper_Warn(t *testing.T) {
	s, buf := newTestStepper()
	s.Warn("Careful", "detail")
	out := buf.String()
	if !strings.Contains(out, "Careful") || !strings.Contains(out, "detail") {
		t.Errorf("expected warn label and detail, got %q", out)
	}
}

func TestStepper_Info(t *testing.T) {
	s, buf := newTestStepper()
	s.Info("World", "w-abc-123")
	if !strings.Contains(buf.String(), "w-abc-123") {
		t.Errorf("expected info value, got %q", buf.String())
	}
}

func TestStepper_Blank(t *testing.T) {
	s, buf := newTestStepper()
	s.Blank()
	if buf.String() != "\n" {
		t.Errorf("expected single newline, got %q", buf.String())
	}
}

func TestStepper_Writer_AlwaysDiscards(t *testing.T) {
	s, _ := newTestStepper()
	w := s.Writer()
	if w != io.Discard {
		t.Errorf("expected io.Discard, got %T", w)
	}
}

func TestStepper_Log_NoOp(t *testing.T) {
	s, buf := newTestStepper()
	s.Log("this should not appear")
	if buf.String() != "" {
		t.Errorf("Log is a no-op, expected empty output, got %q", buf.String())
	}
}

func TestStepper_FailHint_ReturnsDisplayedError(t *testing.T) {
	s, _ := newTestStepper()
	err := s.FailHint("Broken", errors.New("boom"), "try X")
	var de *DisplayedError
	if !errors.As(err, &de) {
		t.Errorf("expected *DisplayedError, got %T", err)
	}
}
