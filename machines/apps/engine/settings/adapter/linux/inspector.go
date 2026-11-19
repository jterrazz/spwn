package linux

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Inspector struct{ run Runner }

var _ contract.Inspector = (*Inspector)(nil)

func New(run Runner) *Inspector { return &Inspector{run: run} }

func (i *Inspector) Read(ctx context.Context, f catalog.Fact) (settings.Fact, error) {
	if f == catalog.FactDNS {
		return i.dns(ctx)
	}
	if r, ok := firsts()[f]; ok {
		return i.first(ctx, r.cmd, r.word)
	}
	return settings.Unread(), nil
}

type firstRead struct {
	cmd local.Cmd
	// value: the fact is the first word printed, not the whole line.
	word bool
}

func firsts() map[catalog.Fact]firstRead {
	return map[catalog.Fact]firstRead{
		catalog.FactKernel:  {cmd: local.Command("uname", "-sr")},
		catalog.FactLocalIP: {cmd: local.Command("hostname", "-I"), word: true},
	}
}

func (i *Inspector) ask(ctx context.Context, c local.Cmd) (out local.Output, ok bool, err error) {
	out, err = i.run.Output(ctx, c)
	if errors.Is(err, local.ErrTimeout) || ctx.Err() != nil {
		return local.Output{}, false, local.ErrTimeout
	}
	return out, err == nil && out.ExitCode == 0, nil
}

// invariant: a value a tool prints whole (the kernel) or as its first word
// (an address); nothing printed is absent.
func (i *Inspector) first(ctx context.Context, c local.Cmd, word bool) (settings.Fact, error) {
	out, ok, err := i.ask(ctx, c)
	if err != nil {
		return settings.Fact{}, err
	}
	value := strings.TrimSpace(out.Stdout)
	if fields := strings.Fields(value); word && len(fields) > 0 {
		value = fields[0]
	}
	if !ok || value == "" {
		return settings.Fact{State: catalog.StateAbsent}, nil
	}
	return settings.Fact{State: catalog.StateOK, Value: value}, nil
}

// outside: /etc/resolv.conf lists `nameserver 10.0.2.3`; loopback and IPv6
// resolvers are the local stub, not a provider.
func (i *Inspector) dns(ctx context.Context) (settings.Fact, error) {
	out, ok, err := i.ask(ctx, local.Command("cat", "/etc/resolv.conf"))
	if err != nil {
		return settings.Fact{}, err
	}
	var servers []string
	for _, line := range strings.Split(out.Stdout, "\n") {
		rest, found := strings.CutPrefix(strings.TrimSpace(line), "nameserver")
		server := strings.TrimSpace(rest)
		switch {
		case !found, server == "", server == "127.0.0.1", strings.Contains(server, ":"), len(servers) >= 2, slices.Contains(servers, server):
			continue
		}
		servers = append(servers, server)
	}
	if !ok || len(servers) == 0 {
		return settings.Fact{State: catalog.StateAbsent}, nil
	}
	return settings.Fact{State: catalog.StateOK, Value: strings.Join(servers, ", ")}, nil
}
