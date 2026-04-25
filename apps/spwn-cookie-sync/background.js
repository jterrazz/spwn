// spwn cookie sync — service worker
//
// Watches allowlisted hosts for cookie changes and tab loads, pushes
// fresh cookies to a locally-running spwn-gate at /sync/<provider>.
// No pairing, no secret — the gate listens on 127.0.0.1 and rejects
// anything not in its per-provider cookie allowlist, which is enough
// for personal use.
//
// Provider list comes from the gate at /sync/providers, refreshed
// every 5 min — so adding a new provider in spwn doesn't require an
// extension reinstall.

const GATE = "http://127.0.0.1:9000";

let providers = {}; // domain → { name, cookies: [string] }

async function loadProviders() {
  try {
    const resp = await fetch(`${GATE}/sync/providers`);
    if (!resp.ok) return;
    const list = await resp.json();
    const next = {};
    for (const p of list) {
      for (const d of p.domains || []) next[d] = { name: p.name, cookies: p.cookies };
    }
    providers = next;
  } catch (_) {
    /* gate down — keep last-known providers */
  }
}

function matchProvider(host) {
  host = host.replace(/^www\./, "");
  for (const domain in providers) {
    if (host === domain || host.endsWith("." + domain)) return providers[domain];
  }
  return null;
}

async function pushCookies(provider, url) {
  const cookies = {};
  for (const name of provider.cookies) {
    const c = await chrome.cookies.get({ url, name });
    if (c) cookies[name] = c.value;
  }
  if (Object.keys(cookies).length === 0) return; // user not logged in to this site

  try {
    await fetch(`${GATE}/sync/${provider.name}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        cookies,
        captured: new Date().toISOString(),
      }),
    });
  } catch (_) {
    /* gate down — silent retry on next event */
  }
}

chrome.tabs.onUpdated.addListener(async (_tabId, info, tab) => {
  if (info.status !== "complete" || !tab.url) return;
  let host;
  try {
    host = new URL(tab.url).hostname;
  } catch (_) {
    return;
  }
  const p = matchProvider(host);
  if (p) await pushCookies(p, tab.url);
});

// Real-time sync when the site rotates a cookie mid-session (X
// rotates ct0 on actions, LinkedIn refreshes JSESSIONID).
chrome.cookies.onChanged.addListener(async (change) => {
  if (change.removed) return;
  const c = change.cookie;
  const p = matchProvider(c.domain);
  if (!p || !p.cookies.includes(c.name)) return;
  const url = (c.secure ? "https://" : "http://") + c.domain.replace(/^\./, "") + (c.path || "/");
  await pushCookies(p, url);
});

loadProviders();
setInterval(loadProviders, 5 * 60 * 1000);
