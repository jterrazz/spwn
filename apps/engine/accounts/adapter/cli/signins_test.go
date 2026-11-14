package cli_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/cli"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

// value: a machine whose programs answer as scripted: each detect by its
// program's name, each login by the lines it prints, then waits for its
// input or its end.
type scripted struct {
	detect map[string]cli.Ran
	prints string
	mu     sync.Mutex
	lines  []string
	input  strings.Builder
	opened []string
	hold   chan struct{}
}

func (s *scripted) named(line string) string {
	for _, p := range catalog.LoginPrograms {
		if strings.HasSuffix(line, " '"+p.Name+"'") {
			return p.Name
		}
	}
	return ""
}

func (s *scripted) Exec(_ context.Context, _, line, _ string) (cli.Ran, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = append(s.lines, line)
	if ran, ok := s.detect[s.named(line)]; ok {
		return ran, nil
	}
	return cli.Ran{Code: 127}, nil
}

type session struct {
	io.Reader
	s    *scripted
	ctx  context.Context
	done chan struct{}
}

func (r session) Write(p []byte) (int, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.input.Write(p)
	return len(p), nil
}

func (r session) Wait() (int, error) {
	select {
	case <-r.ctx.Done():
		return 143, nil
	case <-r.done:
		return 0, nil
	}
}

func (s *scripted) Attach(ctx context.Context, _, line string) (cli.Session, error) {
	s.mu.Lock()
	s.lines = append(s.lines, line)
	s.mu.Unlock()
	out, w := io.Pipe()
	go func() {
		_, _ = w.Write([]byte(s.prints))
		if s.hold != nil {
			<-s.hold
		}
		<-ctx.Done()
		_ = w.Close()
	}()
	return session{Reader: out, s: s, ctx: ctx, done: make(chan struct{})}, nil
}

func (s *scripted) Where(context.Context, string) (accounts.Where, bool) {
	return accounts.Where{Kind: machine.Computer}, true
}

func (s *scripted) Open(_ context.Context, page string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = append(s.opened, page)
	return nil
}

func (s *scripted) Now() time.Time { return time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC) }

func flowed(f catalog.Flow) catalog.LoginProgram {
	for _, p := range catalog.LoginPrograms {
		if p.Flow == f {
			return p
		}
	}
	return catalog.LoginProgram{}
}

func machineOf(prints string) *scripted {
	code, link, paste := flowed(catalog.FlowDeviceCode), flowed(catalog.FlowLink), flowed(catalog.FlowPaste)
	return &scripted{detect: map[string]cli.Ran{
		code.Name:  {Code: 1, Out: "You are not logged into any hosts"},
		link.Name:  {Code: 0, Out: "jterrazz\n"},
		paste.Name: {Code: 0, Out: `{"loggedIn":false}`},
	}, prints: prints}
}

const deviceCode = "! First copy your one-time code: 0A1B-2C3D\n"

func TestSignIns(t *testing.T) {
	contracttest.SignIns(t, func(*testing.T) contracttest.SignInsSubject {
		return contracttest.SignInsSubject{
			SignIns:  cli.NewSignIns(machineOf(deviceCode)),
			SignedIn: accounts.Program(flowed(catalog.FlowLink).Name), SignedOut: accounts.Program(flowed(catalog.FlowDeviceCode).Name),
		}
	})
}

func firstOf(t *testing.T, s *cli.SignIns, p catalog.LoginProgram) (*accounts.Step, error) {
	t.Helper()
	updates, err := s.Login(t.Context(), "", accounts.Program(p.Name))
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	bound, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	select {
	case u := <-updates:
		return u.Read.Step, u.Err
	case <-bound.Done():
		t.Fatal("no update came")
	}
	return nil, nil
}

func refusal(err error) fail.Code {
	var f *fail.Error
	if errors.As(err, &f) {
		return f.Code
	}
	return ""
}

