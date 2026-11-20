import { describe, expect, test } from 'vitest';

import type { WindowProgress } from '../../../../domain/window-update';
import { tauriWindowUpdate } from '../tauri-window-update';

type Invoked = { command: string; args: Record<string, unknown> | undefined };
type Sent = (message: unknown) => void;

const SILENT: Sent = () => undefined;

function fakeShell(answer: (command: string, send: Sent) => Promise<unknown>) {
    const invoked: Invoked[] = [];
    let send: Sent = SILENT;
    const ipc = {
        channel: (heard: Sent) => {
            send = heard;
            return 'the channel';
        },
        invoke: async (command: string, args?: Record<string, unknown>) => {
            invoked.push({ args, command });
            return await answer(command, send);
        },
    };
    return { invoked, ipc };
}

const UNREAD = 'the shell answered in a shape the window does not read';

function refusal(code: string, message: string): Error {
    return Object.assign(new Error(message), { code });
}

describe('tauri window update', () => {
    test('each answer of the check is read as the shell sends it', async () => {
        // Given - a shell answering each shape of window_update_check in turn
        const answers: unknown[] = [
            { kind: 'unsupported' },
            { current: '0.21.0', kind: 'current' },
            { current: '0.21.0', date: '2026-11-19T09:00:00Z', kind: 'out', version: '0.22.0' },
            { current: '0.21.0', kind: 'out', version: '0.22.0' },
        ];
        const { invoked, ipc } = fakeShell(async () => await Promise.resolve(answers.shift()));
        const port = tauriWindowUpdate(ipc);

        // When - the window checks four times
        const read = [
            await port.check(),
            await port.check(),
            await port.check(),
            await port.check(),
        ];

        // Then - each as sent, from the one command with no argument
        expect(read).toStrictEqual([
            { kind: 'unsupported' },
            { current: '0.21.0', kind: 'current' },
            { current: '0.21.0', date: '2026-11-19T09:00:00Z', kind: 'out', version: '0.22.0' },
            { current: '0.21.0', kind: 'out', version: '0.22.0' },
        ]);
        expect(invoked.map(({ args, command }) => [command, args])).toStrictEqual(
            Array.from({ length: 4 }, () => ['window_update_check', undefined]),
        );
    });

    test('an answer of an unknown shape is refused, and a refusal keeps its code and message', async () => {
        // Given - a shell answering a shape the window does not know, then refusing
        const answers: unknown[] = [{ kind: 'out', version: '0.22.0' }, { kind: 'later' }];
        const { ipc } = fakeShell(async () => {
            const next = answers.shift();
            return next === undefined
                ? await Promise.reject(refusal('window.update-unpublished', 'not found'))
                : await Promise.resolve(next);
        });
        const port = tauriWindowUpdate(ipc);

        // When - the window checks three times
        const read = [await port.check(), await port.check(), await port.check()];

        // Then - two refusals of its own, then the shell's
        expect(read).toStrictEqual([
            { code: 'window.update-unreachable', kind: 'refused', message: UNREAD },
            { code: 'window.update-unreachable', kind: 'refused', message: UNREAD },
            { code: 'window.update-unpublished', kind: 'refused', message: 'not found' },
        ]);
    });

    test('an install hands the shell a channel, hears its progress, and ends installed', async () => {
        // Given - a shell that sends two chunks, a message it does not know, and the install
        const { invoked, ipc } = fakeShell(async (_command, send) => {
            send({ downloaded: 4_000_000, total: 12_000_000 });
            send({ downloaded: 12_000_000 });
            send({ downloaded: -1 });
            send({ installed: true });
            return await Promise.resolve(null);
        });
        const heard: WindowProgress[] = [];

        // When - the install runs
        const end = await tauriWindowUpdate(ipc).install((step) => {
            heard.push(step);
        });

        // Then - one command with its channel, the known progress in order, then installed
        expect(invoked).toStrictEqual([
            { args: { progress: 'the channel' }, command: 'window_update_install' },
        ]);
        expect(heard).toStrictEqual([
            { downloaded: 4_000_000, kind: 'downloaded', total: 12_000_000 },
            { downloaded: 12_000_000, kind: 'downloaded' },
            { kind: 'installed' },
        ]);
        expect(end).toStrictEqual({ kind: 'installed' });
    });

    test('a refused install keeps the shell code, and a refusal with an unknown code is an install refusal', async () => {
        // Given - a shell refusing as busy, then with a code the window does not know
        const refusals = [
            refusal('window.update-busy', 'the window installs its update'),
            refusal('window.elsewhere', 'something else'),
        ];
        const { ipc } = fakeShell(
            async () => await Promise.reject(refusals.shift() ?? new Error('no refusal left')),
        );
        const port = tauriWindowUpdate(ipc);

        // When - the install is asked twice
        const ends = [await port.install(() => undefined), await port.install(() => undefined)];

        // Then - busy as sent, then an install refusal in the shell's words
        expect(ends).toStrictEqual([
            {
                code: 'window.update-busy',
                kind: 'refused',
                message: 'the window installs its update',
            },
            { code: 'window.update-install', kind: 'refused', message: 'something else' },
        ]);
    });
});
