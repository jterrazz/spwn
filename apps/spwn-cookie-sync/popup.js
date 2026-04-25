const GATE = "http://127.0.0.1:9000";

const $ = (id) => document.getElementById(id);

async function refresh() {
  const { secret } = await chrome.storage.local.get("secret");
  if (!secret) {
    $("state").innerHTML = '<span class="status-bad">·</span> not paired';
    $("pair-form").hidden = false;
    $("paired").hidden = true;
    return;
  }

  $("pair-form").hidden = true;
  $("paired").hidden = false;

  // Probe gate health
  try {
    const r = await fetch(`${GATE}/sync/status`, { headers: { "X-Spwn-Secret": secret } });
    if (!r.ok) throw new Error("gate rejected");
    const status = await r.json();
    $("state").innerHTML = '<span class="status-ok">✓</span> paired with spwn-gate';
    const rows = (status.providers || []).map((p) => {
      const seen = p.last_sync ? new Date(p.last_sync).toLocaleString() : "no sync yet";
      return `<div class="row"><span>${p.name}</span><span class="muted">${seen}</span></div>`;
    });
    $("providers").innerHTML = rows.length ? rows.join("") : '<div class="muted">no providers configured</div>';
  } catch (_) {
    $("state").innerHTML = '<span class="status-bad">·</span> gate not reachable on 127.0.0.1:9000';
  }
}

$("pair").addEventListener("click", async () => {
  const v = $("secret").value.trim();
  if (!v) return;
  await chrome.storage.local.set({ secret: v });
  refresh();
});

$("unpair").addEventListener("click", async () => {
  await chrome.storage.local.remove("secret");
  refresh();
});

refresh();
