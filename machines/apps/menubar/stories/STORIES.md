# Spwn Menu stories: the catalogue of every screen × state

Source of truth: for the panel's material, width and colours, [ADR-053](../../../docs/decisions/053-the-panel-is-untinted-glass-over-a-grey-ground.md), which replaces the prototype's 340 pt, its tint and its white cards; for everything else, the accepted prototype `proto/index.html` (with `proto/PATTERNS.md` and `ia/PROPOSAL.md`), driven headlessly (Chromium headless shell + Playwright, 1440×1000 @2x). This file is decision K3's catalogue (DECISIONS.md round 14): every story here becomes one page fixture (the page-language JSON the engine sends) with this id, rendered by the Spwn Menu Catalog window, by the light/dark snapshot goldens, and by the HTML gallery. A prototype state without a story fails the coverage test. The prototype drew agents (rows, waits, steps, a fleet page); Machines keeps none ([ADR-052](../../../docs/decisions/052-station-keeps-no-agents.md)), so a capture's agent parts are the prototype's alone, and each story's golden is its reference.

Each story gives: what it shows; the data the page needs (what the engine must send); the interactions on screen (read from the live DOM: label → verb(argument), shortcut in brackets); the animation entering it (the gesture that leads here, recorded from the Web Animations calls and CSS animations it fired) and leaving it (recorded from esc, or from the inverse gesture named); and its capture `stories/<id>.png` (dark) plus `stories/<id>.light.png` where light matters.

