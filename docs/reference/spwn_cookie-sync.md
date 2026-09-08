---
title: "spwn cookie-sync"
slug: "spwn-cookie-sync"
---

## spwn cookie-sync

Browser extension that auto-syncs session cookies to the gate (status + providers)

### Synopsis

Browser extension that auto-syncs session cookies to the gate.

The extension watches your normal browser sessions on sites the gate
knows about (X today; LinkedIn etc. as elements are added) and pushes
the relevant session cookies to a locally-running spwn-gate. No
pairing, no secret — the gate listens on 127.0.0.1 only and accepts
just the cookie names each element declared, so other local processes
can't sneak unrelated cookies in.

Setup is two steps:

  1. spwn gate start                     # if not already running
  2. Open chrome://extensions/ → Developer mode → Load unpacked →
     select apps/spwn-cookie-sync/ in this repo

Then browse normally. The popup shows ● connected / ○ pending per
provider in real-time.

```
spwn cookie-sync [flags]
```

### Options

```
  -h, --help   help for cookie-sync
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn cookie-sync providers](./spwn_cookie-sync_providers.md)	 - List the providers the gate accepts cookie syncs for
* [spwn cookie-sync status](./spwn_cookie-sync_status.md)	 - Show registered providers and per-provider last-sync timestamps

