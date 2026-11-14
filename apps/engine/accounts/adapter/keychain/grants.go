package keychain

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

const PresenceBudget = time.Minute // value: how long the owner has to answer the presence sheet.

type Answer int

const (
	Given Answer = iota
	Refused
	Absent
)

type Guest interface {
	// value: a shell line run inside a VM of this Mac, its stdin given; the
	// exit code it ended with.
	GuestExec(ctx context.Context, vm, line, stdin string) (code int, err error)
}

type Machine struct {
	Name  string
	Where accounts.Where
	// value: what the machine's own read says of each program, empty when
	// it could not be read.
	Programs map[accounts.Program]accounts.Facts
}

type GrantDeps struct {
	Guest Guest
	// value: this Mac's machines, each as its engine knows it.
	Machines func(ctx context.Context) ([]Machine, error)
	// value: the owner's answer to the presence sheet, asked with the reason given.
	Presence func(ctx context.Context, reason string) Answer
}

type Grants struct{ k *Kept }

var _ contract.Grants = (*Grants)(nil)

const duePrefix = "grant:"

// invariant: a program a machine's own read finds signed in came from a
// grant when the book holds that grant: the facts say so and name its
// entry, so a grant given again is never refused as signed in by itself.
func (k *Kept) machines(ctx context.Context) ([]Machine, error) {
	all, err := k.d.Machines(ctx)
	if err != nil {
		return nil, err
	}
	b, err := k.book.Read(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Machine, 0, len(all))
	for _, m := range all {
		programs := maps.Clone(m.Programs)
		for _, e := range b.Entries {
			f, held := programs[e.Program]
			i := slices.IndexFunc(e.Grants, func(v accounts.VaultGrant) bool { return v.ToKind == accounts.ToMachine && v.To == m.Name })
			if held && i >= 0 && f.State == accounts.ProgramSignedIn {
				f.How, f.Entry, f.EntryLabel = accounts.HowGranted, e.ID, e.Label
				programs[e.Program] = f
			}
		}
		m.Programs = programs
		out = append(out, m)
	}
	return out, nil
}

func (k *Kept) lookup(ctx context.Context, name string) (Machine, bool, error) {
	all, err := k.machines(ctx)
	if err != nil {
		return Machine{}, false, err
	}
	i := slices.IndexFunc(all, func(m Machine) bool { return m.Name == name })
	if i < 0 {
		return Machine{}, false, nil
	}
	return all[i], true, nil
}

func envFile(d catalog.Delivery) string {
	return `"$HOME/` + catalog.GrantsDir + `/` + d.Program + `.env"`
}

// invariant: a token reaches a VM on the guest's stdin, never in its
// command line: into the program's own store by its own line, or into an
// environment file of mode 0600 the zsh block sources.
func injectLine(d catalog.Delivery) string {
	if d.Env == "" {
		return d.Inject
	}
	return `umask 077; mkdir -p "$HOME/` + catalog.GrantsDir + `"; IFS= read -r t; printf "export ` + d.Env + `='%s'\n" "$t" > ` + envFile(d) + `.station && mv ` + envFile(d) + `.station ` + envFile(d)
}

func removeLine(d catalog.Delivery) string {
	if d.Env == "" {
		return d.Remove
	}
	return `rm -f ` + envFile(d)
}

func deliverable(d catalog.Delivery, value []byte) bool {
	if len(value) == 0 || strings.ContainsAny(string(value), "\n\r\x00") {
		return false
	}
	return d.Env == "" || !strings.Contains(string(value), "'")
}

func (k *Kept) takeBack(ctx context.Context, program accounts.Program, vm string) error {
	d, ok := program.Delivery()
	if !ok {
		return nil
	}
	code, err := k.d.Guest.GuestExec(ctx, vm, removeLine(d), "")
	if err == nil && code != 0 {
		err = errors.New("its removal exited " + strconv.Itoa(code))
	}
	return err
}

