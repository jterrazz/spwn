import { button, component, content, file, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('mini-workspace', 'one-mac', 'stories', 'two-macs');

function portsOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Ports' });
}

describe('ports', () => {
    test("a workspace's shares carry their addresses, and a QR code on demand", async () => {
        // Given - linux-1's Ports over two-macs, two of its ports shared
        const result = await component.render(portsOver('two-macs', 'linux-1'), async (visitor) => {
            await visitor.see(within(region('Shared'), content(':5173')));

            // When - the QR code of its first address is asked for
            await visitor.click(within(region('Port 5173'), button('QR code')));

            // Then - the code is drawn, and the share's own gesture stays offered
            await visitor.see(button('Hide QR code'));
            await visitor.see(within(region('Port 5173'), button('Stop sharing')));
        });
        await expect(result.tree).toMatch(file('./__expected__/two-macs-linux-1.aria.yaml'));
    });

    test("a workspace's listeners that are not shared are listed apart, each with its Share", async () => {
        // Given - linux-1's Ports over stories, 3000 and 5173 shared, 4321 and 9229 listening
        const result = await component.render(portsOver('stories', 'linux-1'), async (visitor) => {
            // When - the listing has been read
            await visitor.see(region('Listening, not shared'));
        });

        // Then - the shares, then only the listeners the engine lists no share of
        expect(result.content.text).not.toContain('What listens on');
        await expect(result.tree).toMatch(file('./__expected__/stories-linux-1.aria.yaml'));
    });

    test("a workspace whose ports are all shared draws no listener panel, and its Mac's range heads it", async () => {
        // Given - linux-1's Ports over mini-workspace, both of its listeners shared
        const result = await component.render(
            portsOver('mini-workspace', 'linux-1'),
            async (visitor) => {
                // When - the listing has been read
                await visitor.see(region('Shared'));
            },
        );

        // Then - the range of its Mac, and no panel of listeners
        expect(result.content.text).toContain(
            'It may share ports 40000 to 40999 with your devices',
        );
        expect(result.content.text).not.toContain('Listening, not shared');
    });

    test("a far Mac lists its own listeners, its machines' ports left to their tabs", async () => {
        // Given - mac-mini's Ports over stories
        const result = await component.render(portsOver('stories', 'mac-mini'), async (visitor) => {
            // When - the listing has been read
            await visitor.see(content('screensharingd'));
        });

        // Then - port 5900 alone, its reach the engine's
        await expect(result.tree).toMatch(file('./__expected__/stories-mac-mini.aria.yaml'));
    });

    test('a Mac where nothing listens says the listing is empty', async () => {
        // Given - This Mac's Ports over one-mac
        const result = await component.render(
            portsOver('one-mac', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // Then - the empty listing is said, not drawn as zero rows
                await visitor.see(content('Nothing listed on jterrazz-macbook-pro.'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
