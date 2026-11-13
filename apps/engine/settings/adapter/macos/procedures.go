package macos

import (
	"context"
	"io/fs"
	"path"
	"strings"
	"time"

	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
)

const readBudget = 5 * time.Second

type MacOptions struct {
	Home string
	// value: where the DNS profile is written, and what Time Machine skips.
	DNSProfile       string
	BackupExclusions []string
	Jobs             backgroundcontract.Jobs
	Identity         setupcontract.IdentityPrefs
}

type MacFiles interface {
	Stat(path string) (fs.FileInfo, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	Chmod(path string, perm fs.FileMode) error
	Chown(path string, uid, gid int) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

type Mac struct {
	run   local.Runner
	files MacFiles
	o     MacOptions
}

func NewMac(run local.Runner, files MacFiles, o MacOptions) *Mac {
	return &Mac{run: run, files: files, o: o}
}

// value: one procedure as the Applier of its kind: observe reads what is
// there with the values station sets, apply writes them, undo, when
// station can, takes them back.
type procedure struct {
	of      catalog.Procedure
	observe func(ctx context.Context, want []settings.KeyValue) settings.Observation
	apply   func(ctx context.Context, it catalog.Setting, values map[string]string) (settings.Change, error)
	undo    func(ctx context.Context, it catalog.Setting) (settings.Change, error)
}

func (p procedure) Kind() catalog.EffectKind { return p.of.Kind() }

func (p procedure) Observe(ctx context.Context, it catalog.Setting) (settings.Observation, error) {
	return p.observe(ctx, settings.Sets(it)), nil
}

func (p procedure) Apply(ctx context.Context, it catalog.Setting, values map[string]string) (settings.Change, error) {
	return p.apply(ctx, it, values)
}

func (p procedure) Undo(ctx context.Context, it catalog.Setting) (settings.Change, error) {
	if p.undo == nil {
		return settings.Change{}, contract.ErrNoUndo
	}
	return p.undo(ctx, it)
}

func (m *Mac) Appliers() []contract.Applier {
	return []contract.Applier{
		procedure{of: catalog.ProcDNS, observe: m.observeDNS, apply: bare(m.applyDNS)},
		procedure{of: catalog.ProcSpotlight, observe: m.observeSpotlight, apply: bare(m.applySpotlight)},
		procedure{of: catalog.ProcRestartAfterPowerCut, observe: m.observeRestart, apply: m.applyRestart, undo: m.undoRestart},
		procedure{of: catalog.ProcAutologin, observe: m.observeAutologin, apply: valued(m.applyAutologin), undo: kept(m.undoAutologin)},
		procedure{of: catalog.ProcLockAfterLogin, observe: m.observeLock, apply: bare(m.applyLock), undo: kept(m.undoLock)},
		procedure{of: catalog.ProcSshd, observe: m.observeRemoteLogin, apply: bare(m.applyRemoteLogin), undo: kept(m.undoRemoteLogin)},
		procedure{of: catalog.ProcBackupExclusions, observe: m.observeExclusions, apply: bare(m.applyExclusions), undo: kept(m.undoExclusions)},
		procedure{of: catalog.ProcSSHKey, observe: m.observeSSHKey, apply: bare(m.applySSHKey)},
	}
}

func bare(fn func(ctx context.Context) (settings.Change, error)) func(context.Context, catalog.Setting, map[string]string) (settings.Change, error) {
	return func(ctx context.Context, _ catalog.Setting, _ map[string]string) (settings.Change, error) {
		return fn(ctx)
	}
}

func valued(fn func(ctx context.Context, values map[string]string) (settings.Change, error)) func(context.Context, catalog.Setting, map[string]string) (settings.Change, error) {
	return func(ctx context.Context, _ catalog.Setting, values map[string]string) (settings.Change, error) {
		return fn(ctx, values)
	}
}

func kept(fn func(ctx context.Context) (settings.Change, error)) func(context.Context, catalog.Setting) (settings.Change, error) {
	return func(ctx context.Context, _ catalog.Setting) (settings.Change, error) { return fn(ctx) }
}

// value: an observation from what is there and what station sets: OK when
// every value is station's, missing when none is, partly there between.
func judged(before, after []settings.KeyValue, ok, partly string) settings.Observation {
	o := settings.Observation{Before: before, After: after}
	was := map[string]string{}
	for _, kv := range before {
		was[kv.Key] = kv.Value
	}
	matched := 0
	for _, kv := range after {
		if was[kv.Key] == kv.Value {
			matched++
		}
	}
	switch matched {
	case len(after):
		o.State, o.Detail = catalog.StateOK, ok
	case 0:
		o.State = catalog.StateAbsent
	default:
		o.State, o.Detail = catalog.StateDegraded, partly
	}
	o.Digest = digestOf(before)
	return o
}

func digestOf(values []settings.KeyValue) string {
	parts := make([]string, len(values))
	for i, kv := range values {
		parts[i] = kv.Key + "=" + kv.Value
	}
	return strings.Join(parts, "\n")
}

func (m *Mac) output(ctx context.Context, name string, args ...string) (string, bool) {
	return local.Read(ctx, m.run, local.Cmd{Name: name, Args: args, Timeout: readBudget})
}

func (m *Mac) must(ctx context.Context, name string, args ...string) (string, error) {
	return local.Must(ctx, m.run, local.Command(name, args...))
}

func (m *Mac) exists(p string) bool {
	_, err := m.files.Stat(p)
	return err == nil
}

func (m *Mac) write(p string, data []byte, perm fs.FileMode) error {
	if err := m.files.MkdirAll(path.Dir(p), 0o750); err != nil {
		return err
	}
	return m.files.WriteFile(p, data, perm)
}

func (m *Mac) home(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return path.Join(m.o.Home, rest)
	}
	return p
}

func done(label, detail string) settings.ChangeStep {
	return settings.ChangeStep{Label: label, State: catalog.StateOK, Detail: detail}
}

func next(label, detail string) settings.ChangeStep {
	return settings.ChangeStep{Label: label, State: catalog.StateAbsent, Detail: detail}
}

func presence(ok bool) string {
	if ok {
		return "present"
	}
	return "absent"
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
