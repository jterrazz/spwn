import { describe, expect, test } from 'vitest';

import { menuBarScriptOf, scriptedMenuBar } from '../scripted-menu-bar';

function scripted(name: string) {
    const script = menuBarScriptOf(name);
    if (script === undefined) {
        throw new Error(`no menu bar script ${name}`);
    }
    return scriptedMenuBar(script);
}

describe('scripted menu bar', () => {
    test('a hidden menu bar shown reads shown, and hidden again reads hidden', async () => {
        // Given - the script whose menu bar starts hidden
        const port = scripted('hidden');

        // When - it is read, shown, read, hidden and read
        const read = [
            await port.read(),
            await port.show(true),
            await port.read(),
            await port.show(false),
            await port.read(),
        ];

        // Then - each answer as macOS would hold it
        expect(read.map((answer) => answer.kind)).toStrictEqual([
            'hidden',
            'shown',
            'shown',
            'hidden',
            'hidden',
        ]);
    });

    test('a refused show keeps the menu bar hidden, an approval holds, and an unknown name is no script', async () => {
        // Given - the refusing and the approval scripts
        const refusing = scripted('refusing');
        const approval = scripted('approval');

        // When - each is shown, then read
        const ends = [
            await refusing.show(true),
            await refusing.read(),
            await approval.show(true),
            await approval.read(),
        ];

        // Then - refused then still hidden; approval, then approval again
        expect(ends.map((answer) => answer.kind)).toStrictEqual([
            'refused',
            'hidden',
            'approval',
            'approval',
        ]);
        expect(menuBarScriptOf('later')).toBeUndefined();
    });
});
