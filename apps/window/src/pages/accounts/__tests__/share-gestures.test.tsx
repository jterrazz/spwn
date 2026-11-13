import { button, component, content, dialog, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Recording } from '../../../engine/replay';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('stories', 'two-macs');

function accountsOver(recording: Recording, machine: string) {
    return pageOver(recording, { machine, page: 'machine', tab: 'Accounts' });
}

const TOKEN = ['ghp', '0123456789abcdefghijABCDEFGHIJ'].join('_');

function failing(recording: Recording, call: string, message: string): Recording {
    return {
        ...recording,
        gestures: (recording.gestures ?? []).map((gesture) =>
            gesture.request.call === call
                ? { ...gesture, after: [], answers: [{ error: { message, type: 'failed' }, v: 2 }] }
                : gesture,
        ),
    };
}

describe('share gestures', () => {
    test('extend asks nothing, and the share holds 12 h from now', async () => {
        // Given - linux-1's Accounts over stories, its claude copy ending in 24 min
        const result = await component.render(
            accountsOver(recordingFor('stories'), 'linux-1'),
            async (visitor) => {
                const shared = region('Shared with it');
                await visitor.see(within(shared, content('ends in 24 min')));

                // When - Extend of claude is clicked
                await visitor.click(within(shared, button('Extend claude by 12 h')));

                // Then - accounts.list, read again a few seconds on, says the copy holds 12 h
                await visitor.see(within(shared, content('ends in 11 h 59 min')));
                await visitor.gone(within(shared, content('ends in 24 min')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('take back asks first, then the share leaves the machine', async () => {
        // Given - linux-1's Accounts over two-macs, github shared with it
        const result = await component.render(
            accountsOver(recordingFor('two-macs'), 'linux-1'),
            async (visitor) => {
                const shared = region('Shared with it');
                await visitor.see(within(shared, button('Take back github')));

                // When - Take back is clicked, then confirmed
                await visitor.click(within(shared, button('Take back github')));
                const asked = dialog('Take back github from linux-1?');
                await visitor.see(asked);
                await visitor.click(within(asked, button('Take back')));

                // Then - accounts.list, read again, holds claude's share alone
                await visitor.gone(asked);
                await visitor.gone(within(shared, button('Take back github')));
                await visitor.see(within(shared, button('Take back claude')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a cancel at the confirm leaves the share as it was', async () => {
        // Given - linux-1's Accounts over two-macs, two shares from its Mac
        const result = await component.render(
            accountsOver(recordingFor('two-macs'), 'linux-1'),
            async (visitor) => {
                const shared = region('Shared with it');

                // When - Take back is clicked, then cancelled
                await visitor.click(within(shared, button('Take back github')));
                const asked = dialog('Take back github from linux-1?');
                await visitor.click(within(asked, button('Cancel')));

                // Then - no dialog, and github is still shared
                await visitor.gone(asked);
                await visitor.see(within(shared, button('Take back github')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("an engine's failure is shown as it came, and Dismiss clears it", async () => {
        // Given - stories, where the extend of claude fails
        const failed = failing(recordingFor('stories'), 'accounts.share', 'linux-1 is not reached');
        const result = await component.render(accountsOver(failed, 'linux-1'), async (visitor) => {
            // When - Extend of claude is clicked
            await visitor.click(button('Extend claude by 12 h'));

            // Then - the engine's words, then nothing once dismissed
            const said = content('linux-1 is not reached', { exact: false });
            await visitor.see(said);
            await visitor.click(button('Dismiss'));
            await visitor.gone(said);
        });
        expect(result.errors.text).toBe('');
    });

    test('an answer that holds a token is withheld, its value never drawn', async () => {
        // Given - stories, where the extend of claude fails with a token in its words
        const failed = failing(recordingFor('stories'), 'accounts.share', `export ${TOKEN}`);
        const result = await component.render(accountsOver(failed, 'linux-1'), async (visitor) => {
            // When - Extend of claude is clicked
            await visitor.click(button('Extend claude by 12 h'));

            // Then - the answer is said withheld
            await visitor.see(content('withheld', { exact: false }));
        });
        expect(result.content.text).not.toContain(TOKEN);
    });
});
