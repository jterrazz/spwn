package calls

import (
	"encoding/json"
	"time"
)

const knownVersion = 1

type knownFile struct {
	V     int                   `json:"v"`
	Sizes map[string]knownEntry `json:"sizes"`
}

type knownEntry struct {
	Bytes int64     `json:"bytes"`
	At    time.Time `json:"at"`
}

func (s *Surveyor) knownPath() string {
	if s.o.Expand.Roots.Cache == "" {
		return ""
	}
	return s.o.Expand.Expand("{cache}/station/sizes.json")
}

func (s *Surveyor) loadKnown() map[string]knownEntry {
	file := s.knownPath()
	if file == "" {
		return nil
	}
	raw, err := s.m.ReadFile(file)
	if err != nil {
		return nil
	}
	var k knownFile
	if json.Unmarshal(raw, &k) != nil || k.V != knownVersion {
		return nil
	}
	return k.Sizes
}

// invariant: last-known sizes are stale-while-revalidate, never a reason to skip
// a walk (a directory's mtime does not move when a file deeper down grows): every
// walk rewrites the file with what it answered, by an atomic rename.
func (s *Surveyor) saveKnown(roots []string, sizes []int64) {
	file := s.knownPath()
	if file == "" {
		return
	}
	known := s.loadKnown()
	if known == nil {
		known = map[string]knownEntry{}
	}
	at := s.m.Now().UTC().Truncate(time.Second)
	for i, root := range roots {
		if sizes[i] >= 0 {
			known[root] = knownEntry{Bytes: sizes[i], At: at}
		}
	}
	raw, err := json.Marshal(knownFile{V: knownVersion, Sizes: known})
	if err != nil {
		return
	}
	_ = s.m.Keep(file, raw)
}
