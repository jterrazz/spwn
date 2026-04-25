// spwn cookie sync — service worker
//
// Listens for tab loads and cookie changes on allowlisted hosts, and
// pushes the relevant session cookies to a locally-running spwn-gate
// at http://127.0.0.1:9000/sync/<provider>. Auth is a shared secret
// the user pastes once into the popup after running
// `spwn cookie-sync register` on their host.
//
// Provider list is fetched from the gate at /sync/providers — no
// extension update needed when spwn adds a new provider.

const GATE = "http://127.0.0.1:9000";

// Throttle per-provider so we don't hammer the gate when the user
// browses several pages on the same site quickly.
const THROTTLE_MS = 30_000;
const lastSync = new Map(); // provider name → epoch ms

let providers = {}; // domain → { name, cookies: [string] }

async function getSecret() {
  const r = await chrome.storage.local.get("secret");
  return r.secret || null;
}

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
  const now = Date.now();
  if ((lastSync.get(provider.name) || 0) + THROTTLE_MS > now) return;

  const secret = await getSecret();
  if (!secret) return; // not paired yet

  const cookies = {};
  for (const name of provider.cookies) {
    const c = await chrome.cookies.get({ url, name });
    if (c) cookies[name] = c.value;
  }
  if (Object.keys(cookies).length === 0) return; // user not logged in to this site

  try {
    const resp = await fetch(`${GATE}/sync/${provider.name}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Spwn-Secret": secret,
      },
      body: JSON.stringify({
        cookies,
        captured: new Date().toISOString(),
      }),
    });
    if (resp.ok) {
      lastSync.set(provider.name, now);
    }
  } catch (_) {
    /* gate down — silent retry on next page load */
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

// Real-time sync when X rotates ct0 or LinkedIn refreshes JSESSIONID
// mid-session (without a full page reload).
chrome.cookies.onChanged.addListener(async (change) => {
  if (change.removed) return;
  const c = change.cookie;
  const p = matchProvider(c.domain);
  if (!p || !p.cookies.includes(c.name)) return;
  // Reconstruct a URL the cookie applies to so chrome.cookies.get finds it.
  const url = (c.secure ? "https://" : "http://") + c.domain.replace(/^\./, "") + (c.path || "/");
  await pushCookies(p, url);
});

// Refresh provider registry on startup + every 5 min so the extension
// picks up new providers added to the gate without a reinstall.
loadProviders();
setInterval(loadProviders, 5 * 60 * 1000);
