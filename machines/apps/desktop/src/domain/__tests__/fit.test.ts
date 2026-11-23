import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { STATE_REVISION } from '../fit';

const ENGINE_STATE = join(
    import.meta.dirname,
    '..',
    '..',
    '..',
    '..',
    'engine',
    'engine',
    'state.go',
);

describe('fit', () => {
    test("the window reads its tree's engine's state revision", () => {
        // Given - the revision of station.state/v1 the engine of this tree speaks
        const revision = /StateRevision = (?<n>\d+)/u.exec(readFileSync(ENGINE_STATE, 'utf8'))
            ?.groups?.n;

        // When - the window's own revision is read against it

        // Then - the window reads that revision
        expect(Number(revision)).toBe(STATE_REVISION);
    });
});
