import { done, given, number, OPERATION_TIMEOUT_MS, required, text } from './fields.ts';
import type { Body, Reply } from './fields.ts';
import { captured, waitResponse } from './responses.ts';
import { HARD_TTL_MS } from './session.ts';
import type { Sessions } from './sessions.ts';

export type Route = {
    action: (sessions: Sessions, id: string, body: Body) => Promise<Reply>;
    method: 'DELETE' | 'GET' | 'POST';
    path: RegExp;
};

const SCROLL_DELTA = 4000;
const SCROLL_WAIT_MS = 1500;

const LOAD_STATES = ['commit', 'domcontentloaded', 'load', 'networkidle'] as const;
const SELECTOR_STATES = ['attached', 'detached', 'hidden', 'visible'] as const;

function oneOf<T extends string>(values: readonly T[], value: string, fallback: T): T {
    return values.find((candidate) => candidate === value) ?? fallback;
}

async function scroll(sessions: Sessions, id: string, body: Body): Promise<Reply> {
    const { page } = sessions.get(id);
    for (let step = 0; step < number(body, 'count', 1); step += 1) {
        await page.mouse.wheel(0, number(body, 'delta_y', SCROLL_DELTA));
        // Stryker disable next-line StringLiteral: the wait only paces a page's lazy loading between wheels, and no answer reads it
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
            // Stryker disable next-line StringLiteral: a 204 carries no body, so node drops whatever is written
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
            const selector = required(body, 'selector');
            const typed = given(body, 'text');
            await page.click(selector, {
                timeout: number(body, 'timeout_ms', OPERATION_TIMEOUT_MS),
            });
            await page.keyboard.type(typed);
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