func (g *Grants) Check(ctx context.Context, id string, to accounts.GrantTo, target string) error {
	e, err := g.k.entry(ctx, id, accounts.GrantUnknownEntry)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(e.Grants, func(v accounts.VaultGrant) bool { return v.ToKind == to && v.To == target }) {
		return nil
	}
	if to == accounts.ToAgent {
		return accounts.Refusal(fail.NotFound, accounts.GrantUnknownAgent, target)
	}
	if _, known, err := g.k.lookup(ctx, target); err != nil {
		return err
	} else if !known {
		return accounts.Refusal(fail.NotFound, accounts.GrantUnknownMachine, target)
	}
	return accounts.Refusal(fail.NotFound, accounts.GrantNotGranted, target)
}

func (g *Grants) Candidates(ctx context.Context, id string) (accounts.Candidates, error) {
	// value: every machine of this Mac the password could go to; no agent is
	// listed until an agent has an identity a grant can name.
	e, err := g.k.entry(ctx, id, accounts.GrantUnknownEntry)
	if err != nil {
		return accounts.Candidates{}, err
	}
	all, err := g.k.machines(ctx)
	if err != nil {
		return accounts.Candidates{}, err
	}
	slices.SortFunc(all, func(a, b Machine) int { return strings.Compare(a.Name, b.Name) })
	found := accounts.Candidates{AgentGrantable: e.AgentGrantable, List: []accounts.Candidate{}}
	for _, m := range all {
		granted := slices.ContainsFunc(e.Grants, func(v accounts.VaultGrant) bool { return v.ToKind == accounts.ToMachine && v.To == m.Name })
		found.List = append(found.List, accounts.Candidate{ToKind: accounts.ToMachine, To: m.Name, Where: m.Where, Facts: m.Programs[e.Program], Granted: granted})
	}
	return found, nil
}

func (g *Grants) target(ctx context.Context, in contract.GrantIn) error {
	if in.ToKind == accounts.ToAgent {
		return accounts.Refusal(fail.NotFound, accounts.GrantUnknownAgent, in.To)
	}
	m, known, err := g.k.lookup(ctx, in.To)
	switch {
	case err != nil:
		return err
	case !known:
		return accounts.Refusal(fail.NotFound, accounts.GrantUnknownMachine, in.To)
	case m.Where.Kind == machine.Server:
		return accounts.Refusal(fail.Refused, accounts.GrantToServer, in.To)
	case m.Where.FarMac != "":
		e := accounts.Refusal(fail.Refused, accounts.GrantOtherMac, in.To)
		e.Detail = map[string]string{"mac": m.Where.FarMac}
		return e
	case m.Where.Stopped:
		return accounts.Refusal(fail.Refused, accounts.GrantMachineStopped, in.To)
	}
	return nil
}

func (g *Grants) Add(ctx context.Context, in contract.GrantIn) (accounts.Granted, error) {
	// invariant: a grant is judged before the owner is asked, the owner is
	// asked before the Keychain is read, and the value read goes to the
	// machine's stdin and nowhere else; only then is the grant written down.
	e, err := g.k.entry(ctx, in.Entry, accounts.GrantUnknownEntry)
	if err != nil {
		return accounts.Granted{}, err
	}
	if refused := g.target(ctx, in); refused != nil {
		return accounts.Granted{}, refused
	}
	m, _, err := g.k.lookup(ctx, in.To)
	if err != nil {
		return accounts.Granted{}, err
	}
	if f := m.Programs[e.Program]; f.State == accounts.ProgramSignedIn && f.How == accounts.HowOwn {
		return accounts.Granted{}, accounts.Refusal(fail.Refused, accounts.GrantSignedInHere, in.To)
	}
	d, ok := e.Program.Delivery()
	if !ok {
		return accounts.Granted{}, accounts.Refusal(fail.Failed, accounts.GrantCannotDeliver, in.To)
	}
	switch g.k.d.Presence(ctx, "give the password "+e.Label+" to "+in.To) {
	case Given:
	case Refused:
		return accounts.Granted{}, accounts.Refusal(fail.Refused, accounts.GrantPresenceRefused, in.Entry)
	default:
		return accounts.Granted{}, accounts.Refusal(fail.Failed, accounts.GrantNoPresence, in.Entry)
	}
	value, err := g.k.keys.Get(ctx, keyOf(e.ID))
	if err != nil || !deliverable(d, value) {
		return accounts.Granted{}, accounts.Refusal(fail.Failed, accounts.GrantCannotDeliver, in.To)
	}
	code, err := g.k.d.Guest.GuestExec(ctx, in.To, injectLine(d), string(value)+"\n")
	if err != nil || code != 0 {
		return accounts.Granted{}, accounts.Refusal(fail.Failed, accounts.GrantCannotDeliver, in.To)
	}
	err = g.k.book.Change(ctx, func(b *accounts.Book) error {
		for i := range b.Entries {
			if b.Entries[i].Program == e.Program {
				b.Entries[i].Grants = slices.DeleteFunc(b.Entries[i].Grants, func(v accounts.VaultGrant) bool { return v.ToKind == accounts.ToMachine && v.To == in.To })
			}
		}
		b.Owed = slices.DeleteFunc(b.Owed, func(o accounts.Owed) bool { return o.Program == e.Program && o.To == in.To })
		kept, ok := b.Find(e.ID)
		if !ok {
			return accounts.Refusal(fail.NotFound, accounts.GrantUnknownEntry, e.ID)
		}
		kept.Grants = append(kept.Grants, accounts.VaultGrant{ToKind: accounts.ToMachine, To: in.To, State: accounts.GrantActive})
		return nil
	})
	return accounts.Granted{}, err
}

