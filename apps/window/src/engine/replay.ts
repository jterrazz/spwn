import type { EngineHello, Request } from '@jterrazz/station-api';

import { isRecord, keyOf } from './frame';
import type { Link, Exchange as Opened, Receiver, Transport } from './transport';

export type Exchange = { request: Record<string, unknown>; answers: readonly unknown[] };

export type GestureExchange = Exchange & { after: readonly Exchange[] };

export type Recording = {
    world: string;
    station: string;
    exchanges: readonly Exchange[];
    gestures?: readonly GestureExchange[];
};

function exchangeOf(world: string, value: unknown): Exchange {
    if (!isRecord(value) || !isRecord(value.request) || !Array.isArray(value.answers)) {
        throw new Error(`An exchange of ${world} holds a request and its answers.`);
    }
    const answers: readonly unknown[] = value.answers;
    return { answers, request: value.request };
}

function gestureOf(world: string, value: unknown): GestureExchange {
    const exchange = exchangeOf(world, value);
    if (!isRecord(value) || !Array.isArray(value.after)) {
        throw new Error(`A gesture of ${world} lists the reads made after it.`);
    }
    const after: readonly unknown[] = value.after;
    return { ...exchange, after: after.map((read) => exchangeOf(world, read)) };
}

export function recordingOf(value: unknown): Recording {
    if (
        !isRecord(value) ||
        typeof value.world !== 'string' ||
        typeof value.station !== 'string' ||
        !Array.isArray(value.exchanges)
    ) {
        throw new Error('A recording names its world and its station, and lists its exchanges.');
    }
    const { world } = value;
    const gestures: readonly unknown[] = Array.isArray(value.gestures) ? value.gestures : [];
    const exchanges: readonly unknown[] = value.exchanges;
    return {
        exchanges: exchanges.map((exchange) => exchangeOf(world, exchange)),
        gestures: gestures.map((gesture) => gestureOf(world, gesture)),
        station: value.station,
        world,
    };
}

function withId(frame: unknown, id: string): unknown {
    return isRecord(frame) ? { ...frame, id } : frame;
}

type Played = { frames: readonly unknown[]; gap: number; first: number; onLast?: () => void };

function play(receiver: Receiver, id: string, { frames, gap, first, onLast }: Played) {
    const timers: ReturnType<typeof setTimeout>[] = [];
    const last = frames.length - 1;
    for (const [index, frame] of frames.entries()) {
        timers.push(
            setTimeout(
                () => {
                    if (index === last) {
                        onLast?.();
                    }
                    receiver.frame(withId(frame, id));
                },
                first + index * gap,
            ),
        );
    }
    timers.push(
        setTimeout(
            () => {
                if (last < 0) {
                    onLast?.();
                }
                receiver.end('answered', true);
            },
            first + frames.length * gap,
        ),
    );
    return timers;
}

// invariant: a request the world holds no answer for ends with no reason, as an engine that never answers does: the
// invariant: window draws its own no-answer state, and the world's name never reaches the page.
function reasonOf(shown: Link): string {
    if (shown.kind === 'absent') {
        return shown.reason;
    }
    return shown.kind === 'open' ? '' : 'connecting';
}

export type Pace = {
    gestureFrame: number;
    // value: how long the first answer to a request waits, by the request and how often its key was asked before.
    latency?: (request: Request, asked: number) => number;
};

// value: a replay the spec drives: the link flips as shown, a drop ends every open exchange after its request was
// value: sent, a restart drops them and answers from the world the engine came back with, and `open` counts the exchanges a page holds at once.
export type Stage = {
    transport: Transport;
    show: (link: Link) => void;
    drop: () => void;
    restart: (hello: EngineHello, after?: Recording) => void;
    open: () => number;
    asked: () => readonly Request[];
};

type Answers = {
    reads: Map<string, readonly unknown[]>;
    gestures: Map<string, GestureExchange>;
};

function answersOf(recording: Recording): Answers {
    return {
        gestures: new Map(
            (recording.gestures ?? []).map((gesture) => [keyOf(gesture.request), gesture]),
        ),
        reads: new Map(
            recording.exchanges.map((exchange) => [keyOf(exchange.request), exchange.answers]),
        ),
    };
}

