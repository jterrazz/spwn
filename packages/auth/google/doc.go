// Package google handles spwn's Google Workspace authentication.
//
// Why not the same pattern as packages/auth/mcp?
//
// MCP-side (Notion, etc.) leans on Dynamic Client Registration: the
// hosted MCP server registers a fresh OAuth client per spwn install,
// no per-user setup. Google explicitly does NOT support DCR; every
// OAuth client must be pre-registered in a Google Cloud project. We
// can't ship one shared client_id either — Google's "restricted
// scopes" (Gmail) require a paid annual security audit (CASA) to be
// usable beyond ~100 explicitly-allowlisted test users.
//
// So the OSS-realistic pattern is "bring your own GCP client":
//
//   1. User creates a GCP project once (~10 min, free, no audit).
//   2. `spwn auth login google` walks them through it interactively,
//      captures their client_id (+ optional client_secret), and runs
//      a standard OAuth installed-app PKCE flow.
//   3. Tokens land at ~/.spwn/credentials/google/. Bind-mounted into
//      the gate, where gws/the gate uses them per-request via
//      GOOGLE_WORKSPACE_CLI_TOKEN. Refresh on demand.
//
// Once Google verifies an OAuth app for sensitive scopes (Calendar)
// or restricted scopes (Gmail), we could ship a baked-in client_id
// and skip the per-user GCP setup. That's a future move — for now,
// every other OSS Google MCP server (taylorwilsdon, aaronsb, gws
// itself) uses the same BYO pattern, and the friction is small
// enough that users don't seem to mind.
package google
