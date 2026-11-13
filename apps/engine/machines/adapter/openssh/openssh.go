package openssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

// value: the name sits in both markers, so each block names itself to a human who opens the file.
const (
	beginPrefix = "# >>> jterrazz machine "
	beginSuffix = " >>>"
	endPrefix   = "# <<< jterrazz machine "
	endSuffix   = " <<<"
)

const defaultKey = "~/.ssh/id_ed25519"

func beginMarker(name string) string { return beginPrefix + name + beginSuffix }
func endMarker(name string) string   { return endPrefix + name + endSuffix }

type Options struct {
	Home       string
	KnownHosts string
	// value: this station binary, the ProxyCommand of a VM of this Mac.
	Executable string
}

// invariant: the owner's own ssh to a VM of another Mac hops through that
// Mac's station (machines pipe); station's own code never takes this way.
const hop = `'PATH="$HOME/.local/bin:/usr/local/bin:$PATH" exec station "$@"'`

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

type sshFile struct {
	fs Files
	o  Options
}

func New(files Files, o Options) contract.SSHConfig { return &sshFile{fs: files, o: o} }

func (c *sshFile) file() string { return path.Join(c.o.Home, ".ssh/config") }

func (c *sshFile) Put(_ context.Context, blocks []contract.MachineBlock) error {
	return c.edit(blocks, false, nil)
}

func (c *sshFile) Replace(_ context.Context, blocks []contract.MachineBlock) error {
	return c.edit(blocks, true, nil)
}

func (c *sshFile) Remove(_ context.Context, names ...string) error {
	return c.edit(nil, false, names)
}

func (c *sshFile) block(b contract.MachineBlock) string {
	// invariant: the name IS the HostName, and each machine keeps its own
	// known_hosts, so a name reused by a new VM meets no changed host key; a VM
	// is reached through its Mac, never on the tailnet.
	lines := []string{beginMarker(b.Name), "Host " + b.Name, "  HostName " + b.Name}
	if b.User != "" {
		lines = append(lines, "  User "+b.User)
	}
	key := b.Key
	if key == "" {
		key = defaultKey
	}
	lines = append(lines,
		"  IdentityFile "+key,
		"  ServerAliveInterval 30",
		"  ServerAliveCountMax 3",
		"  UserKnownHostsFile "+c.tilde(path.Join(c.o.KnownHosts, b.Name)),
		"  StrictHostKeyChecking accept-new",
	)
	if b.Behind != "" {
		lines = append(lines, "  ProxyCommand "+c.proxy(b))
	}
	return strings.Join(append(lines, endMarker(b.Name), ""), "\n")
}

func (c *sshFile) proxy(b contract.MachineBlock) string {
	words := []string{"machines", "pipe", b.Name}
	if b.ThisMac {
		return c.o.Executable + " " + strings.Join(words, " ")
	}
	// invariant: ssh runs a ProxyCommand through a local shell, then joins
	// its words for the far one: the far command line is one word here,
	// double-quoted, so the far shell still reads hop as one argument.
	far := strings.Join(append([]string{"sh", "-c", hop, "station"}, words...), " ")
	return "ssh " + b.Behind + " -- " + doubleQuoted(far)
}

