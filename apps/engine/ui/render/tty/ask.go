package tty

import (
	"context"
	"errors"
	"io"

	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"

	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ui/organism"
)

type Terminal struct {
	in  io.Reader
	out io.Writer
	// value: how a form runs on this terminal; nil is huh's own run, the
	// line-by-line one a dumb terminal takes.
	answer Answer
}

type Answer func(form *huh.Form, in io.Reader, out io.Writer) error

func NewTerminal(in io.Reader, out io.Writer, answer Answer) *Terminal {
	return &Terminal{in: in, out: out, answer: answer}
}

func (t *Terminal) Confirm(c page.Confirm) (bool, error) {
	// invariant: a Danger question defaults to No and Esc declines it, so a
	// destructive step only ever runs on a deliberate yes; any other question
	// treats Esc and ctrl-c as an interrupt.
	answer := false
	field := organism.ConfirmField(c).Value(&answer)
	form := huh.NewForm(huh.NewGroup(field)).
		WithTheme(huh.ThemeFunc(organism.FormStyles)).
		WithInput(t.in).
		WithOutput(t.out).
		WithShowHelp(false)
	if c.Danger {
		keys := huh.NewDefaultKeyMap()
		keys.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
		form = form.WithKeyMap(keys)
	}
	err := t.run(form)
	switch {
	case errors.Is(err, huh.ErrUserAborted) && c.Danger:
		return false, nil
	case errors.Is(err, huh.ErrUserAborted):
		return false, context.Canceled
	}
	return answer, err
}

func (t *Terminal) Form(f page.Form) (map[string]string, error) {
	// invariant: a secret field is masked as it is typed and its value is returned
	// to the caller alone: never echoed, logged or kept by the form.
	fields, read := organism.Fields(f)
	group := huh.NewGroup(fields...)
	if f.Title != "" {
		group = group.Title(f.Title)
	}
	err := t.run(huh.NewForm(group).
		WithTheme(huh.ThemeFunc(organism.FormStyles)).
		WithInput(t.in).
		WithOutput(t.out).
		WithShowHelp(false))
	if errors.Is(err, huh.ErrUserAborted) {
		return nil, context.Canceled
	}
	if err != nil {
		return nil, err
	}
	return read(), nil
}

func (t *Terminal) run(form *huh.Form) error {
	if t.answer == nil {
		return form.Run()
	}
	return t.answer(form, t.in, t.out)
}
