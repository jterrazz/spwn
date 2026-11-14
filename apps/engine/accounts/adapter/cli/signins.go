package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

// value: how long an exchange waits for the owner, the shortest any of the
// programs gives its code.
const exchangeLife = 5 * time.Minute

// value: a missing program, as a shell answers it.
const notFound = 127

var ErrStopped = errors.New("the machine is stopped or its guest is not connected")

type Ran struct {
	Code int
	// value: what the command printed, its errors after its output.
	Out string
}

type Session interface {
	// value: a line running on a machine: its output and errors in one
	// stream, its input, and its exit code once it ended.
	io.Reader
	io.Writer
	Wait() (int, error)
}

type Machines interface {
	// invariant: a line runs under /bin/sh on the machine named, empty for
	// this one, a VM through its Mac's own connection; a stopped VM or one
	// whose guest is absent answers ErrStopped. A secret rides stdin, never
	// the line.
	Exec(ctx context.Context, machine, line, stdin string) (Ran, error)
	Attach(ctx context.Context, machine, line string) (Session, error)
	// value: where a row is read: this machine, a VM of this Mac or a VM of
	// another Mac; false when no machine has that name.
	Where(ctx context.Context, machine string) (accounts.Where, bool)
	// value: opens a page in the owner's browser on this Mac.
	Open(ctx context.Context, page string) error
	Now() time.Time
}

type SignIns struct {
	m Machines

	mu        sync.Mutex
	exchanges map[string]*exchange
	counter   int
}

type exchange struct {
	machine string
	program accounts.Program
	kind    accounts.StepKind
	link    string
	expires time.Time
	input   io.Writer
}

var _ contract.SignIns = (*SignIns)(nil)

func NewSignIns(m Machines) *SignIns {
	return &SignIns{m: m, exchanges: map[string]*exchange{}}
}

func loginOf(p accounts.Program) catalog.LoginProgram {
	for _, known := range catalog.LoginPrograms {
		if known.Name == string(p) {
			return known
		}
	}
	return catalog.LoginProgram{Name: string(p)}
}

