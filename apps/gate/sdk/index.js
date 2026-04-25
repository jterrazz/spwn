// @spwn/gate-tool — Node SDK for catalog tools that plug into the
// spwn gate. A tool's index.js looks like:
//
//   const { Tool, openSession } = require('@spwn/gate-tool');
//
//   const tool = new Tool({ name: 'x' });
//
//   tool.method('fetch-favorites', {
//     description: 'Fetch the user's bookmarks.',
//     schema: { type: 'object', properties: { limit: { type: 'integer' } } },
//     async handler({ args }) {
//       const session = await openSession('x');
//       try {
//         await session.goto('https://x.com/i/bookmarks');
//         const r = await session.waitResponse('/Bookmarks');
//         return { items: r.body, count: ... };
//       } finally { await session.end(); }
//     },
//   });
//
//   tool.run();   // dispatches based on argv
//
// Lifecycle:
//   - `node index.js mcp-manifest`  → prints JSON tool list (used by gate)
//   - `node index.js mcp-serve`     → HTTP MCP server on $GATE_TOOL_PORT
//   - `node index.js <method> --k v` → CLI invocation, JSON to stdout
//                                       (used by host scripts like publish.sh)

const http = require('http');

// ── Browser-session client ──────────────────────────────────────

const BROWSER_URL = process.env.GATE_BROWSER_URL || 'http://127.0.0.1:9001';

class Session {
  constructor(id, baseUrl) { this.id = id; this.baseUrl = baseUrl; this._closed = false; }
  goto(url, opts = {}) { return this._post('goto', { url, ...opts }); }
  waitResponse(urlPattern, opts = {}) { return this._post('wait-response', { url_pattern: urlPattern, ...opts }); }
  click(selector, opts = {}) { return this._post('click', { selector, ...opts }); }
  type(selector, text, opts = {}) { return this._post('type', { selector, text, ...opts }); }
  scroll(opts = {}) { return this._post('scroll', opts); }
  waitSelector(selector, opts = {}) { return this._post('wait-selector', { selector, ...opts }); }
  eval(script) { return this._post('eval', { script }); }
  capturedResponses(opts = {}) { return this._post('captured-responses', opts); }
  async end() {
    if (this._closed) return;
    this._closed = true;
    return httpReq('DELETE', `${this.baseUrl}/sessions/${this.id}`);
  }
  _post(p, body) { return httpReq('POST', `${this.baseUrl}/sessions/${this.id}/${p}`, body); }
}

async function openSession(provider) {
  if (!provider) throw new Error('openSession: provider name required');
  const r = await httpReq('POST', `${BROWSER_URL}/sessions`, { provider });
  return new Session(r.id, BROWSER_URL);
}

function httpReq(method, url, body) {
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const opts = {
      method,
      hostname: u.hostname,
      port: u.port,
      path: u.pathname + (u.search || ''),
      headers: { 'content-type': 'application/json' },
    };
    const req = http.request(opts, (res) => {
      const chunks = [];
      res.on('data', (c) => chunks.push(c));
      res.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8');
        if (res.statusCode >= 400) {
          let msg = text;
          try { msg = JSON.parse(text).error || text; } catch (_) {}
          return reject(new Error(`${method} ${url} → ${res.statusCode}: ${msg}`));
        }
        if (!text) return resolve(null);
        try { resolve(JSON.parse(text)); } catch (_) { resolve(text); }
      });
    });
    req.on('error', reject);
    if (body !== undefined && body !== null) req.write(JSON.stringify(body));
    req.end();
  });
}

// ── Tool harness ────────────────────────────────────────────────

class Tool {
  constructor(opts = {}) {
    this.name = opts.name || process.env.GATE_TOOL_NAME;
    if (!this.name) throw new Error('Tool: name required (constructor opts or GATE_TOOL_NAME env)');
    this.title = opts.title || this.name;
    this.version = opts.version || '0.1.0';
    this.methods = new Map();
  }

  // Register a method. handler({ args }) returns any JSON-serializable
  // value. Schema is JSON-schema for the args (used both for MCP and
  // for CLI flag parsing — booleans become flags, strings/numbers
  // expect a value).
  method(name, { description, schema, handler }) {
    if (this.methods.has(name)) throw new Error(`method ${name} already registered`);
    this.methods.set(name, { name, description: description || '', schema: schema || { type: 'object' }, handler });
    return this;
  }

  // Dispatch based on argv. See top-of-file comment for the three modes.
  async run(argv = process.argv.slice(2)) {
    const [sub, ...rest] = argv;
    try {
      if (sub === 'mcp-manifest') return this._dumpManifest();
      if (sub === 'mcp-serve') return this._serve();
      if (this.methods.has(sub)) return this._direct(sub, rest);
      this._usage();
      process.exit(2);
    } catch (e) {
      process.stderr.write(`tool error: ${e && e.stack ? e.stack : String(e)}\n`);
      process.exit(4);
    }
  }

