package contract

import (
	"context"
	"strconv"
	"strings"
)

type MachineBlock struct {
	Name, User string
	// value: the private key file; empty is the owner's default key.
	Key string
	// value: the Mac a VM runs on, empty for a Mac; ThisMac when that Mac is this one.
	Behind  string
	ThisMac bool
}

const sshWord = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-"

func Unwritable(b MachineBlock) string {
	// invariant: a block's name, user and Mac each stand in ssh_config as
	// one word, so none may hold a space, a newline, a quote or a shell
	// character; the reason a block is refused, empty when it is written.
	if b.Name == "" || strings.Trim(b.Name, sshWord) != "" {
		return "the name " + strconv.Quote(b.Name) + notOneWord
	}
	for _, f := range []struct{ what, value string }{{"user", b.User}, {"Mac", b.Behind}} {
		if strings.Trim(f.value, sshWord) != "" {
			return b.Name + "'s " + f.what + " " + strconv.Quote(f.value) + notOneWord
		}
	}
	return ""
}

const notOneWord = ` is not one word of letters, digits, ".", "_" and "-"`

type SSHConfigError struct {
	Path  string
	Cause error
}

func (e *SSHConfigError) Error() string { return e.Cause.Error() }

func (e *SSHConfigError) Unwrap() error { return e.Cause }

type SSHConfig interface {
	// invariant: a Host the owner wrote outside station's blocks is never
	// shadowed: its block is refused with an SSHConfigError, as is a block
	// Unwritable refuses, and the others are still written.
	Put(ctx context.Context, blocks []MachineBlock) error
	// value: the blocks are every one station manages; any other block goes.
	Replace(ctx context.Context, blocks []MachineBlock) error
	Remove(ctx context.Context, names ...string) error
}
