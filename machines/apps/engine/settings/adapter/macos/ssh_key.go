package macos

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const (
	sshKey         = "~/.ssh/id_ed25519"
	sshConfig      = "~/.ssh/config"
	sshKeyStanza   = "\nHost *\n  AddKeysToAgent yes\n  UseKeychain yes\n  IdentityFile ~/.ssh/id_ed25519\n"
	keychainStanza = "the Keychain stanza"
)

func (m *Mac) observeSSHKey(_ context.Context, want []settings.KeyValue) settings.Observation {
	now := m.sshKeyValues()
	o := judged(now, want, sshKey, sshConfig+" has no Keychain stanza")
	if now[0].Value != "present" {
		o.State, o.Detail = catalog.StateAbsent, ""
	}
	return o
}

func (m *Mac) sshKeyValues() []settings.KeyValue {
	text, _ := m.files.ReadFile(m.home(sshConfig))
	stanza := "none"
	if strings.Contains(string(text), "AddKeysToAgent yes") {
		stanza = keychainStanza
	}
	return []settings.KeyValue{{Key: sshKey, Value: presence(m.exists(m.home(sshKey)))}, {Key: sshConfig, Value: stanza}}
}

func (m *Mac) applySSHKey(ctx context.Context) (settings.Change, error) {
	person, err := m.o.Identity.Get()
	if err != nil {
		return settings.Change{}, err
	}
	dir, key := m.home("~/.ssh"), m.home(sshKey)
	if fault := m.files.MkdirAll(dir, 0o700); fault != nil {
		return settings.Change{}, fault
	}
	change := settings.Change{Before: m.sshKeyValues()}
	if m.exists(key) {
		change.Steps = append(change.Steps, done("ssh key", "already exists"))
	} else {
		if fault := local.Attach(ctx, m.run, local.Command("ssh-keygen", "-t", "ed25519", "-C", person.Email(), "-f", key)); fault != nil {
			return change, fault
		}
		change.Steps = append(change.Steps, done("ssh key", "generated"))
	}
	file := m.home(sshConfig)
	existing, err := m.files.ReadFile(file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return change, err
	}
	if strings.Contains(string(existing), "AddKeysToAgent yes") {
		change.Steps = append(change.Steps, done(sshConfig, "already configured"))
	} else {
		if fault := m.files.WriteFile(file, append(existing, sshKeyStanza...), 0o600); fault != nil {
			return change, fault
		}
		change.Steps = append(change.Steps, done(sshConfig, "keychain stanza added"))
	}
	if fault := local.Attach(ctx, m.run, local.Command("ssh-add", "--apple-use-keychain", key)); fault != nil {
		return change, fault
	}
	change.Steps = append(change.Steps, done("ssh-add", "passphrase kept in the Keychain"))
	pub, err := m.files.ReadFile(key + ".pub")
	if err != nil {
		return change, err
	}
	change.After = []settings.KeyValue{{Key: "public key", Value: strings.TrimSpace(string(pub))}}
	change.Steps = append(change.Steps, next("next", "add the public key at "+catalog.GitHubSSHKeys))
	return change, nil
}
