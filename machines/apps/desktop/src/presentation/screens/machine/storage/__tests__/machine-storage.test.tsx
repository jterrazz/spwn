import { button, component, content, enabled, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../../composition/__tests__/page-over';
import { unansweredAt } from '../../../../../infrastructure/outbound/engine-replay/__tests__/unanswered';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'stories', 'two-macs');

function cardOf(html: string): string | undefined {
    return /<section aria-label="Disk"[^>]*>.*?<\/section>/su.exec(html)?.[0];
}

function windowOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Storage' });
}

describe('machine-storage', () => {
    test('what a Mac holds is one row per app, each with its kinds, its gestures offered', async () => {
        // Given - the window on This Mac's Storage over two-macs
        const result = await component.render(
            windowOver('two-macs', 'macbook-pro'),
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

    test("a Mac's Storage tab and the Storage page draw its disk as one card, part for part", async () => {
        // Given - the window over two-macs, on This Mac's Storage tab and on the Storage page
        const tab = await component.render(
            windowOver('two-macs', 'macbook-pro'),
            async (visitor) => {
                await visitor.see(region('By app'));
            },
        );
        const page = await component.render(
            pageOver(recordingFor('two-macs'), { page: 'storage' }),
            async (visitor) => {
                await visitor.see(region('Machines'));
            },
        );

        // When - the Disk card is cut out of each
        const card = cardOf(tab.html.text);
        const onThePage = cardOf(page.html.text);

        // Then - one card, byte for byte: what the Mac itself holds and what Machines holds beside it, every
        // Then - part, and the same gestures to free
        expect(card).toBeDefined();
        expect(card).toBe(onThePage);
        expect(card).toContain('This Mac itself holds');
        expect(card).toContain('Machine disks');
        expect(card).toContain('Your data');
    });

    test("a machine's disk says it is one file on its Mac, read in Storage", async () => {
        // Given - the window on linux-1's Storage over two-macs
        const result = await component.render(
            windowOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(content('cache 1.2 GB'));

                // Then - the line under its figure opens Storage
                await visitor.click(within(region('Disk'), button('Storage')));
                await visitor.see(heading('3.0 GB can be freed.'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a kind of 0 bytes draws no segment, the free space drawn last', async () => {
        // Given - the window on linux-1's Storage over stories, whose system, apps and caches hold 0 bytes
        const result = await component.render(windowOver('stories', 'linux-1'), async (visitor) => {
            // When - its disk has been read
            await visitor.see(within(region('Disk'), content('Your data')));
        });

        // Then - one segment for your data, one for free, none for what holds nothing
        expect(result.html.text).toContain('title="Your data · 10 GB"');
        expect(result.html.text).toContain('title="Free · 30 GB"');
        expect(result.html.text).not.toMatch(/title="(?:System|Apps|Caches) · /u);
    });

    test('a disk the engine reads as empty says so, and offers nothing to free', async () => {
        // Given - the window on This Mac's Storage over one-mac, whose disk holds no row
        const result = await component.render(
            windowOver('one-mac', 'macbook-pro'),
            async (visitor) => {
                // When - its disk has been read
                await visitor.see(content('Nothing listed on macbook-pro.'));
            },
        );

        // Then - the volume is the engine's numbers, and no Free button is drawn
        await expect(result.tree).toMatch(file('./__expected__/one-mac-this-mac.aria.yaml'));
    });

    test('a Mac that does not answer shows the transport saying so', async () => {
        // Given - the window on the Mac mini's Storage over mini-offline, no read asked at the mini ever answered
        const unanswered = unansweredAt(recordingFor('mini-offline'), 'mac-mini');
        const result = await component.render(
            pageOver(unanswered, { machine: 'mac-mini', page: 'machine', tab: 'Storage' }),
            async (visitor) => {
                // When - its read ends with no answer
                await visitor.see(heading('No answer from the engine.'));
            },
        );

        // Then - nothing is drawn in place of the answer
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-mini.aria.yaml'));
    });
});