**Motion tokens** (the prototype's three, shared with CSS): snappy = 200 ms cubic-bezier(.3,.9,.3,1), for small controls and menus; smooth = 340 ms cubic-bezier(.32,.72,0,1), for push, sheet, Find and height morphs; bouncy = 460 ms spring (linear() with 11 % overshoot), for a success check only. Under Reduce Motion every movement becomes an opacity dissolve (150 to 240 ms ease-in-out) and heights change in 200 ms.

**Global keys** (every panel story): esc pops then closes; ⌘K Find (or a letter on All machines); ↑ ↓ ↩ ⇥ move and open, ⌘↩ attaches a VM; ⌘N New VM, ⌘T Terminal; ⌘0 All machines, ⌘1…3 a Mac; ⌘I ⌘P ⌘U pick Activity, Ports, Accounts (a segment on a machine page, the fleet page on home); ⌘E Awake; ⌘[ and ← pop too; Space presses a focused row; ⌘↩ on a down server restarts it; ↩ on a workspace page with nothing highlighted attaches; ⌘K with the panel closed opens it on Find. Pointer: a click on the status item toggles the panel, a click outside it closes it, a pointer press clears the keyboard highlight; a right-click on a VM, Mac, grant or the Awake tile opens its menu.

**Shared fixture**: unless a story says otherwise, the data is the default fleet: This Mac (MacBook Pro M4 Max 48 GB: linux-1 and agent-7f running ephemeral workspaces, api stopped persistent), mac-mini (Mac mini M4 Pro 64 GB, never sleeps: infra server, linux-3 running ephemeral workspace, review-bot running persistent workspace), macbook-air (asleep, seen 2 hr ago); 5 shared ports and 6 listeners; sign-ins per Mac and per VM (npm signed out on This Mac) and the grants made from This Mac (GitHub and Claude to linux-1); upkeep on This Mac (firewall off, 1 failed login item, 2 updates) and linux-1 (2 updates); This Mac awake until 18:00. Times (HH:MM, "N min") are live and differ per capture.

## Index

- **Menu bar and panel shell** (6): `menubar.down`, `menubar.restarting`, `menubar.fine`, `menubar.open`, `menubar.stopped`, `menubar.connecting`
- **All machines (home)** (12): `home.needs`, `home.hover`, `home.keyboard`, `home.down`, `home.restarting`, `home.down-and-exposed`, `home.loading`, `home.loaded`, `home.fine`, `home.empty`, `home.awake-off`, `home.capped`
- **Needs you (fleet list)** (5): `needs.all`, `needs.empty`, `needs.updating`, `needs.updated`, `needs.catalog-refused`
- **A Mac** (20): `mac.this.overview`, `mac.this.upkeep`, `mac.this.activity`, `mac.this.ports`, `mac.this.accounts`, `mac.this.exposed`, `mac.mini.overview`, `mac.mini.activity`, `mac.mini.activity.6m`, `mac.mini.activity.24h`, `mac.mini.activity.scrub`, `mac.mini.ports`, `mac.mini.ports.https`, `mac.mini.accounts`, `mac.mini.down`, `mac.mini.loading`, `mac.mini.creating`, `mac.mini.created`, `mac.air.asleep`, `mac.this.empty`
- **A workspace** (8): `workspace.overview`, `workspace.ephemeral`, `workspace.working.activity`, `workspace.working.ports`, `workspace.working.accounts`, `workspace.waiting.ports`, `workspace.stopped`, `workspace.started`
- **A server** (8): `server.healthy`, `server.health-unfolded`, `server.activity`, `server.down`, `server.restarting`, `server.restarting.late`, `server.recovered`, `server.stopped`
- **Ports (fleet and scoped)** (5): `ports.all`, `ports.all.exposed`, `ports.all.down`, `ports.shared-new`, `ports.audiences`
- **Accounts** (3): `accounts.all`, `accounts.signing-in`, `accounts.signed-in`
- **Awake** (3): `awake.all`, `awake.off`, `awake.4h`
- **Sheets** (14): `sheet.new-vm`, `sheet.new-vm.options`, `sheet.new-vm.server`, `sheet.new-vm.first`, `sheet.grant`, `sheet.grant.vm`, `sheet.share-port`, `sheet.share-port.vm`, `sheet.share-port.empty`, `sheet.cleanup.mac`, `sheet.cleanup.all`, `sheet.hint.add-mac`, `sheet.hint.firewall`, `sheet.hint.declare-public`
- **Confirmations (alerts)** (14): `alert.restart-server`, `alert.stop-server`, `alert.stop-server.typed`, `alert.delete-server`, `alert.stop-vm`, `alert.delete-vm.ephemeral`, `alert.delete-vm.persistent`, `alert.stop-all`, `alert.stop-all.server`, `alert.restart-mac`, `alert.forget-mac`, `alert.remove-grant`, `alert.stop-share`, `alert.let-sleep`
- **Menus** (21): `menu.station`, `menu.context.workspace`, `menu.context.workspace.copy`, `menu.context.workspace.share`, `menu.context.stopped`, `menu.context.server`, `menu.context.server.down`, `menu.context.mac`, `menu.context.mac.asleep`, `menu.context.grant`, `menu.awake`, `menu.page.mac`, `menu.page.this-mac`, `menu.page.mac.asleep`, `menu.page.workspace`, `menu.page.server`, `menu.page.ports`, `menu.new-vm.on`, `menu.new-vm.create`, `menu.grant.to`, `menu.share-port.of`
- **Popovers, tooltips, HUD** (5): `pop.qr`, `pop.qr.public`, `pop.info`, `pop.hud.copy`, `pop.hud.terminal`
- **Find (Cmd-K)** (10): `find.empty`, `find.down`, `find.port`, `find.machine`, `find.action`, `find.awake`, `find.keyboard`, `find.more-open`, `find.nomatch`, `find.type-to-find`
- **Transitions (frozen mid-flight)** (10): `transition.push`, `transition.pop`, `transition.jump`, `transition.segment`, `transition.sheet`, `transition.alert`, `transition.find`, `transition.menu`, `transition.row-leave`, `transition.reduce-motion.push`
- **Phone (the explainer page)** (1): `phone.explainer`

## Menu bar and panel shell

### `menubar.down`

![menubar.down](stories/menubar.down.png) ![menubar.down light](stories/menubar.down.light.png)

- **Capture**: [dark](stories/menubar.down.png), [light](stories/menubar.down.light.png)
- **Shows**: A server down or a port exposed: the status item's two floors faded to 34 % under its dot, grown and solid, in the bar's one ink, never red; aria-label "Spwn Menu, infra down".
- **Data**: a server in state down
- **Enters**: #hud: CSS hud 2600 ms linear

### `menubar.restarting`

![menubar.restarting](stories/menubar.restarting.png)

- **Capture**: [dark](stories/menubar.restarting.png)
- **Shows**: A server boots: the two floors light in turn, bottom then top, each 0.6 s after the other, a 1.2 s loop that runs under Reduce Motion too, the dot solid; aria-label "infra restarting, n of 4 checks".
- **Data**: a server in state restarting with its step index
- **Enters**: .busy: CSS spin 800 ms linear; #hud: CSS hud 2600 ms linear

### `menubar.fine`

![menubar.fine](stories/menubar.fine.png) ![menubar.fine light](stories/menubar.fine.light.png)

- **Capture**: [dark](stories/menubar.fine.png), [light](stories/menubar.fine.light.png)
- **Shows**: Nothing is down: the two floors and the dot on top, plain; aria-label "all machines are fine".
- **Data**: needs() empty
- **Enters**: #hud: CSS hud 2600 ms linear

### `menubar.open`

![menubar.open](stories/menubar.open.png)

- **Capture**: [dark](stories/menubar.open.png)
- **Shows**: The status item while its panel is open: the plain mark on the system's highlight (pressed capsule), the coach is gone.
- **Data**: none
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 33%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 69%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: #panel: opacity+transform 0 scale(.96) → 1 none 340 ms smooth; #sItem: CSS background-color 120 ms

### `menubar.stopped`

- **Shows**: The engine stopped or station not installed: the floors and the dot at 45 % opacity under a solid slash cut out of them, so silence is never mistaken for all good; aria-label "station: engine stopped".
- **Data**: the engine's socket absent

### `menubar.connecting`

- **Shows**: A connection being made: the bottom floor solid, the top floor and the dot at 32 % opacity; aria-label "station: connecting".
- **Data**: the socket present, no page yet

## All machines (home)

### `home.needs`

![home.needs](stories/home.needs.png) ![home.needs light](stories/home.needs.light.png)

- **Capture**: [dark](stories/home.needs.png), [light](stories/home.needs.light.png)
- **Shows**: What needs you leads Home: port 5000 exposed, its red dot and Review over This Mac, then the servers, VMs and Macs, each running row ending in its CPU spark.
- **Data**: fleet snapshot (macs, vms, ports, listeners, accounts, grants, upkeep, awake) + live metrics sample; needs() ranked; upkeep() count; servers with health and uptime days; per-Mac CPU, free memory, VM count
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 37%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 60%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: #panel: opacity+transform 0 scale(.96) → 1 none 340 ms smooth; #sItem: CSS background-color 120 ms
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms (esc)

### `home.hover`

![home.hover](stories/home.hover.png)

- **Capture**: [dark](stories/home.hover.png)
- **Shows**: Hovering a VM row: it tints, Open in Terminal fades in beside its CPU.
- **Data**: as home.needs
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 35%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 65%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: .row.two: CSS background-color 120 ms; .qa: CSS opacity 120 ms
- **Leaves**: .row.two: CSS background-color 120 ms; .qa: CSS opacity 120 ms (inverse gesture)

### `home.keyboard`

![home.keyboard](stories/home.keyboard.png)

- **Capture**: [dark](stories/home.keyboard.png)
- **Shows**: The keyboard's one highlight: the arrows and Tab walk it through every row in reading order, here on the Accounts row.
- **Data**: as home.needs
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 34%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 58%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: .place.nav: CSS background-color 120 ms

### `home.down`

![home.down](stories/home.down.png) ![home.down light](stories/home.down.light.png)

- **Capture**: [dark](stories/home.down.png), [light](stories/home.down.light.png)
- **Shows**: infra is down: its row under Needs you with Restart, and infra down since HH:MM under Servers.
- **Data**: server state down with cause, time, last seen; ports of the server marked unreachable
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Restart" → restart(infra); "2 more need you · review-bot, linux-1" → allNeeds; "Ports, 2 unreachable · 5 shared" → push(ports:all) [2 unreachable · 5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 37%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 70%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra down since 14:28" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 down2 agents 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; #hud: CSS hud 2600 ms linear (esc)

### `home.restarting`

![home.restarting](stories/home.restarting.png)

- **Capture**: [dark](stories/home.restarting.png)
- **Shows**: infra restarting: its row's ring counts the steps.
- **Data**: server restarting: step name, step index of 4 (Running, Has an address, SSH answers, Check passes)
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "2 more need you · review-bot, linux-1" → allNeeds; "Ports, 2 wait for the restart · 5 shared" → push(ports:all) [2 wait for the restart · 5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 41%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 70%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra restarting · 2 of 4" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: .hero-top: opacity+transform 0 translateY(-3px) → 1 none 340 ms smooth; .hero-rows: opacity 0 → 1 340 ms smooth +60 ms; svg: CSS color 120 ms; .sitext: CSS max-width 340 ms cubic-bezier(0.32, 0.72, 0, 1); .sitext: CSS opacity 120 ms; .busy: CSS spin 800 ms linear; .spin: CSS spin 800 ms linear; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; .busy: CSS spin 800 ms linear; .spin: CSS spin 800 ms linear (esc)

### `home.down-and-exposed`

![home.down-and-exposed](stories/home.down-and-exposed.png)

- **Capture**: [dark](stories/home.down-and-exposed.png)
- **Shows**: Two reds under Needs you: infra down with Restart, then the exposed port with Review; red rows are never merged.
- **Data**: two red needs
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Restart" → restart(infra); "3 more need you · This Mac, review-bot, linux-1" → allNeeds; "Ports, 1 exposed · 5 shared" → push(ports:all) [1 exposed · 5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 30%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 52%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra down since 14:29" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 down2 agents 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; #hud: CSS hud 2600 ms linear (esc)

### `home.loading`

![home.loading](stories/home.loading.png)

- **Capture**: [dark](stories/home.loading.png)
- **Shows**: mac-mini still answering: a spinner beside the title, its rows as last read.
- **Data**: per-Mac freshness (last seen) and a pending refresh
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 31%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra seen 40 s ago · checking…" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini seen 40 s ago · checking… 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: .spin.sm: CSS spin 800 ms linear; .dot.ring: CSS ping 600 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; .spin.sm: CSS spin 800 ms linear (esc)

### `home.loaded`

![home.loaded](stories/home.loaded.png)

- **Capture**: [dark](stories/home.loaded.png)
- **Shows**: mac-mini answered: its rows drawn at full strength again.
- **Data**: as home.needs
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 38%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 71%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: .spin.sm: CSS spin 800 ms linear; .dot.ring: CSS ping 600 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; .dot.ping: CSS ping 600 ms linear; .row.two: CSS revealIn 340 ms linear (esc)

### `home.fine`

![home.fine](stories/home.fine.png) ![home.fine light](stories/home.fine.light.png)

- **Capture**: [dark](stories/home.fine.png), [light](stories/home.fine.light.png)
- **Shows**: Nothing needs you: no Needs you section, This Mac, then each kind as it runs.
- **Data**: needs() and upkeep() empty; fleet facts
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 4 granted" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "linux-1This Mac running tests · 41/58 pass 34%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 54%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-botmac-mini review done, idle" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 3 VMs1 working 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 3 VMs1 working 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: .badge.none: CSS max-width 340 ms cubic-bezier(0.32, 0.72, 0, 1); #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1); #hud: CSS hud 2600 ms linear (esc)

### `home.empty`

![home.empty](stories/home.empty.png) ![home.empty light](stories/home.empty.light.png)

- **Capture**: [dark](stories/home.empty.png), [light](stories/home.empty.light.png)
- **Shows**: First run: This Mac alone, no VM: the empty state with New VM.
- **Data**: one Mac, zero VMs
- **Interactions**: "More options for Spwn Menu" → gear; "New VM on This Mac" → newvm(thismac); "This Mac 0 VMs 35 GB free" → push(mac:thismac); "Add a Mac to run VMs on it" → addMac; "Find a machine, a port, an action… ⌘K" → find
- **Enters**: path: CSS draw 340 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; path: CSS draw 340 ms linear (esc)

### `home.awake-off`

![home.awake-off](stories/home.awake-off.png)

- **Capture**: [dark](stories/home.awake-off.png)
- **Shows**: Awake switched off from its row: the switch off, Off.
- **Data**: awake.on, until, from
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, This Mac may sleep" → push(awake:all) [Awake · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 31%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 73%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: .place.nav: CSS background-color 120 ms; .atog: CSS background-color 120 ms; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: opacity+transform 1 none → 0 scale(.98) 200 ms snappy; #sItem: CSS background-color 120 ms; #hud: CSS hud 2600 ms linear (esc)

### `home.capped`

![home.capped](stories/home.capped.png)

- **Capture**: [dark](stories/home.capped.png)
- **Shows**: A fleet taller than the panel's 720 pt cap: the body is cut at the cap and fades out, as it scrolls under the header.
- **Data**: as home.down-and-exposed
- **Interactions**: as home.down-and-exposed

## Needs you (fleet list)

### `needs.all`

![needs.all](stories/needs.all.png) ![needs.all light](stories/needs.all.light.png)

- **Capture**: [dark](stories/needs.all.png), [light](stories/needs.all.light.png)
- **Shows**: The full list: "Needs you · All machines", infra down first, then review-bot waiting, every row with its verb; then Upkeep (firewall off, npm needs sign-in, a login item failed, updates to install).
- **Data**: needs() and upkeep(), unscoped
- **Interactions**: "All machines" → back; "Review" → push(vm:review-bot); "How to" → howto(firewall); "Sign in" → signin(thismac:npm); "Show" → goUpkeep(thismac); "Update" → updateAll
- **Enters**: #panel: height 700px → 438px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth
- **Leaves**: #panel: height 438px → 700px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `needs.empty`

![needs.empty](stories/needs.empty.png)

- **Capture**: [dark](stories/needs.empty.png)
- **Shows**: Nothing needs you: one empty card; facts "nothing blocks a server".
- **Data**: needs() and upkeep() empty
- **Interactions**: "All machines" → back
- **Enters**: .badge.none: CSS max-width 340 ms cubic-bezier(0.32, 0.72, 0, 1); #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 172px → 637px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `needs.updating`

![needs.updating](stories/needs.updating.png)

- **Capture**: [dark](stories/needs.updating.png)
- **Shows**: Update pressed on the upkeep row "4 updates to install": the verb reads "Updating…" (busy, t3) while the updates install; the row stays in place.
- **Data**: upkeep updates; the install job running
- **Interactions**: "All machines" → back; "Review" → push(vm:review-bot); "How to" → howto(firewall); "Sign in" → signin(thismac:npm); "Show" → goUpkeep(thismac); "Updating…" → updateAll
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 342px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .findcap: CSS background-color 56 ms; .fwrap.morph: CSS morph 340 ms linear (esc)

### `needs.updated`

![needs.updated](stories/needs.updated.png)

- **Capture**: [dark](stories/needs.updated.png)
- **Shows**: Updates installed: the upkeep row collapses (height to zero, 10 pt slide, fade) and HUD "Everything is up to date"; the Upkeep count drops.
- **Data**: upkeep updates back to 0
- **Interactions**: "All machines" → back; "Review" → push(vm:review-bot); "How to" → howto(firewall); "Sign in" → signin(thismac:npm); "Show" → goUpkeep(thismac)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 310px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `needs.catalog-refused`

![needs.catalog-refused](stories/needs.catalog-refused.png) ![needs.catalog-refused light](stories/needs.catalog-refused.light.png)

- **Capture**: [dark](stories/needs.catalog-refused.png), [light](stories/needs.catalog-refused.light.png)
- **Shows**: A file of the owner's catalog is refused: the upkeep row "A file of yours is refused on macbook-pro · 1 file refused" with its warning glyph and its verb Show; unfolded, it names the file's problem (its line and the unknown key) and the file to fix, saying the built-in packages are read meanwhile.
- **Data**: needs() with a refused catalog file (its worded problem and its path)
- **Interactions**: "All machines" → back; "Review" → push(vm:review-bot); "How to" → howto(firewall); "Sign in" → signin(thismac:npm); "Show" → goUpkeep(thismac)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 342px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

## A Mac

### `mac.this.overview`

![mac.this.overview](stories/mac.this.overview.png) ![mac.this.overview light](stories/mac.this.overview.light.png)

- **Capture**: [dark](stories/mac.this.overview.png), [light](stories/mac.this.overview.light.png)
- **Shows**: This Mac: back "All machines", laptop glyph, title, New VM button, More menu; facts "MacBook Pro · M4 Max · 48 GB · up 3 days"; segments Overview/Activity/Ports/Accounts; Needs you card (Claude on linux-1 ends…); gauges CPU, Memory (GB free + pressure bar), Network with sparklines; VMs card (in use GB) with linux-1, agent-7f, api; care card: Awake row (switch, until 18:00) and Upkeep row (4 items, Update).
- **Data**: mac model, memory, uptime; needs(mac); metrics cpu/mem/net series; vmsOf(mac); awake; upkeep(mac)
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "CPU 17%" → seg(activity); "Memory 23 GB free" → seg(activity); "Network 1.0 MB/s" → seg(activity); "linux-1 running tests · 41/58 pass 40%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "agent-7f finished · tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "api stopped · persistent" → push(vm:api); "Keep This Mac awake" → awakeToggle; "Upkeep firewall off · 3 more" → unfold(upkeep:thismac); "How to" → howto; "Copy fix" → copyCmd(launchctl bootout system/com.docker.vmnetd); "Sign in" → signin(thismac:npm); "Update" → updateAll
- **Enters**: #panel: height 689px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth
- **Leaves**: #panel: height 506px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.this.upkeep`

![mac.this.upkeep](stories/mac.this.upkeep.png)

- **Capture**: [dark](stories/mac.this.upkeep.png)
- **Shows**: This Mac's upkeep unfolded in place: firewall off (How to), npm needs sign-in (Sign in), 1 login item failed (Show), 2 updates (Update).
- **Data**: upkeep(thismac) items with their verbs
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "CPU 19%" → seg(activity); "Memory 23 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "linux-1 running tests · 41/58 pass 30%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "agent-7f finished · tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "api stopped · persistent" → push(vm:api); "Keep This Mac awake" → awakeToggle; "Upkeep firewall off · 3 more" → unfold(upkeep:thismac); "How to" → howto; "Copy fix" → copyCmd(launchctl bootout system/com.docker.vmnetd); "Sign in" → signin(thismac:npm); "Update" → updateAll
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 700px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.this.activity`

![mac.this.activity](stories/mac.this.activity.png)

- **Capture**: [dark](stories/mac.this.activity.png)
- **Shows**: Activity of This Mac: window picker 6 min / 1 hr / 24 hr, CPU chart with its peak, network mirror chart (up/down), memory, then top processes (VMs link to their pages), sorted live by CPU.
- **Data**: cpu series per window (72x5 s, 60x1 min, 48x30 min), net up/down series, memory series, top processes
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "linux-1 · 4 cores 30%" → push(vm:linux-1); "agent-7f · 4 cores 3%" → push(vm:agent-7f)
- **Enters**: #panel: height 506px → 700px 340 ms smooth; .pb: opacity+transform 0 translateX(10px) → 1 none 200 ms snappy; .: CSS color 120 ms; .sel: CSS color 120 ms
- **Leaves**: #panel: height 700px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.this.ports`

![mac.this.ports](stories/mac.this.ports.png)

- **Capture**: [dark](stories/mac.this.ports.png)
- **Shows**: Ports of This Mac and its VMs: Private on your tailnet, grouped by VM (linux-1:5173 Vite, :3000 API dev server) with Copy and QR; "Listening, not shared" suggestions (4321 Astro, 9229 Node inspector, 6006 Storybook) with Share; HTTPS links switch for this Mac.
- **Data**: ports scoped to the Mac, listeners per VM, the Mac's HTTPS setting
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Share Astro" → sharePort(linux-1:4321); "Share Node inspector" → sharePort(linux-1:9229); "HTTPS links on This Mac (switch)" → toggleHttps
- **Enters**: #panel: height 506px → 468px 340 ms smooth; .pb: opacity+transform 0 translateX(10px) → 1 none 200 ms snappy; .: CSS color 120 ms; .sel: CSS color 120 ms
- **Leaves**: #panel: height 468px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.this.accounts`

![mac.this.accounts](stories/mac.this.accounts.png)

- **Capture**: [dark](stories/mac.this.accounts.png)
- **Shows**: Accounts of This Mac: Needs sign-in (npm, Sign in ↗); On This Mac (GitHub, Claude, Cloudflare); then what its VM linux-1 holds (Claude, GitHub granted from This Mac).
- **Data**: sign-ins of the Mac and of its VMs; grants made from it
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Sign in" → signin(thismac:npm); "Grant a password…" → grant
- **Enters**: #panel: height 506px → 600px 340 ms smooth; .pb: opacity+transform 0 translateX(10px) → 1 none 200 ms snappy; .: CSS color 120 ms; .sel: CSS color 120 ms
- **Leaves**: #panel: height 600px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.this.exposed`

![mac.this.exposed](stories/mac.this.exposed.png)

- **Capture**: [dark](stories/mac.this.exposed.png)
- **Shows**: This Mac while port 5000 is exposed: its Needs you card puts the red row "Port 5000 exposed" (shield) first with Review; the rest of Overview is unchanged.
- **Data**: needs(mac) with the exposed listener ranked first
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → goExposed(thismac); "CPU 16%" → seg(activity); "Memory 23 GB free" → seg(activity); "Network 0.8 MB/s" → seg(activity); "linux-1 running tests · 41/58 pass 27%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "agent-7f finished · tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "api stopped · persistent" → push(vm:api); "Keep This Mac awake" → awakeToggle; "Upkeep firewall off · 3 more" → unfold(upkeep:thismac); "How to" → howto; "Copy fix" → copyCmd(launchctl bootout system/com.docker.vmnetd); "Sign in" → signin(thismac:npm); "Update" → updateAll
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 538px → 669px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.overview`

![mac.mini.overview](stories/mac.mini.overview.png) ![mac.mini.overview light](stories/mac.mini.overview.light.png)

- **Capture**: [dark](stories/mac.mini.overview.png), [light](stories/mac.mini.overview.light.png)
- **Shows**: mac-mini: Mac mini · M4 Pro · 64 GB · up 21 days; Needs you (review-bot waits); gauges CPU, Memory, Network; VMs "36 GB in use": infra (healthy · up 12 days), linux-3 (working), review-bot; care card: Awake "Never sleeps · hosts infra, which pauses if it sleeps" (switch), Upkeep "all fine" unfolding to Settings (FileVault, firewall, SIP · 12 of 12 on), Background (9 jobs fine), Apps (all 38 up to date), Disk (VMs 186 GB · caches 22 GB · 731 GB free), Ready to host servers (6 of 6).
- **Data**: as mac.this.overview for a remote Mac; policy never-sleeps; servers it hosts
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → push(vm:review-bot); "CPU 49%" → seg(activity); "Memory 19 GB free" → seg(activity); "Network 0.8 MB/s" → seg(activity); "infra healthy · up 12 days 17%" → push(vm:infra); "linux-3 editing src/auth/refresh.ts 70%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini)
- **Enters**: #panel: height 689px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth
- **Leaves**: #panel: height 506px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.activity`

![mac.mini.activity](stories/mac.mini.activity.png) ![mac.mini.activity light](stories/mac.mini.activity.light.png)

- **Capture**: [dark](stories/mac.mini.activity.png), [light](stories/mac.mini.activity.light.png)
- **Shows**: Activity of mac-mini over 1 hr (the remembered window): CPU with the 3 a.m. spike visible on 24 hr; network; processes including "Virtualization · VM linux-3".
- **Data**: as mac.this.activity
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "linux-3 · 4 cores 52%" → push(vm:linux-3); "infra · 8 cores 15%" → push(vm:infra); "review-bot · 4 cores 2%" → push(vm:review-bot)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 700px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.activity.6m`

![mac.mini.activity.6m](stories/mac.mini.activity.6m.png)

- **Capture**: [dark](stories/mac.mini.activity.6m.png)
- **Shows**: Window 6 min: the chart re-grows from .92 with a fade; the axis ends read 6 min ago / now.
- **Data**: 6-minute CPU and network series
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "linux-3 · 4 cores 68%" → push(vm:linux-3); "infra · 8 cores 20%" → push(vm:infra); "review-bot · 4 cores 1%" → push(vm:review-bot)
- **Enters**: svg: opacity+transform 0.3 scaleY(.92) → 1 none 340 ms smooth
- **Leaves**: #panel: height 700px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.activity.24h`

![mac.mini.activity.24h](stories/mac.mini.activity.24h.png)

- **Capture**: [dark](stories/mac.mini.activity.24h.png)
- **Shows**: Window 24 hr: CPU bars per 30 min with the 03:10 peak (94%) marked.
- **Data**: 24-hour CPU series with its peak index
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "linux-3 · 4 cores 50%" → push(vm:linux-3); "infra · 8 cores 15%" → push(vm:infra); "review-bot · 4 cores 3%" → push(vm:review-bot)
- **Enters**: svg: opacity+transform 0.3 scaleY(.92) → 1 none 340 ms smooth
- **Leaves**: #panel: height 700px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.activity.scrub`

![mac.mini.activity.scrub](stories/mac.mini.activity.scrub.png)

- **Capture**: [dark](stories/mac.mini.activity.scrub.png)
- **Shows**: Scrubbing the CPU chart: a cursor line and the value at that time replace the header value ("43% at 14:20").
- **Data**: the series at the scrub index, its timestamp
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "linux-3 · 4 cores 65%" → push(vm:linux-3); "infra · 8 cores 16%" → push(vm:infra); "review-bot · 4 cores 2%" → push(vm:review-bot)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: no recorded animation (inverse gesture)

### `mac.mini.ports`

![mac.mini.ports](stories/mac.mini.ports.png)

- **Capture**: [dark](stories/mac.mini.ports.png)
- **Shows**: Ports of mac-mini and its VMs: Public (443 example.com via Cloudflare Tunnel), Private (6443 Kubernetes API, linux-3:8080 Preview), Listening not shared (5432 Postgres), HTTPS switch.
- **Data**: ports with reach public/tailnet and route; listeners
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "About private links" → info(ports); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Open Preview" → openUrl(http://linux-3.tail3e9a.ts.net:8080); "Copy link to Preview" → copy(http://linux-3.tail3e9a.ts.net:8080); "Show QR for Preview (8080)" → qrpop(p5); "Share Postgres" → sharePort(linux-3:5432); "HTTPS links on mac-mini (switch)" → toggleHttps
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 440px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.ports.https`

![mac.mini.ports.https](stories/mac.mini.ports.https.png)

- **Capture**: [dark](stories/mac.mini.ports.https.png)
- **Shows**: HTTPS links switched on: the switch flips (snappy). The IA rewrites the private links to https:// MagicDNS names (SPEC §A4, Ports); the prototype draws only the switch, so the build owes the rewritten rows in this story's fixture.
- **Data**: per-Mac HTTPS setting
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "About private links" → info(ports); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Open Preview" → openUrl(http://linux-3.tail3e9a.ts.net:8080); "Copy link to Preview" → copy(http://linux-3.tail3e9a.ts.net:8080); "Show QR for Preview (8080)" → qrpop(p5); "Share Postgres" → sharePort(linux-3:5432); "HTTPS links on mac-mini (switch)" → toggleHttps
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 440px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .findcap: CSS background-color 120 ms; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.accounts`

![mac.mini.accounts](stories/mac.mini.accounts.png)

- **Capture**: [dark](stories/mac.mini.accounts.png)
- **Shows**: Accounts of mac-mini: GitHub (ci-bot · repo) and Claude signed in there; then what linux-3 and review-bot are signed in to.
- **Data**: as mac.this.accounts
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Grant a password…" → grant
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 424px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.down`

![mac.mini.down](stories/mac.mini.down.png)

- **Capture**: [dark](stories/mac.mini.down.png)
- **Shows**: mac-mini while infra is down: Needs you card first with the red row "infra is down since HH:MM [Restart]"; infra's VM row red.
- **Data**: needs(mac) with a red
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Restart" → restart(infra); "Review" → push(vm:review-bot); "CPU 46%" → seg(activity); "Memory 19 GB free" → seg(activity); "Network 0.8 MB/s" → seg(activity); "infra down since 14:30 Restart" → push(vm:infra); "linux-3 editing src/auth/refresh.ts 53%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini)
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 538px → 669px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `mac.mini.loading`

![mac.mini.loading](stories/mac.mini.loading.png)

- **Capture**: [dark](stories/mac.mini.loading.png)
- **Shows**: The Mac page while its Mac is still being checked: the prototype renders the last known values with no stale marking (only home marks a Mac "seen 40 s ago · checking…"). The Mac's own page carries no marking: an Unseen Mac keeps no last values.
- **Data**: last-seen timestamp
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → push(vm:review-bot); "CPU 46%" → seg(activity); "Memory 19 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "infra healthy · up 12 days" → push(vm:infra); "linux-3 working" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini)
- **Enters**: #panel: height 689px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth; .spin.sm: CSS spin 800 ms linear; .dot.ring: CSS ping 600 ms linear; .dot.ping: CSS ping 600 ms linear
- **Leaves**: #panel: height 506px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .spin.sm: CSS spin 800 ms linear; .dot.ring: CSS ping 600 ms linear; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.mini.creating`

![mac.mini.creating](stories/mac.mini.creating.png)

- **Capture**: [dark](stories/mac.mini.creating.png)
- **Shows**: A VM being created lands on its host's page: the new row linux-4 "creating · image 2 of 3" with a progress ring and a birth bar, scrolled into view; HUD "Creating linux-4 on mac-mini…".
- **Data**: vm creating: progress and step
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → push(vm:review-bot); "CPU 45%" → seg(activity); "Memory 11 GB free" → seg(activity); "Network 1.1 MB/s" → seg(activity); "infra healthy · up 12 days 15%" → push(vm:infra); "linux-3 editing src/auth/refresh.ts 51%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-4 creating · image 2 of 3" → push(vm:linux-4); "Open linux-4 in Terminal" → attach(linux-4) [Open in Terminal · ⌘↩]; "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini)
- **Enters**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; #panel: height 689px → 544px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); .row.two: CSS land 340 ms linear; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 544px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .row.two: CSS land 340 ms linear; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `mac.mini.created`

![mac.mini.created](stories/mac.mini.created.png)

- **Capture**: [dark](stories/mac.mini.created.png)
- **Shows**: Ready: the row's ring becomes a green check with the success spring (kept 8 s), then "up 1 min"; HUD "linux-4 is ready on mac-mini".
- **Data**: vm running; fresh timestamp
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → push(vm:review-bot); "CPU 44%" → seg(activity); "Memory 11 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "infra healthy · up 12 days 16%" → push(vm:infra); "linux-3 editing src/auth/refresh.ts 54%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-4 ready · up 1 min Open in Terminal" → push(vm:linux-4); "Open in Terminal" → attach(linux-4); "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini)
- **Enters**: .row.two: CSS land 340 ms linear; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 544px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .row.two: CSS freshbg 2600 ms linear; .mk.green: CSS born 460 ms linear; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `mac.air.asleep`

![mac.air.asleep](stories/mac.air.asleep.png) ![mac.air.asleep light](stories/mac.air.asleep.light.png)

- **Capture**: [dark](stories/mac.air.asleep.png), [light](stories/mac.air.asleep.light.png)
- **Shows**: macbook-air asleep: no segments, no New VM; facts "seen 2 hr ago"; empty state "macbook-air is asleep · A laptop that sleeps is not down" with Wake and Tell me when it wakes; footnote on Wake-on-LAN.
- **Data**: mac asleep, last seen
- **Interactions**: "All machines" → back; "More options for macbook-air" → pageMenu(mac:macbook-air); "Wake" → wake(macbook-air); "Tell me when it wakes" → remind(macbook-air)
- **Enters**: #panel: height 689px → 325px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth; path: CSS draw 340 ms linear
- **Leaves**: #panel: height 325px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; path: CSS draw 340 ms linear; .fwrap.morph: CSS morph 340 ms linear (esc)

### `mac.this.empty`

![mac.this.empty](stories/mac.this.empty.png)

- **Capture**: [dark](stories/mac.this.empty.png)
- **Shows**: This Mac with no VM: VMs card is the empty state "No VMs on This Mac yet · N GB free for the first one" with New VM; no New VM header button.
- **Data**: one Mac, zero VMs
- **Interactions**: "All machines" → back; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "CPU 19%" → seg(activity); "Memory 35 GB free" → seg(activity); "Network 1.0 MB/s" → seg(activity); "New VM" → newvm(thismac); "Keep This Mac awake" → awakeToggle; "Upkeep all fine" → unfold(upkeep:thismac)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 421px → 353px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; path: CSS draw 340 ms linear; .fwrap.morph: CSS morph 340 ms linear (esc)

## A workspace

### `workspace.overview`

- **Capture**: none (designed after the prototype; its golden is the reference); the prototype's of `workspace.working` ([dark](stories/workspace.working.png)) shows the same machine
- **Shows**: linux-1's Overview: facts "ephemeral · on This Mac › · up 2 hr"; segments; gauges CPU · 4 cores, Memory, Network; Ports (2 shared: Vite :5173, API :3000; listening); its accounts; footer ssh linux-1 / Open in Terminal.
- **Data**: vm kind, life, uptime; needs(vm); metrics; ports and listeners; accounts lent to it
- **Interactions**: "All machines" → back; "More options for linux-1" → pageMenu(vm:linux-1); "This Mac ›" → push(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]
- **Enters**: a Mac's VM row pushes it; the push of `transition.push` (the page arrives from the trailing side on smooth 340 ms; the panel's height morphs to the page's)
- **Leaves**: the pop of `transition.pop` (esc)

### `workspace.ephemeral`

- **Capture**: none (designed after the prototype; its golden is the reference); the prototype's of `workspace.finished` ([dark](stories/workspace.finished.png)) shows the same machine
- **Shows**: agent-7f, an ephemeral workspace still running: facts "ephemeral · on This Mac › · up 2 hr"; gauges near zero; no port shared and no account lent.
- **Data**: vm kind, life, uptime; metrics; ports and listeners
- **Interactions**: "All machines" → back; "More options for agent-7f" → pageMenu(vm:agent-7f); "ssh agent-7f" → copy(ssh agent-7f); "Open in Terminal" → attach(agent-7f) [⌘↩]
- **Enters**: the push of `transition.push` (the page arrives from the trailing side on smooth 340 ms; the panel's height morphs to the page's)
- **Leaves**: the pop of `transition.pop` (esc)

### `workspace.working.activity`

![workspace.working.activity](stories/workspace.working.activity.png)

- **Capture**: [dark](stories/workspace.working.activity.png)
- **Shows**: Activity of linux-1, then its Upkeep card: "2 updates" (node 24.12 · pnpm 10.5, disk free) with Update.
- **Data**: vm metrics series; upkeep of the guest
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "Update" → updateAll; "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 700px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth (esc)
- **Since the prototype**: the facts line is the workspace's own ("ephemeral · on This Mac › · up 2 hr").

### `workspace.working.ports`

![workspace.working.ports](stories/workspace.working.ports.png)

- **Capture**: [dark](stories/workspace.working.ports.png)
- **Shows**: Ports of linux-1: the two shared links with Copy/QR/Open, the listeners to share, Share a port….
- **Data**: ports and listeners of the VM
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Share Astro" → sharePort(linux-1:4321); "Share Node inspector" → sharePort(linux-1:9229); "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 387px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth (esc)
- **Since the prototype**: facts as workspace.overview.

### `workspace.working.accounts`

![workspace.working.accounts](stories/workspace.working.accounts.png)

- **Capture**: [dark](stories/workspace.working.accounts.png)
- **Shows**: Accounts of linux-1: what it is signed in to, GitHub and Claude granted from This Mac, each with its Remove menu; Grant a password…; the footnote says a grant stays until removed.
- **Data**: sign-ins of the VM; grants made to it
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Grant a password…" → grant(linux-1); "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 346px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth (esc)
- **Since the prototype**: facts as workspace.overview.

### `workspace.waiting.ports`

![workspace.waiting.ports](stories/workspace.waiting.ports.png)

- **Capture**: [dark](stories/workspace.waiting.ports.png)
- **Shows**: Ports of a workspace with nothing shared and nothing listening (review-bot): Private · on your tailnet 0, the one-line empty card "Nothing shared yet"; no Public section (a VM with no public route hides it).
- **Data**: ports and listeners of the VM, both empty
- **Interactions**: "mac-mini" → back; "More options for review-bot" → pageMenu(vm:review-bot); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "About private links" → info(ports); "ssh review-bot" → copy(ssh review-bot); "Open in Terminal" → attach(review-bot) [⌘↩]
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 246px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth (esc)
- **Since the prototype**: review-bot is a running persistent workspace; the segment is unchanged.

### `workspace.stopped`

![workspace.stopped](stories/workspace.stopped.png) ![workspace.stopped light](stories/workspace.stopped.light.png)

- **Capture**: [dark](stories/workspace.stopped.png), [light](stories/workspace.stopped.light.png)
- **Shows**: api, a stopped persistent workspace: no segments, no footer; empty state "api is stopped · Disk kept · starts with 8 GB" with Start.
- **Data**: vm stopped, memory it will take
- **Interactions**: "This Mac" → back; "More options for api" → pageMenu(vm:api); "Start api" → startVm(api)
- **Enters**: path: CSS draw 340 ms linear
- **Leaves**: #panel: height 254px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; path: CSS draw 340 ms linear (esc)

### `workspace.started`

![workspace.started](stories/workspace.started.png)

- **Capture**: [dark](stories/workspace.started.png)
- **Shows**: api started (direct, no confirmation): a plain workspace: segments, gauges, footer ssh api / Open in Terminal; HUD "api is starting".
- **Data**: vm running
- **Interactions**: "This Mac" → back; "More options for api" → pageMenu(vm:api); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "CPU · 4 cores 0%" → seg(activity); "Memory 4.4 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "ssh api" → copy(ssh api); "Open in Terminal" → attach(api) [⌘↩]
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 287px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .gauge.nav: CSS background-color 120 ms; #hud: CSS hud 2600 ms linear (esc)

## A server

### `server.healthy`

![server.healthy](stories/server.healthy.png) ![server.healthy light](stories/server.healthy.light.png)

- **Capture**: [dark](stories/server.healthy.png), [light](stories/server.healthy.light.png)
- **Shows**: infra healthy: rack glyph, More; facts "server · on mac-mini › · up 12 days · restarts after a power cut"; segments Overview/Activity; Health card folded "Healthy · 4 of 4 · checked 40 s ago"; example.com card (latency ms, req/min, p50 and peak with its time, chart); Last 30 days (99.98% available, day cells); gauges; Ports (443 public Cloudflare Tunnel, 6443 private); footnote "Apps, settings and accounts: your own tooling (Ansible)"; footer ssh admin@infra and Open in Terminal.
- **Data**: server state, uptime, restarts, health checks (4 steps, URL, code, checkedAt), latency and rpm series, 30-day availability, ports
- **Interactions**: "All machines" → back; "More options for infra" → pageMenu(server:infra); "mac-mini ›" → push(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Healthy · 4 of 4 checked after every boot · 41 s" → unfold(health:infra); "Restart…" → restartAsk(infra); "Show log" → terminal(journalctl -u k3s on infra); "CPU · 8 cores 15%" → seg(activity); "Memory 11.5 GB free" → seg(activity); "Network 0.9 MB/s" → seg(activity); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "ssh admin@infra" → copy(ssh admin@infra); "Open in Terminal" → terminal(infra)
- **Enters**: #panel: height 689px → 658px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear
- **Leaves**: #panel: height 658px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .gauge.nav: CSS background-color 120 ms; .fwrap.morph: CSS morph 340 ms linear (esc)

### `server.health-unfolded`

![server.health-unfolded](stories/server.health-unfolded.png)

- **Capture**: [dark](stories/server.health-unfolded.png)
- **Shows**: Health unfolded in place: Running, Has an address, SSH answers, Check passes (https://example.com/healthz · 200), then Restart… and Stop….
- **Data**: the four health steps with their result
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Healthy · 4 of 4 checked after every boot · 40 s" → unfold(health:infra); "Restart…" → restartAsk(infra); "Show log" → terminal(journalctl -u k3s on infra); "CPU · 8 cores 19%" → seg(activity); "Memory 11.0 GB free" → seg(activity); "Network 0.9 MB/s" → seg(activity); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "ssh admin@infra" → copy(ssh admin@infra); "Open in Terminal" → terminal(infra)
- **Enters**: .row.two: CSS background-color 120 ms; svg: CSS transform 340 ms cubic-bezier(0.32, 0.72, 0, 1); .fold.open: CSS grid-template-rows 340 ms cubic-bezier(0.32, 0.72, 0, 1); .fin: CSS opacity 120 ms; .fin: CSS transform 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: #panel: height 700px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .row.two: CSS background-color 120 ms (esc)

### `server.activity`

![server.activity](stories/server.activity.png)

- **Capture**: [dark](stories/server.activity.png)
- **Shows**: Activity of infra over the window.
- **Data**: server metrics series
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "ssh admin@infra" → copy(ssh admin@infra); "Open in Terminal" → terminal(infra)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 700px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth (esc)

### `server.down`

![server.down](stories/server.down.png) ![server.down light](stories/server.down.light.png)

- **Capture**: [dark](stories/server.down.png), [light](stories/server.down.light.png)
- **Shows**: infra down: red banner "Down since HH:MM · out of memory" with Restart; "Last lines before it went down" log card (etcd slow fdatasync, Out of memory: killed k3s-server, …); 30 days at 99.91%; stale gauges "CPU · last", "Memory · last", "Last seen"; Ports "unreachable while down" greyed; footer note "Needs infra running" with a disabled Terminal button.
- **Data**: down time, cause, last log lines, last-known metrics
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Restart" → restart(infra); "Show full log" → terminal(journalctl -b -1 on infra); "Open in Terminal" → terminal(infra) (disabled)
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 666px → 538px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .dot.red: CSS ping 600 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `server.restarting`

![server.restarting](stories/server.restarting.png)

- **Capture**: [dark](stories/server.restarting.png)
- **Shows**: Restarting: health unfolds by itself and each step waits in turn (spinner on the current one); gauges become skeletons; ports "unreachable until ready"; footer "Opens once SSH answers".
- **Data**: restart step index, restart start time
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Restarting · step 2 of 4 started 1 s ago" → unfold(health:infra); "Open in Terminal" → terminal(infra) (disabled)
- **Enters**: svg: CSS color 120 ms; .sitext: CSS max-width 340 ms cubic-bezier(0.32, 0.72, 0, 1); .sitext: CSS opacity 120 ms; .busy: CSS spin 800 ms linear; .spin: CSS spin 800 ms linear; .gauge.skel: CSS shim 1200 ms linear; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 637px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .busy: CSS spin 800 ms linear; .spin: CSS spin 800 ms linear; .gauge.skel: CSS shim 1200 ms linear (esc)

### `server.restarting.late`

![server.restarting.late](stories/server.restarting.late.png)

- **Capture**: [dark](stories/server.restarting.late.png)
- **Shows**: Restart at step 4 of 4 (Check passes pending).
- **Data**: as server.restarting
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Restarting · step 4 of 4 started 3 s ago" → unfold(health:infra); "Open in Terminal" → terminal(infra) (disabled)
- **Enters**: svg: CSS color 120 ms; .sitext: CSS max-width 340 ms cubic-bezier(0.32, 0.72, 0, 1); .sitext: CSS opacity 120 ms; .busy: CSS spin 800 ms linear; .spin: CSS spin 800 ms linear; .gauge.skel: CSS shim 1200 ms linear; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 637px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .busy: CSS spin 800 ms linear; .spin: CSS spin 800 ms linear; .gauge.skel: CSS shim 1200 ms linear (esc)

### `server.recovered`

![server.recovered](stories/server.recovered.png)

- **Capture**: [dark](stories/server.recovered.png)
- **Shows**: Healthy again: the page returns to its healthy layout, up 1 min, restarts 1, 30 days 99.91%; HUD "infra is healthy again · down 1 min".
- **Data**: wasDown info
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Healthy · 4 of 4 checked after every boot · 1 s " → unfold(health:infra); "Restart…" → restartAsk(infra); "Show log" → terminal(journalctl -u k3s on infra); "CPU · 8 cores 40%" → seg(activity); "Memory 8.5 GB free" → seg(activity); "Network 1.2 MB/s" → seg(activity); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "ssh admin@infra" → copy(ssh admin@infra); "Open in Terminal" → terminal(infra)
- **Enters**: .sitext: CSS max-width 340 ms cubic-bezier(0.32, 0.72, 0, 1); .busy: CSS spin 800 ms linear; .spin: CSS spin 800 ms linear; .gauge.skel: CSS shim 1200 ms linear; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 700px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; #hud: CSS hud 2600 ms linear (esc)

### `server.stopped`

![server.stopped](stories/server.stopped.png)

- **Capture**: [dark](stories/server.stopped.png)
- **Shows**: infra stopped by its owner (after the typed confirmation).
- **Data**: server stopped
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Healthy · 4 of 4 checked after every boot · 41 s" → unfold(health:infra); "Restart…" → restartAsk(infra); "Show log" → terminal(journalctl -u k3s on infra); "CPU · 8 cores 17%" → seg(activity); "Memory 11.1 GB free" → seg(activity); "Network 0.0 MB/s" → seg(activity); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 612px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .dot.sq: CSS ping 600 ms linear (esc)

## Ports (fleet and scoped)

### `ports.all`

![ports.all](stories/ports.all.png) ![ports.all light](stories/ports.all.light.png)

- **Capture**: [dark](stories/ports.all.png), [light](stories/ports.all.light.png)
- **Shows**: Ports · All machines: Public (443 example.com · Cloudflare Tunnel), Private on your tailnet grouped by machine (linux-1 5173, 3000; infra 6443; linux-3 8080) with an info button; Listening, not shared; footer "Full list in Terminal ↗" and Share a port….
- **Data**: every port with reach, route, host; listeners
- **Interactions**: "All machines" → back; "More options for Ports" → pageMenu(ports:all); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Open Preview" → openUrl(http://linux-3.tail3e9a.ts.net:8080); "Copy link to Preview" → copy(http://linux-3.tail3e9a.ts.net:8080); "Show QR for Preview (8080)" → qrpop(p5); "Share Astro" → sharePort(linux-1:4321); "Share Node inspector" → sharePort(linux-1:9229); "Share Postgres" → sharePort(linux-3:5432); "Full list in Terminal" → terminal(station ports --all-machines); "Share a port…" → sharePortSheet
- **Enters**: #panel: height 689px → 556px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear
- **Leaves**: #panel: height 556px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `ports.all.exposed`

![ports.all.exposed](stories/ports.all.exposed.png) ![ports.all.exposed light](stories/ports.all.exposed.light.png)

- **Capture**: [dark](stories/ports.all.exposed.png), [light](stories/ports.all.exposed.light.png)
- **Shows**: Exposed first: a red card "5000 · AirPlay Receiver (ControlCenter) · reachable outside Tailscale" with Declare public…, Show the process, Review in Terminal ↗.
- **Data**: exposed listener with its process
- **Interactions**: "All machines" → back; "More options for Ports" → pageMenu(ports:all); "Declare public…" → declarePublic; "Show process" → showProc; "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Open Preview" → openUrl(http://linux-3.tail3e9a.ts.net:8080); "Copy link to Preview" → copy(http://linux-3.tail3e9a.ts.net:8080); "Show QR for Preview (8080)" → qrpop(p5); "Share Astro" → sharePort(linux-1:4321); "Share Node inspector" → sharePort(linux-1:9229); "Share Postgres" → sharePort(linux-3:5432); "Full list in Terminal" → terminal(station ports --all-machines); "Share a port…" → sharePortSheet
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 662px → 669px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `ports.all.down`

![ports.all.down](stories/ports.all.down.png)

- **Capture**: [dark](stories/ports.all.down.png)
- **Shows**: Ports while infra is down: its two ports greyed "unreachable".
- **Data**: port reachability follows its VM state
- **Interactions**: "All machines" → back; "More options for Ports" → pageMenu(ports:all); "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Open Preview" → openUrl(http://linux-3.tail3e9a.ts.net:8080); "Copy link to Preview" → copy(http://linux-3.tail3e9a.ts.net:8080); "Show QR for Preview (8080)" → qrpop(p5); "Share Astro" → sharePort(linux-1:4321); "Share Node inspector" → sharePort(linux-1:9229); "Share Postgres" → sharePort(linux-3:5432); "Full list in Terminal" → terminal(station ports --all-machines); "Share a port…" → sharePortSheet
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 556px → 669px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .dot.red: CSS ping 600 ms linear; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `ports.shared-new`

![ports.shared-new](stories/ports.shared-new.png)

- **Capture**: [dark](stories/ports.shared-new.png)
- **Shows**: A listener shared in one tap: the new link row arrives at the top of the shared list (entering highlight) and its QR pops open; the Ports tile shows "+1".
- **Data**: new port id; listener removed
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Open Astro" → openUrl(http://linux-1.tail3e9a.ts.net:4321); "Copy link to Astro" → copy(http://linux-1.tail3e9a.ts.net:4321); "Show QR for Astro (4321)" → qrpop(p1794058377718); "Share Node inspector" → sharePort(linux-1:9229); "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]; "Copy" → copy(http://linux-1.tail3e9a.ts.net:4321); "Open" → openUrl(http://linux-1.tail3e9a.ts.net:4321); "Stop sharing…" → stopShare(p1794058377718)
- **Enters**: #pop: opacity+transform 0 scale(.94) → 1 none 200 ms snappy; .row.two: CSS land 340 ms linear; #hud: CSS hud 2600 ms linear
- **Leaves**: #pop: opacity+transform 1 → 0 scale(.97) 200 ms snappy; .round.sm: CSS opacity 120 ms; .row.two: CSS land 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `ports.audiences`

- **Capture**: none (the gap state ADR-035 rules and ADR-039 draws: the prototype shows every share as private)
- **Shows**: linux-1 shares one port with each audience: 3001 in Public (a Funnel port), 4000 in This Mac only (its loopback), the rest in Private; 8443 listens outside its rights and reads "listens, not allowed" under Listening, not shared, with Share.
- **Data**: a share's audience (`to`), and a listener's `not-allowed` state with the rights it is outside
- **Interactions**: as `ports.all`; "Share Node inspector" → sharePort(linux-1:8443)
- **Enters**: no animation (a state reached in place)
- **Leaves**: the row leaves in place once the rights allow it or it stops listening

## Accounts

### `accounts.all`

![accounts.all](stories/accounts.all.png) ![accounts.all light](stories/accounts.all.light.png)

- **Capture**: [dark](stories/accounts.all.png), [light](stories/accounts.all.light.png)
- **Shows**: Accounts · All machines: Needs sign-in (npm on This Mac); then one card per machine, On This Mac, On linux-1, On mac-mini, On linux-3, On review-bot, each with the accounts it is signed in to or granted (a grant to a VM of This Mac with its Remove menu); the Macs not read; footer Full list in Terminal and Grant a password….
- **Data**: sign-ins per Mac and per VM; grants made from This Mac
- **Interactions**: "All machines" → back; "Sign in" → signin(thismac:npm); "Full list in Terminal" → terminal(station accounts --all-machines); "Grant a password…" → grant
- **Enters**: #panel: height 689px → 658px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear
- **Leaves**: #panel: height 658px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `accounts.signing-in`

![accounts.signing-in](stories/accounts.signing-in.png)

- **Capture**: [dark](stories/accounts.signing-in.png)
- **Shows**: Sign in pressed: the button reads "Waiting…" (busy) while Terminal runs the interactive login; HUD "Terminal · npm login on This Mac".
- **Data**: pending sign-in
- **Interactions**: "All machines" → back; "Waiting…" → signin(thismac:npm); "Full list in Terminal" → terminal(station accounts --all-machines); "Grant a password…" → grant
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 658px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `accounts.signed-in`

![accounts.signed-in](stories/accounts.signed-in.png)

- **Capture**: [dark](stories/accounts.signed-in.png)
- **Shows**: Signed in: the need collapses (height to zero) and npm joins the signed-in list "you · publish".
- **Data**: account becomes signed in with its scope
- **Interactions**: "All machines" → back; "Full list in Terminal" → terminal(station accounts --all-machines); "Grant a password…" → grant
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 664px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `awake.all`

![awake.all](stories/awake.all.png) ![awake.all light](stories/awake.all.light.png)

- **Capture**: [dark](stories/awake.all.png), [light](stories/awake.all.light.png)
- **Shows**: Awake · All Macs: This Mac (Keep This Mac awake switch, segments 1 hr / 4 hr / Until 18:00, With the lid closed, On battery), under its card the note "linux-1 pauses while This Mac sleeps"; mac-mini "Never sleeps · hosts infra" with Change…; macbook-air "May sleep · asleep · nothing depends on it".
- **Data**: awake state of This Mac (on, mode, until, lid, battery); each Mac's sleep policy and dependants
- **Interactions**: "All machines" → back; "Keep This Mac awake (switch)" → awakeToggle; "1 hr" → awakeFor(1h); "4 hr" → awakeFor(4h); "Until 18:00" → awakeFor(until); "With the lid closed (switch)" → awakeOpt(lid); "On battery (switch)" → awakeOpt(battery); "Keep mac-mini awake" → awakeChange(mac-mini)
- **Enters**: #panel: height 689px → 447px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth
- **Leaves**: #panel: height 447px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear (esc)

### `awake.off`

![awake.off](stories/awake.off.png)

- **Capture**: [dark](stories/awake.off.png)
- **Shows**: This Mac may sleep: switch off, moon glyph, no duration selected.
- **Data**: awake.on false
- **Interactions**: "All machines" → back; "Keep This Mac awake (switch)" → awakeToggle; "1 hr" → awakeFor(1h); "4 hr" → awakeFor(4h); "Until 18:00" → awakeFor(until); "With the lid closed (switch)" → awakeOpt(lid); "On battery (switch)" → awakeOpt(battery); "Keep mac-mini awake" → awakeChange(mac-mini)
- **Enters**: #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 447px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `awake.4h`

![awake.4h](stories/awake.4h.png)

- **Capture**: [dark](stories/awake.4h.png)
- **Shows**: Awake for 4 hr: the segment slides to 4 hr; the tile and the menu bar say the new end.
- **Data**: awake mode 4h
- **Interactions**: "All machines" → back; "Keep This Mac awake (switch)" → awakeToggle; "1 hr" → awakeFor(1h); "4 hr" → awakeFor(4h); "Until 18:00" → awakeFor(until); "With the lid closed (switch)" → awakeOpt(lid); "On battery (switch)" → awakeOpt(battery); "Keep mac-mini awake" → awakeChange(mac-mini)
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 447px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .place.nav: CSS background-color 120 ms; .fwrap.morph: CSS morph 340 ms linear (esc)

## Sheets

Keys: ↩ the primary button, ⌥↩ / ⌘↩ the split variants (New VM), Tab cycles inside the sheet only, esc cancels.

### `sheet.new-vm`

![sheet.new-vm](stories/sheet.new-vm.png) ![sheet.new-vm light](stories/sheet.new-vm.light.png)

- **Capture**: [dark](stories/sheet.new-vm.png), [light](stories/sheet.new-vm.light.png)
- **Shows**: New VM, unrolled from the panel top over a scrim (the page behind dims). Opened from home's +, it picks the roomiest Mac: subtitle "On This Mac, the Mac with the most free memory. Ready in about a minute."; Task field focused ("What Claude should do · optional", caption "Empty: a workspace of your own"); On popup (This Mac · 23 GB free); Options disclosure "Ephemeral · Linux · 8 GB" with the caption "linux-4"; footer Cancel and the split button Create / ▾.
- **Data**: Macs with free memory and sleep state; next free name; last kind used; host accounts
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "Task for Claude"; "This Mac 23 GB free" → nvOn; "Ephemeral · Linux · 8 GB" → nvOpts; "Cancel" → closeSheet; "Create" → createVm [↩]; "More ways to create" → nvMore
- **Enters**: .sheet: clipPath+transform+opacity inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 1 → inset(0 0 0 0 round 14px) none 1 340 ms smooth +40 ms; .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)
- **Since the prototype**: no Task field: a VM is made with nothing started in it. The split button is Create / Create and Open in Terminal.

### `sheet.new-vm.options`

![sheet.new-vm.options](stories/sheet.new-vm.options.png)

- **Capture**: [dark](stories/sheet.new-vm.options.png)
- **Shows**: Options open on mac-mini: Kind (Workspace | Server), Lifetime (Ephemeral | Persistent), System (Linux | macOS), Memory slider 4/8/16/32 GB on a log scale ("Over 19 GB would squeeze mac-mini", hatched past free memory), Name (linux-4, "Generated, editable").
- **Data**: host free memory
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "Task for Claude"; "mac-mini 19 GB free" → nvOn; "Ephemeral · Linux · 8 GB" → nvOpts; "Workspace" → nv(kind:ws); "Server" → nv(kind:server); "Ephemeral" → nv(life:ephemeral); "Persistent" → nv(life:persistent); "Linux" → nv(sys:linux); "macOS" → nv(sys:macos); "4 GB" → nv(mem:4); "8 GB" → nv(mem:8); "16 GB" → nv(mem:16); "32 GB" → nv(mem:32) (disabled); field "Name"; "Cancel" → closeSheet; "Create" → createVm [↩]; "More ways to create" → nvMore
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)
- **Since the prototype**: no Task field: a VM is made with nothing started in it. The split button is Create / Create and Open in Terminal.

### `sheet.new-vm.server`

![sheet.new-vm.server](stories/sheet.new-vm.server.png)

- **Capture**: [dark](stories/sheet.new-vm.server.png)
- **Shows**: Kind Server on This Mac: no Task, no Lifetime; Name first-class ("A server needs a name you will recognise"); orange warning "This Mac may sleep: a server pauses with it."
- **Data**: host sleep policy
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "This Mac 23 GB free" → nvOn; field "Name"; "Server · Linux · 8 GB" → nvOpts; "Workspace" → nv(kind:ws); "Server" → nv(kind:server); "Linux" → nv(sys:linux); "macOS" → nv(sys:macos); "4 GB" → nv(mem:4); "8 GB" → nv(mem:8); "16 GB" → nv(mem:16); "32 GB" → nv(mem:32) (disabled); "Cancel" → closeSheet; "Create" → createVm [↩]; "More ways to create" → nvMore
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)
- **Since the prototype**: unchanged but for the Task row, gone from every kind.

### `sheet.new-vm.first`

![sheet.new-vm.first](stories/sheet.new-vm.first.png)

- **Capture**: [dark](stories/sheet.new-vm.first.png)
- **Shows**: New VM from the empty fleet: the same sheet, on This Mac.
- **Data**: one Mac
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "Task for Claude"; "This Mac 35 GB free" → nvOn; "Ephemeral · Linux · 8 GB" → nvOpts; "Cancel" → closeSheet; "Create" → createVm [↩]; "More ways to create" → nvMore
- **Enters**: .sheet: clipPath+transform+opacity inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 1 → inset(0 0 0 0 round 14px) none 1 340 ms smooth +40 ms; .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); path: CSS draw 340 ms linear
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); path: CSS draw 340 ms linear (esc)
- **Since the prototype**: no Task field: a VM is made with nothing started in it. The split button is Create / Create and Open in Terminal.

### `sheet.grant`

![sheet.grant](stories/sheet.grant.png) ![sheet.grant light](stories/sheet.grant.light.png)

- **Capture**: [dark](stories/sheet.grant.png), [light](stories/sheet.grant.light.png)
- **Shows**: Grant a password (Accounts footer): "From the passwords of This Mac. It stays until you remove it. Touch ID will ask you to confirm before the password leaves this Mac."; To popup (linux-1, the workspaces of This Mac); Password list of the vault (GitHub token and Claude token ticked or dimmed with "Granted already; granting again replaces it"; Cloudflare token disabled, "Signed out here"); Cancel and "Grant GitHub token".
- **Data**: workspaces of This Mac; the passwords of This Mac's vault and the grant each fits
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "linux-1" → shTo; "GitHub token" → sh(entry:pw-github); "Claude token" → sh(entry:pw-claude); "Cloudflare token" → sh(entry:pw-cloudflare) (disabled); "Cancel" → closeSheet; "Grant GitHub token" → doGrant
- **Enters**: .sheet: clipPath+transform+opacity inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 1 → inset(0 0 0 0 round 14px) none 1 340 ms smooth +40 ms; .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `sheet.grant.vm`

![sheet.grant.vm](stories/sheet.grant.vm.png)

- **Capture**: [dark](stories/sheet.grant.vm.png)
- **Shows**: Grant a password pre-filled with linux-1, from its Accounts page: To linux-1 said, not asked; the passwords of This Mac.
- **Data**: as sheet.grant
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "GitHub token" → sh(entry:pw-github); "Claude token" → sh(entry:pw-claude); "Cancel" → closeSheet; "Grant GitHub token" → doGrant
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `sheet.share-port`

![sheet.share-port](stories/sheet.share-port.png)

- **Capture**: [dark](stories/sheet.share-port.png)
- **Shows**: Share a port: "Private on your tailnet. Pick a listener, no typing."; Of popup (linux-1 on This Mac); "Listening now in linux-1": Astro linux-1:4321 and Node inspector linux-1:9229, each with Share; Cancel.
- **Data**: running workspaces; listeners per VM
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "linux-1 on This Mac" → spOf; "Share" → sharePort(linux-1:4321); "Share" → sharePort(linux-1:9229); "Cancel" → closeSheet
- **Enters**: .sheet: clipPath+transform+opacity inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 1 → inset(0 0 0 0 round 14px) none 1 340 ms smooth +40 ms; .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `sheet.share-port.vm`

![sheet.share-port.vm](stories/sheet.share-port.vm.png)

- **Capture**: [dark](stories/sheet.share-port.vm.png)
- **Shows**: Share a port pre-filled with linux-3: Postgres linux-3:5432 [Share].
- **Data**: as sheet.share-port
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Share" → sharePort(linux-3:5432); "Cancel" → closeSheet
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `sheet.share-port.empty`

![sheet.share-port.empty](stories/sheet.share-port.empty.png)

- **Capture**: [dark](stories/sheet.share-port.empty.png)
- **Shows**: Share a port of a VM where nothing listens (review-bot): Of review-bot, the label "Listening now in review-bot", then the plain line "Nothing listens in review-bot right now." in place of the listener list; no Share button.
- **Data**: listeners of the VM, empty
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `sheet.cleanup.mac`

![sheet.cleanup.mac](stories/sheet.cleanup.mac.png) ![sheet.cleanup.mac light](stories/sheet.cleanup.mac.light.png)

- **Capture**: [dark](stories/sheet.cleanup.mac.png), [light](stories/sheet.cleanup.mac.light.png)
- **Shows**: Clean up on This Mac: "Stopped ephemeral workspaces. Each goes with its disk, its ports and the accounts lent to it."; agent-7f pre-checked (stopped · 42.9 GB disk); "linux-1 is still running, so it stays."; Cancel and the red "Delete 1 ephemeral workspace".
- **Data**: ephemeral workspaces of the Mac: done/stopped, idle time, ports, shares
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "agent-7f stopped · 42.9 GB disk" → cu(agent-7f); "Cancel" → closeSheet; "Delete 1 ephemeral workspace" → doCleanup
- **Enters**: .sheet: clipPath+transform+opacity inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 1 → inset(0 0 0 0 round 14px) none 1 340 ms smooth +40 ms; .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)
- **Since the prototype**: the world stops agent-7f, since Clean up offers only stopped ephemeral workspaces; the button is worded from what is checked.

