export function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
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
