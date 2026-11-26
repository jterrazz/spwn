import type { PortsSharedPort } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { Openable, shareAddressOf } from '../../../../presentation/kit/openable';
import { NO_OPENER, tauriOpener } from '../tauri-opener';

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

describe('tauri opener', () => {
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

    test("a share's address is the engine's field as sent", async () => {
        // Given - a shared port with a public address
        const { invoked, ipc } = fakeInvoke();
        const share = Openable.ofShare(SHARE, true);

        // When - it is opened
        if (share !== undefined) {
            await tauriOpener(ipc)(share);
        }

        // Then - the public address, in a browser
        expect(invoked).toStrictEqual([
            { args: { url: 'https://mac.tail1234.ts.net:40001' }, command: 'open_url' },
        ]);
        expect(shareAddressOf(SHARE)).toBe('https://mac.tail1234.ts.net:40001');
    });

    test("a pane is handed the engine's URL to the shell's settings command", async () => {
        // Given - a need that names its pane, and the shell's invoke
        const { invoked, ipc } = fakeInvoke();
        const pane = Openable.ofPane({
            Pane: 'x-apple.systempreferences:com.apple.Network-Settings.extension?Firewall',
        });

        // When - it is opened
        const opened = pane ? await tauriOpener(ipc)(pane) : undefined;

        // Then - one command, the pane as sent
        expect(opened).toStrictEqual({ kind: 'opened' });
        expect(invoked).toStrictEqual([
            {
                args: {
                    url: 'x-apple.systempreferences:com.apple.Network-Settings.extension?Firewall',
                },
                command: 'open_settings',
            },
        ]);
    });

    test('a line the owner runs goes to the waiting terminal command, never the running one', async () => {
        // Given - a need whose command is the owner's to run
        const { invoked, ipc } = fakeInvoke();
        const line = Openable.ofFixLine({
            Command: 'sudo launchctl bootout system/x',
            InTerminal: true,
        });

        // When - it is opened
        if (line) {
            await tauriOpener(ipc)(line);
        }

        // Then - open_fix_in_terminal alone
        expect(invoked).toStrictEqual([
            {
                args: { command: 'sudo launchctl bootout system/x' },
                command: 'open_fix_in_terminal',
            },
        ]);
    });

    test('a need with no pane, or a command nobody asked the owner to run, opens nothing', () => {
        // Given - a need with no pane, a command the owner is not asked to run, and a flag with no command
        const none = Openable.ofPane({ Pane: '' });
        const notAsked = Openable.ofFixLine({ Command: 'station disk', InTerminal: false });
        const empty = Openable.ofFixLine({ Command: '', InTerminal: true });

        // Then - none of them opens anything
        expect([none, notAsked, empty]).toStrictEqual([undefined, undefined, undefined]);
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
        const share = Openable.ofShare(SHARE, true);

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
        // Given - a proof build's opener and machines.ssh's answer for linux-1
        const ssh = Openable.ofSsh({
            Address: 'linux-1',
            Command: 'ssh admin@linux-1',
            User: 'admin',
        });

        // When - it is opened
        const opened = ssh ? await NO_OPENER(ssh) : undefined;

        // Then - refused, saying why
        expect(opened?.kind).toBe('refused');
    });
});