func quote(v string) string { return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'" }

// invariant: every line names its program as the shell's $0 and runs the
// program found by catalog.LoginBinary, never a shim, by exec, so a signal
// to the line reaches the program itself; its errors join its output.
func line(p catalog.LoginProgram, args []string, prelude string) string {
	words := make([]string, 0, len(args))
	for _, a := range args {
		words = append(words, quote(a))
	}
	script := catalog.LoginBinary + "; " + prelude + `exec "$p" ` + strings.Join(words, " ") + " 2>&1"
	return "/bin/sh -c " + quote(script) + " " + quote(p.Name)
}

// value: a program that would open a browser finds the true command as its
// browser: it opens nothing, so no browser start is attempted inside a VM
// and no file is left behind; the link or code is printed on the line's own
// output.
const noBrowser = "export BROWSER=/usr/bin/true; "

func (s *SignIns) Read(ctx context.Context, machine string) (accounts.ProgramsRead, error) {
	where, ok := s.m.Where(ctx, machine)
	name := machine
	if name == "" {
		name = "this machine"
	}
	if !ok {
		return accounts.ProgramsRead{Verdict: unknown(name), Programs: []accounts.ProgramRow{}}, nil
	}
	rows := make([]accounts.ProgramRow, 0, len(catalog.LoginPrograms))
	for _, program := range accounts.Programs() {
		facts := accounts.Facts{Program: program, State: accounts.ProgramSignedOut}
		if !where.Stopped && where.FarMac == "" {
			facts = s.detect(ctx, machine, program)
		}
		rows = append(rows, accounts.Describe(facts, where))
	}
	return accounts.ProgramsRead{Verdict: accounts.Summarize(rows, name, where), Programs: rows}, nil
}

func (s *SignIns) detect(ctx context.Context, machine string, program accounts.Program) accounts.Facts {
	p := loginOf(program)
	ran, err := s.m.Exec(ctx, machine, line(p, p.Detect, ""), "")
	facts := accounts.Facts{Program: program, State: accounts.ProgramSignedOut}
	if err != nil || ran.Code == notFound {
		return facts
	}
	facts.State = signedIn(p.Reads, ran)
	if facts.State == accounts.ProgramSignedIn {
		facts.How = accounts.HowOwn
		facts.Who, facts.Plan = who(p.Reads, ran.Out)
	}
	return facts
}

func signedIn(r catalog.SignInRead, ran Ran) accounts.ProgramState {
	if r.JSON != "" {
		var answer map[string]any
		if json.Unmarshal([]byte(jsonPart(ran.Out)), &answer) != nil {
			return accounts.ProgramBroken
		}
		if in, _ := answer[r.JSON].(bool); in {
			return accounts.ProgramSignedIn
		}
		return accounts.ProgramSignedOut
	}
	switch {
	case ran.Code == 0:
		return accounts.ProgramSignedIn
	case r.Out == "" || strings.Contains(ran.Out, r.Out):
		return accounts.ProgramSignedOut
	}
	return accounts.ProgramBroken
}

// value: the JSON object of an answer whose program printed a line before it.
func jsonPart(out string) string {
	if i := strings.Index(out, "{"); i >= 0 {
		return out[i:]
	}
	return out
}

func who(r catalog.SignInRead, out string) (name, plan string) {
	// invariant: whatever the program printed, a row holds no value shaped
	// like a token.
	screen := func(v string) string {
		c, _ := accounts.Redact(strings.TrimSpace(v))
		return c
	}
	if r.WhoKey != "" || r.PlanKey != "" {
		var answer map[string]any
		_ = json.Unmarshal([]byte(jsonPart(out)), &answer)
		name, _ = answer[r.WhoKey].(string)
		plan, _ = answer[r.PlanKey].(string)
		if said, ok := r.Plans[plan]; ok {
			plan = said
		}
		return screen(name), screen(plan)
	}
	if r.Who != "" {
		if m := regexp.MustCompile(r.Who).FindStringSubmatch(out); len(m) > 1 {
			return screen(m[1]), ""
		}
		return "", ""
	}
	first, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	return screen(first), ""
}

func (s *SignIns) Login(ctx context.Context, machine string, program accounts.Program) (<-chan contract.LoginUpdate, error) {
	refuse := func(code fail.Code) error { return accounts.Refusal(fail.Refused, code, string(program)) }
	where, ok := s.m.Where(ctx, machine)
	switch {
	case !ok:
		return nil, accounts.Refusal(fail.NotFound, accounts.LoginFailed, machine)
	case where.FarMac != "":
		return nil, accounts.Refusal(fail.Refused, accounts.LoginOtherMac, where.FarMac)
	case where.Stopped:
		return nil, refuse(accounts.LoginMachineStopped)
	case s.busy(machine, program):
		return nil, refuse(accounts.LoginBusy)
	}
	p := loginOf(program)
	ran, err := s.m.Exec(ctx, machine, line(p, p.Detect, ""), "")
	switch {
	case errors.Is(err, ErrStopped):
		return nil, refuse(accounts.LoginMachineStopped)
	case err != nil:
		return nil, accounts.Refusal(fail.Failed, accounts.LoginFailed, string(program))
	case ran.Code == notFound:
		return nil, refuse(accounts.LoginNotInstalled)
	case signedIn(p.Reads, ran) == accounts.ProgramSignedIn:
		return nil, refuse(accounts.LoginAlreadySignedIn)
	}
	now := s.m.Now()
	run, cancel := context.WithDeadline(ctx, now.Add(exchangeLife))
	session, err := s.m.Attach(run, machine, line(p, p.Login, noBrowser))
	if err != nil {
		cancel()
		if errors.Is(err, ErrStopped) {
			return nil, refuse(accounts.LoginMachineStopped)
		}
		return nil, accounts.Refusal(fail.Failed, accounts.LoginFailed, string(program))
	}
	ex := &exchange{machine: machine, program: program, kind: program.Flow(), expires: now.Add(exchangeLife), input: session}
	id := s.hold(ex)
	out := make(chan contract.LoginUpdate)
	go func() {
		defer close(out)
		defer cancel()
		defer s.drop(id)
		send := func(u contract.LoginUpdate) bool {
			select {
			case out <- u:
				return true
			case <-ctx.Done():
				return false
			}
		}
		step, err := s.step(run, session, p, ex, id, now)
		if err != nil {
			cancel()
			_, _ = session.Wait()
			send(contract.LoginUpdate{Err: err})
			return
		}
		if !send(contract.LoginUpdate{Read: accounts.LoginRead{Step: &step}}) {
			return
		}
		code, err := session.Wait()
		switch {
		case run.Err() != nil && ctx.Err() == nil:
			send(contract.LoginUpdate{Err: accounts.Refusal(fail.Failed, accounts.LoginExpired, string(program))})
			return
		case err != nil || code != 0:
			send(contract.LoginUpdate{Err: accounts.Refusal(fail.Failed, accounts.LoginFailed, string(program))})
			return
		}
		facts := s.detect(ctx, machine, program)
		send(contract.LoginUpdate{Read: accounts.LoginRead{Programs: []accounts.ProgramRow{accounts.Describe(facts, where)}}})
	}()
	return out, nil
}

var linkPattern = regexp.MustCompile(`https://[^\s"'<>]+`)

// value: how long a program that printed only links off its site is given
// to print its own before the login is refused.
const siteGrace = 5 * time.Second

// invariant: the step is sent once the program printed what its kind needs
// (the code for a device code, the link otherwise); the link kept is the
// first on the program's site or one of its subdomains, any other is
// skipped and never kept, so no other site is opened; a program that
// printed links off its site and none on it is refused.
func (s *SignIns) step(ctx context.Context, session io.Reader, p catalog.LoginProgram, ex *exchange, id string, now time.Time) (accounts.Step, error) {
	var code *regexp.Regexp
	if p.Code != "" {
		code = regexp.MustCompile(p.Code)
	}
	armed := p.CodeAfter == ""
	step := accounts.Step{Kind: ex.kind, Site: p.Site, Exchange: id, Expires: ex.expires, Now: now}
	texts := make(chan string)
	go func() {
		defer close(texts)
		lines := bufio.NewScanner(session)
		for lines.Scan() {
			texts <- lines.Text()
		}
	}()
	defer func() { go drain(texts) }()
	refused := accounts.Refusal(fail.Refused, accounts.LoginSiteRefused, p.Name)
	var late <-chan struct{}
	held, stopHeld := context.WithCancel(context.WithoutCancel(ctx))
	defer stopHeld()
	offSite := false
	for {
		var text string
		select {
		case t, ok := <-texts:
			if !ok {
				if offSite {
					return accounts.Step{}, refused
				}
				return accounts.Step{}, accounts.Refusal(fail.Failed, accounts.LoginFailed, p.Name)
			}
			text = t
		case <-late:
			return accounts.Step{}, refused
		}
		if code != nil && armed && step.Code == "" {
			if m := code.FindStringSubmatch(text); len(m) > 1 {
				step.Code = m[1]
			}
		}
		armed = armed || strings.Contains(text, p.CodeAfter)
		for _, found := range linkPattern.FindAllString(text, -1) {
			if s.linkOf(id) != "" {
				break
			}
			if onSite(found, p.Site) {
				s.keep(id, found)
				late = nil
				break
			}
			if !offSite {
				offSite = true
				grace, stop := context.WithTimeout(held, siteGrace)
				context.AfterFunc(grace, stop)
				late = grace.Done()
			}
		}
		if ex.kind == accounts.StepDeviceCode && step.Code != "" || ex.kind != accounts.StepDeviceCode && s.linkOf(id) != "" {
			step.Line = ex.program.StepLine(ex.kind, p.Site)
			return step, nil
		}
	}
}

// invariant: once the step is sent, the program's later output is read and
// dropped, so a program that keeps printing never blocks on its output.
func drain(texts <-chan string) {
	for range texts {
	}
}

func onSite(link, site string) bool {
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	name := u.Hostname()
	return name == site || strings.HasSuffix(name, "."+site)
}

func (s *SignIns) busy(machine string, program accounts.Program) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(slices.Collect(maps.Values(s.exchanges)), func(ex *exchange) bool {
		return ex.machine == machine && ex.program == program
	})
}

