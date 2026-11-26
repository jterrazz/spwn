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

import { composeReplay } from '../../../../composition/compose';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../../dependencies/dependencies';
import { machineRoute } from '../../../navigation/route';
import type { Route } from '../../../navigation/route';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('one-mac', 'two-macs', 'mini-offline');

const onThisMac = dialog('New machine on macbook-pro');
const thisMac = button('macbook-pro');
const onMini = dialog('New machine on mac-mini');
const NEW_MACHINE = button('New machine');

const HOME: Route = { page: 'home' };

function windowOver(world: string, initial: Route = HOME) {
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('New machine', () => {
    test('opened from Home, the sheet is on the Mac the engine names, with its presets and lifetimes', async () => {
        // Given - the window over two-macs, where the mini has the most room
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            // When - New machine is opened from Home
            await visitor.click(NEW_MACHINE);

            // Then - the sheet is on the mini, offers both Macs, the presets the engine judges and both lifetimes
            await visitor.see(within(onMini, field('Name')));
            await visitor.see(within(region('On'), thisMac));
            await visitor.see(within(onMini, button('8 GB · workspace')));
            await visitor.see(within(onMini, button('persistent')));
            await visitor.see(within(onMini, button('ephemeral')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a workspace on This Mac is launched there, and the sheet hands over to the Guide of its act, whose action opens its page', async () => {
        // Given - the New machine sheet over two-macs, opened on the mini
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(NEW_MACHINE);
            await visitor.see(onMini);

            // When - This Mac is picked, gate5 named ephemeral, then launched, which asks nothing
            await visitor.click(within(region('On'), thisMac));
            await visitor.see(onThisMac);
            await visitor.fill(within(onThisMac, field('Name')), 'gate5');
            await visitor.click(within(onThisMac, button('ephemeral')));
            await visitor.click(within(onThisMac, button('Launch')));

            // Then - the sheet is gone and gate5's Guide reads it made, its action opening gate5's page
            await visitor.gone(onThisMac);
            const guide = dialog('New machine gate5');
            await visitor.see(within(guide, heading('gate5 is ready. Open it in your terminal.')));
            await visitor.click(within(guide, button('Open gate5')));
            await visitor.see(heading('gate5 running'));
        });
        expect(result.errors.text).toBe('');
    });

    test("from a Mac's page the create is sent to that Mac, and the sheet hands over to the machine it made there, which no Guide follows", async () => {
        // Given - the mini's page over two-macs
        const result = await component.render(
            windowOver('two-macs', machineRoute('mac-mini')),
            async (visitor) => {
                // When - its New machine opens the sheet on it, and gate5 of 8 GB is launched there
                await visitor.click(button('New machine on mac-mini'));
                await visitor.fill(within(onMini, field('Name')), 'gate5');
                await visitor.click(within(onMini, button('8 GB · workspace')));
                await visitor.click(within(onMini, button('Launch')));

                // Then - the mini's engine made it, the sheet is gone and gate5's page is open
                await visitor.gone(onMini);
                await visitor.see(heading('gate5 running'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a name the engine finds taken is refused under the name field, in its words', async () => {
        // Given - the New machine sheet over two-macs, where the mini holds infra
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(NEW_MACHINE);

            // When - infra is launched on the mini
            await visitor.fill(within(onMini, field('Name')), 'infra');
            await visitor.click(within(onMini, button('Launch')));

            // Then - the sheet stays, with the engine's refusal under the name, off the On row
            const said = content('"infra" is taken (on mac-mini)', { exact: false });
            await visitor.see(within(region('What it is called'), said));
            await visitor.gone(within(region('On'), said));
        });
        expect(result.errors.text).toBe('');
    });

    test("a name the template's chain already bears is refused under the name field, in the engine's sentence", async () => {
        // Given - the New machine sheet over two-macs, following base-ubuntu
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(NEW_MACHINE);

            // When - base-ubuntu is launched on the mini
            await visitor.fill(within(onMini, field('Name')), 'base-ubuntu');
            await visitor.click(within(onMini, button('Launch')));

            // Then - the engine's sentence stands under the name
            const said = content("base-ubuntu names a Machinefile of base-ubuntu's chain", {
                exact: false,
            });
            await visitor.see(within(region('What it is called'), said));
        });
        expect(result.errors.text).toBe('');
    });

    test('a name the engine finds malformed is refused under the name field', async () => {
        // Given - the New machine sheet over two-macs
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(NEW_MACHINE);

            // When - Bad_Name is launched on the mini
            await visitor.fill(within(onMini, field('Name')), 'Bad_Name');
            await visitor.click(within(onMini, button('Launch')));

            // Then - the engine's sentence stands under the name
            const said = content('"Bad_Name" is not a valid name', { exact: false });
            await visitor.see(within(region('What it is called'), said));
        });
        expect(result.errors.text).toBe('');
    });

    test('a purpose the engine refuses is drawn under the purpose field', async () => {
        // Given - the New machine sheet over two-macs
        const result = await component.render(windowOver('two-macs'), async (visitor) => {
            await visitor.click(NEW_MACHINE);

            // When - a purpose of 121 characters is launched
            await visitor.fill(within(onMini, field('What is it for?')), 'x'.repeat(121));
            await visitor.click(within(onMini, button('Launch')));

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
            await visitor.click(NEW_MACHINE);
            await visitor.click(within(region('On'), thisMac));

            // When - the kind is turned to Server under Options, web named and launched
            await visitor.click(within(onThisMac, button('Options')));
            await visitor.click(within(onThisMac, button('Server')));
            await visitor.gone(within(onThisMac, button('persistent')));
            await visitor.fill(within(onThisMac, field('Name')), 'web');
            await visitor.click(within(onThisMac, button('Launch')));

            // Then - the sheet hands over to web's page
            await visitor.gone(onThisMac);
            await visitor.see(heading('web healthy'));
        });
        expect(result.errors.text).toBe('');
    });

    test("with one Mac there is no On row, and a Mac the engine finds not ready holds Launch with the engine's words", async () => {
        // Given - the window over one-mac, whose Mac has said no memory
        const result = await component.render(windowOver('one-mac'), async (visitor) => {
            // When - New machine is opened
            await visitor.click(NEW_MACHINE);

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
            await visitor.click(NEW_MACHINE);

            // When - a purpose of 121 characters is launched
            await visitor.fill(within(onMini, field('What is it for?')), purpose);
            await visitor.click(within(onMini, button('Launch')));

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
            await visitor.click(NEW_MACHINE);

            // Then - the sheet is on This Mac, and the mini's card is shut with the engine's sentence
            await visitor.see(onThisMac);
            const mini = disabled(button('mac-mini'));
            await visitor.see(within(region('On'), mini));
            await visitor.see(within(region('On'), content('mac-mini does not answer')));
        });
        expect(result.errors.text).toBe('');
    });
});
