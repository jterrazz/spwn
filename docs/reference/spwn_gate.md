---
title: "spwn gate"
slug: "spwn-gate"
---

## spwn gate

Manage the host-side credential broker (start/stop/status/logs/restart)

### Synopsis

Manage the host-side credential broker.

The gate is a long-running container that holds OAuth credentials,
hosts upstream MCP servers, and exposes them to world containers as
authenticated MCP endpoints. World containers never see credentials —
they get tiny CLI wrappers that route through the gate.

Auto-started on first `spwn up`; explicit lifecycle commands let you
inspect, restart, or troubleshoot.

### Options

```
  -h, --help   help for gate
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn gate logs](./spwn_gate_logs.md)	 - Stream gate logs (docker logs)
* [spwn gate restart](./spwn_gate_restart.md)	 - Stop + start the gate container; --rebuild forces a fresh image build first
* [spwn gate start](./spwn_gate_start.md)	 - Build the gate image (if missing) and start the container
* [spwn gate status](./spwn_gate_status.md)	 - Show whether the gate is running
* [spwn gate stop](./spwn_gate_stop.md)	 - Stop and remove the gate container (image stays on disk)

