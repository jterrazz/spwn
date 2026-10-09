// outside: Chrome MV3 reads a site's cookies only once the user grants its host, which takes a gesture — so the popup
// grants, and this worker syncs every granted provider's cookies to the gate on 127.0.0.1, which keeps only the cookies
// its provider allowlist names.

const GATE = 'http://127.0.0.1:9000';
const REFRESH_MS = 5 * 60 * 1000;

let providers = [];

async function loadProviders() {
    try {
        const response = await fetch(`${GATE}/sync/providers`);
        if (response.ok) {
            providers = await response.json();
        }
    } catch {
        // value: with the gate down, the last providers it listed stay in use
    }
}

function providerOf(host) {
    const bare = host.replace(/^www\./u, '').replace(/^\./u, '');
    return providers.find((provider) =>
        (provider.domains ?? []).some((domain) => bare === domain || bare.endsWith(`.${domain}`)),
    );
}

async function pushCookies(provider, url) {
    const found = await Promise.all(
        provider.cookies.map(async (name) => await chrome.cookies.get({ name, url })),
    );
    const cookies = Object.fromEntries(
        found.filter((cookie) => cookie !== null).map((cookie) => [cookie.name, cookie.value]),
    );
    if (Object.keys(cookies).length === 0) {
        return;
    }
    try {
        await fetch(`${GATE}/sync/${provider.name}`, {
            body: JSON.stringify({ captured: new Date().toISOString(), cookies }),
            headers: { 'Content-Type': 'application/json' },
            method: 'POST',
        });
    } catch {
        // value: with the gate down, the next tab load or cookie change syncs again
    }
}

function hostOf(url) {
    try {
        return new URL(url).hostname;
    } catch {
        return null;
    }
}

chrome.tabs.onUpdated.addListener((_tabId, info, tab) => {
    const host = info.status === 'complete' && typeof tab.url === 'string' ? hostOf(tab.url) : null;
    const provider = host === null ? undefined : providerOf(host);
    if (provider !== undefined) {
        void pushCookies(provider, tab.url);
    }
});

// outside: a site rotates a session cookie mid-session (X its ct0 on every action, LinkedIn its JSESSIONID), so a change syncs at once
chrome.cookies.onChanged.addListener((change) => {
    const { cookie } = change;
    const provider = change.removed ? undefined : providerOf(cookie.domain);
    if (provider?.cookies.includes(cookie.name) === true) {
        void pushCookies(
            provider,
            `${cookie.secure ? 'https' : 'http'}://${cookie.domain.replace(/^\./u, '')}${cookie.path ?? '/'}`,
        );
    }
});

chrome.runtime.onMessage.addListener((message, _sender, respond) => {
    if (message.type === 'list-providers') {
        respond({ providers });
        return true;
    }
    if (message.type === 'refresh-providers') {
        void (async () => {
            await loadProviders();
            respond({ providers });
        })();
        return true;
    }
    return false;
});

await loadProviders();
setInterval(() => {
    void loadProviders();
}, REFRESH_MS);
