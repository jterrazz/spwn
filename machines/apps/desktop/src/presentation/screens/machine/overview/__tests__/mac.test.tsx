import { button, component, content, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../../composition/compose';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../../../dependencies/dependencies';
import type { Route } from '../../../../navigation/route';
import { Window } from '../../../../navigation/window';
import { PAGES } from '../../../registry';

const recordingFor = await recordingsOf('create-failed', 'creating');

function macOver(world: string) {
    const initial: Route = { machine: 'macbook-pro', page: 'machine', tab: 'Overview' };
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('mac', () => {
    test("a machine whose create failed reads its standing on its Mac's line, the step it failed at beside it", async () => {
        // Given - the MacBook Pro's Overview over create-failed, linux-2's create failed at its boot
        const result = await component.render(macOver('create-failed'), async (visitor) => {
            // When - its machines are read
            const machines = region('Its machines');
            await visitor.see(within(machines, button('linux-2', { exact: false })));

            // Then - its state is the engine's standing, and the step is its line's first part
            await visitor.see(within(machines, content('not created')));
            await visitor.see(within(machines, content('Failed at boot 2 of 4 · by you')));
        });
        expect(result.errors.text).toBe('');
    });

    test("a machine being created reads its create's step as its state", async () => {
        // Given - the MacBook Pro's Overview over creating, linux-4 booting
        const result = await component.render(macOver('creating'), async (visitor) => {
            // When - its machines are read
            // Then - linux-4's state is the step its create runs
            await visitor.see(within(region('Its machines'), content('Creating: boot 2 of 4')));
        });
        expect(result.errors.text).toBe('');
    });
});
