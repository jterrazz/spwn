#!/usr/bin/env node
// gate-browser — Playwright Chromium sidecar inside the spwn-gate
// container. Exposes an HTTP API on 127.0.0.1:9001 (gate-internal,
// never bound to the host) that catalog tools call to drive a
// browser session loaded with the user's session cookies.
//
// Why a sidecar and not in-process: catalog tools are Node (or
// Python in the future) — keeping the Playwright process separate
// lets them all share one warm Chromium pool, and isolates a tool
// crash from the gate's MCP routing.
//
// Auth model: the only trust boundary is "anything inside the gate
// container can drive any session". That's intentional — tools are
// vetted at install time (catalog), the gate itself is the trust
// root, and we don't want to invent a per-tool token scheme. The
// service is bound to 127.0.0.1 inside the container; nothing on
// the host or in worlds can reach it.
//
// Cookie loading: when a session is opened with `provider: "x"`,
// the sidecar reads /credentials/x/cookies.json (the same file the
// extension writes via /sync/x) and seeds the browser context.
//
// Lifecycle: sessions are reaped 5 min after their last use, hard
// cap 30 min total. The browser stays warm across sessions.

const http = require('http');
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { chromium } = require('playwright');

const PORT = parseInt(process.env.GATE_BROWSER_PORT || '9001', 10);
const HOST = '127.0.0.1';
const CREDENTIALS_DIR = process.env.CREDENTIALS_DIR || '/credentials';
const IDLE_TTL_MS = 5 * 60 * 1000;
const HARD_TTL_MS = 30 * 60 * 1000;
const REAPER_INTERVAL_MS = 30 * 1000;
const DEFAULT_OP_TIMEOUT_MS = 20000;

const USER_AGENT =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 ' +
  '(KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36';

// One Chromium for the whole sidecar. Contexts (= isolated cookie
// jars) are created per session.
let browser = null;
async function ensureBrowser() {
  if (browser && browser.isConnected()) return browser;
  browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-dev-shm-usage'],
  });
  browser.on('disconnected', () => { browser = null; });
  return browser;
}

// Map<id, Session>
const sessions = new Map();

class Session {
  constructor(id, provider, ctx, page) {
    this.id = id;
    this.provider = provider;
    this.ctx = ctx;
    this.page = page;
    this.createdAt = Date.now();
    this.lastUsedAt = Date.now();
    // Captured XHR responses, ring-style. Tools poll this to harvest
    // GraphQL bodies that the page emitted while doing something else.
    this.captured = [];
    page.on('response', async (resp) => {
      const url = resp.url();
      // Skip noise — we only care about API/XHR responses, not
      // images/css/fonts/html.
      const ct = (resp.headers()['content-type'] || '').toLowerCase();
      if (!ct.includes('json') && !ct.includes('javascript')) return;
      try {
        // Don't await body here — store a thunk so consumers pay
        // the cost only for responses they actually want.
        this.captured.push({
          url,
          status: resp.status(),
          method: resp.request().method(),
          ts: Date.now(),
          _resp: resp,
        });
        if (this.captured.length > 500) this.captured.splice(0, 100);
      } catch (_) { /* page closed mid-response */ }
    });
  }
  touch() { this.lastUsedAt = Date.now(); }
  isExpired(now) {
    return now - this.lastUsedAt > IDLE_TTL_MS || now - this.createdAt > HARD_TTL_MS;
  }
  async close() {
    try { await this.ctx.close(); } catch (_) {}
    sessions.delete(this.id);
  }
}

function loadProviderCookies(provider) {
  const file = path.join(CREDENTIALS_DIR, provider, 'cookies.json');
  if (!fs.existsSync(file)) {
    const err = new Error(`no cookies for provider "${provider}" — run the cookie-sync extension`);
    err.code = 'NO_COOKIES';
    throw err;
  }
  const raw = JSON.parse(fs.readFileSync(file, 'utf8'));
  const cookies = raw.cookies || {};
  // Build Playwright cookies. We don't know the original domain shape
  // (extension writes name+value only), so we fan out to common
  // shapes: bare domain + .domain on the provider's primary host.
  // For X this means .x.com; for LinkedIn .linkedin.com; etc.
  // Provider→domains mapping is owned by the gate Go side (cookie
  // provider config) — but we can introspect /credentials/<p>/.domains
  // if the gate writes it, or fall back to a per-provider hint via env.
  const domainsHintPath = path.join(CREDENTIALS_DIR, provider, '.domains');
  let domains;
  if (fs.existsSync(domainsHintPath)) {
    domains = fs.readFileSync(domainsHintPath, 'utf8')
      .split('\n').map((s) => s.trim()).filter(Boolean);
  } else {
    // Fallback: a single bare-domain guess. Better than nothing.
    domains = [provider + '.com'];
  }
  const out = [];
  for (const d of domains) {
    const dotted = d.startsWith('.') ? d : '.' + d;
    for (const [name, value] of Object.entries(cookies)) {
      out.push({
        name, value, domain: dotted, path: '/',
        secure: true, httpOnly: false, sameSite: 'Lax',
      });
    }
  }
  return out;
}

