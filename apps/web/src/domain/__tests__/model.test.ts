import { describe, expect, test } from 'vitest';

import { getWorkspaceSummary, getWorldName } from '../model';

describe('model', () => {
    test('a world is known by its name, trimmed, else by the middle word of its id, else by its id', () => {
        // Given - worlds with a padded name, a blank name, no name, an id with no middle word and an empty one
        const names = [
            { id: 'w-titan-abc123', name: '  Atlas ' },
            { id: 'w-titan-abc123', name: '   ' },
            { id: 'w-titan-abc123' },
            { id: 'matrix' },
            { id: 'w--abc123' },
        ].map((world) => getWorldName(world));

        // Then - the trimmed name, the capitalized middle twice, then the id itself twice
        expect(names).toStrictEqual(['Atlas', 'Titan', 'Titan', 'matrix', 'w--abc123']);
    });

    test('the workspaces of a world sum up as ephemeral, the one path, or a count', () => {
        // Given - no mounts, an empty list, one mount and two
        const summaries = [
            {},
            { workspaces: [] },
            { workspaces: [{ name: 'app', path: '/src/app' }] },
            {
                workspaces: [
                    { name: 'app', path: '/src/app' },
                    { name: 'lib', path: '/src/lib' },
                ],
            },
        ].map((world) => getWorkspaceSummary(world));

        // Then - ephemeral twice, the path, the count
        expect(summaries).toStrictEqual(['ephemeral', 'ephemeral', '/src/app', '2 workspaces']);
    });
});
