package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"unicode"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

const (
	nameKey  = "user.name"
	emailKey = "user.email"
	// value: the key that holds the digest of the identity Machines wrote, so
	// off takes back only a value still exactly as Machines left it.
	wroteKey = "station.identity"
)

type Identity struct{ run local.Runner }

var _ contract.Applier = (*Identity)(nil)

func NewIdentity(run local.Runner) *Identity { return &Identity{run: run} }

func (*Identity) Kind() catalog.EffectKind { return catalog.ProcGitIdentity.Kind() }

func (i *Identity) Observe(ctx context.Context, it catalog.Setting) (settings.Observation, error) {
	now := i.values(ctx)
	o := settings.Observation{State: catalog.StateAbsent, Before: now, After: settings.Sets(it), Digest: digestOf(now[0].Value, now[1].Value)}
	if now[0].Value != "" && now[1].Value != "" {
		o.State, o.Detail = catalog.StateOK, now[0].Value+" <"+now[1].Value+">"
	}
	return o, nil
}

func (i *Identity) values(ctx context.Context) []settings.KeyValue {
	out := make([]settings.KeyValue, 0, 2)
	for _, key := range []string{nameKey, emailKey} {
		v, _ := local.Read(ctx, i.run, local.Cmd{Name: "git", Args: []string{"config", "--global", key}, Timeout: readBudget})
		out = append(out, settings.KeyValue{Key: key, Value: v})
	}
	return out
}

func (i *Identity) Apply(ctx context.Context, _ catalog.Setting, values map[string]string) (settings.Change, error) {
	name, email := strings.TrimSpace(values["name"]), strings.TrimSpace(values["email"])
	for _, v := range [][2]string{{"name", name}, {"email", email}} {
		if err := checked(v[0], v[1]); err != nil {
			return settings.Change{}, err
		}
	}
	change := settings.Change{Before: i.values(ctx)}
	if change.Before[0].Value == name && change.Before[1].Value == email {
		change.Steps = append(change.Steps, done("git", "already "+name+" <"+email+">"))
		change.After = change.Before
		return change, nil
	}
	for _, kv := range [][2]string{{nameKey, name}, {emailKey, email}, {wroteKey, digestOf(name, email)}} {
		if _, err := local.Must(ctx, i.run, local.Command("git", "config", "--global", kv[0], kv[1])); err != nil {
			return change, err
		}
	}
	change.Steps = append(change.Steps, done("git", "commits as "+name+" <"+email+">"))
	change.After = i.values(ctx)
	return change, nil
}

func (i *Identity) Undo(ctx context.Context, _ catalog.Setting) (settings.Change, error) {
	now := i.values(ctx)
	change := settings.Change{Before: now, After: now}
	wrote, _ := local.Read(ctx, i.run, local.Cmd{Name: "git", Args: []string{"config", "--global", wroteKey}, Timeout: readBudget})
	if wrote == "" || wrote != digestOf(now[0].Value, now[1].Value) {
		change.Steps = append(change.Steps, done("git", "yours, kept as it is"))
		return change, nil
	}
	for _, key := range []string{nameKey, emailKey, wroteKey} {
		if _, err := local.Must(ctx, i.run, local.Command("git", "config", "--global", "--unset", key)); err != nil {
			return change, err
		}
	}
	change.Steps = append(change.Steps, done("git", "identity removed"))
	change.After = i.values(ctx)
	return change, nil
}

func checked(label, value string) error {
	if value == "" {
		return errors.New(label + " is empty: git needs one")
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return errors.New(label + " holds a control character")
	}
	return nil
}

func digestOf(name, email string) string {
	sum := sha256.Sum256([]byte(name + "\x00" + email))
	return hex.EncodeToString(sum[:])
}
