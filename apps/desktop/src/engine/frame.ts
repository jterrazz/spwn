import type { Answer, CoreCallWireError, EngineHello } from '@jterrazz/station-api';

import { isRecord } from '../domain/request-key';

function isWireError(value: unknown): value is CoreCallWireError {
    return isRecord(value) && typeof value.message === 'string';
}

export function answerOf(value: unknown): Answer | undefined {
    if (!isRecord(value) || value.v !== 2 || typeof value.id !== 'string') {
        return undefined;
    }
    const { id } = value;
    if ('out' in value) {
        return { id, out: value.out, v: 2 };
    }
    if (isWireError(value.error)) {
        return { error: value.error, id, v: 2 };
    }
    if (value.done === true) {
        return { done: true, id, v: 2 };
    }
    if (isRecord(value.progress) && typeof value.progress.task === 'string') {
        return { id, progress: { ...value.progress, task: value.progress.task }, v: 2 };
    }
    return undefined;
}

export function helloOf(value: unknown): EngineHello | undefined {
    if (
        isRecord(value) &&
        typeof value.version === 'string' &&
        typeof value.pid === 'number' &&
        typeof value.started_at === 'string'
    ) {
        const calls = Array.isArray(value.calls)
            ? value.calls.filter((call): call is string => typeof call === 'string')
            : undefined;
        return {
            pid: value.pid,
            started_at: value.started_at,
            version: value.version,
            ...(calls ? { calls } : {}),
        };
    }
    return undefined;
}

export function wireErrorOf(value: unknown): CoreCallWireError | undefined {
    return isWireError(value) ? value : undefined;
}
