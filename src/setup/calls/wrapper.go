package calls

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/src/setup"

	"github.com/jterrazz/studio-station/src/catalog"
)

const stepApps = "apps"

type Installer interface {
	Missing(ctx context.Context) ([]string, error)
	Install(ctx context.Context, names []string) error
	Superseded(ctx context.Context) []string
	RemoveFormula(ctx context.Context, formula string) error
}

func (s *Provisioner) apps(ctx context.Context, interactive bool) ([]setup.Step, error) {
	step, err := s.appsStep(ctx, interactive)
	var batch *setup.AppsFailedError
	if !errors.As(err, &batch) {
		return []setup.Step{step}, err
	}
	// invariant: every app of a failed batch gets its own row; an interrupt
	// is no failure of the step, so it travels as itself.
	rows := make([]setup.Step, 0, len(batch.Failed))
	for _, f := range batch.Failed {
		rows = append(rows, setup.Step{Name: f.Name, State: catalog.StateBroken, Detail: f.Detail})
	}
	if batch.Interrupted {
		return rows, batch
	}
	return rows, &setup.StepFailedError{Step: stepApps, Cause: batch}
}

func (s *Provisioner) appsStep(ctx context.Context, interactive bool) (setup.Step, error) {
	step := setup.Step{Name: stepApps, State: catalog.StateOK}
	var done []string
	if s.o.Installer != nil {
		missing, err := s.o.Installer.Missing(ctx)
		if err != nil {
			return failed(stepApps, err)
		}
		switch {
		case len(missing) == 0:
		case !interactive:
			step.State = catalog.StateAbsent
			done = append(done, "missing: "+strings.Join(missing, " ")+": run station setup on a terminal")
		default:
			if err := s.o.Installer.Install(ctx, missing); err != nil {
				return failed(stepApps, err)
			}
			done = append(done, "installed "+strings.Join(missing, " "))
		}
		for _, formula := range s.o.Installer.Superseded(ctx) {
			if err := s.o.Installer.RemoveFormula(ctx, formula); err != nil {
				return failed(stepApps, err)
			}
			done = append(done, formula+" formula uninstalled: the app replaces it")
		}
	}
	wrote, err := s.writeWrapper()
	if err != nil {
		return failed(stepApps, err)
	}
	if wrote {
		done = append(done, catalog.TailnetWrapperWritten(s.o.Expand.Display(s.wrapperPath())))
	}
	step.Detail = strings.Join(done, ", ")
	step.Changed = len(done) > 0
	return step, nil
}

func (s *Provisioner) appsCheck() setup.Step {
	if s.wrapperDue() {
		return setup.Step{Name: stepApps, State: catalog.StateDegraded, Detail: catalog.TailnetWrapperDue}
	}
	return setup.Step{Name: stepApps, State: catalog.StateOK}
}

func (s *Provisioner) wrapperPath() string { return path.Join(s.o.Bin, catalog.TailnetCLI) }

func wrapper(cli string) []byte { return []byte("#!/bin/sh\nexec " + cli + " \"$@\"\n") }

// invariant: the App Store app's CLI answers only when its bundle's executable
// runs directly (a symlink to it crashes), so PATH gets a two-line script, and
// only where nothing else answers: a symlink is replaced, the script's own
// bytes are adopted, another tailscale on PATH or a file station did not
// write is never touched.
func (s *Provisioner) wrapperDue() bool {
	if s.o.AppStoreCLI == nil {
		return false
	}
	cli := s.o.AppStoreCLI()
	if cli == "" {
		return false
	}
	at := s.wrapperPath()
	if s.m.IsLink(at) {
		return true
	}
	if s.m.Exists(at) {
		return false
	}
	if found, ok := s.m.LookPath(catalog.TailnetCLI); ok && found != at {
		return false
	}
	return true
}

func (s *Provisioner) writeWrapper() (bool, error) {
	if !s.wrapperDue() {
		return false, nil
	}
	at := s.wrapperPath()
	if _, err := s.m.Remove(at); err != nil {
		return false, err
	}
	if err := s.m.MkdirAll(s.o.Bin, 0o755); err != nil {
		return false, err
	}
	if err := s.m.WriteFile(at, wrapper(s.o.AppStoreCLI()), 0o755); err != nil {
		return false, err
	}
	return true, s.m.Chmod(at, 0o755)
}
