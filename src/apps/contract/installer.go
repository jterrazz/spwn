package contract

import (
	"context"
	"errors"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
)

var (
	ErrMissing = errors.New("its manager is not installed")
	// invariant: a method that keeps no list of what it installed: an app's
	// own probe, or its bundle, tells whether it is there.
	ErrUnlisted = errors.New("its method keeps no list")
	// invariant: a verb its Abilities deny is refused, never faked.
	ErrCannot = errors.New("its method cannot do this")
)

// invariant: a read of what is behind goes to the network, so it is bounded: a
// manager that does not answer in this time leaves its apps' currency unknown
// and never holds `station apps`.
const (
	CurrencyBudget = 5 * time.Second
)

type Logs []string // value: the files a verb appends its program's raw output to, one per app it serves (`~/.local/state/station/logs/<app>.log`); none leaves it on the terminal's stderr.

type Installer interface {
	Method() catalog.Method
	Manager() Manager
	Can() Abilities
	Ready(ctx context.Context) error
	Inventory(ctx context.Context) (map[string]string, error)
	Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error)
	Install(ctx context.Context, installs []catalog.Install, logs Logs) error
	Update(ctx context.Context, installs []catalog.Install, logs Logs) error
	Remove(ctx context.Context, installs []catalog.Install, logs Logs) error
}

type For struct {
	// value: whom a verb serves: the catalogue app it runs for, and whether
	// the owner named that app in person.
	App string
	// value: the owner's own word for the app, a removal they asked of it
	// by name and confirmed; an apply's removal, which the ledger decides,
	// carries none.
	Word bool
}

type Holder interface {
	// invariant: a method that runs as root over a package database the
	// owner also installs into by hand (apt) records each package its
	// install adds by the app it ran for, and takes back one it never
	// recorded adding only on the owner's word for that app. The apps model
	// runs its installs and removals through these, never through
	// Installer's own.
	InstallFor(ctx context.Context, f For, installs []catalog.Install, logs Logs) error
	RemoveFor(ctx context.Context, f For, installs []catalog.Install, logs Logs) error
	// value: whether station recorded adding any package for the app.
	Holds(ctx context.Context, app string) (bool, error)
}

type Newest interface {
	// invariant: a method that reads the newest version its manager offers
	// of an install, whatever range holds it, as `mise latest node` does; a
	// method with no such read leaves a newer major unsaid, never guessed.
	Newest(ctx context.Context, in catalog.Install) (string, error)
}

type Manager struct {
	// value: the manager's word in a row's detail: "brew did not answer".
	Name string
	// value: the catalogue app that provides the manager, named when it is missing.
	App string
}

type Abilities struct {
	Install bool
	Update  bool
	Remove  bool
	// invariant: a bare update reaches the method; one it does not reach is
	// updated by name only.
	UpdateAll bool
}

type Installers map[catalog.Method]Installer // invariant: the one registry; no other code switches on a method.

type Lined interface {
	// invariant: a method whose installs and updates ride one shared run (D7's
	// brew bundle) spells each install's line; the Bundler runs the lines of
	// every such method at once.
	Line(i catalog.Install) string
}

type Bundler interface {
	// value: the run's name in a verdict: "not listed after brew bundle".
	Name() string
	Bundle(ctx context.Context, lines []string, update bool, logs Logs) error
}
