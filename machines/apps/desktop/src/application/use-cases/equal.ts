function isPlain(value: unknown): value is Readonly<Record<string, unknown>> {
    if (typeof value !== 'object' || value === null) {
        return false;
    }
    const prototype: unknown = Object.getPrototypeOf(value);
    return prototype === Object.prototype || prototype === null;
}

function sharedList(before: readonly unknown[], after: readonly unknown[]): readonly unknown[] {
    const merged = after.map((item, at) => replaceEqualDeep(before[at], item));
    const same = merged.length === before.length && merged.every((item, at) => item === before[at]);
    return same ? before : merged;
}

function sharedRecord(
    before: Readonly<Record<string, unknown>>,
    after: Readonly<Record<string, unknown>>,
): Readonly<Record<string, unknown>> {
    const keys = Object.keys(after);
    const merged = Object.fromEntries(
        keys.map((key) => [key, replaceEqualDeep(before[key], after[key])]),
    );
    const same =
        keys.length === Object.keys(before).length &&
        keys.every((key) => Object.hasOwn(before, key) && merged[key] === before[key]);
    return same ? before : merged;
}

export function replaceEqualDeep(before: unknown, after: unknown): unknown {
    if (Object.is(before, after)) {
        return before;
    }
    if (Array.isArray(before) && Array.isArray(after)) {
        return sharedList(before, after);
    }
    if (isPlain(before) && isPlain(after)) {
        return sharedRecord(before, after);
    }
    return after;
}
