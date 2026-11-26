package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type File struct {
	// value: a slash path inside the agent's run folder.
	Path string
	Data []byte
	// value: a file's permission bits, as 0o644.
	Mode uint32
	// value: a link whose target is Data, its Mode unread.
	Link bool `json:",omitempty"`
}

func local(p string) bool {
	// invariant: a slash path that names a file under its folder: never
	// empty, absolute, dotted or climbing out, and holding no NUL nor any
	// other control character.
	return p != "" && p != "." && path.Clean(p) == p && !strings.HasPrefix(p, "/") && p != ".." && !strings.HasPrefix(p, "../") && !strings.ContainsFunc(p, unicode.IsControl)
}

func InRunFolder(p string) bool {
	// invariant: a path from the project's root that names a file under one
	// agent's run folder, local and clean; the only path apply reads,
	// writes or removes, whatever its record says.
	rest, ok := strings.CutPrefix(p, StateFolder+"/agents/")
	agent, file, _ := strings.Cut(rest, "/")
	return ok && local(p) && agent != "" && local(file)
}

type Kind string

const (
	Add       Kind = "add"
	Overwrite Kind = "overwrite"
	Remove    Kind = "remove"
)

func (Kind) Enum() []string { return []string{string(Add), string(Overwrite), string(Remove)} }

type Change struct {
	Agent string
	// value: the file, from the project's root: RunFolder(Agent)/<path>.
	Path string
	Kind Kind
	// value: what the disk holds there is not what apply last wrote: a
	// file edited by hand since, or one apply never wrote.
	ByHand bool `json:",omitempty"`
	// value: what an add or an overwrite writes.
	File File `json:"-"`
}

type Record map[string]string

type Applied struct {
	Root     string
	Manifest string
	// value: what apply wrote and removed, by path; none when the project
	// already was as it declares.
	Changes []Change
}

const (
	// value: the folder of the project's state no file of it tracks.
	StateFolder = ".spwn"
	// value: the record of what apply wrote, in StateFolder.
	RecordName = "applied.json"
)

var (
	ErrOutside = errors.New("a rendered path leaves its run folder")
	// value: a link, or a file with another name, where apply reaches: on a
	// folder of a run folder's path, on the record, or at a file it reads.
	ErrLink = errors.New("is a link, which apply never follows")
	// value: a name of the declaration that is neither a file nor a folder,
	// a FIFO or a device say, which a read could wait on forever.
	ErrNotAFile = errors.New("is neither a file nor a folder")
	// value: a name on a rendered file's way that holds a file, where apply
	// makes a folder: never removed, as it is the owner's.
	ErrNotAFolder = errors.New("is a file where apply makes a folder")
)

func Declaration() []string {
	// value: the files and the folder a project is declared in, from its
	// root; everything else at the root is the owner's.
	return []string{ManifestName, "spwn.lock", "spwn"}
}

func Declared(p string) bool {
	// value: p, a slash path from the root, is the declaration or under it.
	for _, d := range Declaration() {
		if p == d || strings.HasPrefix(p, d+"/") {
			return true
		}
	}
	return false
}

type Unwritten struct {
	// value: the path from the project's root a call cannot reach, read or
	// write, and why.
	Path string
	Err  error
}

func (u *Unwritten) Error() string { return u.Path + " " + u.Err.Error() }

func (u *Unwritten) Unwrap() error { return u.Err }

func RunFolder(agent string) string { return StateFolder + "/agents/" + agent }

func Digest(f File) string {
	// invariant: two files with one digest hold the same bytes with the
	// same mode, a link's target as its bytes.
	head := "file " + strconv.FormatUint(uint64(f.Mode), 8)
	if f.Link {
		head = "link"
	}
	sum := sha256.Sum256(append([]byte(head+"\n"), f.Data...))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func Paths(rendered map[string][]File, record Record) []string {
	// value: every path a plan of changes reads on disk: each one rendered,
	// and each one apply wrote before that is in a run folder; the record is
	// read as untrusted, so any other path it holds is never read.
	var out []string
	for agent, files := range rendered {
		for _, f := range files {
			out = append(out, RunFolder(agent)+"/"+f.Path)
		}
	}
	for p := range record {
		if InRunFolder(p) {
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func ChangesOf(rendered map[string][]File, record, disk Record) ([]Change, Record, error) {
	// invariant: the changes that make each agent's run folder what its
	// harness renders, by path, and the record apply then keeps. A file
	// the disk holds as rendered is no change; one it lacks is an add; one
	// that differs is an overwrite, by hand when apply did not leave it
	// so. A file apply wrote that is no longer rendered is removed, by hand
	// when edited since, and dropped unsaid when already gone. A file apply
	// never wrote and that is not rendered is never read nor touched, and
	// so is a path of the record that is not in a run folder.
	var out []Change
	next := Record{}
	for agent, files := range rendered {
		for _, f := range files {
			if !local(f.Path) {
				return nil, nil, fmt.Errorf("%w: %s renders %q", ErrOutside, agent, f.Path)
			}
			p := RunFolder(agent) + "/" + f.Path
			want := Digest(f)
			next[p] = want
			have, on := disk[p]
			switch {
			case !on:
				out = append(out, Change{Agent: agent, Path: p, Kind: Add, File: f})
			case have != want:
				out = append(out, Change{Agent: agent, Path: p, Kind: Overwrite, ByHand: record[p] != have, File: f})
			}
		}
	}
	for p, was := range record {
		if _, kept := next[p]; kept || !InRunFolder(p) {
			continue
		}
		have, on := disk[p]
		if !on {
			continue
		}
		out = append(out, Change{Agent: agentOf(p), Path: p, Kind: Remove, ByHand: have != was})
	}
	slices.SortFunc(out, func(a, b Change) int { return strings.Compare(a.Path, b.Path) })
	return out, next, nil
}

func agentOf(p string) string {
	rest, _ := strings.CutPrefix(p, StateFolder+"/agents/")
	agent, _, _ := strings.Cut(rest, "/")
	return agent
}

func Asked(changes []Change) []Change {
	// value: the changes an apply asks before: every one but an add, as
	// each loses what the disk held.
	var out []Change
	for _, c := range changes {
		if c.Kind != Add {
			out = append(out, c)
		}
	}
	return out
}

func Kept(was, next Record, changes []Change, done int) Record {
	// value: the record an apply keeps when it stops after its first done
	// changes: what it wrote, each file the disk already held as rendered,
	// and, for a change it did not make, the file it wrote before as it
	// was then.
	out := maps.Clone(next)
	for _, c := range changes[done:] {
		if d, ok := was[c.Path]; ok {
			out[c.Path] = d
			continue
		}
		delete(out, c.Path)
	}
	return out
}

func Goes(changes []Change) string {
	// value: what an apply loses, each file with what becomes of it, as
	// its question and its refusal name it.
	said := make([]string, 0, len(changes))
	for _, c := range changes {
		verb := "overwrites "
		if c.Kind == Remove {
			verb = "removes "
		}
		line := verb + c.Path
		if c.ByHand {
			line += ", changed by hand"
		}
		said = append(said, line)
	}
	return strings.Join(said, "; ")
}
