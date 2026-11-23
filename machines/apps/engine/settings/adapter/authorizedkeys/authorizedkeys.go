package authorizedkeys

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	Replace(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Chmod(path string, perm fs.FileMode) error
	Remove(path string) error
}

type Options struct {
	// value: the folder the system's absolute paths are read under ("" on
	// the machine itself), and the file station keeps what it last wrote in.
	Root, Record string
	// value: a path made the user's, as sshd's strict modes ask.
	Give func(path string, uid, gid int) error
}

type Applier struct {
	files Files
	o     Options
}

var _ contract.Applier = (*Applier)(nil)

func New(files Files, o Options) *Applier { return &Applier{files: files, o: o} }

func (*Applier) Kind() catalog.EffectKind { return catalog.EffectAuthorizedKeys }

const (
	InputUser = "user"
	InputKey  = "key"
)

// value: what station last wrote: the file, as the system names it, and
// the block's text.
type record struct {
	File string `json:"file"`
	Body string `json:"body"`
}

type account struct {
	home     string
	uid, gid int
}

func of(it catalog.Setting) (catalog.AuthorizedKeys, error) {
	a, ok := it.Effect.(catalog.AuthorizedKeys)
	if !ok || a.Block == "" {
		return catalog.AuthorizedKeys{}, errors.New(it.Name + " is not authorized keys Machines writes")
	}
	return a, nil
}

func (a *Applier) at(p string) string { return path.Join("/", a.o.Root, p) }

func (a *Applier) last() (record, bool, error) {
	data, err := a.files.ReadFile(a.o.Record)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return record{}, false, nil
	case err != nil:
		return record{}, false, err
	}
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return record{}, false, fmt.Errorf("%s: %w", a.o.Record, err)
	}
	return r, r.File != "", nil
}

func (a *Applier) text(file string) (text string, there bool, err error) {
	data, err := a.files.ReadFile(a.at(file))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", false, nil
	case err != nil:
		return "", false, err
	}
	return string(data), true, nil
}

func (a *Applier) Observe(_ context.Context, it catalog.Setting) (settings.Observation, error) {
	k, err := of(it)
	if err != nil {
		return settings.Observation{}, err
	}
	r, ok, err := a.last()
	if err != nil || !ok {
		return settings.Observation{State: catalog.StateAbsent}, err
	}
	text, _, err := a.text(r.File)
	if err != nil {
		return settings.Observation{}, err
	}
	o := settings.Observation{Where: r.File}
	have, state := settings.PackageBlock(text, k.Block)
	switch {
	case state == settings.BlockAbsent:
		o.State = catalog.StateAbsent
		return o, nil
	case state == settings.BlockHere && have == r.Body:
		o.State, o.Detail = catalog.StateOK, r.File
	default:
		o.State, o.Detail = catalog.StateDegraded, "differs from Machines': station settings apply "+it.Name
		o.Diff = settings.UnifiedDiff(r.Body, have, "Machines' "+r.File, r.File)
	}
	sum := sha256.Sum256([]byte(r.File + "\x00" + have))
	o.Digest = "sha256:" + hex.EncodeToString(sum[:])
	return o, nil
}

func (a *Applier) Apply(_ context.Context, it catalog.Setting, values map[string]string) (settings.Change, error) {
	// invariant: every value is read, and refused, before the first write:
	// a key that does not parse never reaches the file sshd reads.
	k, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	body, err := Keys(values[InputKey])
	if err != nil {
		return settings.Change{}, err
	}
	user := strings.TrimSpace(values[InputUser])
	who, err := a.account(user)
	if err != nil {
		return settings.Change{}, err
	}
	dir := path.Join(who.home, ".ssh")
	file := path.Join(dir, "authorized_keys")
	text, there, err := a.text(file)
	if err != nil {
		return settings.Change{}, err
	}
	updated, changed := settings.UpsertPackageBlock(text, k.Block, body)
	err = a.files.MkdirAll(a.at(dir), 0o700)
	if err != nil {
		return settings.Change{}, err
	}
	if changed || !there {
		err = a.files.Replace(a.at(file), []byte(updated), 0o600)
		if err != nil {
			return settings.Change{}, err
		}
	}
	for _, p := range []struct {
		at   string
		mode fs.FileMode
	}{{dir, 0o700}, {file, 0o600}} {
		err = a.files.Chmod(a.at(p.at), p.mode)
		if err != nil {
			return settings.Change{}, err
		}
		err = a.o.Give(a.at(p.at), who.uid, who.gid)
		if err != nil {
			return settings.Change{}, err
		}
	}
	data, err := json.Marshal(record{File: file, Body: body})
	if err != nil {
		return settings.Change{}, err
	}
	err = a.files.MkdirAll(path.Dir(a.o.Record), 0o700)
	if err != nil {
		return settings.Change{}, err
	}
	err = a.files.Replace(a.o.Record, data, 0o600)
	if err != nil {
		return settings.Change{}, err
	}
	detail := "written"
	if !changed && there {
		detail = "already there"
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: file, State: catalog.StateOK, Detail: detail}}}, nil
}

