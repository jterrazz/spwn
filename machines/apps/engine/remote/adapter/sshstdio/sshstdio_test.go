package sshstdio_test

import (
	"context"
	"errors"
	"io"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/sshstdio"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

// value: ssh as a pipe to the far Mac's `station engine attach`, its server
// running in this process; a Mac that is down ends ssh with 255.
type ssh struct {
	far  serve.Calls
	down bool
	// value: the far Mac answers ssh and has no station: its shell ends 127.
	bare bool
	ran  [][]string
}

type process struct {
	in   *io.PipeWriter
	out  *io.PipeReader
	done chan struct{}
	code int
}

func (p *process) Stdin() io.WriteCloser { return p.in }
func (p *process) Stdout() io.Reader     { return p.out }
func (p *process) Kill()                 { _ = p.in.Close() }
func (p *process) Wait() (code int, stderr string, err error) {
	<-p.done
	switch p.code {
	case 0:
	case 127:
		return p.code, "sh: station: not found", nil
	default:
		return p.code, "ssh: connect to host mac-mini port 22: Host is down", nil
	}
	return 0, "", nil
}

func (s *ssh) Spawn(ctx context.Context, c local.Cmd) (local.Process, error) {
	s.ran = append(s.ran, append([]string{c.Name}, c.Args...))
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	p := &process{in: inW, out: outR, done: make(chan struct{})}
	go func() {
		defer close(p.done)
		if s.down || s.bare {
			p.code = 255
			if s.bare {
				p.code = 127
			}
			_ = inR.Close()
			_ = outW.Close()
			return
		}
		_ = serve.Stream(context.WithoutCancel(ctx), inR, outW, s.far, func(err error) *fail.Error {
			var e *fail.Error
			if errors.As(err, &e) {
				return e
			}
			return &fail.Error{Message: err.Error(), Exit: fail.ExitFailure}
		})
		_ = outW.Close()
	}()
	return p, nil
}

func TestSSHToTheFarEngineKeepsTheEndpointContract(t *testing.T) {
	contracttest.Endpoint(t, func(_ *testing.T, far *call.Registry, profile machine.Profile, version string) contracttest.EndpointSubject {
		s := &ssh{far: serve.Local(far, profile, version)}
		return contracttest.EndpointSubject{Dialer: sshstdio.New(s, version), Mac: profile.Ref.Name, Target: profile.Ref, Down: func() { s.down = true }}
	})
}

func TestAnotherMacIsReachedOnlyThroughItsEngineInBatchMode(t *testing.T) {
	s := &ssh{far: serve.Local(contracttest.Far(), contracttest.Mini, contracttest.Version)}
	ep, err := sshstdio.New(s, contracttest.Version).Dial(t.Context(), "mac-mini")
	if err != nil {
		t.Fatal(err)
	}
	_ = ep.Close()
	want := []string{"ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-o", "ServerAliveInterval=2", "-o", "ServerAliveCountMax=2", "--", "mac-mini", "sh", "-c"}
	if len(s.ran) != 1 || !slices.Equal(s.ran[0][:len(want)], want) || !strings.HasSuffix(strings.Join(s.ran[0], " "), " station engine attach") {
		t.Errorf("dialing mac-mini ran %q, want one BatchMode ssh with keepalives to its `station engine attach`", s.ran)
	}
}

func TestAMacThatAnswersWithoutStationIsToldApartFromOneThatDoesNotAnswer(t *testing.T) {
	s := &ssh{far: serve.Local(contracttest.Far(), contracttest.Mini, contracttest.Version), bare: true}
	ep, err := sshstdio.New(s, contracttest.Version).Dial(t.Context(), "mac-mini")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ep.Close() }()
	_, err = ep.Hello(t.Context())
	var e *fail.Error
	if !errors.As(err, &e) || e.Code != call.NoStation || e.Subject != "mac-mini" {
		t.Fatalf("Hello = %v, want mac-mini's missing station told as %s", err, call.NoStation)
	}
}

