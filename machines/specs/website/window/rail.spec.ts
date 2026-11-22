import { button, content, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const rail = navigation('Rail');
const sidebar = navigation('Sidebar');
const AGENT_7F = heading('agent-7f running');

describe('rail', () => {
    test("A10 the rail's Inbox opens beside the Machine space's sidebar, and Machine returns where I was", async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website.visit('/?world=stories&machine=agent-7f', async (visitor) => {
            await visitor.see(AGENT_7F);

            // When - the rail's Inbox is pressed, then its Machine
            await visitor.click(within(rail, button('Inbox 2')));
            await visitor.see(within(sidebar, heading('Machine')));
            await visitor.gone(AGENT_7F);
            await visitor.click(within(rail, button('Machine')));

            // Then - agent-7f again
            await visitor.see(AGENT_7F);
        });
        expect(result.errors.text).toBe('');
    });

    test('A10 ⌘I opens Inbox, ⌃1 returns to the Machine space, and Agent says soon', async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website.visit('/?world=stories&machine=agent-7f', async (visitor) => {
            await visitor.see(AGENT_7F);

            // When - ⌘I, then ⌃1, then Agent is hovered
            await visitor.press('Meta+i');
            await visitor.gone(AGENT_7F);
            await visitor.press('Control+1');
            await visitor.see(AGENT_7F);
            await visitor.hover(within(rail, button('Agent soon')));

            // Then - agent-7f stays, and Agent says what it will hold
            await visitor.see(AGENT_7F);
            await visitor.see(content('The workforce: personalities, roles, focus'));
        });
        expect(result.errors.text).toBe('');
    });

    test("A11 the rail's Settings opens the app's Settings, apart from the catalog's", async () => {
        // Given - the proof build on Home over stories
        const result = await website.visit('/?world=stories&page=home', async (visitor) => {
            await visitor.see(content('2 things need you.'));

            // When - the rail's Settings is pressed
            await visitor.click(within(rail, button('Settings')));

            // Then - the Engine panel, and the catalog's shelf still named Machine settings
            await visitor.see(within(region('Engine'), content('Engine running · v4.0.0')));
            await visitor.see(within(sidebar, button('Machine settings', { exact: false })));
        });
        expect(result.errors.text).toBe('');
    });
});
