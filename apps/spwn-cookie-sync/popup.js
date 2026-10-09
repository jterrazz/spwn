const GATE = 'http://127.0.0.1:9000';
const POLL_MS = 5000;
const UNITS = [
    [86_400, 'd'],
    [3600, 'h'],
    [60, 'm'],
];

const element = (id) => document.querySelector(`#${id}`);

function relativeTime(iso) {
    if (typeof iso !== 'string' || iso === '') {
        return 'no sync yet';
    }
    const seconds = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
    if (seconds < 5) {
        return 'just now';
    }
    const [size, unit] = UNITS.find(([unitSeconds]) => seconds >= unitSeconds) ?? [1, 's'];
    return `${Math.floor(seconds / size)}${unit} ago`;
}

// outside: Chrome grants a host permission by origin pattern, `https://*.x.com/*`, and the gate lists bare hosts
const originsFor = (domains) => (domains ?? []).map((domain) => `https://*.${domain}/*`);

function stateOf(provider, granted) {
    const domains = provider.domains ?? [];
    if (
        provider.has_cookies === true ||
        (typeof provider.last_sync === 'string' && provider.last_sync !== '')
    ) {
        return {
            connected: true,
            meta: `synced ${relativeTime(provider.last_sync)} — ${domains.join(', ')}`,
            status: 'connected',
            tag: 'tag-connected',
        };
    }
    return granted
        ? {
              connected: false,
              meta: `visit ${domains.join(' or ')} once to sync`,
              status: 'pending',
              tag: 'tag-pending',
          }
        : {
              connected: false,
              meta: 'click "Grant access" → confirm in Chrome → visit the site',
              status: 'needs access',
              tag: 'tag-needs',
          };
}

function renderProvider(provider, granted) {
    const state = stateOf(provider, granted);
    const button = granted
        ? ''
        : `<button class="grant" data-domains="${(provider.domains ?? []).join(',')}" type="button">Grant access</button>`;
    return `
    <div class="provider">
      <span class="glyph ${state.connected ? 'glyph-on' : 'glyph-off'}">${state.connected ? '●' : '○'}</span>
      <span class="name">${provider.name}</span>
      <span class="tag ${state.tag}">${state.status}</span>
      ${button}
    </div>
    <div class="meta">${state.meta}</div>
  `;
}

async function gateStatus() {
    try {
        const response = await fetch(`${GATE}/sync/status`);
        return response.ok ? await response.json() : null;
    } catch {
        return null;
    }
}

function showEmpty(text) {
    element('providers').className = 'empty';
    element('providers').textContent = text;
}

// invariant: chrome.permissions.request needs the click's own user gesture, which is why the grant runs here and not in the worker
async function grant(button) {
    const domains = (button.dataset.domains ?? '').split(',').filter((domain) => domain !== '');
    if (await chrome.permissions.request({ origins: originsFor(domains) })) {
        await chrome.runtime.sendMessage({ type: 'refresh-providers' });
        await refresh();
    }
}

async function refresh() {
    const status = await gateStatus();
    if (status === null) {
        element('gate-status').innerHTML =
            '<span class="gate-bad">●</span> gate not reachable on 127.0.0.1:9000 — run `spwn gate start`';
        showEmpty('—');
        return;
    }
    element('gate-status').innerHTML = '<span class="gate-ok">●</span> gate connected';
    const list = status.providers ?? [];
    if (list.length === 0) {
        showEmpty(
            'no providers registered yet — install a gate tool that uses cookies (e.g. `spwn install spwn:x`)',
        );
        return;
    }
    const granted = await Promise.all(
        list.map(
            async (provider) =>
                await chrome.permissions.contains({ origins: originsFor(provider.domains) }),
        ),
    );
    element('providers').className = '';
    element('providers').innerHTML = list
        .map((provider, index) => renderProvider(provider, granted[index]))
        .join('');
    for (const button of document.querySelectorAll('button.grant')) {
        button.addEventListener('click', () => {
            void grant(button);
        });
    }
}

await refresh();
setInterval(() => {
    void refresh();
}, POLL_MS);
