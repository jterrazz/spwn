import { button, component, content, dialog, file, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('stories', 'two-macs');

function overviewOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Overview' });
}

describe('awake', () => {
    test("a Mac's Awake… opens a sheet of the engine's choices, in its order, and no Stop while none is kept", async () => {
        // Given - This Mac's Overview over two-macs, whose awake.read answers three choices and no Session
        const result = await component.render(
            overviewOver('two-macs', 'macbook-pro'),
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - the sheet names the Mac and offers each choice the engine labelled
                const sheet = dialog('Keep macbook-pro awake');
                await visitor.see(within(sheet, button('1 hr')));
                await visitor.see(within(sheet, button('4 hr')));
                await visitor.see(within(sheet, button('Until 18:00')));
                await visitor.gone(within(sheet, button('Stop keeping awake')));
            },
        );
        await expect(result.tree).toMatch(file('./__expected__/two-macs-awake-sheet.aria.yaml'));
    });

    test('a Mac kept awake adds Stop keeping awake, and a far Mac opens its own sheet', async () => {
        // Given - This Mac's Overview over stories, whose awake.read answers a Session, then the mini's
        const here = await component.render(
            overviewOver('stories', 'macbook-pro'),
            async (visitor) => {
                // When - Awake… is pressed
                await visitor.click(button('Keep macbook-pro awake'));

                // Then - Stop keeping awake follows the choices
                await visitor.see(
                    within(dialog('Keep macbook-pro awake'), button('Stop keeping awake')),
                );
            },
        );
        const far = await component.render(overviewOver('stories', 'mac-mini'), async (visitor) => {
            // When - the mini's Awake… is pressed
            await visitor.click(button('Keep mac-mini awake'));

            // Then - its sheet offers the choices the mini's engine answered
            await visitor.see(within(dialog('Keep mac-mini awake'), content('1 hr')));
        });
        expect(here.errors.text).toBe('');
        expect(far.errors.text).toBe('');
    });
});
