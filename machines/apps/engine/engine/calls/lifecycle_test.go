package calls

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type startControl struct {
	reading   engine.Reading
	installed *bool
}

func (c startControl) Check() error {
	if !*c.installed {
		return &NotInstalledError{}
	}
	return nil
}
func (c startControl) Read(context.Context) (engine.Reading, error) { return c.reading, nil }
func (startControl) Plan(context.Context) (engine.Work, error)      { return engine.Work{}, nil }
func (startControl) Log(context.Context) (engine.Log, error)        { return engine.Log{}, nil }
func (startControl) Stop(context.Context, engine.Work, string) (engine.StopReport, error) {
	return engine.StopReport{}, nil
}
func (startControl) Start(context.Context) (engine.StartReport, error) {
	return engine.StartReport{}, nil
}

type startInstaller struct {
	installed *bool
	ran       int
}

func (i *startInstaller) Install(context.Context) (engine.Installed, error) {
	i.ran++
	*i.installed = true
	return engine.Installed{Written: true}, nil
}
func (*startInstaller) Uninstall(context.Context) error { return nil }

func TestStartMovesTheJobsAtAnEqualVersion(t *testing.T) {
	installed := false
	installer := &startInstaller{installed: &installed}
	d := Deps{
		Control:   startControl{reading: engine.Reading{State: machines.EngineRunning, Version: "1.2.3", Station: "1.2.3"}, installed: &installed},
		Installer: installer,
	}
	report, err := Start.Run(context.Background(), d, StartIn{}, nil)
	if err != nil {
		t.Fatalf("start at an equal version with the bare jobs running: %v", err)
	}
	if installer.ran != 1 || !report.Installed.Written {
		t.Fatalf("the jobs were not moved: installs %d, installed %+v", installer.ran, report.Installed)
	}
}
