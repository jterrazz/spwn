import { button, component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { isRecord } from '../../../../domain/request-key';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('creating', 'mini-offline', 'one-mac', 'two-macs');

function storageOver(world: string) {
    return pageOver(recordingFor(world), { page: 'storage' });
}

function reversedAnswer(answer: unknown): unknown {
    if (!isRecord(answer) || !Array.isArray(answer.out)) {
        return answer;
    }
    return {
        ...answer,
        out: answer.out.map((image: unknown) =>
            isRecord(image) ? { ...image, Deletable: image.Deletable !== true } : image,
        ),
    };
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

    test("an image's Delete is offered exactly when the engine says it is deletable", async () => {
        // Given - Storage over two-macs, whose older version is deletable and whose newest is not
        const recording = recordingFor('two-macs');
        const flipped = {
            ...recording,
            exchanges: recording.exchanges.map((exchange) =>
                exchange.request.call === 'images.list'
                    ? {
                          ...exchange,
                          answers: exchange.answers.map(reversedAnswer),
                      }
                    : exchange,
            ),
        };

        const images = region('Images');
        const older = within(images, button('Delete station-linux 2026.10.20'));
        const newest = within(images, button('Delete station-linux 2026.11.01'));

        // When - the answer is drawn as recorded, then with every Deletable reversed
        const asked = await component.render(storageOver('two-macs'), async (visitor) => {
            // Then - Delete is on the version the engine marked, and not on the other
            await visitor.see(older);
            await visitor.gone(newest);
        });
        const reversed = await component.render(
            pageOver(flipped, { page: 'storage' }),
            async (visitor) => {
                await visitor.see(newest);
                await visitor.gone(older);
            },
        );
        expect(asked.errors.text).toBe('');
        expect(reversed.errors.text).toBe('');
    });

    test('an image being built is one row at its step, with no date, size or Delete', async () => {
        // Given - Storage over creating, a build held at Packages
        const images = region('Images');
        const result = await component.render(storageOver('creating'), async (visitor) => {
            // When - its images have been read
            await visitor.see(within(images, content('Packages · Step 2 of 4')));

            // Then - the engine's step and sentence stand, nothing of a built image does
            await visitor.see(within(images, content('2026.11.04')));
            await visitor.gone(within(images, content('built 0', { exact: false })));
            await visitor.gone(within(images, content('GB', { exact: false })));
            await visitor.gone(within(images, button('Delete station-linux 2026.11.04')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a machine of the Mac opens its own Storage tab', async () => {
        // Given - Storage over two-macs
        const result = await component.render(storageOver('two-macs'), async (visitor) => {
            await visitor.see(within(region('Machines'), content('linux-1')));

            // When - linux-1 is opened from the Machines panel
            await visitor.click(
                within(region('Machines'), button('linux-1 1.5 GB of caches inside 50 GB')),
            );

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
            await visitor.see(heading('Nothing to free.'));

            // When - the Mac mini, which does not answer, is chosen
            await visitor.click(button('mac-mini'));

            // Then - its disk is no answer, it has no verdict, and its machines are still listed
            await visitor.see(heading('No answer from the engine.'));
            await visitor.see(heading('No verdict from the engine.'));
            await visitor.see(within(region('Machines'), content('infra')));
        });
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-storage.aria.yaml'));
    });

    test('a volume the engine reads empty offers nothing to free', async () => {
        // Given - Storage over one-mac
        const result = await component.render(storageOver('one-mac'), async (visitor) => {
            // When - its disk has been read
            await visitor.see(content('No image on macbook-pro.'));
        });

        // Then - the volume as the engine gave it, no Free and no image
        expect(result.content.text).not.toContain('Free 0');
        await expect(result.tree).toMatch(file('./__expected__/one-mac-storage.aria.yaml'));
    });
});
