import type { Response } from 'playwright';

import { RefusedError } from './http.ts';
import { HARD_TTL_MS } from './session.ts';
import type { Sessions } from './sessions.ts';

type Body = Record<string, unknown>;
type Reply = { body: unknown; status: number };
export type Route = {
    action: (sessions: Sessions, id: string, body: Body) => Promise<Reply>;
    method: 'DELETE' | 'GET' | 'POST';
    path: RegExp;
};

const OPERATION_TIMEOUT_MS = 20_000;
const SCROLL_DELTA = 4000;
const SCROLL_WAIT_MS = 1500;

const text = (body: Body, key: string): string => {
    const value = body[key];
    return typeof value === 'string' ? value : '';
};
const number = (body: Body, key: string, fallback: number): number => {
    const value = body[key];
    return typeof value === 'number' && value !== 0 ? value : fallback;
};
const required = (body: Body, key: string): string => {
    const value = text(body, key);
    if (value === '') {
        throw new RefusedError(400, `${key} is required`);
    }
    return value;
};
const done: Reply = { body: { ok: true }, status: 200 };

const LOAD_STATES = ['commit', 'domcontentloaded', 'load', 'networkidle'] as const;
const SELECTOR_STATES = ['attached', 'detached', 'hidden', 'visible'] as const;

function oneOf<T extends string>(values: readonly T[], value: string, fallback: T): T {
    return values.find((candidate) => candidate === value) ?? fallback;
}

async function bodyOf(response: Response): Promise<unknown> {
    try {
        const json: unknown = await response.json();
        return json;
    } catch {
        return await response.text().catch(() => null);
    }
}

async function waitResponse(sessions: Sessions, id: string, body: Body): Promise<Reply> {
    const pattern = new RegExp(required(body, 'url_pattern'), 'u');
    const method = text(body, 'method').toUpperCase();
    const anyType = body.allow_non_json === true;
    const response = await sessions
        .get(id)
        .page.waitForResponse(
            (candidate) =>
                pattern.test(candidate.url()) &&
                (method === '' || candidate.request().method() === method) &&
                (anyType ||
                    (candidate.headers()['content-type'] ?? '').toLowerCase().includes('json')),
            { timeout: number(body, 'timeout_ms', OPERATION_TIMEOUT_MS) },
        );
    return {
        body: { body: await bodyOf(response), status: response.status(), url: response.url() },
        status: 200,
    };
}

async function captured(sessions: Sessions, id: string, body: Body): Promise<Reply> {
    const pattern =
        text(body, 'url_pattern') === '' ? null : new RegExp(text(body, 'url_pattern'), 'u');
    const since = number(body, 'since_ts', 0);
    const kept = sessions
        .get(id)
        .captured.filter((entry) => entry.ts >= since && (pattern?.test(entry.url) ?? true));
    const responses = await Promise.all(
        kept.map(async (entry) => ({
            body: await bodyOf(entry.response),
            method: entry.method,
            status: entry.status,
            ts: entry.ts,
            url: entry.url,
        })),
    );
    return { body: { responses }, status: 200 };
}

async function scroll(sessions: Sessions, id: string, body: Body): Promise<Reply> {
    const { page } = sessions.get(id);
    for (let step = 0; step < number(body, 'count', 1); step += 1) {
        await page.mouse.wheel(0, number(body, 'delta_y', SCROLL_DELTA));
        await page.waitForTimeout(number(body, 'wait_ms', SCROLL_WAIT_MS));
    }
    return done;
}

/** The sidecar's routes; `:id` is the first capture of each path. */
export const ROUTES: Route[] = [
    {
        action: async (sessions, _id, body) => {
            const provider = required(body, 'provider');
            const { id, session } = await sessions.create(provider);
            return {
                body: {
                    expires_at: new Date(session.createdAt + HARD_TTL_MS).toISOString(),
                    id,
                    provider,
                },
                status: 201,
            };
        },
        method: 'POST',
        path: /^\/sessions$/u,
    },
    {
        action: async (sessions, id) => {
            await sessions.end(id);
            return { body: '', status: 204 };
        },
        method: 'DELETE',
        path: /^\/sessions\/(?<id>[^/]+)$/u,
    },
    {
        action: async (sessions, id, body) => {
            const { page } = sessions.get(id);
            const response = await page.goto(required(body, 'url'), {
                timeout: number(body, 'timeout_ms', OPERATION_TIMEOUT_MS),
                waitUntil: oneOf(LOAD_STATES, text(body, 'wait_until'), 'domcontentloaded'),
            });
            return {
                body: { ok: true, status: response?.status() ?? null, url: page.url() },
                status: 200,
            };
        },
        method: 'POST',
        path: /^\/sessions\/(?<id>[^/]+)\/goto$/u,
    },
    {
        action: async (sessions, id, body) => {
            await sessions.get(id).page.click(required(body, 'selector'), {
                timeout: number(body, 'timeout_ms', OPERATION_TIMEOUT_MS),
            });
            return done;
        },
        method: 'POST',
        path: /^\/sessions\/(?<id>[^/]+)\/click$/u,
    },
    {
        action: async (sessions, id, body) => {
            const { page } = sessions.get(id);
            await page.click(required(body, 'selector'), {
                timeout: number(body, 'timeout_ms', OPERATION_TIMEOUT_MS),
            });
            await page.keyboard.type(text(body, 'text'));
            return done;
        },
        method: 'POST',
        path: /^\/sessions\/(?<id>[^/]+)\/type$/u,
    },
    { action: scroll, method: 'POST', path: /^\/sessions\/(?<id>[^/]+)\/scroll$/u },
    {
        action: async (sessions, id, body) => {
            await sessions.get(id).page.waitForSelector(required(body, 'selector'), {
                state: oneOf(SELECTOR_STATES, text(body, 'state'), 'visible'),
                timeout: number(body, 'timeout_ms', OPERATION_TIMEOUT_MS),
            });
            return done;
        },
        method: 'POST',
        path: /^\/sessions\/(?<id>[^/]+)\/wait-selector$/u,
    },
    { action: waitResponse, method: 'POST', path: /^\/sessions\/(?<id>[^/]+)\/wait-response$/u },
    {
        action: async (sessions, id, body) => ({
            body: { result: await sessions.get(id).page.evaluate(required(body, 'script')) },
            status: 200,
        }),
        method: 'POST',
        path: /^\/sessions\/(?<id>[^/]+)\/eval$/u,
    },
    { action: captured, method: 'POST', path: /^\/sessions\/(?<id>[^/]+)\/captured-responses$/u },
    {
        action: async (sessions) => ({
            body: { browser_connected: sessions.connected, ok: true, sessions: sessions.size },
            status: 200,
        }),
        method: 'GET',
        path: /^\/healthz$/u,
    },
];
