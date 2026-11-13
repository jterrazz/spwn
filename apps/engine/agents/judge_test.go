package agents_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func at(minute int) time.Time { return time.Date(2026, 11, 7, 9, minute, 0, 0, time.UTC) }

func begun() agents.Agent {
	return agents.Judge(agents.Agent{VM: "linux-1"}, agents.Event{Kind: agents.Began, App: "claudecode", Session: "s-1", Folder: "/Users/admin/app", At: at(0)})
}

func TestEachHookEventOfClaudeCodeTurnsTheAgentToItsState(t *testing.T) {
	cases := []struct {
		event string
		e     agents.Event
		want  agents.State
	}{
		{"SessionStart", agents.Event{Kind: agents.Began, At: at(1)}, agents.Starting},
		{"UserPromptSubmit", agents.Event{Kind: agents.Prompted, Prompt: "build the landing page", At: at(1)}, agents.Working},
		{"PreToolUse", agents.Event{Kind: agents.UseStarting, Act: agents.ActEdit, Path: "/Users/admin/app/src/a.ts", At: at(1)}, agents.Working},
		{"PostToolUse", agents.Event{Kind: agents.UseDone, Act: agents.ActEdit, Added: 3, At: at(1)}, agents.Working},
		{"Notification", agents.Event{Kind: agents.Notified, Message: "Claude needs your permission to use Bash", At: at(1)}, agents.Waiting},
		{"Stop", agents.Event{Kind: agents.Stopped, Message: "PR #42 is ready", At: at(1)}, agents.Waiting},
		{"SessionEnd", agents.Event{Kind: agents.Ended, Reason: "logout", At: at(1)}, agents.Finished},
	}
	for _, c := range cases {
		t.Run(c.event, func(t *testing.T) {
			if got := agents.Judge(begun(), c.e); got.State != c.want {
				t.Errorf("%s left the agent %s, want %s", c.event, got.State, c.want)
			}
		})
	}
}

func TestTheFirstPromptIsTheTaskAndALaterOneLeavesIt(t *testing.T) {
	a := agents.Judge(begun(),
		agents.Event{Kind: agents.Prompted, Prompt: "Build the landing page\nwith a pricing section, then open a PR", At: at(1)},
		agents.Event{Kind: agents.Prompted, Prompt: "also fix the footer", At: at(2)},
	)
	if a.Task != "Build the landing page" {
		t.Errorf("task %q, want the first prompt's first line", a.Task)
	}
	if !a.Since.Equal(at(0)) || !a.Updated.Equal(at(2)) {
		t.Errorf("since %v, updated %v; want the session's start and the last event", a.Since, a.Updated)
	}
}

func TestAStepKeepsAVerbARelativePathAndCountsOnly(t *testing.T) {
	a := agents.Judge(begun(),
		agents.Event{Kind: agents.UseStarting, Act: agents.ActRead, Path: "/Users/admin/app/src/auth/session.ts", At: at(1)},
		agents.Event{Kind: agents.UseStarting, Act: agents.ActEdit, Path: "/Users/admin/app/src/auth/refresh.ts", At: at(2)},
		agents.Event{Kind: agents.UseDone, Act: agents.ActEdit, Added: 86, Removed: 12, At: at(2)},
		agents.Event{Kind: agents.UseStarting, Act: agents.ActRun, Command: "cd web && API_TOKEN=x pytest -q", At: at(3)},
		agents.Event{Kind: agents.UseDone, Act: agents.ActRun, Output: " Test Files  2 failed | 9 passed (11)\n      Tests  17 failed | 41 passed (58)\n", At: at(4)},
	)
	if len(a.Steps) != catalog.AgentStepsKept {
		t.Fatalf("%d steps kept, want the last %d", len(a.Steps), catalog.AgentStepsKept)
	}
	if got := agents.Words(a); !slices.Equal(got, []string{"running tests", "41/58 pass"}) {
		t.Errorf("step line %q, want running tests, 41/58 pass", got)
	}
	if got := agents.Done(a.Steps[1]); !slices.Equal(got, []string{"edited src/auth/refresh.ts", "+86 −12"}) {
		t.Errorf("past step %q, want the edit by its relative path and its lines", got)
	}
	if got := agents.Done(a.Steps[0]); !slices.Equal(got, []string{"read src/auth/session.ts"}) {
		t.Errorf("past step %q", got)
	}
}

func TestAShellCommandIsSaidByItsClassAndNeverKept(t *testing.T) {
	for _, class := range catalog.AgentCommands {
		for _, begins := range class.Begins {
			for _, command := range []string{begins + " --flag value", "cd web && CI=1 " + begins, "ls -la; " + begins + " | tee out.log"} {
				if got := agents.Classify(command); got != class.Verb {
					t.Errorf("%q is %s, want %s", command, got, class.Verb)
				}
				a := agents.Judge(begun(), agents.Event{Kind: agents.UseStarting, Act: agents.ActRun, Command: command, At: at(1)})
				if len(a.Steps) != 1 || a.Steps[0].Object != "" {
					t.Errorf("%q kept %+v, want its class alone", command, a.Steps)
				}
			}
		}
	}
	if got := agents.Classify("ls -la ~/.ssh"); got != catalog.AgentCommand {
		t.Errorf("an unknown command is %s, want %s", got, catalog.AgentCommand)
	}
}

