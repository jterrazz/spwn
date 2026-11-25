package wiki

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: the most content kinds a corpus reads, the engine's included.
const maxContentKinds = 16

// value: how many floors a corpus declares.
const floorCount = 5

var (
	seatID      = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	kindID      = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	fieldID     = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	placeholder = regexp.MustCompile(`\{([^{}]*)\}`)
	decodeLine  = regexp.MustCompile(`line (\d+): `)
)

// value: the types a corpus's field may take; the engine's own kinds also
// read glob, object and query.
func corpusTypes() []string {
	return []string{"sentence", "text", "enum", "ref", "anchor", "date", "bool", "int"}
}

func budgets() []Budget { return []Budget{Budget30s, Budget2min, Budget10min} }

type SchemaError struct {
	File string
	Line int
	Says string
}

func (e *SchemaError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: H42 %s", e.File, e.Line, e.Says)
	}
	return fmt.Sprintf("%s: H42 %s", e.File, e.Says)
}

func ReadSchema(file string, body []byte) (Schema, error) {
	// invariant: a pure function of its arguments, like Compile: wiki.yaml is
	// read by the catalog's strict decoder, then judged here, the first
	// refusal in the file's order returned with its line.
	w, err := catalog.DecodeWiki(file, body)
	if err != nil {
		var fe *catalog.FileError
		if errors.As(err, &fe) {
			line, says := fe.Line, fe.Problem
			if m := decodeLine.FindStringSubmatch(says); m != nil {
				line, _ = strconv.Atoi(m[1])
				says = strings.Replace(says, m[0], "", 1)
			}
			if fe.Unknown != "" {
				says = fmt.Sprintf("%q is no key wiki.yaml reads here", fe.Unknown)
			}
			return Schema{}, &SchemaError{File: file, Line: line, Says: says}
		}
		return Schema{}, &SchemaError{File: file, Says: strings.TrimPrefix(err.Error(), file+" ")}
	}
	refuse := func(line int, says string, args ...any) error {
		return &SchemaError{File: file, Line: line, Says: fmt.Sprintf(says, args...)}
	}
	s := Schema{File: file, Kinds: map[string]Kind{}}
	if len(w.Floors) != floorCount {
		return Schema{}, refuse(w.Lines["floors"], "%d floors, a corpus has %d: intention, product, domains, architecture, code", len(w.Floors), floorCount)
	}
	for i, f := range w.Floors {
		switch {
		case !seatID.MatchString(f.ID):
			return Schema{}, refuse(f.Line, "floor %d has no id: lower-case words joined by dashes", i)
		case slices.ContainsFunc(s.Floors, func(o Floor) bool { return o.ID == f.ID }):
			return Schema{}, refuse(f.Line, "floor %q is declared twice", f.ID)
		case f.Name == "" || f.Question == "":
			return Schema{}, refuse(f.Line, "floor %q names no name or no question: a floor is the question it answers", f.ID)
		}
		s.Floors = append(s.Floors, Floor{ID: f.ID, Name: f.Name, Question: f.Question})
	}
	for _, f := range w.FloorOf {
		switch {
		case !strings.HasSuffix(f.Path, "/") || strings.HasPrefix(f.Path, "/"):
			return Schema{}, refuse(f.Line, "floor_of %q is no folder: a folder of the corpus, ending in /", f.Path)
		case f.Floor < 0 || f.Floor >= floorCount:
			return Schema{}, refuse(f.Line, "floor_of %s is %d: a floor is 0 to %d", f.Path, f.Floor, floorCount-1)
		}
		s.FloorOf = append(s.FloorOf, FloorOf{Path: f.Path, Floor: f.Floor})
	}
	if len(w.Seats) == 0 {
		return Schema{}, refuse(w.Lines["seats"], "no seat: a corpus declares who reads it")
	}
	var paths []string
	for _, seat := range w.Seats {
		switch {
		case !seatID.MatchString(seat.ID):
			return Schema{}, refuse(seat.Line, "seat %q is no id: lower-case words joined by dashes", seat.ID)
		case slices.ContainsFunc(s.Seats, func(o Seat) bool { return o.ID == seat.ID }):
			return Schema{}, refuse(seat.Line, "seat %q is declared twice", seat.ID)
		case seat.Name == "":
			return Schema{}, refuse(seat.Line, "seat %q has no name", seat.ID)
		case len(seat.Path) == 0:
			return Schema{}, refuse(seat.Line, "seat %q names no path: a seat holds the pages under its folders", seat.ID)
		case seat.Home == "":
			return Schema{}, refuse(seat.Line, "seat %q names no home: the page it opens on", seat.ID)
		}
		out := Seat{ID: seat.ID, Name: seat.Name, Paths: seat.Path, Home: seat.Home, Plan: seat.Plan, Line: seat.Line}
		for _, p := range seat.Path {
			if !strings.HasSuffix(p, "/") || strings.HasPrefix(p, "/") {
				return Schema{}, refuse(seat.Line, "seat %q: path %q is no folder: a folder of the corpus, ending in /", seat.ID, p)
			}
			for _, other := range paths {
				if strings.HasPrefix(p, other) || strings.HasPrefix(other, p) {
					return Schema{}, refuse(seat.Line, "seat paths %s and %s overlap", other, p)
				}
			}
			paths = append(paths, p)
		}
		for _, r := range seat.Reads {
			if !slices.Contains(intents(), Intent(r)) {
				return Schema{}, refuse(seat.Line, "seat %q reads %q, which is no intent: tell, define, decide, locate, guard or prove", seat.ID, r)
			}
			out.Reads = append(out.Reads, Intent(r))
		}
		s.Seats = append(s.Seats, out)
	}
	for _, r := range w.Repos {
		if !seatID.MatchString(r.Name) || r.Path == "" {
			return Schema{}, refuse(r.Line, "repos: %q is a clone's name and its path", r.Name)
		}
		s.Repos = append(s.Repos, Repo{Name: r.Name, Path: r.Path})
	}
	for _, m := range w.Mounts {
		switch {
		case !seatID.MatchString(m.At):
			return Schema{}, refuse(m.Line, "mount at %q: a mount sits at one folder of the corpus, its name alone", m.At)
		case !slices.ContainsFunc(s.Repos, func(r Repo) bool { return r.Name == m.Repo }):
			return Schema{}, refuse(m.Line, "mount %s reads the repository %q, which repos does not name", m.At, m.Repo)
		case m.Path == "":
			return Schema{}, refuse(m.Line, "mount %s names no path", m.At)
		case slices.ContainsFunc(s.Mounts, func(o Mount) bool { return o.At == m.At }):
			return Schema{}, refuse(m.Line, "two mounts sit at %s", m.At)
		}
		s.Mounts = append(s.Mounts, Mount{At: m.At, Repo: m.Repo, Path: m.Path})
	}
	s.Budget = Budget(w.Budget)
	if !slices.Contains(budgets(), s.Budget) {
		return Schema{}, refuse(w.Lines["budget"], "budget %q does not exist: 30s, 2min or 10min", w.Budget)
	}
	if n := len(contentKinds()) + len(w.Kinds); n > maxContentKinds {
		return Schema{}, refuse(w.Lines["kinds"], "%d kinds of the corpus and %d of the engine make %d content kinds, %d at most", len(w.Kinds), len(contentKinds()), n, maxContentKinds)
	}
	names := make([]string, 0, len(w.Kinds))
	for _, k := range w.Kinds {
		names = append(names, k.ID)
	}
	for _, k := range w.Kinds {
		kind, err := readKind(k, names, refuse)
		if err != nil {
			return Schema{}, err
		}
		s.Kinds[k.ID] = kind
	}
	for _, i := range w.Icons {
		if !seatID.MatchString(i.Name) {
			return Schema{}, refuse(i.Line, "icon %q is no name: lower-case words joined by dashes", i.Name)
		}
		s.Icons = append(s.Icons, Icon{Name: i.Name, Rows: i.Rows, Line: i.Line})
	}
	return s, nil
}

