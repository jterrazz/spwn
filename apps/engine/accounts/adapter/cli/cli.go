package cli

import (
	"context"
	"errors"
	"path"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
	LookPath(name string) (string, bool)
}

type Files interface {
	ReadFile(path string) ([]byte, error)
}

type Options struct {
	Home string
	// value: the folder a provider's process is recorded in while it runs, so
	// a read past its budget takes the whole process group with it.
	Track string
}

type Reads struct {
	run   Runner
	files Files
	o     Options
}

var _ contract.Identities = (*Reads)(nil)

func New(run Runner, files Files, o Options) *Reads { return &Reads{run: run, files: files, o: o} }

func (r *Reads) Offers(p catalog.Provider) bool {
	if p.ID != catalog.ProviderDocker {
		return true
	}
	_, listed := r.dockerStore()
	return listed
}

func (r *Reads) dockerStore() (string, bool) {
	data, err := r.files.ReadFile(path.Join(r.o.Home, catalog.DockerSettingsFile))
	if err != nil {
		return "", false
	}
	return dockerStore(data)
}

func (r *Reads) Read(ctx context.Context, p catalog.Provider) (accounts.Identity, error) {
	// invariant: whatever its tool printed, a read hands back no value shaped
	// like a token.
	outs, late := r.answer(ctx, p)
	id := parse(p, outs, late)
	if p.ID == catalog.ProviderDocker && id.State == catalog.StateOK {
		id.Source, _ = r.dockerStore()
	}
	return redactIdentity(id), nil
}

func redactIdentity(id accounts.Identity) accounts.Identity {
	one := func(v string) string {
		c, _ := accounts.Redact(v)
		return c
	}
	all := func(vs []string) []string {
		out := slices.Clone(vs)
		for i, v := range out {
			out[i] = one(v)
		}
		return out
	}
	id.Who, id.Scopes, id.Notes = all(id.Who), all(id.Scopes), all(id.Notes)
	id.Source, id.Detail = one(id.Source), one(id.Detail)
	return id
}

// value: what the provider's commands printed, in order; none when a tool is
// missing; late when the budget ran out first.
func (r *Reads) answer(ctx context.Context, p catalog.Provider) (outs []local.Output, late bool) {
	reads := p.Reads
	if p.ID == catalog.ProviderDocker {
		store, _ := r.dockerStore()
		if store == "" {
			return []local.Output{{Stdout: "{}"}}, false
		}
		reads = []catalog.ProviderRead{{Name: "docker-credential-" + store, Args: []string{"list"}}}
	}
	if p.Command != "" {
		if _, found := r.run.LookPath(p.Command); !found {
			return nil, false
		}
	}
	ctx, cancel := context.WithTimeout(ctx, p.Budget)
	defer cancel()
	for _, read := range reads {
		if _, found := r.run.LookPath(read.Name); !found {
			return nil, false
		}
		out, err := r.run.Output(ctx, local.Cmd{Name: read.Name, Args: read.Args, Timeout: p.Budget, Track: r.o.Track})
		if errors.Is(err, local.ErrTimeout) || ctx.Err() != nil {
			return nil, true
		}
		if err != nil {
			out = local.Output{ExitCode: -1}
		}
		outs = append(outs, out)
	}
	return outs, false
}
