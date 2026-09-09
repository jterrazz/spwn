# ADR-009: Pi-mono as the Primary Multi-Provider Runtime

**Status:** Accepted
**Date:** 2026-03-29

## Context

Spwn agents need to run on different LLM providers: Anthropic, OpenAI, Google, Bedrock, Groq, Mistral, OpenRouter, Ollama (local), and more. Each provider has its own API, authentication, and streaming format. Without a multi-provider abstraction, each provider requires its own integration in the runtime layer.

Additionally, some users authenticate via subscriptions (Claude Pro/Max, ChatGPT Plus) rather than API keys. Subscription auth requires OAuth flows that are distinct from API key auth.

## Decision

Use **Pi-mono** as the primary multi-provider runtime behind Rivet.

Pi-mono provides:

- **17+ providers**: Anthropic, OpenAI, Google, Bedrock, Groq, Mistral, OpenRouter, Ollama, and more.
- **Subscription OAuth**: supports Claude Pro/Max, ChatGPT Plus, and other subscription-based auth.
- **Embeddable SDK**: can be embedded inside the Rivet normalization layer.
- **Unified streaming**: consistent streaming format across all providers.

## Consequences

### Positive

- **Broad provider support**: 17+ providers out of the box, no per-provider integration work.
- **Subscription auth**: users can use their existing Claude Pro or ChatGPT Plus subscriptions without managing API keys.
- **Cost optimization**: Governors on expensive models (Opus), Citizens on cheaper models (Sonnet), NPCs on cheapest (Haiku)—all through the same runtime.
- **Local models**: Ollama support means fully offline operation for privacy-sensitive environments.

### Negative

- **Dependency**: Pi-mono becomes a critical dependency for multi-provider support.
- **Abstraction leaks**: provider-specific features (streaming format quirks, rate limits, token counting) may not be perfectly normalized.

### Alternatives Considered

- **Direct provider SDKs**: each provider integrated separately. Maximum control but massive maintenance burden.
- **LiteLLM**: similar multi-provider abstraction but lacks subscription OAuth and embeddable SDK.
- **OpenRouter only**: simpler single integration but adds a middleman for all API calls and doesn't support local models.
