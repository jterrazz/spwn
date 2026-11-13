import { describe, expect, test } from 'vitest';

import { identityOf } from '../identity';

describe('identity', () => {
    test('each machine takes the next colour by its place in the listing, every Mac in turn', () => {
        // Given - the mockup's two Macs: five machines on the first, infra and relay on the second
        const listed = ['workspace', 'gate', 'gate2', 'gate3', 'gate4', 'infra', 'relay'];

        // When - each is coloured
        const colours = listed.map((name) => identityOf(name, listed));

        // Then - the mockup's seven colours in its order, so no two machines of one Mac share one
        expect(colours).toStrictEqual(['blue', 'purple', 'teal', 'indigo', 'pink', 'sky', 'lilac']);
    });

    test('two names whose letters sum alike still differ, and the eighth machine starts the colours again', () => {
        // Given - infra and linux-3 on one Mac, which a hash of the name coloured alike
        const listed = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'infra', 'linux-3'];

        // Then - infra wraps to the first colour and linux-3 takes the second
        expect(identityOf('infra', listed)).toBe('blue');
        expect(identityOf('linux-3', listed)).toBe('purple');
    });

    test('a name the listing does not hold, a Mac among them, has no colour', () => {
        // Given - a listing of the machines alone
        const listed = ['infra'];

        // Then - a Mac or a machine gone draws in no identity colour
        expect(identityOf('jterrazz-mac-mini', listed)).toBeUndefined();
    });
});
