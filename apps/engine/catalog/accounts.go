package catalog

type Provider struct {
	// value: a brand the owner's accounts are drawn by: its id and the label
	// a login program carries for it.
	ID, Label string
}

var ClaudePlans = map[string]string{"max": "Max plan", "pro": "Pro plan", "team": "Team plan", "enterprise": "Enterprise plan"} // value: Claude's plan in words, by the subscription `claude auth status` prints; one it does not list is said as printed.

type LoginProgram struct {
	// value: a program station signs in for the owner: the name its wire calls
	// it by, which is the command it runs, and its label.
	Name, Label string
	// value: how its login asks the owner, and the site it sends them to.
	Flow Flow
	Site string
	// value: the page a device code is entered at, opened on the owner's Mac.
	Page string
	// value: the read that says whether it is signed in, which never prints
	// a secret, and how its answer reads.
	Detect []string
	Reads  SignInRead
	// value: its own login, run with no terminal; it prints the code or the
	// link and waits for the owner at Site.
	Login []string
	// value: the one-time code its login prints, as the first group of Code,
	// read on a line after one holding CodeAfter (any line when empty).
	Code, CodeAfter string
	// value: what ends its sign-in on the machine; the token stays valid at its site.
	Remove []string
}

type SignInRead struct {
	// value: a JSON answer says signed in by this key being true; empty reads
	// the exit code, 0 signed in.
	JSON string
	// value: a word of a failing answer that means signed out; any other
	// failure is a broken sign-in.
	Out string
	// value: who is signed in: a JSON key, or the first group of a pattern
	// over the answer, its first line when both are empty.
	Who, WhoKey string
	// value: the plan, by a JSON key, said through Plans.
	PlanKey string
	Plans   map[string]string
}

type Flow string

const (
	FlowDeviceCode Flow = "device-code"
	FlowLink       Flow = "link"
	FlowPaste      Flow = "paste"
)

var LoginPrograms = []LoginProgram{
	{
		Name: "gh", Label: "GitHub", Flow: FlowDeviceCode, Site: "github.com", Page: "https://github.com/login/device",
		Detect: []string{"auth", "status", "--hostname", "github.com"}, Reads: SignInRead{Who: `account (\S+)`},
		Login:  []string{"auth", "login", "--web", "--hostname", "github.com", "--git-protocol", "https", "--skip-ssh-key"},
		Code:   `one-time code: ([0-9A-F]{4}-[0-9A-F]{4})`,
		Remove: []string{"auth", "logout", "--hostname", "github.com"},
	},
	{
		Name: "npm", Label: "npm", Flow: FlowLink, Site: "www.npmjs.com",
		Detect: []string{"whoami"}, Reads: SignInRead{Out: "ENEEDAUTH"},
		Login:  []string{"login", "--auth-type=web"},
		Remove: []string{"logout"},
	},
	{
		Name: "claude", Label: "Claude", Flow: FlowPaste, Site: "claude.com",
		Detect: []string{"auth", "status"}, Reads: SignInRead{JSON: "loggedIn", WhoKey: "email", PlanKey: "subscriptionType", Plans: ClaudePlans},
		Login:  []string{"auth", "login", "--claudeai"},
		Remove: []string{"auth", "logout"},
	},
	{
		Name: "wrangler", Label: "Cloudflare", Flow: FlowDeviceCode, Site: "dash.cloudflare.com", Page: "https://dash.cloudflare.com/oauth2/device/verify",
		Detect: []string{"whoami", "--json"}, Reads: SignInRead{JSON: "loggedIn", WhoKey: "email"},
		Login: []string{"login", "--device"},
		Code:  `^\s*([A-Za-z0-9]{8})\s*$`, CodeAfter: "enter the code",
		Remove: []string{"logout"},
	},
}

const LoginBinary = `PATH="$HOME/.local/bin:$PATH"; p=$(command -v "$0") || exit 127; case "$p" in */shims/*) p=$(mise which "$0") || exit 127; PATH="${p%/*}:$PATH";; esac` // value: how a sign-in's line finds its program: on the machine's PATH and the owner's own bin folder, and through the version manager when PATH holds only its shim, since a shim is not the program a signal must reach.

const (
	ProviderGitHub     = "github"
	ProviderNPM        = "npm"
	ProviderClaude     = "claude"
	ProviderTailscale  = "tailscale"
	ProviderCloudflare = "cloudflare"
	ProviderDocker     = "docker"
)

var Providers = []Provider{
	{ID: ProviderGitHub, Label: "GitHub"},
	{ID: ProviderNPM, Label: "npm"},
	{ID: ProviderClaude, Label: "Claude"},
	{ID: ProviderTailscale, Label: "Tailscale"},
	{ID: ProviderCloudflare, Label: "Cloudflare"},
	{ID: ProviderDocker, Label: "Docker"},
}

const ClaudeTokenName = "operator:claude:oauth-token"
