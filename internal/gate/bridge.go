package gate

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jterrazz/universe/internal/config"
)

// GenerateWrapperScript returns a shell script that bridges an element invocation
// through the Gate's Unix socket.
func GenerateWrapperScript(elementName string) string {
	return fmt.Sprintf(`#!/bin/sh
# Gate bridge wrapper for %q — forwards to host-side Gate via Unix socket.
# Auto-generated at universe creation time.

ARGS=""
for arg in "$@"; do
  arg=$(printf '%%s' "$arg" | sed 's/\\/\\\\/g; s/"/\\"/g')
  if [ -z "$ARGS" ]; then
    ARGS="\"$arg\""
  else
    ARGS="$ARGS,\"$arg\""
  fi
done

RESPONSE=$(curl -s --unix-socket /gate/gate.sock \
  -X POST http://localhost/invoke \
  -H 'Content-Type: application/json' \
  -d "{\"element\":\"%s\",\"args\":[$ARGS]}" 2>/dev/null)

if [ $? -ne 0 ]; then
  echo "error: gate bridge unreachable" >&2
  exit 1
fi

# Extract fields from JSON response
STDOUT=$(printf '%%s' "$RESPONSE" | sed -n 's/.*"stdout":"\([^"]*\)".*/\1/p')
STDERR=$(printf '%%s' "$RESPONSE" | sed -n 's/.*"stderr":"\([^"]*\)".*/\1/p')
EXIT_CODE=$(printf '%%s' "$RESPONSE" | sed -n 's/.*"exit_code":\([0-9]*\).*/\1/p')

[ -n "$STDOUT" ] && printf '%%s\n' "$STDOUT"
[ -n "$STDERR" ] && printf '%%s\n' "$STDERR" >&2
exit ${EXIT_CODE:-1}
`, elementName, elementName)
}

// SetupBridges writes wrapper scripts for each gate bridge into gateDir/bin/.
func SetupBridges(gateDir string, bridges []config.GateBridge) error {
	binDir := filepath.Join(gateDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("create gate bin dir: %w", err)
	}

	for _, bridge := range bridges {
		script := GenerateWrapperScript(bridge.As)
		path := filepath.Join(binDir, bridge.As)
		if err := os.WriteFile(path, []byte(script), 0755); err != nil {
			return fmt.Errorf("write bridge wrapper %q: %w", bridge.As, err)
		}
	}

	return nil
}
