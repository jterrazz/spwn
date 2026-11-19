import { button, component, content, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../../composition/compose';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../../../dependencies/dependencies';
import type { Route } from '../../../../navigation/route';
import { Window } from '../../../../navigation/window';
import { PAGES } from '../../../registry';

const recordingFor = await recordingsOf('create-failed', 'two-macs');

function machineOver(world: string, machine: string) {
    const initial: Route = { machine, page: 'machine', tab: 'Overview' };
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

const FAILED = 'Failed at boot 2 of 4';

describe('birth', () => {
    test('a create that failed draws its head, the step it failed at and the last lines its console wrote', async () => {
        // Given - linux-2's page over create-failed, its create failed at its boot
        const result = await component.render(
            machineOver('create-failed', 'linux-2'),
            async (visitor) => {
                // When - machines.list has answered
                await visitor.see(heading('linux-2 not created'));

                // Then - the head is the engine's, its line the cause, and the step titles the console's lines
                await visitor.see(content('linux-2 was not created: it failed at boot.'));
                const cause =
                    'linux-2 did not boot: the hypervisor refused to boot it, as the world scripted it';
                await visitor.see(content(cause));
                const step = region(FAILED);
                await visitor.see(within(step, content('What its console last wrote')));
                await visitor.see(
                    within(step, content('cloud-init[498]: running init-local', { exact: false })),
                );
                await visitor.see(button('Start'));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('Linux version');
        expect(result.content.text).not.toContain('Creating');
    });

    test('a VM stopped by its owner draws no failed create', async () => {
        // Given - api's page over two-macs, stopped, its create long done
        const result = await component.render(machineOver('two-macs', 'api'), async (visitor) => {
            // When - machines.list has answered
            await visitor.see(heading('api stopped'));
        });

        // Then - no step and no console line are drawn
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('What its console last wrote');
    });
});