// ── HTTP helpers ─────────────────────────────────────────────────

async function readJson(req) {
  const chunks = [];
  for await (const c of req) chunks.push(c);
  if (!chunks.length) return {};
  try { return JSON.parse(Buffer.concat(chunks).toString('utf8')); }
  catch (e) { throw httpErr(400, `bad json: ${e.message}`); }
}

function httpErr(code, msg) {
  const e = new Error(msg);
  e.statusCode = code;
  return e;
}

function send(res, code, body) {
  const data = typeof body === 'string' ? body : JSON.stringify(body);
  res.writeHead(code, {
    'Content-Type': typeof body === 'string' ? 'text/plain' : 'application/json',
    'Content-Length': Buffer.byteLength(data),
  });
  res.end(data);
}

function getSession(id) {
  const s = sessions.get(id);
  if (!s) throw httpErr(404, `unknown session "${id}"`);
  s.touch();
  return s;
}

// ── Routes ───────────────────────────────────────────────────────

const routes = [
  // POST /sessions { provider } → { id }
  ['POST', /^\/sessions$/, async (req, res, _m, body) => {
    if (!body.provider) throw httpErr(400, 'provider is required');
    const cookies = loadProviderCookies(body.provider);
    const br = await ensureBrowser();
    const ctx = await br.newContext({ userAgent: USER_AGENT, viewport: { width: 1280, height: 900 } });
    await ctx.addCookies(cookies);
    const page = await ctx.newPage();
    const id = crypto.randomUUID();
    const s = new Session(id, body.provider, ctx, page);
    sessions.set(id, s);
    send(res, 201, { id, provider: body.provider, expires_at: new Date(s.createdAt + HARD_TTL_MS).toISOString() });
  }],

  // DELETE /sessions/:id
  ['DELETE', /^\/sessions\/([^/]+)$/, async (req, res, m) => {
    const s = sessions.get(m[1]);
    if (s) await s.close();
    send(res, 204, '');
  }],

  // POST /sessions/:id/goto { url, wait_until }
  ['POST', /^\/sessions\/([^/]+)\/goto$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    if (!body.url) throw httpErr(400, 'url is required');
    const resp = await s.page.goto(body.url, {
      waitUntil: body.wait_until || 'domcontentloaded',
      timeout: body.timeout_ms || DEFAULT_OP_TIMEOUT_MS,
    });
    send(res, 200, { ok: true, status: resp ? resp.status() : null, url: s.page.url() });
  }],

  // POST /sessions/:id/click { selector, timeout_ms }
  ['POST', /^\/sessions\/([^/]+)\/click$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    if (!body.selector) throw httpErr(400, 'selector is required');
    await s.page.click(body.selector, { timeout: body.timeout_ms || DEFAULT_OP_TIMEOUT_MS });
    send(res, 200, { ok: true });
  }],

  // POST /sessions/:id/type { selector, text, timeout_ms }
  ['POST', /^\/sessions\/([^/]+)\/type$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    if (!body.selector || body.text == null) throw httpErr(400, 'selector and text required');
    await s.page.click(body.selector, { timeout: body.timeout_ms || DEFAULT_OP_TIMEOUT_MS });
    await s.page.keyboard.type(body.text);
    send(res, 200, { ok: true });
  }],

  // POST /sessions/:id/scroll { delta_y, count, wait_ms }
  ['POST', /^\/sessions\/([^/]+)\/scroll$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    const dy = body.delta_y || 4000;
    const count = body.count || 1;
    const wait = body.wait_ms || 1500;
    for (let i = 0; i < count; i++) {
      await s.page.mouse.wheel(0, dy);
      await s.page.waitForTimeout(wait);
    }
    send(res, 200, { ok: true });
  }],

  // POST /sessions/:id/wait-selector { selector, state, timeout_ms }
  ['POST', /^\/sessions\/([^/]+)\/wait-selector$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    if (!body.selector) throw httpErr(400, 'selector required');
    await s.page.waitForSelector(body.selector, {
      state: body.state || 'visible',
      timeout: body.timeout_ms || DEFAULT_OP_TIMEOUT_MS,
    });
    send(res, 200, { ok: true });
  }],

  // POST /sessions/:id/wait-response { url_pattern, method?, timeout_ms, allow_non_json? }
  // Skips responses whose content-type isn't JSON by default —
  // keeps page-load JS bundles + images from accidentally matching
  // a substring-based pattern. Set allow_non_json:true to opt out.
  ['POST', /^\/sessions\/([^/]+)\/wait-response$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    if (!body.url_pattern) throw httpErr(400, 'url_pattern required');
    const re = new RegExp(body.url_pattern);
    const want = (body.method || '').toUpperCase();
    const allowNonJson = !!body.allow_non_json;
    const resp = await s.page.waitForResponse(
      (r) => {
        if (!re.test(r.url())) return false;
        if (want && r.request().method() !== want) return false;
        if (!allowNonJson) {
          const ct = (r.headers()['content-type'] || '').toLowerCase();
          if (!ct.includes('json')) return false;
        }
        return true;
      },
      { timeout: body.timeout_ms || DEFAULT_OP_TIMEOUT_MS },
    );
    let parsed = null;
    try { parsed = await resp.json(); } catch (_) { try { parsed = await resp.text(); } catch (_) { parsed = null; } }
    send(res, 200, { url: resp.url(), status: resp.status(), body: parsed });
  }],

  // POST /sessions/:id/eval { script } — runs script in page ctx
  // and returns its return value (must be JSON-serializable).
  ['POST', /^\/sessions\/([^/]+)\/eval$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    if (!body.script) throw httpErr(400, 'script required');
    const result = await s.page.evaluate(body.script);
    send(res, 200, { result });
  }],

  // POST /sessions/:id/captured-responses { url_pattern, since_ts? }
  // Tools call this after a navigation/scroll to harvest XHRs they
  // didn't explicitly wait for. Body is fetched lazily here.
  ['POST', /^\/sessions\/([^/]+)\/captured-responses$/, async (req, res, m, body) => {
    const s = getSession(m[1]);
    const re = body.url_pattern ? new RegExp(body.url_pattern) : null;
    const since = body.since_ts || 0;
    const out = [];
    for (const c of s.captured) {
      if (c.ts < since) continue;
      if (re && !re.test(c.url)) continue;
      let parsed = null;
      try { parsed = await c._resp.json(); } catch (_) { try { parsed = await c._resp.text(); } catch (_) {} }
      out.push({ url: c.url, status: c.status, method: c.method, ts: c.ts, body: parsed });
    }
    send(res, 200, { responses: out });
  }],

  // GET /healthz
  ['GET', /^\/healthz$/, async (req, res) => {
    send(res, 200, { ok: true, sessions: sessions.size, browser_connected: !!(browser && browser.isConnected()) });
  }],
];

const server = http.createServer(async (req, res) => {
  // 127.0.0.1 binding is the trust boundary; no auth header needed.
  for (const [method, pattern, handler] of routes) {
    if (req.method !== method) continue;
    const m = req.url.split('?')[0].match(pattern);
    if (!m) continue;
    try {
      const body = (method === 'POST') ? await readJson(req) : null;
      await handler(req, res, m, body);
    } catch (e) {
      const code = e.statusCode || 500;
      send(res, code, { error: e.message, code: e.code || null });
    }
    return;
  }
  send(res, 404, { error: 'no route' });
});

// Reap idle sessions.
setInterval(async () => {
  const now = Date.now();
  for (const s of sessions.values()) {
    if (s.isExpired(now)) {
      try { await s.close(); } catch (_) {}
    }
  }
}, REAPER_INTERVAL_MS);

server.listen(PORT, HOST, () => {
  process.stderr.write(`gate-browser listening on ${HOST}:${PORT}\n`);
});

// Clean shutdown.
function bye() {
  for (const s of sessions.values()) s.close().catch(() => {});
  if (browser) browser.close().catch(() => {});
  process.exit(0);
}
process.on('SIGTERM', bye);
process.on('SIGINT', bye);
