package fake

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

const (
	exchangeLife = 5 * time.Minute
	loginTick    = time.Second
	usualTakes   = 3
)

type Clock struct {
	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error
}

type SignIns struct {
	w     *world.World
	mac   string
	clock Clock
}

var _ contract.SignIns = (*SignIns)(nil)

func scripted(refuse string) fail.Code {
	switch refuse {
	case string(accounts.LoginBusy):
		return accounts.LoginBusy
	case string(accounts.LoginMachineStopped):
		return accounts.LoginMachineStopped
	case string(accounts.LoginNotInstalled):
		return accounts.LoginNotInstalled
	}
	return accounts.LoginFailed
}

func NewSignIns(w *world.World, mac string, clock Clock) *SignIns {
	return &SignIns{w: w, mac: mac, clock: clock}
}

func state(m *world.Mac) *world.Accounts {
	if m.Accounts == nil {
		m.Accounts = &world.Accounts{}
	}
	return m.Accounts
}

func kindOf(m *world.Mac) machine.Kind {
	if m.Self != nil && m.Self.Kind != "" {
		return m.Self.Kind
	}
	return machine.Computer
}

func (s *SignIns) parentOf(name string) (string, *world.VM, bool) {
	for _, parent := range slices.Sorted(maps.Keys(s.w.Macs)) {
		if vm, ok := s.w.Macs[parent].VMs[name]; ok {
			return parent, vm, true
		}
	}
	return "", nil, false
}

func (s *SignIns) standOf(name string) (accounts.Where, bool) {
	m := mac(s.w, name)
	if name == s.mac {
		stand := accounts.Where{Kind: kindOf(m)}
		if _, vm, ok := s.parentOf(name); ok {
			stand.Stopped = vm.PID == 0
		}
		return stand, true
	}
	parent, vm, ok := s.parentOf(name)
	if !ok {
		return accounts.Where{}, false
	}
	stand := accounts.Where{Kind: kindOf(m), Stopped: vm.PID == 0}
	if parent != s.mac {
		stand.FarMac = parent
	}
	return stand, true
}

func (s *SignIns) Read(_ context.Context, name string) (accounts.ProgramsRead, error) {
	s.w.Lock()
	defer s.w.Unlock()
	target := name
	if target == "" {
		target = s.mac
	}
	stand, ok := s.standOf(target)
	if !ok {
		return accounts.ProgramsRead{Verdict: unknownMachine(target), Programs: []accounts.ProgramRow{}}, nil
	}
	own := state(mac(s.w, target))
	rows := make([]accounts.ProgramRow, 0, len(accounts.Programs()))
	for _, program := range accounts.Programs() {
		facts, ok := own.Programs[program]
		if !ok {
			facts = accounts.Facts{State: accounts.ProgramSignedOut}
		}
		facts.Program = program
		rows = append(rows, accounts.Describe(facts, stand))
	}
	return accounts.ProgramsRead{Verdict: accounts.Summarize(rows, target, stand), Programs: rows}, nil
}

func (s *SignIns) Login(ctx context.Context, name string, program accounts.Program) (<-chan contract.LoginUpdate, error) {
	now := s.clock.Now()
	s.w.Lock()
	if parent, ok := s.far(name); ok {
		s.w.Unlock()
		return nil, accounts.Refusal(fail.Refused, accounts.LoginOtherMac, parent)
	}
	target := s.target(name)
	own := state(mac(s.w, target))
	script := own.Logins[program]
	if script.Refuse != "" {
		s.w.Unlock()
		return nil, accounts.Refusal(fail.Refused, scripted(script.Refuse), string(program))
	}
	if stand, ok := s.standOf(target); ok && stand.Stopped {
		s.w.Unlock()
		return nil, accounts.Refusal(fail.Refused, accounts.LoginMachineStopped, string(program))
	}
	if own.Programs[program].State == accounts.ProgramSignedIn {
		s.w.Unlock()
		return nil, accounts.Refusal(fail.Refused, accounts.LoginAlreadySignedIn, string(program))
	}
	step := usual(program, script)
	own.Counter++
	id := "ex-" + strconv.Itoa(own.Counter)
	if own.Exchanges == nil {
		own.Exchanges = map[string]*world.Exchange{}
	}
	own.Exchanges[id] = &world.Exchange{Program: program, Kind: step.Kind, Site: step.Site, Expires: now.Add(exchangeLife), Asking: step.Kind == accounts.StepPaste, Answer: script.Answer}
	s.w.Unlock()

	sent := accounts.Step{Kind: step.Kind, Code: step.Code, Site: step.Site, Exchange: id, Expires: now.Add(exchangeLife), Now: now, Line: lineOf(step), Expired: "The code expired"}
	out := make(chan contract.LoginUpdate)
	go func() {
		defer close(out)
		send := func(u contract.LoginUpdate) bool {
			select {
			case out <- u:
				return true
			case <-ctx.Done():
				return false
			}
		}
		if !send(contract.LoginUpdate{Read: accounts.LoginRead{Step: &sent}}) {
			return
		}
		if err := s.finish(ctx, target, program, id, script); err != nil {
			send(contract.LoginUpdate{Err: err})
			return
		}
		row, _ := s.row(target, program)
		send(contract.LoginUpdate{Read: accounts.LoginRead{Programs: []accounts.ProgramRow{row}}})
	}()
	return out, nil
}

func (s *SignIns) target(name string) string {
	if name == "" {
		return s.mac
	}
	return name
}

