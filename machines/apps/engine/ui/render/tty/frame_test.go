package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func picker() *answered {
	return &answered{form: huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title("On which Mac?").Options(huh.NewOptions("mac-mini", "macbook-pro")...)))}
}

func TestAFormDrawsItsFirstFrameOnceTheTerminalSizeIsApplied(t *testing.T) {
	a := picker()
	a.Init()
	if got := a.View().Content; got != "" {
		t.Errorf("before the program applies the terminal's size, the form draws %q; want nothing, as applying it erases the screen and draws the frame again", got)
	}
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	first := a.View().Content
	for _, want := range []string{"On which Mac?", "mac-mini", "macbook-pro"} {
		if !strings.Contains(first, want) {
			t.Errorf("once the size is applied, the form draws %q; want it to show %q", first, want)
		}
	}
}

func TestAFormsProgramAppliesTheTerminalSizeOnce(t *testing.T) {
	a := picker()
	if got := unasked(a, tea.RequestWindowSize()); got != nil {
		t.Errorf("huh's question for the size reaches the program as %#v; want it dropped, so no second size erases the first frame", got)
	}
	for _, msg := range []tea.Msg{tea.WindowSizeMsg{Width: 80, Height: 24}, tea.KeyPressMsg{Code: tea.KeyEnter}, tea.EnvMsg{"TERM=xterm"}} {
		if got := unasked(a, msg); got == nil {
			t.Errorf("the program's filter drops %#v; want only the size question dropped", msg)
		}
	}
}

func TestACancelledFormQuitsItsProgram(t *testing.T) {
	form := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title("Remove lazygit?")))
	ends(form)
	if msg := form.CancelCmd(); msg != tea.Quit() {
		t.Errorf("ctrl-c at a form ends its program with %#v; want tea.Quit(), as an interrupted program leaves its input reader to take the terminal's next reply", msg)
	}
	if msg := form.SubmitCmd(); msg != tea.Quit() {
		t.Errorf("an answer ends the form's program with %#v; want tea.Quit()", msg)
	}
}
