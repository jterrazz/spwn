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

function renderProvider(p) {
  const connected = p.has_cookies || !!p.last_sync;
  const status = connected ? "connected" : "pending";
  const meta = connected
    ? `synced ${relativeTime(p.last_sync)} — ${(p.domains || []).join(", ")}`
    : `visit ${(p.domains || []).join(" or ")} once to sync`;

  return `
    <div class="provider">
      <span class="glyph ${connected ? "glyph-on" : "glyph-off"}">${connected ? "●" : "○"}</span>
      <span class="name">${p.name}</span>
      <span class="tag ${connected ? "tag-connected" : "tag-pending"}">${status}</span>
    </div>
    <div class="meta">${meta}</div>
  `;
}

async function refresh() {
  try {
    const r = await fetch(`${GATE}/sync/status`);
    if (!r.ok) throw new Error(`status ${r.status}`);
    const data = await r.json();
    $("gate-status").innerHTML = '<span class="gate-ok">●</span> gate connected';
    const list = data.providers || [];
    if (list.length === 0) {
      $("providers").className = "empty";
      $("providers").textContent = "no providers registered yet — start a gate element that uses cookies";
    } else {
      $("providers").className = "";
      $("providers").innerHTML = list.map(renderProvider).join("");
    }
  } catch (_) {
    $("gate-status").innerHTML = '<span class="gate-bad">●</span> gate not reachable on 127.0.0.1:9000 — run `spwn gate start`';
    $("providers").className = "empty";
    $("providers").textContent = "—";
  }
}

refresh();
setInterval(refresh, 5000); // popup is open: poll every 5s for live updates
