import { button, component, content, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { EngineProvider } from '../../dependencies/dependencies';
import { PAGES } from '../../screens/registry';
import { machineRoute } from '../route';
import type { Route } from '../route';
import { Window } from '../window';

const recordingFor = await recordingsOf('stories', 'two-macs');

const rail = navigation('Rail');
const sidebar = navigation('Sidebar');
const machine = button('Machine');
const AGENT_7F = heading('agent-7f running');
const TOGGLE = button('Show or hide the sidebar');

function windowOn(initial: Route, world = 'stories') {
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('Rail', () => {
    test("the rail's Engine line opens the Engine page, with no release out", async () => {
        // Given - the window on Home over stories, whose engine is not behind
        const result = await component.render(windowOn({ page: 'home' }), async (visitor) => {
            // When - the rail's Engine line is pressed
            await visitor.click(within(rail, button('Engine running · v4.0.0')));

            // Then - the Engine page, its head as the engine sent it
            await visitor.see(heading('The latest release is not known yet.'));
        });
        expect(result.errors.text).toBe('');
    });

    test("the rail's Inbox opens with its count while the Machine space's sidebar stays, and Machine returns where it was", async () => {
        // Given - the window on agent-7f over stories, two lines needing me
        const result = await component.render(
            windowOn(machineRoute('agent-7f')),
            async (visitor) => {
                await visitor.see(AGENT_7F);

                // When - the rail's Inbox is pressed
                await visitor.click(within(rail, button('Inbox 2')));

                // Then - Inbox, the sidebar still the Machine space's
                await visitor.see(within(sidebar, heading('Machine')));
                await visitor.see(within(sidebar, button('Apps', { exact: false })));
                await visitor.gone(AGENT_7F);

                // When - the Machine segment is pressed
                await visitor.click(within(rail, machine));

                // Then - agent-7f again
                await visitor.see(AGENT_7F);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('the Agent and OS segments are named soon and say what they will hold', async () => {
        // Given - the window on Home over stories
        const result = await component.render(windowOn({ page: 'home' }), async (visitor) => {
            await visitor.see(content('2 things need you.'));

            // When - Agent, then OS, is hovered
            await visitor.hover(within(rail, button('Agent soon')));
            await visitor.see(content('The workforce: personalities, roles, focus'));
            await visitor.hover(within(rail, button('OS soon')));

            // Then - OS says what it will hold, and Home stays
            await visitor.see(content('Flows, projects, vision'));
            await visitor.see(content('2 things need you.'));
        });
        expect(result.errors.text).toBe('');
    });

    test("⌘I opens Inbox, ⌃1 the Machine space where it was, ⌃2 nothing yet; the rail's Settings opens Settings", async () => {
        // Given - the window on agent-7f over stories
        const result = await component.render(
            windowOn(machineRoute('agent-7f')),
            async (visitor) => {
                await visitor.see(AGENT_7F);

                // When - ⌘I, then ⌃2, then ⌃1
                await visitor.press('Meta>}i{/Meta');
                await visitor.gone(AGENT_7F);
                await visitor.press('Control>}2{/Control');
                await visitor.gone(AGENT_7F);
                await visitor.press('Control>}1{/Control');

                // Then - agent-7f again, and Settings from the rail
                await visitor.see(AGENT_7F);
                await visitor.click(within(rail, button('Settings')));
                await visitor.see(within(region('Engine'), content('Engine running · v4.0.0')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("the rail's foot opens Passwords, Storage and Settings, pages of the whole window with no space's sidebar", async () => {
        // Given - the window on agent-7f over two-macs
        const result = await component.render(
            windowOn(machineRoute('linux-1'), 'two-macs'),
            async (visitor) => {
                await visitor.see(heading('linux-1 running'));
                await visitor.see(sidebar);

                // When - the rail's Passwords is pressed
                await visitor.click(within(rail, button('Passwords')));

                // Then - the vault, with no sidebar and no toggle for it
                await visitor.see(heading('Passwords on macbook-pro'));
                await visitor.gone(sidebar);
                await visitor.gone(TOGGLE);

                // When - the rail's Storage, then its Settings, is pressed
                await visitor.click(within(rail, button('Storage')));
                await visitor.see(heading('3.0 GB can be freed.'));
                await visitor.gone(sidebar);
                await visitor.click(within(rail, button('Settings')));

                // Then - Settings with no sidebar, and Machine brings the sidebar back where it was
                await visitor.see(region('Engine'));
                await visitor.gone(sidebar);
                await visitor.click(within(rail, machine));
                await visitor.see(heading('linux-1 running'));
                await visitor.see(within(sidebar, heading('Machine')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("the Machine space's sidebar holds the machines and its Catalog, Machine files among it, and no global page", async () => {
        // Given - the window on Home over two-macs
        const result = await component.render(
            windowOn({ page: 'home' }, 'two-macs'),
            async (visitor) => {
                await visitor.see(sidebar);

                // When - the Catalog's Machine files is pressed
                await visitor.see(within(sidebar, button('Apps 10')));
                await visitor.see(within(sidebar, button('Machine settings 3')));
                await visitor.click(within(sidebar, button('Machine files')));

                // Then - Machine files beside the sidebar, which names neither Passwords nor Storage
                await visitor.see(within(sidebar, heading('Machine')));
                await visitor.gone(within(sidebar, button('Passwords')));
                await visitor.gone(within(sidebar, button('Storage')));
                await visitor.see(within(rail, button('Passwords')));
                await visitor.see(within(rail, button('Storage')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("the sidebar's toggle hides and shows the space's sidebar", async () => {
        // Given - the window on Home over stories
        const result = await component.render(windowOn({ page: 'home' }), async (visitor) => {
            await visitor.see(sidebar);

            // When - the toggle is pressed, then pressed again
            await visitor.click(TOGGLE);
            await visitor.gone(sidebar);
            await visitor.click(TOGGLE);

            // Then - the sidebar is back
            await visitor.see(within(sidebar, heading('Machine')));
        });
        expect(result.errors.text).toBe('');
    });
});
