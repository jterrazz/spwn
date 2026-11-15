package accounts

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type KeyKind string

const (
	KeyToken       KeyKind = "token"
	KeyFineGrained KeyKind = "fine-grained"
	KeyAPIKey      KeyKind = "api-key"
	KeyOAuth       KeyKind = "oauth-token"
	KeyPassword    KeyKind = "password"
)

func (KeyKind) Enum() []string {
	return []string{string(KeyToken), string(KeyFineGrained), string(KeyAPIKey), string(KeyOAuth), string(KeyPassword)}
}

type AddFrom string

const (
	FromSignIn AddFrom = "sign-in"
	FromTyped  AddFrom = "typed"
)

func (AddFrom) Enum() []string { return []string{string(FromSignIn), string(FromTyped)} }

type GrantTo string

const (
	ToMachine GrantTo = "machine"
	ToAgent   GrantTo = "agent"
)

func (GrantTo) Enum() []string { return []string{string(ToMachine), string(ToAgent)} }

type EntryReach string

const (
	ReachGranted    EntryReach = "granted"
	ReachNotGranted EntryReach = "not-granted"
)

func (EntryReach) Enum() []string {
	return []string{string(ReachGranted), string(ReachNotGranted)}
}

type VaultGrant struct {
	ToKind GrantTo
	To     string
	// value: the workspace an agent's grant is for, empty for a machine's.
	Workspace string `json:",omitempty"`
}

type VaultEntry struct {
	ID      string
	Label   string
	Program Program
	Kind    KeyKind
	Who     string `json:",omitempty"`
	Added   time.Time
	// value: whether an agent may be granted it: a fine-grained token, never the owner's own login.
	AgentGrantable bool
	Grants         []VaultGrant
	// value: whether any grant was made from it, from its grants.
	Reach EntryReach
	// value: the engine's consequence of removing it, counted from its grants.
	RemoveSentence string
}

type VaultRead struct {
	// value: the Mac whose vault this is: a vault is one Mac's.
	Mac     string
	Entries []VaultEntry
	// value: the head of the Passwords page, from the entries and their grants.
	Verdict verdict.Verdict
}

type Added struct {
	ID string
}

type Removed struct {
	// value: the grants that ended with the password.
	Removed []VaultGrant
}

type Granted struct{}

type Revoked struct {
	// value: where the owner revokes the token itself, which station cannot.
	Sentence string
}

func Refusal(t fail.Type, code fail.Code, subject string) *fail.Error {
	return fail.New(t, code, subject, nil)
}

func DescribeEntry(e VaultEntry) VaultEntry {
	// invariant: an entry's reach and its removal sentence come from its
	// grants alone, so every adapter answers the same words.
	e.Reach, e.RemoveSentence = ReachNotGranted, SentenceRemoveNothing.Text()
	if len(e.Grants) == 0 {
		return e
	}
	e.Reach = ReachGranted
	machines, agents := 0, 0
	for _, g := range e.Grants {
		if g.ToKind == ToAgent {
			agents++
		} else {
			machines++
		}
	}
	parts := make([]string, 0, 2)
	if machines > 0 {
		parts = append(parts, counted(machines, "machine", "machines"))
	}
	if agents > 0 {
		parts = append(parts, counted(agents, "agent", "agents"))
	}
	e.RemoveSentence = fmt.Sprintf(SentenceRemoveEnds.Text(), strings.Join(parts, " and "))
	return e
}

func counted(n int, one, other string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, other)
}

const VaultLine = "a value never leaves this Mac's Keychain to be shown"

func SummarizeVault(entries []VaultEntry) verdict.Verdict {
	// invariant: the head of a Mac's passwords, one sentence and one tone,
	// from its entries alone.
	if len(entries) == 0 {
		return verdict.Verdict{Tone: verdict.Plain, Sentence: SentencePasswordsNone.Text(), Line: VaultLine}
	}
	granted := 0
	for _, e := range entries {
		if len(e.Grants) > 0 {
			granted++
		}
	}
	return verdict.Verdict{Tone: verdict.Plain, Sentence: fmt.Sprintf(SentencePasswordsSome.Text(), len(entries), granted), Line: VaultLine}
}

func (t Program) Delivery() (catalog.Delivery, bool) {
	// value: how a password granted for the program reaches it in a VM.
	i := slices.IndexFunc(catalog.Deliveries, func(d catalog.Delivery) bool { return d.Program == string(t) })
	if i < 0 {
		return catalog.Delivery{}, false
	}
	return catalog.Deliveries[i], true
}

func RevokedSentence(t Program) string {
	// invariant: a removal says where the token itself is revoked, which
	// station cannot do; a program that reads it from an environment file
	// loses it in new shells only, and the sentence says so.
	line := "The token stays valid at " + t.Label() + "; revoke it there to end it everywhere"
	if d, ok := t.Delivery(); ok && d.Env != "" {
		line += ". New shells no longer see it; a shell open now keeps it until it closes"
	}
	return line
}
