import { describe, expect, test } from 'vitest';

import { mouseShortcutOf, shortcutOf } from '../shortcut';
import type { Keystroke } from '../shortcut';

const bare: Keystroke = { ctrlKey: false, key: '', metaKey: false, shiftKey: false };

describe('shortcut', () => {
    test.each([
        { expected: 'back', stroke: { ...bare, key: '[', metaKey: true } },
        { expected: 'home', stroke: { ...bare, key: '0', metaKey: true } },
        { expected: 'find', stroke: { ...bare, key: 'K', metaKey: true } },
        { expected: 'find', stroke: { ...bare, ctrlKey: true, key: 'k' } },
        { expected: 'forward', stroke: { ...bare, key: ']', metaKey: true } },
        { expected: 'Overview', stroke: { ...bare, key: '1', metaKey: true } },
        { expected: 'Storage', stroke: { ...bare, key: '6', metaKey: true } },
        { expected: undefined, stroke: { ...bare, key: '7', metaKey: true } },
        { expected: 'gallery', stroke: { ...bare, key: 'g', metaKey: true, shiftKey: true } },
        { expected: undefined, stroke: { ...bare, key: '[' } },
        { expected: undefined, stroke: { ...bare, key: 'g', metaKey: true } },
    ])('reads $stroke.key as $expected', ({ expected, stroke }) => {
        // Given - a keystroke in a window that holds the gallery
        const shortcut = shortcutOf(stroke, true);

        // Then - the command it names, or none without a command key
        expect(shortcut).toBe(expected);
    });

    test("a mouse's back and forward buttons are back and forward, and no other button is", () => {
        // Given - the buttons a mouse reports: main, middle, secondary, back, forward
        const read = [0, 1, 2, 3, 4].map((pressed) => mouseShortcutOf(pressed));

        // Then - only the fourth and fifth name a command
        expect(read).toStrictEqual([undefined, undefined, undefined, 'back', 'forward']);
    });

    test('a window without the gallery has no gallery shortcut', () => {
        // Given - ⌘⇧G in a production window
        const shortcut = shortcutOf({ ...bare, key: 'g', metaKey: true, shiftKey: true }, false);

        // Then - nothing answers it
        expect(shortcut).toBeUndefined();
    });
});
