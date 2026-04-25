const GATE = "http://127.0.0.1:9000";

const $ = (id) => document.getElementById(id);

function relativeTime(iso) {
  if (!iso) return "no sync yet";
  const t = new Date(iso).getTime();
  const sec = Math.floor((Date.now() - t) / 1000);
  if (sec < 5) return "just now";
  if (sec < 60) return `${sec}s ago`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
  return `${Math.floor(sec / 86400)}d ago`;
}

// Build the host-permission origins for one provider's domains.
// Chrome wants "https://*.x.com/*" pattern. Provider domains arrive
// as bare hosts ("x.com").
function originsFor(domains) {
  return (domains || []).map((d) => `https://*.${d}/*`);
}

async function hasPermission(domains) {
  return chrome.permissions.contains({ origins: originsFor(domains) });
}

function requestPermission(domains) {
  // Must be called synchronously from a user gesture — that's the
  // popup's button click handler, hence wiring this here vs the bg
  // worker.
  return chrome.permissions.request({ origins: originsFor(domains) });
}

function renderProvider(p, granted) {
  const connected = p.has_cookies || !!p.last_sync;
  const status = connected ? "connected" : (granted ? "pending" : "needs access");
  const tagClass = connected ? "tag-connected" : (granted ? "tag-pending" : "tag-needs");
  const meta = connected
    ? `synced ${relativeTime(p.last_sync)} — ${(p.domains || []).join(", ")}`
    : (granted
        ? `visit ${(p.domains || []).join(" or ")} once to sync`
        : `click "Grant access" → confirm in Chrome → visit the site`);

  const grantBtn = !granted
    ? `<button class="grant" data-provider="${p.name}" data-domains="${(p.domains || []).join(",")}">Grant access</button>`
    : "";

  return `
    <div class="provider">
      <span class="glyph ${connected ? "glyph-on" : "glyph-off"}">${connected ? "●" : "○"}</span>
      <span class="name">${p.name}</span>
      <span class="tag ${tagClass}">${status}</span>
      ${grantBtn}
    </div>
    <div class="meta">${meta}</div>
  `;
}

async function refresh() {
  let statusData = null;
  try {
    const r = await fetch(`${GATE}/sync/status`);
    if (r.ok) statusData = await r.json();
  } catch (_) {}

  if (!statusData) {
    $("gate-status").innerHTML = '<span class="gate-bad">●</span> gate not reachable on 127.0.0.1:9000 — run `spwn gate start`';
    $("providers").className = "empty";
    $("providers").textContent = "—";
    return;
  }

  $("gate-status").innerHTML = '<span class="gate-ok">●</span> gate connected';

  const list = statusData.providers || [];
  if (list.length === 0) {
    $("providers").className = "empty";
    $("providers").textContent = "no providers registered yet — install a gate tool that uses cookies (e.g. `spwn install spwn:x`)";
    return;
  }

  // Annotate each provider with whether the user has granted the
  // host permissions for it. Renders "needs access" + Grant button
  // when not granted, "pending" otherwise.
  const enriched = await Promise.all(
    list.map(async (p) => ({ ...p, _granted: await hasPermission(p.domains) })),
  );

  $("providers").className = "";
  $("providers").innerHTML = enriched.map((p) => renderProvider(p, p._granted)).join("");

  // Wire up the Grant buttons.
  for (const btn of document.querySelectorAll("button.grant")) {
    btn.addEventListener("click", async () => {
      const domains = (btn.dataset.domains || "").split(",").filter(Boolean);
      const granted = await requestPermission(domains);
      if (granted) {
        // Tell the bg worker to re-load providers (it'll pick up
        // the new permission state on its next cookie/tab event).
        chrome.runtime.sendMessage({ type: "refresh-providers" });
        await refresh();
      }
    });
  }
}

refresh();
setInterval(refresh, 5000); // popup is open: poll every 5s for live updates
