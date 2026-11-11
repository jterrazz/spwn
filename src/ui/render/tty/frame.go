package tty

import (
	"errors"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// invariant: a form that ends leaves on the terminal the frame the owner
// answered on, the one drawn before the key that ended it. huh draws
// nothing once a form ends, so what stayed was whichever frame the
// renderer flushed last: the field focused, or blurred by the key, by
// chance of the frame clock.
type answered struct {
	form  *huh.Form
	shown string
}

func (a *answered) Init() tea.Cmd { return a.form.Init() }

func (a *answered) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok && a.form.State == huh.StateNormal {
		a.shown = a.form.View()
	}
	_, cmd := a.form.Update(msg)
	return a, cmd
}

func (a *answered) View() tea.View {
	content := a.form.View()
	if a.form.State != huh.StateNormal {
		content = a.shown
	}
	v := tea.NewView(content)
	v.ReportFocus = true
	return v
}

func OnFrame(form *huh.Form, in io.Reader, out io.Writer) error {
	form.SubmitCmd, form.CancelCmd = tea.Quit, tea.Interrupt
	_, err := tea.NewProgram(&answered{form: form}, tea.WithInput(in), tea.WithOutput(out)).Run()
	switch {
	case form.State == huh.StateAborted || errors.Is(err, tea.ErrInterrupted):
		return huh.ErrUserAborted
	case err != nil:
		return fmt.Errorf("huh: %w", err)
	}
	return nil
}
