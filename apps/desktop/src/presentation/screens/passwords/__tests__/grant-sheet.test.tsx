import { button, component, content, dialog, file, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';

const recordingFor = await recordingsOf('two-macs', 'presence-refused');

function entryOver(item: string) {
    return pageOver(recordingFor('two-macs'), { item, page: 'passwords' });
}

function refusedEntry(item: string) {
    return pageOver(recordingFor('presence-refused'), { item, page: 'passwords' });
}

describe('grant sheet', () => {
    test("the picker draws every target the engine offers, one that does not fit held with the engine's line", async () => {
        // Given - GitHub deploy's page over two-macs
        const result = await component.render(entryOver('pw-1'), async (visitor) => {
            // When - Grant… is clicked
            await visitor.click(button('Grant…'));

            // Then - the machines, linux-1 to pick, api and infra held with why, and the engine's presence sentence
            const sheet = dialog('Grant GitHub deploy');
            await visitor.see(within(sheet, button('linux-1', { exact: false })));
            await visitor.see(within(sheet, content('Start the machine first')));
            await visitor.see(within(sheet, content('A server keeps its own sign-ins')));
            await visitor.see(within(sheet, content('Touch ID will ask you', { exact: false })));
        });
        expect(result.errors.text).toBe('');
        expect(result.content.text).not.toContain('until');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-pw-1-grant.aria.yaml'));
    });

    test('a target that fits is picked and granted, and the sheet closes on the engine’s answer', async () => {
        // Given - GitHub deploy's picker over two-macs
        const result = await component.render(entryOver('pw-1'), async (visitor) => {
            await visitor.click(button('Grant…'));
            const sheet = dialog('Grant GitHub deploy');

            // When - linux-1 is picked and granted
            await visitor.click(within(sheet, button('linux-1', { exact: false })));
            await visitor.click(within(sheet, button('Grant to linux-1')));

            // Then - the sheet is gone, and the password is granted to linux-1
            await visitor.gone(sheet);
            await visitor.see(within(region('Granted to'), content('linux-1')));
        });
        expect(result.errors.text).toBe('');
    });

    test("a grant refused at the presence check is said in the sheet's foot, and nothing is granted", async () => {
        // Given - GitHub deploy's picker over presence-refused, whose Mac does not confirm the owner is there
        const result = await component.render(refusedEntry('pw-1'), async (visitor) => {
            await visitor.click(button('Grant…'));
            const sheet = dialog('Grant GitHub deploy');
            await visitor.click(within(sheet, button('linux-1', { exact: false })));

            // When - the grant to linux-1 is sent
            await visitor.click(within(sheet, button('Grant to linux-1')));

            // Then - the engine's refusal is said in the sheet, which stays open, and the password is granted to nothing
            await visitor.see(
                within(sheet, content('did not confirm you are there', { exact: false })),
            );
            await visitor.gone(within(region('Granted to'), content('linux-1')));
        });
        expect(result.errors.text).toBe('');
    });

    test('a target that does not fit cannot be picked', async () => {
        // Given - Claude Max's picker over two-macs, every machine signed in or held
        const result = await component.render(entryOver('pw-3'), async (visitor) => {
            await visitor.click(button('Grant…'));
            const sheet = dialog('Grant Claude Max');

            // When - the targets are drawn
            await visitor.see(within(sheet, content('It has its own sign-in')));

            // Then - no grant can be sent
            await visitor.see(within(sheet, button('Grant')));
        });
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-pw-3-grant.aria.yaml'));
    });
});
