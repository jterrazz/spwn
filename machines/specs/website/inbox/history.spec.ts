import { button, content, heading, main, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const STORIES = '/?world=stories&page=inbox';
const history = region('History');
const night = within(history, region('Last night'));

describe('history', () => {
    test('C1 the night is a group of its own with its sum, what moved and what was held and why', async () => {
        // Given - the Inbox over stories, whose night moved three packages, held one and failed one
        const result = await website.visit(STORIES, async (visitor) => {
            // When - the History is read
            await visitor.see(within(history, heading('Last night 3 moved, 1 held, 1 failed')));

            // Then - each package from its version to the next, and orbstack held with the engine's reason
            await visitor.see(within(night, content('2.97.0 → 2.98.0')));
            await visitor.see(within(night, content('0.9.7 → 0.9.9')));
            await visitor.see(within(night, content('updates itself')));
            await visitor.see(within(night, content('held')));
        });
        expect(result.errors.text).toBe('');
    });

    test("C2 the night's failure is marked failed and offers Retry, in the Inbox and on Home", async () => {
        // Given - stories, whose night failed ripgrep on linux-1
        const inbox = await website.visit(STORIES, async (visitor) => {
            // When - the Inbox is opened
            // Then - the failure reads failed with the engine's sentence, and offers Retry
            await visitor.see(within(night, button('Retry ripgrep')));
            await visitor.see(within(night, content('failed')));
            await visitor.see(within(night, content('download timed out')));
        });

        // When - Home is opened
        const home = await website.visit('/?world=stories&page=home', async (visitor) => {
            // Then - its Last night offers the same Retry
            await visitor.see(within(region('Last night'), button('Retry ripgrep')));
        });
        expect([inbox.errors.text, home.errors.text]).toStrictEqual(['', '']);
    });

    test('C2 Retry sends the update the line names, and the line says done', async () => {
        // Given - the Inbox over two-macs, whose night failed mise on this Mac
        const result = await website.visit('/?world=two-macs&page=inbox', async (visitor) => {
            await visitor.see(within(history, button('Retry mise')));

            // When - Retry is pressed
            await visitor.click(within(history, button('Retry mise')));

            // Then - apps.update has answered and the line says done
            await visitor.see(within(history, content('done')));
        });
        expect(result.errors.text).toBe('');
    });

    test('C3 a line says who made a VM, when, and for which purpose', async () => {
        // Given - the Inbox over stories, where agent-7f created linux-3 yesterday
        const result = await website.visit(STORIES, async (visitor) => {
            // When - the History is read
            // Then - its line gives the time, who and the purpose; today's restart says it was mine
            await visitor.see(
                within(
                    region('Yesterday'),
                    button('18:02 linux-3 created by agent-7f · review the docs lot'),
                ),
            );
            await visitor.see(within(region('Today'), button('09:05 infra restarted by you')));
        });
        expect(result.errors.text).toBe('');
    });

    test.todo("C4 waits for a gesture's line kept in its Mac's journal");

    test("C5 a Mac chosen keeps only its own lines and its machines', and All machines brings back the rest", async () => {
        // Given - the Inbox over stories, gh moved on macbook-pro alone
        const result = await website.visit(STORIES, async (visitor) => {
            await visitor.see(within(night, content('2.97.0 → 2.98.0')));

            // When - mac-mini is chosen above the list
            await visitor.click(within(main(), button('mac-mini')));

            // Then - gh has left, and the night's node stays, on mac-mini's own machines
            await visitor.gone(within(history, content('2.97.0 → 2.98.0')));
            await visitor.see(within(night, content('24.11.0 → 24.11.1')));

            // When - all machines are chosen again
            await visitor.click(within(main(), button('All machines')));

            // Then - gh is back
            await visitor.see(within(night, content('2.97.0 → 2.98.0')));
        });
        expect(result.errors.text).toBe('');
    });

    test('C6 a line opens the machine it names', async () => {
        // Given - the Inbox over stories
        const result = await website.visit(STORIES, async (visitor) => {
            // When - the line of linux-3's creation is pressed
            await visitor.click(
                within(history, button('18:02 linux-3 created by agent-7f · review the docs lot')),
            );

            // Then - linux-3's page is open, its head saying its state
            await visitor.see(heading('linux-3 running'));
        });
        expect(result.errors.text).toBe('');
    });

    test('C6 a line opens the package it names', async () => {
        // Given - the Inbox over stories
        const result = await website.visit(STORIES, async (visitor) => {
            // When - the night's line of node is pressed
            await visitor.click(within(night, button('node 24.11.0 → 24.11.1', { exact: false })));

            // Then - node's catalog page is open
            await visitor.see(region('How it is installed'));
        });
        expect(result.errors.text).toBe('');
    });

    test("C7 a Mac that does not answer is named with the engine's reason, and the others' lines still read", async () => {
        // Given - the Inbox over stories, where macbook-air is not seen
        const result = await website.visit(STORIES, async (visitor) => {
            // When - the History is read
            // Then - the night of the two Macs that answer, and the silent one named
            await visitor.see(within(night, button('Retry ripgrep')));
            await visitor.see(within(history, content('macbook-air does not answer')));
        });
        expect(result.errors.text).toBe('');
    });

    test("C8 a fleet where nothing was done reads the engine's sentence, and no day", async () => {
        // Given - the Inbox over one-mac, whose journal keeps no line
        const result = await website.visit('/?world=one-mac&page=inbox', async (visitor) => {
            // When - the History is read
            // Then - the engine's sentence, and no group
            await visitor.see(within(history, content('Nothing was done in the last 7 days.')));
            await visitor.gone(within(history, heading('Today')));
        });
        expect(result.errors.text).toBe('');
    });

    test('C9 a line about a machine deleted since keeps its name and opens nothing', async () => {
        // Given - the Inbox over stories, where gate-2 was deleted today
        const result = await website.visit(STORIES, async (visitor) => {
            // When - the History is read
            await visitor.see(within(region('Today'), content('8 GB freed')));

            // Then - the line keeps gate-2's name and is no button, while today's restart of infra is one
            await visitor.see(within(region('Today'), content('gate-2 deleted')));
            await visitor.see(within(region('Today'), button('09:05 infra restarted by you')));
            await visitor.gone(within(region('Today'), button('gate-2 deleted', { exact: false })));
        });
        expect(result.errors.text).toBe('');
    });
});
