import { describe, expect, test } from 'vitest';

import { EngineError } from '../../../domain/engine-error';
import { replaceEqualDeep } from '../equal';

const LAST = {
    macs: [
        { name: 'studio', vms: [{ name: 'api', state: 'running' }] },
        { name: 'laptop', vms: [] },
    ],
    read: true,
};

function macsOf(value: unknown): readonly unknown[] {
    const macs: unknown =
        typeof value === 'object' && value !== null && 'macs' in value ? value.macs : undefined;
    return Array.isArray(macs) ? macs : [];
}

describe('equal', () => {
    test('an answer deep-equal to the last is the last itself', () => {
        // Given - the last answer, and a new one built apart with the same values
        const next = structuredClone(LAST);

        // When - the new one is merged into the last
        const merged = replaceEqualDeep(LAST, next);

        // Then - the last answer is kept whole
        expect(merged).toBe(LAST);
    });

    test('a part that moved is new, and every part that did not keeps its identity', () => {
        // Given - a new answer where one VM of the first Mac stopped
        const next = structuredClone(LAST);
        next.macs[0] = { name: 'studio', vms: [{ name: 'api', state: 'stopped' }] };

        // When - it is merged into the last
        const merged = replaceEqualDeep(LAST, next);

        // Then - it reads as the new answer, the moved Mac is new, and the other Mac is the last's own
        expect(merged).toStrictEqual(next);
        expect(merged).not.toBe(LAST);
        const [first, second] = macsOf(merged);
        expect([first === LAST.macs[0], second === LAST.macs[1]]).toStrictEqual([false, true]);
    });

    test('a key added or removed, a longer list, and an equal error of another instance are changes', () => {
        // Given - four changes of shape the values alone would hide
        const added = { ...LAST, extra: undefined };
        const removed = { macs: LAST.macs };
        const longer = [1, 2, 3];
        const error = new EngineError(undefined, 'no answer');

        const short = [1, 2];
        const before = new EngineError(undefined, 'no answer');

        // When - each is merged into what it replaces
        const merged = [
            replaceEqualDeep(LAST, added),
            replaceEqualDeep(LAST, removed),
            replaceEqualDeep(short, longer),
            replaceEqualDeep(before, error),
        ];

        // Then - each reads as the new value and none is the one it replaces; the error is the new one itself
        expect(merged).toStrictEqual([added, removed, longer, error]);
        expect(merged.map((value, at) => value === [LAST, LAST, short, before][at])).toStrictEqual([
            false,
            false,
            false,
            false,
        ]);
        expect(merged[3]).toBe(error);
    });
});
