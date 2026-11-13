package calls

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
)

const (
	stepBudget   = 2 * time.Minute
	ExportBudget = stepBudget
)

type SharingOptions struct {
	ThisMachine catalog.MachineKind
	// value: whether a VM of this Mac runs now, as its engine says.
	Running func(ctx context.Context, vm string) (bool, error)
	// value: whether a running VM of this Mac keeps accounts, by
	// accounts.SharedInto over the kind the VM itself says it is.
	Receives func(ctx context.Context, vm string) (bool, error)
	// value: keeps a failed step's cause, never shown raw, and answers
	// where it is kept, empty when it is not.
	Keep func(line string) string
	Now  func() time.Time
}

type Guest interface {
	// invariant: a command inside a VM rides its engine's one connection to it.
	GuestExec(ctx context.Context, vm, line, stdin string) (GuestRun, error)
}

type GuestRun struct {
	Stdout, Stderr string
	ExitCode       int
}

type SharingStores struct {
	ShareList contract.ShareListPrefs
	Macs      machinescontract.MacsPrefs
	Name      setupcontract.NamePrefs
	Shares    contract.ShareStore
	Machines  machinescontract.MachineStore
}

type Sharing struct {
	guest  Guest
	st     SharingStores
	source contract.SecretSource
	ids    contract.Identities
	o      SharingOptions
}

func NewSharing(guest Guest, st SharingStores, source contract.SecretSource, ids contract.Identities, o SharingOptions) *Sharing {
	return &Sharing{guest: guest, st: st, source: source, ids: ids, o: o}
}

func (s *Sharing) ShareList() []string {
	list, err := s.st.ShareList.Get()
	if err != nil {
		return nil
	}
	return list
}

func (s *Sharing) Shares(ctx context.Context) (map[string][]accounts.Share, error) {
	return s.st.Shares.List(ctx)
}

func (s *Sharing) thisMac() string {
	name, err := s.st.Name.Get()
	if err != nil || name == "" {
		return "This Mac"
	}
	return name
}

func (s *Sharing) CheckShare(ctx context.Context, account, vm string) (secret string, err error) {
	// invariant: a share is refused before anything is read or copied: an
	// unknown account, one the share list does not hold, a VM that may not
	// receive it, an account not signed in here.
	if s.o.ThisMachine != catalog.Mac {
		return "", &accounts.OffMacError{ThisMachine: s.o.ThisMachine}
	}
	if !accounts.Known(account) {
		return "", &accounts.UnknownError{Account: account}
	}
	secret, ok := accounts.SecretOf(account)
	list, err := s.st.ShareList.Get()
	if err != nil {
		return "", err
	}
	if !ok || !accounts.Shareable(secret, list) {
		return "", &accounts.NotShareableError{Account: account}
	}
	if err := s.CheckReceiver(ctx, vm); err != nil {
		return "", err
	}
	if s.ids != nil {
		for _, p := range catalog.Providers {
			if p.ID != account {
				continue
			}
			if id, err := s.ids.Read(ctx, p); err != nil || id.State != catalog.StateOK {
				return "", &accounts.NotSignedInError{Account: account, Mac: "This Mac"}
			}
		}
	}
	return secret, nil
}

func (s *Sharing) CheckFrom(ctx context.Context, account, vm, from string) error {
	// invariant: a share is its VM's own Mac's: asked from another Mac, the
	// VM is judged first, so a server's refusal stands, then it is refused.
	if from == "" || from == s.thisMac() {
		return nil
	}
	if err := s.CheckReceiver(ctx, vm); err != nil {
		return err
	}
	return &accounts.OtherMacError{Account: account, VM: vm, Mac: s.thisMac()}
}

func (s *Sharing) CheckReceiver(ctx context.Context, vm string) error {
	// invariant: a share lands only in a running VM this Mac keeps that
	// keeps accounts: a server keeps its own secrets.
	if err := s.checkVM(ctx, vm); err != nil {
		return err
	}
	if s.o.Running != nil {
		running, err := s.o.Running(ctx, vm)
		if err != nil {
			return err
		}
		if !running {
			return &accounts.VMStoppedError{VM: vm}
		}
	}
	if s.o.Receives != nil {
		receives, err := s.o.Receives(ctx, vm)
		if err != nil {
			return &accounts.StepError{VM: vm, Step: "reaching it", Cause: err}
		}
		if !receives {
			return &accounts.ToServerError{VM: vm}
		}
	}
	return nil
}