func TestALinkOffTheProgramsSiteIsRefusedAndNeverOpened(t *testing.T) {
	paste := flowed(catalog.FlowPaste)
	m := machineOf("Visit https://" + paste.Site + ".example.net/oauth?code=FAKE000 to sign in\n")
	s := cli.NewSignIns(m)
	if _, err := firstOf(t, s, paste); refusal(err) != accounts.LoginSiteRefused {
		t.Errorf("a link off %s = %v; want %s", paste.Site, err, accounts.LoginSiteRefused)
	}
}

func TestAPastedCodeReachesTheProgramOnItsInputNeverItsLine(t *testing.T) {
	paste := flowed(catalog.FlowPaste)
	m := machineOf("https://" + paste.Site + "/oauth/authorize?state=FAKE000\nPaste code here if prompted >\n")
	s := cli.NewSignIns(m)
	step, err := firstOf(t, s, paste)
	if err != nil || step == nil || step.Kind != accounts.StepPaste {
		t.Fatalf("step = %+v, %v; want a paste", step, err)
	}
	if err := s.Answer(t.Context(), "", step.Exchange, secret.Sent{Value: secret.Hold("", []byte("FAKE000#state"))}); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if err := s.Answer(t.Context(), "", step.Exchange, secret.Sent{Value: secret.Hold("", []byte("two\nlines"))}); refusal(err) != accounts.LoginCodeRefused {
		t.Errorf("a code of two lines = %v; want %s", err, accounts.LoginCodeRefused)
	}
	if _, err := s.Open(t.Context(), "", step.Exchange); err != nil {
		t.Fatalf("Open: %v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.input.String() != "FAKE000#state\n" {
		t.Errorf("the program read %q; want the code and its newline", m.input.String())
	}
	for _, line := range m.lines {
		if strings.Contains(line, "FAKE000#state") {
			t.Errorf("the code rode a line: %s", line)
		}
	}
	if len(m.opened) != 1 || !strings.HasPrefix(m.opened[0], "https://"+paste.Site+"/oauth/authorize") {
		t.Errorf("opened %v; want the program's own link", m.opened)
	}
}

func TestADeviceCodeOpensTheCatalogsPageNotALinkItPrinted(t *testing.T) {
	code := flowed(catalog.FlowDeviceCode)
	m := machineOf("Open this URL to continue in your web browser: https://" + code.Site + "/login/device?x=1\n" + deviceCode)
	s := cli.NewSignIns(m)
	step, err := firstOf(t, s, code)
	if err != nil || step == nil || step.Code != "0A1B-2C3D" {
		t.Fatalf("step = %+v, %v; want the printed code", step, err)
	}
	if _, err := s.Open(t.Context(), "", step.Exchange); err != nil {
		t.Fatal(err)
	}
	if len(m.opened) != 1 || m.opened[0] != code.Page {
		t.Errorf("opened %v; want %s", m.opened, code.Page)
	}
}

func TestEverySignInLineRunsItsProgramByExecWithItsErrorsJoined(t *testing.T) {
	m := machineOf(deviceCode)
	s := cli.NewSignIns(m)
	if _, err := firstOf(t, s, flowed(catalog.FlowDeviceCode)); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, line := range m.lines {
		if !strings.HasPrefix(line, "/bin/sh -c ") || !strings.Contains(line, `exec "$p" `) || !strings.Contains(line, " 2>&1") {
			t.Errorf("line %s does not exec its program with its errors joined", line)
		}
	}
}

func signedOut(m *scripted, p catalog.LoginProgram) *scripted {
	out := p.Reads.Out
	if p.Reads.JSON != "" {
		out = `{"` + p.Reads.JSON + `":false}`
	}
	m.detect[p.Name] = cli.Ran{Code: 1, Out: out}
	return m
}

func codeAfterLink() catalog.LoginProgram {
	for _, p := range catalog.LoginPrograms {
		if p.Flow == catalog.FlowDeviceCode && p.CodeAfter != "" {
			return p
		}
	}
	return catalog.LoginProgram{}
}

func TestTheLinkKeptIsTheFirstOnTheProgramsSiteNotTheFirstPrinted(t *testing.T) {
	link := flowed(catalog.FlowLink)
	root := strings.TrimPrefix(link.Site, "www.")
	m := signedOut(machineOf(strings.Join([]string{"notice Log in on https://registry." + root + "/", "notice details: https://other.example.net/doc-FAKE000", "Login at:", "https://" + link.Site + "/login?x=FAKE000", ""}, "\n")), link)
	s := cli.NewSignIns(m)
	step, err := firstOf(t, s, link)
	if err != nil || step == nil || step.Kind != accounts.StepLink {
		t.Fatalf("step = %+v, %v; want a link", step, err)
	}
	if _, err := s.Open(t.Context(), "", step.Exchange); err != nil {
		t.Fatal(err)
	}
	if want := "https://" + link.Site + "/login?x=FAKE000"; len(m.opened) != 1 || m.opened[0] != want {
		t.Errorf("opened %v; want %s", m.opened, want)
	}
}

func TestAProgramThatPrintsOnlyLinksOffItsSiteIsRefusedWhileItKeepsRunning(t *testing.T) {
	link := flowed(catalog.FlowLink)
	m := signedOut(machineOf(strings.Join([]string{"Log in on https://registry." + strings.TrimPrefix(link.Site, "www.") + "/", "Login at:", ""}, "\n")), link)
	m.hold = make(chan struct{})
	t.Cleanup(func() { close(m.hold) })
	s := cli.NewSignIns(m)
	if _, err := firstOf(t, s, link); refusal(err) != accounts.LoginSiteRefused {
		t.Errorf("links off %s = %v; want %s", link.Site, err, accounts.LoginSiteRefused)
	}
}

func TestACodePrintedAfterItsLinkIsNotLostToTheLink(t *testing.T) {
	p := codeAfterLink()
	m := signedOut(machineOf(strings.Join([]string{"To authorize, visit:", "", "  https://" + p.Site + "/device/verify", "", "and " + p.CodeAfter + ":", "", "  FAKE0000", "", "You have 5 minutes.", "Opening https://" + p.Site + "/device/verify?x=FAKE000", ""}, "\n")), p)
	s := cli.NewSignIns(m)
	step, err := firstOf(t, s, p)
	if err != nil || step == nil || step.Kind != accounts.StepDeviceCode || step.Code != "FAKE0000" {
		t.Fatalf("step = %+v, %v; want a device code FAKE0000", step, err)
	}
	if _, err := s.Open(t.Context(), "", step.Exchange); err != nil {
		t.Fatal(err)
	}
	if len(m.opened) != 1 || m.opened[0] != p.Page {
		t.Errorf("opened %v; want %s", m.opened, p.Page)
	}
}

func TestAPastePromptWithNoNewlineDoesNotHoldTheStep(t *testing.T) {
	paste := flowed(catalog.FlowPaste)
	m := machineOf("Opening the browser\nIf it did not open, visit: https://" + paste.Site + "/oauth/authorize?x=FAKE000\nPaste code here if prompted > ")
	s := cli.NewSignIns(m)
	if step, err := firstOf(t, s, paste); err != nil || step == nil || step.Kind != accounts.StepPaste {
		t.Fatalf("step = %+v, %v; want a paste", step, err)
	}
}

func TestALoginLineStandsInForTheBrowserWithoutAFile(t *testing.T) {
	m := machineOf(deviceCode)
	s := cli.NewSignIns(m)
	if _, err := firstOf(t, s, flowed(catalog.FlowDeviceCode)); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	login := m.lines[len(m.lines)-1]
	if !strings.Contains(login, "BROWSER=/usr/bin/true") || strings.Contains(login, "mktemp") || strings.Contains(login, "chmod") {
		t.Errorf("login line %s: want true as the browser and no file written", login)
	}
}