func (a *Applier) Undo(_ context.Context, it catalog.Setting) (settings.Change, error) {
	// invariant: the block leaves the file only while it is still exactly
	// station's, and nothing else of the file moves: off never destroys a
	// line of the owner's.
	k, err := of(it)
	if err != nil {
		return settings.Change{}, err
	}
	r, ok, err := a.last()
	if err != nil || !ok {
		return settings.Change{}, err
	}
	text, _, err := a.text(r.File)
	if err != nil {
		return settings.Change{}, err
	}
	have, state := settings.PackageBlock(text, k.Block)
	switch {
	case state == settings.BlockAbsent:
		return settings.Change{}, a.files.Remove(a.o.Record)
	case state != settings.BlockHere || have != r.Body:
		return settings.Change{Steps: []settings.ChangeStep{{Label: r.File, State: catalog.StateOK, Detail: "yours, kept"}}}, nil
	}
	updated, _ := settings.RemovePackageBlock(text, k.Block)
	err = a.files.Replace(a.at(r.File), []byte(updated), 0o600)
	if err != nil {
		return settings.Change{}, err
	}
	err = a.files.Remove(a.o.Record)
	if err != nil {
		return settings.Change{}, err
	}
	return settings.Change{Steps: []settings.ChangeStep{{Label: r.File, State: catalog.StateOK, Detail: "removed"}}}, nil
}

func Keys(given string) (string, error) {
	// value: the block's text for the public keys given one per line, each
	// `type base64 [comment]`, or the refusal of the first that does not parse.
	var b strings.Builder
	n := 0
	for i, line := range strings.Split(given, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, err := parse(line)
		if err != nil {
			return "", fmt.Errorf("line %d of the key %w", i+1, err)
		}
		b.WriteString(key + "\n")
		n++
	}
	if n == 0 {
		return "", errors.New("no public key given: `type base64 [comment]`, one per line")
	}
	return b.String(), nil
}

func fields(kind string) int {
	// value: how many length-prefixed strings a key type's blob holds, its
	// type the first; 0 for a type Machines does not write (a hardware
	// key's sk- types among them).
	switch kind {
	case "ssh-ed25519":
		return 2
	case "ssh-rsa", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		return 3
	}
	return 0
}

// invariant: a key is its type, then its blob in base64, whose first string
// is that type and whose strings, as many as the type holds, fill it
// exactly; an ed25519 key is 32 bytes. Anything before the type (sshd's
// options) is refused.
func parse(line string) (string, error) {
	f := strings.Fields(line)
	if len(f) > 1 && fields(f[0]) == 0 {
		for _, w := range f[1:] {
			if fields(w) > 0 {
				return "", errors.New("carries options: a key alone, `type base64 [comment]`")
			}
		}
	}
	refused := errors.New("is not a public key: `type base64 [comment]`")
	if len(f) < 2 || fields(f[0]) == 0 {
		return "", refused
	}
	blob, err := base64.StdEncoding.DecodeString(f[1])
	if err != nil {
		return "", refused
	}
	var parts [][]byte
	for len(blob) > 0 {
		if len(blob) < 4 {
			return "", refused
		}
		size, rest := int64(binary.BigEndian.Uint32(blob)), blob[4:]
		if size > int64(len(rest)) {
			return "", refused
		}
		parts, blob = append(parts, rest[:size]), rest[size:]
	}
	if len(parts) != fields(f[0]) || string(parts[0]) != f[0] || (f[0] == "ssh-ed25519" && len(parts[1]) != 32) {
		return "", refused
	}
	key := f[0] + " " + f[1]
	if comment := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, f[0])), f[1])); comment != "" {
		key += " " + comment
	}
	return key, nil
}

// invariant: root's home is /root; any other user's, and its ids, are its
// line of the system's /etc/passwd.
func (a *Applier) account(user string) (account, error) {
	if user == "" {
		return account{}, errors.New("no user given: the account whose ~/.ssh/authorized_keys takes the key")
	}
	data, err := a.files.ReadFile(a.at("/etc/passwd"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return account{}, err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		f := strings.Split(line, ":")
		if len(f) < 7 || f[0] != user || !path.IsAbs(f[5]) {
			continue
		}
		uid, uerr := strconv.Atoi(f[2])
		gid, gerr := strconv.Atoi(f[3])
		if uerr == nil && gerr == nil {
			return account{home: path.Clean(f[5]), uid: uid, gid: gid}, nil
		}
	}
	if user == "root" {
		return account{home: "/root"}, nil
	}
	return account{}, fmt.Errorf("no user %s on this machine", user)
}
