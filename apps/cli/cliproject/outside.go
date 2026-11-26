package cliproject

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/platform"
)

// NotGrantedCode is the code of the refusal of a host path outside the
// project that no answer has granted.
const NotGrantedCode = "outside.not-granted"

// ControlCode is the code of the refusal of a name or a path of spwn.yaml
// that holds a control character.
const ControlCode = "spwn-yaml.control-character"

// TildeCode is the code of the refusal of a host path of spwn.yaml that
// starts with ~.
const TildeCode = "spwn-yaml.tilde"

const maxLinks = 40

// DeclaredError refuses a name or a host path of spwn.yaml before any
// reach is built from it or shown.
type DeclaredError struct {
	code string
	msg  string
}

func (e *DeclaredError) Error() string { return e.code + ": " + e.msg }

// Code is ControlCode, TildeCode or ColonCode.
func (e *DeclaredError) Code() string { return e.code }

// CheckText refuses a name or a path of spwn.yaml that holds a control
// character (below 0x20, 0x7f, or the C1 range) or is not UTF-8, so no
// text the project writes can move the cursor or rewrite a line of what
// spwn prints.
func CheckText(what, s string) error {
	bad := !utf8.ValidString(s)
	for _, r := range s {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			bad = true
		}
	}
	if bad {
		return &DeclaredError{code: ControlCode, msg: fmt.Sprintf("%s %q holds a control character: a name or a path of spwn.yaml holds none", what, s)}
	}
	return nil
}

// ColonCode is the code of the refusal of a host path that holds a colon
// once spwn.yaml's `name=path:ro` grammar is read off it.
const ColonCode = "spwn-yaml.colon"

// CheckHostPath refuses a host path of spwn.yaml that CheckText refuses,
// that starts with ~ (spwn.yaml never expands it), or that holds a `:`:
// Docker reads a bind as `source:target:mode`, so a colon in the source
// would bind a prefix of the path that was resolved.
func CheckHostPath(what, p string) error {
	if err := CheckText(what, p); err != nil {
		return err
	}
	if strings.HasPrefix(strings.TrimSpace(p), "~") {
		return &DeclaredError{code: TildeCode, msg: fmt.Sprintf("%s %q starts with ~, which spwn.yaml never expands: write the absolute path (one outside the project is asked for once)", what, p)}
	}
	if strings.Contains(p, ":") {
		return &DeclaredError{code: ColonCode, msg: fmt.Sprintf("%s %q holds a colon, which a mount's source cannot: only `name=path:ro` ends a workspace with one", what, p)}
	}
	return nil
}

// ResolveHostPath checks a host path of spwn.yaml, resolves it against the
// project at root, and refuses what it resolved to if it holds a colon,
// since a link's text of the project can bring one the declared text did
// not hold.
func ResolveHostPath(root, what, p string) (resolved string, outside bool, err error) {
	if err := CheckHostPath(what, p); err != nil {
		return "", false, err
	}
	resolved, outside = Resolve(root, p)
	if strings.Contains(resolved, ":") {
		return "", false, &DeclaredError{code: ColonCode, msg: fmt.Sprintf("%s %q resolved %q through the project's links, which holds a colon, which a mount's source cannot", what, p, resolved)}
	}
	return resolved, outside, nil
}

// Reach is a host path spwn.yaml names that leaves the project, as far as
// the project's own links tell, with the access it would be given and
// what it grants, in words.
type Reach struct {
	Path   string
	Access string
	Grants string
	World  string
}

// NotGrantedError refuses the reaches no answer has granted, off a
// terminal.
type NotGrantedError struct {
	Root    string
	Reaches []Reach
}

func (e *NotGrantedError) Error() string {
	lines := make([]string, 0, len(e.Reaches))
	for _, r := range e.Reaches {
		lines = append(lines, fmt.Sprintf("%q (%s)", r.Path, r.Grants))
	}
	return fmt.Sprintf("%s: spwn.yaml names a host path outside the project %q, and no answer grants it: %s", NotGrantedCode, e.Root, strings.Join(lines, "; "))
}

// Code is NotGrantedCode.
func (e *NotGrantedError) Code() string { return NotGrantedCode }

// Resolve is p, absolute against root when relative and cleaned by name,
// through the links the project itself holds, and whether it leaves
// root. Each link is read inside the project through its Tree, never
// followed by the system; the first path that leaves the project is
// returned as it reads, and nothing outside it is opened, stat'd or
// read. A name that does not exist is joined by name.
func Resolve(root, p string) (resolved string, outside bool) {
	tree := platform.Within(root)
	defer tree.Close()
	at := filepath.Clean(p)
	if !filepath.IsAbs(at) {
		at = filepath.Join(tree.Dir(), at)
	}
	for range maxLinks {
		rel, err := tree.Rel(at)
		if err != nil {
			return at, true
		}
		next, linked, err := throughFirstLink(tree, rel)
		if err != nil {
			return at, true
		}
		if !linked {
			return filepath.Join(tree.Dir(), rel), false
		}
		at = next
	}
	return at, true
}