func TestEachRunnersSummaryGivesItsCounts(t *testing.T) {
	cases := map[string]agents.Tests{
		"=== RUN   TestA\n--- PASS: TestA (0.00s)\n--- FAIL: TestB (0.01s)\n--- PASS: TestC (0.00s)\nFAIL\n": {Pass: 2, Total: 3},
		"      Tests  41 passed (41)\n":                                                                         {Pass: 41, Total: 41},
		"Tests:       1 failed, 41 passed, 42 total\n":                                                          {Pass: 41, Total: 42},
		"============ 2 failed, 41 passed, 1 skipped in 1.20s ============\n":                                   {Pass: 41, Total: 43},
		"test result: ok. 12 passed; 0 failed; 0 ignored\ntest result: FAILED. 3 passed; 1 failed; 0 ignored\n": {Pass: 15, Total: 16},
	}
	for output, want := range cases {
		if got, ok := agents.TestCounts(output); !ok || got != want {
			t.Errorf("counts %+v %v of\n%s\nwant %+v", got, ok, output, want)
		}
	}
	if _, ok := agents.TestCounts("Compiling...\nall done\n"); ok {
		t.Error("an output with no summary gave counts")
	}
}

func opening() string {
	for _, class := range catalog.AgentCommands {
		if class.Verb == catalog.AgentOpenPR {
			return class.Begins[0]
		}
	}
	return ""
}

func TestAPRTheAgentOpenedIsKeptByItsNumberAndURL(t *testing.T) {
	a := agents.Judge(begun(),
		agents.Event{Kind: agents.UseStarting, Act: agents.ActRun, Command: opening() + " --fill", At: at(1)},
		agents.Event{Kind: agents.UseDone, Act: agents.ActRun, Output: "Creating pull request for feat/x into main\n\nhttps://github.com/jterrazz/review-bot/pull/42\n", At: at(2)},
	)
	if a.PR == nil || a.PR.Number != 42 || a.PR.URL != "https://github.com/jterrazz/review-bot/pull/42" {
		t.Errorf("PR %+v, want #42 with its URL", a.PR)
	}
}

func TestAStopAsksWithItsMessageAndItsChangeAndAPromptAnswersIt(t *testing.T) {
	stopped := agents.Judge(begun(), agents.Event{
		Kind:    agents.Stopped,
		Message: "PR #42 is ready\nI moved the refresh into its own module. Tests pass. The old helper is gone.",
		Change:  &agents.Change{Files: 3, Plus: 86, Minus: 12},
		At:      at(5),
	})
	if stopped.Ask == nil || stopped.Ask.Title != "PR #42 is ready" || stopped.Ask.Detail != "I moved the refresh into its own module." || !stopped.Ask.At.Equal(at(5)) {
		t.Fatalf("ask %+v, want the first line and one sentence, at the stop", stopped.Ask)
	}
	if stopped.Change == nil || *stopped.Change != (agents.Change{Files: 3, Plus: 86, Minus: 12}) {
		t.Errorf("change %+v", stopped.Change)
	}
	replied := agents.Judge(stopped, agents.Event{Kind: agents.Prompted, Prompt: "merge it", At: at(6)})
	if replied.State != agents.Working || replied.Ask != nil {
		t.Errorf("after a reply: %s, ask %+v; want working with nothing asked", replied.State, replied.Ask)
	}
}

func TestAnEndedSessionIsKeptSevenDays(t *testing.T) {
	a := agents.Judge(begun(), agents.Event{Kind: agents.Ended, Reason: "exit", At: at(10)})
	if a.Expired(at(10).Add(catalog.AgentKeptFor)) {
		t.Error("expired on its seventh day")
	}
	if !a.Expired(at(11).Add(catalog.AgentKeptFor)) {
		t.Error("kept past its seventh day")
	}
	if begun().Expired(at(0).AddDate(1, 0, 0)) {
		t.Error("a session that has not ended expired")
	}
}

func TestATurnOfAToolWithoutToolEventsWaits(t *testing.T) {
	a := agents.Judge(agents.Agent{}, agents.Event{Kind: agents.TurnDone, App: "turns", Session: "t-1", Message: "Done. The flaky test now waits for the server.", At: at(1)})
	if a.State != agents.Waiting || a.Ask == nil || !strings.HasPrefix(a.Ask.Title, "Done.") {
		t.Errorf("%s %+v, want waiting with its message", a.State, a.Ask)
	}
	if len(a.Steps) != 0 {
		t.Errorf("steps %+v, want none", a.Steps)
	}
}
