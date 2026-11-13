import type { Request } from '@jterrazz/station-api';
import { clock } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import { Client } from '../client';
import { EngineError } from '../engine-error';
import type { Link, Receiver, Transport } from '../transport';

type Opened = { request: Request; receiver: Receiver; closed: boolean };

function fakeTransport() {
    const opened: Opened[] = [];
    const listeners: ((link: Link) => void)[] = [];
    const transport: Transport = {
        link: (listener) => {
            listeners.push(listener);
            return () => {
                listeners.splice(listeners.indexOf(listener), 1);
            };
        },
        open: (request, receiver) => {
            const entry = { closed: false, receiver, request };
            opened.push(entry);
            return {
                close: () => {
                    entry.closed = true;
                },
            };
        },
    };
    return { listeners, opened, transport };
}

function last(opened: readonly Opened[]): Opened {
    const entry = opened.at(-1);
    if (!entry) {
        throw new Error('nothing was opened');
    }
    return entry;
}

async function outcome(answer: Promise<unknown>): Promise<unknown> {
    try {
        return await answer;
    } catch (error) {
        return error;
    }
}

describe('Client', () => {
    test('a call is one v2 frame and resolves on the out of its id', async () => {
        // Given - a call of engine.read with progress followed
        const { opened, transport } = fakeTransport();
        const progress: string[] = [];
        const answer = new Client(transport).call(
            'engine.read',
            {},
            {
                progress: (step) => {
                    progress.push(step.task);
                },
                target: { name: 'jterrazz-mac-mini' },
            },
        );
        const { request, receiver } = last(opened);

        // When - another id's out, a progress, then its own out arrive
        receiver.frame({ id: 'other', out: { State: 'stopped' }, v: 2 });
        receiver.frame({ id: request.id, progress: { task: 'Reading the engine' }, v: 2 });
        receiver.frame({ id: request.id, out: { State: 'running', Version: 'v3.1.0' }, v: 2 });

        // Then - the frame sent is the call's, the out its own, the exchange closed
        expect(request).toStrictEqual({
            call: 'engine.read',
            id: request.id,
            in: {},
            target: { name: 'jterrazz-mac-mini' },
            v: 2,
        });
        await expect(answer).resolves.toStrictEqual({ State: 'running', Version: 'v3.1.0' });
        expect(progress).toStrictEqual(['Reading the engine']);
        expect(last(opened).closed).toBe(true);
    });

    test("an error frame rejects with the engine's own words", async () => {
        // Given - a call the engine refuses
        const { opened, transport } = fakeTransport();
        const answer = new Client(transport).call('engine.read', {});
        const { request, receiver } = last(opened);

        // When - the error frame arrives
        receiver.frame({
            error: {
                hint: 'station engine start',
                message: 'the engine is not running',
                type: 'engine-stopped',
            },
            id: request.id,
            v: 2,
        });

        // Then - the rejection carries the wire error as the engine wrote it
        const error: unknown = await answer.catch((error_: unknown) => error_);
        expect(error).toBeInstanceOf(EngineError);
        expect(error instanceof EngineError ? error.wire : undefined).toStrictEqual({
            hint: 'station engine start',
            message: 'the engine is not running',
            type: 'engine-stopped',
        });
    });

    test('a connection that never reached an engine rejects as unsent', async () => {
        // Given - a call on a socket nobody listens on
        const { opened, transport } = fakeTransport();
        const answer = new Client(transport).call('engine.read', {});

        // When - the transport ends it before the request was sent
        last(opened).receiver.end('no socket at engine.sock', false);

        // Then - the rejection is the transport's, not the engine's, and nothing was sent
        const error: unknown = await answer.catch((error_: unknown) => error_);
        expect(error).toStrictEqual(
            new EngineError(undefined, 'no socket at engine.sock', 'unsent'),
        );
        expect(error instanceof EngineError ? error.wire : 'not a failure').toBeUndefined();
    });

    test('a connection that drops after the request was sent rejects as lost', async () => {
        // Given - a call the engine received
        const { opened, transport } = fakeTransport();
        const answer = new Client(transport).call('engine.stop', {});

        // When - its connection drops before the answer
        last(opened).receiver.end('the engine closed the connection', true);

        // Then - its outcome is lost: the engine may have done it
        const error: unknown = await answer.catch((error_: unknown) => error_);
        expect(error instanceof EngineError ? error.fate : 'not a failure').toBe('lost');
    });

    test('a call silent past its time is closed and lost, and progress keeps it alive', async () => {
        // Given - a call with the default time, and one given 2 s
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        const client = new Client(transport);
        const slow = outcome(client.call('engine.read', {}));
        const first = last(opened);
        const quick = outcome(client.call('engine.read', {}, { within: 2000 }));
        const second = last(opened);

        // When - the first hears a progress at 20 s, and 45 s pass in all
        await clock.advance(1999);
        const quickOpen = !second.closed;
        await clock.advance(18_001);
        first.receiver.frame({ id: first.request.id, progress: { task: 'Reading' }, v: 2 });
        await clock.advance(29_999);
        const slowOpen = !first.closed;
        await clock.advance(1);

        // Then - each ended at its own time after its last frame, as lost
        expect([quickOpen, slowOpen, first.closed, second.closed]).toStrictEqual([
            true,
            true,
            true,
            true,
        ]);
        await expect(quick).resolves.toStrictEqual(
            new EngineError(undefined, 'the engine did not answer in 2 s', 'lost'),
        );
        await expect(slow).resolves.toStrictEqual(
            new EngineError(undefined, 'the engine did not answer in 30 s', 'lost'),
        );
    });

    test('a watch streams its outs, ends at done, and asks nothing more', async () => {
        // Given - a watch of overview.read on one machine
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        const outs: unknown[] = [];
        const done = vi.fn<() => void>();
        new Client(transport).watch(
            'overview.read',
            {},
            {
                done,
                error: vi.fn<(error: EngineError) => void>(),
                out: (out) => {
                    outs.push(out);
                },
            },
            { name: 'linux-1' },
        );
        const { request, receiver } = last(opened);

        // When - two outs then done arrive, and a minute passes
        receiver.frame({ id: request.id, out: { Summary: { Pending: true } }, v: 2 });
        receiver.frame({ id: request.id, out: { Summary: { Pending: false } }, v: 2 });
        receiver.frame({ done: true, id: request.id, v: 2 });
        await clock.advance(60_000);

        // Then - the frame was a watch, both outs came in order, and nothing was opened again
        expect(request).toStrictEqual({
            id: request.id,
            in: {},
            target: { name: 'linux-1' },
            v: 2,
            watch: 'overview.read',
        });
        expect(outs).toStrictEqual([
            { Summary: { Pending: true } },
            { Summary: { Pending: false } },
        ]);
        expect(done).toHaveBeenCalledOnce();
        expect(opened).toHaveLength(1);
        expect(last(opened).closed).toBe(true);
    });

    test('a watch whose connection drops is opened again after 1, 2, then 4 s', async () => {
        // Given - a watch of machines.list
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        const errors: unknown[] = [];
        new Client(transport).watch(
            'machines.list',
            {},
            {
                error: (error) => {
                    errors.push(error);
                },
                out: vi.fn<(out: unknown) => void>(),
            },
        );

        // When - its connection ends three times, each wait spent
        last(opened).receiver.end('the engine closed the connection', true);
        await clock.advance(999);
        const beforeFirst = opened.length;
        await clock.advance(1);
        last(opened).receiver.end('the engine closed the connection', true);
        await clock.advance(1999);
        const beforeSecond = opened.length;
        await clock.advance(1);
        last(opened).receiver.end('the engine closed the connection', true);
        await clock.advance(4000);

        // Then - each reopening waited its backoff, with a fresh id, and each drop was told
        expect([beforeFirst, beforeSecond, opened.length]).toStrictEqual([1, 2, 4]);
        expect(new Set(opened.map((entry) => entry.request.id)).size).toBe(4);
        expect(errors).toHaveLength(3);
        expect(
            errors.every((error) => error instanceof EngineError && error.wire === undefined),
        ).toBe(true);
    });

    test('a page watch silent for 45 s is dropped and made again', async () => {
        // Given - a page watch that opened and then says nothing
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        new Client(transport).watch(
            'page',
            { pages: ['menubar'] },
            { error: vi.fn<(error: EngineError) => void>(), out: vi.fn<(out: unknown) => void>() },
        );
        last(opened).receiver.frame({
            id: last(opened).request.id,
            out: { stream: 'pages' },
            v: 2,
        });

        // When - 45 s pass without a frame, then the backoff of 1 s
        await clock.advance(45_000);
        const dropped = last(opened).closed;
        await clock.advance(1000);

        // Then - the silent connection was closed and a new one asked
        expect(dropped).toBe(true);
        expect(opened).toHaveLength(2);
    });

    test('a watch the engine answers with an error ends there, and is never opened again', async () => {
        // Given - a watch of engine.events on an engine that does not know it
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        const errors: EngineError[] = [];
        new Client(transport).watch(
            'engine.events',
            {},
            {
                error: (error) => {
                    errors.push(error);
                },
                out: vi.fn<(out: unknown) => void>(),
            },
        );
        const { request, receiver } = last(opened);

        // When - the engine refuses it, and a minute passes
        receiver.frame({
            error: { code: 'call.unknown', message: 'no call engine.events', type: 'refused' },
            id: request.id,
            v: 2,
        });
        await clock.advance(60_000);

        // Then - the refusal was told once, the exchange closed, and nothing asked again
        expect(errors.map((error) => error.wire?.code)).toStrictEqual(['call.unknown']);
        expect(last(opened).closed).toBe(true);
        expect(opened).toHaveLength(1);
    });

    test('a closed watch asks nothing more, even after a drop', async () => {
        // Given - a watch closed by its page
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        const watch = new Client(transport).watch(
            'machines.list',
            {},
            { error: vi.fn<(error: EngineError) => void>(), out: vi.fn<(out: unknown) => void>() },
        );
        watch.close();

        // When - its connection ends late and time passes
        last(opened).receiver.end('the engine closed the connection', true);
        await clock.advance(60_000);

        // Then - nothing is opened again
        expect(opened).toHaveLength(1);
    });
});
