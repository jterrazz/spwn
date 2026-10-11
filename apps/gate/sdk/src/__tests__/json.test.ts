import { describe, expect, test } from 'vitest';

import { fieldOf, parsedOrText } from '../json.ts';

describe('json', () => {
    test('fieldOf reads a key of an object, and nothing from null or a value that is no object', () => {
        // Given - an object, null, a string and a number
        const read = [{ key: 'value' }, null, 'text', 7].map((value) => fieldOf(value, 'key'));

        // Then - the object answers, the rest hold none
        expect(read).toStrictEqual(['value', undefined, undefined, undefined]);
    });

    test('parsedOrText parses JSON and hands back the text itself when it is none', () => {
        // Given - a JSON object, a JSON string and a text that is no JSON
        const read = ['{"a":1}', '"quoted"', 'plain words'].map((text) => parsedOrText(text));

        // Then - the first two are parsed, the last is returned as written
        expect(read).toStrictEqual([{ a: 1 }, 'quoted', 'plain words']);
    });
});
