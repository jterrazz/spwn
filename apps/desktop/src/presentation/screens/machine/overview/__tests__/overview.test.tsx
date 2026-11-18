import { button, component, content, dialog, heading, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../../composition/compose';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../../../dependencies/dependencies';
import type { Route } from '../../../../navigation/route';
import { Window } from '../../../../navigation/window';
import { PAGES } from '../../../registry';

const recordingFor = await recordingsOf('stories', 'two-macs');

function machineOver(world: string, machine: string) {
    const initial: Route = { machine, page: 'machine', tab: 'Overview' };
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('overview', () => {
    test("a delete's dialog says its consequence alone while the machine's hold is calm", async () => {
        // Given - api's page over two-macs, api yours and stopped
        const result = await component.render(machineOver('two-macs', 'api'), async (visitor) => {
            await visitor.see(heading('api stopped'));

            // When - Delete api is clicked
            await visitor.click(button('Delete api…'));

            // Then - the dialog says what the delete ends, and nothing of its hold
            const asked = dialog('Delete api?');
            const said = content('api and everything on its disk are deleted for good.');
            await visitor.see(within(asked, said));
            await visitor.gone(within(asked, content('Yours', { exact: false })));
        });
        expect(result.errors.text).toBe('');
    });

    test("a delete's dialog draws a held machine's hold as the engine sends it, under the consequence", async () => {
        // Given - linux-1's page over stories, held by its lot, by studio-os-fe
        const result = await component.render(
            machineOver('stories', 'linux-1'),
            async (visitor) => {
                // When - Delete linux-1 is clicked
                await visitor.click(button('Delete linux-1…'));

                // Then - the consequence, then the hold on a line of its own
                const asked = dialog('Delete linux-1?');
                const said = content('linux-1 and everything on its disk are deleted for good.');
                const held = content(
                    'It is held by its lot, by studio-os-fe: work may be in progress.',
                );
                await visitor.see(within(asked, said));
                await visitor.see(within(asked, held));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