func doubleQuoted(s string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, r := range s {
		if strings.ContainsRune(`\"$`+"`", r) {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	out.WriteByte('"')
	return out.String()
}

func (c *sshFile) tilde(p string) string {
	home := strings.TrimSuffix(c.o.Home, "/")
	if home != "" && strings.HasPrefix(p, home+"/") {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func (c *sshFile) edit(upsert []contract.MachineBlock, prune bool, remove []string) error {
	if len(remove) == 0 {
		if err := c.fs.MkdirAll(c.o.KnownHosts, 0o700); err != nil {
			return &contract.SSHConfigError{Path: c.o.KnownHosts, Cause: err}
		}
	}
	file := c.file()
	existing, err := c.fs.ReadFile(file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return &contract.SSHConfigError{Path: file, Cause: err}
	}
	updated := existing
	var conflict error
	for _, b := range upsert {
		next, clash := upsertBlock(updated, b.Name, c.block(b))
		if clash != "" && conflict == nil {
			conflict = fmt.Errorf("refusing to modify %s: %s", file, clash)
			continue
		}
		updated = next
	}
	for _, name := range remove {
		updated = removeBlock(updated, name)
	}
	if prune {
		keep := map[string]bool{}
		for _, b := range upsert {
			keep[b.Name] = true
		}
		for _, name := range managedNames(updated) {
			if !keep[name] {
				updated = removeBlock(updated, name)
			}
		}
	}
	if !bytes.Equal(updated, existing) {
		if err := c.fs.MkdirAll(path.Dir(file), 0o700); err != nil {
			return &contract.SSHConfigError{Path: file, Cause: err}
		}
		if err := c.fs.WriteFile(file, updated, 0o600); err != nil {
			return &contract.SSHConfigError{Path: file, Cause: err}
		}
	}
	if conflict != nil {
		return &contract.SSHConfigError{Path: file, Cause: conflict}
	}
	return nil
}

func managedNames(text []byte) []string {
	var names []string
	for _, line := range strings.Split(string(text), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, beginPrefix) && strings.HasSuffix(trimmed, beginSuffix) {
			names = append(names, strings.TrimSuffix(strings.TrimPrefix(trimmed, beginPrefix), beginSuffix))
		}
	}
	sort.Strings(names)
	return names
}

func upsertBlock(existing []byte, name, block string) (updated []byte, conflict string) {
	text := string(existing)
	begin, end := beginMarker(name), endMarker(name)
	beginIdx := strings.Index(text, begin)
	endIdx := strings.Index(text, end)
	if beginIdx >= 0 && endIdx > beginIdx {
		endLine := endIdx + len(end)
		if endLine < len(text) && text[endLine] == '\n' {
			endLine++
		}
		return []byte(text[:beginIdx] + block + text[endLine:]), ""
	}
	if clash := foreignBlock(text, name); clash != "" {
		return existing, clash
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" {
		text += "\n"
	}
	return []byte(text + block), ""
}

func removeBlock(existing []byte, name string) []byte {
	text := string(existing)
	begin, end := beginMarker(name), endMarker(name)
	beginIdx := strings.Index(text, begin)
	endIdx := strings.Index(text, end)
	if beginIdx < 0 || endIdx <= beginIdx {
		return existing
	}
	endLine := endIdx + len(end)
	if endLine < len(text) && text[endLine] == '\n' {
		endLine++
	}
	// invariant: an append always writes exactly one blank line before its block
	// (upsertBlock), even after a blank line the user wrote, so removal takes
	// exactly one and an add then remove gives the file back byte for byte.
	before, after := text[:beginIdx], text[endLine:]
	switch {
	case strings.HasSuffix(before, "\n\n"):
		before = before[:len(before)-1]
	case before == "" && strings.HasPrefix(after, "\n"):
		after = after[1:]
	}
	return []byte(before + after)
}

// invariant: any managed block counts, not only this machine's: a neighbour's
// block is not a foreign Host.
func foreignBlock(text, name string) string {
	inManaged := false
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, beginPrefix) && strings.HasSuffix(trimmed, beginSuffix) {
			inManaged = true
			continue
		}
		if strings.HasPrefix(trimmed, endPrefix) && strings.HasSuffix(trimmed, endSuffix) {
			inManaged = false
			continue
		}
		if inManaged || !strings.HasPrefix(trimmed, "Host ") {
			continue
		}
		for _, word := range strings.Fields(trimmed)[1:] {
			if word == name {
				return fmt.Sprintf("a block of yours already names %s at line %d (outside any managed block)", name, i+1)
			}
		}
	}
	return ""
}
