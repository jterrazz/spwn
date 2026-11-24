package calls

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/disk"

	"github.com/jterrazz/studio-station/apps/engine/setup"

	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const (
	stepName  = "name"
	stepZshrc = "zshrc"
	stepKind  = "kind"
)

type Registrar interface {
	ThisMac() (string, bool)
	Register(name string) (string, error)
}

type ProvisionerOptions struct {
	Kind catalog.MachineKind
	// value: this machine is a server: its own tooling keeps it, never station.
	Server bool
	// value: the preferences' file, station's state folder and the user's bin.
	PreferencesFile, State string
	Apps                   []catalog.App
	Expand                 disk.Expander
	Files                  map[catalog.AppFile]string
	Configurer             Configurer
	// value: the bytes under path as `station disk` measures them, -1 past
	// ctx; nil measures nothing.
	Size func(ctx context.Context, path string) int64
	// value: this machine's ~/.zshrc, every write of it one edit under the
	// lock each station process takes for it.
	Zshrc Zshrc
}

// value: edit gets the text of ~/.zshrc and answers the text to write.

type Zshrc interface {
	Edit(ctx context.Context, edit func(text string) (string, error)) error
}

type ProvisionerPrefs struct {
	Name setupcontract.NamePrefs
}

type Provisioner struct {
	m   Machine
	p   ProvisionerPrefs
	reg Registrar
	o   ProvisionerOptions
}

func NewProvisioner(m Machine, p ProvisionerPrefs, reg Registrar, o ProvisionerOptions) *Provisioner {
	return &Provisioner{m: m, p: p, reg: reg, o: o}
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
	if thisMac, ok := s.reg.ThisMac(); ok && thisMac != name {
		return &setup.NameMismatchError{Named: thisMac, Asked: name}
	}
	return nil
}

func (s *Provisioner) DefaultName(ctx context.Context) string {
	var local string
	if out, code, err := s.m.Output(ctx, "scutil", []string{"--get", "LocalHostName"}, 2*time.Second); err == nil && code == 0 {
		local = out
	}
	return setup.DefaultName(local, s.m.Hostname())
}

func (s *Provisioner) Run(ctx context.Context, r Request) (setup.Report, error) {
	if err := s.CheckName(r.Name); err != nil {
		return setup.Report{}, err
	}
	if s.o.Server {
		return setup.Report{Steps: []setup.Step{{Name: stepKind, State: catalog.StateOK, Detail: "server: its own tooling keeps it, Machines sets nothing up"}}}, nil
	}
	if !s.onMac() {
		zshrc, err := s.zshrc(ctx)
		relocate, relocateErr := s.relocate(ctx, r.Interactive)
		return setup.Report{Steps: []setup.Step{zshrc, relocate}}, errors.Join(err, relocateErr)
	}
	if _, named := s.reg.ThisMac(); r.Name == "" && (!r.Interactive || r.AskName == nil) && !named && !s.exists(s.settingsPath()) {
		return setup.Report{}, setup.ErrNeedsName
	}
	name, nameErr := s.name(r)
	if nameErr != nil && !isStepFailure(nameErr) {
		return setup.Report{}, nameErr
	}
	zshrc, zshrcErr := s.zshrc(ctx)
	relocate, relocateErr := s.relocate(ctx, r.Interactive)
	return setup.Report{Steps: []setup.Step{name, zshrc, relocate}}, errors.Join(nameErr, zshrcErr, relocateErr)
}

func isStepFailure(err error) bool {
	var failed *setup.StepFailedError
	return errors.As(err, &failed)
}

func (s *Provisioner) name(r Request) (setup.Step, error) {
	if _, err := s.p.Name.Get(); err != nil {
		return failed(stepName, err)
	}
	if thisMac, ok := s.reg.ThisMac(); ok {
		return setup.Step{Name: stepName, State: catalog.StateOK, Detail: thisMac}, nil
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
	note, err := s.reg.Register(name)
	if err != nil {
		return failed(stepName, err)
	}
	detail := name
	if note != "" {
		detail += "; " + note
	}
	return setup.Step{Name: stepName, State: catalog.StateOK, Detail: detail, Changed: true}, nil
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
	return setup.Report{Steps: steps}, nil
}
