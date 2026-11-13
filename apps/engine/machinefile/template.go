package machinefile

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: a Machinefile a VM image is made from: its name (the image's),
// the image its disk starts from, its system, the packages it bakes, what its
// vm says of how a VM made from it runs (the folders it mounts included)
// and the accounts it is shared
// (each term as written), the files of its chain a build VM needs (every
// one but station's built-in ones, which its binary embeds), the lock its
// build installs, and the hash of all of them, which a changed chain or
// lock changes.

type Template struct {
	Name     string
	From     string
	OS       catalog.OS
	Kind     catalog.MachineRole
	Bake     []string
	Lifetime catalog.Lifetime
	Network  string
	CPUs     int
	Memory   string
	Disk     string
	Mounts   []catalog.Mount
	Shares   []catalog.AccountShare
	// value: each file by its slash path under the owner's folder
	// (`machinefiles/<name>.machine.yaml`, `machinefiles/<name>.local.yaml`).
	Files []File
	Lock  []byte
	SHA   string
}

func TemplateOf(lib Library, name string, lock []byte) (Template, error) {
	// invariant: a template is a Machinefile of a VM (a workspace or a
	// server) that says the image its disk starts from; a mixin, a
	// computer's, or one that names no image is none.
	files, err := lib.chain(name)
	if err != nil {
		return Template{}, err
	}
	m := merge(files)
	kind := catalog.MachineRole(m.values["machine.kind"].Text)
	from := m.values["vm.image.from"].Text
	switch {
	case m.machine && kind == catalog.RoleComputer && m.vm != nil:
		return Template{}, &VMOnComputerError{Name: name, File: m.vm.File}
	case !m.machine || kind == catalog.RoleComputer || from == "":
		return Template{}, &NotTemplateError{Name: name}
	}
	t := Template{
		Name: name, From: from, OS: catalog.OS(m.values["machine.system.os"].Text), Kind: kind, Lock: lock,
		Lifetime: catalog.Lifetime(m.values["vm.lifetime"].Text), Network: m.values["vm.network"].Text,
		Memory: m.values["vm.resources.memory"].Text, Disk: m.values["vm.resources.disk"].Text, Mounts: m.mounts,
	}
	t.CPUs, _ = strconv.Atoi(m.values["vm.resources.cpus"].Text)
	for _, key := range m.order {
		if account, ok := strings.CutPrefix(key, "accounts.share."); ok {
			term := m.values[key].Text
			if term == "shared" {
				term = ""
			}
			t.Shares = append(t.Shares, catalog.AccountShare{Account: account, For: term})
		}
	}
	for _, b := range m.bake {
		t.Bake = append(t.Bake, b.name)
	}
	h := sha256.New()
	for _, f := range files {
		file := path.Base(f.Path)
		h.Write([]byte(file + "\x00"))
		h.Write(f.Body)
		h.Write([]byte{0})
		if f.Layer != Builtin {
			t.Files = append(t.Files, File{Path: machinefilesDir + "/" + file, Body: f.Body})
		}
	}
	h.Write(lock)
	t.SHA = hex.EncodeToString(h.Sum(nil))
	return t, nil
}

func (l Library) Templates() []string {
	// value: the names of every template the library holds, in name order.
	var out []string
	for _, name := range sortedNames(l.Machinefiles) {
		if _, err := TemplateOf(l, name, nil); err == nil {
			out = append(out, name)
		}
	}
	return out
}
