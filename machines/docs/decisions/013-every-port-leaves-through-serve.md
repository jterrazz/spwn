# ADR-013: Every service leaves through Serve

**Status:** Superseded by [ADR-018](018-two-port-classes.md)
**Date:** 2026-11-01

Supersedes [ADR-012](012-ports-are-reserved-and-shared.md).

## Context

[ADR-012](012-ports-are-reserved-and-shared.md) published each lease on 127.0.0.1 and on the host's tailnet ip, with `tailscale serve` only for HTTP. The real run on two hosts showed nothing left the host: Serve and HTTPS certificates were off on the tailnet, and the macOS Application Firewall dropped the ad-hoc signed station binary's bytes on the tailnet ip, with no prompt. A listener of station's own on the tailnet would need a firewall exception or a signed binary on every host.

## Decision

Station listens on 127.0.0.1 only; the tailnet-ip listener is dropped. tailscaled publishes every lease: an HTTP service as `tailscale serve --https=<port>`, any other TCP service as `tailscale serve --tcp=<port> tcp://127.0.0.1:<port>`. Who may reach a lease is the tailnet policy's, never station's. When Serve is off, every service is local only and every row says so, with the admin step to take. The rest of ADR-012 stands: the guest reports, the host owns the policy and the sticky lease, connections ride ssh `direct-tcpip`, profiles reserve their URL, and a lease is released by delete, a temporary poweroff, `station services release` or 7 idle days.

## Consequences

- No firewall exception and no signing requirement: tailscaled, not station, carries the tailnet's bytes.
- A service is reachable from another machine only once the owner enables Serve and HTTPS certificates in the Tailscale admin console, which puts the host names in Certificate Transparency logs.
- A raw TCP service gets no https: its public address is `<host>:<port>`.
- One thing stays to prove on a real host: another machine and an iPhone reaching a VM's Vite over https and a raw TCP service over `--tcp`.
