package wiki

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

type Record struct {
	// value: a decision record's head as its file writes it: its number and
	// title, then the lines of its mold, each read and never restated.
	Number    string
	Title     string
	Status    string
	Date      string
	AmendedBy []string
	Amends    []string
}

type ServedDecision struct {
	// value: a decision block as its record says it: the record's number,
	// title, status and date read from the page its adr names, and what the
	// block says it decides. Resolves is false when the adr names no record
	// of this corpus, and then nothing of the record is served.
	Page      string
	Number    string
	Title     string
	Status    string
	Date      string
	Accepted  bool
	AmendedBy []string
	Amends    []string
	Decides   string
	Resolves  bool
}

func (s *server) decision(b *Block) *ServedDecision {
	d := &ServedDecision{Decides: textOf(b.Props, "decides")}
	id, ok := s.c.pageAt(textOf(b.Props, "adr"))
	r, read := s.c.Records[id]
	if !ok || !read {
		return d
	}
	d.Page, d.Number, d.Title, d.Status, d.Date = id, r.Number, r.Title, r.Status, r.Date
	d.Accepted = r.Status == "Accepted"
	d.AmendedBy, d.Amends, d.Resolves = r.AmendedBy, r.Amends, true
	return d
}

var (
	recordTitle = regexp.MustCompile(`^# (ADR-\d+): (.+)$`)
	recordLine  = regexp.MustCompile(`^(?:\*\*)?(Status|Date|Amended by|Amends):?(?:\*\*)? (.+)$`)
	recordLink  = regexp.MustCompile(`\[([^\]]+)\]`)
)

// invariant: a page under a decisions/ folder whose title reads "ADR-n:
// title" is a record, and its head is read from its mold, above its first
// section: the status, the date, and the records it amends or is amended by,
// as `**Key:** value` or as a sentence that opens with the key.
func readRecord(id, body string) (Record, bool) {
	if path.Base(path.Dir(id)) != "decisions" {
		return Record{}, false
	}
	lines := strings.Split(body, "\n")
	m := recordTitle.FindStringSubmatch(strings.TrimSpace(lines[0]))
	if m == nil {
		return Record{}, false
	}
	r := Record{Number: m[1], Title: m[2]}
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		f := recordLine.FindStringSubmatch(strings.TrimSpace(line))
		if f == nil {
			continue
		}
		switch f[1] {
		case "Status":
			r.Status = f[2]
		case "Date":
			r.Date = f[2]
		case "Amended by":
			r.AmendedBy = linksOf(f[2])
		case "Amends":
			r.Amends = linksOf(f[2])
		}
	}
	return r, true
}

func linksOf(s string) []string {
	links := recordLink.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(links))
	for _, m := range links {
		out = append(out, m[1])
	}
	return out
}

// invariant: an anchor names a page of the corpus when its file is read by
// a mount, or when its repository holds the corpus: a repository whose path
// climbs n folders holds the corpus n folders down, so the anchor's path
// past its first n folders is the page.
func (c Corpus) pageAt(anchor string) (string, bool) {
	m := anchorForm.FindStringSubmatch(anchor)
	if len(m) != 4 || m[3] != "" {
		return "", false
	}
	file, md := strings.CutSuffix(m[2], ".md")
	if !md {
		return "", false
	}
	for _, r := range c.Schema.Repos {
		if r.Name != m[1] {
			continue
		}
		for _, mount := range c.Schema.Mounts {
			if mount.Repo != r.Name {
				continue
			}
			in, ok := strings.CutPrefix(mount.Path, r.Path+"/")
			if rest, under := strings.CutPrefix(file, in+"/"); ok && under {
				return c.known(mount.At + "/" + rest)
			}
		}
		up := strings.Split(r.Path, "/")
		parts := strings.Split(file, "/")
		if !slices.ContainsFunc(up, func(s string) bool { return s != ".." }) && len(parts) > len(up) {
			return c.known(strings.Join(parts[len(up):], "/"))
		}
	}
	return "", false
}

func (c Corpus) known(id string) (string, bool) {
	for _, p := range c.Pages {
		if p.ID == id {
			return id, true
		}
	}
	return "", false
}
