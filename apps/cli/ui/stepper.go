package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const labelWidth = 24

// Stepper manages sequential step-by-step CLI output with spinners.
//
// Output modes are minimal: either human output (stderr spinners + rows)
// or JSON mode, which suppresses human output entirely so callers can
// print clean structured output to stdout.
type Stepper struct {
	mode  stepperMode
	w     io.Writer
	isTTY bool

	mu     sync.Mutex
	stopCh chan struct{}
	doneCh chan struct{}
}

type stepperMode int

const (
	modeHuman stepperMode = iota
	modeJSON
)

// New creates a Stepper. Pass jsonOut=true to suppress human output so
// the caller can write structured output cleanly to stdout.
func New(jsonOut bool) *Stepper {
	mode := modeHuman
	if jsonOut {
		mode = modeJSON
	}
	return &Stepper{
		mode:  mode,
		w:     os.Stderr,
		isTTY: term.IsTerminal(int(os.Stderr.Fd())),
	}
}

// Start begins a new step with a spinner animation.
func (s *Stepper) Start(msg string) {
	if s.mode == modeJSON {
		return
	}

	s.stopSpinner()

	if !s.isTTY {
		fmt.Fprintf(s.w, "  → %s\n", msg)
		return
	}

	s.mu.Lock()
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
			frame := spinnerFrames[i%len(spinnerFrames)]
			fmt.Fprintf(s.w, "\r  %s %s", green.Sprint(frame), msg)
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

// Done completes the current step with a checkmark.
func (s *Stepper) Done(label, detail string) {
	if s.mode == modeJSON {
		return
	}

	s.stopSpinner()

	if detail != "" {
		fmt.Fprintf(s.w, "  %s %-*s %s\n", check(), labelWidth, strong(label), faint(detail))
	} else {
		fmt.Fprintf(s.w, "  %s %s\n", check(), strong(label))
	}
}

// Fail completes the current step with a cross mark.
func (s *Stepper) Fail(label string, err error) {
	if s.mode == modeJSON {
		return
	}

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
	if hint != "" && s.mode != modeJSON {
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

// Writer returns an io.Writer for piping output (e.g. Docker build logs).
// In the current design we discard piped output — builds run silently
// through the Stepper. If we reintroduce verbose logging this is where
// it would flow.
func (s *Stepper) Writer() io.Writer {
	return io.Discard
}

// Blank prints an empty line for spacing.
func (s *Stepper) Blank() {
	if s.mode == modeJSON {
		return
	}
	fmt.Fprintln(s.w)
}

// Success prints a final green success message.
func (s *Stepper) Success(msg string) {
	if s.mode == modeJSON {
		return
	}
	fmt.Fprintf(s.w, "  %s %s\n", check(), green.Sprint(msg))
}

// Warn prints a warning line with a yellow "!" prefix.
func (s *Stepper) Warn(label, detail string) {
	if s.mode == modeJSON {
		return
	}

	s.stopSpinner()

	if detail != "" {
		fmt.Fprintf(s.w, "  %s %-*s %s\n", warn(), labelWidth, yellow.Sprint(label), faint(detail))
	} else {
		fmt.Fprintf(s.w, "  %s %s\n", warn(), yellow.Sprint(label))
	}
}

// Info prints a label-value pair for summary output.
func (s *Stepper) Info(label, value string) {
	if s.mode == modeJSON {
		return
	}
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

// indentWriter prefixes each line with a string. Kept for potential
// reuse; currently unreferenced.
type indentWriter struct {
	w      io.Writer
	prefix string
	atBOL  bool
}

func (iw *indentWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		if iw.atBOL || written == 0 {
			if _, err := fmt.Fprint(iw.w, iw.prefix); err != nil {
				return written, err
			}
			iw.atBOL = false
		}

		idx := strings.IndexByte(string(p), '\n')
		if idx < 0 {
			n, err := iw.w.Write(p)
			written += n
			return written, err
		}

		n, err := iw.w.Write(p[:idx+1])
		written += n
		if err != nil {
			return written, err
		}
		p = p[idx+1:]
		iw.atBOL = true
	}
	return written, nil
}
