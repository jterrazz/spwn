import { RefusedError } from './http.ts';

/** A request's JSON object. */
export type Body = Record<string, unknown>;
/** What an action answers: the status and the body the sidecar sends. */
export type Reply = { body: unknown; status: number };

/** How long an action waits on the page unless the request names its own `timeout_ms`. */
export const OPERATION_TIMEOUT_MS = 20_000;

/** The string at `key`, or empty when there is none. */
export const text = (body: Body, key: string): string => {
    const value = body[key];
    return typeof value === 'string' ? value : '';
};
/** The number at `key`, the fallback for none or zero. */
export const number = (body: Body, key: string, fallback: number): number => {
    const value = body[key];
    return typeof value === 'number' && value !== 0 ? value : fallback;
};
/** The non-empty string at `key`, refused with a 400 otherwise. */
export const required = (body: Body, key: string): string => {
    const value = text(body, key);
    if (value === '') {
        throw new RefusedError(400, `${key} is required`);
    }
    return value;
};
/** The string at `key`, empty included, refused with a 400 when there is none. */
export const given = (body: Body, key: string): string => {
    const value = body[key];
    if (typeof value !== 'string') {
        throw new RefusedError(400, `${key} is required`);
    }
    return value;
};
/** The answer of an action that returns nothing. */
export const done: Reply = { body: { ok: true }, status: 200 };
