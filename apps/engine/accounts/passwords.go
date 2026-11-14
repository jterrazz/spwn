package accounts

import (
	"fmt"
	"time"

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

type GrantState string

const (
	GrantActive  GrantState = "active"
	GrantEnding  GrantState = "ending"
	GrantFailing GrantState = "failing"
)

func (GrantState) Enum() []string {
	return []string{string(GrantActive), string(GrantEnding), string(GrantFailing)}
}

type VaultGrant struct {
	ToKind GrantTo
	To     string
	// value: the workspace an agent's grant is for, empty for a machine's.
	Workspace string    `json:",omitempty"`
	Until     time.Time `json:",omitzero"`
	State     GrantState
}

type VaultEntry struct {
	ID      string
	Label   string
	Program Program
	Kind    KeyKind
	Who     string `json:",omitempty"`
	Added   time.Time
	// value: whether an agent may be granted it: a fine-grained, expiring token, never the owner's own login.
	AgentGrantable bool
	Grants         []VaultGrant
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

type Granted struct {
	// value: when the grant ends; zero for one that lasts until it is removed.
	Until time.Time `json:",omitzero"`
}

type Revoked struct {
	// value: where the owner revokes the token itself, which station cannot.
	Sentence string
}

func TermInBounds(to GrantTo, term time.Duration) bool {
	// invariant: a grant lasts 1 min to 7 d; a machine's alone may last until
	// it is removed, which is a term of 0.
	if term == 0 {
		return to == ToMachine
	}
	return term >= ShortestGrant && term <= LongestGrant
}

func Refusal(t fail.Type, code fail.Code, subject string) *fail.Error {
	return fail.New(t, code, subject, nil)
}

func SummarizeVault(entries []VaultEntry) verdict.Verdict {
	// invariant: the head of a Mac's passwords, one sentence and one tone,
	// from its entries alone.
	if len(entries) == 0 {
		return verdict.Verdict{Tone: verdict.Plain, Sentence: SentencePasswordsNone.Text()}
	}
	granted := 0
	for _, e := range entries {
		if len(e.Grants) > 0 {
			granted++
		}
	}
	return verdict.Verdict{Tone: verdict.Plain, Sentence: fmt.Sprintf(SentencePasswordsSome.Text(), len(entries), granted)}
}
