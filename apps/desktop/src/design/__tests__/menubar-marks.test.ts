import { describe, expect, test } from 'vitest';

import { fileOf, marksOf } from '../../../scripts/menubar-marks';
import { MENUBAR_MARKS } from '../menubar-marks';

function rawOf(sources: Record<string, string>): string {
    return Object.values(sources)[0] ?? '';
}

const SYMBOLS = rawOf(
    import.meta.glob<string>('../../../../menubar/scripts/symbols.swift', {
        eager: true,
        import: 'default',
        query: '?raw',
    }),
);
const WRITTEN = rawOf(
    import.meta.glob<string>('../menubar-marks.ts', {
        eager: true,
        import: 'default',
        query: '?raw',
    }),
);

describe('menu bar marks', () => {
    test("the window's marks are the menu bar app's, as its symbols script writes them", () => {
        // Given - the menu bar app's script, the marks' one owner
        const swift = SYMBOLS;

        // When - its marks are read and written as the window's file
        const marks = marksOf(swift);

        // Then - the window's file is that writing, byte for byte, and holds each mark the menu bar draws
        expect(WRITTEN).toBe(fileOf(marks));
        expect(MENUBAR_MARKS).toStrictEqual(marks);
        expect(Object.keys(marks)).toHaveLength(15);
    });

    test('a mark the menu bar draws by code is read as the code draws it', () => {
        // Given - Claude's six strokes and Tailscale's nine dots, each written by a map in the script
        const marks = marksOf(SYMBOLS);

        // When - their drawings are counted
        const strokes = (marks.claude ?? '').match(/<line /gu) ?? [];
        const dots = (marks.tailscale ?? '').match(/<circle /gu) ?? [];

        // Then - each is drawn whole, the rotations and the opacities in place
        expect(strokes).toHaveLength(6);
        expect(marks.claude).toContain('transform="rotate(150 10 10)"');
        expect(dots).toHaveLength(9);
        expect(marks.tailscale).toContain(
            '<circle cx="10" cy="16" r="2.1" fill="#000" opacity="1.0"/>',
        );
    });

    test('a form of mark the reader does not know is refused, never drawn half', () => {
        // Given - a script whose mark is written by a form the reader does not know
        const swift = 'let strokes = ""\nlet marks: [String: String] = [\n    "odd": glyph(),\n]\n';

        // When - it is read
        const read = () => marksOf(swift);

        // Then - the reader throws, naming what it cannot read
        expect(read).toThrow('symbols.swift: the window cannot read glyph()');
    });
});
