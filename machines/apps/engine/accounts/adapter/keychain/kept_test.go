package keychain_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const fakeValue = "FAKE000"

func pick(fits func(catalog.Delivery) bool) catalog.Delivery {
	for _, d := range catalog.Deliveries {
		if fits(d) {
			return d
		}
	}
	return catalog.Delivery{}
}

// value: a program that keeps the token in its own store and whose sign-in it can be taken from.
func own() catalog.Delivery { return pick(func(d catalog.Delivery) bool { return len(d.Capture) > 0 }) }

// value: a program that reads its token from an environment file.
func env() catalog.Delivery { return pick(func(d catalog.Delivery) bool { return len(d.Env) > 0 }) }

// value: a program of its own store whose token cannot be taken.
func bare() catalog.Delivery {
	return pick(func(d catalog.Delivery) bool { return len(d.Env) == 0 && len(d.Capture) == 0 })
}

type sent struct{ vm, line, stdin string }

// value: a Mac with a login keychain, three VMs (linux-1 and linux-3
// running, linux-2 stopped) and a presence check, each step it took
// recorded in order.
type rig struct {
	login    *login
	steps    []string
	sent     []sent
	answer   keychain.Answer
	unreach  bool
	captured string
	now      time.Time
	vault    *keychain.Kept
	kept     *memBook
	// value: the VMs whose own read finds the program of own() signed in.
	signedIn map[string]bool
}

// value: the book's store in memory, each change on a copy kept only when
// its edit answers no error.
type memBook struct {
	mu   sync.Mutex
	held []byte
}

func (m *memBook) Read(context.Context) (accounts.Book, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.read()
}

func (m *memBook) read() (accounts.Book, error) {
	b := accounts.Book{Entries: []accounts.VaultEntry{}}
	if m.held == nil {
		return b, nil
	}
	err := json.Unmarshal(m.held, &b)
	return b, err
}

func (m *memBook) Change(_ context.Context, edit func(*accounts.Book) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, err := m.read()
	if err != nil {
		return err
	}
	if refused := edit(&b); refused != nil {
		return refused
	}
	m.held, err = json.Marshal(b)
	return err
}

func newRig(_ *testing.T) *rig {
	r := &rig{login: &login{items: map[string]string{}}, signedIn: map[string]bool{}, now: time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)}
	run := local.Played(func(c local.Cmd) (local.Output, error) {
		r.steps = append(r.steps, c.Name+" "+strings.Join(c.Args, " "))
		if c.Name != "security" {
			if r.captured == "" {
				return local.Output{ExitCode: 1, Stderr: "not logged in"}, nil
			}
			return local.Output{Stdout: r.captured + "\n"}, nil
		}
		return r.login.play(c)
	}, "security", own().Capture[0])
	r.kept = &memBook{}
	r.vault = keychain.NewKept(run, r.kept, keychain.Options{Mac: "mac-mini", Now: func() time.Time { return r.now }}, keychain.GrantDeps{
		Guest: r,
		Machines: func(context.Context) ([]keychain.Machine, error) {
			read := func(vm string) map[accounts.Program]accounts.Facts {
				if !r.signedIn[vm] {
					return nil
				}
				p := accounts.Program(own().Program)
				return map[accounts.Program]accounts.Facts{p: {Program: p, State: accounts.ProgramSignedIn, Who: "docs-bot", How: accounts.HowOwn}}
			}
			return []keychain.Machine{
				{Name: "linux-2", Where: accounts.Where{Kind: machine.Workspace, Stopped: true}},
				{Name: "linux-1", Where: accounts.Where{Kind: machine.Workspace}, Programs: read("linux-1")},
				{Name: "linux-3", Where: accounts.Where{Kind: machine.Workspace}, Programs: read("linux-3")},
				{Name: "infra", Where: accounts.Where{Kind: machine.Server}},
			}, nil
		},
		Presence: func(context.Context, string) keychain.Answer {
			r.steps = append(r.steps, "presence")
			return r.answer
		},
	})
	return r
}

