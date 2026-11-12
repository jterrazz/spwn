package calls

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/disk"

	"github.com/jterrazz/studio-station/src/setup"

	accountscontract "github.com/jterrazz/studio-station/src/accounts/contract"
	"github.com/jterrazz/studio-station/src/core/store"
	machinescontract "github.com/jterrazz/studio-station/src/machines/contract"
	portscontract "github.com/jterrazz/studio-station/src/ports/contract"
	setupcontract "github.com/jterrazz/studio-station/src/setup/contract"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/settings"
)

const (
	stepName   = "name"
	stepZshrc  = "zshrc"
	stepLayout = "layout"
	stepKind   = "kind"
	stepZsh    = "zsh"

	stepPreferences = "preferences"
)

type Registrar interface {
	ThisMac() (string, bool)
	Register(name string) error
}

type ProvisionerOptions struct {
	Kind catalog.MachineKind
	// value: this machine is a server: its own tooling keeps it, never station.
	Server bool
	// value: the preferences' file (a v1 config.json converts only while it
	// is absent), station's state folder and the user's bin.
	PreferencesFile, State string
	Executable             string
	Detached               bool
	Apps                   []catalog.App
	Expand                 disk.Expander
	Files                  map[catalog.AppFile]string
	Configurer             Configurer
	// value: moves the preferences out of the file station v2 kept them in,
	// once; nil where there is none to move.
	MovePreferences func(ctx context.Context) (moved bool, err error)
	// value: this machine's ~/.zshrc, every write of it one edit under the
	// lock each station process takes for it.
	Zshrc Zshrc
}

// value: edit gets the text of ~/.zshrc and answers the text to write.

type Zshrc interface {
	Edit(ctx context.Context, edit func(text string) (string, error)) error
}

type ProvisionerPrefs struct {
	Name      setupcontract.NamePrefs
	Identity  setupcontract.IdentityPrefs
	Macs      machinescontract.MacsPrefs
	VMUser    machinescontract.VMUserPrefs
	ShareList accountscontract.ShareListPrefs
	Ports     portscontract.PortsPrefs
	// value: every store station keeps, each brought to its version by the
	// layout step.
	Stores []store.Migrator
}

type Provisioner struct {
	m     Machine
	p     ProvisionerPrefs
	vault Vault
	reg   Registrar
	o     ProvisionerOptions
}

type Vault interface {
	Forget(ctx context.Context, name string) error
}

func NewProvisioner(m Machine, p ProvisionerPrefs, v Vault, reg Registrar, o ProvisionerOptions) *Provisioner {
	return &Provisioner{m: m, p: p, vault: v, reg: reg, o: o}
}

type Request struct {
	Name        string
	Interactive bool
	AskName     func() (string, error)
}

func (s *Provisioner) onMac() bool { return s.o.Kind == catalog.Mac }

func (s *Provisioner) CheckName(name string) error {
	if name == "" {
		return nil
	}
	if !setup.ValidName(name) {
		return &setup.BadNameError{Name: name}
	}
	if !s.onMac() {
		return setup.ErrNameOnVM
	}
	if thisMac := s.currentThisMac(); thisMac != "" && thisMac != name {
		return &setup.NameMismatchError{Named: thisMac, Asked: name}
	}
	return nil
}

// invariant: a Mac whose settings.json is not written yet but whose v1
// config.json names it is already registered: the layout step will say so.
func (s *Provisioner) currentThisMac() string {
	if thisMac, ok := s.reg.ThisMac(); ok {
		return thisMac
	}
	if s.m.Exists(s.settingsPath()) {
		return ""
	}
	data, found, err := s.m.ReadFile(s.v1Path())
	if err != nil || !found {
		return ""
	}
	converted, err := setup.ConvertV1(data, s.m.User())
	if err != nil {
		return ""
	}
	return converted.ThisMachine
}

func (s *Provisioner) DefaultName(ctx context.Context) string {
	raw := s.m.Hostname()
	if out, code, err := s.m.Output(ctx, "scutil", []string{"--get", "LocalHostName"}, 2*time.Second); err == nil && code == 0 && strings.TrimSpace(out) != "" {
		raw = strings.TrimSpace(out)
	}
	return sanitize(raw)
}

func sanitize(raw string) string {
	label, _, _ := strings.Cut(strings.ToLower(raw), ".")
	label = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, label)
	label = strings.Trim(label, "-")
	if len(label) > 40 {
		label = strings.TrimRight(label[:40], "-")
	}
	return label
}

