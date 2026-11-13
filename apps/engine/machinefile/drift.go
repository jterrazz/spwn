package machinefile

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

// value: a part of what a Machinefile puts on a machine, as drift counts
// it: its apps (managers and the blocks of ~/.zshrc their methods write
// included), its settings, the ports a VM shares and the accounts it is
// given.

type Chapter string

const (
	ChapterApps     Chapter = "apps"
	ChapterSettings Chapter = "settings"
	ChapterPorts    Chapter = "ports"
	ChapterAccounts Chapter = "accounts"
)

func (Chapter) Enum() []string {
	return []string{string(ChapterApps), string(ChapterSettings), string(ChapterPorts), string(ChapterAccounts)}
}

// value: a machine against its Machinefile: the machine, the plan its
// packages read as, each chapter with what differs in it, how many lines
// differ in all, what apply would do with them, and the verdict that says
// it.

type Drift struct {
	Machine     string
	Plan        Plan
	Chapters    []ChapterDrift
	Differences int
	Apply       Applying
	Verdict     verdict.Verdict
}

// value: one chapter of a drift: how many of its items the Machinefile
// lists, how many of them match, each line that differs (one the file
// does not list, an extra, included), and its pill.

type ChapterDrift struct {
	Chapter   Chapter
	Listed    int
	Matching  int
	Differing []Differing
	Verdict   verdict.Verdict
}

// value: one line that differs from the Machinefile: the package, the
// block of ~/.zshrc, the port or the account it is, what it reads as, and
// what the machine holds of it against what the file asks (a version, an
// audience, shared or not).

type Differing struct {
	Package   string
	Block     string
	Port      int
	Account   string
	Condition Condition
	Seen      string
	Asked     string
}

// value: what apply would do with the lines that differ, by its verb, and
// the sentence that says it.

type Applying struct {
	Install, Adopt, Remove, Leave int
	Sentence                      string
}

// value: what a VM's Mac holds of what its Machinefile gives it: each port
// the VM shares, with the audience it is shared with against the one its
// file opens it to ("" where the file does not open it, or it is not
// shared), and each account, its copy there or not against shared by the
// file or not.

type Access struct {
	Ports    []PortHeld
	Accounts []AccountHeld
}

type PortHeld struct {
	Port        int
	Seen, Asked catalog.Audience
}

type AccountHeld struct {
	Account     string
	Seen, Asked bool
}

func DriftOf(p Plan, access *Access) Drift {
	// invariant: Differences is the one count of a drift: every package
	// that does not read present, every block of ~/.zshrc of a listed
	// package that does not, and every extra and extra block; each counts
	// in the chapter of its package, an extra in apps. A package matches
	// when it reads present and none of its blocks differs.
	chapters := map[Chapter]*ChapterDrift{ChapterApps: {Chapter: ChapterApps}, ChapterSettings: {Chapter: ChapterSettings}}
	of := map[string]Chapter{}
	for _, l := range p.Lines {
		of[l.Step.Package] = chapterOf(l.Step.Kind)
	}
	var apply Applying
	differs := map[string]bool{}
	for _, l := range p.Lines {
		if l.Seen.Condition == Present {
			continue
		}
		differs[l.Step.Package] = true
		c := chapters[of[l.Step.Package]]
		c.Differing = append(c.Differing, Differing{Package: l.Step.Package, Condition: l.Seen.Condition, Seen: l.Seen.Version, Asked: l.Step.Version.String()})
		apply.count(l.Action)
	}
	for _, b := range p.Blocks {
		if b.Condition == Present || b.Condition == Extra {
			continue
		}
		differs[b.Package] = true
		c := chapters[cmpChapter(of, b.Package)]
		c.Differing = append(c.Differing, Differing{Package: b.Package, Block: b.ID, Condition: b.Condition})
		apply.count(b.Action)
	}
	apps := chapters[ChapterApps]
	for _, e := range p.Extra {
		apps.Differing = append(apps.Differing, Differing{Package: e.Step.Package, Condition: Extra, Seen: e.Seen.Version})
		if e.Takes() {
			apply.Remove++
		} else {
			apply.Leave++
		}
	}
	for _, b := range p.Blocks {
		if b.Condition != Extra {
			continue
		}
		apps.Differing = append(apps.Differing, Differing{Package: b.Package, Block: b.ID, Condition: Extra})
		if b.Action == Remove {
			apply.Remove++
		} else {
			apply.Leave++
		}
	}
	for _, l := range p.Lines {
		c := chapters[of[l.Step.Package]]
		c.Listed++
		if !differs[l.Step.Package] {
			c.Matching++
		}
	}
	d := Drift{Plan: p, Apply: apply.worded()}
	order := []Chapter{ChapterApps, ChapterSettings}
	if access != nil {
		chapters[ChapterPorts], chapters[ChapterAccounts] = access.ports(), access.accounts()
		order = append(order, ChapterPorts, ChapterAccounts)
	}
	for _, name := range order {
		d.Chapters = append(d.Chapters, chapters[name].worded())
		d.Differences += len(chapters[name].Differing)
	}
	d.Verdict = Verdict(d.Differences, p.Name)
	return d
}

