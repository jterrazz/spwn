package world

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

type Accounts struct {
	// value: what this machine's own programs say, by program; a program absent here is signed out.
	Programs map[accounts.Program]accounts.Facts `json:"programs,omitempty"`
	// value: how each program's sign-in goes on this machine, by program; a program absent here goes its usual way.
	Logins map[accounts.Program]Login `json:"logins,omitempty"`
	// value: the sign-ins in progress on this machine, by exchange.
	Exchanges map[string]*Exchange `json:"exchanges,omitempty"`
	Counter   int                  `json:"counter,omitempty"`
	// value: what this Mac's vault keeps: each password as the window lists it, its value never stored.
	Passwords []accounts.VaultEntry `json:"passwords,omitempty"`
	// value: the sites this Mac's engine opened for the owner, in order, never a URL.
	Opened []string `json:"opened,omitempty"`
	// value: a delivery into a machine that fails, by machine.
	Undeliverable []string `json:"undeliverable,omitempty"`
}

type Login struct {
	// value: the kind of step the program gives; empty is the program's usual one.
	Kind accounts.StepKind `json:"kind,omitempty"`
	Code string            `json:"code,omitempty"`
	Site string            `json:"site,omitempty"`
	// value: how many seconds of the world's clock a login takes after its step.
	Takes int `json:"takes,omitempty"`
	// value: the refusal code the login ends in before it starts, when it does.
	Refuse string `json:"refuse,omitempty"`
	// value: how the login ends after its step when it does not succeed: expired or failed.
	End string `json:"end,omitempty"`
	// value: the answer a paste login takes as the right one.
	Answer string `json:"answer,omitempty"`
}

type Exchange struct {
	Program accounts.Program  `json:"program"`
	Kind    accounts.StepKind `json:"kind"`
	Site    string            `json:"site"`
	Expires time.Time         `json:"expires"`
	Asking  bool              `json:"asking,omitempty"`
	Answer  string            `json:"answer,omitempty"`
	Given   bool              `json:"given,omitempty"`
}