func (s *SignIns) hold(ex *exchange) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	id := "ex-" + strconv.Itoa(s.counter)
	s.exchanges[id] = ex
	return id
}

func (s *SignIns) drop(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.exchanges, id)
}

func (s *SignIns) keep(id, link string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ex := s.exchanges[id]; ex != nil {
		ex.link = link
	}
}

func (s *SignIns) linkOf(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ex := s.exchanges[id]; ex != nil {
		return ex.link
	}
	return ""
}

func (s *SignIns) find(machine, id string) (exchange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ex := s.exchanges[id]
	if ex == nil || ex.machine != machine {
		return exchange{}, accounts.Refusal(fail.NotFound, accounts.LoginNoExchange, id)
	}
	return *ex, nil
}

func (s *SignIns) CheckSignOut(ctx context.Context, machine string, program accounts.Program) error {
	where, ok := s.m.Where(ctx, machine)
	switch {
	case !ok:
		return accounts.Refusal(fail.NotFound, accounts.LoginNotSignedIn, machine)
	case where.FarMac != "":
		return accounts.Refusal(fail.Refused, accounts.LoginOtherMac, where.FarMac)
	case where.Stopped:
		return accounts.Refusal(fail.Refused, accounts.LoginMachineStopped, string(program))
	}
	if facts := s.detect(ctx, machine, program); facts.State != accounts.ProgramSignedIn && facts.State != accounts.ProgramBroken {
		return accounts.Refusal(fail.Refused, accounts.LoginNotSignedIn, string(program))
	}
	return nil
}

