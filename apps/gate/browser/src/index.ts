import { createServer } from 'node:http';
import type { IncomingMessage, ServerResponse } from 'node:http';

import { readJson, RefusedError, send } from './http.ts';
import { ROUTES } from './routes.ts';
import type { Route } from './routes.ts';
import { Sessions } from './sessions.ts';

const HOST = '127.0.0.1';
const PORT = Number.parseInt(process.env.GATE_BROWSER_PORT ?? '9001', 10);
const REAPER_INTERVAL_MS = 30 * 1000;

const sessions = new Sessions(process.env.CREDENTIALS_DIR ?? '/credentials');

function routed(request: IncomingMessage): undefined | { id: string; route: Route } {
    const path = (request.url ?? '').split('?')[0] ?? '';
    for (const route of ROUTES) {
        const match = route.method === request.method ? route.path.exec(path) : null;
        if (match !== null) {
            return { id: match.groups?.id ?? '', route };
        }
    }
    return undefined;
}

function refusal(error: unknown): { body: unknown; status: number } {
    const refused = error instanceof RefusedError ? error : null;
    return {
        body: {
            code: refused?.code ?? null,
            error: error instanceof Error ? error.message : String(error),
        },
        status: refused?.status ?? 500,
    };
}

async function serve(request: IncomingMessage, response: ServerResponse): Promise<void> {
    const found = routed(request);
    if (found === undefined) {
        send(response, 404, { error: 'no route' });
        return;
    }
    try {
        const body = found.route.method === 'POST' ? await readJson(request) : {};
        const reply = await found.route.action(sessions, found.id, body);
        send(response, reply.status, reply.body);
    } catch (error) {
        const { body, status } = refusal(error);
        send(response, status, body);
    }
}

// invariant: bound to 127.0.0.1 inside the gate container, so the trust boundary is the container: anything in it may drive any session, nothing on the host or in a world can reach it
const server = createServer((request, response) => {
    void serve(request, response);
});

const reaper = setInterval(() => {
    void sessions.reap(Date.now());
}, REAPER_INTERVAL_MS);

async function shutdown(): Promise<void> {
    clearInterval(reaper);
    server.close();
    await sessions.shutdown();
}

process.once('SIGTERM', () => {
    void shutdown();
});
process.once('SIGINT', () => {
    void shutdown();
});

server.listen(PORT, HOST, () => {
    process.stderr.write(`gate-browser listening on ${HOST}:${PORT}\n`);
});
