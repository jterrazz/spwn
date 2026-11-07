package catalog

import "time"

type AgentVerb string

const (
	AgentEdit     AgentVerb = "edit"
	AgentRead     AgentVerb = "read"
	AgentTest     AgentVerb = "test"
	AgentCommit   AgentVerb = "commit"
	AgentOpenPR   AgentVerb = "open-pr"
	AgentBuild    AgentVerb = "build"
	AgentInstall  AgentVerb = "install"
	AgentCommand  AgentVerb = "command"
	AgentWeb      AgentVerb = "web"
	AgentDelegate AgentVerb = "delegate"
)

type AgentStepWords struct {
	// value: what a step says while it is the agent's current one, and once
	// a later step has followed it.
	Verb        AgentVerb
	Doing, Done string
}

var AgentStepsSaid = []AgentStepWords{
	{Verb: AgentEdit, Doing: "editing", Done: "edited"},
	{Verb: AgentRead, Doing: "reading", Done: "read"},
	{Verb: AgentTest, Doing: "running tests", Done: "ran the tests"},
	{Verb: AgentCommit, Doing: "committing", Done: "committed"},
	{Verb: AgentOpenPR, Doing: "opening a PR", Done: "opened a PR"},
	{Verb: AgentBuild, Doing: "building", Done: "built"},
	{Verb: AgentInstall, Doing: "installing", Done: "installed"},
	{Verb: AgentCommand, Doing: "running a command", Done: "ran a command"},
	{Verb: AgentWeb, Doing: "reading the web", Done: "read the web"},
	{Verb: AgentDelegate, Doing: "delegating", Done: "delegated"},
}

type AgentCommandClass struct {
	// value: the class of a shell command an agent runs, by the words one of
	// its parts opens with; the command itself is never kept.
	Verb   AgentVerb
	Begins []string
}

var AgentCommands = []AgentCommandClass{
	{Verb: AgentOpenPR, Begins: []string{"gh pr create"}},
	{Verb: AgentCommit, Begins: []string{"git commit"}},
	{Verb: AgentTest, Begins: []string{
		"go test", "gotestsum", "vitest", "npx vitest", "jest", "npx jest", "npm test", "npm run test", "pnpm test", "pnpm run test",
		"yarn test", "bun test", "bun run test", "pytest", "python -m pytest", "python3 -m pytest", "uv run pytest", "cargo test",
		"make test", "make check", "swift test",
	}},
	{Verb: AgentBuild, Begins: []string{
		"go build", "npm run build", "pnpm build", "pnpm run build", "yarn build", "bun run build", "cargo build", "make build",
		"tsc", "npx tsc", "swift build", "xcodebuild",
	}},
	{Verb: AgentInstall, Begins: []string{
		"npm install", "npm ci", "npm i", "pnpm install", "pnpm i", "yarn install", "yarn add", "bun install", "bun add",
		"pip install", "uv pip install", "uv sync", "uv add", "go get", "go mod download", "go mod tidy", "cargo add", "cargo fetch",
		"brew install",
	}},
}

type AgentTestSummary struct {
	// value: how one test runner's summary line says its counts: each group
	// pass, fail and total of Pattern holds a number; a match of a pattern
	// with a group one_pass or one_fail counts one test.
	Runner  string
	Pattern string
}

var AgentTestSummaries = []AgentTestSummary{
	{Runner: "go test", Pattern: `(?m)^\s*--- (?:(?P<one_pass>PASS)|(?P<one_fail>FAIL)): `},
	{Runner: "vitest", Pattern: `(?m)^\s*Tests\s+(?:(?P<fail>\d+) failed \| )?(?P<pass>\d+) passed(?: \| \d+ \w+)* \((?P<total>\d+)\)`},
	{Runner: "jest", Pattern: `(?m)^Tests:\s+(?:(?P<fail>\d+) failed, )?(?:\d+ skipped, )?(?:\d+ todo, )?(?P<pass>\d+) passed, (?P<total>\d+) total`},
	{Runner: "pytest", Pattern: `(?m)^=+ (?:(?P<fail>\d+) failed, )?(?P<pass>\d+) passed(?:, \d+ \w+)* in [\d.]+s`},
	{Runner: "cargo", Pattern: `(?m)^test result: \w+\. (?P<pass>\d+) passed; (?P<fail>\d+) failed`},
}

var AgentRecord = []string{"station", "agents", "record", "--app"} // value: the leaf a hook runs, each app's adapter id after it; it hands the event on to agents.record.

const AgentPullRequest = `https://github\.com/[\w.-]+/[\w.-]+/pull/(\d+)` // value: the URL a forge prints for a PR it opened, its number the one group.

const (
	AgentStepsKept  = 3                  // value: how many of an agent's last steps its journal keeps.
	AgentLineCap    = 120                // value: the most characters a kept title or detail holds.
	AgentNameCap    = 40                 // value: the most characters an inline code span may hold and still be kept, as a path or a name.
	AgentKeptFor    = 7 * 24 * time.Hour // value: how long a session's journal outlives its end.
	AgentCodeSaid   = "replied with code"
	AgentWatchEvery = time.Second // value: how often the guest looks at a VM's journal for a change.
	// value: a hook's run of station ends within AgentHookBudget, the
	// binary's start included: its event waits AgentRecordBudget at most,
	// then is dropped. AgentPayloadCap is the most of a payload it reads.
	AgentHookBudget   = 200 * time.Millisecond
	AgentRecordBudget = 150 * time.Millisecond
	AgentPayloadCap   = 1 << 20
)
