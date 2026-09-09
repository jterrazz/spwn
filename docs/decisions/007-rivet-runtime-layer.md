# ADR-007: Rivet Sandbox Agent SDK as the Runtime Normalization Layer

**Status:** Accepted
**Date:** 2026-03-29

## Context

Spwn supports multiple agent runtimes (Claude Code via ACP, Pi SDK via RPC, Codex via AppServer, OpenCode via ACP) and multiple providers (Anthropic, OpenAI, Google, Bedrock, Groq, Mistral, OpenRouter, Ollama). Without a normalization layer, the Gate, Governor, CLI, and Observatory would each need to understand every runtime's protocol and event format.

The current container-side Gate speaks raw ACP to a single agent CLI. This works for Epoch 3 (Body) but does not scale to multi-runtime, multi-agent universes.

## Decision

Adopt the **Rivet Sandbox Agent SDK** as the runtime normalization layer inside universe containers.

Rivet wraps all agent runtimes behind one consistent API:

- **Event streaming**: unified event format regardless of which runtime powers the agent.
- **Session persistence**: consistent session management across all runtimes.
- **Audit**: structured logging of all agent actions.
- **Runtime abstraction**: the Gate, Governor, and Observatory see one API regardless of whether the agent uses Claude Code (ACP), Pi (RPC), or Codex (AppServer).

## Consequences

### Positive

- **Single integration point**: the Gate only needs to speak to Rivet, not to each runtime individually.
- **Mixed runtimes per universe**: a Governor on Claude Opus and Citizens on GPT-4 or Sonnet all look identical to the orchestration layer.
- **Observatory compatibility**: Rivet's event stream powers the real-time dashboard without runtime-specific adapters.
- **Future-proof**: new runtimes are added as Rivet adapters, not as Gate protocol changes.

### Negative

- **Additional abstraction layer**: Rivet adds latency and complexity between the Gate and the agent runtime.
- **Dependency**: Rivet becomes a critical dependency. If it has bugs, all runtimes are affected.

### Migration

- Epoch 3 (Body) continues using raw ACP directly.
- Epoch 4 (Runtimes) introduces Rivet as a wrapper around ACP, then extends it to support Pi, Codex, and OpenCode.
- The Gate protocol between host-side and container-side remains unchanged.