func (g *Grants) Remove(ctx context.Context, id string, to accounts.GrantTo, target string) (accounts.Revoked, error) {
	// invariant: a removal takes the token out of the machine by its
	// program's Remove; a machine that cannot be reached owes the removal,
	// so the expiry takes it out when the machine is back.
	if err := g.Check(ctx, id, to, target); err != nil {
		return accounts.Revoked{}, err
	}
	e, err := g.k.entry(ctx, id, accounts.GrantUnknownEntry)
	if err != nil {
		return accounts.Revoked{}, err
	}
	failed := g.k.takeBack(ctx, e.Program, target) != nil
	err = g.k.book.Change(ctx, func(b *accounts.Book) error {
		kept, ok := b.Find(id)
		if !ok {
			return nil
		}
		kept.Grants = slices.DeleteFunc(kept.Grants, func(v accounts.VaultGrant) bool { return v.ToKind == to && v.To == target })
		if failed {
			b.Owed = append(b.Owed, accounts.Owed{Program: e.Program, To: target})
		}
		return nil
	})
	return accounts.Revoked{Sentence: accounts.RevokedSentence(e.Program)}, err
}

func (g *Grants) Due(ctx context.Context) (map[string][]accounts.Share, error) {
	// value: what the expiry owes each machine: a removal that could not
	// reach it, by machine, each named grant:<program>.
	b, err := g.k.book.Read(ctx)
	if err != nil {
		return nil, err
	}
	due := map[string][]accounts.Share{}
	for _, o := range b.Owed {
		due[o.To] = append(due[o.To], accounts.Share{Name: duePrefix + string(o.Program)})
	}
	return due, nil
}

func Due(name string) (accounts.Program, bool) {
	p, ok := strings.CutPrefix(name, duePrefix)
	return accounts.Program(p), ok
}

func (g *Grants) Expire(ctx context.Context, vm string, programs []accounts.Program) error {
	// invariant: the expiry retries each removal owed to the machine; one
	// that is taken out is no longer owed, one that fails stays owed.
	var failed []error
	return errors.Join(g.k.book.Change(ctx, func(b *accounts.Book) error {
		for _, p := range programs {
			err := g.k.takeBack(ctx, p, vm)
			if err != nil {
				failed = append(failed, err)
				continue
			}
			b.Owed = slices.DeleteFunc(b.Owed, func(o accounts.Owed) bool { return o.Program == p && o.To == vm })
		}
		return nil
	}), errors.Join(failed...))
}

func (g *Grants) Forget(ctx context.Context, vm string) error {
	// value: a machine deleted takes what was granted to it with its disk.
	return g.k.book.Change(ctx, func(b *accounts.Book) error {
		for i := range b.Entries {
			b.Entries[i].Grants = slices.DeleteFunc(b.Entries[i].Grants, func(v accounts.VaultGrant) bool { return v.ToKind == accounts.ToMachine && v.To == vm })
		}
		b.Owed = slices.DeleteFunc(b.Owed, func(o accounts.Owed) bool { return o.To == vm })
		return nil
	})
}
