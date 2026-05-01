import http from 'node:http';
import { createRequire } from 'node:module';
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest';

const require = createRequire(import.meta.url);

const sdkPath = require.resolve('./index.js');

function loadSdk(browserUrl) {
  delete require.cache[sdkPath];
  if (browserUrl) {
    process.env.GATE_BROWSER_URL = browserUrl;
  } else {
    delete process.env.GATE_BROWSER_URL;
  }
  return require('./index.js');
}

async function captureStdout(fn) {
  const original = process.stdout.write;
  let out = '';
  process.stdout.write = vi.fn((chunk, encoding, cb) => {
    out += Buffer.isBuffer(chunk) ? chunk.toString('utf8') : String(chunk);
    if (typeof encoding === 'function') encoding();
    if (typeof cb === 'function') cb();
    return true;
  });
  try {
    await fn();
    return out;
  } finally {
    process.stdout.write = original;
  }
}

function createJsonRpcResponse() {
  return {
    status: 0,
    headers: undefined,
    body: '',
    writeHead(code, headers) {
      this.status = code;
      this.headers = headers;
      return this;
    },
    end(body = '') {
      this.body = body;
      return this;
    },
    json() {
      return JSON.parse(this.body);
    },
  };
}

function createBrowserServer() {
  const requests = [];
  const server = http.createServer((req, res) => {
    const chunks = [];
    req.on('data', (chunk) => chunks.push(chunk));
    req.on('end', () => {
      const raw = Buffer.concat(chunks).toString('utf8');
      const body = raw ? JSON.parse(raw) : null;
      requests.push({ method: req.method, url: req.url, body });

      if (req.method === 'POST' && req.url === '/sessions') {
        res.writeHead(201, { 'content-type': 'application/json' });
        res.end(JSON.stringify({ id: 'session-1' }));
        return;
      }
      if (req.method === 'DELETE' && req.url === '/sessions/session-1') {
        res.writeHead(204).end();
        return;
      }

      res.writeHead(200, { 'content-type': 'application/json' });
      res.end(JSON.stringify({ ok: true, route: req.url, body }));
    });
  });

  return new Promise((resolve, reject) => {
    server.on('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      resolve({
        baseUrl: `http://127.0.0.1:${address.port}`,
        requests,
        close: () => new Promise((done) => server.close(done)),
      });
    });
  });
}

describe('@spwn/gate-tool Tool harness', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    delete process.env.GATE_BROWSER_URL;
  });

  it('prints a manifest with registered tool schemas', async () => {
    const { Tool } = loadSdk();
    const tool = new Tool({ name: 'catalog-demo', title: 'Catalog Demo', version: '1.2.3' });
    tool.method('search', {
      description: 'Search the provider',
      schema: { type: 'object', properties: { query: { type: 'string' } } },
      handler: async () => ({ ok: true }),
    });

    const out = await captureStdout(() => tool.run(['mcp-manifest']));

    expect(JSON.parse(out)).toEqual({
      name: 'catalog-demo',
      title: 'Catalog Demo',
      version: '1.2.3',
      methods: [
        {
          name: 'search',
          description: 'Search the provider',
          inputSchema: { type: 'object', properties: { query: { type: 'string' } } },
        },
      ],
    });
  });

  it('runs direct CLI methods with parsed flags', async () => {
    const { Tool } = loadSdk();
    const tool = new Tool({ name: 'catalog-demo' });
    tool.method('publish', {
      description: 'Publish a post',
      schema: { type: 'object' },
      handler: async ({ args }) => ({ args }),
    });

    const out = await captureStdout(() => tool.run(['publish', '--text', 'hello', '--dry-run']));

    expect(JSON.parse(out)).toEqual({ args: { text: 'hello', 'dry-run': true } });
  });

  it('serves MCP initialize, tools/list, tools/call, and tool-not-found contracts', async () => {
    const { Tool } = loadSdk();
    const tool = new Tool({ name: 'catalog-demo', title: 'Catalog Demo' });
    tool.method('count', {
      description: 'Count items',
      schema: { type: 'object', properties: { limit: { type: 'number' } } },
      handler: async ({ args }) => ({ count: Number(args.limit ?? 0) }),
    });

    const initialize = createJsonRpcResponse();
    await tool._dispatchJsonRpc({ jsonrpc: '2.0', id: 1, method: 'initialize' }, initialize);
    expect(initialize.status).toBe(200);
    expect(initialize.json().result.serverInfo).toMatchObject({ name: 'catalog-demo', title: 'Catalog Demo' });

    const list = createJsonRpcResponse();
    await tool._dispatchJsonRpc({ jsonrpc: '2.0', id: 2, method: 'tools/list' }, list);
    expect(list.json().result.tools).toHaveLength(1);
    expect(list.json().result.tools[0]).toMatchObject({ name: 'count', description: 'Count items' });

    const call = createJsonRpcResponse();
    await tool._dispatchJsonRpc(
      { jsonrpc: '2.0', id: 3, method: 'tools/call', params: { name: 'count', arguments: { limit: 4 } } },
      call,
    );
    expect(call.json().result.content).toEqual([{ type: 'text', text: '{"count":4}' }]);

    const missing = createJsonRpcResponse();
    await tool._dispatchJsonRpc(
      { jsonrpc: '2.0', id: 4, method: 'tools/call', params: { name: 'missing' } },
      missing,
    );
    expect(missing.json().error).toMatchObject({ code: -32601, message: 'tool missing not found' });
  });
});