### `sheet.cleanup.all`

![sheet.cleanup.all](stories/sheet.cleanup.all.png)

- **Capture**: [dark](stories/sheet.cleanup.all.png)
- **Shows**: Clean up across every Mac: agent-7f (This Mac) and linux-3 (mac-mini) pre-checked, each naming its Mac; "linux-1 is still running, so it stays."
- **Data**: as sheet.cleanup.mac, unscoped
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "agent-7f This Mac · stopped" → cu(agent-7f); "linux-3 mac-mini · stopped" → cu(linux-3); "Cancel" → closeSheet; "Delete 2 ephemeral workspaces" → doCleanup
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1); .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)
- **Since the prototype**: as sheet.cleanup.mac, unscoped.

### `sheet.hint.add-mac`

![sheet.hint.add-mac](stories/sheet.hint.add-mac.png)

- **Capture**: [dark](stories/sheet.hint.add-mac.png)
- **Shows**: Add a Mac hint: "On the new Mac, run this once…" with the join command and Copy.
- **Data**: tailnet name
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Copy" → copy(curl -fsSL https://station.sh/join | sh -s -- --tailnet tail3e9a); "Done" → closeSheet
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1); .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `sheet.hint.firewall`

![sheet.hint.firewall](stories/sheet.hint.firewall.png)

