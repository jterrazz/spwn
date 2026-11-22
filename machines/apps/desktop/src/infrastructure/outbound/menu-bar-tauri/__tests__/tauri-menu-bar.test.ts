import { describe, expect, test } from 'vitest';

import { tauriMenuBar } from '../tauri-menu-bar';

type Invoked = { command: string; args: Record<string, unknown> | undefined };

const UNREAD = 'the shell answered in a shape the window does not read';

function fakeShell(answer: () => Promise<unknown>) {
    const invoked: Invoked[] = [];
    const ipc = {
        invoke: async (command: string, args?: Record<string, unknown>) => {
            invoked.push({ args, command });
            return await answer();
        },
    };
    return { invoked, ipc };
}

function refusal(code: string, message: string): Error {
    return Object.assign(new Error(message), { code });
}

describe('tauri menu bar', () => {
    test('each state the shell reads is read as sent, and a show names what it asks', async () => {
        // Given - a shell answering each state in turn
        const answers: unknown[] = [
            { kind: 'shown' },
            { kind: 'hidden' },
            { kind: 'approval' },
            { kind: 'unsupported' },
        ];
        const { invoked, ipc } = fakeShell(async () => await Promise.resolve(answers.shift()));
        const port = tauriMenuBar(ipc);

        // When - the window reads, shows, hides and reads again
        const read = [
            await port.read(),
            await port.show(false),
            await port.show(true),
            await port.read(),
        ];

        // Then - each as sent, from the two commands
        expect(read).toStrictEqual([
            { kind: 'shown' },
            { kind: 'hidden' },
            { kind: 'approval' },
            { kind: 'unsupported' },
        ]);
        expect(invoked).toStrictEqual([
            { args: undefined, command: 'menu_bar_read' },
            { args: { shown: false }, command: 'menu_bar_show' },
            { args: { shown: true }, command: 'menu_bar_show' },
            { args: undefined, command: 'menu_bar_read' },
        ]);
    });

    test('an unknown shape is refused as the step asked, and a refusal keeps its code unless unknown', async () => {
        // Given - a shell answering an unknown shape, then refusing twice
        const answers: (() => Promise<unknown>)[] = [
            async () => await Promise.resolve({ kind: 'later' }),
            async () =>
                await Promise.reject(refusal('menubar.not-switched', 'Operation not permitted')),
            async () => await Promise.reject(refusal('menubar.elsewhere', 'something else')),
        ];
        const { ipc } = fakeShell(async () => {
            const next = answers.shift();
            return next === undefined ? undefined : await next();
        });
        const port = tauriMenuBar(ipc);

        // When - the window reads, then shows twice
        const ends = [await port.read(), await port.show(true), await port.read()];

        // Then - its own read refusal, the shell's, then a read refusal in the shell's words
        expect(ends).toStrictEqual([
            { code: 'menubar.not-read', kind: 'refused', message: UNREAD },
            { code: 'menubar.not-switched', kind: 'refused', message: 'Operation not permitted' },
            { code: 'menubar.not-read', kind: 'refused', message: 'something else' },
        ]);
    });
});