type Live = { receiver: Receiver; timers: ReturnType<typeof setTimeout>[] };

function stopped(held: Live): void {
    for (const timer of held.timers) {
        clearTimeout(timer);
    }
}

// invariant: a recorded gesture changes the world it was recorded in: once its last answer is played, every read
// invariant: recorded after it in the same world answers as it did then, and the others as before.
class Staging {
    private shown: Link;
    private answers: Answers;
    private readonly recording: Recording;
    private readonly pace: Pace | undefined;
    private readonly listeners = new Set<(link: Link) => void>();
    private readonly live = new Set<Live>();
    private readonly times = new Map<string, number>();
    readonly requests: Request[] = [];

    constructor(recording: Recording, link: Link | undefined, pace: Pace | undefined) {
        this.recording = recording;
        this.pace = pace;
        this.answers = answersOf(recording);
        this.shown = link ?? {
            hello: undefined,
            kind: 'open',
            refused: undefined,
            socket: this.socket(),
        };
    }

    socket(): string {
        return `replay of ${this.recording.world}`;
    }

    show(next: Link): void {
        this.shown = next;
        for (const listener of this.listeners) {
            listener(next);
        }
    }

    drop(): void {
        for (const held of this.live) {
            stopped(held);
            held.receiver.end('the engine closed the connection', true);
        }
        this.live.clear();
    }

    restart(hello: EngineHello, after: Recording | undefined): void {
        this.drop();
        this.answers = answersOf(after ?? this.recording);
        this.show({ hello, kind: 'open', refused: undefined, socket: this.socket() });
    }

    open(): number {
        return this.live.size;
    }

    link(listener: (link: Link) => void): () => void {
        this.listeners.add(listener);
        const timer = setTimeout(() => {
            listener(this.shown);
        }, 0);
        return () => {
            clearTimeout(timer);
            this.listeners.delete(listener);
        };
    }

    exchange(request: Request, receiver: Receiver): Opened {
        this.requests.push(request);
        const key = keyOf(request);
        const played = this.playing(request, key);
        const held: Live = { receiver, timers: [] };
        const tracked: Receiver = {
            end: (reason, sent) => {
                this.live.delete(held);
                receiver.end(reason, sent);
            },
            frame: receiver.frame,
        };
        const { shown } = this;
        held.timers = played
            ? play(tracked, request.id, played)
            : [
                  setTimeout(() => {
                      tracked.end(reasonOf(shown), false);
                  }, 0),
              ];
        this.live.add(held);
        return {
            cancel: (frame) => {
                this.requests.push(frame);
            },
            close: () => {
                stopped(held);
                this.live.delete(held);
            },
        };
    }

    private firstOf(request: Request, key: string): number {
        const before = this.times.get(key) ?? 0;
        this.times.set(key, before + 1);
        return this.pace?.latency?.(request, before) ?? 0;
    }

    private playing(request: Request, key: string): Played | undefined {
        if (this.shown.kind !== 'open') {
            return undefined;
        }
        const first = this.firstOf(request, key);
        const { gestures, reads } = this.answers;
        const gesture = gestures.get(key);
        if (gesture) {
            const changed = () => {
                for (const read of gesture.after) {
                    reads.set(keyOf(read.request), read.answers);
                }
            };
            return {
                first,
                frames: gesture.answers,
                gap: this.pace?.gestureFrame ?? 0,
                onLast: changed,
            };
        }
        const recorded = reads.get(key);
        return recorded ? { first, frames: recorded, gap: 0 } : undefined;
    }
}

export function stagedReplay(recording: Recording, link?: Link, pace?: Pace): Stage {
    const staging = new Staging(recording, link, pace);
    return {
        asked: () => staging.requests,
        drop: () => {
            staging.drop();
        },
        open: () => staging.open(),
        restart: (hello, after) => {
            staging.restart(hello, after);
        },
        show: (next) => {
            staging.show(next);
        },
        transport: {
            link: (listener) => staging.link(listener),
            open: (request, receiver) => staging.exchange(request, receiver),
        },
    };
}

export function replayTransport(recording: Recording, link?: Link, pace?: Pace): Transport {
    return stagedReplay(recording, link, pace).transport;
}