- **Capture**: [dark](stories/sheet.hint.firewall.png)
- **Shows**: Turn the firewall on: Machines never changes a security setting; the socketfilterfw command to copy.
- **Data**: none
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Copy" → copy(sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on); "Done" → closeSheet
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `sheet.hint.declare-public`

![sheet.hint.declare-public](stories/sheet.hint.declare-public.png)

- **Capture**: [dark](stories/sheet.hint.declare-public.png)
- **Shows**: Declare a port public: what declaring means, and the command.
- **Data**: the exposed port
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Copy" → copy(station ports declare-public 5000 --on this-mac); "Done" → closeSheet
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .sheet: clipPath+transform+opacity inset(0 0 0 0 round 14px) none 1 → inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 0 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

## Confirmations (alerts)

Keys: Tab inside the alert only, ↩ the default (the red button is never the default while typed confirmation is empty), esc cancels.

### `alert.restart-server`

![alert.restart-server](stories/alert.restart-server.png) ![alert.restart-server light](stories/alert.restart-server.light.png)

- **Capture**: [dark](stories/alert.restart-server.png), [light](stories/alert.restart-server.light.png)
- **Shows**: Restart infra on mac-mini? (its name typed, as `machines.restart` asks of a server, `--confirm=infra` in the CLI) "infra is unreachable for about a minute…", the name field, Cancel / Restart infra in red, disabled until infra is typed. The alert scales in from 1.08 at the panel centre over a scrim.
- **Data**: server, host
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "infra"; "Cancel" → closeSheet; "Restart infra" → sheetYes (disabled)
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.stop-server`

![alert.stop-server](stories/alert.stop-server.png)

- **Capture**: [dark](stories/alert.stop-server.png)
- **Shows**: Stop infra (typed): "example.com goes offline"; a field asks to type infra; the red button is disabled until it matches.
- **Data**: server, its public host
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "infra"; "Cancel" → closeSheet; "Stop infra" → sheetYes (disabled)
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.stop-server.typed`

