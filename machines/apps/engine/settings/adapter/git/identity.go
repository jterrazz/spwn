package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
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
	// value: the key that holds the keys of the identity Machines wrote and
	// the digest of their values, `user.name,user.email:<sha256>`, so off
	// takes back only those keys, and only while they hold what it wrote.
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

// value: the keys of the identity Machines wrote that still hold what it
// wrote; none when no record is there or a hand changed one of them since.
func (i *Identity) ours(ctx context.Context, now []settings.KeyValue) []string {
	wrote, _ := local.Read(ctx, i.run, local.Cmd{Name: "git", Args: []string{"config", "--global", wroteKey}, Timeout: readBudget})
	list, sum, ok := strings.Cut(wrote, ":")
	if !ok || list == "" {
		return nil
	}
	keys := strings.Split(list, ",")
	held := make([]string, 0, len(keys))
	for _, k := range keys {
		at := slices.IndexFunc(now, func(kv settings.KeyValue) bool { return kv.Key == k })
		if at < 0 {
			return nil
		}
		held = append(held, now[at].Value)
	}
	if digestOf(held...) != sum {
		return nil
	}
	return keys
}

func (i *Identity) Apply(ctx context.Context, _ catalog.Setting, values map[string]string) (settings.Change, error) {
	// invariant: a name or an email git already holds that Machines did not
	// write is the person's: apply refuses to replace it, so off never has
	// to restore a value it cannot know.
	name, email := strings.TrimSpace(values["name"]), strings.TrimSpace(values["email"])
	for _, v := range [][2]string{{"name", name}, {"email", email}} {
		if err := checked(v[0], v[1]); err != nil {
			return settings.Change{}, err
		}
	}
	change := settings.Change{Before: i.values(ctx)}
	want := []string{name, email}
	if change.Before[0].Value == name && change.Before[1].Value == email {
		change.Steps = append(change.Steps, done("git", "already "+name+" <"+email+">"))
		change.After = change.Before
		return change, nil
	}
	ours := i.ours(ctx, change.Before)
	var keys, held []string
	for n, kv := range change.Before {
		mine := slices.Contains(ours, kv.Key)
		if kv.Value != "" && kv.Value != want[n] && !mine {
			return change, errors.New(kv.Key + " is already set for every repository, and not by Machines: unset it with `git config --global --unset " + kv.Key + "`, or drop the git-identity setting")
		}
		if mine || kv.Value != want[n] {
			keys, held = append(keys, kv.Key), append(held, want[n])
		}
	}
	for n, kv := range change.Before {
		if kv.Value == want[n] {
			continue
		}
		if _, err := local.Must(ctx, i.run, local.Command("git", "config", "--global", kv.Key, want[n])); err != nil {
			return change, err
		}
	}
	if _, err := local.Must(ctx, i.run, local.Command("git", "config", "--global", wroteKey, strings.Join(keys, ",")+":"+digestOf(held...))); err != nil {
		return change, err
	}
	change.Steps = append(change.Steps, done("git", "commits as "+name+" <"+email+">"))
	change.After = i.values(ctx)
	return change, nil
}

func (i *Identity) Undo(ctx context.Context, _ catalog.Setting) (settings.Change, error) {
	now := i.values(ctx)
	change := settings.Change{Before: now, After: now}
	keys := i.ours(ctx, now)
	if len(keys) == 0 {
		change.Steps = append(change.Steps, done("git", "yours, kept as it is"))
		return change, nil
	}
	for _, key := range append(keys, wroteKey) {
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

func digestOf(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	return hex.EncodeToString(sum[:])
}
