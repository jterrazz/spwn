package catalog

import "time"

type ProviderRead struct {
	Name string
	Args []string
}

type Provider struct {
	// value: a provider of rights the Accounts page shows: where it is read,
	// the commands that read it (never one that prints a secret), and the
	// budget the whole read gets.
	ID, Label string
	Mac, VM   bool
	Command   string
	Reads     []ProviderRead
	Budget    time.Duration
	// value: where the owner ends a right station did not lend: shown on a
	// revoke of an own row, never run.
	Logout string
	// value: what the owner runs to fix a row, per reason, on this machine
	// and in a VM (HintVM stands for the VM's name); empty is no hint.
	Hints []ProviderHint
	// value: a row shown only once it is read: a VM whose guest is not
	// connected shows none for it.
	Optional bool
	// value: what the menu bar says beside a signed-in account, in order:
	// the parts the provider's own read gives; a provider with none (git's
	// identity, ssh's keys) is no sign-in and is left off the menu bar.
	Words []Part
}

type Part string

const (
	PartWho    Part = "who"
	PartScopes Part = "scopes"
	PartPlan   Part = "plan"
)

var ClaudePlans = map[string]string{"max": "Max plan", "pro": "Pro plan", "team": "Team plan", "enterprise": "Enterprise plan"} // value: Claude's plan in words, by the subscription `claude auth status` prints; one it does not list is said as printed.

type ProviderHint struct {
	Reason, ThisMachine, InVM string
}

const HintVM = "<vm>"

const (
	ProviderGitHub     = "github"
	ProviderNPM        = "npm"
	ProviderClaude     = "claude"
	ProviderTailscale  = "tailscale"
	ProviderCloudflare = "cloudflare"
	ProviderGit        = "git"
	ProviderSSH        = "ssh"
	ProviderDocker     = "docker"
	ProviderEntry      = "entry"
)

var Providers = []Provider{
	{ID: ProviderGitHub, Label: "GitHub", Logout: "gh auth logout", Mac: true, VM: true, Command: "gh", Reads: []ProviderRead{{Name: "gh", Args: []string{"auth", "status", "--hostname", "github.com"}}}, Budget: 5 * time.Second, Words: []Part{PartWho, PartScopes}, Hints: []ProviderHint{
		{Reason: "unset", ThisMachine: "station settings apply github-login", InVM: "ssh " + HintVM + " gh auth login"},
		{Reason: "missing", ThisMachine: "station apps install gh"},
	}},
	{ID: ProviderNPM, Label: "npm", Logout: "npm logout", Mac: true, VM: true, Command: "npm", Reads: []ProviderRead{{Name: "npm", Args: []string{"whoami"}}}, Budget: 5 * time.Second, Words: []Part{PartWho}, Hints: []ProviderHint{
		{Reason: "unset", ThisMachine: "npm login", InVM: "npm login"},
		{Reason: "refused", ThisMachine: "npm login", InVM: "npm login"},
	}},
	{ID: ProviderClaude, Label: "Claude", Logout: "claude /logout", Mac: true, VM: true, Command: "claude", Reads: []ProviderRead{{Name: "claude", Args: []string{"auth", "status"}}}, Budget: 5 * time.Second, Words: []Part{PartPlan}, Hints: []ProviderHint{
		{Reason: "unset", ThisMachine: "claude /login"},
	}},
	{ID: ProviderTailscale, Label: "Tailscale", Logout: "tailscale logout", Mac: true, Words: []Part{PartWho}},
	{ID: ProviderCloudflare, Label: "Cloudflare", Logout: "wrangler logout", Mac: true, VM: true, Command: "wrangler", Reads: []ProviderRead{{Name: "wrangler", Args: []string{"whoami"}}}, Budget: 10 * time.Second, Words: []Part{PartWho}, Hints: []ProviderHint{
		{Reason: "unset", ThisMachine: "wrangler login", InVM: "wrangler login"},
	}},
	{ID: ProviderGit, Label: "git", Mac: true, VM: true, Command: "git", Reads: []ProviderRead{
		{Name: "git", Args: []string{"config", "--global", "user.name"}},
		{Name: "git", Args: []string{"config", "--global", "user.email"}},
	}, Budget: 2 * time.Second, Hints: []ProviderHint{
		{Reason: "unset", ThisMachine: "git config --global user.name <name>", InVM: "git config --global user.name <name>"},
	}},
	{ID: ProviderSSH, Label: "ssh", Logout: "ssh-add -D", Mac: true, VM: true, Command: "ssh-add", Reads: []ProviderRead{{Name: "ssh-add", Args: []string{"-l"}}}, Budget: 2 * time.Second},
	{ID: ProviderDocker, Label: "Docker", Logout: "docker logout", Mac: true, VM: true, Budget: 5 * time.Second, Optional: true, Words: []Part{PartWho}},
	{ID: ProviderEntry, Label: "entry", VM: true},
}

type ShareScope struct {
	// value: a lend-list name, the provider whose right its copy carries,
	// and what that right lets the VM do.
	Name, Provider, Scope string
}

const ClaudeTokenName = "operator:claude:oauth-token"

var ShareScopes = []ShareScope{
	{Name: ClaudeTokenName, Provider: ProviderClaude, Scope: "Claude Code inference"},
	{Name: "session:github", Provider: ProviderGitHub, Scope: "GitHub browser session"},
}

const DockerSettingsFile = ".docker/config.json"
