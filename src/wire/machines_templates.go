package wire

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

// value: the Machinefiles this Mac makes workspaces from, read from every
// source as plan reads them, and the drift of the images built from them.
type machineTemplates struct{ m machine }

func (t machineTemplates) read(ctx context.Context, name string) (machinefile.Template, error) {
	got, err := machinefilecalls.Template(ctx, t.m.machinefileDeps(nil), name)
	if err != nil {
		return machinefile.Template{}, machinefilecalls.Worded(err)
	}
	return got, nil
}

func (t machineTemplates) Template(ctx context.Context, name string) (model.Template, error) {
	got, err := t.read(ctx, name)
	if err != nil {
		return model.Template{}, err
	}
	size, err := model.ParseResources(got.CPUs, got.Memory, got.Disk)
	if err != nil {
		return model.Template{}, err
	}
	out := model.Template{
		Name: got.Name, OS: got.OS, Server: got.Kind == catalog.RoleServer, Lifetime: model.Lifetime(got.Lifetime),
		Network: got.Network, Size: size,
	}
	for _, s := range got.Shares {
		out.Shares = append(out.Shares, model.TemplateShare{Account: s.Account, For: term(s.For)})
	}
	for _, f := range got.Files {
		out.Files = append(out.Files, model.TemplateFile{Path: f.Path, Body: f.Body})
	}
	if got.Lock != nil {
		out.Files = append(out.Files, model.TemplateFile{Path: "machinefiles/" + got.Name + catalog.LockSuffix, Body: got.Lock})
	}
	if src, ok, err := t.m.machinefileDeps(nil).Prefs.Get(); err == nil && ok {
		out.Source = src.URL
	}
	return out, nil
}

// value: a Machinefile's term as a share takes it: weeks in days; none, or
// one a share cannot last, is the share's own default.
func term(text string) time.Duration {
	if weeks, ok := strings.CutSuffix(text, "w"); ok {
		if n, err := strconv.Atoi(weeks); err == nil {
			text = strconv.Itoa(n*7) + "d"
		}
	}
	d, _ := accounts.ParseTerm(text)
	return d
}

func (t machineTemplates) Drift(ctx context.Context, name, version string) ([]string, error) {
	rec, err := t.m.stores.images.Get(ctx, name, version)
	if err != nil {
		return nil, err
	}
	now, err := t.read(ctx, name)
	if err != nil {
		return nil, err
	}
	return machinefile.ImageDrift([]byte(rec.Lock), now), nil
}

// value: accounts share's own use case, run for a workspace's first start.
func (m machine) shareAccount(ctx context.Context, account, vm string, term time.Duration) error {
	d := accountscalls.Deps{Accounts: m.accounts(), Shares: m.secrets(), Kind: m.profile.Kind}
	if _, err := accountscalls.Share.Run(ctx, d, accountscalls.ShareIn{Account: account, With: vm, For: term}, nil); err != nil {
		return errors.New(kit.Translate(err, "").Message)
	}
	return nil
}
