package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const labelWidth = 24

// Stepper manages sequential step-by-step CLI output with spinners.
type Stepper struct {
	w     io.Writer
	isTTY bool

	mu     sync.Mutex
	label  string // current spinner label; live-updatable via UpdateLabel
	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a Stepper that writes to stderr.
func New() *Stepper {
	return &Stepper{
		w:     os.Stderr,
		isTTY: term.IsTerminal(int(os.Stderr.Fd())),
	}
}

// Start begins a new step with a spinner animation.
func (s *Stepper) Start(msg string) {
	s.stopSpinner()

	if !s.isTTY {
		fmt.Fprintf(s.w, "  → %s\n", msg)
		return
	}

	s.mu.Lock()
	s.label = msg
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.mu.Unlock()

	go func() {
		defer close(doneCh)
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			s.mu.Lock()
			label := s.label
			s.mu.Unlock()

			frame := spinnerFrames[i%len(spinnerFrames)]
			// Clear the entire line before redraw so a shorter
			// label doesn't leave trailing characters from the
			// previous (longer) render.
			fmt.Fprintf(s.w, "\r\033[2K  %s %s", green.Sprint(frame), label)
			i++

			select {
			case <-stopCh:
				// Clear the spinner line using ANSI escape (handles color codes)
				fmt.Fprintf(s.w, "\r\033[2K")
				return
			case <-ticker.C:
			}
		}
	}()
}

// UpdateLabel replaces the spinner label in place. Safe to call
// from any goroutine (the spinner re-reads the label each frame).
// No-op on non-TTY output where labels are one-shot prints.
func (s *Stepper) UpdateLabel(msg string) {
	if !s.isTTY {
		return
	}
	s.mu.Lock()
	s.label = msg
	s.mu.Unlock()
}

// Done completes the current step with a checkmark.
func (s *Stepper) Done(label, detail string) {
	s.stopSpinner()

	if detail != "" {
		fmt.Fprintf(s.w, "  %s %-*s %s\n", check(), labelWidth, strong(label), faint(detail))
	} else {
		fmt.Fprintf(s.w, "  %s %s\n", check(), strong(label))
	}
}

// Fail completes the current step with a cross mark.
func (s *Stepper) Fail(label string, err error) {
	s.stopSpinner()

	if err != nil {
		fmt.Fprintf(s.w, "  %s %-*s %s\n", cross(), labelWidth, red.Sprint(label), faint(err.Error()))
	} else {
		fmt.Fprintf(s.w, "  %s %s\n", cross(), red.Sprint(label))
	}
}

// FailHint displays an error with an actionable hint and returns a
// displayedError so Execute() won't re-print it.
func (s *Stepper) FailHint(label string, err error, hint string) error {
	s.Fail(label, err)
	if hint != "" {
		fmt.Fprintf(s.w, "  %s %s\n", " ", faint(hint))
	}
	s.Blank()
	return &DisplayedError{Err: err}
}

// DisplayedError wraps an error that was already shown to the user.
type DisplayedError struct{ Err error }

func (e *DisplayedError) Error() string { return e.Err.Error() }
func (e *DisplayedError) Unwrap() error { return e.Err }

// Log is a no-op in the current design. Kept for callers that used to
// emit verbose-mode logs; if we reintroduce a real debug logger later,
// this is the hook. Today it does nothing.
func (s *Stepper) Log(format string, args ...any) {}

// Writer returns a discarding writer for callers that don't need
// build progress - the spinner is the whole UX.
func (s *Stepper) Writer() io.Writer {
	return io.Discard
}

// BuildProgress is the interface spawnRunE uses to stream docker
// Build output into the Stepper's tree view and to flush the live
// In-progress step when the build finishes. Writing to it consumes
// `Step N/M : …` lines; calling CompleteCurrent freezes any live
// Spinner as a permanent tree line so the summary row that follows
// Sits cleanly below.
type BuildProgress interface {
	io.Writer
	CompleteCurrent()
}

// discardBuildProgress is a BuildProgress that eats everything. Used
// On non-TTY output where there is no spinner to drive.
type discardBuildProgress struct{}

func (discardBuildProgress) Write(p []byte) (int, error) { return len(p), nil }
func (discardBuildProgress) CompleteCurrent()            {}

// BuildProgressWriter returns a BuildProgress that streams docker
// Build output into the Stepper's tree view — one frozen ├ line per
// `Step N/M` with a live spinner on the currently-running step.
// Non-TTY writers get a discard sink (the phase-summary lines cover
// Non-interactive mode).
func (s *Stepper) BuildProgressWriter() BuildProgress {
	if !s.isTTY {
		return discardBuildProgress{}
	}
	return &buildProgressWriter{stepper: s}
}

// Blank prints an empty line for spacing.
func (s *Stepper) Blank() {
	fmt.Fprintln(s.w)
}

// Success prints a final green success message.
func (s *Stepper) Success(msg string) {
	fmt.Fprintf(s.w, "  %s %s\n", check(), green.Sprint(msg))
}

// Warn prints a warning line with a yellow "!" prefix.
func (s *Stepper) Warn(label, detail string) {
	s.stopSpinner()

	if detail != "" {
		fmt.Fprintf(s.w, "  %s %-*s %s\n", warn(), labelWidth, yellow.Sprint(label), faint(detail))
	} else {
		fmt.Fprintf(s.w, "  %s %s\n", warn(), yellow.Sprint(label))
	}
}

// Info prints a label-value pair for summary output.
func (s *Stepper) Info(label, value string) {
	fmt.Fprintf(s.w, "  %-12s %s\n", strong(label), value)
}

