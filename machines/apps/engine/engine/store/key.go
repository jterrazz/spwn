package store

import (
	"errors"
	"fmt"
	"strings"
)

type Key string

type Family string

const (
	FamilyMac      Family = "mac"
	FamilyPorts    Family = "ports"
	FamilyOverview Family = "overview"
	FamilyAwake    Family = "awake"
	FamilyActivity Family = "activity"
	FamilySignins  Family = "signins"
	FamilyTailnet  Family = "tailnet"
	FamilyRecords  Family = "records"
	FamilyVault    Family = "vault"
)

func Of(f Family, machine string) Key {
	// invariant: a family that names no machine has one key, its own name.
	if !f.PerMachine() {
		return Key(f)
	}
	return Key(string(f) + ":" + machine)
}

func (f Family) PerMachine() bool {
	switch f {
	case FamilyMac, FamilyPorts, FamilyOverview, FamilyAwake, FamilyActivity, FamilySignins:
		return true
	default:
		return false
	}
}

func (f Family) known() bool {
	switch f {
	case FamilyTailnet, FamilyRecords, FamilyVault:
		return true
	default:
		return f.PerMachine()
	}
}

func ParseKey(s string) (Key, error) {
	family, machine, per := strings.Cut(s, ":")
	f := Family(family)
	if !f.known() {
		return "", fmt.Errorf("%q names no family of the store", s)
	}
	if f.PerMachine() && !per {
		return "", fmt.Errorf("%q: the family %s names a machine", s, family)
	}
	if !f.PerMachine() && per {
		return "", fmt.Errorf("%q: the family %s names no machine", s, family)
	}
	if per && !machineName(machine) {
		return "", fmt.Errorf("%q: a machine is lowercase letters, digits, dashes and dots", s)
	}
	return Key(s), nil
}

func machineName(s string) bool {
	// invariant: a machine is named as every call names it: a registered
	// name, or an unregistered Mac's hostname lowercased, dots and all.
	if s == "" || s[0] == '.' {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '.' {
			return false
		}
	}
	return true
}

func (k Key) Family() Family {
	family, _, _ := strings.Cut(string(k), ":")
	return Family(family)
}

func (k Key) Machine() string {
	_, machine, _ := strings.Cut(string(k), ":")
	return machine
}

type Pattern string

const Local Pattern = "local"

var errNoPattern = errors.New("a pattern is a key, <family>:* or local")

func ParsePattern(s string) (Pattern, error) {
	// invariant: Local is every exported key of the Mac that answers; the
	// other patterns are an exact key and `<family>:*`.
	if Pattern(s) == Local {
		return Local, nil
	}
	if family, ok := strings.CutSuffix(s, ":*"); ok {
		if !Family(family).PerMachine() {
			return "", fmt.Errorf("%q: %w; %s holds no slice per machine", s, errNoPattern, family)
		}
		return Pattern(s), nil
	}
	if _, err := ParseKey(s); err != nil {
		return "", fmt.Errorf("%w: %w", errNoPattern, err)
	}
	return Pattern(s), nil
}

func (p Pattern) Match(k Key, thisMac func(machine string) bool) bool {
	// invariant: thisMac says whether a machine is the answering Mac or one of
	// its VMs; Local alone asks it, so a far Mac's copy is never re-exported,
	// and a key that names no machine (the tailnet, the records, the vault)
	// is This Mac's own, never exported.
	if p == Local {
		return k.Family().PerMachine() && thisMac(k.Machine())
	}
	if family, ok := strings.CutSuffix(string(p), ":*"); ok {
		return k.Family() == Family(family) && k.Family().PerMachine()
	}
	return Key(p) == k
}
