import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { tauriTransport } from '../tauri';
import type { Ipc } from '../tauri';
import type { Link } from '../transport';

const SOCKET = '/run/engine.sock';
const CONNECTING = { kind: 'connecting', socket: SOCKET };
const OPEN = {
    answer: {
        id: 'hello',
        out: { pid: 7, started_at: '2026-11-13T09:00:00Z', version: 'v1' },
        v: 2,
    },
    kind: 'open',
    socket: SOCKET,
};
const ABSENT = { kind: 'absent', reason: 'connection refused', retryIn: 2000, socket: SOCKET };

type Invoked = { command: string; args: Record<string, unknown> | undefined };

// value: the bridge as Rust runs it: a link held and emitted on a change only, to the listeners registered by then;
// value: `listen` registers after `registering` ms, and `engine_link` reads the link when asked, answering `reading` ms later.
function fakeBridge({ registering = 0, reading = 0 } = {}) {
    let link: unknown = CONNECTING;
    const listeners = new Set<(payload: unknown) => void>();
    const invoked: Invoked[] = [];
    const channels: ((message: unknown) => void)[] = [];
    const answers = new Map<string, () => Promise<unknown>>();
    const ipc: Ipc = {
        channel: (heard) => {
            channels.push(heard);
            return heard;
        },
        invoke: async (command, args) => {
            invoked.push({ args, command });
            const answer = answers.get(command);
            if (answer) {
                return await answer();
            }
            const read = command === 'engine_link' ? link : undefined;
            return await new Promise((resolve) => {
                setTimeout(() => {
                    resolve(read);
                }, reading);
            });
        },
        listen: async (_event, heard) =>
            await new Promise((resolve) => {
                setTimeout(() => {
                    listeners.add(heard);
                    resolve(() => {
                        listeners.delete(heard);
                    });
                }, registering);
            }),
    };
    const emit = (next: unknown) => {
        link = next;
        for (const listener of listeners) {
            listener(next);
        }
    };
    return { answers, channels, emit, invoked, ipc, listening: () => listeners.size };
}

function heardBy(transport: ReturnType<typeof tauriTransport>) {
    const links: Link['kind'][] = [];
    const stop = transport.link((link) => {
        links.push(link.kind);
    });
    return { links, stop };
}

