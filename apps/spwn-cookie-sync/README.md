# spwn cookie sync — Chrome extension

A tiny browser extension that syncs your logged-in-session cookies to
a locally-running [spwn-gate](../gate). Lets spwn agents act as you
on sites where the agent UX would otherwise need OAuth, scraper-cookie
juggling, or per-container login.

The pattern: you stay logged in to X, LinkedIn, etc. on your normal
browser. As you browse, this extension pushes the session cookies
(`auth_token`, `ct0`, `li_at`, …) to the gate. The gate persists
them. Spwn elements that need those cookies (the X scraper today,
linkedin-mcp tomorrow) read them at request time. **The agent's
identity is your identity** — it's literally your session, signed
in by you.

Anti-bot detection is the lowest possible because the cookies come
from a real human's real browser. The gate respects rate limits;
your browsing patterns refresh sessions naturally.

## Install (development mode)

1. Make sure the gate is running: `spwn gate start`.

2. Open `chrome://extensions/` (Brave / Edge / Arc work too — same
   extension format).

3. Toggle **Developer mode** (top-right).

4. Click **Load unpacked** → select the `apps/spwn-cookie-sync/`
   folder of this repo.

That's it. No pairing, no secret. The popup shows live status:

```
spwn cookie sync                     ● gate connected
─────────────────────────────────────────────────────
● x          connected   synced 12s ago — x.com, twitter.com
○ linkedin   pending     visit linkedin.com or linkedin.fr to sync
─────────────────────────────────────────────────────
gate at 127.0.0.1:9000 — visit a listed site to sync its cookies
```

The provider list is driven by the gate — every gate element that
needs cookies declares its `CookieProvider`, and the extension picks
them up automatically via `/sync/providers`. Add a new element with
cookies and it shows up in the popup at the next refresh.

## How it works

```
your Chrome (this extension)
  page-load on x.com           → cookies → POST 127.0.0.1:9000/sync/x
  cookie rotation (e.g. ct0)   → cookies → POST /sync/x
  SPA route change (X profile) → cookies → POST /sync/x
                                            │
spwn-gate (host container)                  ▼
  /sync/<provider>                  ~/.spwn/credentials/<provider>/cookies.json
  rejects unknown provider          atomic write, 0600
  rejects unknown cookie names      └── x-mcp, linkedin-mcp, … pick up fresh cookies
```

## Trust model

No shared secret. Two layers:

- **Localhost-only binding.** The gate listens on `127.0.0.1:9000`,
  so only processes already on your machine can reach `/sync`.
  Anything that has local execution can already do worse to you.
- **Per-element cookie allowlist.** Each gate element declares the
  exact cookie names it consumes (e.g. X = `auth_token` + `ct0`).
  The gate drops anything else from the body, so a fork of this
  extension can't sneak unrelated cookies onto disk.

## Permissions explained

| Permission | Why |
|---|---|
| `cookies` | Read session cookies for allowlisted domains so they can be synced. Cookies for non-allowlisted domains are never touched. |
| `tabs` | Detect when a tab finishes loading on an allowlisted domain so we know when to sync. Page content is never read. |
| `webNavigation` | Catch SPA route changes (X is heavily SPA — the page never reloads, but the URL and the relevant cookies change). |
| `storage` | Reserved for future per-provider toggles. Currently unused. |
| `host_permissions: x.com, …` | Required for the cookies API to read those specific hosts. |
| `host_permissions: 127.0.0.1:9000` | Allow `fetch()` to the local gate. |

## Troubleshooting

- **Popup says "gate not reachable on 127.0.0.1:9000"** — start the
  gate: `spwn gate start`. Check `spwn gate status`.
- **Popup says ○ pending but I'm logged in** — visit (or refresh) the
  site once. Sync fires on page-load / SPA navigation / cookie
  rotation; just opening the popup doesn't trigger a sync.
- **Cookies sync but the agent says "no creds"** — gate may be
  caching stale token state. `spwn gate restart` (no `--rebuild`
  needed).

## Future

- Publish to Chrome Web Store (currently dev-mode only).
- Firefox port (`browser.cookies` API is the same shape).
- Per-provider on/off toggles in the popup (don't sync some sites
  even if configured in the gate).
