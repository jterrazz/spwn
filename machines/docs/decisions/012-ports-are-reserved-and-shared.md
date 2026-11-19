# ADR-012: Services are leased and published by the host

**Status:** Superseded by [ADR-013](013-every-port-leaves-through-serve.md)
**Date:** 2026-11-01

## Context

A dev server in a VM must be reachable from the owner's other machines and an iPhone, over https for a secure context, and the tool must know the URL it is reached at. The VM is not on the tailnet ([ADR-010](010-vms-are-reached-through-their-mac.md)), and a VM choosing ports on its host would let any VM take any port.

## Decision

Inside each VM, `station guest`, run by the host's daemon over its ssh connection, reports every listening TCP port, loopback and Docker's included; the VM never initiates and never picks a host port. The host owns the policy in `settings.json` (range 40000-40999, 20 per VM, ports below 1024 and 22 ignored, https mode) and leases a host port sticky on (VM, guest port). It publishes on 127.0.0.1 and the host's tailnet ip only, never `0.0.0.0`, carrying each connection over ssh `direct-tcpip`; an HTTP service gets `tailscale serve` https on its host port, with the path mode on 443 as the fallback if the app refuses arbitrary https ports, and `off`. A lease is released by delete, a temporary VM's poweroff, `station services release`, or 7 idle days. Catalogue profiles (Expo 8081, Vite 5173) reserve their port at create and carry the public URL into the VM's environment for its whole life; `station services url <port>` reserves any other port's URL before anything listens.

## Consequences

- A URL survives a VM's restart and a dev server's restart; deleting `leases.json` changes every URL.
- There is no forward verb: listening is the request.
- Two things stay to prove on a real host: https on any port of the range, and a listener on the tailnet ip without a firewall prompt.
