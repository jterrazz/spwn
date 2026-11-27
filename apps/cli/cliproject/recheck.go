package cliproject

import "fmt"

// ChangedCode is the code of the refusal of a world whose declared host
// path resolves, just before its container is made or once it is started,
// to another path than the one resolved, and asked for, when the run began.
const ChangedCode = "outside.changed-while-starting"

// Declared is a host path as spwn.yaml writes it and the path Resolve
// gave it when the run began.
type Declared struct {
	Text     string
	Resolved string
}

// ChangedError refuses a declared path that resolves differently now.
type ChangedError struct {
	Text, Was, Now string
}

func (e *ChangedError) Error() string {
	return fmt.Sprintf("%s: the host path %q of spwn.yaml resolved to %q when the run began and to %q now: a link of the project changed while starting, so this world is not started", ChangedCode, e.Text, e.Was, e.Now)
}

// Code is ChangedCode.
func (e *ChangedError) Code() string { return ChangedCode }

// Recheck resolves each declared path of the project at root again and
// refuses the first that no longer resolves to the path it had: a world of
// the same project, already running, may rewrite the project's links
// between the run's resolve and the mount.
func Recheck(root string, declared []Declared) error {
	for _, d := range declared {
		if now, _ := Resolve(root, d.Text); now != d.Resolved {
			return &ChangedError{Text: d.Text, Was: d.Resolved, Now: now}
		}
	}
	return nil
}
