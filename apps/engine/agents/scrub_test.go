package agents_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestOnlyTheFirstLineAndOneMoreSentenceAreKept(t *testing.T) {
	title, detail := agents.Scrub("PR #42 is ready\n\nI split the refresh out. It now retries twice. The old helper is gone.\nAsk me anything.")
	if title != "PR #42 is ready" || detail != "I split the refresh out." {
		t.Errorf("kept %q / %q", title, detail)
	}
}

func TestAFencedBlockOrACodeShapedLineSaysRepliedWithCode(t *testing.T) {
	cases := map[string][2]string{
		"```go\nfunc main() { os.Exit(1) }\n```\nThat is the fix.":                 {catalog.AgentCodeSaid, "That is the fix."},
		"Here is the patch:\n```\nrm -rf build && make\n```":                       {"Here is the patch:", catalog.AgentCodeSaid},
		"const token = process.env.API_KEY ?? (await read('key'));":                {catalog.AgentCodeSaid, ""},
		"if (a && b) { return c; }\nThat guard was missing.":                       {catalog.AgentCodeSaid, "That guard was missing."},
		"~~~\nSELECT * FROM users;\n~~~":                                           {catalog.AgentCodeSaid, ""},
		"Fixed the bug in parse() and render().":                                   {"Fixed the bug in parse() and render().", ""},
		"```\nunterminated fence, everything after it is code\nsecret = hunter2\n": {catalog.AgentCodeSaid, ""},
	}
	for text, want := range cases {
		if title, detail := agents.Scrub(text); title != want[0] || detail != want[1] {
			t.Errorf("Scrub(%q) = %q / %q, want %q / %q", text, title, detail, want[0], want[1])
		}
	}
}

func TestAnInlineSpanKeepsAPathOrANameAndDropsTheRest(t *testing.T) {
	title, _ := agents.Scrub("Edited `src/auth/refresh.ts` and ran `make test -- --watch=false` then `refreshToken`.")
	if title != "Edited src/auth/refresh.ts and ran then refreshToken." {
		t.Errorf("kept %q", title)
	}
	long := strings.Repeat("a", catalog.AgentNameCap+1)
	if title, _ := agents.Scrub("Renamed `" + long + "` done."); strings.Contains(title, long) || strings.Contains(title, "`") {
		t.Errorf("a span past %d characters was kept: %q", catalog.AgentNameCap, title)
	}
}

func TestASecretIsScrubbedBeforeAnythingIsKept(t *testing.T) {
	planted := []string{
		"ghp_" + strings.Repeat("A1b2", 9),
		"github_pat_" + strings.Repeat("Z9", 20),
		"sk-ant-api03-" + strings.Repeat("x7", 24),
		"xoxb-1234-5678-" + strings.Repeat("abc", 8),
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.c2lnbmF0dXJl",
	}
	for _, secret := range planted {
		title, detail := agents.Scrub("I set the token to " + secret + ", as asked.\nIt is " + secret + " now.")
		if strings.Contains(title+detail, secret) {
			t.Errorf("kept a secret: %q / %q", title, detail)
		}
		if !strings.HasPrefix(title, "I set the token to") {
			t.Errorf("lost the words around a secret: %q", title)
		}
		task := agents.Judge(agents.Agent{}, agents.Event{Kind: agents.Prompted, Prompt: "use " + secret + " to deploy"}).Task
		if strings.Contains(task, secret) {
			t.Errorf("a task kept a secret: %q", task)
		}
	}
}

func TestATitleOrADetailIsCutAtAWord(t *testing.T) {
	title, detail := agents.Scrub(strings.Repeat("refresh ", 40) + "\n" + "then " + strings.Repeat("refresh ", 40) + ".")
	for _, s := range []string{title, detail} {
		if utf8.RuneCountInString(s) > catalog.AgentLineCap {
			t.Errorf("%d characters, the cap is %d: %q", utf8.RuneCountInString(s), catalog.AgentLineCap, s)
		}
		if !strings.HasSuffix(s, " refresh") {
			t.Errorf("not cut at a word: %q", s)
		}
	}
}
