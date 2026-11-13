package git

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
)

const readBudget = 5 * time.Second

type Read func(ctx context.Context, c local.Cmd) (local.Output, error)

type Signing struct {
	run      local.Runner
	identity setupcontract.IdentityPrefs
}

var _ contract.Applier = (*Signing)(nil)

func NewSigning(run local.Runner, identity setupcontract.IdentityPrefs) *Signing {
	return &Signing{run: run, identity: identity}
}

func (*Signing) Kind() catalog.EffectKind { return catalog.ProcGitSigning.Kind() }

func (s *Signing) Observe(ctx context.Context, it catalog.Setting) (settings.Observation, error) {
	now := s.values(ctx)
	o := settings.Observation{State: catalog.StateAbsent, Before: now, After: settings.Sets(it)}
	if now[0].Value == "true" {
		o.State, o.Detail = catalog.StateOK, "commit.gpgsign=true"
	}
	return o, nil
}

func (s *Signing) values(ctx context.Context) []settings.KeyValue {
	out := make([]settings.KeyValue, 0, 2)
	for _, key := range []string{"commit.gpgsign", "gpg.program"} {
		v, ok := local.Read(ctx, s.run, local.Cmd{Name: "git", Args: []string{"config", "--global", key}, Timeout: readBudget})
		if !ok {
			v = ""
		}
		out = append(out, settings.KeyValue{Key: key, Value: v})
	}
	return out
}

func (s *Signing) Apply(ctx context.Context, _ catalog.Setting, _ map[string]string) (settings.Change, error) {
	person, err := s.identity.Get()
	if err != nil {
		return settings.Change{}, err
	}
	email := person.Email()
	if _, ok := s.run.LookPath("gpg"); !ok {
		return settings.Change{}, errors.New("gpg is not installed: station apps install gpg")
	}
	change := settings.Change{Before: s.values(ctx)}
	if out, ok := local.Read(ctx, s.run, local.Cmd{Name: "gpg", Args: []string{"--list-secret-keys", "--keyid-format", "long", email}, Timeout: readBudget}); ok && out != "" {
		change.Steps = append(change.Steps, done("gpg key", "already exists for "+email))
	} else {
		batch := local.Command("gpg", "--batch", "--generate-key")
		batch.Stdin = "%no-protection\nKey-Type: eddsa\nKey-Curve: ed25519\nName-Real: " + person.Name() + "\nName-Email: " + email + "\nExpire-Date: 0\n%commit\n"
		if fault := local.Attach(ctx, s.run, batch); fault != nil {
			return change, fault
		}
		change.Steps = append(change.Steps, done("gpg key", "generated for "+email+" (ed25519)"))
	}
	listing, err := local.Must(ctx, s.run, local.Command("gpg", "--list-secret-keys", "--keyid-format", "long", email))
	if err != nil {
		return change, err
	}
	keyID := signingKeyID(listing)
	if keyID == "" {
		return change, errors.New("could not find the gpg key id for " + email)
	}
	for _, kv := range [][2]string{{"user.signingkey", keyID}, {"commit.gpgsign", "true"}, {"gpg.program", "gpg"}} {
		if _, err := local.Must(ctx, s.run, local.Command("git", "config", "--global", kv[0], kv[1])); err != nil {
			return change, err
		}
	}
	change.Steps = append(change.Steps, done("git", "signs every commit with "+keyID))
	if err := local.Attach(ctx, s.run, local.Command("gpg", "--armor", "--export", email)); err != nil {
		return change, err
	}
	change.Steps = append(change.Steps, settings.ChangeStep{Label: "public key", State: catalog.StateAbsent, Detail: "printed above: add it at " + catalog.GitHubGPGKeys})
	change.After = s.values(ctx)
	return change, nil
}

func (*Signing) Undo(context.Context, catalog.Setting) (settings.Change, error) {
	return settings.Change{}, contract.ErrNoUndo
}

func signingKeyID(listing string) string {
	for _, line := range strings.Split(listing, "\n") {
		if !strings.Contains(line, "ed25519/") && !strings.Contains(line, "rsa") {
			continue
		}
		if _, after, ok := strings.Cut(line, "/"); ok {
			if fields := strings.Fields(after); len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

type Login struct {
	run  local.Runner
	read Read
}

var _ contract.Applier = (*Login)(nil)

func NewLogin(run local.Runner, read Read) *Login {
	if read == nil {
		read = run.Output
	}
	return &Login{run: run, read: read}
}

func (*Login) Kind() catalog.EffectKind { return catalog.ProcGitHubLogin.Kind() }

func (l *Login) Observe(ctx context.Context, it catalog.Setting) (settings.Observation, error) {
	o := settings.Observation{State: catalog.StateAbsent, Before: []settings.KeyValue{{Key: "gh", Value: "logged out"}}, After: settings.Sets(it)}
	if out, err := l.read(ctx, local.Cmd{Name: "gh", Args: []string{"auth", "status"}, Timeout: readBudget}); err == nil && out.ExitCode == 0 {
		o.State, o.Detail, o.Before = catalog.StateOK, "authenticated", o.After
	}
	return o, nil
}

func (l *Login) Apply(ctx context.Context, _ catalog.Setting, _ map[string]string) (settings.Change, error) {
	if err := local.Attach(ctx, l.run, local.Command("gh", "auth", "login", "--hostname", catalog.GitHubDomain, "--git-protocol=ssh", "--web", "--skip-ssh-key")); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{done("gh", "logged in to "+catalog.GitHubDomain+" over ssh")}}, nil
}

func (*Login) Undo(context.Context, catalog.Setting) (settings.Change, error) {
	return settings.Change{}, contract.ErrNoUndo
}

func done(label, detail string) settings.ChangeStep {
	return settings.ChangeStep{Label: label, State: catalog.StateOK, Detail: detail}
}
