package cliproject

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"spwn.sh/packages/platform"
)

// NotGrantedCode is the code of the refusal of a host path outside the
// project that no answer has granted.
const NotGrantedCode = "outside.not-granted"

// Reach is a host path spwn.yaml names that leaves the project, resolved
// through its links and absolute, with the access it would be given and
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
		lines = append(lines, fmt.Sprintf("%s (%s)", r.Path, r.Grants))
	}
	return fmt.Sprintf("%s: spwn.yaml names a host path outside the project %s, and no answer grants it: %s", NotGrantedCode, e.Root, strings.Join(lines, "; "))
}

// Code is NotGrantedCode.
func (e *NotGrantedError) Code() string { return NotGrantedCode }

// Resolve is p, absolute against root when relative, through every link
// of its longest part that exists, the missing rest joined by name, and
// whether it leaves root's own resolved folder. A link that does not
// resolve leaves the project, since where it points is not proved.
func Resolve(root, p string) (resolved string, outside bool) {
	tree := platform.Within(root)
	defer tree.Close()
	abs := filepath.Clean(p)
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, p)
	}
	rest := ""
	for at := abs; ; at = filepath.Dir(at) {
		if _, err := os.Lstat(at); err == nil {
			resolved, err := filepath.EvalSymlinks(at)
			if err != nil {
				return abs, true
			}
			resolved = filepath.Join(resolved, rest)
			return resolved, !tree.Holds(resolved)
		}
		if filepath.Dir(at) == at {
			return abs, !tree.Holds(abs)
		}
		rest = filepath.Join(filepath.Base(at), rest)
	}
}

type grant struct {
	Project string `yaml:"project"`
	Path    string `yaml:"path"`
	Access  string `yaml:"access"`
}

func grantsPath() string { return filepath.Join(platform.UserDir(), "grants.yaml") }

func readGrants() []grant {
	data, err := os.ReadFile(grantsPath())
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
	return os.WriteFile(grantsPath(), data, 0o600)
}

// StdinIsTerminal reports whether the answer can be asked of a person.
func StdinIsTerminal() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// Consent grants the reaches of the project at root: one already
// granted for this project, path and access is taken silently; the rest
// are shown once and asked, on a terminal, or granted by yes; a yes is
// kept in the user's own grants (~/.spwn/grants.yaml), never in the
// project. Off a terminal and without yes it refuses with a
// NotGrantedError; a no refuses too.
func Consent(root string, reaches []Reach, yes, terminal bool, in io.Reader, out io.Writer) error {
	grants := readGrants()
	var asked []Reach
	for _, r := range reaches {
		if !slices.Contains(grants, grant{Project: projectKey(root), Path: r.Path, Access: r.Access}) {
			asked = append(asked, r)
		}
	}
	if len(asked) == 0 {
		return nil
	}
	if !yes {
		if !terminal {
			return &NotGrantedError{Root: projectKey(root), Reaches: asked}
		}
		fmt.Fprintf(out, "\n  spwn.yaml names host paths outside the project %s:\n", projectKey(root))
		for _, r := range asked {
			fmt.Fprintf(out, "    %s\n      %s\n", r.Path, r.Grants)
		}
		fmt.Fprintf(out, "  The world's agents run what this project gives them, with this access.\n")
		fmt.Fprintf(out, "  Allow them for this project? The answer is kept in %s. [y/N] ", grantsPath())
		line, _ := bufio.NewReader(in).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
			return errors.New("the host paths outside the project were not allowed: nothing was mounted")
		}
	}
	return rememberGrants(root, asked)
}
