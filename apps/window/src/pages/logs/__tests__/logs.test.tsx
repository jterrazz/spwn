import { button, clock, component, field, file, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'two-macs');

function logsOf(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Logs' });
}

describe('Logs', () => {
    test("a VM's log is the engine's lines, counted and timed by the read", async () => {
        // Given - linux-1 over two-macs, its log read at 14:20
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(logsOf('two-macs', 'linux-1'), async (visitor) => {
            // When - its log has been read
            await visitor.see(heading('0 lines, read at 14:20.'));
        });

        // Then - the tab says the engine read no line, with its filter and Refresh
        await expect(result.tree).toMatch(file('./__expected__/two-macs-linux-1.aria.yaml'));
    });

    test('a filter and a refresh keep the tab on the engine read', async () => {
        // Given - api over two-macs, its log read
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(logsOf('two-macs', 'api'), async (visitor) => {
            await visitor.see(heading('0 lines, read at 14:20.'));

            // When - a filter is typed, then the log is read again
            await visitor.fill(field('Filter the lines'), 'sshd');
            await visitor.click(button('Refresh'));

            // Then - the count says what the filter left of the whole
            await visitor.see(heading('0 lines, read at 14:20.'));
        });
        expect(result.errors.text).toBe('');
    });

    test('the setup run is a source the engine does not give', async () => {
        // Given - linux-1 over two-macs
        using _ = clock.at('2026-11-04T14:20:00Z');
        const result = await component.render(logsOf('two-macs', 'linux-1'), async (visitor) => {
            await visitor.see(heading('0 lines, read at 14:20.'));

            // When - the setup run is chosen
            await visitor.click(button('Setup run'));

            // Then - the tab says no call reads it
            await visitor.see(heading('No setup run from the engine.'));
        });
        await expect(result.tree).toMatch(file('./__expected__/two-macs-setup-run.aria.yaml'));
    });

    test("a Mac's Logs says what no call reads", async () => {
        // Given - This Mac over one-mac
        const result = await component.render(
            logsOf('one-mac', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its Logs tab is shown
                await visitor.see(heading('No log of this Mac from the engine.'));
            },
        );

        // Then - the gap is listed, not filled
        await expect(result.tree).toMatch(file('./__expected__/one-mac-this-mac.aria.yaml'));
    });

    test('a VM of a Mac that is not This Mac is read at that Mac', async () => {
        // Given - infra over mini-offline, a VM of the Mac mini, which the recording does not answer for
        const result = await component.render(logsOf('mini-offline', 'infra'), async (visitor) => {
            // When - its Logs tab is shown
            await visitor.see(heading('No answer from the engine.'));
        });

        // Then - its log is asked at the Mac mini, and the tab shows that it went unanswered
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-infra.aria.yaml'));
    });
});