func (r *rig) GuestExec(_ context.Context, vm, line, stdin string) (int, error) {
	r.steps = append(r.steps, "guest "+vm)
	r.sent = append(r.sent, sent{vm, line, stdin})
	if r.unreach {
		return 0, errors.New("the guest is not connected")
	}
	return 0, nil
}

func (r *rig) add(t *testing.T, program accounts.Program, label string) string {
	t.Helper()
	return r.addKind(t, program, program.Kinds()[0], label)
}

func (r *rig) addKind(t *testing.T, program accounts.Program, kind accounts.KeyKind, label string) string {
	t.Helper()
	id, err := r.vault.Passwords().Add(t.Context(), contract.PasswordIn{Label: label, Program: program, Kind: kind, From: accounts.FromTyped, Value: secret.Sent{Value: secret.Hold("", []byte(fakeValue))}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return id
}

func codeOf(err error) fail.Code {
	var e *fail.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestPasswords(t *testing.T) {
	contracttest.Passwords(t, func(t *testing.T) contract.Passwords { return newRig(t).vault.Passwords() })
}

func TestGrants(t *testing.T) {
	contracttest.Grants(t, func(t *testing.T) contracttest.GrantsSubject {
		r := newRig(t)
		return contracttest.GrantsSubject{Grants: r.vault.Grants(), Passwords: r.vault.Passwords(), Running: "linux-1", Stopped: "linux-2"}
	})
}

func TestAPasswordsValueIsInTheKeychainOnStdinAndNowhereElse(t *testing.T) {
	r := newRig(t)
	id := r.add(t, accounts.Program(own().Program), "GitHub deploy")
	if _, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	for _, step := range append(slices.Clone(r.steps), r.login.argv...) {
		if strings.Contains(step, fakeValue) {
			t.Errorf("a command line carries the value: %s", step)
		}
	}
	if strings.Contains(string(r.kept.held), fakeValue) {
		t.Errorf("the book holds the value: %s", r.kept.held)
	}
	if r.login.items["vault:"+id] != fakeValue {
		t.Errorf("the Keychain keeps %q under vault:%s; want the value", r.login.items["vault:"+id], id)
	}
	if len(r.sent) != 1 || r.sent[0].stdin != fakeValue+"\n" || strings.Contains(r.sent[0].line, fakeValue) {
		t.Fatalf("sent %+v; want the value on the guest's stdin only", r.sent)
	}
	if r.sent[0].line != own().Inject {
		t.Errorf("line %q; want the program's own store", r.sent[0].line)
	}
	read, _ := r.vault.Passwords().List(t.Context())
	if strings.Contains(read.Entries[0].Label+read.Entries[0].Who, fakeValue) {
		t.Errorf("the list shows the value")
	}
}

func TestPresenceIsAskedAfterTheChecksAndBeforeTheKeychainIsRead(t *testing.T) {
	for _, c := range []struct {
		name   string
		answer keychain.Answer
		to     string
		want   fail.Code
		asked  bool
	}{
		{"a stopped machine is refused before the owner is asked", keychain.Given, "linux-2", accounts.GrantMachineStopped, false},
		{"a server is refused before the owner is asked", keychain.Given, "infra", accounts.GrantToServer, false},
		{"a refused sheet reads nothing", keychain.Refused, "linux-1", accounts.GrantPresenceRefused, true},
		{"a Mac that cannot ask reads nothing", keychain.Absent, "linux-1", accounts.GrantNoPresence, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRig(t)
			r.answer = c.answer
			id := r.add(t, accounts.Program(own().Program), "GitHub deploy")
			r.steps = nil
			_, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: c.to})
			if codeOf(err) != c.want {
				t.Fatalf("Add = %v; want %s", err, c.want)
			}
			if slices.Contains(r.steps, "presence") != c.asked {
				t.Errorf("steps %v; asked = %v, want %v", r.steps, !c.asked, c.asked)
			}
			if slices.ContainsFunc(r.steps, func(s string) bool {
				return strings.Contains(s, "find-generic-password") || strings.HasPrefix(s, "guest")
			}) {
				t.Errorf("steps %v; the Keychain was read or the machine reached", r.steps)
			}
		})
	}
	r := newRig(t)
	id := r.add(t, accounts.Program(own().Program), "GitHub deploy")
	r.steps = nil
	if _, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	asked := slices.Index(r.steps, "presence")
	read := slices.IndexFunc(r.steps, func(s string) bool { return strings.Contains(s, "find-generic-password") })
	if asked < 0 || read < asked {
		t.Fatalf("steps %v; want presence, then the Keychain read", r.steps)
	}
}

