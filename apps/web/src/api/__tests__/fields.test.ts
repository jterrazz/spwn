import { describe, expect, test } from 'vitest';

import { fieldOf, listOf, numberOf, recordOf, textOf } from '../fields';

describe('fields', () => {
    test('a field is read from an object, and from nothing else', () => {
        // Given - an object, null, an array, a string and nothing
        const read = [{ key: 'value' }, null, ['x'], 'text', undefined].map((value) =>
            fieldOf(value, 'key'),
        );

        // Then - only the object answers
        expect(read).toStrictEqual(['value', undefined, undefined, undefined, undefined]);
    });

    test('each reader falls back when the field is absent or of another type', () => {
        // Given - a payload holding a string, a list, a number and an object
        const payload = { count: 3, list: ['a', 'b'], name: 'neo', object: { a: 1 } };

        // Then - a field of the right type reads whole, any other reads its fallback
        expect([
            textOf(payload, 'name'),
            textOf(payload, 'count'),
            textOf(null, 'name'),
            listOf(payload, 'list'),
            listOf(payload, 'name'),
            listOf({ list: null }, 'list'),
            numberOf(payload, 'count'),
            numberOf(payload, 'name'),
            numberOf({ count: Number.NaN }, 'count'),
        ]).toStrictEqual(['neo', '', '', ['a', 'b'], [], [], 3, undefined, Number.NaN]);
    });

    test('a record keeps the fields of an object, and is empty for null, a list or a value', () => {
        // Given - an object field, a null one, a list, a string and a missing one
        const payload = { list: [1], nothing: null, object: { a: 1, b: 'two' }, text: 'x' };

        // Then - the object's own fields, and {} for the rest
        expect([
            recordOf(payload, 'object'),
            recordOf(payload, 'nothing'),
            recordOf(payload, 'list'),
            recordOf(payload, 'text'),
            recordOf(payload, 'missing'),
        ]).toStrictEqual([{ a: 1, b: 'two' }, {}, {}, {}, {}]);
    });
});
