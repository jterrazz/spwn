package output

import (
	"io"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagejson "github.com/jterrazz/studio-station/apps/engine/ui/render/json"
	"github.com/jterrazz/studio-station/apps/engine/ui/render/pipe"
	"github.com/jterrazz/studio-station/apps/engine/ui/render/tty"
)

type Want struct {
	JSON  bool
	Page  bool
	Piped []byte
	// value: the page the pipe prints instead, when the eye's nesting is not
	// the records' order (Accounts: one record per account, machine first).
	Records *page.Page
	// value: what every document opens with, the machine and the time it
	// was read; nil for a document printed as its call gave it.
	Envelope *Envelope
}

func Terminal(in io.Reader, out io.Writer, dumb bool) *tty.Terminal {
	// invariant: a dumb terminal answers a form line by line, as huh asks
	// it there; any other runs it as the frame the owner answers on.
	if dumb {
		return tty.NewTerminal(in, out, nil)
	}
	return tty.NewTerminal(in, out, tty.OnFrame)
}

func (w *Writer) Page(p page.Page, contract any, want Want) error {
	w.settle()
	c := Detect(w.stdout, w.env, want.JSON || want.Page)
	out := w.stdout.W
	switch {
	case want.Page:
		body := pagejson.Render(p)
		secret.Tripwire(body)
		_, err := out.Write(body)
		return err
	case c.Mode == JSON:
		body, err := Document(contract, want.Envelope)
		if err != nil {
			return err
		}
		secret.Tripwire(body)
		_, err = out.Write(body)
		return err
	case c.Mode == Pipe && want.Piped != nil:
		secret.Tripwire(want.Piped)
		_, err := out.Write(want.Piped)
		return err
	case c.Mode == Pipe:
		if want.Records != nil {
			p = *want.Records
		}
		body, err := pipe.Render(p)
		if err != nil {
			return err
		}
		secret.Tripwire(body)
		_, err = out.Write(body)
		return err
	case c.Mode == TTY:
	}
	body := tty.Render(p, tty.Options{Width: c.Width, Dark: c.Dark})
	secret.Tripwire(body)
	_, err := w.colored(w.stdout, c).Write(body)
	return err
}

func (w *Writer) LivePage(pages <-chan page.Page) error {
	// invariant: the live region is redrawn in place on every page and erased
	// when the reading ends; the final page is then printed once, normally, so
	// the scrollback holds exactly what a non-live read prints.
	w.settle()
	c := Detect(w.stdout, w.env, false)
	o := tty.Options{Width: c.Width, Dark: c.Dark}
	out := w.colored(w.stdout, c)
	var (
		last  page.Page
		got   bool
		drawn int
	)
	for p := range pages {
		secret.TripwireJSON(p)
		last, got = p, true
		lines := tty.Frame(p, o, w.stdout.Height)
		if _, err := io.WriteString(out, tty.Draw(lines, drawn)); err != nil {
			return err
		}
		drawn = len(lines)
	}
	if _, err := io.WriteString(out, tty.Erase(drawn)); err != nil {
		return err
	}
	if !got {
		return nil
	}
	_, err := out.Write(tty.Render(last, o))
	return err
}