func throughFirstLink(tree *platform.Tree, rel string) (string, bool, error) {
	parts := strings.Split(rel, string(filepath.Separator))
	for i := range parts {
		name := filepath.Join(parts[:i+1]...)
		info, err := tree.Lstat(name)
		if err != nil {
			return "", false, nil
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			continue
		}
		target, err := tree.Readlink(name)
		if err != nil {
			return "", false, err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(tree.Dir(), filepath.Dir(name), target)
		}
		return filepath.Join(append([]string{target}, parts[i+1:]...)...), true, nil
	}
	return "", false, nil
}

type grant struct {
	Project string `yaml:"project"`
	Path    string `yaml:"path"`
	Access  string `yaml:"access"`
}

// GrantsPath is the user's own file of the kept answers, one entry per
// project folder, path and access; deleting an entry revokes it.
func GrantsPath() string { return filepath.Join(platform.UserDir(), "grants.yaml") }

func readGrants() []grant {
	data, err := os.ReadFile(GrantsPath())
	if err != nil {
		return nil
	}
	var grants []grant
	if yaml.Unmarshal(data, &grants) != nil {
		return nil
	}
	return grants
}

func projectKey(root string) string {
	if r, err := filepath.EvalSymlinks(root); err == nil {
		return r
	}
	return filepath.Clean(root)
}

func rememberGrants(root string, reaches []Reach) error {
	grants := readGrants()
	for _, r := range reaches {
		g := grant{Project: projectKey(root), Path: r.Path, Access: r.Access}
		if !slices.Contains(grants, g) {
			grants = append(grants, g)
		}
	}
	data, err := yaml.Marshal(grants)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(platform.UserDir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(GrantsPath(), data, 0o600)
}


// Allowed is what Consent let through: the reaches a typed yes allowed,
// kept only once the run they were asked for has succeeded.
type Allowed struct {
	root  string
	typed []Reach
}

// Keep writes the reaches a typed yes allowed that are among of into the
// user's grants file: a run asked once for several worlds keeps each
// world's own reaches once that world has started. Reaches allowed by
// --yes, or already kept, write nothing.
func (a Allowed) Keep(of []Reach) error {
	var kept []Reach
	for _, r := range a.typed {
		if slices.Contains(of, r) {
			kept = append(kept, r)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return rememberGrants(a.root, kept)
}

func agentsOf(asked []Reach) string {
	var worlds []string
	for _, r := range asked {
		if !slices.Contains(worlds, r.World) {
			worlds = append(worlds, r.World)
		}
	}
	if len(worlds) > 1 {
		return "These worlds' agents"
	}
	return "The world's agents"
}

// Consent allows the reaches of the project at root: one already kept for
// this project folder, path and access passes silently; with yes the rest
// are allowed for this run only and each is printed to out, never kept;
// on a terminal the rest are shown once and asked, what was typed before
// the ask discarded when in is a terminal, and a typed yes is returned to
// be kept once the run succeeds. Off a terminal and without
// yes it refuses with a NotGrantedError; a no refuses too.
func Consent(root string, reaches []Reach, yes, terminal bool, in io.Reader, out io.Writer) (Allowed, error) {
	grants := readGrants()
	var asked []Reach
	for _, r := range reaches {
		if !slices.Contains(grants, grant{Project: projectKey(root), Path: r.Path, Access: r.Access}) {
			asked = append(asked, r)
		}
	}
	switch {
	case len(asked) == 0:
		return Allowed{}, nil
	case yes:
		for _, r := range asked {
			fmt.Fprintf(out, "  allowed for this run: %q (%s)\n", r.Path, ui.Clean(r.Grants))
		}
		return Allowed{}, nil
	case !terminal:
		return Allowed{}, &NotGrantedError{Root: projectKey(root), Reaches: asked}
	}
	if f, ok := in.(*os.File); ok {
		if err := flushInput(int(f.Fd())); err != nil {
			return Allowed{}, fmt.Errorf("the host paths outside the project were not allowed: what was typed before the ask could not be discarded: %w", err)
		}
	}
	fmt.Fprintf(out, "\n  spwn.yaml names host paths outside the project %q:\n", projectKey(root))
	for _, r := range asked {
		fmt.Fprintf(out, "    %q\n      %s\n", r.Path, ui.Clean(r.Grants))
	}
	fmt.Fprintf(out, "  %s run what this project gives them, with this access.\n", ui.Clean(agentsOf(asked)))
	fmt.Fprintf(out, "  Allow them for this project folder? A yes is kept in %s. [y/N] ", ui.Clean(GrantsPath()))
	line, _ := bufio.NewReader(in).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
		return Allowed{}, errors.New("the host paths outside the project were not allowed: nothing was mounted")
	}
	return Allowed{root: root, typed: asked}, nil
}