describe('tauri transport', () => {
    test('a link that changes before listening has started is read once it has', async () => {
        // Given - a bridge whose listener registers 20 ms after it is asked
        using _ = clock.run('2026-11-13T10:00:00Z');
        const bridge = fakeBridge({ registering: 20 });
        const { links } = heardBy(tauriTransport(bridge.ipc));

        // When - the engine answers 10 ms in, an event emitted to nobody, and no change follows
        await clock.advance(10);
        bridge.emit(OPEN);
        await clock.advance(50);

        // Then - the window reads the link once listening, and shows it open, never stuck on connecting
        expect(links).toStrictEqual(['open']);
    });

    test('an event heard while the link is being read wins over the older answer', async () => {
        // Given - a bridge listening at once, whose link read takes 30 ms
        using _ = clock.run('2026-11-13T10:00:00Z');
        const bridge = fakeBridge({ reading: 30 });
        const { links } = heardBy(tauriTransport(bridge.ipc));
        await clock.advance(1);

        // When - the link goes open, then absent, while the read is under way
        bridge.emit(OPEN);
        bridge.emit(ABSENT);
        await clock.advance(50);

        // Then - both events are shown in order, and the read's answer is not shown after them
        expect(links).toStrictEqual(['open', 'absent']);
    });

    test('a link read that fails leaves no rejection unhandled, and the next event is shown', async () => {
        // Given - a bridge whose engine_link rejects
        using _ = clock.run('2026-11-13T10:00:00Z');
        const bridge = fakeBridge();
        bridge.answers.set('engine_link', async () => await Promise.reject(new Error('ipc down')));
        const { links } = heardBy(tauriTransport(bridge.ipc));
        await clock.advance(5);

        // When - the link changes afterwards
        bridge.emit(ABSENT);

        // Then - the event is shown, nothing before it
        expect(links).toStrictEqual(['absent']);
    });

    test('a link left stops listening and shows nothing more', async () => {
        // Given - a transport whose link is listened to
        using _ = clock.run('2026-11-13T10:00:00Z');
        const bridge = fakeBridge();
        const { links, stop } = heardBy(tauriTransport(bridge.ipc));
        await clock.advance(5);

        // When - it is left, then the link changes
        stop();
        await clock.advance(5);
        bridge.emit(OPEN);

        // Then - the listener was removed, and nothing was shown after the first read
        expect(bridge.listening()).toBe(0);
        expect(links).toStrictEqual(['connecting']);
    });

    test("an exchange forwards the channel's frames and its end, and closes its handle", async () => {
        // Given - a bridge that opens exchange 4
        using _ = clock.run('2026-11-13T10:00:00Z');
        const bridge = fakeBridge();
        bridge.answers.set('engine_open', async () => await Promise.resolve(4));
        const heard: unknown[] = [];
        const exchange = tauriTransport(bridge.ipc).open(
            { call: 'engine.read', id: '1', in: {}, v: 2 },
            {
                end: (reason, sent) => {
                    heard.push({ reason, sent });
                },
                frame: (frame) => {
                    heard.push(frame);
                },
            },
        );
        await clock.advance(1);

        // When - the channel carries a frame, its end, then the exchange is closed and a late frame comes
        const channel = bridge.channels[0];
        channel?.({ frame: { id: '1', out: {}, v: 2 } });
        channel?.({ end: 'answered' });
        exchange.close();
        channel?.({ frame: { id: '1', out: { late: true }, v: 2 } });
        await clock.advance(1);

        // Then - the frame and the end were heard, the late frame not, and handle 4 was closed
        expect(heard).toStrictEqual([
            { id: '1', out: {}, v: 2 },
            { reason: 'answered', sent: true },
        ]);
        expect(bridge.invoked.at(-1)).toStrictEqual({
            args: { handle: 4 },
            command: 'engine_close',
        });
    });

    test('a cancel is handed to the bridge on its exchange, and the exchange closes after it', async () => {
        // Given - a watch the bridge opened as exchange 5, whose cancel the bridge sends only when released
        using _ = clock.run('2026-11-13T10:00:00Z');
        const bridge = fakeBridge();
        bridge.answers.set('engine_open', async () => await Promise.resolve(5));
        const cancel: { release?: (value: unknown) => void } = {};
        bridge.answers.set(
            'engine_cancel',
            async () =>
                await new Promise((resolve) => {
                    cancel.release = resolve;
                }),
        );
        const exchange = tauriTransport(bridge.ipc).open(
            { id: '3', in: {}, v: 2, watch: 'machines.new' },
            { end: () => {}, frame: () => {} },
        );
        await clock.advance(1);

        // When - it is cancelled then closed at once
        exchange.cancel({ cancel: true, id: '3', v: 2 });
        exchange.close();
        await clock.advance(10);
        const whileCancelling = bridge.invoked.map((one) => one.command);
        cancel.release?.(undefined);
        await clock.advance(1);

        // Then - the cancel named exchange 5 and its id, and the close was asked only once it was sent
        expect(whileCancelling).toStrictEqual(['engine_open', 'engine_cancel']);
        expect(bridge.invoked.slice(1)).toStrictEqual([
            {
                args: { frame: { cancel: true, id: '3', v: 2 }, handle: 5 },
                command: 'engine_cancel',
            },
            { args: { handle: 5 }, command: 'engine_close' },
        ]);
    });

    test('an exchange the bridge cannot open ends unsent, and closing it closes nothing', async () => {
        // Given - a bridge whose engine_open rejects as no engine answering
        using _ = clock.run('2026-11-13T10:00:00Z');
        const bridge = fakeBridge();
        bridge.answers.set(
            'engine_open',
            async () => await Promise.reject(new Error('no engine answers at /run/engine.sock')),
        );
        const ends: { reason: string; sent: boolean }[] = [];

        // When - a call is opened, then closed
        const exchange = tauriTransport(bridge.ipc).open(
            { call: 'engine.read', id: '2', in: {}, v: 2 },
            {
                end: (reason, sent) => {
                    ends.push({ reason, sent });
                },
                frame: () => {},
            },
        );
        await clock.advance(1);
        exchange.close();
        await clock.advance(1);

        // Then - it ended unsent with the bridge's reason, and no engine_close was asked
        expect(ends).toStrictEqual([
            { reason: 'Error: no engine answers at /run/engine.sock', sent: false },
        ]);
        expect(bridge.invoked.map((one) => one.command)).toStrictEqual(['engine_open']);
    });
});