func (s *SignIns) finish(ctx context.Context, target string, program accounts.Program, id string, script world.Login) error {
	takes := script.Takes
	if takes == 0 {
		takes = usualTakes
	}
	for range takes {
		if err := s.clock.Sleep(ctx, loginTick); err != nil {
			return err
		}
		if s.given(target, id) {
			break
		}
	}
	switch script.End {
	case "expired":
		return accounts.Refusal(fail.Failed, accounts.LoginExpired, string(program))
	case "failed":
		return accounts.Refusal(fail.Failed, accounts.LoginFailed, string(program))
	case "site-refused":
		return accounts.Refusal(fail.Refused, accounts.LoginSiteRefused, string(program))
	}
	s.w.Lock()
	defer s.w.Unlock()
	own := state(mac(s.w, target))
	if ex := own.Exchanges[id]; ex.Asking && !ex.Given {
		return accounts.Refusal(fail.Failed, accounts.LoginExpired, string(program))
	}
	if own.Programs == nil {
		own.Programs = map[accounts.Program]accounts.Facts{}
	}
	own.Programs[program] = accounts.Facts{Program: program, State: accounts.ProgramSignedIn, Who: "jterrazz", How: accounts.HowOwn}
	delete(own.Exchanges, id)
	return nil
}

func (s *SignIns) given(target, id string) bool {
	s.w.Lock()
	defer s.w.Unlock()
	ex := state(mac(s.w, target)).Exchanges[id]
	return ex != nil && ex.Asking && ex.Given
}

func (s *SignIns) row(target string, program accounts.Program) (accounts.ProgramRow, error) {
	s.w.Lock()
	defer s.w.Unlock()
	stand, _ := s.standOf(target)
	facts, ok := state(mac(s.w, target)).Programs[program]
	if !ok {
		facts = accounts.Facts{State: accounts.ProgramSignedOut}
	}
	facts.Program = program
	return accounts.Describe(facts, stand), nil
}

func (s *SignIns) CheckSignOut(_ context.Context, name string, program accounts.Program) error {
	s.w.Lock()
	defer s.w.Unlock()
	if parent, ok := s.far(name); ok {
		return accounts.Refusal(fail.Refused, accounts.LoginOtherMac, parent)
	}
	target := s.target(name)
	facts, ok := state(mac(s.w, target)).Programs[program]
	stand, _ := s.standOf(target)
	switch {
	case stand.Stopped:
		return accounts.Refusal(fail.Refused, accounts.LoginMachineStopped, string(program))
	case !ok || facts.State != accounts.ProgramSignedIn && facts.State != accounts.ProgramBroken:
		return accounts.Refusal(fail.Refused, accounts.LoginNotSignedIn, string(program))
	case facts.How == accounts.HowGranted:
		return accounts.Refusal(fail.Refused, accounts.LoginGranted, string(program))
	}
	return nil
}

func (s *SignIns) SignOut(_ context.Context, name string, program accounts.Program) (accounts.ProgramRow, error) {
	target := s.target(name)
	s.w.Lock()
	delete(state(mac(s.w, target)).Programs, program)
	s.w.Unlock()
	return s.row(target, program)
}

func (s *SignIns) Open(_ context.Context, name, exchange string) (string, error) {
	now := s.clock.Now()
	s.w.Lock()
	defer s.w.Unlock()
	if parent, ok := s.far(name); ok {
		return "", accounts.Refusal(fail.Refused, accounts.LoginOtherMac, parent)
	}
	ex, err := s.exchange(name, exchange)
	if err != nil {
		return "", err
	}
	if !ex.Expires.After(now) {
		return "", accounts.Refusal(fail.Failed, accounts.LoginExpired, exchange)
	}
	log := state(mac(s.w, s.mac))
	log.Opened = append(log.Opened, ex.Site)
	return ex.Site, nil
}

func (s *SignIns) Answer(_ context.Context, name, exchange string, code secret.Sent) error {
	s.w.Lock()
	defer s.w.Unlock()
	ex, err := s.exchange(name, exchange)
	if err != nil {
		return err
	}
	if !ex.Asking {
		return accounts.Refusal(fail.Refused, accounts.LoginNotAsking, exchange)
	}
	if string(code.Reveal()) != cmp.Or(ex.Answer, "good") {
		return accounts.Refusal(fail.Refused, accounts.LoginCodeRefused, exchange)
	}
	ex.Given = true
	return nil
}

func (s *SignIns) far(name string) (string, bool) {
	if name == "" || name == s.mac {
		return "", false
	}
	parent, _, ok := s.parentOf(name)
	return parent, ok && parent != s.mac
}

func (s *SignIns) exchange(name, id string) (*world.Exchange, error) {
	target := name
	if target == "" {
		target = s.mac
	}
	ex := state(mac(s.w, target)).Exchanges[id]
	if ex == nil {
		return nil, accounts.Refusal(fail.NotFound, accounts.LoginNoExchange, id)
	}
	return ex, nil
}

func usual(program accounts.Program, script world.Login) world.Login {
	by := world.Login{Kind: program.Flow(), Site: program.Site()}
	if by.Kind == accounts.StepDeviceCode {
		by.Code = "WDJB-MJHT"
	}
	if script.Kind != "" {
		by.Kind = script.Kind
	}
	if script.Code != "" {
		by.Code = script.Code
	}
	if script.Site != "" {
		by.Site = script.Site
	}
	return by
}

func lineOf(l world.Login) string {
	switch l.Kind {
	case accounts.StepDeviceCode:
		return "Enter this code at " + l.Site
	case accounts.StepLink:
		return "Open the link to sign in at " + l.Site
	case accounts.StepPaste:
		return "Sign in at " + l.Site + ", then paste the code it gives you here"
	}
	return "Sign in at " + l.Site
}

func unknownMachine(name string) verdict.Verdict {
	return verdict.Verdict{Tone: verdict.Bad, Sentence: "No machine named " + strconv.Quote(name)}
}
