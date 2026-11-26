package devices

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type Head struct {
	// invariant: what this Mac last wrote of its device book: its
	// generation, which every add and forget raises by one, and the SHA-256
	// of the book as written; the zero Head is a Mac that never wrote one.
	Generation int    `json:"generation"`
	Digest     string `json:"digest"`
}

func (b Book) Head() Head {
	// invariant: a book with no generation and no entry is the one never
	// written, whose head is the zero Head; any other book's digest covers
	// its generation and every byte of every entry, its seals included.
	if b.Generation == 0 && len(b.Devices) == 0 {
		return Head{}
	}
	data, err := json.Marshal(Book{Generation: b.Generation, Devices: b.Devices})
	if err != nil {
		return Head{Generation: b.Generation}
	}
	sum := sha256.Sum256(data)
	return Head{Generation: b.Generation, Digest: hex.EncodeToString(sum[:])}
}

func (h Head) String() string { return strconv.Itoa(h.Generation) + " " + h.Digest }

const (
	// value: the highest generation Spwn writes, past any count of adds
	// and forgets, so the next one never overflows and a head above it is
	// one Spwn never wrote.
	MaxGeneration = 1 << 53
)

var errBadHead = errors.New("a device book's head is its generation then its digest")

func ParseHead(s string) (Head, error) {
	gen, digest, ok := strings.Cut(strings.TrimSpace(s), " ")
	n, err := strconv.Atoi(gen)
	if !ok || err != nil || n < 1 || n > MaxGeneration || len(digest) != sha256.Size*2 {
		return Head{}, errBadHead
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return Head{}, errBadHead
	}
	return Head{Generation: n, Digest: digest}, nil
}

func (b Book) Anchored(anchored Head) bool { return b.Head() == anchored }

func Base(b Book, anchored Head) Book {
	// invariant: a change starts from the book this Mac last wrote, or from
	// an empty one when the book on disk is not that one: nothing of a book
	// copied back, restored or edited is ever sealed again by a change.
	if !b.Anchored(anchored) {
		return Book{Devices: []Device{}}
	}
	return Book{Generation: b.Generation, Devices: clone(b.Devices)}
}

func Next(b Book, anchored Head, edit func(*Book) error) (Book, error) {
	// invariant: the book a change writes: its edit over Base, one
	// generation past the anchored one, and never past MaxGeneration.
	next := Base(b, anchored)
	if err := edit(&next); err != nil {
		return Book{}, err
	}
	next.Generation = min(anchored.Generation, MaxGeneration-1) + 1
	if next.Devices == nil {
		next.Devices = []Device{}
	}
	return next, nil
}

func clone(in []Device) []Device {
	out := make([]Device, 0, len(in))
	for _, d := range in {
		d.Channels = append([]Channel(nil), d.Channels...)
		d.Seal = append([]byte(nil), d.Seal...)
		out = append(out, d)
	}
	return out
}
