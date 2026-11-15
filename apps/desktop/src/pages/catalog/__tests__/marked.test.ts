import { describe, expect, test } from 'vitest';

import { codeSpansOf } from '../marked';

function shown(text: string) {
    return codeSpansOf(text).map((span) => [span.code, span.text]);
}

describe('marked', () => {
    test('a literal between backticks is a code span, the words around it plain', () => {
        // Given - a catalog note that marks a key
        const spans = shown("station's hook under `hooks`, every other key the owner's");

        // Then - the key alone is code, its backticks gone
        expect(spans).toStrictEqual([
            [false, "station's hook under "],
            [true, 'hooks'],
            [false, ", every other key the owner's"],
        ]);
    });

    test('a note opening on a literal starts with its code span', () => {
        // Given - a note whose first word is a program
        const spans = shown('`notify`, unless the owner set it to another program');

        // Then - the program, then the sentence
        expect(spans).toStrictEqual([
            [true, 'notify'],
            [false, ', unless the owner set it to another program'],
        ]);
    });

    test('an unmatched backtick leaves the text as sent', () => {
        // Given - a note with one backtick
        const spans = shown('a stray ` in a note');

        // Then - one plain span, the backtick kept
        expect(spans).toStrictEqual([[false, 'a stray ` in a note']]);
    });
});
