import { describe, expect, test } from 'vitest';

import { targetRowsOf } from '../passwords-of';

describe('passwords-of', () => {
    test("the targets that fit come first, each group in the engine's order, every line as sent", () => {
        // Given - the targets of grants.new, a held one first
        const targets = [
            { Fit: 'no', Line: 'Start the machine first', To: 'api' },
            { Fit: 'fits', Line: 'Signed out here', To: 'linux-1' },
            { Fit: 'no', Line: 'A server keeps its own sign-ins', To: 'infra' },
            { Fit: 'fits', Line: '', To: 'linux-2' },
        ] as const;

        // When - they are made rows
        const rows = targetRowsOf(targets);

        // Then - linux-1 and linux-2 to choose, then api and infra held, none worded by the window
        expect(rows).toStrictEqual([
            { held: false, id: 'linux-1', line: 'Signed out here', title: 'linux-1' },
            { held: false, id: 'linux-2', line: '', title: 'linux-2' },
            { held: true, id: 'api', line: 'Start the machine first', title: 'api' },
            { held: true, id: 'infra', line: 'A server keeps its own sign-ins', title: 'infra' },
        ]);
    });
});
