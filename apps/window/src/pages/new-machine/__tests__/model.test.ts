import { describe, expect, test } from 'vitest';

import { EMPTY_DRAFT, imagesOf, serverInputOf, sizeIsWhole, workspaceInputOf } from '../model';

function record(name: string, version: string) {
    return {
        Mac: 'jterrazz-macbook-pro',
        Newest: true,
        Record: { base: '', built_at: '', name, recipe_sha: '', size_gb: 4, station: '', version },
        VMs: null,
    };
}

describe('model', () => {
    test('an untouched draft asks the engine with no field, so each is its default', () => {
        // Given - the empty draft

        // When - the inputs of both calls are made of it
        const workspace = workspaceInputOf(EMPTY_DRAFT);
        const server = serverInputOf(EMPTY_DRAFT);

        // Then - neither carries a field the owner did not give
        expect(workspace).toStrictEqual({ Request: {} });
        expect(server).toStrictEqual({});
    });

    test('a filled draft carries its name, image, lifetime and size in the units of the input', () => {
        // Given - a persistent workspace of 2 CPUs, 6 GB of memory and 40 GB of disk
        const draft = {
            ...EMPTY_DRAFT,
            cpus: '2',
            diskGB: '40',
            image: 'station-linux',
            lifetime: 'persistent' as const,
            memoryGB: ' 6 ',
            name: ' gate5 ',
        };

        // When - its inputs are made
        const workspace = workspaceInputOf(draft);
        const server = serverInputOf({ ...draft, kind: 'server' });

        // Then - sizes are bytes, and a server never carries a lifetime
        const Size = { CPUs: 2, DiskBytes: 40 * 2 ** 30, MemoryBytes: 6 * 2 ** 30 };
        expect(workspace).toStrictEqual({
            Request: { Image: 'station-linux', Lifetime: 'persistent', Name: 'gate5', Size },
        });
        expect(server).toStrictEqual({ Image: 'station-linux', Name: 'gate5', Size });
    });

    test('a size that is not a whole count is not sent', () => {
        // Given - drafts whose memory is a word, a fraction and zero
        const words = ['eight', '1.5', '0'];

        // When - each is judged sendable
        const sendable = words.map((memoryGB) => sizeIsWhole({ ...EMPTY_DRAFT, memoryGB }));

        // Then - none is, while an empty one is
        expect(sendable).toStrictEqual([false, false, false]);
        expect(sizeIsWhole(EMPTY_DRAFT)).toBe(true);
    });

    test('the images offered are the names images.list gives, each once', () => {
        // Given - two versions of station-linux and one of macos, then no answer
        const reports = [
            record('station-linux', '2'),
            record('station-linux', '1'),
            record('macos', '1'),
        ];

        // When - the names are read
        const names = imagesOf(reports);

        // Then - each name once, in the engine's order
        expect(names).toStrictEqual(['station-linux', 'macos']);
        expect(imagesOf(null)).toStrictEqual([]);
    });
});
