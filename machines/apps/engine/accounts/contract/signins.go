package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

type SignIns interface {
	// value: the programs of the machine named, empty for this one, each row with its action.
	Read(ctx context.Context, machine string) (accounts.ProgramsRead, error)
	// value: where the machine named is, empty for this one: false for a name
	// no record holds.
	Where(ctx context.Context, machine string) (accounts.Where, bool)
	// invariant: Login answers the refusals before it returns; the channel
	// then carries the step, then the signed-in row, and closes when the
	// exchange ends or ctx does. A failure after it rides as a refusal. The
	// machine is a VM of this Mac, empty for this one.
	Login(ctx context.Context, machine string, program accounts.Program) (<-chan LoginUpdate, error)
	// invariant: what cannot be signed out is refused here, before the
	// question; the machine is a VM of this Mac, empty for this one.
	CheckSignOut(ctx context.Context, machine string, program accounts.Program) error
	SignOut(ctx context.Context, machine string, program accounts.Program) (accounts.ProgramRow, error)
	// value: the site the owner is sent to for an exchange of the machine named, opened by this Mac's engine.
	Open(ctx context.Context, machine, exchange string) (string, error)
	Answer(ctx context.Context, machine, exchange string, code secret.Sent) error
}

type LoginUpdate struct {
	Read accounts.LoginRead
	Err  error
}

type Passwords interface {
	List(ctx context.Context) (accounts.VaultRead, error)
	Check(ctx context.Context, id string) error
	Add(ctx context.Context, in PasswordIn) (string, error)
	Remove(ctx context.Context, id string) ([]accounts.VaultGrant, error)
}

type PasswordIn struct {
	Label   string
	Program accounts.Program
	Kind    accounts.KeyKind
	From    accounts.AddFrom
	Value   secret.Sent
}

type Grants interface {
	Check(ctx context.Context, entry, target string) error
	Add(ctx context.Context, in GrantIn) (accounts.Granted, error)
	Remove(ctx context.Context, entry, target string) (accounts.Revoked, error)
	// value: every machine of this Mac a password could go to, each as the Mac reads it.
	Candidates(ctx context.Context, entry string) ([]accounts.Candidate, error)
}

type GrantIn struct {
	Entry string
	To    string
}
