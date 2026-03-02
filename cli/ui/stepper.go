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

// Mode controls output behavior.
type Mode int

const (
	ModeNormal  Mode = iota
	ModeQuiet          // suppress all step output
	ModeVerbose        // disable spinners, show inline logs
	ModeJSON           // suppress human output entirely
)

// Stepper manages sequential step-by-step CLI output with spinners.
type Stepper struct {
	mode   Mode
	w      io.Writer
	isTTY  bool

	mu     sync.Mutex
	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a Stepper based on the CLI flags.
func New(quiet, verbose, jsonOut bool) *Stepper {
	mode := ModeNormal
	switch {
	case jsonOut:
		mode = ModeJSON
	case quiet:
		mode = ModeQuiet
	case verbose:
		mode = ModeVerbose
	}

	return &Stepper{
		mode:  mode,
		w:     os.Stderr,
		isTTY: term.IsTerminal(int(os.Stderr.Fd())),
	}
}

// Start begins a new step with a spinner animation.
func (s *Stepper) Start(msg string) {
	if s.mode == ModeQuiet || s.mode == ModeJSON {
		return
	}

	s.stopSpinner()

	if s.mode == ModeVerbose || !s.isTTY {
		fmt.Fprintf(s.w, "  → %s\n", msg)
		return
	}

	s.mu.Lock()
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.doneCh)
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			frame := spinnerFrames[i%len(spinnerFrames)]
			fmt.Fprintf(s.w, "\r  %s %s", green.Sprint(frame), msg)
			i++

			select {
			case <-s.stopCh:
				// Clear the spinner line
				clearLen := len(msg) + 10
				fmt.Fprintf(s.w, "\r%s\r", strings.Repeat(" ", clearLen))
				return
			case <-ticker.C:
			}
		}
	}()
}

// Done completes the current step with a checkmark.
func (s *Stepper) Done(label, detail string) {
	if s.mode == ModeQuiet || s.mode == ModeJSON {
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
	if s.mode == ModeQuiet || s.mode == ModeJSON {
		return
	}

	s.stopSpinner()

	if err != nil {
		fmt.Fprintf(s.w, "  %s %-*s %s\n", cross(), labelWidth, red.Sprint(label), faint(err.Error()))
	} else {
		fmt.Fprintf(s.w, "  %s %s\n", cross(), red.Sprint(label))
	}
}

// Log writes a line of output during an active step (verbose mode only).
func (s *Stepper) Log(format string, args ...any) {
	if s.mode != ModeVerbose {
		return
	}
	fmt.Fprintf(s.w, "    "+format+"\n", args...)
}

// Writer returns an io.Writer for piping output (e.g., Docker build logs).
// In verbose mode, returns a writer that indents each line.
// In all other modes, returns io.Discard.
func (s *Stepper) Writer() io.Writer {
	if s.mode == ModeVerbose {
		return &indentWriter{w: s.w, prefix: "    "}
	}
	return io.Discard
}

// Blank prints an empty line for spacing.
func (s *Stepper) Blank() {
	if s.mode == ModeQuiet || s.mode == ModeJSON {
		return
	}
	fmt.Fprintln(s.w)
}

// Success prints a final green success message.
func (s *Stepper) Success(msg string) {
	if s.mode == ModeQuiet || s.mode == ModeJSON {
		return
	}
	fmt.Fprintf(s.w, "  %s %s\n", check(), green.Sprint(msg))
}

// Info prints a label-value pair for summary output.
func (s *Stepper) Info(label, value string) {
	if s.mode == ModeQuiet || s.mode == ModeJSON {
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

// indentWriter prefixes each line with a string.
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
