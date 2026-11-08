package agents

import (
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
)

func Judge(a Agent, events ...Event) Agent {
	// invariant: the hook's events fold, in order, into what the panel says
	// of an agent; nothing an event carries is kept but what Scrub lets
	// through, a step's verb, its relative path and its counts.
	for _, e := range events {
		a = fold(a, e)
	}
	return a
}

func fold(a Agent, e Event) Agent {
	if a.Session == "" {
		a.Session = e.Session
	}
	if a.Name == "" {
		a.Name = e.Name
	}
	if a.App == "" {
		a.App = e.App
	}
	if a.Folder == "" {
		a.Folder = e.Folder
	}
	if e.Terminal != "" {
		a.Terminal = e.Terminal
	}
	if !e.At.IsZero() {
		a.Updated = e.At
		if a.Since.IsZero() {
			a.Since = e.At
		}
	}
	switch e.Kind {
	case Began:
		a.State, a.Ended, a.Reason = Starting, time.Time{}, ""
		if a.Task == "" {
			a.Task, _ = Scrub(e.Prompt)
		}
	case Prompted:
		a.State, a.Ask = Working, nil
		if a.Task == "" {
			a.Task, _ = Scrub(e.Prompt)
		}
	case UseStarting:
		a.State, a.Ask = Working, nil
		if s, ok := stepOf(a.Folder, e); ok {
			a.Steps = append(a.Steps, s)
			if len(a.Steps) > catalog.AgentStepsKept {
				a.Steps = a.Steps[len(a.Steps)-catalog.AgentStepsKept:]
			}
		}
	case UseDone:
		a.State = Working
		a = counted(a, e)
	case Notified:
		title, _ := Scrub(e.Message)
		a.State, a.Ask = Waiting, &Ask{Title: title, At: e.At}
	case Stopped, TurnDone:
		title, detail := Scrub(e.Message)
		a.State, a.Ask = Waiting, &Ask{Title: title, Detail: detail, At: e.At}
		if e.Change != nil {
			c := *e.Change
			a.Change = &c
		}
	case Ended:
		a.State, a.Ended = Finished, e.At
		a.Reason, _ = Scrub(e.Reason)
		a.Ask = nil
	}
	return a
}

func stepOf(folder string, e Event) (Step, bool) {
	var verb catalog.AgentVerb
	switch e.Act {
	case ActEdit:
		verb = catalog.AgentEdit
	case ActRead:
		verb = catalog.AgentRead
	case ActRun:
		verb = Classify(e.Command)
	case ActWeb:
		verb = catalog.AgentWeb
	case ActDelegate:
		verb = catalog.AgentDelegate
	default:
		return Step{}, false
	}
	s := Step{At: e.At, Verb: verb}
	if verb == catalog.AgentEdit || verb == catalog.AgentRead {
		s.Object = relative(folder, e.Path)
	}
	return s, true
}

func relative(folder, p string) string {
	// invariant: a step's path is said relative to the agent's folder; a
	// path outside it is said by its name alone, and one shaped like a
	// secret is not said.
	if p == "" {
		return ""
	}
	p = path.Clean(p)
	if folder != "" && path.IsAbs(p) {
		f := path.Clean(folder)
		if rest, ok := strings.CutPrefix(p, f+"/"); ok {
			p = rest
		} else {
			p = path.Base(p)
		}
	}
	if strings.HasPrefix(p, "../") {
		p = path.Base(p)
	}
	if _, redacted := accounts.Redact(p); redacted {
		return ""
	}
	return p
}

func Classify(command string) catalog.AgentVerb {
	// invariant: a shell command is said by its class alone: the first class
	// of catalog.AgentCommands whose words open one of the command's parts,
	// else running a command; the command itself is never kept.
	parts := commandParts(command)
	for _, class := range catalog.AgentCommands {
		for _, part := range parts {
			for _, begins := range class.Begins {
				if part == begins || strings.HasPrefix(part, begins+" ") {
					return class.Verb
				}
			}
		}
	}
	return catalog.AgentCommand
}

func commandParts(command string) []string {
	var out []string
	for _, line := range strings.FieldsFunc(command, func(r rune) bool { return r == '\n' || r == ';' || r == '|' || r == '&' || r == '(' || r == ')' }) {
		words := strings.Fields(line)
		for len(words) > 0 && (strings.Contains(words[0], "=") || words[0] == "time" || words[0] == "env" || words[0] == "exec") {
			words = words[1:]
		}
		if len(words) > 0 {
			out = append(out, strings.Join(words, " "))
		}
	}
	return out
}

func counted(a Agent, e Event) Agent {
	if len(a.Steps) == 0 {
		return a
	}
	last := &a.Steps[len(a.Steps)-1]
	if last.Verb == catalog.AgentEdit && (e.Added > 0 || e.Removed > 0) {
		d := Diff{Plus: e.Added, Minus: e.Removed}
		if last.Diff != nil {
			d.Plus, d.Minus = d.Plus+last.Diff.Plus, d.Minus+last.Diff.Minus
		}
		last.Diff = &d
	}
	if t, ok := TestCounts(e.Output); ok && last.Verb == catalog.AgentTest {
		last.Tests = &t
	}
	if pr, ok := PullRequest(e.Output); ok && last.Verb == catalog.AgentOpenPR {
		a.PR = &pr
	}
	return a
}

func TestCounts(output string) (Tests, bool) {
	// invariant: a test run's counts are read from its runner's summary by
	// catalog.AgentTestSummaries, the first runner that says any; a run no
	// runner's summary matches has no counts.
	for _, summary := range catalog.AgentTestSummaries {
		re := regexp.MustCompile(summary.Pattern)
		matches := re.FindAllStringSubmatch(output, -1)
		if len(matches) == 0 {
			continue
		}
		var pass, fail, total int
		for _, m := range matches {
			for i, name := range re.SubexpNames() {
				n, _ := strconv.Atoi(m[i])
				switch {
				case m[i] == "":
				case name == "pass":
					pass += n
				case name == "fail":
					fail += n
				case name == "total":
					total += n
				case name == "one_pass":
					pass++
				case name == "one_fail":
					fail++
				}
			}
		}
		if total == 0 {
			total = pass + fail
		}
		return Tests{Pass: pass, Total: total}, true
	}
	return Tests{}, false
}

func PullRequest(output string) (PR, bool) {
	// invariant: the PR a forge opened is the last PR URL its output prints.
	matches := regexp.MustCompile(catalog.AgentPullRequest).FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return PR{}, false
	}
	m := matches[len(matches)-1]
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return PR{}, false
	}
	return PR{Number: n, URL: m[0]}, true
}
