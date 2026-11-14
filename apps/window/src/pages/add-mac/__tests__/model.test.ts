import { describe, expect, test } from 'vitest';

import { addInputOf, EMPTY_MAC } from '../model';

describe('model', () => {
    test('only the fields given are sent, trimmed', () => {
        // Given - a name alone, then a name, a login and a key with spaces around them
        const alone = { ...EMPTY_MAC, name: ' jterrazz-mac-mini ' };
        const all = { key: ' ~/.ssh/id_ed25519 ', name: 'jterrazz-mac-mini', user: ' jb ' };

        // When - their inputs are made
        const sent = [addInputOf(alone), addInputOf(all), addInputOf(EMPTY_MAC)];

        // Then - an empty field is left to the engine's default
        expect(sent).toStrictEqual([
            { Name: 'jterrazz-mac-mini' },
            { Key: '~/.ssh/id_ed25519', Name: 'jterrazz-mac-mini', User: 'jb' },
            {},
        ]);
    });
});
