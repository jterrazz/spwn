package catalog

const (
	GrantsDir = ".config/station/grants"                                             // value: where a VM keeps the environment files of the programs granted to it, from its home; the zsh block sources every `*.env` in it.
	GuestPath = `export PATH="$HOME/.local/bin:$HOME/.local/share/mise/shims:$PATH"` // value: the PATH a delivery line runs under inside a VM, so it finds the program a non-interactive shell would not.
)

type Delivery struct {
	// value: how a password granted for a program reaches it inside a VM,
	// by the program's wire name.
	Program string
	// value: the variable an environment-file program reads its token from;
	// empty for a program that keeps the token in its own store.
	Env string
	// value: the shell lines that write the token read on stdin into the
	// program's own store, and that delete it; empty for an
	// environment-file program, whose file station writes and deletes.
	Inject, Remove string
	// value: the command on this Mac that prints the token its sign-in left,
	// empty when that token cannot be taken.
	Capture []string
	// value: the kind of the token Capture prints, by the wire name of a
	// key kind; empty when nothing can be captured.
	CaptureKind string
	// value: the kinds a token typed for the program may be, the first the
	// one offered by default, by the wire name of a key kind.
	Kinds []string
}

const npmRegistryLine = `^//registry.npmjs.org/:_authToken=`

var Deliveries = []Delivery{
	{Program: "gh", Inject: GuestPath + "; gh auth login --with-token -h github.com", Remove: GuestPath + "; gh auth logout -h github.com", Capture: []string{"gh", "auth", "token", "-h", "github.com"}, CaptureKind: "oauth-token", Kinds: []string{"fine-grained", "token"}},
	{Program: "npm",
		Inject: `umask 077; IFS= read -r t; f="$HOME/.npmrc"; touch "$f"; { grep -v '` + npmRegistryLine + `' "$f"; printf '//registry.npmjs.org/:_authToken=%s\n' "$t"; } > "$f.station" && mv "$f.station" "$f"`,
		Remove: `f="$HOME/.npmrc"; [ -f "$f" ] || exit 0; umask 077; grep -v '` + npmRegistryLine + `' "$f" > "$f.station"; mv "$f.station" "$f"`,
		Kinds:  []string{"token", "fine-grained"}},
	{Program: "claude", Env: "CLAUDE_CODE_OAUTH_TOKEN", Kinds: []string{"oauth-token"}},
	{Program: "wrangler", Env: "CLOUDFLARE_API_TOKEN", Kinds: []string{"token", "fine-grained"}},
}