![alert.stop-server.typed](stories/alert.stop-server.typed.png)

- **Capture**: [dark](stories/alert.stop-server.typed.png)
- **Shows**: The name typed: Stop infra is enabled.
- **Data**: as alert.stop-server
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "infra"; "Cancel" → closeSheet; "Stop infra" → sheetYes
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.delete-server`

![alert.delete-server](stories/alert.delete-server.png)

- **Capture**: [dark](stories/alert.delete-server.png)
- **Shows**: Delete infra? (typed) "It goes with its disk (61 GB), its address and 2 ports, and example.com stops answering."
- **Data**: disk size, port count, public host
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "infra"; "Cancel" → closeSheet; "Delete infra" → sheetYes (disabled)
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.stop-vm`

![alert.stop-vm](stories/alert.stop-vm.png)

- **Capture**: [dark](stories/alert.stop-vm.png)
- **Shows**: Stop linux-1 on This Mac? "Its disk is kept. 2 shared ports stop answering until it starts again."
- **Data**: ports of the VM
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Stop linux-1" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.delete-vm.ephemeral`

![alert.delete-vm.ephemeral](stories/alert.delete-vm.ephemeral.png) ![alert.delete-vm.ephemeral light](stories/alert.delete-vm.ephemeral.light.png)

- **Capture**: [dark](stories/alert.delete-vm.ephemeral.png), [light](stories/alert.delete-vm.ephemeral.light.png)
- **Shows**: Delete agent-7f? (y/n for an ephemeral workspace) "This can't be undone. It goes with its disk (12 GB)."
- **Data**: disk, ports and shares of the VM
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Delete agent-7f" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.delete-vm.persistent`

![alert.delete-vm.persistent](stories/alert.delete-vm.persistent.png)

- **Capture**: [dark](stories/alert.delete-vm.persistent.png)
- **Shows**: Delete api? (typed for a persistent workspace).
- **Data**: as alert.delete-vm.ephemeral
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "api"; "Cancel" → closeSheet; "Delete api" → sheetYes (disabled)
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); path: CSS draw 340 ms linear
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); path: CSS draw 340 ms linear (esc)

### `alert.stop-all`

![alert.stop-all](stories/alert.stop-all.png)

- **Capture**: [dark](stories/alert.stop-all.png)
- **Shows**: Stop every workspace on This Mac? "linux-1, agent-7f stop and keep their disk. Awake ends."
- **Data**: running workspaces of the Mac; servers kept
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Stop 2 workspaces" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.stop-all.server`

![alert.stop-all.server](stories/alert.stop-all.server.png)

- **Capture**: [dark](stories/alert.stop-all.server.png)
- **Shows**: Stop every workspace on mac-mini? "linux-3, review-bot stop and keep their disk. Servers keep running." (a Mac hosting a server says so; only This Mac adds "Awake ends."); red "Stop 2 workspaces".
- **Data**: running workspaces of the Mac; whether it hosts a server; this-Mac flag
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Stop 2 workspaces" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.restart-mac`

![alert.restart-mac](stories/alert.restart-mac.png)

- **Capture**: [dark](stories/alert.restart-mac.png)
- **Shows**: Restart mac-mini? (typed, it hosts a server) "infra is down while mac-mini restarts, about 2 min."
- **Data**: servers the Mac hosts
- **Interactions**: "the scrim (click outside the card)" → closeSheet; field "mac-mini"; "Cancel" → closeSheet; "Restart mac-mini" → sheetYes (disabled)
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.forget-mac`

![alert.forget-mac](stories/alert.forget-mac.png)

- **Capture**: [dark](stories/alert.forget-mac.png)
- **Shows**: Forget macbook-air? "Its VMs keep running; Machines stops showing it. You can add it again."
- **Data**: mac
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Forget macbook-air" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); path: CSS draw 340 ms linear
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); path: CSS draw 340 ms linear (esc)

### `alert.remove-grant`

![alert.remove-grant](stories/alert.remove-grant.png)

- **Capture**: [dark](stories/alert.remove-grant.png)
- **Shows**: Remove GitHub? "The grant ends here; the token stays yours." Remove is red, as the grant row's Remove is.
- **Data**: grant
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Remove" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.stop-share`

