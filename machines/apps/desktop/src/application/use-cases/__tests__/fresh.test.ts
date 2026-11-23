import type { EngineHello } from '@jterrazz/station-api';
import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Asked } from '../entry';
import { offers } from '../fresh';
import { DONE, EVERY, HELLO, KEY, PENDING, RUNNING, settled, STOPPED } from './fake-engine';

const EVENTS: EngineHello = { ...HELLO, calls: ['engine.events', 'machines.list'] };

const DISK: Asked = { input: {}, kind: 'watch', name: 'disk.read', target: undefined };

const DISK_KEY = 'watch disk.read {} {}';

const NOTICE = { out: { event: 'notice', notice: { kind: 'vm_ready', text: 'api is ready' } } };

describe('fresh', () => {
    test('a settled read is never asked again by a timer or a focus', async () => {
        // Given - machines.list settled on running
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, seen } = await settled();

        // When - an hour passes, the window hidden then shown again
        await clock.advance(EVERY * 60);
        seen.set(false);
        seen.set(true);
        await clock.advance(EVERY);

        // Then - its one request
        expect(engine.asked('machines.list')).toBe(1);
    });

    test("a quiet re-ask's held outs are shown by the engine's next revision of the view, never by a timer", async () => {
        // Given - machines.list settled, a view the engine keeps, then reread and answered pending, never ending
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        store.reread(['machines.list']);
        engine.answer('machines.list', PENDING);

        // When - three minutes pass, then the engine's stream brings its next revision of the fleet
        await clock.advance(EVERY * 6);
        const meanwhile = store.reading(KEY);
        engine.answer('engine.state', {
            out: {
                event: 'changed',
                rev: 2,
                slices: { 'machines.list': { state: 'fresh', value: [] } },
            },
        });

        // Then - the settled answer stayed shown, and the revision showed what the re-ask said, as of when it was heard
        expect(meanwhile).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.001Z'),
            done: true,
            kind: 'read',
            out: RUNNING.out,
        });
        expect(store.reading(KEY)).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.001Z'),
            done: false,
            kind: 'read',
            out: PENDING.out,
        });
    });

    test('a watch with no engine revision behind it never draws its pending first out over a settled answer', async () => {
        // Given - disk.read settled beside machines.list, its subscriber's calls counted
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        let told = 0;
        store.subscribe(DISK, () => {
            told += 1;
        });
        await clock.advance(1);
        engine.answer('disk.read', RUNNING, DONE);
        const shown = store.reading(DISK_KEY);
        const before = told;

        // When - a gesture rereads it and its first out is pending, three minutes pass, then it answers stopped and ends
        store.reread(['disk.read']);
        engine.answer('disk.read', PENDING);
        await clock.advance(EVERY * 6);
        const meanwhile = store.reading(DISK_KEY);
        const toldMeanwhile = told - before;
        engine.answer('disk.read', STOPPED, DONE);

        // Then - the settled answer stayed the one shown, its subscriber never told, and the end showed the new answer
        expect(meanwhile).toBe(shown);
        expect(toldMeanwhile).toBe(0);
        expect(store.reading(DISK_KEY)).toStrictEqual({
            at: new Date('2026-11-13T10:03:00.002Z'),
            done: true,
            kind: 'read',
            out: STOPPED.out,
        });
    });

    test('a watch with no engine revision behind it shows its outs as they come over an answer still filling', async () => {
        // Given - disk.read answered pending, its stream still open
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        store.subscribe(DISK, () => {});
        await clock.advance(1);
        engine.answer('disk.read', PENDING);

        // When - a gesture rereads it and its first out is running
        store.reread(['disk.read']);
        engine.answer('disk.read', RUNNING);

        // Then - the running out is shown at once
        expect(store.reading(DISK_KEY)).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.002Z'),
            done: false,
            kind: 'read',
            out: RUNNING.out,
        });
    });

    test('an engine that offers engine.events has its reads asked again on a notice the window is shown for', async () => {
        // Given - an engine whose hello lists engine.events, and machines.list settled
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, seen, store } = await settled(EVENTS);
        engine.answer('engine.events', { out: { stream: 'engine.events' } });
        const opened = engine.asked('machines.list');

        // When - the engine tells a notice while the window is hidden, then one while it is shown
        seen.set(false);
        engine.answer('engine.events', NOTICE);
        const hidden = engine.asked('machines.list');
        seen.set(true);
        engine.answer('engine.events', NOTICE);
        engine.answer('machines.list', STOPPED, DONE);

        // Then - the opening and the hidden notice said nothing, the shown one asked once, and the new answer shows
        expect([opened, hidden, engine.asked('engine.events')]).toStrictEqual([1, 1, 1]);
        expect(engine.asked('machines.list')).toBe(2);
        expect(store.reading(KEY)).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.001Z'),
            done: true,
            kind: 'read',
            out: STOPPED.out,
        });
    });

    test('an engine that does not offer engine.events is never asked it, and one that refuses it is asked once', async () => {
        // Given - one engine without engine.events in its hello, one whose hello lists it and refuses it
        using _ = clock.run('2026-11-13T10:00:00Z');
        const plain = await settled();
        const refusing = await settled(EVENTS);
        refusing.engine.answer('engine.events', {
            error: { code: 'call.unknown', message: 'no call engine.events', type: 'refused' },
        });

        // When - a backoff's worth of time passes
        await clock.advance(5000);

        // Then - the first never asked it, the second asked it once
        expect([
            plain.engine.asked('engine.events'),
            refusing.engine.asked('engine.events'),
        ]).toStrictEqual([0, 1]);
    });

    test('a call is offered only when the hello lists it, and no hello offers nothing', () => {
        // Given - a hello listing the release reading, one without calls, and none
        const listing: EngineHello = {
            calls: ['engine.read-release'],
            pid: 1,
            started_at: '',
            version: 'v0.22.0',
        };
        const bare: EngineHello = { pid: 1, started_at: '', version: 'v0.20.0' };

        // When - each is asked for the release reading
        const offered = [listing, bare, undefined].map((hello) =>
            offers(hello, 'engine.read-release'),
        );

        // Then - the listing hello alone offers it
        expect(offered).toStrictEqual([true, false, false]);
    });
});
