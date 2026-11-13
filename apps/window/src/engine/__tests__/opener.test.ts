import type { AccountsAccount, PortsSharedPort } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { NO_OPENER, Openable, shareAddressOf, tauriOpener } from '../opener';

type Invoked = { command: string; args: Record<string, unknown> | undefined };

function fakeInvoke(answer: () => Promise<unknown> = async () => await Promise.resolve(null)) {
    const invoked: Invoked[] = [];
    const ipc = {
        invoke: async (command: string, args?: Record<string, unknown>) => {
            invoked.push({ args, command });
            return await answer();
        },
    };
    return { invoked, ipc };
}

const SHARE: PortsSharedPort = {
    bind: '0.0.0.0',
    detail: '',
    guest_port: 5173,
    http: true,
    last_seen_at: null,
    pinned: false,
    process: 'node',
    profile: '',
    reserved: 40_001,
    state: 'shared',
    urls: { local: 'http://127.0.0.1:40001', public: 'https://mac.tail1234.ts.net:40001' },
    vm: 'linux-1',
};

const NPM: AccountsAccount = { hint: 'npm login', provider: 'npm', state: 'broken' };

describe('opener', () => {
    test("a terminal is handed the engine's command, and nothing else", async () => {
        // Given - machines.ssh's answer for linux-1, and the shell's invoke
        const { invoked, ipc } = fakeInvoke();
        const ssh = Openable.ofSsh({
            Address: 'linux-1',
            Command: 'ssh admin@linux-1',
            User: 'admin',
        });

        // When - it is opened
        const opened = ssh ? await tauriOpener(ipc)(ssh) : undefined;

        // Then - one command, its one argument the engine's string
        expect(opened).toStrictEqual({ kind: 'opened' });
        expect(invoked).toStrictEqual([
            { args: { command: 'ssh admin@linux-1' }, command: 'open_in_terminal' },
        ]);
    });

    test("a share's address and an account's fix are the engine's fields as sent", async () => {
        // Given - a shared port with a public address, and npm's fix
        const { invoked, ipc } = fakeInvoke();
        const share = Openable.ofShare(SHARE);
        const fix = Openable.ofFix(NPM);

        // When - each is opened
        const opener = tauriOpener(ipc);
        for (const one of [share, fix]) {
            if (one !== undefined) {
                await opener(one);
            }
        }

        // Then - the public address in a browser, the fix in a terminal
        expect(invoked).toStrictEqual([
            { args: { url: 'https://mac.tail1234.ts.net:40001' }, command: 'open_url' },
            { args: { command: 'npm login' }, command: 'open_in_terminal' },
        ]);
        expect(shareAddressOf(SHARE)).toBe('https://mac.tail1234.ts.net:40001');
    });

    test('an answer that gives no value makes nothing to open', () => {
        // Given - no command, no fix, a raw TCP share and a share with no address
        const answers = [
            Openable.ofSsh({ Address: 'api', Command: '', User: 'admin' }),
            Openable.ofFix({ provider: 'npm', state: 'ok' }),
            Openable.ofShare({ ...SHARE, http: false }),
            Openable.ofShare({ ...SHARE, urls: { local: '', public: '' } }),
        ];

        // When - each is read
        const made = answers.filter((one) => one !== undefined);

        // Then - none can be opened
        expect(made).toStrictEqual([]);
    });

    test("the shell's refusal comes back as a refusal the page draws", async () => {
        // Given - a shell that refuses the address
        const { ipc } = fakeInvoke(
            async () =>
                await Promise.reject(
                    Object.assign(new Error('the window opens no file address'), {
                        code: 'opener.not-a-web-address',
                    }),
                ),
        );
        const share = Openable.ofShare(SHARE);

        // When - it is opened
        const opened = share ? await tauriOpener(ipc)(share) : undefined;

        // Then - refused, with the shell's code and words
        expect(opened?.kind).toBe('refused');
        const error = opened?.kind === 'refused' ? opened.error : undefined;
        expect(error?.wire).toStrictEqual({
            code: 'opener.not-a-web-address',
            message: 'the window opens no file address',
            subject: 'window',
            type: 'refused',
        });
    });

    test('outside the Tauri shell nothing is opened', async () => {
        // Given - a proof build's opener and npm's fix
        const fix = Openable.ofFix(NPM);

        // When - it is opened
        const opened = fix ? await NO_OPENER(fix) : undefined;

        // Then - refused, saying why
        expect(opened?.kind).toBe('refused');
    });
});
