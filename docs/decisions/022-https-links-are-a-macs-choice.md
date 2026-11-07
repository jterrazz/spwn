# ADR-022: HTTPS links are a Mac's choice, through Serve

**Status:** Proposed
**Date:** 2026-11-07

## Context

[ADR-012](012-ports-are-reserved-and-shared.md) had an `https` `off` mode: an HTTP service published plain on the host's tailnet ip. [ADR-013](013-every-port-leaves-through-serve.md) sent every port through Tailscale Serve and dropped the tailnet-ip listener, so `off` had nothing left to mean and was read as `port`. The v4 menu bar (SPEC B5.13) gives each Mac the switch "HTTPS links on mac-mini": an https:// link with a valid certificate is what an iPhone needs for a secure context, but it costs the certificate's first issuance and puts the machine's name in Certificate Transparency logs.

## Decision

`ports.https` takes `off` again, still through Serve: every share of the Mac is served raw (`tailscale serve --bg --tcp=<port> tcp://127.0.0.1:<port>`), an HTTP server under `http://<MagicDNS name>:<port>`, and no certificate is asked for. `ports.set-https` (`station ports https on|off`) is its one writer, run on the Mac it targets; on keeps a `path` mode, else `port`. The engine serves its shares again in the new mode at its next tick, and its start removes an entry served in the other mode.

## Consequences

- Nothing of station listens on the tailnet: ADR-013 stands.
- A secure context (a service worker, the camera) is only on with HTTPS links on.
- The switch is per Mac: a VM inherits its Mac's links and cannot set them.