![alert.stop-share](stories/alert.stop-share.png)

- **Capture**: [dark](stories/alert.stop-share.png)
- **Shows**: Stop sharing linux-1:5173? "The link stops answering on your tailnet. The service keeps running inside the VM."
- **Data**: port
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Stop sharing" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `alert.let-sleep`

![alert.let-sleep](stories/alert.let-sleep.png)

- **Capture**: [dark](stories/alert.let-sleep.png)
- **Shows**: Let mac-mini sleep? "infra pauses while mac-mini sleeps: example.com stops answering."
- **Data**: servers on the Mac, their public host
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Let it sleep" → sheetYes
- **Enters**: .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)
- **Leaves**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

## Menus

Keys: ↑ ↓ move, → opens a submenu, ↩ runs, esc closes.

### `menu.station`

![menu.station](stories/menu.station.png) ![menu.station light](stories/menu.station.light.png)

- **Capture**: [dark](stories/menu.station.png), [light](stories/menu.station.light.png)
- **Shows**: The Spwn Menu menu (the … button): New VM… ⌘N, Add a Mac…, Open station in Terminal ⌘T, ✓ Open at login, Quit Spwn Menu ⌘Q.
- **Data**: roomiest Mac; settings
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 40%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 74%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "New VM… ⌘N"; menu item "Add a Mac…"; menu item "Open station in Terminal ⌘T"; menu item "Open at login"; … 1 more
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .round: CSS background-color 120 ms
- **Leaves**: no recorded animation (esc)

### `menu.context.workspace`

![menu.context.workspace](stories/menu.context.workspace.png) ![menu.context.workspace light](stories/menu.context.workspace.light.png)

- **Capture**: [dark](stories/menu.context.workspace.png), [light](stories/menu.context.workspace.light.png)
- **Shows**: Right-click a workspace row: header "linux-1 on This Mac", Open ↩, Open in Terminal ⌘↩, Open Vite, Copy ›, Share ›, Restart, Stop…, Delete… (red).
- **Data**: vm state, its first web port
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 32%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 60%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "Open ↩"; menu item "Open in Terminal ⌘↩"; menu item "Open Vite"; menu item "Copy"; menu item "Share"; … 3 more
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .qa: CSS opacity 120 ms
- **Leaves**: .row.two: CSS background-color 120 ms (esc)

### `menu.context.workspace.copy`

![menu.context.workspace.copy](stories/menu.context.workspace.copy.png)

- **Capture**: [dark](stories/menu.context.workspace.copy.png)
- **Shows**: The Copy submenu: ssh command, Link to Vite; This Mac's page drawn with Upkeep unfolded, so the menu stays a share of the glass.
- **Data**: as menu.context.workspace
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 29%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 60%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "Open ↩"; menu item "Open in Terminal ⌘↩"; menu item "Open Vite"; menu item "Copy"; menu item "Share"; … 5 more
- **Enters**: .menu.child: opacity 0 → 1 120 ms ease-out
- **Leaves**: .row.two: CSS background-color 120 ms; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `menu.context.workspace.share`

![menu.context.workspace.share](stories/menu.context.workspace.share.png)

- **Capture**: [dark](stories/menu.context.workspace.share.png)
- **Shows**: The Share submenu: Show QR…, A port…; This Mac's page drawn with Upkeep unfolded, so the menu stays a share of the glass.
- **Data**: as menu.context.workspace
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 31%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 61%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "Open ↩"; menu item "Open in Terminal ⌘↩"; menu item "Open Vite"; menu item "Copy"; menu item "Share"; … 6 more
- **Enters**: .menu.child: opacity 0 → 1 120 ms ease-out
- **Leaves**: .row.two: CSS background-color 120 ms; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `menu.context.stopped`

![menu.context.stopped](stories/menu.context.stopped.png)

- **Capture**: [dark](stories/menu.context.stopped.png)
- **Shows**: A stopped workspace's menu offers Start instead of Restart/Stop.
- **Data**: vm stopped
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "CPU 18%" → seg(activity); "Memory 23 GB free" → seg(activity); "Network 1.1 MB/s" → seg(activity); "linux-1 running tests · 41/58 pass 36%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "agent-7f finished · tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "api stopped · persistent" → push(vm:api); "Keep This Mac awake" → awakeToggle; "Upkeep firewall off · 3 more" → unfold(upkeep:thismac); "How to" → howto; "Copy fix" → copyCmd(launchctl bootout system/com.docker.vmnetd); "Sign in" → signin(thismac:npm); "Update" → updateAll; menu item "Open ↩"; menu item "Open in Terminal ⌘↩"; menu item "Copy"; menu item "Share"; menu item "Start"; menu item "Delete…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: .row.two: CSS background-color 120 ms (esc)

### `menu.context.server`

![menu.context.server](stories/menu.context.server.png)

- **Capture**: [dark](stories/menu.context.server.png)
- **Shows**: A server row: Open, Open Traefik, Copy, Share, Restart (asks, red as its typed question is), Stop…, Delete….
- **Data**: server state, ports
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 38%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 52%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "Open ↩"; menu item "Open Traefik"; menu item "Copy"; menu item "Share"; menu item "Restart"; … 2 more
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: .row.two: CSS background-color 120 ms (esc)

### `menu.context.server.down`

![menu.context.server.down](stories/menu.context.server.down.png)

- **Capture**: [dark](stories/menu.context.server.down.png)
- **Shows**: A down server's menu: Restart is direct.
- **Data**: server down
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Restart" → restart(infra); "2 more need you · review-bot, linux-1" → allNeeds; "Ports, 2 unreachable · 5 shared" → push(ports:all) [2 unreachable · 5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 30%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 68%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra down since 14:35" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 down2 agents 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "Open ↩"; menu item "Open Traefik"; menu item "Copy"; menu item "Share"; menu item "Restart"; … 1 more
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1); #hud: CSS hud 2600 ms linear
- **Leaves**: .row.two: CSS background-color 120 ms; #hud: CSS hud 2600 ms linear (esc)

### `menu.context.mac`

![menu.context.mac](stories/menu.context.mac.png)

- **Capture**: [dark](stories/menu.context.mac.png)
- **Shows**: A Mac row: header mac-mini, Open ↩, New VM on mac-mini… ⌘N, Open in Terminal, Activity, Ports, Accounts.
- **Data**: mac awake
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 28%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 73%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "Open ↩"; menu item "New VM on mac-mini… ⌘N"; menu item "Open in Terminal"; menu item "Activity"; menu item "Ports"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .qa: CSS opacity 120 ms
- **Leaves**: .row.two: CSS background-color 120 ms (esc)

### `menu.context.mac.asleep`

![menu.context.mac.asleep](stories/menu.context.mac.asleep.png)

- **Capture**: [dark](stories/menu.context.mac.asleep.png)
- **Shows**: An asleep Mac: Open, Wake macbook-air.
- **Data**: mac asleep
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 37%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 69%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "Open ↩"; menu item "Wake macbook-air"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: .row.two: CSS background-color 120 ms (esc)

### `menu.context.grant`

![menu.context.grant](stories/menu.context.grant.png)

- **Capture**: [dark](stories/menu.context.grant.png)
- **Shows**: Right-click a grant: "GitHub granted to linux-1", Remove… (red).
- **Data**: grant
- **Interactions**: "All machines" → back; "Sign in" → signin(thismac:npm); "Full list in Terminal" → terminal(station accounts --all-machines); "Grant a password…" → grant; menu item "Remove…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: .hov: CSS opacity 120 ms (esc)

### `menu.awake`

![menu.awake](stories/menu.awake.png)

- **Capture**: [dark](stories/menu.awake.png)
- **Shows**: Right-click the Keep awake row: Keep This Mac awake: For 1 hr, For 4 hr, ✓ Until 18:00, then Turn off / Awake settings.
- **Data**: awake mode
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 39%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 52%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "For 1 hr"; menu item "For 4 hr"; menu item "Until 18:00"; menu item "Let it sleep"; … 1 more
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .place.nav: CSS background-color 120 ms
- **Leaves**: no recorded animation (esc)

### `menu.page.mac`

![menu.page.mac](stories/menu.page.mac.png)

- **Capture**: [dark](stories/menu.page.mac.png)
- **Shows**: mac-mini's More menu: Open in Terminal ⌘T, Restart…, Stop everything… (red), Forget… (red).
- **Data**: mac, this-Mac flag, asleep
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → push(vm:review-bot); "CPU 44%" → seg(activity); "Memory 19 GB free" → seg(activity); "Network 0.8 MB/s" → seg(activity); "infra healthy · up 12 days 17%" → push(vm:infra); "linux-3 editing src/auth/refresh.ts 67%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini); menu item "Open in Terminal ⌘T"; menu item "Restart…"; menu item "Stop everything…"; menu item "Forget…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .iconbtn: CSS background-color 120 ms; .iconbtn: CSS color 120 ms
- **Leaves**: no recorded animation (esc)

### `menu.page.this-mac`

![menu.page.this-mac](stories/menu.page.this-mac.png)

- **Capture**: [dark](stories/menu.page.this-mac.png)
- **Shows**: This Mac's More menu: no Restart (the Apple menu owns it), no Forget; Stop everything….
- **Data**: as menu.page.mac
- **Interactions**: "All machines" → back; "New VM" → newvm(thismac) [New VM on This Mac · ⌘N]; "More options for This Mac" → pageMenu(mac:thismac); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "CPU 21%" → seg(activity); "Memory 23 GB free" → seg(activity); "Network 0.9 MB/s" → seg(activity); "linux-1 running tests · 41/58 pass 36%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "agent-7f finished · tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "api stopped · persistent" → push(vm:api); "Keep This Mac awake" → awakeToggle; "Upkeep firewall off · 3 more" → unfold(upkeep:thismac); "How to" → howto; "Copy fix" → copyCmd(launchctl bootout system/com.docker.vmnetd); "Sign in" → signin(thismac:npm); "Update" → updateAll; menu item "Open in Terminal ⌘T"; menu item "Stop everything…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .iconbtn: CSS background-color 120 ms; .iconbtn: CSS color 120 ms
- **Leaves**: no recorded animation (esc)

### `menu.page.mac.asleep`

![menu.page.mac.asleep](stories/menu.page.mac.asleep.png)

- **Capture**: [dark](stories/menu.page.mac.asleep.png)
- **Shows**: An asleep Mac's More menu (macbook-air): header, Wake macbook-air, then Forget… (red); no Open in Terminal, Restart or Stop everything.
- **Data**: mac asleep, this-Mac flag
- **Interactions**: "All machines" → back; "More options for macbook-air" → pageMenu(mac:macbook-air); "Wake" → wake(macbook-air); "Tell me when it wakes" → remind(macbook-air); menu item "Wake macbook-air"; menu item "Forget…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .iconbtn: CSS background-color 120 ms; .iconbtn: CSS color 120 ms; path: CSS draw 340 ms linear
- **Leaves**: path: CSS draw 340 ms linear (esc)

### `menu.page.workspace`

![menu.page.workspace](stories/menu.page.workspace.png)

- **Capture**: [dark](stories/menu.page.workspace.png)
- **Shows**: linux-1's More menu: Copy ssh command, Share a port…, Grant a password…, Stop…, Delete… (red).
- **Data**: vm
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Full log" → terminal(claude log on linux-1); "CPU · 4 cores 34%" → seg(activity); "Memory 3.0 GB free" → seg(activity); "Network 0.8 MB/s" → seg(activity); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Share Astro" → sharePort(linux-1:4321); "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]; menu item "Copy ssh command"; menu item "Share a port…"; menu item "Grant a password…"; menu item "Stop…"; menu item "Delete…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .iconbtn: CSS background-color 120 ms; .iconbtn: CSS color 120 ms
- **Leaves**: no recorded animation (esc)

### `menu.page.server`

![menu.page.server](stories/menu.page.server.png)

- **Capture**: [dark](stories/menu.page.server.png)
- **Shows**: infra's More menu: Restart…, Stop… (red), Delete… (red).
- **Data**: server
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Healthy · 4 of 4 checked after every boot · 40 s" → unfold(health:infra); "Restart…" → restartAsk(infra); "Show log" → terminal(journalctl -u k3s on infra); "CPU · 8 cores 14%" → seg(activity); "Memory 11.5 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "ssh admin@infra" → copy(ssh admin@infra); "Open in Terminal" → terminal(infra); menu item "Restart…"; menu item "Stop…"; menu item "Delete…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .iconbtn: CSS background-color 120 ms; .iconbtn: CSS color 120 ms
- **Leaves**: no recorded animation (esc)

### `menu.page.ports`

![menu.page.ports](stories/menu.page.ports.png)

- **Capture**: [dark](stories/menu.page.ports.png)
- **Shows**: Ports' More menu: Share a port…, Declare a port public….
- **Data**: none
- **Interactions**: "All machines" → back; "More options for Ports" → pageMenu(ports:all); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Open Preview" → openUrl(http://linux-3.tail3e9a.ts.net:8080); "Copy link to Preview" → copy(http://linux-3.tail3e9a.ts.net:8080); "Show QR for Preview (8080)" → qrpop(p5); "Share Astro" → sharePort(linux-1:4321); "Share Node inspector" → sharePort(linux-1:9229); "Share Postgres" → sharePort(linux-3:5432); "Full list in Terminal" → terminal(station ports --all-machines); "Share a port…" → sharePortSheet; menu item "Share a port…"; menu item "Declare a port public…"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .iconbtn: CSS background-color 120 ms; .iconbtn: CSS color 120 ms
- **Leaves**: no recorded animation (esc)

### `menu.new-vm.on`

![menu.new-vm.on](stories/menu.new-vm.on.png)

- **Capture**: [dark](stories/menu.new-vm.on.png)
- **Shows**: The On popup in New VM: each Mac with its free memory, macbook-air disabled "asleep".
- **Data**: macs with free memory, asleep
- **Interactions**: field "Task for Claude"; "mac-mini 19 GB free" → nvOn; "Ephemeral · Linux · 8 GB" → nvOpts; "Cancel" → closeSheet; "Create" → createVm [↩]; "More ways to create" → nvMore; menu item "This Mac 23 GB free"; menu item "mac-mini 19 GB free"; menu item "macbook-air asleep" (disabled)
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: no recorded animation (esc)

### `menu.new-vm.create`

![menu.new-vm.create](stories/menu.new-vm.create.png)

