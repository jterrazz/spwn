import { component, content } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('mini-workspace');

const REFUSAL = 'infra is a server: Machines does not manage its disk';
const HINT = 'manage it with your own tooling: ssh infra';

function infraOverview(recording = recordingFor('mini-workspace')) {
    return pageOver(recording, { machine: 'infra', page: 'machine', tab: 'Overview' });
}

describe('disk tile', () => {
    test("a server's disk is the engine's refusal, never no answer", async () => {
        // Given - infra's Overview over mini-workspace, where infra's activity carries no disk and disk.read at infra is answered as not-for-this-machine
        const result = await component.render(infraOverview(), async (visitor) => {
            // When - its disk has been read
            await visitor.see(content(REFUSAL));
        });

        // Then - the tile says the engine's word and its sentence, its title carries the hint, and no tile says no answer
        expect(result.tree.text).toContain('not for this machine');
        expect(result.tree.text).not.toContain('no answer');
        expect(result.html.text).toContain(`title="${REFUSAL} · ${HINT}"`);
        expect(result.tree.text).not.toContain(HINT);
    });

    test('a disk read that was never answered stays no answer', async () => {
        // Given - the same Overview, disk.read at infra never answered
        const world = recordingFor('mini-workspace');
        const silent = {
            ...world,
            exchanges: world.exchanges.filter(
                ({ request }) =>
                    !(
                        request.watch === 'disk.read' &&
                        JSON.stringify(request.target) === '{"name":"infra"}'
                    ),
            ),
        };
        const result = await component.render(infraOverview(silent), async (visitor) => {
            // When - the page has been read
            await visitor.see(content('no answer'));
        });

        // Then - the engine's refusal is not drawn
        expect(result.tree.text).not.toContain(REFUSAL);
    });
});
