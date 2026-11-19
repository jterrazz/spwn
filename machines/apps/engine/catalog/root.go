package catalog

import (
	"path/filepath"
	"strings"
)

const Sudo = "sudo" // value: the one spelling of sudo outside the owner's words: station runs a line as root inside a VM of its own, as its admin, and on another Mac only for the planned restart's fdesetup authrestart, the owner's gesture, its password asked by macOS and never stored; a plan reads whether a root line asks a password by sudo's own check, which never prompts and runs only true; any other root line it names for the owner to type.

func AsRoot(line string) string { return Sudo + " " + line }

const InstallDir = "~/.local/bin" // value: the folder the one public installer puts station in when it is given no other; `~` is the home. The installer is another repository's, so a window reads what the engine records for the folder it really runs from, and this only when none is recorded.

func InstallDirIn(home string) string {
	return filepath.Join(home, strings.TrimPrefix(InstallDir, "~/"))
}

const StationBinary = "station" // value: the file station is installed as, in InstallDir.

const StateFolder = "station" // value: the folder station keeps its preferences, data, state and cache in, under each XDG root.

const StationRepo = "jterrazz/spwn" // value: the repository whose releases are station's.

const InstallLine = "curl -fsSL https://spwn.sh/install.sh | bash" // value: the one line that installs station on a new Mac, as docs/start/getting-started.md gives it: the one public installer spwn.sh serves, which then runs station init.

func FarStationLine() string {
	return `'PATH="$HOME/` + strings.TrimPrefix(InstallDir, "~/") + `:/usr/local/bin:$PATH" exec ` + StationBinary + ` "$@"'`
}
