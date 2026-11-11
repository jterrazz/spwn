package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
)

type Pages func(ctx context.Context, words []string) (page.Page, error)

type Answer struct {
	// value: how the owner answered the question a command asked: yes, or the
	// name typed; the zero Answer runs the command unanswered.
	Yes   bool
	Typed string
	// value: what the owner typed in the command's form, by its input's
	// name: the command reads it on its stdin, never in its words.
	Values map[string]string
}

type Do func(ctx context.Context, words []string, a Answer) (said []string, err error)

type Asked struct {
	// value: the command did nothing yet: it asks first, yes or no, or the name
	// typed when Name is set.
	Message, Name string
}

func (a *Asked) Error() string { return a.Message }

type Deps struct {
	ThisMachine page.Context
	// value: the catalog's settings, whose options a form asks.
	Settings []catalog.Setting
	// value: the machine the TUI opens on, by name: the one this terminal
	// uses; empty is This Mac.
	Opens string
	// invariant: the TUI's one read is the page the CLI prints for these
	// words, so a screen is the same page function as its command.
	Pages Pages
	// invariant: the TUI's one gesture is the command the CLI runs for these
	// words: the call asks its own question, the TUI only puts it. It says,
	// one line each, what it did besides its verdict, as the toast says it
	// after its own words: what it skipped (infra skipped (server)).
	Do Do
	// value: opens a URL or a file with its app (a System Settings pane).
	Open func(ctx context.Context, target string) error
	// value: a new terminal window runs this command line.
	Terminal func(ctx context.Context, line string) error
	// value: runs station with the CLI's words, the call a screen's key
	// makes, and answers what it printed.
	Run     Calls
	NoColor bool
	// value: the registry's Meta, the palette's entries (calls.json).
	Calls []call.Meta
	// value: stops every read still running, so q leaves nothing behind.
	Stop func()
	// value: holds the engine's watch on these machines, by name, until the
	// func it answers is called (SPEC §6.3).
	Watch func(machines []string) (release func())
}

func Run(ctx context.Context, d Deps) error {
	s := newShell(ctx, d)
	_, err := tea.NewProgram(s, tea.WithContext(ctx)).Run()
	s.held.end()
	if d.Stop != nil {
		d.Stop()
	}
	return err
}
