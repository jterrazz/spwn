package catalog

const Sudo = "sudo" // value: the one spelling of sudo outside the owner's words: station runs a line as root inside a VM of its own, as its admin, and on another Mac only for the planned restart's fdesetup authrestart, the owner's gesture, its password asked by macOS and never stored; a plan reads whether a root line asks a password by sudo's own check, which never prompts and runs only true; any other root line it names for the owner to type.

func AsRoot(line string) string { return Sudo + " " + line }

const InstallLine = "curl -fsSL https://raw.githubusercontent.com/jterrazz/studio-station/main/scripts/install.sh | sh" // value: the one line that installs station on a new Mac, as docs/start/getting-started.md gives it: the release's script, which then runs station init.
