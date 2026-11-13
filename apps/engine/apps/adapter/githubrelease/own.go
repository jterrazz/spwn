package githubrelease

import (
	"context"
	"maps"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Held struct {
	// value: station's own release, by the key its manager package installs by.
	Key string
}

type owned struct {
	contract.Installer
	self Held
}

func Own(i contract.Installer, self Held) contract.Installer {
	// invariant: station is the github-release manager, and the running
	// binary its own release: listed (its version is station's own
	// line, `station --version`), always current, and
	// never installed, updated nor removed by the installer of releases,
	// whichever that installer is (a machine's, a world's).
	if self.Key == "" {
		return i
	}
	return owned{Installer: i, self: self}
}

func (o owned) Inventory(ctx context.Context) (map[string]string, error) {
	held, err := o.Installer.Inventory(ctx)
	if err != nil {
		return nil, err
	}
	out := maps.Clone(held)
	if out == nil {
		out = map[string]string{}
	}
	out[o.self.Key] = ""
	return out, nil
}

func (o owned) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	var others []string
	for _, k := range keys {
		if k != o.self.Key {
			others = append(others, k)
		}
	}
	out, err := o.Installer.Outdated(ctx, others)
	if err != nil {
		return nil, err
	}
	out = maps.Clone(out)
	if out == nil {
		out = map[string]apps.Currency{}
	}
	if len(others) < len(keys) {
		out[o.self.Key] = apps.Currency{Verdict: apps.Current}
	}
	return out, nil
}

func (o owned) refused(installs []catalog.Install) error {
	for _, i := range installs {
		if i.Key() == o.self.Key {
			return &SelfError{Repo: i.Key()}
		}
	}
	return nil
}

func (o owned) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	if err := o.refused(installs); err != nil {
		return err
	}
	return o.Installer.Install(ctx, installs, logs)
}

func (o owned) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	if err := o.refused(installs); err != nil {
		return err
	}
	return o.Installer.Update(ctx, installs, logs)
}

func (o owned) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	if err := o.refused(installs); err != nil {
		return err
	}
	return o.Installer.Remove(ctx, installs, logs)
}
