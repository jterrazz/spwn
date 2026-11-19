package accounts

import (
	"slices"
	"strconv"
)

type Book struct {
	// invariant: the vault's book lists every password of one Mac's vault
	// with its grants, and never a value; Counter only grows, so an id is
	// never given twice.
	Counter int          `json:"counter"`
	Entries []VaultEntry `json:"entries"`
	// value: the programs a removal could not take out of a machine, taken
	// out by the expiry when the machine is back.
	Owed []Owed `json:"owed,omitempty"`
}

type Owed struct {
	Program Program `json:"program"`
	To      string  `json:"to"`
}

func (b *Book) Find(id string) (*VaultEntry, bool) {
	i := slices.IndexFunc(b.Entries, func(e VaultEntry) bool { return e.ID == id })
	if i < 0 {
		return nil, false
	}
	return &b.Entries[i], true
}

func (b *Book) Next() string {
	b.Counter++
	return "pw-" + strconv.Itoa(b.Counter)
}
