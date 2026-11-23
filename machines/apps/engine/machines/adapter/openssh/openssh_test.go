package openssh

import (
	"context"
	"errors"
	"io/fs"
	"strings"
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

func (f files) Remove(p string) error {
	if _, ok := f[p]; !ok {
		return fs.ErrNotExist
	}
	delete(f, p)
	return nil
}

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

func TestTheFileStationCreatedIsRemovedWithItsLastBlockNeverLeftEmpty(t *testing.T) {
	disk := files{}
	c := adapter(disk)

	if err := c.Put(t.Context(), []contract.MachineBlock{{Name: "vm-1", User: "u"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Remove(t.Context(), "vm-1"); err != nil {
		t.Fatal(err)
	}
	if data, ok := disk[configFile]; ok {
		t.Errorf("the config file stays after its last block went, with %q", data)
	}
	if err := c.Remove(t.Context(), "vm-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := disk[configFile]; ok {
		t.Error("removing a block from no file created one")
	}
}

func TestABlockOnAMacOffersTheDeviceKeyThroughStationsAgentThenTheOwnersKeyAndNothingElse(t *testing.T) {
	f := files{home + "/.ssh/id_ed25519": []byte("key")}
	c := New(f, Options{
		Home: home, KnownHosts: home + "/.local/state/station/known_hosts", Executable: "/bin/station",
		Agent: home + "/.local/state/station/agent.sock", DeviceKey: home + "/.local/state/station/device-key.pub",
	})
	if err := c.Put(context.Background(), []contract.MachineBlock{{Name: "mac-mini"}, {Name: "linux-1", Key: "~/.ssh/agent_key"}}); err != nil {
		t.Fatal(err)
	}
	got := string(read(t, f))
	for _, want := range []string{
		"Host mac-mini\n  HostName mac-mini\n  IdentityAgent ~/.local/state/station/agent.sock\n  IdentityFile ~/.local/state/station/device-key.pub\n  IdentityFile ~/.ssh/id_ed25519\n  IdentitiesOnly yes\n",
		"Host linux-1\n  HostName linux-1\n  IdentityAgent ~/.local/state/station/agent.sock\n  IdentityFile ~/.local/state/station/device-key.pub\n  IdentityFile ~/.ssh/agent_key\n  IdentitiesOnly yes\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the block lacks\n%s\nin\n%s", want, got)
		}
	}
}

func TestABlockOffTheMacNamesNoAgent(t *testing.T) {
	f := files{home + "/.ssh/id_ed25519": []byte("key")}
	if err := adapter(f).Put(context.Background(), []contract.MachineBlock{{Name: "mac-mini"}}); err != nil {
		t.Fatal(err)
	}
	got := string(read(t, f))
	if strings.Contains(got, "IdentityAgent") || strings.Contains(got, "device-key") || !strings.Contains(got, "  IdentityFile ~/.ssh/id_ed25519\n  IdentitiesOnly yes\n") {
		t.Errorf("block:\n%s", got)
	}
}

func TestAVMOfThisMacIsReachedThroughTheEnginesPathAsOneShellWord(t *testing.T) {
	for executable, want := range map[string]string{
		"/bin/station": "  ProxyCommand /bin/station machines pipe linux-1\n",
		home + "/Applications/Spwn.app/Contents/MacOS/Spwn Engine": `  ProxyCommand "` + home + `/Applications/Spwn.app/Contents/MacOS/Spwn Engine" machines pipe linux-1` + "\n",
	} {
		f := files{}
		c := New(f, Options{Home: home, KnownHosts: home + "/.local/state/station/known_hosts", Executable: executable})
		if err := c.Put(context.Background(), []contract.MachineBlock{{Name: "linux-1", Behind: "mac-mini", ThisMac: true}}); err != nil {
			t.Fatal(err)
		}
		if got := string(read(t, f)); !strings.Contains(got, want) {
			t.Errorf("with %s the block lacks\n%s\nin\n%s", executable, want, got)
		}
	}
}

func TestABlockNamesNoDefaultKeyThatIsNotOnThisMac(t *testing.T) {
	f := files{}
	c := New(f, Options{
		Home: home, KnownHosts: home + "/.local/state/station/known_hosts", Executable: "/bin/station",
		Agent: home + "/.local/state/station/agent.sock", DeviceKey: home + "/.local/state/station/device-key.pub",
	})
	if err := c.Put(context.Background(), []contract.MachineBlock{{Name: "linux-1", Behind: "mac-mini", ThisMac: true}, {Name: "linux-2", Key: "~/.ssh/agent_key"}}); err != nil {
		t.Fatal(err)
	}
	got := string(read(t, f))
	if strings.Contains(got, "id_ed25519") {
		t.Errorf("a block names a key this Mac lacks:\n%s", got)
	}
	for _, want := range []string{
		"Host linux-1\n  HostName linux-1\n  IdentityAgent ~/.local/state/station/agent.sock\n  IdentityFile ~/.local/state/station/device-key.pub\n  IdentitiesOnly yes\n",
		"Host linux-2\n  HostName linux-2\n  IdentityAgent ~/.local/state/station/agent.sock\n  IdentityFile ~/.local/state/station/device-key.pub\n  IdentityFile ~/.ssh/agent_key\n  IdentitiesOnly yes\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the block lacks\n%s\nin\n%s", want, got)
		}
	}
}

func TestABlockOffTheMacWithNoKeyNamesNoIdentityAtAll(t *testing.T) {
	f := files{}
	if err := adapter(f).Put(context.Background(), []contract.MachineBlock{{Name: "mac-mini"}}); err != nil {
		t.Fatal(err)
	}
	if got := string(read(t, f)); strings.Contains(got, "Identity") {
		t.Errorf("block:\n%s", got)
	}
}
