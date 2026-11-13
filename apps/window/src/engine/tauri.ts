import { Channel, invoke } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';

import { answerOf, helloOf, isRecord, wireErrorOf } from './frame';
import type { Link, Receiver, Transport } from './transport';

export type Ipc = {
    invoke: (command: string, args?: Record<string, unknown>) => Promise<unknown>;
    listen: (event: string, heard: (payload: unknown) => void) => Promise<() => void>;
    channel: (heard: (message: unknown) => void) => unknown;
};

const TAURI: Ipc = {
    channel: (heard) => new Channel(heard),
    invoke: async (command, args) => await invoke(command, args),
    listen: async (event, heard) =>
        await listen(event, (emitted) => {
            heard(emitted.payload);
        }),
};

function linkOf(value: unknown): Link | undefined {
    if (!isRecord(value) || typeof value.socket !== 'string') {
        return undefined;
    }
    const { socket } = value;
    if (value.kind === 'connecting') {
        return { kind: 'connecting', socket };
    }
    if (value.kind === 'absent') {
        return {
            kind: 'absent',
            reason: typeof value.reason === 'string' ? value.reason : '',
            retryIn: typeof value.retryIn === 'number' ? value.retryIn : 0,
            socket,
        };
    }
    if (value.kind === 'open') {
        const answer = answerOf(value.answer);
        return {
            hello: answer && 'out' in answer ? helloOf(answer.out) : undefined,
            kind: 'open',
            refused: answer && 'error' in answer ? wireErrorOf(answer.error) : undefined,
            socket,
        };
    }
    return undefined;
}

async function opened(ipc: Ipc, request: unknown, channel: unknown, receiver: Receiver) {
    try {
        const handle = await ipc.invoke('engine_open', { channel, request });
        return typeof handle === 'number' ? handle : null;
    } catch (error) {
        receiver.end(String(error), false);
        return null;
    }
}

async function closed(ipc: Ipc, handle: Promise<null | number>) {
    const open = await handle;
    if (open !== null) {
        await ipc.invoke('engine_close', { handle: open }).catch(() => undefined);
    }
}

type Linking = { live: boolean; heard: boolean; show: (value: unknown) => void };

// invariant: the bridge emits the link on a change only, so it is read once listening has started: a change before
// invariant: is in the answer, one after in an event. An event heard first is newer than the answer, which is dropped.
async function linked(ipc: Ipc, linking: Linking, unlisten: Promise<() => void>) {
    const value = await unlisten
        .then(async () => await ipc.invoke('engine_link'))
        .catch(() => undefined);
    if (!linking.heard) {
        linking.show(value);
    }
}

async function unlistened(unlisten: Promise<() => void>) {
    const stop = await unlisten.catch(() => undefined);
    stop?.();
}

export function tauriTransport(ipc: Ipc = TAURI): Transport {
    return {
        link: (listener) => {
            const linking: Linking = {
                heard: false,
                live: true,
                show: (value) => {
                    const link = linkOf(value);
                    if (linking.live && link) {
                        listener(link);
                    }
                },
            };
            const unlisten = ipc.listen('engine://link', (payload) => {
                linking.heard = true;
                linking.show(payload);
            });
            void linked(ipc, linking, unlisten);
            return () => {
                linking.live = false;
                void unlistened(unlisten);
            };
        },
        open: (request, receiver) => {
            let shut = false;
            const channel = ipc.channel((message: unknown) => {
                if (shut || !isRecord(message)) {
                    return;
                }
                if ('frame' in message) {
                    receiver.frame(message.frame);
                } else if (typeof message.end === 'string') {
                    receiver.end(message.end, true);
                }
            });
            const handle = opened(ipc, request, channel, receiver);
            return {
                close: () => {
                    shut = true;
                    void closed(ipc, handle);
                },
            };
        },
    };
}