func TestAMacNameSSHWouldReadAsAnOptionNeverReachesSSH(t *testing.T) {
	for _, mac := range []string{"-oProxyCommand=/usr/bin/touch FAKE000", "-oProxyCommand=FAKE000", "mac-mini;touch FAKE000", ""} {
		s := &ssh{far: serve.Local(contracttest.Far(), contracttest.Mini, contracttest.Version)}
		ep, err := sshstdio.New(s, contracttest.Version).Dial(t.Context(), mac)
		if err == nil {
			_ = ep.Close()
		}
		var e *fail.Error
		if !errors.As(err, &e) || e.Code != call.BadMacName || e.Subject != mac || len(s.ran) != 0 {
			t.Errorf("dialing %q = %v and ran %q, want it refused as %s before any ssh", mac, err, s.ran, call.BadMacName)
		}
	}
}

func TestTheRealSSHGivesTheFarMacExactlyItsEngineAttach(t *testing.T) {
	argv, err := sshstdio.Argv("mac-mini")
	if err != nil {
		t.Fatal(err)
	}
	far := farSSH(t, argv)
	want := "sh -c " + catalog.FarStationLine() + " station engine attach"
	if far.Dialed != "mac-mini" || far.Command != want {
		t.Errorf("ssh %q dialed %q and gave the far side %q, want mac-mini given %q", argv, far.Dialed, far.Command, want)
	}
}

func TestTheFarLineIsNoWordSSHReadsAsAnOption(t *testing.T) {
	if line := catalog.FarStationLine(); line == "" || strings.HasPrefix(line, "-") {
		t.Errorf("the far station line %q starts with \"-\"", line)
	}
}

type farSide struct {
	Dialed, Command string
}

func farSSH(t *testing.T, args []string) farSide {
	t.Helper()
	dir := t.TempDir()
	sys := local.New([]string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin", "HOME=" + dir}, strings.NewReader(""), io.Discard, io.Discard)
	sshd, server := sys.Run.LookPath("sshd")
	_, client := sys.Run.LookPath("ssh")
	_, keygen := sys.Run.LookPath("ssh-keygen")
	if !server || !client || !keygen {
		t.Skip("no OpenSSH client, server and ssh-keygen on this machine")
	}
	ctx := context.Background()
	for _, k := range []string{"host", "client"} {
		out, err := sys.Run.Output(ctx, local.Cmd{Name: "ssh-keygen", Args: []string{"-q", "-t", "ed25519", "-N", "", "-f", path.Join(dir, k)}})
		if err != nil || out.ExitCode != 0 {
			t.Fatalf("ssh-keygen: %v %s", err, out.Stderr)
		}
	}
	pub, err := sys.FS.ReadFile(path.Join(dir, "client.pub"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := strings.Join([]string{
		"HostKey " + path.Join(dir, "host"),
		"AuthorizedKeysFile " + path.Join(dir, "authorized"),
		"StrictModes no",
		"UsePAM no",
		"PidFile none",
		"LogLevel ERROR",
		`ForceCommand printf %s "$SSH_ORIGINAL_COMMAND" > ` + quoted(path.Join(dir, "command")),
		"",
	}, "\n")
	for name, data := range map[string]string{"authorized": string(pub), "sshd_config": cfg} {
		if werr := sys.FS.WriteFile(path.Join(dir, name), []byte(data), 0o600); werr != nil {
			t.Fatal(werr)
		}
	}
	proxy := "ProxyCommand=/bin/sh -c 'printf %%s \"$1\" > " + quoted(path.Join(dir, "dialed")) + "; exec " + sshd + " -i -f " + quoted(path.Join(dir, "sshd_config")) + "' sh %n"
	own := []string{"-F", "/dev/null", "-i", path.Join(dir, "client"),
		"-o", "IdentitiesOnly=yes", "-o", "PubkeyAuthentication=yes", "-o", "PreferredAuthentications=publickey",
		"-o", "UserKnownHostsFile=/dev/null", "-o", "GlobalKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes", "-o", "LogLevel=ERROR", "-o", proxy}
	out, err := sys.Run.Output(ctx, local.Cmd{Name: "ssh", Args: append(own, args...), Env: []string{"SHELL=/bin/sh"}})
	if err != nil {
		t.Fatalf("ssh %q: %v", args, err)
	}
	dialed, _ := sys.FS.ReadFile(path.Join(dir, "dialed"))
	command, err := sys.FS.ReadFile(path.Join(dir, "command"))
	if err != nil {
		t.Fatalf("ssh %q exited %d before the far side ran its command: %s", args, out.ExitCode, out.Stderr)
	}
	return farSide{Dialed: string(dialed), Command: string(command)}
}

func quoted(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
