import { describe, expect, test } from 'vitest';

import type { CoreLine } from '../../../../application/ports/core.port';
import { tauriCore } from '../tauri-core';

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

describe('tauri core', () => {
    test('an install hands the shell a channel and hears its lines, one sent after the answer included, before its end', async () => {
        // Given - a shell that sends one line, answers having counted two, then sends the second
        const { invoked, ipc } = fakeShell(async (_command, send) => {
            send({ stream: 'out', text: 'Version: v0.21.0' });
            setImmediate(() => {
                send({ stream: 'err', text: 'Error: nothing installed' });
            });
            return await Promise.resolve({ cancelled: false, code: 1, lines: 2 });
        });
        const heard: CoreLine[] = [];

        // When - the install runs to its end
        const end = await tauriCore(ipc).install((line) => {
            heard.push(line);
        }).ended;

        // Then - one command with its channel, both lines as sent, then the exit code
        expect(invoked).toStrictEqual([
            { args: { channel: 'the channel' }, command: 'core_install' },
        ]);
        expect(heard).toStrictEqual([
            { stream: 'out', text: 'Version: v0.21.0' },
            { stream: 'err', text: 'Error: nothing installed' },
        ]);
        expect(end).toStrictEqual({ cancelled: false, code: 1, kind: 'ended' });
    });

    test("the shell's refusal is a refusal in its own words, and a cancel is one command", async () => {
        // Given - a shell that refuses a second run
        const { invoked, ipc } = fakeShell(async (command) =>
            command === 'core_start'
                ? await Promise.reject(Object.assign(new Error('core.busy'), { code: 'core.busy' }))
                : await Promise.resolve(null),
        );
        const core = tauriCore(ipc);

        // When - a start is asked, then cancelled
        const run = core.start(() => undefined);
        run.cancel();
        const end = await run.ended;

        // Then - refused with the shell's words, and the cancel sent as such
        expect(end).toStrictEqual({ kind: 'refused', message: 'core.busy' });
        expect(invoked.map((one) => one.command)).toStrictEqual(['core_start', 'core_cancel']);
    });

    test('a start hands the shell the station the engine named, and none when it named none', async () => {
        // Given - a shell that ends every run at once
        const { invoked, ipc } = fakeShell(
            async () => await Promise.resolve({ cancelled: false, code: 0, lines: 0 }),
        );
        const core = tauriCore(ipc);

        // When - one start names a station, one names none
        await core.start(() => undefined, '/Users/you/.local/bin/station').ended;
        await core.start(() => undefined).ended;

        // Then - the first carries it beside the channel, the second the channel alone
        expect(invoked).toStrictEqual([
            {
                args: { channel: 'the channel', executable: '/Users/you/.local/bin/station' },
                command: 'core_start',
            },
            { args: { channel: 'the channel' }, command: 'core_start' },
        ]);
    });

    test('the core is present only when the shell says true', async () => {
        // Given - a shell whose file check answers true, then one that answers anything else
        const yes = fakeShell(async () => await Promise.resolve(true));
        const other = fakeShell(async () => await Promise.resolve('yes'));

        // When - each is asked
        const present = [await tauriCore(yes.ipc).present(), await tauriCore(other.ipc).present()];

        // Then - one present, one not
        expect(present).toStrictEqual([true, false]);
        expect(yes.invoked).toStrictEqual([{ args: undefined, command: 'core_present' }]);
    });
});
