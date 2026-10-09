import type { IncomingMessage, RequestListener, ServerResponse } from 'node:http';

import { readJson, RefusedError, send } from './http.ts';
import { ROUTES } from './routes.ts';
import type { Route } from './routes.ts';
import type { Sessions } from './sessions.ts';

function routed(request: IncomingMessage): undefined | { id: string; route: Route } {
    // Stryker disable next-line StringLiteral: node's http server sets `url` on every request it parses, and a split keeps one piece at least; the types allow neither
    const path = (request.url ?? '').split('?')[0] ?? '';
    for (const route of ROUTES) {
        const match = route.method === request.method ? route.path.exec(path) : null;
        if (match !== null) {
            // Stryker disable next-line StringLiteral: a route whose path captures no id never reads it
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

async function serve(
    sessions: Sessions,
    request: IncomingMessage,
    response: ServerResponse,
): Promise<void> {
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

/** The sidecar's HTTP face over `sessions`: each request routed to its action, a refusal answered with its status. */
export function sidecar(sessions: Sessions): RequestListener {
    return (request, response) => {
        void serve(sessions, request, response);
    };
}
