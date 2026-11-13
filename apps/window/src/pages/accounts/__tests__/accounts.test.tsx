import { button, component, content, file, heading, region } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { tauriOpener } from '../../../engine/opener';
import { recordingOf } from '../../../engine/replay';
import type { Recording } from '../../../engine/replay';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('mini-workspace', 'stories', 'two-macs');

function accountsOver(recording: Recording, machine: string) {
    return pageOver(recording, { machine, page: 'machine', tab: 'Accounts' });
}

const TOKEN = ['ghp', '0123456789abcdefghijABCDEFGHIJ'].join('_');

type Invoked = { command: string; args: Record<string, unknown> | undefined };

function leaking(recording: Recording): Recording {
    const planted = `"provider":"github","detail":"${TOKEN}"`;
    const text = JSON.stringify(recording).replaceAll('"provider":"github"', planted);
    const parsed: unknown = JSON.parse(text);
    return recordingOf(parsed);
}

describe('Accounts tab', () => {
    test("a Mac's accounts carry the engine's states, and its shares their machines", async () => {
        // Given - This Mac's Accounts over two-macs, npm refused, two accounts shared with linux-1
        const result = await component.render(
            accountsOver(recordingFor('two-macs'), 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - the watch has answered past its first, pending read
                await visitor.see(content('Refused'));
            },
        );

        // Then - every account and both shares, each with its gestures
        await expect(result.tree).toMatch(file('./__expected__/two-macs-this-mac.aria.yaml'));
    });

    test('a share says for how long it still holds, with who it is on its Mac', async () => {
        // Given - linux-1's Accounts over stories, its claude copy expiring
        const result = await component.render(
            accountsOver(recordingFor('stories'), 'linux-1'),
            async (visitor) => {
                // When - the accounts and the drift have been read
                await visitor.see(content('ends in 24 min'));
                await visitor.see(region('Differs from its file'));
            },
        );

        // Then - the shares, the share list held, no unread row, and two copies the file names not, each a bare hint
        expect(result.content.text).toContain('Not in the file');
        expect(result.content.text).not.toMatch(/claude\s*Not in the file/u);
        expect(result.content.text).not.toContain('?');
        expect(result.content.text).not.toContain('Accounts on linux-1');
        await expect(result.tree).toMatch(file('./__expected__/stories-linux-1.aria.yaml'));
    });

    test('a server on a far Mac whose accounts are not answered says so', async () => {
        // Given - infra's Accounts over mini-workspace, its Mac never asked by the recorder
        const result = await component.render(
            accountsOver(recordingFor('mini-workspace'), 'infra'),
            async (visitor) => {
                // When - the watch has ended unanswered
                await visitor.see(heading('No answer from the engine.'));
            },
        );

        // Then - the page holds the refusal, and no account row
        await expect(result.tree).toMatch(file('./__expected__/mini-workspace-infra.aria.yaml'));
    });

    test('a machine nothing is shared with draws no panel of shares', async () => {
        // Given - api's Accounts over two-macs, which holds none
        const result = await component.render(
            accountsOver(recordingFor('two-macs'), 'api'),
            async (visitor) => {
                // When - the accounts have been read
                await visitor.see(region('Can be shared'));

                // Then - no Shared with it panel stands empty
                await visitor.gone(region('Shared with it'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a field that holds a token is named, and its value never drawn', async () => {
        // Given - two-macs, with a token planted in every github account's detail
        const leaked = leaking(recordingFor('two-macs'));
        const result = await component.render(
            accountsOver(leaked, 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - the accounts have been read
                await visitor.see(content('Refused'));
            },
        );

        // Then - the page names the detail, and the token is nowhere in it
        expect(result.content.text).toContain('Not shown: detail held what looks like a secret');
        expect(result.content.text).not.toContain(TOKEN);
    });

    test("sign in runs the engine's fix in a terminal, its command still shown with Copy", async () => {
        // Given - This Mac's Accounts over two-macs, npm refused with `npm login` as its fix
        const invoked: Invoked[] = [];
        const opener = tauriOpener({
            invoke: async (command, args) => {
                invoked.push({ args, command });
                return await Promise.resolve(null);
            },
        });
        const page = pageOver(
            recordingFor('two-macs'),
            { machine: 'jterrazz-macbook-pro', page: 'machine', tab: 'Accounts' },
            opener,
        );
        const result = await component.render(page, async (visitor) => {
            await visitor.see(content('Refused'));
            await visitor.see(button('Copy npm login'));

            // When - Sign in is clicked
            await visitor.click(button('Sign in'));

            // Then - the row keeps its state and its command
            await visitor.see(button('Copy npm login'));
        });
        expect(invoked).toStrictEqual([
            { args: { command: 'npm login' }, command: 'open_in_terminal' },
        ]);
        expect(result.content.text).not.toContain('Done');
    });
});
