import type { EngineHello, Request } from '@jterrazz/station-api';
import { clock } from '@jterrazz/test';

import { Client } from '../../../infrastructure/outbound/engine-link/client';
import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';
import type { Link } from '../../ports/engine.port';
import type { PresencePort } from '../../ports/presence.port';
import type { Asked, Reading } from '../entry';
import { StateMirror } from '../mirror';
import { ReadStore } from '../store';
import type { StoreTiming } from '../store';

type Opened = { request: Request; receiver: Receiver; closed: boolean };

export const HELLO: EngineHello = {
    pid: 1,
    started_at: '2026-11-13T09:00:00Z',
    version: 'v1',
};

export const MACHINES: Asked = {
    input: {},
    kind: 'watch',
    name: 'machines.list',
    target: undefined,
};

export const KEY = 'watch machines.list {} {}';

export const EVERY = 30_000;

export const KEEPS_MACHINES = [
    { out: { epoch: 'e', event: 'opened', fit: { app: 1, engine: 1, verdict: 'fits' }, rev: 1 } },
    {
        out: {
            event: 'snapshot',
            rev: 1,
            slices: { 'machines.list': { state: 'unread', why: 'reading' } },
        },
    },
] as const;

export const RUNNING = { out: { State: 'running' } };
export const PENDING = { out: { State: 'pending' } };
export const STOPPED = { out: { State: 'stopped' } };
export const DONE = { done: true };

function nameOf(request: Request): string | undefined {
    if ('watch' in request) {
        return request.watch;
    }
    return 'call' in request ? request.call : undefined;
}

export function fakeEngine(hello: EngineHello) {
    const opened: Opened[] = [];
    const linked = new Set<(link: Link) => void>();
    const link = { hello, kind: 'open', refused: undefined, socket: 'fake' } satisfies Link;
    const transport: Transport = {
        link: (listener) => {
            linked.add(listener);
            const timer = setTimeout(() => {
                listener(link);
            }, 0);
            return () => {
                clearTimeout(timer);
                linked.delete(listener);
            };
        },
        open: (request, receiver) => {
            const entry = { closed: false, receiver, request };
            opened.push(entry);
            return {
                cancel: () => undefined,
                close: () => {
                    entry.closed = true;
                },
            };
        },
    };
    const of = (name: string) => opened.filter((one) => nameOf(one.request) === name);
    const live = (name: string) => of(name).findLast((each) => !each.closed);
    const answer = (name: string, ...frames: Record<string, unknown>[]) => {
        const one = live(name);
        for (const frame of frames) {
            one?.receiver.frame({ ...frame, id: one.request.id, v: 2 });
        }
    };
    const end = (name: string) => {
        live(name)?.receiver.end('the engine closed the connection', true);
    };
    const show = (shown: Link) => {
        for (const listener of linked) {
            listener(shown);
        }
    };
    const restart = () => {
        show({ ...link, hello: { ...hello, pid: hello.pid + 1 } });
    };
    return {
        answer,
        asked: (name: string) => of(name).length,
        end,
        open: (name: string) => of(name).filter((each) => !each.closed).length,
        requests: (name: string) => of(name).map((each) => each.request),
        restart,
        show,
        transport,
    };
}

export function fakePresence() {
    let visible = true;
    const presence: PresencePort = { visible: () => visible };
    return {
        presence,
        set: (shown: boolean) => {
            visible = shown;
        },
    };
}

export async function unsettled(hello: EngineHello = HELLO, timing: Partial<StoreTiming> = {}) {
    const engine = fakeEngine(hello);
    const seen = fakePresence();
    const client = new Client(engine.transport);
    const store = new ReadStore(client, seen.presence, new StateMirror(client), {
        grace: 500,
        ...timing,
    });
    let told = 0;
    const leave = store.subscribe(MACHINES, () => {
        told += 1;
    });
    await clock.advance(1);
    engine.answer('engine.state', ...KEEPS_MACHINES);
    return { engine, heard: () => told, leave, seen, store };
}

export async function settled(hello: EngineHello = HELLO, timing: Partial<StoreTiming> = {}) {
    const asked = await unsettled(hello, timing);
    asked.engine.answer('machines.list', RUNNING, DONE);
    return asked;
}

// value: a re-ask EVERY later, as a gesture's rereads ask it.
export async function later(...stores: readonly ReadStore[]): Promise<void> {
    await clock.advance(EVERY);
    for (const store of stores) {
        store.reread(['machines.list', 'engine.read']);
    }
}

export function rowsOf(reading: Reading<unknown>): readonly unknown[] {
    return reading.kind === 'read' && Array.isArray(reading.out) ? reading.out : [];
}
