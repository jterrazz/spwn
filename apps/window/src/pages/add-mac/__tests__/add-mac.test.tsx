import { button, component, content, dialog, disabled, field, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { recordingsOf } from '../../../fixtures/recordings';
import { Window } from '../../../shell/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('one-mac', 'two-macs');

const sheet = dialog('Add a Mac');

function windowOver(world: string) {
    const client = new Client(replayTransport(recordingFor(world)));
    return (
        <EngineProvider client={client}>
            <Window initial={{ page: 'home' }} pages={PAGES} />
        </EngineProvider>
    );
}

describe('AddMac', () => {
    test('a Mac added from Home opens its page once the engine answers', async () => {
        // Given - the window over one-mac
        const result = await component.render(windowOver('one-mac'), async (visitor) => {
            // When - Add a Mac is opened under the Macs, a name typed, then added
            await visitor.click(button('Add a Mac'));
            const shut = disabled(button('Add'));
            await visitor.see(within(sheet, shut));
            await visitor.fill(within(sheet, field('Its Tailscale name')), 'jterrazz-mac-mini');
            await visitor.click(within(sheet, button('Add')));

            // Then - the dialog closes on the mini's page
            await visitor.gone(sheet);
            await visitor.see(button('New machine on jterrazz-mac-mini'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a Mac already known is refused in the dialog, in the engine words', async () => {
        // Given - the window over two-macs, which knows the mini
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            // When - the mini is added again
            await visitor.click(button('Add a Mac'));
            await visitor.fill(within(sheet, field('Its Tailscale name')), 'jterrazz-mac-mini');
            await visitor.click(within(sheet, button('Add')));

            // Then - the dialog stays with the refusal
            const said = content('machine "jterrazz-mac-mini" already exists', { exact: false });
            await visitor.see(within(sheet, said));
        });
        expect(result.errors.text).toBe('');
    });
});
