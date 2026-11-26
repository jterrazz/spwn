import {
    banner,
    button,
    content,
    dialog,
    focused,
    heading,
    navigation,
    option,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { PINNED, website } from '../website.specification.js';

const bar = banner();
const path = navigation('Path');
const spaces = navigation('Spaces');
const column = navigation('Column');
const AGENT_7F = heading('agent-7f running');
const SPACES_LIST = dialog('Spaces');

// value: the words the bar's Spaces navigation draws pressed in the page as last drawn: one, or none.
function pressedSpaces(html: string): string[] {
    const at = html.indexOf('aria-label="Spaces"');
    const nav = html.slice(at, html.indexOf('</nav>', at));
    return [
        ...nav.matchAll(/<button[^>]*aria-pressed="true"[^>]*><span[^>]*>(?<word>[^<]*)</gu),
    ].map((pressed) => pressed.groups?.word ?? '');
}

describe('path bar', () => {
    test('A10 ⌘I opens Inbox beside the column, Machine still pressed, and ⌘1 returns to the Machine space where I was', async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&machine=agent-7f', async (visitor) => {
                await visitor.see(AGENT_7F);

                // When - ⌘I, then ⌘1
                await visitor.press('Meta+i');
                await visitor.gone(AGENT_7F);
                await visitor.see(within(path, content('Inbox')));
                await visitor.see(column);
                await visitor.press('Meta+1');

                // Then - agent-7f again
                await visitor.see(AGENT_7F);
            });
        expect(pressedSpaces(result.html.text)).toStrictEqual(['Machine']);
        expect(result.errors.text).toBe('');
    });

    test("A10 the bar's Machine returns from Inbox to the page I left", async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&machine=agent-7f', async (visitor) => {
                await visitor.see(AGENT_7F);

                // When - the bar's Inbox, then its Machine
                await visitor.click(within(bar, button('Inbox', { exact: false })));
                await visitor.see(within(path, content('Inbox')));
                await visitor.click(within(spaces, button('Machine')));

                // Then - agent-7f again
                await visitor.see(AGENT_7F);
            });
        expect(result.errors.text).toBe('');
    });

    test("A11 the bar's Settings opens the app's Settings alone under the bar, apart from the catalog's", async () => {
        // Given - the proof build on Home over stories
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&page=home', async (visitor) => {
                await visitor.see(content('2 things need you.'));

                // When - the bar's Settings is pressed
                await visitor.click(within(bar, button('Settings')));

                // Then - the Engine panel, with no Machine space's column and its Machine settings
                await visitor.see(within(region('Engine'), content('Engine running · v4.0.0')));
                await visitor.gone(column);
                await visitor.gone(button('Machine settings', { exact: false }));
            });
        expect(result.errors.text).toBe('');
    });

    test("A18 a VM on another Mac is two presses away, through its Mac's segment", async () => {
        // Given - the proof build on linux-1 of macbook-pro over two-macs
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&machine=linux-1&tab=Overview', async (visitor) => {
                await visitor.see(heading('linux-1 running'));

                // When - the Mac's segment is pressed, then infra, a VM of mac-mini
                await visitor.click(within(path, button('macbook-pro', { exact: false })));
                await visitor.see(dialog('Machines'));
                await visitor.click(option('infra', { exact: false }));

                // Then - infra opens, the path reads Fleet, its Mac, then it
                await visitor.see(heading('infra healthy'));
                await visitor.see(within(path, button('Fleet')));
                await visitor.see(within(path, button('mac-mini', { exact: false })));
                await visitor.see(within(path, button('infra', { exact: false })));
                await visitor.gone(within(path, button('macbook-pro', { exact: false })));
            });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'A19 waits for a viewport: a website visit takes none, and PathNav proves the fold at 640 px',
    );

    test('A24 the spaces are three words before the path, and Agent opens the spaces list, moving nothing', async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&machine=agent-7f', async (visitor) => {
                await visitor.see(AGENT_7F);
                await visitor.see(within(spaces, button('Machine')));
                await visitor.gone(within(path, button('Machine', { exact: false })));

                // When - the bar's Agent is pressed
                await visitor.click(within(spaces, button('Agent soon')));

                // Then - Machine, then Agent and OS greyed with Soon and what each will hold, and agent-7f still open
                await visitor.see(
                    within(
                        SPACES_LIST,
                        content('Soon · The workforce: personalities, roles, focus'),
                    ),
                );
                await visitor.see(within(SPACES_LIST, content('Soon · Flows, projects, vision')));
                await visitor.see(
                    within(SPACES_LIST, content('The fleet, access, security, documents')),
                );
                await visitor.see(AGENT_7F);
            });
        expect(pressedSpaces(result.html.text)).toStrictEqual(['Machine']);
        expect(result.errors.text).toBe('');
    });

    test('A24 ⌘3 opens the spaces list from OS, and Escape closes it on the same page', async () => {
        // Given - the proof build on Storage over stories
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&page=storage', async (visitor) => {
                await visitor.see(within(path, button('Storage')));

                // When - ⌘3, then Escape
                await visitor.press('Meta+3');
                await visitor.see(within(SPACES_LIST, content('Soon · Flows, projects, vision')));
                await visitor.press('Escape');

                // Then - the list closes, the focus on OS, Storage still open
                await visitor.gone(SPACES_LIST);
                const os = within(spaces, button('OS soon'));
                await visitor.see(focused(os));
                await visitor.see(within(path, button('Storage')));
            });
        expect(pressedSpaces(result.html.text)).toStrictEqual(['Machine']);
        expect(result.errors.text).toBe('');
    });

    test('A24 on Settings, above the spaces, no space is pressed', async () => {
        // Given - the proof build on agent-7f over stories
        const result = await website
            .storage(PINNED)
            .visit('/?world=stories&machine=agent-7f', async (visitor) => {
                await visitor.see(AGENT_7F);

                // When - ⌘,
                await visitor.press('Meta+,');

                // Then - Settings, and the three words still drawn
                await visitor.see(within(path, content('Settings')));
                await visitor.see(within(spaces, button('Machine')));
            });
        expect(pressedSpaces(result.html.text)).toStrictEqual([]);
        expect(result.errors.text).toBe('');
    });
});
