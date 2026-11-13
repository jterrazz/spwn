import { button, component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('creating', 'mini-offline', 'one-mac', 'two-macs');

function storageOver(world: string) {
    return pageOver(recordingFor(world), { page: 'storage' });
}

describe('Storage', () => {
    test("a Mac's disk is its volume, what can be freed, its data to erase, its images and its machines", async () => {
        // Given - Storage over two-macs
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            // When - This Mac's disk has been read to the end
            await visitor.see(within(region('Your data'), content('ollama')));
            await visitor.see(
                within(region('Images'), content('2 machines made from it', { exact: false })),
            );
        });

        // Then - the volume and its footer, the data its apps keep, the images and the machines with their sizes
        expect(result.content.text).not.toContain('Move');
        expect(result.content.text).not.toContain('What holds the disk');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-storage.aria.yaml'));
    });

    test('a machine of the Mac opens its own Storage tab', async () => {
        // Given - Storage over two-macs
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            await visitor.see(within(region('Machines'), content('linux-1')));

            // When - linux-1 is opened from the Machines panel
            await visitor.click(within(region('Machines'), button('linux-1 50 GB')));

            // Then - its own Storage tab is shown, with its disk
            await visitor.see(heading('linux-1 Running'));
            await visitor.see(region('By app'));
        });
        expect(result.errors.text).toBe('');
    });

    test("a machine's line under its name is what it is for, and nothing when it was made for nothing", async () => {
        // Given - Storage over creating, whose linux-5 was made for a lot and linux-4 for nothing
        const result = await component.render(storageOver('creating'), async (visitor) => {
            // When - its machines are listed
            await visitor.see(within(region('Machines'), content('linux-4')));
        });

        // Then - the purpose the engine keeps, and no resources line in its place
        expect(result.content.text).toContain('Lot shell, one block per package');
        expect(result.content.text).not.toContain('CPU ·');
    });

    test('a Mac the world holds no answer for shows that no answer came', async () => {
        // Given - Storage over mini-offline
        const result = await component.render(storageOver('mini-offline'), async (visitor) => {
            await visitor.see(heading('No verdict from the engine.'));

            // When - the Mac mini, which does not answer, is chosen
            await visitor.click(button('jterrazz-mac-mini'));

            // Then - its disk is no answer, and its machines are still listed
            await visitor.see(heading('No answer from the engine.'));
            await visitor.see(within(region('Machines'), content('infra')));
        });
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-storage.aria.yaml'));
    });

    test('a volume the engine reads empty offers nothing to free', async () => {
        // Given - Storage over one-mac
        const result = await component.render(storageOver('one-mac'), async (visitor) => {
            // When - its disk has been read
            await visitor.see(content('No image on jterrazz-macbook-pro.'));
        });

        // Then - the volume as the engine gave it, no footer and no image
        expect(result.content.text).not.toContain('What can be rebuilt holds');
        await expect(result.tree).toMatch(file('./__expected__/one-mac-storage.aria.yaml'));
    });
});