// invariant: a port or an account the file gives counts as listed; one
// held otherwise than the file asks differs: absent when the file gives it
// and the Mac holds none, extra when the Mac holds what the file does not
// give, changed when a port is shared with another audience.
func (a Access) ports() *ChapterDrift {
	c := &ChapterDrift{Chapter: ChapterPorts}
	for _, h := range a.Ports {
		if h.Asked != "" {
			c.Listed++
		}
		if h.Seen == h.Asked {
			if h.Asked != "" {
				c.Matching++
			}
			continue
		}
		c.Differing = append(c.Differing, Differing{Port: h.Port, Condition: held(h.Seen != "", h.Asked != ""), Seen: string(h.Seen), Asked: string(h.Asked)})
	}
	return c
}

func (a Access) accounts() *ChapterDrift {
	c := &ChapterDrift{Chapter: ChapterAccounts}
	word := func(shared bool) string {
		if shared {
			return "shared"
		}
		return ""
	}
	for _, h := range a.Accounts {
		if h.Asked {
			c.Listed++
		}
		if h.Seen == h.Asked {
			if h.Asked {
				c.Matching++
			}
			continue
		}
		c.Differing = append(c.Differing, Differing{Account: h.Account, Condition: held(h.Seen, h.Asked), Seen: word(h.Seen), Asked: word(h.Asked)})
	}
	return c
}

func held(seen, asked bool) Condition {
	switch {
	case !seen:
		return Absent
	case !asked:
		return Extra
	}
	return Changed
}

func (d Drift) AccessUnread() Drift {
	// value: a VM's drift whose ports and accounts were not read (asked
	// inside it, or its Mac's shares unread) says so after its count.
	d.Verdict.Sentence += " Its ports and accounts are read from its Mac."
	return d
}

func chapterOf(k catalog.Kind) Chapter {
	if k == catalog.KindSetting {
		return ChapterSettings
	}
	return ChapterApps
}

func cmpChapter(of map[string]Chapter, pkg string) Chapter {
	if c, ok := of[pkg]; ok {
		return c
	}
	return ChapterApps
}

func (a *Applying) count(act Action) {
	switch act {
	case Install, Set:
		a.Install++
	case Adopt:
		a.Adopt++
	case Remove:
		a.Remove++
	case Leave:
		a.Leave++
	case Keep:
	}
}

func (a Applying) worded() Applying {
	// value: the mockup's sentence: each verb with its count, in the order
	// install, adopt, remove, leave; nothing to apply when apply would only
	// leave lines as they are.
	if a.Install+a.Adopt+a.Remove == 0 {
		a.Sentence = "Nothing to apply."
		return a
	}
	var parts []string
	for _, v := range []struct {
		verb string
		n    int
	}{{"install", a.Install}, {"adopt", a.Adopt}, {"remove", a.Remove}, {"leave", a.Leave}} {
		if v.n > 0 {
			parts = append(parts, v.verb+" "+strconv.Itoa(v.n))
		}
	}
	words := parts[len(parts)-1]
	if len(parts) > 1 {
		words = strings.Join(parts[:len(parts)-1], ", ") + " and " + words
	}
	a.Sentence = "Apply will " + words + "."
	return a
}

func (c ChapterDrift) worded() ChapterDrift {
	c.Verdict = verdict.Verdict{Tone: verdict.OK, Sentence: "Matches"}
	if n := len(c.Differing); n > 0 {
		c.Verdict = verdict.Verdict{Tone: verdict.Warn, Sentence: Differences(n)}
	}
	return c
}

func Differences(n int) string {
	// value: a count of differences in the owner's words.
	if n == 1 {
		return "1 difference"
	}
	return strconv.Itoa(n) + " differences"
}

func Verdict(n int, file string) verdict.Verdict {
	// value: a drift's heading: the machine matches its file, or how many
	// lines differ from it.
	if n == 0 {
		return verdict.Verdict{Tone: verdict.OK, Sentence: "Matches " + file + "."}
	}
	return verdict.Verdict{Tone: verdict.Warn, Sentence: Differences(n) + " from " + file + "."}
}
