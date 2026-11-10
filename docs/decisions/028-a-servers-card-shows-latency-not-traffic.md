# ADR-028: A server's card shows its latency, not its traffic

**Status:** Accepted
**Date:** 2026-11-08

Written 2026-11-10 from DECISIONS D-v4 (round 14), SPEC v4 §B5.4 and §D3, and `src/pages/menubar/server.go`.

## Context

The prototype's server page shows requests per minute beside the latency. Station cannot see a server's traffic: what it measures is its own health check, from the Mac the server runs on. SPEC §D3 offered the owner a metrics URL a server could declare (Prometheus text, read over the tailnet from its Mac) or no traffic line at all. The owner's word (D-v4, 2026-11-08): "ok", the card shows latency only.

## Decision

- A server's page has a Latency card: the site its check asks, its latency now, its p50 over the last hour and its peak with when. It has no request rate, and no server declares a metrics URL.

## Consequences

- No call reads a server's traffic, and no third-party or in-guest metrics endpoint is reached.
- A declared metrics URL stays an option for later ([TODO](../../TODO.md)).
