import { button, component, content, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('stories', 'two-macs');

function windowOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Storage' });
}

describe('machine-storage', () => {
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
});
