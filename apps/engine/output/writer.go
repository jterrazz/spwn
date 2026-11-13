package output

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/render/tty"
)

type Writer struct {
	stdout  Stream
	stderr  Stream
	env     map[string]string
	environ []string
	waiting func()
}

func NewWriter(stdout, stderr Stream, env map[string]string) *Writer {
	environ := make([]string, 0, len(env))
	for k, v := range env {
		environ = append(environ, k+"="+v)
	}
	sort.Strings(environ)
	return &Writer{stdout: stdout, stderr: stderr, env: env, environ: environ}
}

func (w *Writer) Raw(p []byte) error {
	secret.Tripwire(p)
	w.settle()
	_, err := w.stdout.W.Write(p)
	return err
}

func (w *Writer) CanLive() bool { return Detect(w.stdout, w.env, false).Mode == TTY }

func (w *Writer) Failure(err error) {
	if err == nil {
		return
	}
	var e *fail.Error
	if !errors.As(err, &e) {
		e = &fail.Error{Message: err.Error(), Exit: fail.ExitFailure}
	}
	if e.Message == "" {
		return
	}
	w.settle()
	c := Detect(w.stderr, w.env, false)
	block := tty.ErrorBlock(page.Error{Message: screened(e.Message), Hint: screened(e.Hint)}, c.Mode == TTY, c.Dark)
	secret.Tripwire([]byte(block))
	_, _ = io.WriteString(w.colored(w.stderr, c), block)
}

func (w *Writer) Finding(m page.Mark, parts ...string) error {
	line := tty.FindingLine(m, parts...)
	secret.Tripwire([]byte(line))
	w.settle()
	_, err := io.WriteString(w.stdout.W, line)
	return err
}

func (w *Writer) Note(parts ...string) {
	line := tty.NoteLine(parts...)
	secret.Tripwire([]byte(line))
	w.settle()
	_, _ = io.WriteString(w.stderr.W, line)
}

func (w *Writer) Now(command string) {
	// value: the one muted line a hidden alias leaves on stderr, naming the
	// command it ran (SPEC §4.3).
	w.settle()
	c := Detect(w.stderr, w.env, false)
	command = screened(command)
	line := tty.NowLine(command, c.Mode == TTY, c.Dark)
	secret.Tripwire([]byte(line))
	_, _ = io.WriteString(w.colored(w.stderr, c), line)
}

func (w *Writer) Menu(render func(io.Writer)) {
	w.settle()
	var menu bytes.Buffer
	render(&menu)
	secret.Tripwire(menu.Bytes())
	_, _ = w.stderr.W.Write(menu.Bytes())
}

func (w *Writer) Wait(label string) func() {
	c := Detect(w.stderr, w.env, false)
	if c.Mode != TTY {
		return func() {}
	}
	secret.Tripwire([]byte(label))
	w.settle()
	stop := tty.Wait(w.colored(w.stderr, c), label, c.Dark)
	w.waiting = stop
	return func() {
		stop()
		w.waiting = nil
	}
}

func Said(parts ...string) string {
	// value: words said as one line, each part apart by the separator.
	return strings.TrimSuffix(tty.NoteLine(parts...), "\n")
}

func (w *Writer) Ahead(parts ...string) {
	// value: a step said once before a program asks the owner on the
	// terminal (`▸ Installing homebrew · … · asks your password once`), so
	// its prompt has a line of its own; a pipe hears nothing of it.
	// invariant: the line is for whoever sits at the terminal, so NO_COLOR
	// keeps it, its glyph and words without color.
	if !w.stderr.IsTerminal {
		return
	}
	c := Detect(w.stderr, w.env, false)
	line := tty.AheadLine(tty.NoteLine(parts...), c.Mode == TTY, c.Dark)
	secret.Tripwire([]byte(line))
	w.settle()
	_, _ = io.WriteString(w.colored(w.stderr, c), line)
}

func (w *Writer) settle() {
	// invariant: a spinner is erased before anything else is written, so a
	// slow read never leaves its frame glued to the page that follows it.
	if w.waiting != nil {
		w.waiting()
		w.waiting = nil
	}
}

func (w *Writer) colored(s Stream, c Context) io.Writer {
	if c.Mode != TTY {
		return s.W
	}
	return tty.Colored(s.W, w.environ)
}
