import { button, component, content, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'two-macs');

const needs = region('Needs you');

function inboxOver(world: string) {
    return pageOver(recordingFor(world), { page: 'inbox' });
}

describe('Inbox', () => {
    test('a world with something broken lists each need with its machine and its gesture', async () => {
        // Given - the Inbox over two-macs, where npm's sign-in expired
        const result = await component.render(inboxOver('two-macs'), async (visitor) => {
            // When - This Mac's overview has been read
            await visitor.see(within(needs, content('npm sign-in expired')));
        });

        // Then - the needs in the engine's words, and what the engine does not give said as such
        await expect(result.tree).toMatch(file('./__expected__/two-macs-inbox.aria.yaml'));
    });

    test('a Mac whose machines need nothing says so, and All shows them again', async () => {
        // Given - the Inbox over two-macs
        const result = await component.render(inboxOver('two-macs'), async (visitor) => {
            await visitor.see(within(needs, content('npm sign-in expired')));

            // When - the Mac mini is chosen
            await visitor.click(button('jterrazz-mac-mini'));

            // Then - none of the needs concerns it
            await visitor.see(
                content('None of the needs listed concerns jterrazz-mac-mini or its machines.'),
            );

            // When - all machines are chosen again
            await visitor.click(button('All machines'));

            // Then - the needs are back
            await visitor.see(within(needs, content('npm sign-in expired')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a calm Mac is a Needs you panel that lists nothing', async () => {
        // Given - the Inbox over one-mac, where nothing needs the owner
        const result = await component.render(inboxOver('one-mac'), async (visitor) => {
            // When - its overview has been read
            await visitor.see(content('No need is listed.'));
        });

        // Then - no verdict, an empty Needs you, and Can wait and History not given
        await expect(result.tree).toMatch(file('./__expected__/one-mac-inbox.aria.yaml'));
    });

    test('a Mac that does not answer still shows the needs This Mac reads', async () => {
        // Given - the Inbox over mini-offline
        const result = await component.render(inboxOver('mini-offline'), async (visitor) => {
            // When - its overview has been read
            await visitor.see(heading('No verdict from the engine.'));
            await visitor.see(within(needs, content('npm sign-in expired')));
        });

        // Then - the needs of This Mac, and the Mac mini only a choice of the switch
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-inbox.aria.yaml'));
    });
});