func readKind(k catalog.WikiKind, corpus []string, refuse func(int, string, ...any) error) (Kind, error) {
	if !kindID.MatchString(k.ID) {
		return Kind{}, refuse(k.Line, "kind %q is no id: lower-case letters and digits, its tag the id capitalised", k.ID)
	}
	if _, engine := builtinKinds()[k.ID]; engine {
		return Kind{}, refuse(k.Line, "kind %s is the engine's: a corpus declares kinds of its own", k.ID)
	}
	if k.Weighs < 0 || k.Weighs > 2 {
		return Kind{}, refuse(k.Line, "kind %s weighs %d: 0, 1 or 2", k.ID, k.Weighs)
	}
	kind := Kind{ID: k.ID, Title: k.Title, Icon: k.Icon, Weighs: k.Weighs, State: k.State, Faces: FaceTemplates(k.Faces), Line: k.Line}
	targets := append(contentKinds(), corpus...)
	for _, f := range k.Fields {
		switch {
		case !fieldID.MatchString(f.Name):
			return Kind{}, refuse(f.Line, "kind %s: field %q is no name: lower-case letters, digits and _", k.ID, f.Name)
		case slices.Contains(commonFields(), f.Name):
			return Kind{}, refuse(f.Line, "kind %s: %s is a field every kind takes, never declared", k.ID, f.Name)
		case !typeKnown(f.Type, corpusTypes()):
			return Kind{}, refuse(f.Line, "kind %s: field %s's type %q is none of %s, list<T> or a union a|b", k.ID, f.Name, f.Type, strings.Join(corpusTypes(), ", "))
		case hasBase(f.Type, "enum") != (len(f.Values) > 0):
			return Kind{}, refuse(f.Line, "kind %s: field %s: an enum names its values, and only an enum does", k.ID, f.Name)
		case f.Max != 0 && (f.Max < 0 || !hasBase(f.Type, "sentence")):
			return Kind{}, refuse(f.Line, "kind %s: field %s: max caps a sentence, at 1 character or more", k.ID, f.Name)
		case f.To != "" && !hasBase(f.Type, "ref"):
			return Kind{}, refuse(f.Line, "kind %s: field %s: to narrows a ref, and the field is no ref", k.ID, f.Name)
		case f.To != "" && !slices.Contains(targets, f.To):
			return Kind{}, refuse(f.Line, "kind %s: field %s points to %q, which is no kind", k.ID, f.Name, f.To)
		}
		kind.Fields = append(kind.Fields, Field{Name: f.Name, Type: f.Type, Required: f.Required, Max: f.Max, Values: f.Values, To: f.To})
	}
	if k.State != "" {
		if f, ok := kind.field(k.State); !ok || f.Type != "enum" {
			return Kind{}, refuse(k.Line, "kind %s: its state is %q, which is no enum field of it", k.ID, k.State)
		}
	}
	for _, t := range []string{k.Faces.Word, k.Faces.Line, k.Faces.Icon} {
		for _, m := range placeholder.FindAllStringSubmatch(t, -1) {
			if _, ok := kind.field(m[1]); !ok && m[1] != "id" {
				return Kind{}, refuse(k.Line, "kind %s: the face %q names {%s}, which is no field of it", k.ID, t, m[1])
			}
		}
	}
	return kind, nil
}

// value: whether a type is a base of bases, a list of one, or a union of
// them: `ref|anchor|sentence`, `list<ref>`, `list<ref|anchor>`.
func typeKnown(t string, bases []string) bool {
	if inner, ok := listOf(t); ok {
		_, nested := listOf(inner)
		return !nested && typeKnown(inner, bases)
	}
	for _, b := range strings.Split(t, "|") {
		if !slices.Contains(bases, b) {
			return false
		}
	}
	return true
}

func listOf(t string) (string, bool) {
	if strings.HasPrefix(t, "list<") && strings.HasSuffix(t, ">") {
		return t[len("list<") : len(t)-1], true
	}
	return "", false
}

func hasBase(t, base string) bool {
	if inner, ok := listOf(t); ok {
		t = inner
	}
	return slices.Contains(strings.Split(t, "|"), base)
}
