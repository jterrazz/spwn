package gitaliases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

type Applier struct {
	files Files
	home  string
}

var _ contract.Applier = (*Applier)(nil)

func New(files Files, home string) *Applier { return &Applier{files: files, home: home} }

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectGitAliases }

func of(it catalog.Setting) (catalog.GitAliases, error) {
	g, ok := it.Effect.(catalog.GitAliases)
	if !ok || len(g.Aliases) == 0 {
		return catalog.GitAliases{}, errors.New(it.Name + " is not git aliases station writes")
	}
	return g, nil
}

// invariant: station's aliases live in one block between its two marker
// lines, so the owner's own [alias] entries elsewhere in the file are never
// read, changed nor removed.
func markers(it catalog.Setting) (start, end string) {
	return "# >>> station " + it.Name + " >>>", "# <<< station " + it.Name + " <<<"
}

func block(it catalog.Setting, g catalog.GitAliases) string {
	start, end := markers(it)
	var b strings.Builder
	b.WriteString(start + "\n[alias]\n")
	for _, a := range g.Aliases {
		b.WriteString("\t" + a.Name + " = " + quoted(a.Command) + "\n")
	}
	b.WriteString(end + "\n")
	return b.String()
}

// value: a value git reads back as written: quoted, its quotes and
// backslashes escaped, so `;` and `#` stay part of it.
func quoted(v string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v) + `"`
}

// value: the file split around station's block: what comes before it, the
// block itself, and what follows; found is false when no block is there.
func split(text string, it catalog.Setting) (before, inside, after string, found bool) {
	start, end := markers(it)
	i := strings.Index(text, start+"\n")
	if i < 0 {
		return text, "", "", false
	}
	j := strings.Index(text[i:], end)
	if j < 0 {
		return text[:i], text[i:], "", true
	}
	stop := i + j + len(end)
	if stop < len(text) && text[stop] == '\n' {
		stop++
	}
	return text[:i], text[i:stop], text[stop:], true
}

func (a *Applier) read(g catalog.GitAliases) (string, error) {
	data, err := a.files.ReadFile(expand(a.home, g.File))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(data), err
}

func (a *Applier) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	g, err := of(it)
	if err != nil {
		return settings.Observation{}, err
	}
	text, err := a.read(g)
	if err != nil {
		return settings.Observation{}, err
	}
	where := g.File + " [alias]"
	want := block(it, g)
	_, have, _, found := split(text, it)
	o := settings.Observation{Where: where, Diff: settings.UnifiedDiff(want, have, "station's "+where, where)}
	if !found {
		o.State = catalog.StateAbsent
		return o, nil
	}
	sum := sha256.Sum256([]byte(have))
	o.Digest = "sha256:" + hex.EncodeToString(sum[:])
	if have == want {
		o.State, o.Detail, o.Diff = catalog.StateOK, where, ""
		return o, nil
	}
	o.State, o.Detail = catalog.StateDegraded, "differs from station's: station settings apply "+it.Name
	return o, nil
}

func (a *Applier) Apply(_ context.Context, it catalog.Setting, _ map[string]string) (settings.Change, error) {
	g, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	text, err := a.read(g)
	if err != nil {
		return settings.Change{}, err
	}
	before, _, after, found := split(text, it)
	if !found && before != "" && !strings.HasSuffix(before, "\n") {
		before += "\n"
	}
	full := expand(a.home, g.File)
	if err := a.files.MkdirAll(path.Dir(full), 0o750); err != nil {
		return settings.Change{}, err
	}
	if err := a.files.WriteFile(full, []byte(before+block(it, g)+after), 0o644); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: g.File + " [alias]", State: catalog.StateOK, Detail: "written"}}}, nil
}

func (a *Applier) Undo(_ context.Context, it catalog.Setting) (settings.Change, error) {
	// invariant: Undo removes the block only while it is still exactly
	// station's: off never destroys a change of the owner's.
	g, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	text, err := a.read(g)
	if err != nil {
		return settings.Change{}, err
	}
	label := g.File + " [alias]"
	before, have, after, found := split(text, it)
	switch {
	case !found:
		return settings.Change{}, nil
	case have != block(it, g):
		return settings.Change{Steps: []settings.ChangeStep{{Label: label, State: catalog.StateOK, Detail: "yours, kept"}}}, nil
	}
	if err := a.files.WriteFile(expand(a.home, g.File), []byte(before+after), 0o644); err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: label, State: catalog.StateOK, Detail: "removed"}}}, nil
}

func expand(home, p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return path.Join(home, rest)
	}
	return p
}
