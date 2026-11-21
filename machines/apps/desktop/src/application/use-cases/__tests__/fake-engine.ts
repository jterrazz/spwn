import type { EngineHello, Request } from '@jterrazz/station-api';
import { clock } from '@jterrazz/test';

import { PAGE_REVISION } from '../../../domain/fit';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';
import type { Link } from '../../ports/engine.port';
import type { PresencePort } from '../../ports/presence.port';
import type { Asked, Reading } from '../entry';
import { ReadStore } from '../store';
import type { StoreTiming } from '../store';

type Opened = { request: Request; receiver: Receiver; closed: boolean };

export const HELLO: EngineHello = {
    page: PAGE_REVISION,
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
    const listeners = new Set<(seen: 'back' | 'hidden') => void>();
    const presence: PresencePort = {
        subscribe: (listener) => {
            listeners.add(listener);
            return () => {
                listeners.delete(listener);
            };
        },
        visible: () => visible,
    };
    const set = (seen: 'back' | 'hidden') => {
        visible = seen === 'back';
        for (const listener of listeners) {
            listener(seen);
        }
    };
    return { listening: () => listeners.size, presence, set };
}

export async function unsettled(hello: EngineHello = HELLO, timing: Partial<StoreTiming> = {}) {
    const engine = fakeEngine(hello);
    const seen = fakePresence();
    const store = new ReadStore(new Client(engine.transport), seen.presence, {
        every: EVERY,
        grace: 500,
        ...timing,
    });
    let told = 0;
    const leave = store.subscribe(MACHINES, () => {
        told += 1;
    });
    await clock.advance(1);
    return { engine, heard: () => told, leave, seen, store };
}

export async function settled(hello: EngineHello = HELLO, timing: Partial<StoreTiming> = {}) {
    const asked = await unsettled(hello, timing);
    asked.engine.answer('machines.list', RUNNING, DONE);
    return asked;
}

export function rowsOf(reading: Reading<unknown>): readonly unknown[] {
    return reading.kind === 'read' && Array.isArray(reading.out) ? reading.out : [];
}
