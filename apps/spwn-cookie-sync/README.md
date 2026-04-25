# spwn cookie sync — Chrome extension

A tiny browser extension that syncs your logged-in-session cookies to
a locally-running [spwn-gate](../gate). Lets spwn agents act as you on
sites where the agent UX would otherwise need OAuth, scraper-cookie
juggling, or per-container login.

The pattern: you stay logged in to X/LinkedIn/etc. on your normal
browser. As you browse, this extension pushes the session cookies
(`auth_token`, `ct0`, `li_at`, …) to the gate. The gate persists them.
Spwn elements that need those cookies (XActions, linkedin-mcp, …)
read them at request time. **The agent's identity is your identity** —
it's literally your session, signed in by you, not a service account.

Anti-bot detection risk is the lowest possible because the cookies
come from a real human's real browser. The gate respects rate limits;
your browsing patterns refresh sessions naturally.

## Install (development mode)

1. Run `spwn cookie-sync register` on your host to generate the
   pairing secret. Copy the printed `SP-XXXX-XXXX-XXXX` string.

2. Open `chrome://extensions/` (Brave / Edge / Arc work too — same
   extension format).

3. Toggle **Developer mode** (top-right).

4. Click **Load unpacked** → select the `apps/spwn-cookie-sync/`
   folder of this repo.

5. Click the extension icon in your toolbar → paste the secret →
   click **Pair**.

The popup will show ✓ paired and list every provider configured in
the gate's registry.

## How it works

```
your Chrome (with this extension)
  visits x.com         → cookies → POST 127.0.0.1:9000/sync/x
  visits linkedin.com  → cookies → POST 127.0.0.1:9000/sync/linkedin
                                          │
spwn-gate (host container)                ▼
  /sync/<provider>                  ~/.spwn/credentials/<provider>/cookies.json
  validates X-Spwn-Secret           atomic write, 0600
  rejects unknown provider          notifies elements (cache invalidation)
  rejects unknown cookie names      └── XActions, linkedin-mcp, … now have fresh cookies
```

## Permissions explained

| Permission | Why |
|---|---|
| `cookies` | Read session cookies for allowlisted domains so they can be synced. Cookies for non-allowlisted domains are never touched. |
| `tabs` | Detect when a tab finishes loading on an allowlisted domain so we know when to sync. The page content itself is never read. |
| `storage` | Persist the pairing secret locally between browser restarts. |
| `host_permissions: x.com, linkedin.com, …` | Allow the cookies API for those specific hosts. |
| `host_permissions: 127.0.0.1:9000` | Allow `fetch()` to the local gate. |

## Troubleshooting

- **Popup says "gate not reachable on 127.0.0.1:9000"** — start the
  gate: `spwn gate start`. Check `spwn gate status`.
- **Popup says "paired" but no syncs happen** — visit one of the
  allowlisted sites. Sync only fires on tab-load, not on extension
  install. Check `spwn cookie-sync status` for last-sync timestamps.
- **Cookies sync but agent says "no creds"** — gate may be caching
  stale token state. `spwn gate restart` (no `--rebuild` needed).

## Future

- Publish to Chrome Web Store (currently dev-mode only).
- Firefox port (`browser.cookies` API is similar).
- Per-provider toggles in the popup (don't sync linkedin even if
  configured in the gate).
