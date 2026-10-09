import { once } from 'node:events';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { createServer } from 'node:http';
import type { Server } from 'node:http';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { sidecar } from '../server.ts';
import { Sessions } from '../sessions.ts';

const PAGE = `<!doctype html>
<title>Site</title>
<body style="height: 10000px">
<input id="name" />
<button id="load">Load</button>
<div id="out"></div>
<script src="/app.js"></script>`;

/** The site's one script, which fetches the JSON when the button is clicked. */
export const APP = `document.getElementById('load').addEventListener('click', async () => {
    const out = document.getElementById('out');
    out.textContent = JSON.stringify(await (await fetch('/data.json')).json());
    out.dataset.ready = 'true';
});`;

const ROUTES: Record<string, { body: string; type: string }> = {
    '/': { body: PAGE, type: 'text/html' },
    '/app.js': { body: APP, type: 'application/javascript' },
    '/data.json': { body: JSON.stringify({ items: [1, 2] }), type: 'application/json' },
    '/note.txt': { body: 'a note', type: 'text/plain' },
};

/** The id a session's creation answered. */
export function idOf(body: unknown): string {
    const id: unknown =
        typeof body === 'object' && body !== null ? Reflect.get(body, 'id') : undefined;
    return typeof id === 'string' ? id : '';
}

async function listening(server: Server): Promise<string> {
    server.listen(0, '127.0.0.1');
    await once(server, 'listening');
    const address = server.address();
    return String(typeof address === 'object' && address !== null ? address.port : 0);
}

/** A sidecar over a real headless Chromium, the provider `site` holding the cookie `token=abc` for `localhost`, and the site it browses. */
export async function gate(credentials: Record<string, Record<string, string>> = {}) {
    const folder = mkdtempSync(join(tmpdir(), 'gate-browser-'));
    const providers = {
        site: {
            'cookies.json': JSON.stringify({ cookies: { token: 'abc' } }),
            '.domains': 'localhost\n',
        },
        ...credentials,
    };
    for (const [provider, files] of Object.entries(providers)) {
        mkdirSync(join(folder, provider));
        for (const [name, content] of Object.entries(files)) {
            writeFileSync(join(folder, provider, name), content);
        }
    }
    const site = createServer((request, response) => {
        const path = (request.url ?? '').split('?')[0] ?? '';
        if (path === '/bare') {
            response.end('{}');
            return;
        }
        if (path === '/hop') {
            response
                .writeHead(302, { 'Content-Type': 'application/json', Location: '/data.json' })
                .end();
            return;
        }
        const served = ROUTES[path];
        if (served === undefined) {
            response.writeHead(404, { 'Content-Type': 'text/plain' }).end('missing');
            return;
        }
        response.writeHead(200, { 'Content-Type': served.type }).end(served.body);
    });
    const sessions = new Sessions(folder);
    const server = createServer(sidecar(sessions));
    const origin = `http://127.0.0.1:${await listening(server)}`;
    return {
        async call(method: string, path: string, body?: unknown, raw?: string) {
            const response = await fetch(`${origin}${path}`, {
                body: raw ?? (body === undefined ? null : JSON.stringify(body)),
                method,
            });
            const text = await response.text();
            const json = (response.headers.get('content-type') ?? '').includes('json');
            return { body: json ? (JSON.parse(text) as unknown) : text, status: response.status };
        },
        sessions,
        site: `http://localhost:${await listening(site)}`,
        async [Symbol.asyncDispose]() {
            await sessions.shutdown();
            server.close();
            site.close();
            rmSync(folder, { force: true, recursive: true });
        },
    };
}
