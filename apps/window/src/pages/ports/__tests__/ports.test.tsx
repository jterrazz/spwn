import type { PortsPort } from '@jterrazz/station-api';
import { button, component, content, file, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Screen } from '../../../design/index';
import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';
import { GestureConfirm } from '../../kit/gestures';
import { MacPortRow } from '../ports';

const recordingFor = await recordingsOf('mini-workspace', 'one-mac', 'stories', 'two-macs');

function portsOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Ports' });
}

const EXPOSED: PortsPort = {
    Answer: '',
    Binds: ['0.0.0.0'],
    Declared: null,
    Device: false,
    Machine: { Name: '', On: '' },
    PID: 4100,
    Port: 5000,
    Process: 'python3',
    Reach: 2,
    Verified: null,
};

function macPortOver(port: PortsPort) {
    const client = new Client(replayTransport(recordingFor('stories')));
    return (
        <EngineProvider client={client}>
            <Screen>
                <MacPortRow mac="macbook-pro" port={port} target={undefined} />
            </Screen>
            <GestureConfirm />
        </EngineProvider>
    );
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
            // When - the listing and the drift have been read
            await visitor.see(region('Listening, not shared'));
            await visitor.see(region('Differs from its file'));
        });

        // Then - the shares, the listeners not shared, and 3000 shared public where its file asks private
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

    test('a port reached outside Tailscale is declared public, and the engine says it is done', async () => {
        // Given - a port of this Mac on every interface, which no declaration names
        const result = await component.render(macPortOver(EXPOSED), async (visitor) => {
            await visitor.see(content('python3'));

            // When - Declare it public is clicked
            await visitor.click(button('Declare :5000 public'));

            // Then - it asks nothing, and the engine's answer settles the gesture done on its row
            await visitor.see(content('Done'));
        });
        expect(result.errors.text).toBe('');
    });

    test("a VM Overview's calm share keeps no empty state column, so its address has the room", async () => {
        // Given - linux-1's Overview over stories, its port 3000 shared and running
        let gap = -1;
        const result = await component.render(
            pageOver(recordingFor('stories'), {
                machine: 'linux-1',
                page: 'machine',
                tab: 'Overview',
            }),
            async (visitor) => {
                // When - its Ports panel is drawn
                await visitor.see(region('Port 3000'));
                const row = document.querySelector('section[aria-label="Port 3000"]');
                const text = row?.querySelector('span[title*="://"]')?.closest('div');
                const stop = [...(row?.querySelectorAll('button') ?? [])].find(
                    (one) => one.textContent === 'Stop sharing',
                );
                gap =
                    (stop?.getBoundingClientRect().left ?? 0) -
                    (text?.getBoundingClientRect().right ?? 0);
            },
        );

        // Then - the address column runs up to the gesture, no state column between them
        expect(gap).toBeGreaterThanOrEqual(0);
        expect(gap).toBeLessThan(40);
        expect(result.errors.text).toBe('');
    });

    test('a port a declaration already names offers no declare', async () => {
        // Given - the same port, declared public by the LAN
        const declared = { ...EXPOSED, Declared: { machine: '', port: 5000, via: 'lan' } };
        const result = await component.render(macPortOver(declared), async (visitor) => {
            await visitor.see(content('python3'));

            // Then - no gesture is drawn on its row
            await visitor.gone(button('Declare :5000 public'));
        });
        expect(result.errors.text).toBe('');
    });
});
