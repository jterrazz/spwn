import type { Response } from 'playwright';

import { number, OPERATION_TIMEOUT_MS, required, text } from './fields.ts';
import type { Body, Reply } from './fields.ts';
import type { Sessions } from './sessions.ts';

async function bodyOf(response: Response): Promise<unknown> {
    try {
        const json: unknown = await response.json();
        return json;
    } catch {
        return await response.text().catch(() => null);
    }
}

/** Awaits the next response of the page matching the address, the method and, unless any is allowed, a JSON type. */
export async function waitResponse(sessions: Sessions, id: string, body: Body): Promise<Reply> {
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
                    candidate.headers()['content-type']?.toLowerCase().includes('json') === true),
            { timeout: number(body, 'timeout_ms', OPERATION_TIMEOUT_MS) },
        );
    return {
        body: { body: await bodyOf(response), status: response.status(), url: response.url() },
        status: 200,
    };
}

/** The responses the page received since an instant, at an address. */
export async function captured(sessions: Sessions, id: string, body: Body): Promise<Reply> {
    const pattern = new RegExp(text(body, 'url_pattern'), 'u');
    const since = number(body, 'since_ts', 0);
    const kept = sessions
        .get(id)
        .captured.filter((entry) => entry.ts >= since && pattern.test(entry.url));
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
