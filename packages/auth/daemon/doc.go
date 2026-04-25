// Package daemon runs the MCP credential refresher as a host-side
// background service.
//
// SyncCredentials is event-driven (it runs on every spawn / talk /
// login), which covers the common case of a user actively using
// spwn. It does NOT cover:
//
//   - Long-running agent sessions (>1h) that don't trip a host event
//   - Autonomous agents looping inside a container with no host
//     activity for hours
//
// This package fills that gap by running mcp.RefreshAll on a ticker.
// It's exposed as `spwn auth daemon {install,start,stop,status,
// uninstall,run}` and registered with the OS init system via
// kardianos/service (launchd on macOS, systemd-user on Linux,
// SCM on Windows). User-mode service: tokens live in the user's
// home, refresh runs as the user.
//
// The same binary serves both as the long-running daemon (`spwn
// auth daemon run`) and as the management CLI — kardianos/service
// plumbs the right entry point depending on whether the OS init
// system or a human invoked it.
package daemon
