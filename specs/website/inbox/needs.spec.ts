import { button, content, dialog, heading, main, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const needs = region('Needs you');
const canWait = region('Can wait');

describe('needs', () => {
    test('N1 a line settled by its verb leaves, and the sidebar counts one less', async () => {
        // Given - the window on Home over engine-stopped, whose one line that needs me is the stopped engine
        const start = button('Start the engine', { exact: false });
        const result = await website.visit('/?world=engine-stopped&page=home', async (visitor) => {
            await visitor.see(within(sidebar, button('Inbox 1')));
            await visitor.see(heading('Needs you 1'));
            await visitor.see(heading('The engine is stopped.'));

            // When - its verb, Start the engine, is pressed
            await visitor.click(within(needs, start));

            // Then - the line has left, and the sidebar no longer counts it
            await visitor.gone(within(needs, start));
            await visitor.see(within(sidebar, button('Inbox')));
        });
        expect(result.errors.text).toBe('');
    });

    test('N2 the lines that press and those that can wait are told apart in two panels, by the engine', async () => {
        // Given - the Inbox over two-macs, one line that needs me beside seven that can wait
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            // When - I open Needs you
            // Then - two panels, each with the lines the engine's upkeep puts in it
            await visitor.see(
                within(needs, content('com.jterrazz.brain.x-snapshot is failing, last exit 2')),
            );
            await visitor.see(within(canWait, content('1 mise update waiting')));
            await visitor.gone(
                within(canWait, content('com.jterrazz.brain.x-snapshot is failing, last exit 2')),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('N3 waits for a recording of an exposed port and the Review verb of its line');

    test('N4 a program held by a grant says which password holds it, gives no end, and opens that password', async () => {
        // Given - linux-1's Sign-ins over two-macs, npm held by a grant of npm publish
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Sign-ins',
            async (visitor) => {
                const rows = region('Sign-ins');

                // When - the row is read
                // Then - it names the password, and its link opens that password's page
                await visitor.see(
                    within(rows, content('from the password npm publish', { exact: false })),
                );
                await visitor.click(within(rows, button('Open the password npm publish')));
                await visitor.see(heading('npm publish'));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('until');
    });

    test.todo("N5 waits for each line's version rule on machinefile.drift");

    test.todo(
        'N6 waits for a recording of an idle VM among the needs, and a kept journal for Changes',
    );

    test('N7 a program signed out offers Sign in, and its sheet shows the engine code and the site to open', async () => {
        // Given - linux-1's Sign-ins over two-macs, gh signed out
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Sign-ins&pace=600',
            async (visitor) => {
                // When - Sign in of gh is pressed
                await visitor.click(within(region('Sign-ins'), button('Sign in gh')));

                // Then - the sheet shows the code to copy and the site to open
                const sheet = dialog('Sign in to gh on linux-1');
                await visitor.see(within(sheet, content('WDJB-MJHT')));
                await visitor.see(within(sheet, button('Copy code')));
                await visitor.see(within(sheet, button('Open github.com')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('N8 an empty Needs you says no need is listed, and draws no table', async () => {
        // Given - the Inbox over one-mac, where no line waits
        const result = await website.visit('/?world=one-mac&page=inbox', async (visitor) => {
            // When - Needs you is read
            await visitor.see(needs);

            // Then - it says so in a line, with no button to press
            await visitor.see(within(needs, content('No need is listed.')));
            await visitor.gone(within(needs, button('Update mise')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo("N9 waits for the engine's kept journal (drawn on a provisional copy)");

    test("N10 a Mac chosen above the list keeps only that Mac's needs, and All brings them back", async () => {
        // Given - the Inbox over two-macs, where every need is on the first Mac
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            await visitor.see(
                within(needs, content('com.jterrazz.brain.x-snapshot is failing, last exit 2')),
            );

            // When - the Mac mini is chosen
            await visitor.click(within(main(), button('jterrazz-mac-mini')));

            // Then - none of the needs concerns it
            await visitor.see(
                content('None of the needs listed concerns jterrazz-mac-mini or its machines.'),
            );
            await visitor.gone(
                within(needs, content('com.jterrazz.brain.x-snapshot is failing, last exit 2')),
            );

            // When - all machines are chosen again
            await visitor.click(within(main(), button('All machines')));

            // Then - the needs are back
            await visitor.see(
                within(needs, content('com.jterrazz.brain.x-snapshot is failing, last exit 2')),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test.todo(
        "N10 waits for the engine's kept journal, read for one Mac alone (drawn on a provisional copy)",
    );

    test('N11 the sidebar holds Home, Inbox, Storage, the machines, Machine files and the Catalog', async () => {
        // Given - the window over two-macs
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            // When - I read the sidebar
            await visitor.see(sidebar);

            // Then - the fleet's places are there, Inbox with its count of the line that needs me
            await visitor.see(within(sidebar, button('Home')));
            await visitor.see(within(sidebar, button('Inbox 1')));
            await visitor.see(within(sidebar, button('Storage')));
            await visitor.see(within(sidebar, button('jterrazz-macbook-pro online')));
            await visitor.see(within(sidebar, button('api Stopped')));
            await visitor.see(within(sidebar, button('linux-1 Running')));
            await visitor.see(within(sidebar, button('jterrazz-mac-mini online')));
            await visitor.see(within(sidebar, button('infra Running')));
            await visitor.see(within(sidebar, button('Machine files')));
            await visitor.see(within(sidebar, button('Apps')));
            await visitor.see(within(sidebar, button('Skills')));
            await visitor.see(within(sidebar, button('Settings')));
        });
        expect(result.errors.text).toBe('');
    });
});
