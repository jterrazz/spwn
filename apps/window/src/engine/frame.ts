import type { Answer, CoreCallWireError, EngineHello } from '@jterrazz/station-api';

export function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}

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

function canonical(value: unknown): unknown {
    if (Array.isArray(value)) {
        return value.map(canonical);
    }
    if (isRecord(value)) {
        return Object.fromEntries(
            Object.keys(value)
                .toSorted()
                .map((key) => [key, canonical(value[key])]),
        );
    }
    return value;
}

export function keyOf(request: unknown): string {
    if (!isRecord(request)) {
        return 'none';
    }
    const name =
        'watch' in request ? `watch ${String(request.watch)}` : `call ${String(request.call)}`;
    return `${name} ${JSON.stringify(canonical(request.in ?? {}))} ${JSON.stringify(canonical(request.target ?? {}))}`;
}
