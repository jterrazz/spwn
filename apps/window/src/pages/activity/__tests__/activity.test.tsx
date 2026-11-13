import { button, component, content, file, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('disk-low', 'mini-offline', 'one-mac', 'two-macs');

function activityOf(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Activity' });
}

describe('Activity', () => {
    test("a calm Mac's Activity is its measures, its busiest and its facts", async () => {
        // Given - This Mac over two-macs, which raises no finding
        const result = await component.render(
            activityOf('two-macs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its activity has been read
                await visitor.see(heading('No finding from the engine.'));
            },
        );

        // Then - the four measures, the busiest by CPU and the Mac's facts are the engine's
        await expect(result.tree).toMatch(file('./__expected__/two-macs-this-mac.aria.yaml'));
    });

    test("a finding heads the tab in the engine's words, with its fix", async () => {
        // Given - This Mac over disk-low, whose disk is almost full
        const result = await component.render(
            activityOf('disk-low', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its activity has been read
                await visitor.see(heading('Disk almost full: 4 % free'));
            },
        );

        // Then - the finding is the verdict, its severity a word and its command beside it
        await expect(result.tree).toMatch(file('./__expected__/disk-low-this-mac.aria.yaml'));
    });

    test('a Mac whose kernel counted nothing shows unknown, never zeros', async () => {
        // Given - This Mac over one-mac, whose meter counted no tick and no memory
        const result = await component.render(
            activityOf('one-mac', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its activity has been read
                await visitor.see(heading('No finding from the engine.'));
            },
        );

        // Then - each measure is a question mark, not a zero
        await expect(result.tree).toMatch(file('./__expected__/one-mac-this-mac.aria.yaml'));
    });

    test('a VM its Mac keeps no history of is read inside it, and its refusal shown', async () => {
        // Given - api over two-macs, a VM that is stopped
        const result = await component.render(activityOf('two-macs', 'api'), async (visitor) => {
            // When - its Mac answered no history, then the VM itself was asked
            await visitor.see(heading('api is stopped.'));
        });

        // Then - the tab is the VM's own refusal, and no measure is drawn
        await expect(result.tree).toMatch(file('./__expected__/two-macs-api.aria.yaml'));
    });

    test('a Mac that does not answer is no answer, never a zero', async () => {
        // Given - the Mac mini over mini-offline, which does not answer
        const result = await component.render(
            activityOf('mini-offline', 'jterrazz-mac-mini'),
            async (visitor) => {
                // When - its activity is asked
                await visitor.see(heading('No answer from the engine.'));
            },
        );

        // Then - the tab draws no measure
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-mini.aria.yaml'));
    });

    test('the switches choose the window and the order of the busiest', async () => {
        // Given - This Mac over two-macs, read over the hour
        const result = await component.render(
            activityOf('two-macs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                await visitor.see(heading('No finding from the engine.'));

                // When - the day is chosen, then the busiest by memory
                await visitor.click(button('24 h'));
                await visitor.see(heading('No finding from the engine.'));
                await visitor.click(button('By memory'));

                // Then - the VM's process leads by its resident memory
                await visitor.see(content('4.1 GB'));
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/two-macs-day-memory.aria.yaml'));
    });

    test("a VM among a Mac's busiest opens that VM's Activity", async () => {
        // Given - This Mac over two-macs, linux-1 among its busiest
        const result = await component.render(
            activityOf('two-macs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                await visitor.see(content('a machine of this Mac'));

                // When - its row is opened
                await visitor.click(content('a machine of this Mac'));

                // Then - linux-1's own Activity is shown
                await visitor.see(heading('linux-1 Running'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
