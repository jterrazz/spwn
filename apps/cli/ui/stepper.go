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

// BuildProgressWriter returns an io.Writer that parses Docker
// build output and updates the active spinner label in place with
// "<base> [N/M] <action>" as each step completes. Pass this as
// the LogWriter for a `spwn up` that's about to build an image
// and the long silent gap turns into live progress. Works only
// on TTY writers; on non-TTY the writer silently discards input
// (the stepper's one-shot prints cover non-interactive mode).
func (s *Stepper) BuildProgressWriter(base string) io.Writer {
	if !s.isTTY {
		return io.Discard
	}
	return &buildProgressWriter{stepper: s, base: base}
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

