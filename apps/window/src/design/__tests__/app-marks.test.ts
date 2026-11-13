import { siGit } from 'simple-icons';
import { describe, expect, test } from 'vitest';

import { ALIASES, LOGOS, markOf, shapesOf } from '../app-marks';
import { MENUBAR_MARKS } from '../menubar-marks';

describe('app marks', () => {
    test("an app or an account the menu bar app marks is drawn by the menu bar's own mark", () => {
        // Given - the names the engine sends for GitHub's CLI, its account, Node and Claude's desktop app
        const names = ['gh', 'github', 'node', 'claude-desktop'];

        // When - each is marked
        const marks = names.map((name) => markOf(name));

        // Then - each draws the menu bar's mark of its brand, in the ink of the text
        expect(marks).toStrictEqual([
            { kind: 'drawn', shapes: shapesOf(MENUBAR_MARKS.github ?? '') },
            { kind: 'drawn', shapes: shapesOf(MENUBAR_MARKS.github ?? '') },
            { kind: 'drawn', shapes: shapesOf(MENUBAR_MARKS.node ?? '') },
            { kind: 'drawn', shapes: shapesOf(MENUBAR_MARKS.claude ?? '') },
        ]);
        expect(JSON.stringify(marks)).not.toContain(':"#000"');
    });

    test('a name the menu bar has no mark for is drawn by its Simple Icons logo, in the same 20 px box', () => {
        // Given - git, which the menu bar does not mark
        const name = 'git';

        // When - it is marked
        const mark = markOf(name);

        // Then - its logo is drawn on 16 px of the 20 px box, as the menu bar's marks are
        expect(mark).toStrictEqual({
            kind: 'drawn',
            shapes: [
                {
                    attributes: { d: siGit.path, transform: 'translate(2 2) scale(0.6667)' },
                    children: [],
                    key: 'git',
                    tag: 'path',
                },
            ],
        });
    });

    test('a mark is read into its shapes, nested as written, its attributes named as React names them', () => {
        // Given - a group of two strokes and a filled dot
        const markup =
            '<g stroke="#000" stroke-width="1.9"><line x1="1" y1="2"/><line x1="3" y1="4"/></g><circle r=".5" fill="#000"/>';

        // When - it is read
        const shapes = shapesOf(markup);

        // Then - the group holds its two lines, the dot follows it, and the ink is the text's
        expect(shapes).toStrictEqual([
            {
                attributes: { stroke: 'currentColor', strokeWidth: '1.9' },
                children: [
                    {
                        attributes: { x1: '1', y1: '2' },
                        children: [],
                        key: 'line x1="1" y1="2"',
                        tag: 'line',
                    },
                    {
                        attributes: { x1: '3', y1: '4' },
                        children: [],
                        key: 'line x1="3" y1="4"',
                        tag: 'line',
                    },
                ],
                key: 'g stroke="#000" stroke-width="1.9"',
                tag: 'g',
            },
            {
                attributes: { fill: 'currentColor', r: '.5' },
                children: [],
                key: 'circle r=".5" fill="#000"',
                tag: 'circle',
            },
        ]);
    });

    test('a name with no mark is drawn by its first two letters or digits, the same each time', () => {
        // Given - names no mark identifies, one opening on punctuation, one empty
        const names = ['herdr', 'session:github', '1password-cli', ''];

        // When - each is marked twice
        const first = names.map((name) => markOf(name));
        const second = names.map((name) => markOf(name));

        // Then - each is its monogram, and the two reads agree
        expect(first).toStrictEqual([
            { kind: 'monogram', letters: 'he' },
            { kind: 'monogram', letters: 'se' },
            { kind: 'monogram', letters: '1p' },
            { kind: 'monogram', letters: '' },
        ]);
        expect(second).toStrictEqual(first);
    });

    test('a setting is drawn by the glyph of its shelf, even when it names an app', () => {
        // Given - a setting named for the app it configures
        const name = 'docker-shortcuts';

        // When - it is marked as a setting, and as an app
        const asSetting = markOf(name, 'setting');
        const asApp = markOf('docker', 'app');

        // Then - the setting draws the glyph, the app its mark
        expect(asSetting).toStrictEqual({ glyph: 'settings', kind: 'glyph' });
        expect(asApp.kind).toBe('drawn');
    });

    test('a name is held by one table only, and every alias names a menu bar mark', () => {
        // Given - the names of the logos, the aliases and the menu bar's marks
        const logos = Object.keys(LOGOS);
        const marks = Object.keys(MENUBAR_MARKS);

        // When - the logos the menu bar already marks, and the aliases that name no mark, are kept
        const shadowed = logos.filter((name) => marks.includes(name) || name in ALIASES);
        const dangling = Object.values(ALIASES).filter((mark) => !marks.includes(mark));

        // Then - none is, and every name is written as the engine sends it, in lower case
        expect(shadowed).toStrictEqual([]);
        expect(dangling).toStrictEqual([]);
        expect(
            [...logos, ...Object.keys(ALIASES)].filter((name) => name !== name.toLowerCase()),
        ).toStrictEqual([]);
    });
});
