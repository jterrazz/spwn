import {
    banner,
    button,
    content,
    dialog,
    heading,
    navigation,
    option,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const bar = banner();
const path = navigation('Path');
const sidebar = navigation('Sidebar');
const AGENT_7F = heading('agent-7f running');

describe('path bar', () => {
    test('A10 the space segment lists the three spaces, and Agent and OS say soon and open nothing', async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website.visit('/?world=stories&machine=agent-7f', async (visitor) => {
            await visitor.see(AGENT_7F);

            // When - the path's space segment is pressed
            await visitor.click(within(path, button('Machine', { exact: false })));

            // Then - Machine, then Agent and OS greyed with Soon and what each will hold
            const spaces = dialog('Spaces');
            await visitor.see(
                within(spaces, content('Soon · The workforce: personalities, roles, focus')),
            );
            await visitor.see(within(spaces, content('Soon · Flows, projects, vision')));
            await visitor.see(within(spaces, content('The fleet, access, security, documents')));
            await visitor.see(AGENT_7F);
        });
        expect(result.errors.text).toBe('');
    });

    test('A10 ⌘I opens Inbox beside the sidebar, and ⌘1 returns to the Machine space where I was', async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website.visit('/?world=stories&machine=agent-7f', async (visitor) => {
            await visitor.see(AGENT_7F);

            // When - ⌘I, then ⌘1
            await visitor.press('Meta+i');
            await visitor.gone(AGENT_7F);
            await visitor.see(within(path, content('Inbox')));
            await visitor.see(sidebar);
            await visitor.press('Meta+1');

            // Then - agent-7f again
            await visitor.see(AGENT_7F);
        });
        expect(result.errors.text).toBe('');
    });

    test("A11 the bar's Settings opens the app's Settings alone under the bar, apart from the catalog's", async () => {
        // Given - the proof build on Home over stories
        const result = await website.visit('/?world=stories&page=home', async (visitor) => {
            await visitor.see(content('2 things need you.'));

            // When - the bar's Settings is pressed
            await visitor.click(within(bar, button('Settings')));

            // Then - the Engine panel, with no Machine space's sidebar and its Machine settings
            await visitor.see(within(region('Engine'), content('Engine running · v4.0.0')));
            await visitor.gone(sidebar);
            await visitor.gone(button('Machine settings', { exact: false }));
        });
        expect(result.errors.text).toBe('');
    });

    test("A18 a VM on another Mac is two presses away, through its Mac's segment", async () => {
        // Given - the proof build on linux-1 of macbook-pro over two-macs
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Overview',
            async (visitor) => {
                await visitor.see(heading('linux-1 running'));

                // When - the Mac's segment is pressed, then infra, a VM of mac-mini
                await visitor.click(within(path, button('macbook-pro', { exact: false })));
                await visitor.see(dialog('Machines'));
                await visitor.click(option('infra', { exact: false }));

                // Then - infra opens, the path reads its Mac, then it
                await visitor.see(heading('infra healthy'));
                await visitor.see(within(path, button('mac-mini', { exact: false })));
                await visitor.see(within(path, button('infra', { exact: false })));
                await visitor.gone(within(path, button('macbook-pro', { exact: false })));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'A19 waits for a viewport: a website visit takes none, and PathNav proves the fold at 560 px',
    );
});
