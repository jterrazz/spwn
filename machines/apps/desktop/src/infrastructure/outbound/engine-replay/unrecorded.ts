import { isRecord } from '../../../domain/request-key';

function nameOf(request: unknown): string {
    if (!isRecord(request)) {
        return '';
    }
    const name = 'call' in request ? request.call : request.watch;
    return typeof name === 'string' ? name : '';
}

function asSent(request: unknown): string {
    if (!isRecord(request)) {
        return '{}';
    }
    const target = isRecord(request.target) ? { target: request.target } : {};
    return JSON.stringify({ in: request.in ?? {}, ...target });
}

export class UnrecordedGestureError extends Error {
    override readonly name = 'UnrecordedGestureError';
}

export function throwIf(unrecorded: undefined | UnrecordedGestureError): void {
    if (unrecorded) {
        throw unrecorded;
    }
}

export function unrecordedOf(
    request: unknown,
    recorded: readonly { request: unknown }[],
): undefined | UnrecordedGestureError {
    const name = nameOf(request);
    const inputs = recorded
        .filter((gesture) => nameOf(gesture.request) === name)
        .map((gesture) => asSent(gesture.request));
    if (name === '' || inputs.length === 0) {
        return undefined;
    }
    return new UnrecordedGestureError(
        `${name} was sent with ${asSent(request)}, but is recorded only with ${inputs.join(' or ')}`,
    );
}
