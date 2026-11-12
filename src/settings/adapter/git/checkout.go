package git

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/settings"
	"github.com/jterrazz/studio-station/src/settings/contract"
)

// value: what os-checkout clones, and where: the repository its option
// names, else the URL of the shared source its Mac handed the boot phase,
// into `~/Developer/<the repository's name>`. Changing what a workspace
// checks out is this file and the item's options alone.
const checkoutFolder = "Developer"

const cloneBudget = 10 * time.Minute

type Checkout struct {
	run    local.Runner
	exists func(path string) bool
	home   string
	source string
}

var _ contract.Applier = (*Checkout)(nil)

func NewCheckout(run local.Runner, exists func(path string) bool, home, source string) *Checkout {
	return &Checkout{run: run, exists: exists, home: home, source: source}
}

func (*Checkout) Kind() catalog.EffectKind { return catalog.ProcOSCheckout.Kind() }

func (c *Checkout) repository(values map[string]string) string {
	if r := strings.TrimSpace(values["repository"]); r != "" {
		return r
	}
	return c.source
}

func (c *Checkout) folder(repository string) string {
	name := strings.TrimSuffix(path.Base(strings.TrimRight(repository, "/")), ".git")
	if i := strings.LastIndex(name, ":"); i >= 0 {
		name = name[i+1:]
	}
	return path.Join(c.home, checkoutFolder, name)
}

var errNoRepository = errors.New("os-checkout names no repository: give it one (repository), or register a shared source on the Mac the workspace runs on")

func (c *Checkout) Observe(_ context.Context, _ catalog.Setting) (settings.Observation, error) {
	o := settings.Observation{State: catalog.StateAbsent}
	repository := c.repository(nil)
	if repository == "" {
		o.Detail = "no repository named"
		return o, nil
	}
	at := c.folder(repository)
	o.Where = at
	if c.exists(path.Join(at, ".git")) {
		o.State, o.Detail = catalog.StateOK, at
	}
	return o, nil
}

func (c *Checkout) Apply(ctx context.Context, _ catalog.Setting, values map[string]string) (settings.Change, error) {
	repository := c.repository(values)
	if repository == "" {
		return settings.Change{}, errNoRepository
	}
	at := c.folder(repository)
	if c.exists(path.Join(at, ".git")) {
		return settings.Change{Steps: []settings.ChangeStep{done(at, "already there")}}, nil
	}
	if _, err := local.Must(ctx, c.run, local.Cmd{Name: "git", Args: []string{"clone", "--", repository, at}, Timeout: cloneBudget}); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{done(at, "cloned from "+repository)}}, nil
}

func (*Checkout) Undo(context.Context, catalog.Setting) (settings.Change, error) {
	return settings.Change{}, contract.ErrNoUndo
}
