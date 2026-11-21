import { describe, expect, test } from 'vitest';

import { fitOf } from '../../../../domain/fit';
import { fittedLink } from '../fitted';
import type { Recording } from '../replay';

const RECORDING: Recording = { exchanges: [], station: 'dev', world: 'test' };

describe('fitted', () => {
    test("each fit the query names is a hello the window judges so, and any other leaves the world's own", () => {
        // Given - the two fits a proof build plays, a name it does not know, and none

        // When - each is made the world's link, and its hello judged
        const links = ['engine-older', 'app-older', 'stale', null].map((fit) =>
            fittedLink(RECORDING, fit),
        );
        const fits = links.map((link) =>
            link?.kind === 'open' ? fitOf(link.hello?.page) : undefined,
        );

        // Then - an older engine and an older window, then the recorded hello twice
        expect(fits).toStrictEqual(['engine-older', 'app-older', undefined, undefined]);
        expect(links[0]?.socket).toBe('replay of test');
    });
});
