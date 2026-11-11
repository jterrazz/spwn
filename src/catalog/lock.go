package catalog

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// value: the schema of the lock `apply` and `upgrade` write beside a
// Machinefile, and the family its versions share.

const (
	LockSchema = "station.lock/v1"
	LockFamily = "station.lock"
	LockSuffix = ".lock"
)

var locks = format{key: "lock", schema: LockSchema, family: LockFamily}

// value: what one machine carries of a Machinefile, as its last apply or
// upgrade left it: the shared source's commit it was read at, and each
// app with the method that holds it and the version there.

type LockFile struct {
	Lock        string      `yaml:"lock"`
	Machinefile string      `yaml:"machinefile"`
	Source      *LockSource `yaml:"source,omitempty"`
	Apps        []LockedApp `yaml:"apps,omitempty"`
}

type LockSource struct {
	Name   string `yaml:"name"`
	Commit string `yaml:"commit"`
}

type LockedApp struct {
	Name    string `yaml:"name"`
	Method  string `yaml:"method"`
	Version string `yaml:"version,omitempty"`
}

var lockCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

func DecodeLock(file string, body []byte) (LockFile, error) {
	// invariant: a lock is read as strictly as a Machinefile, and each app
	// is named once, by a method key: a lock edited by hand into another
	// shape is refused at its file, never half read.
	var l LockFile
	if _, err := strict(file, body, locks, &l); err != nil {
		return LockFile{}, err
	}
	name, _ := MachinefileOf(strings.TrimSuffix(file, LockSuffix) + ".yaml")
	if l.Machinefile != name {
		return LockFile{}, &FileError{File: file, Problem: fmt.Sprintf("locks %q, not the Machinefile %s its name says", l.Machinefile, name)}
	}
	if l.Source != nil && (l.Source.Name == "" || !lockCommit.MatchString(l.Source.Commit)) {
		return LockFile{}, &FileError{File: file, Problem: "its source names no whole commit (40 characters)"}
	}
	seen := map[string]bool{}
	for _, a := range l.Apps {
		switch {
		case a.Name == "" || a.Method == "":
			return LockFile{}, &FileError{File: file, Problem: "an app with no name or no method"}
		case seen[a.Name]:
			return LockFile{}, &FileError{File: file, Problem: "locks " + a.Name + " twice"}
		}
		seen[a.Name] = true
	}
	return l, nil
}

func EncodeLock(l LockFile) ([]byte, error) {
	// invariant: the same lock is the same bytes: its apps by name, so the
	// diff of two locks is the diff of what changed.
	l.Lock = LockSchema
	l.Apps = slices.Clone(l.Apps)
	slices.SortFunc(l.Apps, byName)
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(4)
	if err := enc.Encode(l); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func byName(a, b LockedApp) int { return strings.Compare(a.Name, b.Name) }