- **Capture**: [dark](stories/menu.new-vm.create.png)
- **Shows**: The split button's menu: Create ↩, Create and Open in Terminal ⌥↩, Create and Start Claude ⌘↩.
- **Data**: kind
- **Interactions**: field "Task for Claude"; "mac-mini 19 GB free" → nvOn; "Ephemeral · Linux · 8 GB" → nvOpts; "Cancel" → closeSheet; "Create" → createVm [↩]; "More ways to create" → nvMore; menu item "Create ↩"; menu item "Create and Open in Terminal ⌥↩"; menu item "Create and Start Claude ⌘↩"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: .btn.tint: CSS background-color 120 ms (esc)

### `menu.grant.to`

![menu.grant.to](stories/menu.grant.to.png)

- **Capture**: [dark](stories/menu.grant.to.png)
- **Shows**: The To popup of Grant a password: the workspaces of This Mac.
- **Data**: workspaces not stopped on This Mac
- **Interactions**: "linux-1" → shTo; "GitHub token" → sh(entry:pw-github); "Claude token" → sh(entry:pw-claude); "Cloudflare token" → sh(entry:pw-cloudflare) (disabled); "Cancel" → closeSheet; "Grant GitHub token" → doGrant; menu item "linux-1"; menu item "agent-7f"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: no recorded animation (esc)

### `menu.share-port.of`

![menu.share-port.of](stories/menu.share-port.of.png)

- **Capture**: [dark](stories/menu.share-port.of.png)
- **Shows**: Share a port of: the running workspaces with their Mac.
- **Data**: running workspaces
- **Interactions**: "linux-1 on This Mac" → spOf; "Share" → sharePort(linux-1:4321); "Cancel" → closeSheet; menu item "linux-1 This Mac"; menu item "agent-7f This Mac"; menu item "linux-3 mac-mini"; menu item "review-bot mac-mini"
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy
- **Leaves**: no recorded animation (esc)

## Popovers, tooltips, HUD

### `pop.qr`

![pop.qr](stories/pop.qr.png) ![pop.qr light](stories/pop.qr.light.png)

- **Capture**: [dark](stories/pop.qr.png), [light](stories/pop.qr.light.png)
- **Shows**: QR popover anchored to the port row: a scannable QR of the tailnet link, the short link, Copy link and Stop sharing.
- **Data**: port url
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Full log" → terminal(claude log on linux-1); "CPU · 4 cores 36%" → seg(activity); "Memory 3.0 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Share Astro" → sharePort(linux-1:4321); "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]; "Copy" → copy(http://linux-1.tail3e9a.ts.net:5173); "Open" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Stop sharing…" → stopShare(p1)
- **Enters**: #pop: opacity+transform 0 scale(.94) → 1 none 200 ms snappy; .row.two: CSS background-color 120 ms; .round.sm: CSS opacity 120 ms
- **Leaves**: #pop: opacity+transform 1 → 0 scale(.97) 200 ms snappy (esc)

### `pop.qr.public`

![pop.qr.public](stories/pop.qr.public.png)

- **Capture**: [dark](stories/pop.qr.public.png)
- **Shows**: QR of a public route (infra 443): the link is the domain "example.com" and the line under it is "Anyone can open it · Cloudflare Tunnel" instead of "Opens on any device of your tailnet"; then Copy and Open ↗; no "Stop sharing…" link (a public route is declared, not shared from here).
- **Data**: port url, reach public, route name
- **Interactions**: "mac-mini" → back; "More options for infra" → pageMenu(server:infra); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Healthy · 4 of 4 checked after every boot · 40 s" → unfold(health:infra); "Restart…" → restartAsk(infra); "Show log" → terminal(journalctl -u k3s on infra); "CPU · 8 cores 22%" → seg(activity); "Memory 10.5 GB free" → seg(activity); "Network 0.9 MB/s" → seg(activity); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "ssh admin@infra" → copy(ssh admin@infra); "Open in Terminal" → terminal(infra); "Copy" → copy(https://example.com); "Open" → openUrl(https://example.com)
- **Enters**: #pop: opacity+transform 0 scale(.94) → 1 none 200 ms snappy; .round.sm: CSS opacity 120 ms
- **Leaves**: #pop: opacity+transform 1 → 0 scale(.97) 200 ms snappy; .round.sm: CSS opacity 120 ms (esc)

### `pop.info`

![pop.info](stories/pop.info.png)

- **Capture**: [dark](stories/pop.info.png)
- **Shows**: Info popover: Private links open only on devices signed in to your tailnet; Public routes are a Cloudflare Tunnel or a Funnel you declared.
- **Data**: none
- **Interactions**: "All machines" → back; "More options for Ports" → pageMenu(ports:all); "Open Traefik" → openUrl(https://example.com); "Copy link to Traefik" → copy(https://example.com); "Show QR for Traefik (443)" → qrpop(p3); "About private links" → info(ports); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Copy Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Copy link to Kubernetes API" → copy(http://infra.tail3e9a.ts.net:6443); "Open Preview" → openUrl(http://linux-3.tail3e9a.ts.net:8080); "Copy link to Preview" → copy(http://linux-3.tail3e9a.ts.net:8080); "Show QR for Preview (8080)" → qrpop(p5); "Share Astro" → sharePort(linux-1:4321); "Share Node inspector" → sharePort(linux-1:9229); "Share Postgres" → sharePort(linux-3:5432); "Full list in Terminal" → terminal(station ports --all-machines); "Share a port…" → sharePortSheet
- **Enters**: #pop: opacity+transform 0 scale(.94) → 1 none 200 ms snappy
- **Leaves**: #pop: opacity+transform 1 → 0 scale(.97) 200 ms snappy (esc)

### `pop.hud.copy`

![pop.hud.copy](stories/pop.hud.copy.png)

- **Capture**: [dark](stories/pop.hud.copy.png)
- **Shows**: The HUD under the notch: "Copied ssh linux-1" (black capsule, icon + words, 2.6 s).
- **Data**: none
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Full log" → terminal(claude log on linux-1); "CPU · 4 cores 37%" → seg(activity); "Memory 2.9 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Share Astro" → sharePort(linux-1:4321); "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]
- **Enters**: #hud: CSS hud 2600 ms linear

### `pop.hud.terminal`

![pop.hud.terminal](stories/pop.hud.terminal.png)

- **Capture**: [dark](stories/pop.hud.terminal.png)
- **Shows**: HUD "linux-1 opened in Terminal".
- **Data**: none
- **Interactions**: "This Mac" → back; "More options for linux-1" → pageMenu(vm:linux-1); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Full log" → terminal(claude log on linux-1); "CPU · 4 cores 40%" → seg(activity); "Memory 2.8 GB free" → seg(activity); "Network 0.7 MB/s" → seg(activity); "Open Vite" → openUrl(http://linux-1.tail3e9a.ts.net:5173); "Copy link to Vite" → copy(http://linux-1.tail3e9a.ts.net:5173); "Show QR for Vite (5173)" → qrpop(p1); "Open API dev server" → openUrl(http://linux-1.tail3e9a.ts.net:3000); "Copy link to API dev server" → copy(http://linux-1.tail3e9a.ts.net:3000); "Show QR for API dev server (3000)" → qrpop(p2); "Share Astro" → sharePort(linux-1:4321); "ssh linux-1" → copy(ssh linux-1); "Open in Terminal" → attach(linux-1) [⌘↩]
- **Enters**: .btn.big: CSS background-color 120 ms; #hud: CSS hud 2600 ms linear

## Find (Cmd-K)

Keys: type to filter, ↑ ↓ move, ↩ run, ⌘↩ attach a workspace, esc closes Find.

### `find.empty`

![find.empty](stories/find.empty.png) ![find.empty light](stories/find.empty.light.png)

- **Capture**: [dark](stories/find.empty.png), [light](stories/find.empty.light.png)
- **Shows**: Find opened over home: the field "Find a machine, a port, an action…", then Needs you (the top 3 with their verbs), Recent (linux-1, mac-mini, infra), Actions (New VM…, Keep awake…); the selected row shows its key; a hint line at the bottom.
- **Data**: needs(); recent machines; actions available now
- **Interactions**: field "Find a machine, a port, an action…"; result "Review review-bot · PR #42 is ready"; result "linux-1 working · This Mac"; result "mac-mini 3 VMs · 19 GB free"; result "infra healthy · mac-mini"; result "Keep This Mac awake 4 hr"; result "New VM on This Mac 23 GB free"; result "New VM on mac-mini 19 GB free"
- **Enters**: .find: opacity 0 → 1 200 ms snappy; .fh: transform translateY(-6px) → none 340 ms smooth; #panel: height 689px → 400px 340 ms smooth
- **Leaves**: #panel: height 400px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy (esc)

### `find.down`

![find.down](stories/find.down.png)

- **Capture**: [dark](stories/find.down.png)
- **Shows**: Find opened while infra is down: Needs you leads with "Restart infra" (the direct verb, no "asks first"), then Review review-bot; Recent shows infra as "down since HH:MM · mac-mini"; the footer verb reads Restart ↩.
- **Data**: needs() with the down server first; recent machines with their state
- **Interactions**: field "Find a machine, a port, an action…"; result "Restart infra"; result "Review review-bot · PR #42 is ready"; result "linux-1 working · This Mac"; result "mac-mini 3 VMs · 19 GB free"; result "infra down since 15:10 · mac-mini"; result "Keep This Mac awake 4 hr"; result "New VM on This Mac 23 GB free"; result "New VM on mac-mini 19 GB free"
- **Enters**: .find: opacity 0 → 1 200 ms snappy; .fh: transform translateY(-6px) → none 340 ms smooth; #panel: height 669px → 430px 340 ms smooth; #hud: CSS hud 2600 ms linear
- **Leaves**: #panel: height 430px → 669px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy; .hero-glyph.arrive: CSS arrive 460 ms linear; #hud: CSS hud 2600 ms linear (esc)

### `find.port`

![find.port](stories/find.port.png)

- **Capture**: [dark](stories/find.port.png)
- **Shows**: Find "5173": the port result linux-1:5173 Vite with the match highlighted.
- **Data**: ports
- **Interactions**: field "Find a machine, a port, an action…"; result "Vite · linux-1:5173"
- **Enters**: #panel: height 400px → 172px 200 ms snappy
- **Leaves**: #panel: height 142px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `find.machine`

![find.machine](stories/find.machine.png)

- **Capture**: [dark](stories/find.machine.png)
- **Shows**: Find "mini": Machines (mac-mini with its kind), then Sections of it (Activity of mac-mini, Ports of mac-mini), Actions (New VM on mac-mini).
- **Data**: machines, sections, actions
- **Interactions**: field "Find a machine, a port, an action…"; result "mac-mini 3 VMs · 19 GB free ⌘2"; result "New VM on mac-mini 19 GB free"; result "Ports of mac-mini"; result "Activity of mac-mini"
- **Enters**: #panel: height 400px → 520px 200 ms snappy
- **Leaves**: #panel: height 280px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy (esc)

### `find.action`

![find.action](stories/find.action.png)

- **Capture**: [dark](stories/find.action.png)
- **Shows**: Find "new": the New VM actions per Mac.
- **Data**: actions
- **Interactions**: field "Find a machine, a port, an action…"; result "New VM on This Mac 23 GB free ⌘N"; result "New VM on mac-mini 19 GB free"
- **Enters**: #panel: height 400px → 520px 200 ms snappy
- **Leaves**: #panel: height 172px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy (esc)

### `find.awake`

![find.awake](stories/find.awake.png)

- **Capture**: [dark](stories/find.awake.png)
- **Shows**: Find "awake": Keep This Mac awake 4 hr, and the Awake section.
- **Data**: actions
- **Interactions**: field "Find a machine, a port, an action…"; result "Keep This Mac awake 4 hr"; result "Awake · All Macs"
- **Enters**: #panel: height 400px → 520px 200 ms snappy
- **Leaves**: #panel: height 226px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy (esc)

### `find.keyboard`

![find.keyboard](stories/find.keyboard.png)

- **Capture**: [dark](stories/find.keyboard.png)
- **Shows**: Arrow keys move the selection; the selected row shows ↩ (and ⌘↩ attach for a workspace) in the hint.
- **Data**: as find.machine
- **Interactions**: field "Find a machine, a port, an action…"; result "linux-1 working · This Mac"; result "linux-3 working · mac-mini"; result "Vite · linux-1:5173"; result "API dev server · linux-1:3000"; result "Preview · linux-3:8080"; result "Show 3 more"; result "Open linux-1 in Terminal"; result "Share a port of linux-1"; result "Grant a password to linux-1"; result "Show 4 more"
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 460px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy (esc)

### `find.more-open`

![find.more-open](stories/find.more-open.png)

- **Capture**: [dark](stories/find.more-open.png)
- **Shows**: "Show N more" picked (click, or ↩ on it): its group opens in place to every match (here Ports, 6 rows; the next group keeps its own "Show 4 more"), the field keeps the query, the panel height follows on snappy.
- **Data**: as find.machine
- **Interactions**: field "Find a machine, a port, an action…"; result "linux-1 working · This Mac"; result "linux-3 working · mac-mini"; result "Vite · linux-1:5173"; result "API dev server · linux-1:3000"; result "Preview · linux-3:8080"; result "Astro · linux-1:4321"; result "Node inspector · linux-1:9229"; result "Postgres · linux-3:5432"; result "Open linux-1 in Terminal"; result "Share a port of linux-1"; result "Grant a password to linux-1"; result "Show 4 more"
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)
- **Leaves**: #panel: height 520px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy; .row.two: CSS background-color 120 ms; .qa: CSS opacity 120 ms (esc)

### `find.nomatch`

![find.nomatch](stories/find.nomatch.png)

- **Capture**: [dark](stories/find.nomatch.png)
- **Shows**: No match: one line "No match for zzzq" and a hint.
- **Data**: none
- **Interactions**: field "Find a machine, a port, an action…"
- **Enters**: #panel: height 400px → 142px 200 ms snappy
- **Leaves**: #panel: height 171px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy; circle: CSS stroke-dashoffset 340 ms cubic-bezier(0.32, 0.72, 0, 1) (esc)

### `find.type-to-find`

![find.type-to-find](stories/find.type-to-find.png)

- **Capture**: [dark](stories/find.type-to-find.png)
- **Shows**: Typing a letter on All machines opens Find with that letter.
- **Data**: as find.machine
- **Interactions**: field "Find a machine, a port, an action…"; result "linux-1 working · This Mac"; result "linux-3 working · mac-mini"; result "Vite · linux-1:5173"; result "API dev server · linux-1:3000"; result "Preview · linux-3:8080"; result "Show 3 more"; result "Clean up ephemeral workspaces on This Mac…"; result "Ports · All machines"; result "Accounts · All machines"; result "Awake · All Macs"
- **Enters**: .find: opacity 0 → 1 200 ms snappy; .fh: transform translateY(-6px) → none 340 ms smooth; #panel: height 689px → 514px 340 ms smooth
- **Leaves**: #panel: height 514px → 689px 340 ms smooth; .page: opacity 0 → 1 340 ms smooth; .findwrap.leaving: opacity 1 → 0 200 ms snappy (esc)

## Transitions (frozen mid-flight)

