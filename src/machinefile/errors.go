package machinefile

import "strings"

type NotFoundError struct {
	Name string
	// value: the Machinefile whose extends names it; empty when the owner
	// named it.
	By string
}

func (e *NotFoundError) Error() string {
	if e.By != "" {
		return e.By + " extends " + e.Name + ", which no source holds"
	}
	return "no Machinefile named " + e.Name
}

type MixinError struct{ Name string }

func (e *MixinError) Error() string {
	return e.Name + " is a mixin (it says no machine): it is extended, never applied"
}

type VMOnComputerError struct {
	Name string
	// value: the file that says vm.
	File string
}

func (e *VMOnComputerError) Error() string {
	return e.Name + " is a computer: vm is for a workspace or a server (" + e.File + ")"
}

type CycleError struct {
	// value: the names around the cycle, its first again at its end.
	Path []string
	// value: what the cycle is of: extends, or the items' requires and after.
	Of string
}

func (e *CycleError) Error() string { return "a cycle in " + e.Of + ": " + e.Said() }

func (e *CycleError) Said() string {
	// value: the cycle in words, each name and the one it waits on: `a
	// extends b extends a`, `left needs right needs left`.
	verb := " needs "
	if e.Of == "extends" {
		verb = " extends "
	}
	return strings.Join(e.Path, verb)
}

type UnknownItemError struct {
	Name string
	// value: the file that lists it, or the item that requires it.
	By string
	// value: what it is when it is an item of the other kind.
	Is string
}

func (e *UnknownItemError) Error() string {
	if e.Is != "" {
		return e.Name + " is " + e.Is + " (" + e.By + ")"
	}
	return "the catalog has no item " + e.Name + " (" + e.By + ")"
}

type NoMethodError struct {
	Item, System string
	// value: the manager the Machinefile asked, when it asked one.
	Manager string
	// value: why the item is in the plan: listed, or required by another.
	Why string
}

func (e *NoMethodError) Error() string {
	by := ""
	if e.Manager != "" {
		by = " by " + e.Manager
	}
	return e.Item + " has no method" + by + " for " + e.System + " (" + e.Why + ")"
}

type UnpinnableError struct {
	Item, Version, Method string
	// value: another method of the item for this system that pins, or
	// empty.
	Alternative string
}

func (e *UnpinnableError) Error() string {
	return e.Item + " " + e.Version + ": " + e.Method + " cannot pin a version"
}

type BadValueError struct {
	Item, Key, Problem string
}

func (e *BadValueError) Error() string {
	if e.Key == "" {
		return e.Item + ": " + e.Problem
	}
	return e.Item + "'s " + e.Key + ": " + e.Problem
}

type UnknownKeyError struct{ Key, Name string }

func (e *UnknownKeyError) Error() string { return e.Name + " says nothing of " + e.Key }