func TestAnEnvironmentFileProgramGetsAFileOfItsOwnAndLosesItByName(t *testing.T) {
	r := newRig(t)
	id := r.add(t, accounts.Program(env().Program), "Claude token")
	if _, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	line := r.sent[0].line
	file := `"$HOME/.config/station/grants/` + env().Program + `.env"`
	for _, want := range []string{"umask 077", file, "export " + env().EnvOf(env().Kinds[0]) + "="} {
		if !strings.Contains(line, want) {
			t.Errorf("inject %q; want %q", line, want)
		}
	}
	revoked, err := r.vault.Grants().Remove(t.Context(), id, "linux-1")
	if err != nil || !strings.Contains(revoked.Sentence, "New shells no longer see it") {
		t.Fatalf("Remove = %+v, %v", revoked, err)
	}
	if got := r.sent[1].line; got != "rm -f "+file {
		t.Errorf("remove %q", got)
	}
}

func TestEachKindOfAnEnvironmentFileProgramIsExportedAsItsOwnVariable(t *testing.T) {
	for _, d := range catalog.Deliveries {
		if len(d.Env) == 0 {
			continue
		}
		for _, kind := range d.Kinds {
			r := newRig(t)
			id := r.addKind(t, accounts.Program(d.Program), accounts.KeyKind(kind), d.Program+" "+kind)
			if _, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
				t.Fatalf("%s %s: Add: %v", d.Program, kind, err)
			}
			if want := `printf "export ` + d.EnvOf(kind) + `='%s'\n"`; d.EnvOf(kind) == "" || !strings.Contains(r.sent[0].line, want) {
				t.Errorf("%s %s: inject %q; want %q", d.Program, kind, r.sent[0].line, want)
			}
		}
	}
}

func TestAKindAnEnvironmentFileProgramNamesNoVariableForIsNeverSent(t *testing.T) {
	r := newRig(t)
	id := r.addKind(t, accounts.Program(env().Program), accounts.KeyPassword, "Claude password")
	_, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"})
	if codeOf(err) != accounts.GrantCannotDeliver || len(r.sent) != 0 {
		t.Fatalf("Add = %v, sent %+v; want %s and nothing sent", err, r.sent, accounts.GrantCannotDeliver)
	}
}

func TestAValueTheShellWouldReadOtherwiseIsNeverSent(t *testing.T) {
	r := newRig(t)
	id := r.add(t, accounts.Program(env().Program), "Claude token")
	r.login.items["vault:"+id] = "FAKE'000"
	_, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"})
	if codeOf(err) != accounts.GrantCannotDeliver || len(r.sent) != 0 {
		t.Fatalf("Add = %v, sent %+v; want %s and nothing sent", err, r.sent, accounts.GrantCannotDeliver)
	}
}