func (s *SignIns) SignOut(ctx context.Context, machine string, program accounts.Program) (accounts.ProgramRow, error) {
	p := loginOf(program)
	ran, err := s.m.Exec(ctx, machine, line(p, p.Remove, ""), "")
	switch {
	case errors.Is(err, ErrStopped):
		return accounts.ProgramRow{}, accounts.Refusal(fail.Refused, accounts.LoginMachineStopped, string(program))
	case err != nil || ran.Code != 0:
		return accounts.ProgramRow{}, accounts.Refusal(fail.Failed, accounts.LoginFailed, string(program))
	}
	where, _ := s.m.Where(ctx, machine)
	return accounts.Describe(s.detect(ctx, machine, program), where), nil
}

func (s *SignIns) Open(ctx context.Context, machine, id string) (string, error) {
	// invariant: the page opened is the catalog's own for a device code, or
	// the link the program printed, already held to its site.
	if where, ok := s.m.Where(ctx, machine); ok && where.FarMac != "" {
		return "", accounts.Refusal(fail.Refused, accounts.LoginOtherMac, where.FarMac)
	}
	ex, err := s.find(machine, id)
	if err != nil {
		return "", err
	}
	if !ex.expires.After(s.m.Now()) {
		return "", accounts.Refusal(fail.Failed, accounts.LoginExpired, id)
	}
	p := loginOf(ex.program)
	page := ex.link
	if ex.kind == accounts.StepDeviceCode || page == "" {
		page = p.Page
	}
	if page == "" || !onSite(page, p.Site) {
		return "", accounts.Refusal(fail.Refused, accounts.LoginSiteRefused, p.Name)
	}
	if err := s.m.Open(ctx, page); err != nil {
		return "", accounts.Refusal(fail.Failed, accounts.LoginFailed, p.Name)
	}
	return p.Site, nil
}

func (s *SignIns) Answer(_ context.Context, machine, id string, code secret.Sent) error {
	// invariant: a code reaches the program on its input, one line, never
	// in a line it runs.
	ex, err := s.find(machine, id)
	if err != nil {
		return err
	}
	if ex.kind != accounts.StepPaste {
		return accounts.Refusal(fail.Refused, accounts.LoginNotAsking, id)
	}
	value := string(code.Reveal())
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n") {
		return accounts.Refusal(fail.Refused, accounts.LoginCodeRefused, id)
	}
	if _, err := ex.input.Write([]byte(value + "\n")); err != nil {
		return accounts.Refusal(fail.Refused, accounts.LoginCodeRefused, id)
	}
	return nil
}

func unknown(name string) verdict.Verdict {
	return verdict.Verdict{Tone: verdict.Bad, Sentence: "No machine named " + strconv.Quote(name)}
}
