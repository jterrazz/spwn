import {
    button,
    component,
    content,
    dialog,
    disabled,
    field,
    heading,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { recordingsOf } from '../../../fixtures/recordings';
import { machineRoute } from '../../../shell/route';
import type { Route } from '../../../shell/route';
import { Window } from '../../../shell/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('one-mac', 'two-macs', 'mini-offline');

const onThisMac = dialog('New machine on jterrazz-macbook-pro');
const thisMac = button('jterrazz-macbook-pro');
const onMini = dialog('New machine on jterrazz-mac-mini');

const HOME: Route = { page: 'home' };

function windowOver(world: string, initial: Route = HOME) {
    const client = new Client(replayTransport(recordingFor(world)));
    return (
        <EngineProvider client={client}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('New machine', () => {
    test('opened from Home, the sheet is on the Mac the engine names, with its presets and lifetimes', async () => {
        // Given - the window over two-macs, where the mini has the most room
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            // When - New machine is opened from Home
            await visitor.click(button('New machine'));

            // Then - the sheet is on the mini, offers both Macs, the presets the engine judges and both lifetimes
            await visitor.see(within(onMini, field('Name')));
            await visitor.see(within(region('On'), thisMac));
            await visitor.see(within(onMini, button('8 GB · workspace')));
            await visitor.see(within(onMini, button('persistent')));
            await visitor.see(within(onMini, button('ephemeral')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a workspace on This Mac is launched there, and the sheet hands over to its page', async () => {
        // Given - the New machine sheet over two-macs, opened on the mini
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));
            await visitor.see(onMini);

            // When - This Mac is picked, gate5 named ephemeral, then launched and confirmed
            await visitor.click(within(region('On'), thisMac));
            await visitor.see(onThisMac);
            await visitor.fill(within(onThisMac, field('Name')), 'gate5');
            await visitor.click(within(onThisMac, button('ephemeral')));
            await visitor.click(within(onThisMac, button('Launch')));
            await visitor.click(within(dialog('Launch gate5?'), button('Launch')));

            // Then - the sheet is gone and gate5's page is open
            await visitor.gone(onThisMac);
            await visitor.see(heading('gate5 does not answer.'));
        });
        expect(result.errors.text).toBe('');
    });

    test("from a Mac's page the create is sent to that Mac, and a refusal naming a Mac is drawn on the On row", async () => {
        // Given - the mini's page over two-macs
        const result = await component.render(
            windowOver('two-macs', machineRoute('jterrazz-mac-mini')),
            async (visitor) => {
                // When - its New machine opens the sheet on it, and gate5 of 8 GB is launched there
                await visitor.click(button('New machine on jterrazz-mac-mini'));
                await visitor.fill(within(onMini, field('Name')), 'gate5');
                await visitor.click(within(onMini, button('8 GB · workspace')));
                await visitor.click(within(onMini, button('Launch')));
                await visitor.click(within(dialog('Launch gate5?'), button('Launch')));

                // Then - the engine's answer to the mini's create names a Mac, so it stands on the On row
                const said = content('not enough memory on jterrazz-macbook-pro', { exact: false });
                await visitor.see(within(region('On'), said));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a name the engine finds taken is refused in the foot of the sheet, in its words', async () => {
        // Given - the New machine sheet over two-macs, where linux-1 exists
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));

            // When - linux-1 is launched on the mini
            await visitor.fill(within(onMini, field('Name')), 'linux-1');
            await visitor.click(within(onMini, button('Launch')));
            await visitor.click(within(dialog('Launch linux-1?'), button('Launch')));

            // Then - the sheet stays, with the engine's refusal, off the On row
            const said = content('"linux-1" is taken (on jterrazz-macbook-pro)', { exact: false });
            await visitor.see(within(onMini, said));
            await visitor.gone(within(region('On'), said));
        });
        expect(result.errors.text).toBe('');
    });

    test('a purpose the engine refuses is drawn under the purpose field', async () => {
        // Given - the New machine sheet over two-macs
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));

            // When - a purpose of 121 characters is launched
            await visitor.fill(within(onMini, field('What is it for?')), 'x'.repeat(121));
            await visitor.click(within(onMini, button('Launch')));
            await visitor.click(within(dialog('Launch linux-2?'), button('Launch')));

            // Then - the engine's words stand under the purpose
            const said = content('a purpose is one line of at most 120 characters', {
                exact: false,
            });
            await visitor.see(within(region('Purpose'), said));
        });
        expect(result.errors.text).toBe('');
    });

    test('a server is asked of machines.create-server with the image the engine names', async () => {
        // Given - the New machine sheet over two-macs, on This Mac
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));
            await visitor.click(within(region('On'), thisMac));

            // When - the kind is turned to Server under Options, web named and launched
            await visitor.click(within(onThisMac, button('Options')));
            await visitor.click(within(onThisMac, button('Server')));
            await visitor.gone(within(onThisMac, button('persistent')));
            await visitor.fill(within(onThisMac, field('Name')), 'web');
            await visitor.click(within(onThisMac, button('Launch')));
            await visitor.click(within(dialog('Launch web?'), button('Launch')));

            // Then - the sheet hands over to web's page
            await visitor.gone(onThisMac);
            await visitor.see(heading('web does not answer.'));
        });
        expect(result.errors.text).toBe('');
    });

    test("with one Mac there is no On row, and a Mac the engine finds not ready holds Launch with the engine's words", async () => {
        // Given - the window over one-mac, whose Mac has said no memory
        const result = await component.render(windowOver('one-mac'), async (visitor) => {
            // When - New machine is opened
            await visitor.click(button('New machine'));

            // Then - no On row, the engine's sentence stands under Memory, and Launch cannot be pressed
            await visitor.gone(region('On'));
            await visitor.see(within(onThisMac, content('This Mac has not said its memory')));
            const held = disabled(button('Launch'));
            await visitor.see(within(onThisMac, held));
        });
        expect(result.errors.text).toBe('');
    });

    test('a purpose the engine refuses is drawn under the purpose field, in its own sentence', async () => {
        // Given - the New machine sheet over two-macs, on the mini
        const purpose = 'x'.repeat(121);
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(button('New machine'));

            // When - a purpose of 121 characters is launched
            await visitor.fill(within(onMini, field('What is it for?')), purpose);
            await visitor.click(within(onMini, button('Launch')));
            await visitor.click(within(dialog('Launch linux-2?'), button('Launch')));

            // Then - the engine's sentence stands under the purpose
            const said = content('a purpose is one line of at most 120 characters', {
                exact: false,
            });
            await visitor.see(within(region('Purpose'), said));
        });
        expect(result.errors.text).toBe('');
    });

    test('a Mac that does not answer is offered with what the engine says of it, and cannot be picked', async () => {
        // Given - the window over mini-offline
        const result = await component.render(windowOver('mini-offline'), async (visitor) => {
            // When - New machine is opened
            await visitor.click(button('New machine'));

            // Then - the sheet is on This Mac, and the mini's card is shut with the engine's sentence
            await visitor.see(onThisMac);
            const mini = disabled(button('jterrazz-mac-mini'));
            await visitor.see(within(region('On'), mini));
            await visitor.see(within(region('On'), content('mac-mini does not answer')));
        });
        expect(result.errors.text).toBe('');
    });
});
