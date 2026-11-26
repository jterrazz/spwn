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

	"golang.org/x/term"
	"gopkg.in/yaml.v3"

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

// Code is ControlCode or TildeCode.
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

// CheckHostPath refuses a host path of spwn.yaml that CheckText refuses,
// or that starts with ~: spwn.yaml never expands it.
func CheckHostPath(what, p string) error {
	if err := CheckText(what, p); err != nil {
		return err
	}
	if strings.HasPrefix(strings.TrimSpace(p), "~") {
		return &DeclaredError{code: TildeCode, msg: fmt.Sprintf("%s %q starts with ~, which spwn.yaml never expands: write the absolute path (one outside the project is asked for once)", what, p)}
	}
	return nil
}

// Reach is a host path spwn.yaml names that leaves the project, as far as
// the project's own links tell, with the access it would be given and
// what it grants, in words.
type Reach struct {
	Path   string
	Access string
	Grants string
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

// StdinIsTerminal reports whether the answer can be asked of a person.
func StdinIsTerminal() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// Allowed is what Consent let through: the reaches a typed yes allowed,
// kept only once the run they were asked for has succeeded.
type Allowed struct {
	root  string
	typed []Reach
}

// Keep writes the reaches a typed yes allowed into the user's grants
// file; reaches allowed by --yes, or already kept, write nothing.
func (a Allowed) Keep() error {
	if len(a.typed) == 0 {
		return nil
	}
	return rememberGrants(a.root, a.typed)
}

// Consent allows the reaches of the project at root: one already kept for
// this project folder, path and access passes silently; with yes the rest
// are allowed for this run only and each is printed to out, never kept;
// on a terminal the rest are shown once and asked, and a typed yes is
// returned to be kept once the run succeeds. Off a terminal and without
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
			fmt.Fprintf(out, "  allowed for this run: %q (%s)\n", r.Path, r.Grants)
		}
		return Allowed{}, nil
	case !terminal:
		return Allowed{}, &NotGrantedError{Root: projectKey(root), Reaches: asked}
	}
	fmt.Fprintf(out, "\n  spwn.yaml names host paths outside the project %q:\n", projectKey(root))
	for _, r := range asked {
		fmt.Fprintf(out, "    %q\n      %s\n", r.Path, r.Grants)
	}
	fmt.Fprintf(out, "  The world's agents run what this project gives them, with this access.\n")
	fmt.Fprintf(out, "  Allow them for this project folder? A yes is kept in %s. [y/N] ", GrantsPath())
	line, _ := bufio.NewReader(in).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
		return Allowed{}, errors.New("the host paths outside the project were not allowed: nothing was mounted")
	}
	return Allowed{root: root, typed: asked}, nil
}
