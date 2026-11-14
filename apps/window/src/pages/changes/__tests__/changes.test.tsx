import { button, component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('one-mac', 'stories', 'two-macs');

const history = region('History');
const lastNight = region('Last night');

describe('changes', () => {
    test("the Inbox's History holds the days the engine groups, the night's failure first with its Retry", async () => {
        // Given - the Inbox over stories, whose night moved three packages, held one and failed one
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'inbox' }),
            async (visitor) => {
                // When - the journal has answered
                await visitor.see(within(history, heading('Last night 3 moved, 1 held, 1 failed')));
            },
        );

        // Then - Today, Last night and Yesterday, each line with its time, who, what moved and on how many machines
        await expect(result.tree).toMatch(file('./__expected__/stories-history.aria.yaml'));
    });

    test("home draws the last night alone, with the night's first three lines", async () => {
        // Given - Home over stories
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'home' }),
            async (visitor) => {
                // When - the journal has answered
                await visitor.see(within(lastNight, button('Retry ripgrep')));
            },
        );

        // Then - the failure, node and gh, and neither Today nor Yesterday
        await expect(result.tree).toMatch(file('./__expected__/stories-home.aria.yaml'));
    });

    test("home's All history opens the Inbox", async () => {
        // Given - Home over stories
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'home' }),
            async (visitor) => {
                await visitor.see(within(lastNight, button('All history')));

                // When - All history is pressed
                await visitor.click(within(lastNight, button('All history')));

                // Then - the Inbox's History is open
                await visitor.see(within(history, heading('Yesterday')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a failed line's Retry sends the call it names, and the line marks it done", async () => {
        // Given - the Inbox over two-macs, where last night's mise update failed
        const result = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(history, button('Retry mise')));

                // When - Retry is pressed
                await visitor.click(within(history, button('Retry mise')));

                // Then - apps.update has answered and the line says Done
                await visitor.see(within(history, content('Done')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("choosing a Mac keeps that Mac's changes and its machines' alone, with no day's sum", async () => {
        // Given - the Inbox over stories
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(history, heading('Today')));

                // When - mac-mini is chosen above the list
                await visitor.click(button('mac-mini'));

                // Then - the deletion on the mini, infra's lines and node's upgrade on its two machines remain
                await visitor.gone(within(history, content('gh')));
            },
        );
        await expect(result.tree).toMatch(
            file('./__expected__/stories-history-mac-mini.aria.yaml'),
        );
    });

    test('a line opens the machine or the package it names', async () => {
        // Given - the Inbox over stories
        const result = await component.render(
            pageOver(recordingFor('stories'), { page: 'inbox' }),
            async (visitor) => {
                await visitor.see(within(history, heading('Yesterday')));

                // When - the line of linux-3's creation is pressed
                await visitor.click(
                    within(
                        history,
                        button('18:02 linux-3 created by agent-7f · review the docs lot'),
                    ),
                );

                // Then - linux-3's page is open, its head saying its state
                await visitor.see(heading('linux-3 Running'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a Mac with no change kept says so in its History', async () => {
        // Given - the Inbox over one-mac, whose journal is empty
        const result = await component.render(
            pageOver(recordingFor('one-mac'), { page: 'inbox' }),
            async (visitor) => {
                // When - the journal has answered
                await visitor.see(within(history, content('No change is kept yet.')));
            },
        );

        // Then - no day is drawn
        expect(result.html.text).not.toContain('Last night');
    });
});
