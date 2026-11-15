import { button, component, content, dialog, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('stories', 'two-macs');

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

    test('a share the engine refuses shows its words and its hint', async () => {
        // Given - linux-3's Ports over stories, linux-3 on mac-mini
        const result = await component.render(portsOver('stories', 'linux-3'), async (visitor) => {
            await visitor.see(region('Listening, not shared'));

            // When - Share of port 5432 is clicked
            await visitor.click(button('Share :5432'));

            // Then - the engine's refusal as it came, and Dismiss clears it
            const said = content('linux-3 runs on mac-mini', { exact: false });
            await visitor.see(said);
            await visitor.see(content('add --on mac-mini'));
            await visitor.click(button('Dismiss'));
            await visitor.gone(said);
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