// ── Phase / sub-step / tree primitives ────────────────────────────
//
// These shape the "tight tree" UI used by `spwn up` / `spwn agent
// <name>`: a small Hero header, a phase label (col 2), sub-steps
// (col 4), and tree lines (col 4, prefixed with ├). The existing
// Start/Done/Info primitives are unchanged — callers that don't want
// The phased shape keep working.

const (
	subIndent    = "    " // col 4 — sub-step body
	subLabelCols = 22     // width of the bold label column after the ✓
)

// Hero prints the top-of-command banner like
//
//	⬡ Waking neo
//
// Glyph is cyan-bold; text is bold. Adds a trailing blank line so
// The first phase label breathes.
func (s *Stepper) Hero(glyph, text string) {
	s.stopSpinner()
	fmt.Fprintf(s.w, "  %s %s\n", cyan.Sprint(glyph), strong(text))
	s.Blank()
}

// Phase prints a section heading at col 2 in bold, with a trailing
// Newline so the first child sub-step indents under it. Use before
// The first SubStart/SubDone of a phase. Also closes any live
// Spinner from the previous phase.
func (s *Stepper) Phase(name string) {
	s.stopSpinner()
	fmt.Fprintf(s.w, "  %s\n", strong(name))
}

// PhaseBreak prints a blank line between phases. Callers usually
// Reach for this between Phase()s so the eye can rest.
func (s *Stepper) PhaseBreak() {
	fmt.Fprintln(s.w)
}

// SubStart begins a sub-step inside a phase — the indented cousin
// Of Start(). Works on TTY and non-TTY: the non-TTY fallback emits a
// One-shot "    → <msg>" line since there is no in-place animation.
func (s *Stepper) SubStart(msg string) {
	s.stopSpinner()

	if !s.isTTY {
		fmt.Fprintf(s.w, "%s→ %s\n", subIndent, msg)
		return
	}

	s.mu.Lock()
	s.label = msg
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.mu.Unlock()

	go func() {
		defer close(doneCh)
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			s.mu.Lock()
			label := s.label
			s.mu.Unlock()

			frame := spinnerFrames[i%len(spinnerFrames)]
			fmt.Fprintf(s.w, "\r\033[2K%s%s %s", subIndent, green.Sprint(frame), label)
			i++

			select {
			case <-stopCh:
				fmt.Fprintf(s.w, "\r\033[2K")
				return
			case <-ticker.C:
			}
		}
	}()
}

// SubDone completes a sub-step with a green ✓, sitting at col 4.
// Mirrors Done() but indented to live under a Phase.
func (s *Stepper) SubDone(label, detail string) {
	s.stopSpinner()

	if detail != "" {
		fmt.Fprintf(s.w, "%s%s %-*s %s\n", subIndent, check(), subLabelCols, strong(label), faint(detail))
	} else {
		fmt.Fprintf(s.w, "%s%s %s\n", subIndent, check(), strong(label))
	}
}

// SubFail completes a sub-step with a red ✗, sitting at col 4.
func (s *Stepper) SubFail(label string, err error) {
	s.stopSpinner()

	if err != nil {
		fmt.Fprintf(s.w, "%s%s %-*s %s\n", subIndent, cross(), subLabelCols, red.Sprint(label), faint(err.Error()))
	} else {
		fmt.Fprintf(s.w, "%s%s %s\n", subIndent, cross(), red.Sprint(label))
	}
}

// SubFailHint fails a sub-step AND prints an actionable hint under
// It. Returns a DisplayedError so Execute() suppresses its own
// Error banner (the user has already seen the specific step that
// Broke and what to try next).
func (s *Stepper) SubFailHint(label string, err error, hint string) error {
	s.SubFail(label, err)
	if hint != "" {
		fmt.Fprintf(s.w, "%s  %s\n", subIndent, faint(hint))
	}
	s.Blank()
	return &DisplayedError{Err: err}
}

// TreeLine prints a single tree child at col 4, prefixed with a
// Faint ├. Text is printed as given (callers colour the pieces they
// Care about); a trailing faint detail can follow the main text.
// Used for the build sub-event stream — one line per docker Step
// N/M — so scrollback preserves the whole install history.
func (s *Stepper) TreeLine(text, detail string) {
	s.stopSpinner()
	if detail != "" {
		fmt.Fprintf(s.w, "%s%s %s %s\n", subIndent, faint("├"), text, faint(detail))
	} else {
		fmt.Fprintf(s.w, "%s%s %s\n", subIndent, faint("├"), text)
	}
}

// TreeSpin prints a tree child currently in progress — spinner in
// Place of the ├ char. When the next TreeLine / TreeSpin / SubDone
// Lands, the spinner line is overwritten in place. Non-TTY falls
// Back to a TreeLine with a faint "…" detail so scrollback still
// Shows the attempted step.
func (s *Stepper) TreeSpin(text string) {
	s.stopSpinner()
	if !s.isTTY {
		s.TreeLine(text, "…")
		return
	}
	s.mu.Lock()
	s.label = text
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.mu.Unlock()

	go func() {
		defer close(doneCh)
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			s.mu.Lock()
			label := s.label
			s.mu.Unlock()

			frame := spinnerFrames[i%len(spinnerFrames)]
			fmt.Fprintf(s.w, "\r\033[2K%s%s %s", subIndent, green.Sprint(frame), label)
			i++

			select {
			case <-stopCh:
				fmt.Fprintf(s.w, "\r\033[2K")
				return
			case <-ticker.C:
			}
		}
	}()
}

// stopSpinner stops any running spinner goroutine.
func (s *Stepper) stopSpinner() {
	s.mu.Lock()
	ch := s.stopCh
	done := s.doneCh
	s.stopCh = nil
	s.doneCh = nil
	s.mu.Unlock()

	if ch != nil {
		close(ch)
		<-done
	}
}