func TestAGrantNeverEndsOnItsOwnAndTheRetryTakesOutOnlyWhatIsOwed(t *testing.T) {
	r := newRig(t)
	id := r.add(t, accounts.Program(own().Program), "GitHub deploy")
	g := r.vault.Grants()
	if _, err := g.Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	r.now = r.now.Add(10 * 365 * 24 * time.Hour)
	if due, _ := g.Owed(t.Context()); len(due) != 0 {
		t.Fatalf("a grant years old is owed a removal: %v", due)
	}
	read, _ := r.vault.Passwords().List(t.Context())
	if len(read.Entries[0].Grants) != 1 {
		t.Fatalf("grants %+v; want the one grant still held", read.Entries[0].Grants)
	}
	r.unreach = true
	if _, err := r.vault.Passwords().Remove(t.Context(), id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := g.Retry(t.Context(), "linux-1", []accounts.Program{accounts.Program(own().Program)}); err == nil {
		t.Fatal("a retry that could not reach the machine succeeded")
	}
	if due, _ := g.Owed(t.Context()); len(due["linux-1"]) != 1 {
		t.Fatalf("due %v; want the removal still owed", due)
	}
	r.unreach = false
	if err := g.Retry(t.Context(), "linux-1", []accounts.Program{accounts.Program(own().Program)}); err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if r.sent[len(r.sent)-1].line != own().Remove {
		t.Errorf("last line %q; want the program's own Remove", r.sent[len(r.sent)-1].line)
	}
	if due, _ := g.Owed(t.Context()); len(due) != 0 {
		t.Fatalf("due %v; want nothing owed", due)
	}
}

func TestARemovalThatCannotReachTheMachineIsDueAtItsReturn(t *testing.T) {
	r := newRig(t)
	id := r.add(t, accounts.Program(own().Program), "GitHub deploy")
	g := r.vault.Grants()
	if _, err := g.Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	r.unreach = true
	if _, err := r.vault.Passwords().Remove(t.Context(), id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	due, _ := g.Owed(t.Context())
	if len(due["linux-1"]) != 1 || due["linux-1"][0] != accounts.Program(own().Program) {
		t.Fatalf("due %v; want the program on linux-1 at once", due)
	}
	if err := g.Forget(t.Context(), "linux-1"); err != nil {
		t.Fatal(err)
	}
	if due, _ := g.Owed(t.Context()); len(due) != 0 {
		t.Fatalf("a deleted machine is still owed: %v", due)
	}
}

func TestAPasswordFromASignInIsTheTokenItsCommandPrints(t *testing.T) {
	r := newRig(t)
	r.captured = fakeValue
	id, err := r.vault.Passwords().Add(t.Context(), contract.PasswordIn{Label: "GitHub", Program: accounts.Program(own().Program), Kind: accounts.KeyOAuth, From: accounts.FromSignIn})
	if err != nil || r.login.items["vault:"+id] != fakeValue {
		t.Fatalf("Add = %q, %v; kept %q", id, err, r.login.items["vault:"+id])
	}
	if _, err := r.vault.Passwords().Add(t.Context(), contract.PasswordIn{Label: "no capture", Program: accounts.Program(bare().Program), Kind: accounts.KeyToken, From: accounts.FromSignIn}); codeOf(err) != accounts.VaultCannotCapture {
		t.Fatalf("Add = %v; want %s", err, accounts.VaultCannotCapture)
	}
}

func TestAGrantGivenAgainIsNoSignInOfTheMachinesOwn(t *testing.T) {
	r := newRig(t)
	r.answer = keychain.Given
	id := r.add(t, accounts.Program(own().Program), "GitHub deploy")
	if _, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	r.signedIn["linux-1"], r.signedIn["linux-3"] = true, true
	if _, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-1"}); err != nil {
		t.Fatalf("a grant given again = %v; want it given", err)
	}
	found, err := r.vault.Grants().Candidates(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range found {
		switch c.To {
		case "linux-1":
			if c.Facts.How != accounts.HowGranted || c.Facts.Entry != id {
				t.Errorf("linux-1 facts %+v; want granted from %s", c.Facts, id)
			}
		case "linux-3":
			if c.Facts.How != accounts.HowOwn {
				t.Errorf("linux-3 facts %+v; want its own sign-in", c.Facts)
			}
		}
	}
	if _, err := r.vault.Grants().Add(t.Context(), contract.GrantIn{Entry: id, To: "linux-3"}); codeOf(err) != accounts.GrantSignedInHere {
		t.Fatalf("a grant over the machine's own sign-in = %v; want %s", err, accounts.GrantSignedInHere)
	}
}
