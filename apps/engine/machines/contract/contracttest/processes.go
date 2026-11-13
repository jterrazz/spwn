package contracttest

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

type ProcessesSubject struct {
	// value: the processes under test and one process they know runs.
	Processes contract.Processes
	Running   machines.Process
}

func Processes(t *testing.T, open func(t *testing.T) ProcessesSubject) {
	t.Run("a process known to run runs", func(t *testing.T) {
		s := open(t)
		if !s.Processes.Runs(s.Running) {
			t.Errorf("%+v does not run; want it running", s.Running)
		}
	})

	t.Run("its pid with another start is another process, which does not run", func(t *testing.T) {
		s := open(t)
		other := s.Running
		other.Start += "1"
		if s.Processes.Runs(other) {
			t.Errorf("%+v runs; want a pid taken again to be another process", other)
		}
	})

	t.Run("a process with no pid and no start never runs", func(t *testing.T) {
		if s := open(t); s.Processes.Runs(machines.Process{}) {
			t.Error("the zero process runs")
		}
	})

	t.Run("the process this code runs in, where it reads, runs", func(t *testing.T) {
		s := open(t)
		self, err := s.Processes.Self()
		if err == nil && (self.Start == "" || !s.Processes.Runs(self)) {
			t.Errorf("Self = %+v; want a start read and the process running", self)
		}
	})
}
