// Package defaults blank-imports every built-in runtime so their
// Adapters register with the runtimes package at init time.
//
// Any binary that wants the default runtime set should import this
// package (under `_`) once near its entry point. Individual subpackages
// may be imported directly when a binary needs a specific runtime
// without pulling in the rest.
package defaults

import (
	_ "spwn.sh/packages/runtimes/claudecode"
	_ "spwn.sh/packages/runtimes/codex"
)
