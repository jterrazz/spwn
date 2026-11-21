import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Link } from '../../../../application/ports/engine.port';
import { blippedReplay } from '../blip';

describe('blip', () => {
    test('the link is open, then absent once for its lasting, then open again on the same engine', async () => {
        // Given - a blipped replay of an empty world, absent after 1 s for 3 s
        using _ = clock.run('2026-11-21T10:00:00Z');
        const heard: Link[] = [];
        const transport = blippedReplay(
            { exchanges: [], station: 'v1', world: 'empty' },
            undefined,
            { after: 1000, lasting: 3000 },
        );
        transport.link((link) => {
            heard.push(link);
        });

        // When - the clock runs past the blip
        await clock.advance(1);
        const before = heard.map((link) => link.kind);
        await clock.advance(1000);
        const during = heard.map((link) => link.kind);
        await clock.advance(3000);

        // Then - open, absent, open, the last as the first
        expect(before).toStrictEqual(['open']);
        expect(during).toStrictEqual(['open', 'absent']);
        expect(heard.map((link) => link.kind)).toStrictEqual(['open', 'absent', 'open']);
        expect(heard[2]).toStrictEqual(heard[0]);
    });
});
