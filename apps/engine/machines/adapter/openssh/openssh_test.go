package openssh

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
)

const (
	home       = "/home/o"
	configFile = home + "/.ssh/config"
)

type files map[string][]byte

func (f files) ReadFile(p string) ([]byte, error) {
	data, ok := f[p]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

func (f files) WriteFile(p string, data []byte, _ fs.FileMode) error {
	f[p] = data
	return nil
}

func (files) MkdirAll(string, fs.FileMode) error { return nil }

func adapter(f files) contract.SSHConfig {
	return New(f, Options{
		Home: home, KnownHosts: home + "/.local/state/station/known_hosts", Executable: "/bin/station",
	})
}

func read(t *testing.T, f files) []byte {
	t.Helper()
	data, err := f.ReadFile(configFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func write(t *testing.T, f files, data string) {
	t.Helper()
	if err := f.WriteFile(configFile, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTheOpenSSHConfigKeepsTheContract(t *testing.T) {
	contracttest.SSHConfig(t, func(*testing.T) contracttest.SSHConfigSubject {
		disk := files{}
		return contracttest.SSHConfigSubject{
			Blocks:  adapter(disk),
			Managed: func(t *testing.T) []string { return managedNames(read(t, disk)) },
			Foreign: func(t *testing.T, name string) { write(t, disk, "Host "+name+"\n  HostName legacy\n") },
		}
	})
}

func TestTheOwnersOwnLinesComeBackByteForByteOnceStationsBlockGoes(t *testing.T) {
	disk := files{}
	mine := "Host work\n  HostName 10.0.0.9\n\nHost *\n  AddKeysToAgent yes\n"
	write(t, disk, mine)
	c := adapter(disk)

	if err := c.Put(t.Context(), []contract.MachineBlock{{Name: "mac-mini", User: "u"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Put(t.Context(), []contract.MachineBlock{{Name: "mac-mini", User: "agent"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(t.Context(), "mac-mini"); err != nil {
		t.Fatal(err)
	}

	if got := string(read(t, disk)); got != mine {
		t.Errorf("the owner's file came back as\n%s\nwant\n%s", got, mine)
	}
}
