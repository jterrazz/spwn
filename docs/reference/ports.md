# Ports

`station ports` lists every TCP port that listens on a Mac or a VM, each in the class of who reaches it. A workspace is not on the tailnet, so its Mac shares what it listens on: the Mac's engine reserves a port of its own for it, listens there on 127.0.0.1 only, and carries each connection into the VM over its one ssh connection. Which ports a VM may open, and to whom, are its [rights](#rights-and-audiences): a port that starts listening inside them is shared with their audience within seconds, one outside them is flagged and never shared. The tailnet reaches a private share through the Tailscale app alone: `tailscale serve` shares it, https for an HTTP server and raw TCP for any other, so nothing of station listens on the tailnet. A server's ports are never shared by themselves: its own tooling chooses its routes (its own Tailscale device, a tunnel), and `station ports share infra 8080` shares one on demand.

| Section               | Answers                                                     |
| --------------------- | ----------------------------------------------------------- |
| How a port gets a URL | The guest, the reserved port, the sharing                   |
| Rights and audiences  | What a VM may open, to whom, and what is flagged            |
| The commands          | `ports`, `ports share`, `ports stop-sharing`, `ports https` |
| The policy            | The `ports` group of `preferences.json`                     |
| Serve and HTTPS       | Serve on or off, HTTPS links, the `port` and `path` modes   |
| What an item sets     | How Expo and Vite learn their public URL                    |

## How a port gets a URL