  _usage() {
    let s = `usage: ${this.name} <subcommand>\n\nsubcommands:\n`;
    s += `  mcp-manifest                     print the JSON tool list (gate uses this)\n`;
    s += `  mcp-serve                        run the MCP HTTP server (gate spawns this)\n`;
    for (const m of this.methods.values()) {
      s += `  ${m.name.padEnd(32)} ${m.description}\n`;
    }
    process.stderr.write(s);
  }

  _dumpManifest() {
    const out = {
      name: this.name,
      title: this.title,
      version: this.version,
      methods: [...this.methods.values()].map((m) => ({
        name: m.name, description: m.description, inputSchema: m.schema,
      })),
    };
    process.stdout.write(JSON.stringify(out, null, 2) + '\n');
  }

  async _direct(name, argv) {
    const args = parseFlags(argv);
    const m = this.methods.get(name);
    const result = await m.handler({ args });
    process.stdout.write(JSON.stringify(result));
    process.stdout.write('\n');
  }

  async _serve() {
    const port = parseInt(process.env.GATE_TOOL_PORT || '0', 10);
    if (!port) throw new Error('mcp-serve: GATE_TOOL_PORT not set');
    const server = http.createServer(async (req, res) => {
      if (req.method === 'GET' && req.url.split('?')[0] === '/healthz') {
        return sendJson(res, 200, { ok: true, name: this.name, methods: this.methods.size });
      }
      // The gate strips the /mcp/<name>/ prefix before forwarding,
      // so what arrives here is /mcp/ (or /). Accept both.
      if (req.method === 'POST') {
        try {
          const body = await readJson(req);
          await this._dispatchJsonRpc(body, res);
        } catch (e) {
          sendJson(res, 400, { jsonrpc: '2.0', error: { code: -32700, message: e.message } });
        }
        return;
      }
      sendJson(res, 405, { error: 'POST only' });
    });
    server.listen(port, '127.0.0.1', () => {
      process.stderr.write(`tool ${this.name} MCP server listening on 127.0.0.1:${port}\n`);
    });
    process.on('SIGTERM', () => process.exit(0));
    process.on('SIGINT', () => process.exit(0));
  }

  async _dispatchJsonRpc(req, res) {
    const id = req.id;
    if (req.jsonrpc !== '2.0') return sendJson(res, 200, rpcErr(id, -32600, 'jsonrpc must be 2.0'));

    if (req.method === 'initialize') {
      return sendJson(res, 200, rpcOk(id, {
        protocolVersion: '2025-06-18',
        capabilities: { tools: {} },
        serverInfo: { name: this.name, title: this.title, version: this.version },
      }));
    }
    if (req.method === 'notifications/initialized') {
      return sendJson(res, 202, '');
    }
    if (req.method === 'ping') return sendJson(res, 200, rpcOk(id, {}));

    if (req.method === 'tools/list') {
      const tools = [...this.methods.values()].map((m) => ({
        name: m.name, description: m.description, inputSchema: m.schema,
      }));
      return sendJson(res, 200, rpcOk(id, { tools }));
    }

    if (req.method === 'tools/call') {
      const p = req.params || {};
      const m = this.methods.get(p.name);
      if (!m) return sendJson(res, 200, rpcErr(id, -32601, `tool ${p.name} not found`));
      try {
        const result = await m.handler({ args: p.arguments || {} });
        const text = typeof result === 'string' ? result : JSON.stringify(result);
        return sendJson(res, 200, rpcOk(id, { content: [{ type: 'text', text }] }));
      } catch (e) {
        return sendJson(res, 200, rpcOk(id, { isError: true, content: [{ type: 'text', text: e.message }] }));
      }
    }

    return sendJson(res, 200, rpcErr(id, -32601, `method ${req.method} not implemented`));
  }
}

// ── HTTP + RPC helpers ──────────────────────────────────────────

function rpcOk(id, result) { return { jsonrpc: '2.0', id, result }; }
function rpcErr(id, code, message) { return { jsonrpc: '2.0', id, error: { code, message } }; }

function sendJson(res, code, body) {
  if (typeof body === 'string') {
    res.writeHead(code).end(body);
    return;
  }
  const data = JSON.stringify(body);
  res.writeHead(code, { 'content-type': 'application/json', 'content-length': Buffer.byteLength(data) }).end(data);
}

async function readJson(req) {
  const chunks = [];
  for await (const c of req) chunks.push(c);
  if (!chunks.length) return {};
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}

// Lightweight CLI flag parser. Supports `--key value` and bare
// boolean flags `--flag` (yields true). No `=`-style or short flags
// — we keep it boring to avoid surprising tool authors.
function parseFlags(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (!a.startsWith('--')) continue;
    const k = a.slice(2);
    const next = argv[i + 1];
    if (next === undefined || next.startsWith('--')) {
      out[k] = true;
    } else {
      out[k] = next;
      i++;
    }
  }
  return out;
}

module.exports = { Tool, openSession, Session };