func (s *Sharing) Share(ctx context.Context, account, vm string, term time.Duration) (accounts.Shared, error) {
	term, ok := accounts.CheckTerm(term)
	if !ok {
		return accounts.Shared{}, &accounts.TermError{Account: account, VM: vm, Term: term}
	}
	secret, err := s.CheckShare(ctx, account, vm)
	if err != nil {
		return accounts.Shared{}, err
	}
	found, err := s.inVM(ctx, vm, "command -v studio", "")
	switch {
	case err != nil:
		return accounts.Shared{}, &accounts.StepError{VM: vm, Step: "reaching it", Cause: err}
	case found.ExitCode != 0:
		return accounts.Shared{}, &accounts.NoStudioError{VM: vm}
	}
	exported, err := s.source.Export(ctx, []string{secret})
	if err != nil {
		return accounts.Shared{}, &accounts.UnreadableError{Account: account, Mac: "This Mac", Cause: err, Log: s.keep("share "+account+" with "+vm, err)}
	}
	// invariant: the secret travels from the export's output to the VM's
	// stdin and nowhere else: never argv, never a file on the Mac.
	imported, err := s.inVM(ctx, vm, accounts.VMStudio("import"), string(exported))
	if err != nil || imported.ExitCode != 0 {
		return accounts.Shared{}, &accounts.StepError{VM: vm, Step: "copying " + accounts.Label(account), Cause: failure(err, imported)}
	}
	at := s.o.Now().UTC().Truncate(time.Second)
	share := accounts.Share{Name: secret, At: at, Until: at.Add(term)}
	held, err := s.st.Shares.Get(ctx, vm)
	if err != nil {
		return accounts.Shared{}, err
	}
	held = slices.DeleteFunc(held, func(have accounts.Share) bool { return have.Name == secret })
	if err := s.st.Shares.Put(ctx, vm, append(held, share)); err != nil {
		return accounts.Shared{}, err
	}
	return accounts.Shared{Account: account, VM: vm, At: share.At, Until: share.Until}, nil
}

func (s *Sharing) keep(what string, cause error) string {
	// invariant: a source's own words go to the log and only there; the
	// refusal names the file, shown from home, or nothing when none is kept.
	if s.o.Keep == nil {
		return ""
	}
	return s.o.Keep(s.o.Now().UTC().Format(time.RFC3339) + " " + what + ": " + cause.Error() + "\n")
}

func (s *Sharing) CheckTakeBack(ctx context.Context, account, vm string) (string, error) {
	// invariant: a take-back names an account this Mac shares with that VM;
	// anything else is refused before the VM is asked.
	if !accounts.Known(account) {
		return "", &accounts.UnknownError{Account: account}
	}
	if err := s.checkVM(ctx, vm); err != nil {
		return "", err
	}
	held, err := s.st.Shares.Get(ctx, vm)
	if err != nil {
		return "", err
	}
	secret, _ := accounts.SecretOf(account)
	if !slices.ContainsFunc(held, func(sh accounts.Share) bool { return sh.Name == secret }) {
		return "", &accounts.NotSharedError{Account: account, VM: vm}
	}
	return secret, nil
}

func (s *Sharing) TakeBack(ctx context.Context, account, vm string) error {
	secret, err := s.CheckTakeBack(ctx, account, vm)
	if err != nil {
		return err
	}
	return s.Expire(ctx, vm, []string{secret})
}

func (s *Sharing) Expire(ctx context.Context, vm string, secrets []string) error {
	// value: deletes the VM's copies of these secrets, then their shares.
	out, err := s.inVM(ctx, vm, accounts.VMStudio(append([]string{"delete"}, secrets...)...), "")
	if err != nil || out.ExitCode != 0 {
		return &accounts.StepError{VM: vm, Step: "deleting the copies", Cause: failure(err, out)}
	}
	held, err := s.st.Shares.Get(ctx, vm)
	if err != nil {
		return err
	}
	return s.st.Shares.Put(ctx, vm, slices.DeleteFunc(held, func(sh accounts.Share) bool { return slices.Contains(secrets, sh.Name) }))
}

func (s *Sharing) Forget(ctx context.Context, vm string) error {
	// value: a deleted VM takes its copies with its disk; its shares go too.
	return s.st.Shares.Delete(ctx, vm)
}

func (s *Sharing) checkVM(ctx context.Context, name string) error {
	var unreadable *store.UnreadableError
	if _, err := s.st.Machines.Get(ctx, name); err == nil || errors.As(err, &unreadable) {
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	macs, err := s.st.Macs.Get()
	if err != nil {
		return err
	}
	if _, mac := macs[name]; mac || name == s.thisMac() {
		return &accounts.NotAVMError{Name: name}
	}
	return &accounts.UnknownVMError{VM: name}
}

func (s *Sharing) inVM(ctx context.Context, vm, line, stdin string) (GuestRun, error) {
	ctx, cancel := context.WithTimeout(ctx, stepBudget)
	defer cancel()
	return s.guest.GuestExec(ctx, vm, line, stdin)
}

func failure(err error, out GuestRun) error {
	if err != nil {
		return err
	}
	if text := strings.TrimSpace(out.Stderr); text != "" {
		return errors.New(text)
	}
	return fmt.Errorf("exit %d", out.ExitCode)
}
