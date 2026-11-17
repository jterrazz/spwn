import { button, component, content, dialog, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('range-full', 'share-local', 'stories', 'two-macs');

function portsOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Ports' });
}

describe('port gestures', () => {
    test('share asks nothing, and the port joins the shares the engine lists', async () => {
        // Given - linux-1's Ports over stories, port 4321 listening and not shared
        const result = await component.render(portsOver('stories', 'linux-1'), async (visitor) => {
            await visitor.see(region('Listening, not shared'));

            // When - its Share is clicked
            await visitor.click(button('Share :4321'));

            // Then - ports.list, read again, lists it among the shares, waiting
            await visitor.see(within(region('Shared'), region('Port 4321')));
            await visitor.see(within(region('Port 4321'), content('Waiting')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a share the engine cannot make shows its words and its hint', async () => {
        // Given - linux-1's Ports over range-full, linux-1 sharing the most ports a VM may and listening on 9000
        const result = await component.render(
            portsOver('range-full', 'linux-1'),
            async (visitor) => {
                await visitor.see(region('Listening, not shared'));
                const said = content('linux-1 has 20 shared ports, the most a VM may.', {
                    exact: false,
                });
                const hint = content('station ports stop-sharing linux-1 <port>');
                await visitor.gone(hint);

                // When - Share of port 9000 is clicked
                await visitor.click(button('Share :9000'));

                // Then - the engine's failure as it came, its words then its hint, and Dismiss clears it
                await visitor.see(said);
                await visitor.see(hint);
                await visitor.click(button('Dismiss'));
                await visitor.gone(said);
                await visitor.gone(hint);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a share of a far Mac's VM is sent to that Mac, and the port joins its shares", async () => {
        // Given - linux-3's Ports over stories, linux-3 on mac-mini, port 5432 listening and not shared
        const result = await component.render(portsOver('stories', 'linux-3'), async (visitor) => {
            await visitor.see(region('Listening, not shared'));

            // When - Share of port 5432 is clicked
            await visitor.click(button('Share :5432'));

            // Then - the mini's ports.list, read again, lists it among the shares, waiting
            await visitor.see(within(region('Shared'), region('Port 5432')));
            await visitor.see(within(region('Port 5432'), content('Waiting')));
        });
        expect(result.errors.text).toBe('');
    });

    test('stop sharing asks first, then the share leaves the list', async () => {
        // Given - linux-1's Ports over two-macs, port 5173 shared
        const result = await component.render(portsOver('two-macs', 'linux-1'), async (visitor) => {
            await visitor.see(within(region('Shared'), region('Port 5173')));

            // When - Stop sharing is clicked, then confirmed
            await visitor.click(within(region('Port 5173'), button('Stop sharing')));
            const asked = dialog('Stop sharing linux-1:5173?');
            await visitor.see(asked);
            await visitor.click(within(asked, button('Stop sharing')));

            // Then - ports.list, read again, holds no share of port 5173
            await visitor.gone(asked);
            await visitor.gone(region('Port 5173'));
            await visitor.see(region('Port 7000'));
        });
        expect(result.errors.text).toBe('');
    });

    test("a share's reach moves to Local, and the switch and the address follow", async () => {
        // Given - linux-1's Ports over two-macs, port 5173 recorded Private
        const result = await component.render(portsOver('two-macs', 'linux-1'), async (visitor) => {
            const shared = region('Port 5173');
            await visitor.see(within(shared, content('https://macbook-pro.tail1234.ts.net:40001')));

            // When - Local is clicked
            await visitor.click(within(shared, button('Local')));

            // Then - ports.list, read again, gives it the Mac's own address, and Stop sharing stays offered
            await visitor.see(within(shared, content('http://127.0.0.1:40001')));
            await visitor.see(within(shared, button('Stop sharing')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a share that starts Local reads Local, and Private moves it', async () => {
        // Given - linux-1's Ports over share-local, port 5173 recorded Local
        const result = await component.render(
            portsOver('share-local', 'linux-1'),
            async (visitor) => {
                const shared = region('Port 5173');
                await visitor.see(within(shared, content('http://127.0.0.1:40001')));

                // When - Private is clicked
                await visitor.click(within(shared, button('Private')));

                // Then - it is shared on the devices network
                await visitor.see(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:40001')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a share made public offers no switch, its reach stays a tag', async () => {
        // Given - linux-1's Ports over stories, port 3000 recorded shared Public
        const result = await component.render(portsOver('stories', 'linux-1'), async (visitor) => {
            const shared = region('Port 3000');
            await visitor.see(within(shared, content('Public')));

            // When - the row is read
            // Then - neither Local nor Private is offered on it
            await visitor.gone(within(shared, button('Local')));
            await visitor.gone(within(shared, button('Private')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a cancel at the confirm leaves the share as it was', async () => {
        // Given - linux-1's Ports over two-macs
        const result = await component.render(portsOver('two-macs', 'linux-1'), async (visitor) => {
            await visitor.see(region('Port 5173'));

            // When - Stop sharing is clicked, then cancelled
            await visitor.click(within(region('Port 5173'), button('Stop sharing')));
            const asked = dialog('Stop sharing linux-1:5173?');
            await visitor.click(within(asked, button('Cancel')));

            // Then - no dialog, and the port is still shared
            await visitor.gone(asked);
            await visitor.see(region('Port 5173'));
        });
        expect(result.errors.text).toBe('');
    });
});
