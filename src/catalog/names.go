package catalog

import (
	"regexp"
	"strings"
)

// invariant: a name an item gives to a program run as root, or to a path
// station writes or removes, is checked by one rule of its kind, here:
// the decoder refuses an item that breaks it, and every installer that
// puts the name in an argv or a path checks it again before, since an
// install also crosses the wire from a peer and a record is read back
// from the disk.
var (
	packageRule        = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]+(:[a-z0-9-]+)?$`)
	packageVersionRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+~:-]*$`)
	repoRule           = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$`)
)

type NameError struct {
	// value: the kind of name: `package name`, `package version`, `command name`, `repo`.
	Kind string
	Name string
	// value: the rule it breaks, in the owner's words.
	Rule string
}

func (e *NameError) Error() string {
	return "\"" + e.Name + "\" is no " + e.Kind + ": " + e.Rule
}

func CheckPackage(name string) error {
	// value: a system package's name: lowercase letters, digits and `+.-`,
	// opening with a letter or digit, two characters at least, an `:arch`
	// after it at most; never an option.
	if !packageRule.MatchString(name) {
		return &NameError{Kind: "package name", Name: name, Rule: "lowercase letters, digits and +.- from a letter or digit, an :arch at most"}
	}
	return nil
}

func CheckPackageVersion(version string) error {
	// value: the version a package is pinned at: letters, digits and `.+~:-`,
	// opening with a letter or digit; never a space nor an option.
	if !packageVersionRule.MatchString(version) {
		return &NameError{Kind: "package version", Name: version, Rule: "letters, digits and .+~:- from a letter or digit"}
	}
	return nil
}

func CheckCommand(name string) error {
	// value: a command an app puts on the search path: one path segment,
	// never `.` nor `..`, never an option.
	switch {
	case name == "", name == ".", name == "..":
		return &NameError{Kind: "command name", Name: name, Rule: "one file name, not . or .."}
	case strings.ContainsAny(name, "/\\\x00"):
		return &NameError{Kind: "command name", Name: name, Rule: "one file name, no slash"}
	case strings.HasPrefix(name, "-"):
		return &NameError{Kind: "command name", Name: name, Rule: "one file name, not opening with -"}
	}
	return nil
}

func CheckRepo(repo string) error {
	// value: a hosted repository, `owner/name`: the owner letters, digits and
	// `-`, the name letters, digits and `._-`, never `.` nor `..`.
	_, name, _ := strings.Cut(repo, "/")
	if !repoRule.MatchString(repo) || name == "." || name == ".." {
		return &NameError{Kind: "repo", Name: repo, Rule: "owner/name, letters, digits and -, the name ._- too, never . or .."}
	}
	return nil
}
