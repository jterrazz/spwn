import { button, content, heading, main, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const needs = region('Needs you');

describe('needs', () => {
    test('N1 a line settled by its verb leaves, and the sidebar counts one less', async () => {
        // Given - the window on Home over two-macs, nine lines waiting, one of them a mise update
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            await visitor.see(within(sidebar, button('Inbox 9')));
            await visitor.see(heading('Needs you 9'));
            await visitor.see(within(needs, content('1 mise update waiting')));

            // When - its verb, Update mise, is pressed
            await visitor.click(within(needs, button('Update mise')));

            // Then - the line has left, and the sidebar and the panel count eight
            await visitor.gone(within(needs, button('Update mise')));
            await visitor.see(within(sidebar, button('Inbox 8')));
            await visitor.see(heading('Needs you 8'));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('N1 waits for a verdict and its line on overview.read');

    test.todo("N2 waits for a need's upkeep word on OverviewNeed");

    test.todo('N3 waits for a recording of an exposed port and the Review verb of its line');

    test('N4 Extend gives a share about to end 12 h more, and the machine page says so', async () => {
        // Given - linux-1's Accounts over stories, its claude copy ending in 24 min
        const result = await website.visit(
            '/?world=stories&machine=linux-1&tab=Accounts',
            async (visitor) => {
                const shared = region('Shared with it');
                await visitor.see(within(shared, content('ends in 24 min')));

                // When - Extend of claude is pressed
                await visitor.click(within(shared, button('Extend claude by 12 h')));

                // Then - the copy holds 12 h from now, on the page of the machine
                await visitor.see(within(shared, content('ends in 11 h 59 min')));
                await visitor.gone(within(shared, content('ends in 24 min')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo("N5 waits for each line's version rule on machinefile.drift");

    test.todo(
        'N6 waits for a recording of an idle VM among the needs, and a kept journal for Changes',
    );

    test('N7 a refused account offers Sign in on the Mac, with its own login command', async () => {
        // Given - This Mac's Accounts over two-macs, where npm's token was refused
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-macbook-pro&tab=Accounts',
            async (visitor) => {
                // When - I read the Accounts tab
                const accounts = region('Accounts on this Mac');
                await visitor.see(accounts);

                // Then - the row offers Sign in and the command it runs
                await visitor.see(within(accounts, button('Sign in')));
                await visitor.see(within(accounts, button('Copy npm login')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'N7 waits for a recording of an account signed in again, and for the Tauri window, the only one that opens Terminal',
    );

    test.todo(
        'N8 window bug: an empty Needs you reads "No need is listed.", where the scenario says "Nothing here."',
    );

    test.todo('N9 waits for a kept journal, read by day');

    test("N10 a Mac chosen above the list keeps only that Mac's needs, and All brings them back", async () => {
        // Given - the Inbox over two-macs, where every need is on the first Mac
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            await visitor.see(within(needs, content('npm sign-in expired')));

            // When - the Mac mini is chosen
            await visitor.click(within(main(), button('jterrazz-mac-mini')));

            // Then - none of the needs concerns it
            await visitor.see(
                content('None of the needs listed concerns jterrazz-mac-mini or its machines.'),
            );
            await visitor.gone(content('npm sign-in expired'));

            // When - all machines are chosen again
            await visitor.click(within(main(), button('All machines')));

            // Then - the needs are back
            await visitor.see(within(needs, content('npm sign-in expired')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo('N10 waits for what can wait, and for a kept journal, each read for one Mac alone');

    test('N11 the sidebar holds Home, Inbox with its count, Storage, the machines, Machine files and the Catalog', async () => {
        // Given - the window over two-macs
        const result = await website.visit('/?world=two-macs&page=home', async (visitor) => {
            // When - I read the sidebar
            await visitor.see(sidebar);

            // Then - the fleet's places are there, the count on Inbox
            await visitor.see(within(sidebar, button('Home')));
            await visitor.see(within(sidebar, button('Inbox 9')));
            await visitor.see(within(sidebar, button('Storage')));
            await visitor.see(within(sidebar, button('jterrazz-macbook-pro online')));
            await visitor.see(within(sidebar, button('api stopped')));
            await visitor.see(within(sidebar, button('linux-1 running')));
            await visitor.see(within(sidebar, button('jterrazz-mac-mini online')));
            await visitor.see(within(sidebar, button('infra running')));
            await visitor.see(within(sidebar, button('Machine files')));
            await visitor.see(within(sidebar, button('Apps')));
            await visitor.see(within(sidebar, button('Skills')));
            await visitor.see(within(sidebar, button('Settings')));
        });
        expect(result.errors.text).toBe('');
    });
});