func (s *Provisioner) Run(ctx context.Context, r Request) (setup.Report, error) {
	moved, err := s.movePreferences(ctx)
	if err != nil {
		return setup.Report{Steps: moved}, err
	}
	if err := s.CheckName(r.Name); err != nil {
		return setup.Report{}, err
	}
	if s.o.Server {
		return setup.Report{Steps: append(moved, setup.Step{Name: stepKind, State: catalog.StateOK, Detail: "server: its own tooling keeps it, station sets nothing up"})}, nil
	}
	if !s.onMac() {
		zshrc, err := s.zshrc(ctx)
		relocate, relocateErr := s.relocate(ctx, r.Interactive)
		return setup.Report{Steps: append(moved, zshrc, relocate)}, errors.Join(err, relocateErr)
	}
	if r.Name == "" && (!r.Interactive || r.AskName == nil) && s.currentThisMac() == "" && !s.exists(s.settingsPath()) {
		return setup.Report{Steps: moved}, setup.ErrNeedsName
	}
	migrated := s.migrateSettings()
	name, nameErr := s.name(r, migrated != nil)
	if nameErr != nil && !isStepFailure(nameErr) {
		return setup.Report{}, nameErr
	}
	zshrc, zshrcErr := s.zshrc(ctx)
	layout, layoutErr := s.layout(ctx, migrated)
	var zshErr error
	if layoutErr == nil && zshrcErr == nil {
		var retired []setup.Step
		retired, zshErr = s.retireZshDir(ctx)
		layout = append(layout, retired...)
	}
	relocate, relocateErr := s.relocate(ctx, r.Interactive)
	steps := append(append([]setup.Step{name, zshrc}, layout...), relocate)
	return setup.Report{Steps: append(moved, steps...)}, errors.Join(nameErr, zshrcErr, layoutErr, zshErr, relocateErr)
}

// invariant: the move comes before any step reads a preference, and says so
// once: a second run has nothing to move, and no row of it.
func (s *Provisioner) movePreferences(ctx context.Context) ([]setup.Step, error) {
	if s.o.MovePreferences == nil {
		return nil, nil
	}
	moved, err := s.o.MovePreferences(ctx)
	if err != nil {
		step, stepErr := failed(stepPreferences, err)
		return []setup.Step{step}, stepErr
	}
	if !moved {
		return nil, nil
	}
	return []setup.Step{{Name: stepPreferences, State: catalog.StateOK, Detail: "moved settings.json to preferences.json", Changed: true}}, nil
}

func isStepFailure(err error) bool {
	var failed *setup.StepFailedError
	return errors.As(err, &failed)
}

func (s *Provisioner) name(r Request, blocked bool) (setup.Step, error) {
	if thisMac, ok := s.reg.ThisMac(); ok {
		return setup.Step{Name: stepName, State: catalog.StateOK, Detail: thisMac}, nil
	}
	if blocked {
		return setup.Step{Name: stepName, State: catalog.StateAbsent, Detail: "waits for the layout step"}, nil
	}
	name := r.Name
	if name == "" && r.Interactive && r.AskName != nil {
		asked, err := r.AskName()
		if err != nil {
			return setup.Step{}, err
		}
		name = asked
	}
	if name == "" {
		return setup.Step{Name: stepName, State: catalog.StateAbsent, Detail: "not set: station init --name <name>"}, nil
	}
	if !setup.ValidName(name) {
		return setup.Step{}, &setup.BadNameError{Name: name}
	}
	if err := s.p.Name.Set(name); err != nil {
		return failed(stepName, err)
	}
	if err := s.reg.Register(name); err != nil {
		return failed(stepName, err)
	}
	return setup.Step{Name: stepName, State: catalog.StateOK, Detail: name, Changed: true}, nil
}

func failed(step string, cause error) (setup.Step, error) {
	return setup.Step{Name: step, State: catalog.StateBroken, Detail: cause.Error()}, &setup.StepFailedError{Step: step, Cause: cause}
}

func (s *Provisioner) zshrcPath() string { return path.Join(s.m.Home(), ".zshrc") }

func (s *Provisioner) zshrc(ctx context.Context) (setup.Step, error) {
	changed := false
	err := s.o.Zshrc.Edit(ctx, func(text string) (string, error) {
		updated, upserted := settings.UpsertBlock(text)
		changed = upserted
		return updated, nil
	})
	if err != nil {
		return failed(stepZshrc, err)
	}
	return setup.Step{Name: stepZshrc, State: catalog.StateOK, Changed: changed}, nil
}

func (s *Provisioner) Check(_ context.Context) (setup.Report, error) {
	var steps []setup.Step
	if s.onMac() {
		if thisMac, ok := s.reg.ThisMac(); ok {
			steps = append(steps, setup.Step{Name: stepName, State: catalog.StateOK, Detail: thisMac})
		} else {
			steps = append(steps, setup.Step{Name: stepName, State: catalog.StateAbsent, Detail: "station init --name <name>"})
		}
	}
	zshrc := setup.Step{Name: stepZshrc, State: catalog.StateOK}
	if data, found, err := s.m.ReadFile(s.zshrcPath()); err != nil || !found || !settings.HasBlock(string(data)) {
		zshrc = setup.Step{Name: stepZshrc, State: catalog.StateAbsent, Detail: "station init"}
	}
	steps = append(steps, zshrc)
	if s.onMac() {
		layout := setup.Step{Name: stepLayout, State: catalog.StateOK}
		if s.holdsStationFiles() {
			layout = setup.Step{Name: stepLayout, State: catalog.StateDegraded, Detail: "~/.studio still holds station's files: station init"}
		}
		steps = append(steps, layout)
	}
	return setup.Report{Steps: steps}, nil
}
