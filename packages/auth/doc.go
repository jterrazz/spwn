// Package auth resolves AI-provider credentials (Anthropic, OpenAI,
// Google) from their canonical host-side locations — env vars,
// Claude Code's OAuth cache, Codex's auth.json, the macOS Keychain
// — and syncs them into the bind-mountable /credentials/ directory
// that every spwn container reads at startup.
//
// Callers use Resolve(provider) for a single provider, ResolveAll()
// for every known one, and SyncCredentials() to write the current
// state into platform.CredentialsDir(). The Credential type is
// runtime-neutral; Docker-specific formatting (env flags, -e args)
// lives in packages/architect.
//
// The package keeps host and container auth surfaces decoupled:
// runtimes only ever see /credentials/, never the user's home
// directory.
package auth
