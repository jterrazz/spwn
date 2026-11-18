import { component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'two-macs');

const needs = region('Needs you');

const waiting = region('Upkeep');

function inboxOver(world: string) {
    return pageOver(recordingFor(world), { page: 'inbox' });
}

describe('Inbox', () => {
    test("a world whose needs press lists them under Needs you and the rest under Upkeep, in the engine's head", async () => {
        // Given - the Inbox over two-macs, where npm's sign-in is refused and a background task fails
        const result = await component.render(inboxOver('two-macs'), async (visitor) => {
            // When - This Mac's overview has been read
            await visitor.see(heading('2 things need you.'));
            await visitor.see(
                within(needs, content('com.example.brain.x-snapshot is failing, last exit 2')),
            );
            await visitor.see(within(waiting, content('1 mise update waiting')));
        });

        // Then - the engine's verdict heads the page, then Needs you with the one that presses and Upkeep with the rest
        await expect(result.tree).toMatch(file('./__expected__/two-macs-inbox.aria.yaml'));
    });

    test('a calm Mac is a Needs you panel that lists nothing', async () => {
        // Given - the Inbox over one-mac, where nothing needs the owner
        const result = await component.render(inboxOver('one-mac'), async (visitor) => {
            // When - its overview has been read
            await visitor.see(content('No need is listed.'));
        });

        // Then - the engine's verdict that this Mac is ready, an empty Needs you and an empty Upkeep
        await expect(result.tree).toMatch(file('./__expected__/one-mac-inbox.aria.yaml'));
    });

    test('a Mac that does not answer still shows the needs This Mac reads', async () => {
        // Given - the Inbox over mini-offline
        const result = await component.render(inboxOver('mini-offline'), async (visitor) => {
            // When - its overview has been read
            await visitor.see(heading('Everything is fine.'));
            await visitor.see(within(waiting, content('1 mise update waiting')));
        });

        // Then - the needs of This Mac, and the Mac mini only a choice of the switch
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-inbox.aria.yaml'));
    });
});
