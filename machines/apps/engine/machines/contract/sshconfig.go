package contract

import (
	"context"
	"strconv"
	"strings"
	"unicode"
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
	// invariant: a block's name, user and Mac each stand in ssh_config, and
	// in the ssh command that reaches it, as one word, so none may hold a
	// space, a newline, a quote or a shell character, nor start with "-",
	// which ssh reads as an option, and its key is one ssh_config value; the
	// reason a block is refused, empty when it is written.
	if b.Name == "" || strings.Trim(b.Name, sshWord) != "" {
		return "the name " + strconv.Quote(b.Name) + notOneWord
	}
	if strings.HasPrefix(b.Name, "-") {
		return "the name " + strconv.Quote(b.Name) + anOption
	}
	for _, f := range []struct{ what, value string }{{"user", b.User}, {"Mac", b.Behind}} {
		switch {
		case strings.Trim(f.value, sshWord) != "":
			return b.Name + "'s " + f.what + " " + strconv.Quote(f.value) + notOneWord
		case strings.HasPrefix(f.value, "-"):
			return b.Name + "'s " + f.what + " " + strconv.Quote(f.value) + anOption
		}
	}
	return UnwritableKey(b.Name, b.Key)
}

func UnwritableKey(owner, key string) string {
	if strings.IndexFunc(key, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || strings.ContainsRune(`"'`+"`"+`#%\`, r)
	}) < 0 {
		return ""
	}
	return owner + "'s key " + strconv.Quote(key) + ` holds a space, a quote, "#", "%", "\" or a control character, which ssh_config reads apart`
}

func UnusableLAN(mac, lan string) string {
	if lan == "" || dnsName(lan) || ipv6(lan) {
		return ""
	}
	return mac + "'s LAN address " + strconv.Quote(lan) + " is neither an IP address nor a DNS name"
}

func dnsName(s string) bool {
	if len(s) > 253 {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") ||
			strings.Trim(label, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-") != "" {
			return false
		}
	}
	return true
}

func ipv6(s string) bool {
	return len(s) <= 45 && strings.Count(s, ":") >= 2 && strings.Trim(s, "0123456789abcdefABCDEF:.") == ""
}

const (
	notOneWord = ` is not one word of letters, digits, ".", "_" and "-"`
	anOption   = ` starts with "-", which ssh reads as an option`
)

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
