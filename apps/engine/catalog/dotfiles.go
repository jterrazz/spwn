package catalog

import "embed"

// invariant: `all:` keeps the dotted files (zed/.oxfmtrc.json) the plain pattern drops.

//go:embed all:dotfiles
var DotfilesFS embed.FS

const ZshLoader = "dotfiles/zsh/zshrc.sh" // value: the dotfiles' zsh loader, in DotfilesFS: the first lines of station's block.

// invariant: the version manager station installs tools with is activated
// only when it is on PATH, after the loader's PATH lines, so a machine
// without it gets a quiet shell.
const (
	AppActivation = `if command -v mise >/dev/null 2>&1; then
  eval "$(mise activate zsh)"
fi
`
)

// value: where the tool manager keeps its runtimes (the variable that moves
// them, its default under home, the folder of its shims), and the global
// bins under home the shell loader puts on PATH.
const (
	AppDataVar = "MISE_DATA_DIR"
	AppData    = ".local/share/mise"
	AppShims   = "shims"
)

var GlobalBins = []string{".local/share/bun/bin", ".local/share/go/bin"}