describe('@spwn/gate-tool browser session client', () => {
  let browserServer;

  beforeAll(async () => {
    browserServer = await createBrowserServer();
  });

  afterAll(async () => {
    await browserServer.close();
  });

  afterEach(() => {
    browserServer.requests.length = 0;
    delete process.env.GATE_BROWSER_URL;
  });

  it('opens a session and calls the browser sidecar over HTTP routes', async () => {
    const { openSession } = loadSdk(browserServer.baseUrl);

    const session = await openSession('x');
    await session.goto('https://x.com/home', { wait_until: 'networkidle' });
    await session.waitResponse('/Bookmarks', { method: 'POST' });
    await session.click('[data-testid="tweetButton"]');
    await session.type('textarea', 'hello');
    await session.scroll({ count: 2, wait_ms: 0 });
    await session.waitSelector('[data-ready="true"]');
    await session.eval('document.title');
    await session.capturedResponses({ url_pattern: 'GraphQL' });
    await session.end();
    await session.end();

    expect(browserServer.requests).toEqual([
      { method: 'POST', url: '/sessions', body: { provider: 'x' } },
      {
        method: 'POST',
        url: '/sessions/session-1/goto',
        body: { url: 'https://x.com/home', wait_until: 'networkidle' },
      },
      {
        method: 'POST',
        url: '/sessions/session-1/wait-response',
        body: { url_pattern: '/Bookmarks', method: 'POST' },
      },
      { method: 'POST', url: '/sessions/session-1/click', body: { selector: '[data-testid="tweetButton"]' } },
      { method: 'POST', url: '/sessions/session-1/type', body: { selector: 'textarea', text: 'hello' } },
      { method: 'POST', url: '/sessions/session-1/scroll', body: { count: 2, wait_ms: 0 } },
      {
        method: 'POST',
        url: '/sessions/session-1/wait-selector',
        body: { selector: '[data-ready="true"]' },
      },
      { method: 'POST', url: '/sessions/session-1/eval', body: { script: 'document.title' } },
      {
        method: 'POST',
        url: '/sessions/session-1/captured-responses',
        body: { url_pattern: 'GraphQL' },
      },
      { method: 'DELETE', url: '/sessions/session-1', body: null },
    ]);
  });

  it('surfaces sidecar JSON errors with method, URL, status, and message', async () => {
    const server = http.createServer((_req, res) => {
      res.writeHead(503, { 'content-type': 'application/json' });
      res.end(JSON.stringify({ error: 'sidecar unavailable' }));
    });
    await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
    const address = server.address();
    const { openSession } = loadSdk(`http://127.0.0.1:${address.port}`);

    await expect(openSession('x')).rejects.toThrow(/POST http:\/\/127\.0\.0\.1:\d+\/sessions .+ 503: sidecar unavailable/);
    await new Promise((resolve) => server.close(resolve));
  });
});