Inside each running VM, `station guest` (hidden, run by the Mac's [engine](machines.md#the-engine) over its ssh connection) reports every listening TCP port, loopback included, and the ports Docker maps. The VM never opens a connection to its Mac and never picks a port of the Mac. Its own ssh port (22, which carries station) is never reported, nor a port below 1024 unless its rights open one there.

For each reported port its rights allow, the engine takes a **reserved port**: the lowest free port of the range (default 40000-40999), sticky on (VM, guest port), so the same dev server gets the same URL next week. It then **shares** it: one listener of the Mac on `127.0.0.1:<host port>` (never the tailnet ip, `0.0.0.0` or `::`: station refuses to bind every interface), each connection carried by ssh `direct-tcpip` to the address the VM reported, and, while [Serve](#serve-and-https) is on, the entry its audience asks: none for `local`, a `tailscale serve` entry on the tailnet for `private`, a `tailscale funnel` entry on the internet for `public`. A server bound to 127.0.0.1 in the VM needs no `--host`. tailscaled carries the tailnet's bytes, so a Mac's application firewall, which drops an unsigned binary's bytes on the tailnet ip with no prompt, never stands in the way, and who may reach a shared port is the tailnet policy's, never station's.

| Event                                    | What happens                                                        |
| ---------------------------------------- | ------------------------------------------------------------------- |
| the port starts listening                | reserved and shared, inside the rights; flagged outside them        |
| it listens in a server                   | listed, never shared by itself                                      |
| it stops listening                       | unshared after 5 s (a restart inside them keeps it); reserved kept  |
| a VM stops, then runs again              | unshared, then shared again on the same port of its Mac             |
| `machines delete`, an ephemeral poweroff | released, its serve entry removed                                   |
| `station ports stop-sharing`             | released and remembered: not shared again until a share             |
| 7 days without listening                 | released                                                            |
| the range is full, or the VM holds 20    | waiting, `◐` with the cause and the fix                             |
| the port of the Mac is busy              | the next free port, and the row says so                             |
| the Mac reboots                          | the engine shares every reserved port again as the guests report    |
| a server goes down                       | its ports stay listed, `unreachable while infra is down`, never red |
| a server boots again                     | its ports listed `starting` until its guest reports them            |

Every release (`machines delete`, `ports stop-sharing`, 7 idle days, a right [narrowed](#rights-and-audiences), a switch of [mode](#serve-and-https) that leaves a public port no Funnel port) removes its entry from the tailnet while Serve is on, in the mode it was served in, whether this engine served it or found it at its start, and never an entry at its port and path carried from another port of the Mac: one the owner put there by hand stays. A change to what a reserved port serves removes its old entry the same way before the new one is served: a move off a busy port of the Mac, its scheme once its server is told HTTP or not, an audience or Funnel port its rights change, a switch of mode.

The reserved ports live in `~/.local/state/station/leases.json`, a kept file: deleting it changes every URL. Each says who made it: `by: owner` for `ports share` on a Mac, `by: rights` for its VM's rights, which [judge it again](#rights-and-audiences) at each run of its VM; one made before this key, with none, is the owner's on a server (only `ports share` shared a server's port then) and the rights' on a workspace. Each has two URLs, one shape each: `local` `http://127.0.0.1:<port>` (no scheme for a raw TCP server) and `public`, the one `ports share` prints and an [item's env](#what-an-item-sets) exports; a share open to `local` has the local URL alone, and one open to `public` `https://<public host>:<Funnel port>`. With Serve on, `public` is `https://<public host>:<port>` for an HTTP server or one not yet detected (per [mode](#serve-and-https)) and `<public host>:<port>` for a raw TCP one; with Serve off it is the local URL, the only one there is. `<public host>` is the Mac's MagicDNS name.

## The commands

```sh
station ports                        # the Tailscale line, then every listening port in its class
station ports --all-machines         # every Mac's, asked over ssh
station ports --all                  # the ports that listen on their machine only, too
station ports --json                 # station.ports.list/v1
station ports share api 3000         # reserve 3000's URL now and print it; in a VM: station ports share 3000
station ports stop-sharing api 3000  # on a Mac; in a VM: station ports stop-sharing 3000
```

The list begins with the Tailscale line, then every port that listens on a Mac or a VM, in the class of who reaches it, piped tab-separated, each row saying its class in words, so a `grep` of one port still tells Private from Public:

| Section                     | Holds                                                                                                              | Columns                                                                        |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| Needs you (`✗`)             | a port reachable outside Tailscale (every interface, the LAN) that no declaration names public                     | machine, port, `needs you`, process, bind, `declared no`, the fix              |
| Public (`●` `◐`)            | a port a declaration names public, `●` when its route is seen, `◐` with what is missing when it is not             | machine, port, `public`, process, bind, the route and its proof                |
| Private (`●` `◐` `○`)       | what is shared with you on Tailscale: a VM's reserved port, a machine's own Tailscale device                       | machine, port, `private`, URL, state, process, the item whose env it sets, fix |
| Listens on its machine only | loopback and other ports only their machine reaches; one `N more listen on their machine only` line unless `--all` | machine, port, `this machine only`, process, bind                              |

A VM behind its Mac is reached through Tailscale only, so its ports are Private whatever it binds. Among the shared rows `●` is shared, `○` not listening (reserved, nothing listens), `◐` waiting (the VM shares 20 already, the range is taken, no Funnel port is left), local only (Serve off, Tailscale signed out, a refused serve), `issuing certificate`, not shared while its VM is stopped or while the engine is stopped, or listens, not allowed (outside its [rights](#rights-and-audiences)). In `--json` (`station.ports.list/v1`) a shared port's `state` is `shared`, `waiting`, `refused`, or `not-allowed` for a port that listens outside its VM's rights, which is never reserved: its `reserved` is 0 and its `detail` the rights it is outside. A Mac that does not answer is one `◐` row, never a failure. In `--json` a shared port carries `shared_at`, when the owner shared it (UTC): the menu bar marks a share of the last 8 s fresh.

`station ports declare <machine> <port> --via <route> [--url <link>]` declares a port public on the Mac that keeps the machine's declarations (`~/.config/station/public.json`, a VM's on its Mac), wherever it is asked from; the route is how the port is reached from elsewhere, and what the list looks for to call it seen:

| Route               | Seen when                                                                                                             |
| ------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `cloudflare-tunnel` | a `cloudflared` connector runs on that machine (its metrics listener; inside a VM, as its guest reads it)             |
| `tailscale-funnel`  | the Mac's Funnel state (`tailscale serve status --json`, `AllowFunnel`) lists the port; a VM's own device is not read |
| `lan`               | the port listens where the machine's network reaches it: every interface or a LAN address                             |

No `--via` and an unknown route are refused (exit 2) with the three routes. `undeclare` asks yes or no, then the port is red again while it is still reachable outside Tailscale; a port never declared is refused (exit 2). Inside a VM both are refused: its Mac keeps them. The Remote login setting (`sshd`), applied on a Mac with no battery (an always-on Mac, never a laptop), declares its `22` through `ports.declare` (`your LAN · Remote login · by Settings`), and `settings off sshd` takes it back. `--url` keeps the link the port is reached by from outside (`https://jterrazz.com`): the menu bar's public row names its domain beside the route, opens it, copies it and shows it as a QR; without one the row copies the port's address. In `--json` a declared port carries `declared: {via, url, by, seen, missing}`. In the menu bar, an exposed port's Declare public… is a hint: what declaring means and the line to copy, `station ports declare 5000 --via <route>` (its machine named when it is not This Mac), the route the owner's to name; station declares nothing from the panel. Share a port… types nothing: Of the running workspace (a popup that opens that workspace's sheet), then what listens there and is not shared yet, each with Share (`ports.share` on its Mac), or `Nothing listens in review-bot right now.`

Each Mac's reserved ports are one line at the end: `Reserved on This Mac: 2 of 1000 (40000-40999), 998 free`, one per Mac with `--all-machines`. In `--json` each Mac carries it in `range` (`mac`, `from`, `to`, `reserved`, `free`), read from its policy and reservations file, so it is there with the engine down.

The engine's exposure job reads what listens on its Mac and its VMs at its start, every 5 minutes and whenever a guest reports a change. A port that turns red is a finding of `station overview` (`✗ port 9000 is reachable outside Tailscale`) and is told once, as a `port_exposed` event (`<machine>: port <port> is reachable outside Tailscale`), then again only after it stopped being red.

`ports share` refuses port 22 and every port below 1024 (exit 2, `port 22 is never shared`), from a Mac and from a VM alike: the engine checks a guest's request itself. `ports share <port>` in a VM reserves the port at once, whether or not anything listens yet, so a value can be computed before the server starts: `$(station ports share 3000)`; a guest is never the owner, so its share is judged by its VM's rights: one they do not open is refused (exit 1, `port 5432 is outside the rights of linux-1 (open: 3000-3999; except: 3306)`, hint the owner's share on its Mac), and one they open takes their audience and is theirs to judge again. On a Mac it names its VM, `station ports share api 3000`, and needs the Tailscale app there: without it the share is refused before any port is reserved (`tailscale is required by port sharing`, exit 2, hint `station apps install tailscale`, which a terminal offers to run); with the Mac's engine away it is exit 1 (`Error: macbook is not connected to this VM`, `Hint: on macbook: station engine start`), and in a VM no Mac has connected to yet, so with no Mac name to give, every `ports` command is exit 1 (`Error: no Mac has connected to this VM yet`, `Hint: on the VM's Mac: station engine start`).

`ports stop-sharing` gives a reserved port back, after yes or no (`--yes` piped), and is remembered, so the next listen is not shared until `ports share`. A VM of another Mac is released on that Mac, over ssh. With the engine down, a Mac edits the reservations store itself.

## The policy

The Mac owns it, in `preferences.json`; every key is optional and only `https` has a command, [`ports https`](#serve-and-https):

```json
"ports": { "range": "40000-40999", "max_per_vm": 20, "https": "port" }
```

`range` is the ports of the Mac station picks a reserved port from; one that covers a Funnel port (443, 8443 or 10000) is read as the default, for a private share served there would be on the internet whenever it is funneled. Which ports of a VM are shared is no longer a preference: `min_port` and `ignored_ports` are its [rights](#rights-and-audiences) now, and `station init` removes them from `preferences.json`, their values not carried (a VM with no Machinefile shares as their defaults did).

The engine reads the file again when it changes. [Preferences](preferences.md) has the rest of `preferences.json`.

## Serve and HTTPS

Every shared port leaves the Mac through Tailscale Serve. Serve is **on** when `tailscale status --json` lists a certificate domain (`CertDomains`: the admin has enabled HTTPS certificates, which Serve needs); the tailnet logged out or absent is off. The engine reads it at its start, at each VM run and every 30 s; `station status` shows it as the `Tailscale Serve` gateway and `ports --json` as each Mac's `serve` (`on` or `off`).

To turn it on, once per tailnet: in the [admin console's DNS page](https://login.tailscale.com/admin/dns), MagicDNS on, then HTTPS Certificates (machine names then appear in Certificate Transparency logs), and Serve where the console asks for it.

With Serve on, an HTTP server (detected once per reserved port: a server that speaks first is not HTTP, else `HEAD /` must answer `HTTP/`) is served https with a valid certificate, so an iPhone on the tailnet opens it as a secure context; any other TCP server is served raw:

| Port                 | Serve line                                                                              | Public URL                                  |
| -------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------- |
| HTTP, `https` `port` | `tailscale serve --bg --https=<host port> http://127.0.0.1:<host port>`                 | `https://macbook.tail1234.ts.net:40000`     |
| HTTP, `https` `path` | `tailscale serve --bg --https=443 --set-path=/<vm>/<port> http://127.0.0.1:<host port>` | `https://macbook.tail1234.ts.net/api/8081/` |
| raw TCP              | `tailscale serve --bg --tcp=<host port> tcp://127.0.0.1:<host port>`                    | `macbook.tail1234.ts.net:40002`             |

**HTTPS links** are the Mac's choice, `station ports https on|off [--on <mac>]` (the call `ports.set-https`, refused inside a VM: its Mac serves its shares), kept as `https` in its `ports` group. Off (`"https": "off"`), every share is served raw, `tailscale serve --bg --tcp=<host port> tcp://127.0.0.1:<host port>`, an HTTP server under `http://macbook.tail1234.ts.net:40000` and no certificate asked for; on keeps a `path` mode, else `port`. The engine serves its shares again in the new mode at its next tick (a switch made while Serve is off, when Serve turns on), each unshared first in the mode it was served in, a share made before that tick served in the old mode and moved with the others, and at its start removes an entry served in the other one.

Unsharing runs the same line ending `off`. Any other word of `https` is read as `port`. A serve refused while Serve is on leaves the port shared locally: its row `◐` with `serve:` and tailscale's own line (never the CLI's `Warning:` lines), its public URL unchanged, the serve retried at its next share. In `path` mode a server that builds absolute URLs needs its base path set to `/<vm>/<port>/`.

tailscaled issues the Mac's certificate at the first https request, 34.5 s measured, so the engine warms it: at its start with Serve on, and when Serve turns on, it runs `tailscale cert --cert-file=- --key-file=- <host>` in the background, its output thrown away (`/dev/null` is refused as not a regular file), one at a time, within 120 s, never holding a share. Meanwhile every row served `--https` says `issuing certificate` (`◐`, its URL unchanged); `engine.log` then says `certificate for <host> ready in <d>` or `not issued:` with tailscale's line.

With Serve off no serve runs (one would block waiting for the admin): every reserved port is local only, every row of the Mac says `Serve is off on Tailscale: turn on HTTPS certificates in the Tailscale admin console`, and every public URL is the local one; `engine.log` says it once per change. Serve turning on serves every shared port at the next read, in the HTTPS mode of the Mac then: the mode is judged and the entries no reservation justifies are removed first.

Station owns only the serve entries on its range (both kinds), and the paths `/<vm>/<port>` of 443 (`path`) and the Funnel entries carried from its range, its VM still known or not (a VM deleted while Serve is off leaves its path), and, while Serve is on, removes the ones no reserved port justifies at the engine's start and when Serve turns on (a tailnet it cannot read then is read again at each tick until it is, said once in `engine.log`): a reservation justifies only the entry it makes in the Mac's mode (its port, scheme, audience and path, carried from its port of the Mac), and its VM's rights judge it first. It never touches another entry, nor replaces one: a public port whose Funnel port serves an entry station did not make (the owner's own `tailscale funnel`) is `◐ local only · serve: Funnel port 443 already serves an entry station did not make`, and a private share is refused the same way while its port is funneled (`port 443 is funneled by an entry station did not make: a private share on it would be public`), a path of 443 in `path` mode as a port of the range, for Funnel opens every path of a port. A Funnel port that still serves an entry station made for another share says so instead, `serve: Funnel port 443 still serves station's entry from port 40001 of this Mac`. A switch of mode unshares the private shares first, then moves the Funnel ports, then serves the private shares in the new mode, so in neither direction does a share wait on an entry of the old mode. A path below the root of a funneled port reads as the private share it was opened as.

A removal the tailnet fails (a `tailscale serve` that times out, or a status it cannot read for an entry this engine did not serve) is owed, said once in `engine.log`, `the entry of port 443 from port 40000 of this Mac is not removed yet: tried again at each tick`. Meanwhile the port of the Mac its entry carries from is given to no reservation, and a share at the same port and path replaces it. Each tick tries it again while the tailnet still carries it from that port and no reservation justifies it, then serves again the shares it held.

A port of the Mac a tailnet entry carries from is given to no other reservation. Before a share takes or binds its port, the engine reads the tailnet (its last reading while Serve is off or the read fails), and a reservation whose port an entry not its own carries from moves off it, `reserved port 40000 was carried on the tailnet: now 40001`. While Serve is on and the tailnet was never read, a new share is refused, `the tailnet's entries could not be read:` and tailscale's line, then shared at the tick that reads it.

## Rights and audiences

A VM's rights are the `ports` of its Machinefile ([Machinefiles](machinefiles.md)): which ports it may open, and to whom. Until a VM records the Machinefile it was made from, the engine reads the one named after it, from every source, at each run of the VM (and at its first read when the engine adopted it running: its hello, its first port, or the serve reconciliation at the engine's start):

```yaml
ports: { open: 3000-3999, to: private, except: [3306] }
```

| Rights                                                 | What the engine does                                                                                                                                  |
| ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| a port inside `open`                                   | reserved, then shared with `to`: its audience                                                                                                         |
| a port in `except`, or outside                         | flagged, `◐ 8443 listens, not allowed (open: 3000-3999)`, never reserved nor served; the owner's `ports share` shares it anyway, a guest's is refused |
| a workspace whose Machinefile says no `ports`, or none | `open: 1024-65535, to: private`: today's sharing                                                                                                      |
| a server whose Machinefile says no `ports`             | none: listed, never shared by itself, nothing flagged                                                                                                 |
| a Machinefile the decoder refuses                      | none: every port flagged with the refusal, its file and line                                                                                          |

Each read of the rights judges the VM's reservations again. One the rights made that they no longer allow (a port removed or excepted, a Machinefile refused, a public port no Funnel port is left for) is unshared, its Funnel entry included, and released, and its port is flagged while it listens; one whose audience or Funnel port changed is unshared, then shared again with the new one. One the owner made with `ports share` is kept, private when the rights do not give it their audience. So a narrowed right never keeps a port open until its 7 idle days.

| Audience  | Who reaches the port                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `local`   | its Mac alone: the loopback listener, no serve; its link is the local one                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `private` | your devices on the tailnet: `tailscale serve`, as [Serve and HTTPS](#serve-and-https) says                                                                                                                                                                                                                                                                                                                                                                                                       |
| `public`  | the internet: `tailscale funnel --bg --https=<funnel port> http://127.0.0.1:<port>` (`--tcp` for a raw server); Funnel answers on 443, 8443 and 10000 only, so `public` takes a list of at most three ports, never a range, each mapped in order onto them, and a Funnel port another VM's share holds is refused. In `path` mode 443 carries the private paths, so it is never a public port: the list maps onto 8443 and 10000, and a third port is refused (`no Funnel port is left for 3002`) |

The Audience contract (`src/ports/contract/audience.go`) carries the three; its adapters are `tailscale` and the world's `fake`. A row of a port names its label from the catalog's items' `ports` (5173 `Vite`, 3000 `Node or Next`, 8081 `Expo`, 5432 `Postgres`) as a hint of what probably serves it, never a decision.

## What an item sets

A server that must know its own URL gets it from its environment. An item's `ports` ([Catalog](catalog.md)) names the port it serves on and the variables that carry the URL: at each run of a VM (create, `start`, `restart`, its own run at boot; never when it adopts a running VM) the engine reserves every such port inside the VM's rights, before anything listens there, and renders the variables from Serve's state then, so they are fixed for a run, and a persistent VM picks up the public URLs at its first start after Serve is turned on. The VM's shell (`station shell zsh`, from its ports' env file) exports them:

| Item | Guest port | Variable                                 | Value                                                   |
| ---- | ---------- | ---------------------------------------- | ------------------------------------------------------- |
| expo | 8081       | `EXPO_PACKAGER_PROXY_URL`                | the public URL, `https://macbook.tail1234.ts.net:40000` |
| vite | 5173       | `__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS` | the public name, `macbook.tail1234.ts.net`              |
| any  |            | `STATION_PUBLIC_HOST`                    | the public name, with any of them                       |

With Serve off they hold the local URL, `http://127.0.0.1:<host port>`, and the public name `127.0.0.1`, and `machines create` says so on a `◐ ports` row. So `npx expo start` in a VM writes the shared https URL into its manifests, and Vite accepts requests for the public name, with no flag. A server on another port composes its URL itself: `PORT=3000 BASE_URL=$(station ports share 3000) npm start`. Same-origin traffic (HMR websockets, relative URLs) needs nothing: it rides the same port of the Mac. A server whose setting has no environment variable gets its `ports` in its item.

## Related

- [Machines](machines.md): the engine, and how a VM is reached by ssh.
- [The API](api.md): the engine's calls `ports.share` and the reservations ride.
- [Preferences](preferences.md): `preferences.json`.
- [Glossary](../concepts/glossary.md): port, reserved port, share, right, audience, public name.
