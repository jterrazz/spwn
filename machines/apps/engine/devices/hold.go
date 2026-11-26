package devices

import (
	"crypto/sha256"
	"encoding/hex"
)

type Standing string

const (
	// value: how the book on disk stands with this Mac's running engine:
	// the one it holds; changed outside Spwn, so nothing of it is trusted;
	// or not checked yet, the login keychain unreadable since the engine
	// started, so nothing of it is trusted until it can be read.
	Kept      Standing = ""
	Changed   Standing = "changed"
	Unchecked Standing = "unchecked"
)

type HoldState string

const (
	// value: the engine holds a head it read from the anchor or wrote; it
	// waits for the anchor's first readable answer; or it holds none it
	// may trust until it writes, the start read or its wait having met a
	// book or a head Spwn never writes.
	Holding  HoldState = ""
	Awaiting HoldState = "awaiting"
	Unheld   HoldState = "unheld"
)

type Hold struct {
	// invariant: what this Mac's engine trusts its book by, from its start
	// read on: the head it holds, a head no book has while it awaits or
	// refuses; while it awaits, Start is the SHA-256 of the book's bytes
	// at the engine's start, the only book a first readable anchor may
	// then anchor.
	Head  Head      `json:"head"`
	State HoldState `json:"state,omitempty"`
	Start string    `json:"start,omitempty"`
}

func Sum(data []byte) string {
	if data == nil {
		return "absent"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func unheld(generation int) Head {
	return Head{Generation: max(generation, 1), Digest: hex.EncodeToString(make([]byte, sha256.Size))}
}

func Keep(h Head) Hold { return Hold{Head: h} }

func Refuse(generation int) Hold { return Hold{Head: unheld(generation), State: Unheld} }

func Await(generation int, start string) Hold {
	return Hold{Head: unheld(generation), State: Awaiting, Start: start}
}

func (h Hold) Settle(now string, anchored Head) Hold {
	// invariant: an engine that could not read the anchor at its start
	// takes the first head it reads only with the book it found at its
	// start, byte for byte: a book written since, whatever head the
	// keychain then holds, is trusted nowhere until the engine writes one.
	if h.State != Awaiting {
		return h
	}
	if now != h.Start {
		return Refuse(h.Head.Generation)
	}
	return Keep(anchored)
}

func (h Hold) Standing(b Book, unreadable bool) Standing {
	switch h.State {
	case Awaiting:
		return Unchecked
	case Unheld:
		return Changed
	case Holding:
	}
	if unreadable || !b.Anchored(h.Head) {
		return Changed
	}
	return Kept
}
