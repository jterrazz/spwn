import {
    button,
    clock,
    component,
    content,
    field,
    file,
    heading,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'two-macs');

const READ_AT =
    '0 lines · read at 14:20 · a reading, not a live feed: press Refresh for the newest lines';

function activityOf(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Activity' });
}

describe('Log', () => {
    test("a VM's log is the last panel of its Activity, counted and timed by the read", async () => {
        // Given - linux-1 over two-macs, its log read at 14:20
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(
            activityOf('two-macs', 'linux-1'),
            async (visitor) => {
                // When - its activity and its log have been read
                await visitor.see(heading('No finding from the engine.'));
                await visitor.see(within(region('What linux-1 prints'), content(READ_AT)));
            },
        );

        // Then - its measures first, then its console with its filter, Refresh and Copy all
        await expect(result.tree).toMatch(file('./__expected__/two-macs-linux-1.aria.yaml'));
    });

    test('a filter and a refresh keep the panel on the engine read', async () => {
        // Given - api over two-macs, its log read
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(activityOf('two-macs', 'api'), async (visitor) => {
            const log = region('What api prints');
            await visitor.see(within(log, content(READ_AT)));

            // When - a filter is typed, then the log is read again
            await visitor.fill(field('Filter the lines'), 'sshd');
            await visitor.click(button('Refresh'));

            // Then - the count says what the filter left of the whole
            await visitor.see(
                within(log, content('0 of 0 lines · read at 14:20', { exact: false })),
            );
        });
        expect(result.errors.text).toBe('');
    });

    test('the setup run is a source the engine does not give', async () => {
        // Given - linux-1 over two-macs
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(
            activityOf('two-macs', 'linux-1'),
            async (visitor) => {
                await visitor.see(heading('No finding from the engine.'));
                await visitor.see(within(region('What linux-1 prints'), content(READ_AT)));

                // When - the setup run is chosen
                await visitor.click(button('Setup run'));

                // Then - the panel says no call reads it
                await visitor.see(
                    within(
                        region('Its last setup run'),
                        content('What its file installed and wrote on its last run.'),
                    ),
                );
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/two-macs-setup-run.aria.yaml'));
    });

    test("a Mac's log is a panel that says what no call reads", async () => {
        // Given - This Mac over one-mac
        const result = await component.render(
            activityOf('one-mac', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its Activity is shown
                await visitor.see(heading('No finding from the engine.'));

                // Then - its last panel names the log no call reads, and draws no line
                await visitor.see(
                    within(
                        region('What station did on this Mac'),
                        content("The engine's own lines and its setup runs."),
                    ),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a VM of a Mac that is not This Mac is read at that Mac', async () => {
        // Given - infra over mini-offline, a VM of the Mac mini, which the recording does not answer for
        const result = await component.render(
            activityOf('mini-offline', 'infra'),
            async (visitor) => {
                // When - its Activity is shown
                await visitor.see(heading('No answer from the engine.'));
                await visitor.see(
                    within(
                        region('What infra prints'),
                        content('No answer from the engine', { exact: false }),
                    ),
                );
            },
        );

        // Then - its log is asked at the Mac mini, and the panel shows that it went unanswered
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-infra.aria.yaml'));
    });
});
