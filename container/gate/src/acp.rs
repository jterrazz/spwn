// ACP (Agent Client Protocol) client stub.
//
// This module will use the official `agent-client-protocol` Rust crate
// to communicate with the agent CLI via JSON-RPC over stdio.
//
// For now, the Gate spawns the agent CLI directly as a subprocess.
// ACP integration will replace direct subprocess management with:
//   - initialize: negotiate capabilities
//   - session/new: start a new conversation
//   - session/load: resume an existing session
//   - session/prompt: send a message
//   - session/update: receive streaming events
//
// The `agent-client-protocol` crate is not yet added as a dependency
// because it requires the upstream crate to be published. When available,
// add to Cargo.toml:
//   agent-client-protocol = "0.1"
