import { button, content, dialog, heading, main, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const needs = region('Needs you');
const canWait = region('Upkeep');

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
                within(needs, content('com.example.brain.x-snapshot is failing, last exit 2')),
            );
            await visitor.see(within(canWait, content('1 mise update waiting')));
            await visitor.gone(
                within(canWait, content('com.example.brain.x-snapshot is failing, last exit 2')),
            );

            // Then - each line's pill is its Mark's word: Needs you for the failing task, Upkeep for the update
            await visitor.see(
                within(
                    needs,
                    button('Needs you com.example.brain.x-snapshot is failing', { exact: false }),
                ),
            );
            await visitor.see(
                within(canWait, button('Upkeep 1 mise update waiting', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test("N3 a port reached outside Tailscale opens its Mac's Ports, where declaring it public settles its line", async () => {
        // Given - the Inbox over stories, where ControlCenter listens on macbook-pro's port 5000, reached outside Tailscale and declared by no one
        const line = button('broken port 5000 is reachable outside Tailscale macbook-pro · Ports');
        const result = await website.visit('/?world=stories&page=inbox', async (visitor) => {
            await visitor.see(within(needs, line));
            await visitor.see(within(needs, content('Review')));

            // When - its line is pressed
            await visitor.click(within(needs, line));

            // Then - the Mac's Ports tab opens on the line, above the port it names and its declaration
            const declare = within(
                region('What listens on this Mac'),
                button('Declare :5000 public'),
            );
            await visitor.see(within(region('Needs you, on its ports'), line));
            await visitor.see(declare);

            // When - it is declared public
            await visitor.click(declare);

            // Then - the line has left
            await visitor.gone(line);
        });
        expect(result.errors.text).toBe('');
    });

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

    test.todo("N9 waits for a gesture's line kept in its Mac's journal");

    test("N10 a Mac chosen above the list keeps only that Mac's needs, and All brings them back", async () => {
        // Given - the Inbox over two-macs, where every need is on the first Mac
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            await visitor.see(
                within(needs, content('com.example.brain.x-snapshot is failing, last exit 2')),
            );

            // When - the Mac mini is chosen
            await visitor.click(within(main(), button('mac-mini')));

            // Then - none of the needs concerns it
            await visitor.see(
                content('None of the needs listed concerns mac-mini or its machines.'),
            );
            await visitor.gone(
                within(needs, content('com.example.brain.x-snapshot is failing, last exit 2')),
            );

            // When - all machines are chosen again
            await visitor.click(within(main(), button('All machines')));

            // Then - the needs are back
            await visitor.see(
                within(needs, content('com.example.brain.x-snapshot is failing, last exit 2')),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test("N10 a Mac chosen above the list keeps only that Mac's history, and All brings it back", async () => {
        // Given - the Inbox over two-macs, whose night failed mise on the first Mac and kept nothing on the mini
        const history = region('History');
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            await visitor.see(within(history, button('Retry mise')));

            // When - the Mac mini is chosen
            await visitor.click(within(main(), button('mac-mini')));

            // Then - the History reads the engine's sentence for that Mac, and mise has left
            await visitor.see(within(history, content('Nothing was done in the last 7 days.')));
            await visitor.gone(within(history, button('Retry mise')));

            // When - all machines are chosen again
            await visitor.click(within(main(), button('All machines')));

            // Then - the night is back
            await visitor.see(within(history, button('Retry mise')));
        });
        expect(result.errors.text).toBe('');
    });

    test("N12 a broken sign-in's line opens the sheet of the Sign-ins tab, for its program on its machine", async () => {
        // Given - the Inbox over two-macs, where This Mac's npm sign-in is broken
        const result = await website.visit(
            '/?world=two-macs&page=inbox&pace=600',
            async (visitor) => {
                await visitor.see(within(needs, button('Fix npm')));

                // When - Fix is pressed on its line
                await visitor.click(within(needs, button('Fix npm')));

                // Then - the engine's step for npm on This Mac is on a sheet
                await visitor.see(dialog('Sign in to npm on macbook-pro'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('N11 the sidebar holds Home, Inbox, Machine files, Passwords, the machines, the Catalog and Storage', async () => {
        // Given - the window over two-macs
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            // When - I read the sidebar
            await visitor.see(sidebar);

            // Then - the fleet's places are there, Inbox with its count of the lines that need me, the Catalog with its counts
            await visitor.see(within(sidebar, button('Home')));
            await visitor.see(within(sidebar, button('Inbox 2')));
            await visitor.see(within(sidebar, button('Machine files')));
            await visitor.see(within(sidebar, button('Passwords')));
            await visitor.see(within(sidebar, button('macbook-pro online')));
            await visitor.see(within(sidebar, button('api stopped')));
            await visitor.see(within(sidebar, button('linux-1 running')));
            await visitor.see(within(sidebar, button('mac-mini online')));
            await visitor.see(within(sidebar, button('infra healthy')));
            await visitor.see(within(sidebar, button('Apps 10')));
            await visitor.see(within(sidebar, button('Skills 0')));
            await visitor.see(within(sidebar, button('Settings 3')));
            await visitor.see(within(sidebar, button('Storage')));
        });
        expect(result.errors.text).toBe('');
    });
});
