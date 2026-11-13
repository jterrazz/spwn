import { button, component, content, enabled, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'two-macs');

function windowOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Storage' });
}

describe('machine-storage', () => {
    test('what a Mac holds is one row per app, each with its kinds, its gestures offered', async () => {
        // Given - the window on This Mac's Storage over two-macs
        const result = await component.render(
            windowOver('two-macs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(region('By app'));

                // Then - freeing what can be rebuilt, 34.8 GB, is the screen's one primary gesture, and an
                // Then - app holding a cache and data offers its Free and its Erase… on one row
                await visitor.see(enabled(button('Free 34.8 GB')));
                await visitor.see(within(region('By app'), button('Free claude')));
                await visitor.see(within(region('By app'), button('Erase… claude')));
            },
        );
        expect(result.content.text).not.toContain('No verdict from the engine.');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-this-mac.aria.yaml'));
    });

    test("a machine's disk says it is one file on its Mac, read in Storage", async () => {
        // Given - the window on linux-1's Storage over two-macs
        const result = await component.render(
            windowOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(content('cache 1.2 GB'));

                // Then - the line under its figure opens Storage
                await visitor.click(within(region('Storage'), button('Storage')));
                await visitor.see(heading('No verdict from the engine.'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a disk the engine reads as empty says so, and offers nothing to free', async () => {
        // Given - the window on This Mac's Storage over one-mac, whose disk holds no row
        const result = await component.render(
            windowOver('one-mac', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(content('Nothing listed on jterrazz-macbook-pro.'));
            },
        );

        // Then - the volume is the engine's numbers, and no Free button is drawn
        await expect(result.tree).toMatch(file('./__expected__/one-mac-this-mac.aria.yaml'));
    });

    test('a Mac that does not answer shows the transport saying so', async () => {
        // Given - the window on the Mac mini's Storage over mini-offline
        const result = await component.render(
            windowOver('mini-offline', 'jterrazz-mac-mini'),
            async (visitor) => {
                // When - its read ends with no answer
                await visitor.see(heading('No answer from the engine.'));
            },
        );

        // Then - nothing is drawn in place of the answer
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-mini.aria.yaml'));
    });
});
