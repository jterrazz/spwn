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

import { pageOver } from '../../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'two-macs');

const READ_AT =
    '0 lines · read at 14:20 · a reading, not a live feed: press Refresh for the newest lines';

const macLog = region('What Machines did on this Mac');

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
                await visitor.see(heading('Nothing to look at.'));
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
                await visitor.see(heading('Nothing to look at.'));
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

    test("a Mac's log is the lines station wrote there, a withheld one as the engine sent it", async () => {
        // Given - This Mac over two-macs, its engine's lines read at 14:20
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(
            activityOf('two-macs', 'macbook-pro'),
            async (visitor) => {
                // When - its activity and its engine's lines have been read
                await visitor.see(heading('Nothing to look at.'));
                await visitor.see(
                    within(macLog, content('4 lines · read at 14:20', { exact: false })),
                );

                // Then - each line is the engine's, the one it withheld drawn as it came
                await visitor.see(
                    within(macLog, content('ping https://withheld@example.com/x ok')),
                );
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/two-macs-this-mac-log.aria.yaml'));
    });

    test("a filter narrows a Mac's lines to those that hold it", async () => {
        // Given - This Mac over two-macs, its engine's lines read
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(
            activityOf('two-macs', 'macbook-pro'),
            async (visitor) => {
                await visitor.see(
                    within(macLog, content('4 lines · read at 14:20', { exact: false })),
                );

                // When - a filter is typed
                await visitor.fill(within(macLog, field('Filter the lines')), 'nightly');

                // Then - the count says what the filter left of the whole
                await visitor.see(
                    within(macLog, content('2 of 4 lines · read at 14:20', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a Mac that is not This Mac has its own lines read at that Mac', async () => {
        // Given - the Mac mini over two-macs
        const result = await component.render(
            activityOf('two-macs', 'mac-mini'),
            async (visitor) => {
                // When - its Activity is shown
                // Then - its log holds the lines its own engine wrote
                await visitor.see(within(macLog, content('waiting for the next night run')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a Mac whose engine wrote no line says the engine's sentence for it", async () => {
        // Given - This Mac over one-mac, whose engine has written no line
        const result = await component.render(
            activityOf('one-mac', 'macbook-pro'),
            async (visitor) => {
                // When - its Activity is shown
                // Then - the log says so in the engine's words
                await visitor.see(
                    within(macLog, content('The engine has written no line here yet.')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a Mac that does not answer draws the refusal in its log's place", async () => {
        // Given - the Mac mini over mini-offline, whose engine.log is refused as unreachable
        const result = await component.render(
            activityOf('mini-offline', 'mac-mini'),
            async (visitor) => {
                // When - its Activity is shown
                // Then - the log panel says the Mac does not answer, and no line
                await visitor.see(
                    within(macLog, content('mac-mini does not answer.', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a VM of a Mac that is not This Mac is read at that Mac', async () => {
        // Given - infra over mini-offline, a VM of the Mac mini, which does not answer
        const result = await component.render(
            activityOf('mini-offline', 'infra'),
            async (visitor) => {
                // When - its Activity is shown
                await visitor.see(heading('mac-mini does not answer.'));
                await visitor.see(
                    within(
                        region('What infra prints'),
                        content('mac-mini does not answer.', { exact: false }),
                    ),
                );
            },
        );

        // Then - its log is asked at the Mac mini, and the panel shows the refusal that came from there
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-infra.aria.yaml'));
    });
});
