package wire

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
)

// value: this Mac's vault, its values in station's Keychain service and
// its book in the book's store; grants reach a VM through the guest given,
// the socket's for a call, the engine's own for its expiry.
func (m machine) vaultKept(guest localGuest) *keychain.Kept {
	return keychain.NewKept(m.sys.Run, m.stores.book, keychain.Options{Mac: m.where.Machine, Now: m.sys.Clock.Now}, keychain.GrantDeps{Guest: grantGuest{guest}, Machines: m.grantMachines, Presence: askPresence})
}

func askPresence(_ context.Context, reason string) keychain.Answer {
	switch darwin.AskPresence(reason, keychain.PresenceBudget) {
	case darwin.PresenceGiven:
		return keychain.Given
	case darwin.PresenceRefused:
		return keychain.Refused
	case darwin.PresenceAbsent:
	}
	return keychain.Absent
}

type grantGuest struct{ g localGuest }

func (s grantGuest) GuestExec(ctx context.Context, vm, line, stdin string) (int, error) {
	out, err := s.g.GuestExec(ctx, vm, line, stdin)
	return out.Code, err
}

// value: this Mac's VMs as a grant judges them: their kind by lifetime,
// stopped unless its engine says running, and what each one's own read
// says of its programs, empty when it could not be read.
func (m machine) grantMachines(ctx context.Context) ([]keychain.Machine, error) {
	recs, err := m.machines(false).Records()
	if err != nil {
		return nil, err
	}
	live, _ := m.macEngine().VMs(ctx)
	out := make([]keychain.Machine, 0, len(recs))
	for _, r := range recs {
		where := accounts.Where{Kind: r.Lifetime.Kind(), Stopped: live[r.Name].State != engine.StateRunning}
		programs := map[accounts.Program]accounts.Facts{}
		if !where.Stopped {
			if read, err := m.signIns().Read(ctx, r.Name); err == nil {
				for _, row := range read.Programs {
					programs[row.Program] = accounts.Facts{Program: row.Program, State: row.State, Who: row.Who, How: row.How, Entry: row.Entry, Until: row.Until}
				}
			}
		}
		out = append(out, keychain.Machine{Name: r.Name, Where: where, Programs: programs})
	}
	return out, nil
}
