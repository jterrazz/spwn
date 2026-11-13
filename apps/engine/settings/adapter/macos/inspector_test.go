package macos_test

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/macos"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

type files struct{ m fstest.MapFS }

func (f files) Stat(p string) (fs.FileInfo, error) { return fs.Stat(f.m, p[1:]) }

type clock struct{ now time.Time }

func (c clock) Now() time.Time { return c.now }

// value: a Mac whose FileVault is on and whose firewall is off, as their
// tools print it; csrutil is not on its PATH.
func mac(c local.Cmd) (local.Output, error) {
	switch c.Name {
	case "fdesetup":
		return local.Output{Stdout: "FileVault is On.\n"}, nil
	case "socketfilterfw":
		return local.Output{Stdout: "Firewall is disabled. (State = 0)\n"}, nil
	}
	return local.Output{ExitCode: 97}, nil
}

func TestInspector(t *testing.T) {
	contracttest.Inspector(t, func(*testing.T) contracttest.InspectorSubject {
		i := macos.New(local.Played(mac, "fdesetup", "socketfilterfw"), files{fstest.MapFS{}}, clock{}, macos.Options{})
		return contracttest.InspectorSubject{Inspector: i, On: catalog.FactFileVault, Off: catalog.FactFirewall, Unreadable: catalog.FactSIP, Present: catalog.FactFileVault}
	})
}

func TestFileVaultsPlannedRestartIsReadFromFdesetup(t *testing.T) {
	for printed, want := range map[string]catalog.State{"true\n": catalog.StateOK, "false\n": catalog.StateAbsent} {
		run := func(c local.Cmd) (local.Output, error) {
			if c.Name == "fdesetup" && len(c.Args) == 1 && c.Args[0] == "supportsauthrestart" {
				return local.Output{Stdout: printed}, nil
			}
			return local.Output{ExitCode: 97}, nil
		}
		i := macos.New(local.Played(run, "fdesetup"), files{fstest.MapFS{}}, clock{}, macos.Options{})
		got, err := i.Read(t.Context(), catalog.FactAuthRestart)
		if err != nil || got.State != want {
			t.Errorf("fdesetup supportsauthrestart printing %q reads %+v, %v; want %s", printed, got, err, want)
		}
	}
}
