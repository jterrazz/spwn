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
	// invariant: the first frame is drawn once the terminal's size is
	// applied, and the size is applied once. Bubble Tea erases the screen
	// each time it applies a size, the same one included, so a frame
	// flushed before it is drawn again, and a terminal that keeps a screen
	// cleared from its first cell keeps the first copy in its history.
	sized bool
}

// value: huh asks the program for the size it already sends at its start;
// the answer, a second size, is dropped with the question.
func unasked(_ tea.Model, msg tea.Msg) tea.Msg {
	if msg == tea.RequestWindowSize() {
		return nil
	}
	return msg
}

func (a *answered) Init() tea.Cmd { return a.form.Init() }

func (a *answered) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		a.sized = true
	case tea.KeyPressMsg:
		if a.form.State == huh.StateNormal {
			a.shown = a.form.View()
		}
	}
	_, cmd := a.form.Update(msg)
	return a, cmd
}

func (a *answered) View() tea.View {
	content := a.form.View()
	switch {
	case a.form.State != huh.StateNormal:
		content = a.shown
	case !a.sized:
		content = ""
	}
	v := tea.NewView(content)
	v.ReportFocus = true
	return v
}

func OnFrame(form *huh.Form, in io.Reader, out io.Writer) error {
	form.SubmitCmd, form.CancelCmd = tea.Quit, tea.Interrupt
	_, err := tea.NewProgram(&answered{form: form}, tea.WithInput(in), tea.WithOutput(out), tea.WithFilter(unasked)).Run()
	switch {
	case form.State == huh.StateAborted || errors.Is(err, tea.ErrInterrupted):
		return huh.ErrUserAborted
	case err != nil:
		return fmt.Errorf("huh: %w", err)
	}
	return nil
}
