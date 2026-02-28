package gate

import (
	"fmt"
	"strings"

	"github.com/jterrazz/universe/internal/config"
)

// WrapperScript generates a shell script that invokes an interaction via the gate socket.
func WrapperScript(interaction config.Interaction) string {
	return fmt.Sprintf(`#!/bin/sh
# Gate wrapper for %s (bridged from %s)
set -e

# Build JSON args array from positional parameters.
ARGS="["
FIRST=1
for arg in "$@"; do
  if [ "$FIRST" = "1" ]; then
    FIRST=0
  else
    ARGS="$ARGS,"
  fi
  # Escape double quotes and backslashes in arg.
  escaped=$(printf '%%s' "$arg" | sed 's/\\/\\\\/g; s/"/\\"/g')
  ARGS="$ARGS\"$escaped\""
done
ARGS="$ARGS]"

BODY="{\"interaction\":\"%s\",\"args\":$ARGS}"

RESPONSE=$(curl -s --unix-socket /gate/gate.sock -X POST -H "Content-Type: application/json" -d "$BODY" http://localhost/invoke)

# Extract fields using sed (jq may not be available).
STDOUT=$(printf '%%s' "$RESPONSE" | sed -n 's/.*"stdout":"\([^"]*\)".*/\1/p')
STDERR=$(printf '%%s' "$RESPONSE" | sed -n 's/.*"stderr":"\([^"]*\)".*/\1/p')
EXIT_CODE=$(printf '%%s' "$RESPONSE" | sed -n 's/.*"exit_code":\([0-9]*\).*/\1/p')

if [ -n "$STDOUT" ]; then
  printf '%%s\n' "$STDOUT"
fi
if [ -n "$STDERR" ]; then
  printf '%%s\n' "$STDERR" >&2
fi

exit ${EXIT_CODE:-0}
`, interaction.As, interaction.Source, interaction.As)
}

// AllWrapperCommands returns shell commands that install all wrapper scripts
// into a running container.
func AllWrapperCommands(interactions []config.Interaction) []string {
	var cmds []string

	cmds = append(cmds, "mkdir -p /gate/bin")

	for _, ia := range interactions {
		script := WrapperScript(ia)
		// Write script as a heredoc and make executable.
		escaped := strings.ReplaceAll(script, "'", "'\\''")
		cmds = append(cmds, fmt.Sprintf("cat > /gate/bin/%s << 'WRAPPER_EOF'\n%sWRAPPER_EOF", ia.As, script))
		_ = escaped // using heredoc with single-quote delimiter avoids escaping
		cmds = append(cmds, fmt.Sprintf("chmod +x /gate/bin/%s", ia.As))
		// Symlink into /usr/local/bin for non-login shells.
		cmds = append(cmds, fmt.Sprintf("ln -sf /gate/bin/%s /usr/local/bin/%s", ia.As, ia.As))
	}

	// Add /gate/bin to PATH for login shells.
	cmds = append(cmds, `mkdir -p /etc/profile.d && printf 'export PATH="/gate/bin:$PATH"\n' > /etc/profile.d/gate.sh`)

	return cmds
}
