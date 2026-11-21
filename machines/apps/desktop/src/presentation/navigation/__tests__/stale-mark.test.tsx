import { component, content, status, waitUntil } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { PAGE_REVISION } from '../../../domain/fit';
import { documentPresence } from '../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { counted } from '../../dependencies/__tests__/counted';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { Window } from '../window';

const recordingFor = await recordingsOf('two-macs');

const HELLO = { page: PAGE_REVISION, pid: 41, started_at: '2026-11-13T09:00:00Z', version: 'dev' };

function focusAgain() {
    globalThis.dispatchEvent(new Event('blur'));
    globalThis.dispatchEvent(new Event('focus'));
}

describe('StaleMark', () => {
    test('a refresh refused keeps the page and marks the toolbar with the time of the answer still shown', async () => {
        // Given - linux-1's Ports over two-macs, on a store whose clock says 10:42 then 10:43
        const stage = stagedReplay(recordingFor('two-macs'), {
            hello: HELLO,
            kind: 'open',
            refused: undefined,
            socket: 'replay',
        });
        const count = counted(stage.transport);
        let now = new Date(2026, 10, 13, 10, 42);
        const result = await component.render(
            <EngineProvider
                dependencies={{
                    engine: new Client(count.transport),
                    opener: NO_OPENER,
                    presence: documentPresence(),
                    reads: { now: () => now },
                }}
            >
                <Window
                    initial={{ machine: 'linux-1', page: 'machine', tab: 'Ports' }}
                    pages={PAGES}
                />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content(':5173'));

                // When - a minute on, the window regains the focus and ports.list is refused as unreachable
                now = new Date(2026, 10, 13, 10, 43);
                count.refuse('ports.list');
                focusAgain();
                await waitUntil(() => count.refused() > 0, { why: 'ports.list refused' });

                // Then - the page is drawn as answered, and the mark names the refusal and the answer's time
                await visitor.see(status('macbook-pro did not answer'));
                await visitor.see(content('as of 10:42'));
                await visitor.see(content(':5173'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
