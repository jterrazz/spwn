package automations

import (
	"errors"
	"slices"
	"strings"
	"time"
)

const (
	Cron = "cron"
	FS   = "fs"
)

type Trigger struct {
	// value: Cron or FS, the one trigger the project declares.
	Kind string
	// value: the cron expression, or the path watched, as declared.
	Spec string
}

type Automation struct {
	// value: the group of the manifest that declares the automation, which
	// qualifies its name when two groups declare one name.
	Group   string
	Name    string
	Agent   string
	Trigger Trigger
	// value: when the newest run of this automation started; zero when
	// none ran.
	LastFired time.Time
	Fires     int
	Failed    int
	// value: set by Mark, true when another group declares the same name.
	Shared bool
}

type Run struct {
	Group     string
	Name      string
	Scheduled time.Time
	At        time.Time
	OK        bool
	Error     string
	Output    string
}

var ErrNotFound = errors.New("no automation of that name")

type NotUniqueError struct {
	Name   string
	Groups []string
}

func (e *NotUniqueError) Error() string {
	return "the automation " + e.Name + " is declared by the groups " + strings.Join(e.Groups, ", ")
}

func Reserved(name string) bool {
	// invariant: `automations log` is the command that reads the runs, so
	// an automation of that name, unqualified, is listed and has no detail
	// to ask for.
	return name == "log"
}

func (a Automation) Qualified() string {
	if a.Group == "" {
		return a.Name
	}
	return a.Group + "/" + a.Name
}

func Mark(all []Automation) []Automation {
	// invariant: Shared is true for every automation whose name another
	// group declares too.
	count := map[string]int{}
	for _, a := range all {
		count[a.Name]++
	}
	out := slices.Clone(all)
	for i := range out {
		out[i].Shared = count[out[i].Name] > 1
	}
	return out
}

func Select(all []Automation, ref string) (Automation, error) {
	// invariant: a ref is a name, which two groups declaring it make not
	// unique, or `group/name`, which is always one.
	all = Mark(all)
	var named, qualified []Automation
	for _, a := range all {
		if a.Name == ref {
			named = append(named, a)
		}
		if a.Group != "" && a.Qualified() == ref {
			qualified = append(qualified, a)
		}
	}
	switch {
	case len(named) == 1:
		return named[0], nil
	case len(named) > 1:
		groups := make([]string, 0, len(named))
		for _, a := range named {
			groups = append(groups, a.Group)
		}
		return Automation{}, &NotUniqueError{Name: ref, Groups: groups}
	case len(qualified) == 1:
		return qualified[0], nil
	}
	return Automation{}, ErrNotFound
}

func (r Run) Qualified() string {
	if r.Group == "" {
		return r.Name
	}
	return r.Group + "/" + r.Name
}
