# ADR-010: Ports & Adapters (Trait-Driven Architecture)

**Status:** Accepted
**Date:** 2026-03-29

## Context

Spwn integrates with many external systems: container runtimes (Docker, K8s), LLM providers (Anthropic, OpenAI, Google), agent runtimes (Claude Code, Pi, Codex), messaging channels (Telegram, Slack), persistence layers (filesystem, databases), and tool ecosystems (MCP, built-in). The AI ecosystem changes monthly---new agent CLIs, new providers, new deployment targets appear constantly.

A monolithic architecture that hard-codes these integrations becomes unmaintainable. Every new provider requires changes across the codebase. Every new runtime requires rewiring the lifecycle manager. The system becomes brittle and vendor-locked.

## Decision

**Every layer of spwn that touches an external system is defined as a port (Go interface) with swappable adapters (implementations).**

We adopt a strict ports & adapters architecture with 8 ports:

| Port         | What it abstracts                    | Default adapter       |
| ------------ | ------------------------------------ | --------------------- |
| **Backend**  | Where universes run                  | Docker                |
| **Runtime**  | How agents think (agent loop)        | Claude Code (ACP)     |
| **Provider** | Which LLM serves intelligence        | Anthropic             |
| **Channel**  | How Architect talks to outside world | CLI                   |
| **Memory**   | How Minds persist                    | Filesystem (markdown) |
| **Store**    | How state is tracked                 | JSON file             |
| **Tool**     | What agents can do                   | Built-in + MCP        |
| **Skill**    | Reusable capabilities                | Local files           |

### Rules

1. **Core domain modules program against ports, never against concrete adapters.** The `universe` module calls `backend.Create()`, never `docker.CreateContainer()`.
2. **Adapters are injected at startup.** The CLI or Architect daemon creates adapter instances and passes them to domain constructors.
3. **Ports are Go interfaces defined in the domain that uses them.** The Backend port is defined in `core/universe`, not in a shared package.
4. **One adapter per port at runtime.** No adapter chaining or middleware (simplicity first).
5. **Manifests select adapters.** The `org.yaml`, `worlds/{name}.yaml`, and `profile.yaml` manifests declare which adapter to use for each port. Cascading overrides apply.

### How Adapters Are Registered and Swapped

Adapters are registered in the CLI's startup code (or Architect's startup code):

```go
// Adapter selection based on org.yaml defaults
switch orgConfig.Defaults.Backend {
case "docker":
    backend = docker.NewBackend()
case "k8s":
    backend = k8s.NewBackend()
}

architect := universe.NewArchitect(backend, memory, store, skillSource)
```

Swapping an adapter means changing a manifest value and restarting. No code changes, no recompilation.

## Consequences

### Positive

- **No vendor lock-in.** Switching from Docker to K8s, or from Anthropic to OpenAI, is a configuration change.
- **Testability.** Every port can be mocked in tests. No Docker required for unit tests.
- **Ecosystem growth.** Third parties can write adapters without modifying core spwn.
- **Future-proof.** When a new agent CLI appears, implement the Runtime port. When a new cloud provider appears, implement the Backend port.
- **Framework identity.** Spwn is a framework for orchestrating artificial life, not a product tied to specific vendors.

### Negative

- **Indirection.** One more layer between the caller and the implementation.
- **Interface design burden.** Ports must be designed carefully---too narrow limits adapters, too wide leaks abstraction.
- **Adapter parity.** Not all adapters will support all features. Need a capability negotiation pattern.

### Risks

- **Premature abstraction.** Some ports may only ever have one adapter. Mitigated by starting with defaults and only adding the port when a second adapter is needed.
- **Leaky abstractions.** Docker-specific concepts leaking into the Backend port. Mitigated by designing ports from the domain's perspective, not the adapter's.

## Alternatives Considered

| Alternative                                      | Why rejected                                                                  |
| ------------------------------------------------ | ----------------------------------------------------------------------------- |
| **Plugin system (dynamic loading)**              | Adds complexity, Go's plugin support is limited, most users need 1-2 adapters |
| **No abstraction (hard-code Docker, Anthropic)** | Vendor lock-in, untestable, brittle                                           |
| **Middleware chains**                            | Over-engineering for our use case, adds latency and complexity                |