### `transition.push`

![transition.push](stories/transition.push.png)

- **Capture**: [dark](stories/transition.push.png)
- **Shows**: NavigationStack push: the new page slides in from the right already laid out, opaque with a left shadow; the page below moves a third left and dims; the panel height rides the same spring (smooth 340 ms).
- **Data**: none
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → push(vm:review-bot); "CPU 43%" → seg(activity); "Memory 19 GB free" → seg(activity); "Network 0.6 MB/s" → seg(activity); "infra healthy · up 12 days 13%" → push(vm:infra); "linux-3 editing src/auth/refresh.ts 70%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini)
- **Enters**: #panel: height 689px → 506px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth

### `transition.pop`

![transition.pop](stories/transition.pop.png)

- **Capture**: [dark](stories/transition.pop.png)
- **Shows**: Pop (Esc or the back button): the page leaves to the right, the one below returns from −30%.
- **Data**: none
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 30%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 70%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: #panel: height 506px → 689px 340 ms smooth; .page: transform+opacity none 1 → translateX(100%) 0.6 340 ms smooth; .page: transform+opacity translateX(-30%) 0 → none 1 340 ms smooth; .fwrap.morph: CSS morph 340 ms linear

### `transition.jump`

![transition.jump](stories/transition.jump.png)

- **Capture**: [dark](stories/transition.jump.png)
- **Shows**: A sideways jump (a shortcut to another branch, e.g. ⌘E from a Mac): the old page fades out in 130 ms before the new one fades in (320 ms), so two pages never overlap.
- **Data**: none
- **Interactions**: "mac-mini" → back; "1 hr" → awakeFor(1h); "4 hr" → awakeFor(4h); "Until 18:00" → awakeFor(until); "Keep mac-mini awake" → awakeChange(mac-mini)
- **Enters**: #panel: height 506px → 447px 340 ms smooth; .page: transform+opacity none 1 → translateX(-30%) 0 340 ms smooth; .page: transform translateX(100%) → none 340 ms smooth

### `transition.segment`

![transition.segment](stories/transition.segment.png)

- **Capture**: [dark](stories/transition.segment.png)
- **Shows**: Segment change: the indicator slides, the body cross-slides 10 px in the direction of travel (snappy 200 ms), the panel height morphs (smooth).
- **Data**: none
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "6 min" → win(6m); "1 hr" → win(1h); "24 hr" → win(24h); "linux-3 · 4 cores 51%" → push(vm:linux-3); "infra · 8 cores 23%" → push(vm:infra); "review-bot · 4 cores 0%" → push(vm:review-bot)
- **Enters**: #panel: height 506px → 700px 340 ms smooth; .pb: opacity+transform 0 translateX(10px) → 1 none 200 ms snappy; .: CSS color 120 ms; .sel: CSS color 120 ms

### `transition.sheet`

![transition.sheet](stories/transition.sheet.png)

- **Capture**: [dark](stories/transition.sheet.png)
- **Shows**: A sheet unrolls from the panel top: its header is there from the first frame, a clip-path reveals the body downward with a 6 px drop (smooth, 40 ms delay); the scrim fades in.
- **Data**: none
- **Interactions**: field "Task for Claude"; "This Mac 23 GB free" → nvOn; "Ephemeral · Linux · 8 GB" → nvOpts; "Cancel" → closeSheet; "Create" → createVm [↩]; "More ways to create" → nvMore
- **Enters**: .sheet: clipPath+transform+opacity inset(0 0 calc(100% - 46px) 0 round 14px) translateY(-6px) 1 → inset(0 0 0 0 round 14px) none 1 340 ms smooth +40 ms; .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)

### `transition.alert`

![transition.alert](stories/transition.alert.png)

- **Capture**: [dark](stories/transition.alert.png)
- **Shows**: An alert scales from 1.08 to 1 with opacity at the panel centre (smooth).
- **Data**: none
- **Interactions**: "Cancel" → closeSheet; "Delete agent-7f" → sheetYes
- **Enters**: .alert: opacity+transform 0 translate(-50%,-50%) scale(1.08) → 1 translate(-50%,-50%) 340 ms smooth; .scrim.on: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1)

### `transition.find`

![transition.find](stories/transition.find.png)

- **Capture**: [dark](stories/transition.find.png)
- **Shows**: Find opens: the card fades in (snappy) and its field drops 6 px into place (smooth); the page behind dims.
- **Data**: none
- **Interactions**: field "Find a machine, a port, an action…"; result "Review review-bot · PR #42 is ready"; result "linux-1 working · This Mac"; result "mac-mini 3 VMs · 19 GB free"; result "infra healthy · mac-mini"; result "Keep This Mac awake 4 hr"; result "New VM on This Mac 23 GB free"; result "New VM on mac-mini 19 GB free"
- **Enters**: .find: opacity 0 → 1 200 ms snappy; .fh: transform translateY(-6px) → none 340 ms smooth; #panel: height 689px → 400px 340 ms smooth

### `transition.menu`

![transition.menu](stories/transition.menu.png)

- **Capture**: [dark](stories/transition.menu.png)
- **Shows**: A menu appears: scale .97 → 1 with opacity, snappy 200 ms.
- **Data**: none
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 41%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 73%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find; menu item "New VM… ⌘N"; menu item "Add a Mac…"; menu item "Open station in Terminal ⌘T"; menu item "Open at login"; … 1 more
- **Enters**: .menu: opacity+transform 0 scale(.97) → 1 none 200 ms snappy; .round: CSS background-color 120 ms

### `transition.row-leave`

![transition.row-leave](stories/transition.row-leave.png)

- **Capture**: [dark](stories/transition.row-leave.png)
- **Shows**: A deleted VM leaves its list (Delete agent-7f confirmed, or Clean up): the row slides 10 pt right, fades and collapses to zero height (smooth, 340 ms), the rows below close up; HUD "agent-7f deleted on This Mac".
- **Data**: none
- **Interactions**: "the scrim (click outside the card)" → closeSheet; "Cancel" → closeSheet; "Delete agent-7f" → sheetYes
- **Enters**: .alert: opacity+transform 1 translate(-50%,-50%) → 0 translate(-50%,-50%) scale(.96) 200 ms snappy; .scrim: CSS opacity 340 ms cubic-bezier(0.32, 0.72, 0, 1); #yes: CSS background-color 118 ms; .row.nav: CSS leave 340 ms linear; #hud: CSS hud 2600 ms linear

### `transition.reduce-motion.push`

![transition.reduce-motion.push](stories/transition.reduce-motion.push.png)

- **Capture**: [dark](stories/transition.reduce-motion.push.png)
- **Shows**: Reduce Motion: every movement becomes a dissolve in two halves (240 ms), so two pages never print over each other; heights change in 200 ms ease-in-out.
- **Data**: none
- **Interactions**: "All machines" → back; "New VM" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "More options for mac-mini" → pageMenu(mac:mac-mini); "Overview" → seg(overview); "Activity" → seg(activity) [⌘I]; "Ports" → seg(ports) [⌘P]; "Accounts" → seg(accounts) [⌘U]; "Review" → push(vm:review-bot); "CPU 46%" → seg(activity); "Memory 19 GB free" → seg(activity); "Network 0.6 MB/s" → seg(activity); "infra healthy · up 12 days 22%" → push(vm:infra); "linux-3 editing src/auth/refresh.ts 53%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "review-bot fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "Keep mac-mini awake" → awakeChange(mac-mini); "Upkeep all fine" → unfold(upkeep:mac-mini)
- **Enters**: #panel: height 689px → 506px 200 ms ease-in-out; .page: opacity 1 → 0 240 ms ease-in-out; .page: opacity 0 → 1 240 ms ease-in-out

## Phone (the explainer page)

### `phone.explainer`

![phone.explainer](stories/phone.explainer.png) ![phone.explainer light](stories/phone.explainer.light.png)

- **Capture**: [dark](stories/phone.explainer.png), [light](stories/phone.explainer.light.png)
- **Shows**: The explainer page on a phone (390 pt): the panel shown open and loaded inline, the dock of scenarios, then the "why" sections.
- **Data**: as home.needs
- **Interactions**: "New VM" → newvm(thismac) [New VM · on This Mac, the most free memory · ⌘N]; "More options for Spwn Menu" → gear; "Review" → push(vm:review-bot); "Ports, 5 shared" → push(ports:all) [5 shared · ⌘P]; "Accounts 1 signed out" → push(accounts:all) [Accounts · All machines · ⌘U]; "Awake, until 18:00" → push(awake:all) [Awake until 18:00 · ⌘E · right-click for a duration]; "Keep This Mac awake" → awakeToggle; "review-botmac-mini fixes the token refresh" → push(vm:review-bot); "Open review-bot in Terminal" → attach(review-bot) [Open in Terminal · ⌘↩]; "linux-1This Mac running tests · 41/58 pass 28%" → push(vm:linux-1); "Open linux-1 in Terminal" → attach(linux-1) [Open in Terminal · ⌘↩]; "linux-3mac-mini editing src/auth/refresh.ts 56%" → push(vm:linux-3); "Open linux-3 in Terminal" → attach(linux-3) [Open in Terminal · ⌘↩]; "agent-7f tests pass Clean up" → push(vm:agent-7f); "Clean up" → cleanup(thismac); "infra healthy · up 12 days · 2 ports" → push(vm:infra); "This Mac 1 working4 upkeep 23 GB free" → push(mac:thismac); "New VM on This Mac" → newvm(thismac) [New VM on This Mac · ⌘N]; "mac-mini 1 working1 waiting 19 GB free" → push(mac:mac-mini); "New VM on mac-mini" → newvm(mac-mini) [New VM on mac-mini · ⌘N]; "macbook-air asleep · seen 2 hr ago" → push(mac:macbook-air); "Find a machine, a port, an action… ⌘K" → find
- **Enters**: no animation (a state reached in place; live values tick every 1 to 2 s)

## HUD words: the fixture variants of `pop.hud.*`

Every transient confirmation the prototype fires, read from its code: the gesture, the glyph (prototype icon → SF Symbol) and the words. Each is one fixture of the HUD (the capsule of `pop.hud.copy`, §A3 of SPEC.md); the Catalog lists them under `pop.hud`. Words marked (prototype) are the prototype's placeholders, not product copy.

| Gesture | Glyph | Words |
| --- | --- | --- |
| Copy (a link, a command, ssh) | copy → doc.on.doc | Copied example.com |
| Open in Terminal, ⌘T | term → terminal | Terminal · ssh linux-1 / Terminal · station --on mac-mini / Terminal · station machines |
| Attach (⌘↩, the round Terminal button) | term → terminal | linux-1 opened in Terminal |
| Sign in | term → terminal | Terminal · npm login on This Mac |
| Open a link | ur → safari | Opens example.com in Safari |
| Create (New VM) | box → shippingbox | Creating linux-4 on mac-mini… |
| The new VM answers | checkf → checkmark.circle.fill | linux-4 is ready on mac-mini / linux-4 is ready · opened in Terminal |
| Create with a name in use | exclf → exclamationmark.circle.fill | linux-4 already exists |
| Clean up with nothing finished | broom → (custom, or sparkles) | Nothing to clean up on This Mac |
| Delete confirmed, Clean up | trash → trash | agent-7f deleted on This Mac |
| Start | play → play.fill | api is starting |
| Restart a workspace; Restart a Mac confirmed | restart → arrow.clockwise | linux-1 restarts / mac-mini restarts… |
| Stop confirmed | stop → stop.fill | infra is stopped / linux-1 is stopped · its disk is kept / 2 workspaces stopped on This Mac |
| Stop sharing confirmed | stop → stop.fill | linux-1:5173 is no longer shared |
| Share a listener | checkf → checkmark.circle.fill | linux-3:5432 shared on your tailnet |
| A server goes down | octf → xmark.octagon.fill | infra is down on mac-mini |
| A server recovers | checkf → checkmark.circle.fill | infra is healthy again · down 1 min |
| A port turns exposed | shield → exclamationmark.shield | Port 5000 is exposed on This Mac |
| Show the process (exposed port) | cpu → cpu | ControlCenter (pid 612) listens on *:5000 |
| Grant confirmed | key → key | Granted GitHub token to linux-1 |
| Remove a grant confirmed | key → key | GitHub removed from linux-1 |
| Update (upkeep) | checkf → checkmark.circle.fill | Everything is up to date |
| Awake on (switch, tile, menu, Find) | sun → sun.max | This Mac stays awake until 18:00 |
| Awake off | moon → moon | This Mac may sleep |
| Let mac-mini sleep confirmed | moon → moon | Kept as it was (prototype) |
| Wake (asleep Mac) | wake → (custom, or power) | Wake-on-LAN sent to macbook-air |
| Tell me when it wakes | bell → bell | You will be told when macbook-air wakes |
| Open at login (Spwn Menu) | check → checkmark | Spwn Menu opens at login |
| Fix everything (scenario dock only) | checkf → checkmark.circle.fill | All machines are fine (prototype) |

## Gaps: IA states the prototype does not draw

These are in `ia/PROPOSAL.md` but have no screen in the prototype, so no capture. Each has an id, which `StoryCoverageTests` lists as pending until it is designed into this file as a story of its own; that list only shrinks.

| Gap | Id | Status |
| --- | --- | --- |
| the stale gauges and rows on a remote Mac's own page (see `mac.mini.loading`) | `mac.mini.stale` | pending: its page draws as `mac.mini.loading`'s, so it is that story until its stale marking is drawn |
| a Mac unreachable, told apart from an asleep one | `mac.unreachable` | pending |
| a Mac's disk locked at boot (chip + Unlock the disk ↗) | `mac.disk-locked` | pending |
| the engine stopped (Start) | `engine.stopped` | pending |
| a server in a restart loop (↺ 5 in 10 min banner) | `server.loop` | pending |
| the Apps, Settings, Disk and Background section pages | `section.apps`, `section.settings`, `section.disk`, `section.background` | pending |
| a misplaced folder (Move… sheet) | `sheet.move-folder` | pending |
| a disk under 5 % free | `mac.disk-low` | pending |
| Ready to host servers with a failing check unfolded | `mac.ready.failing` | pending |
| the Activity of a silent guest ("infra's guest does not answer") | `server.activity.silent` | pending |
| a Mac row folding past 4 workspaces | `mac.many-workspaces` | pending |
| more than 3 Macs collapsing | `home.many-macs` | pending |

Every story of this file is green: its story file, its pages, its dark and light goldens and its dark and light view goldens (`INDEX.md`).

---

145 ids: 144 stories (three of them designed after the prototype, without a capture) and `phone.explainer`; 199 captures of the prototype (33 light). This file is edited by hand: `tools/gen.mjs` would rewrite it from the prototype and undo every story designed since, so it regenerates only into a scratch copy, from which a re-captured prototype story is carried over by hand. Regenerate the captures: `node tools/run.mjs [regex]` then `node tools/gen.mjs` (driver, story definitions and generator in `tools/`; Playwright from /private/tmp/station-work/shot, metadata in /private/tmp/station-work/v4/meta.json).
