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
	// at the engine's start and Begun that book's head: a first readable
	// anchor anchors that book alone, and only when it is that head.
	Head  Head      `json:"head"`
	State HoldState `json:"state,omitempty"`
	Start string    `json:"start,omitempty"`
	Begun Head      `json:"begun,omitzero"`
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

func Await(start Book, sum string) Hold {
	return Hold{Head: unheld(start.Generation), State: Awaiting, Start: sum, Begun: start.Head()}
}

func (h Hold) Settle(now string, anchored Head) Hold {
	// invariant: an engine that could not read the anchor at its start
	// holds the first head it reads only when that head anchors the book
	// it found at its start and the file is still that book, byte for
	// byte; any other head, and any book written since, is trusted
	// nowhere until the engine writes one.
	if h.State != Awaiting {
		return h
	}
	if now != h.Start || anchored != h.Begun {
		return Refuse(h.Head.Generation)
	}
	return Keep(anchored)
}

func (h Hold) Standing(b Book, unreadable bool) Standing {
	switch h.State {
	case Awaiting:
		// invariant: a book with no device waiting for its check is no
		// need: it lets no one in, checked or not.
		if len(b.Devices) == 0 {
			return Kept
		}
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
