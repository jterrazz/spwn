import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { StateMirror } from '../mirror';
import { viewOf } from '../sliced';
import { fakeEngine, HELLO } from './fake-engine';

const EPOCH = '2026-11-21T09:00:00Z/4123';
const FLEET = { All: { Macs: [{ Name: 'macbook-pro' }] } };
const GROWN = { All: { Macs: [{ Name: 'macbook-pro' }, { Name: 'mac-mini' }] } };

const OPENED = {
    out: { epoch: EPOCH, event: 'opened', fit: { app: 1, engine: 1, verdict: 'fits' }, rev: 7 },
};

function changed(rev: number, slice: Record<string, unknown>) {
    return { out: { event: 'changed', rev, slices: { 'machines.list': slice } } };
}

async function mirrored() {
    const engine = fakeEngine(HELLO);
    const mirror = new StateMirror(new Client(engine.transport), {
        grace: 500,
        lost: 45_000,
        retry: 1000,
    });
    let renders = 0;
    mirror.subscriberOf('machines.list')(() => {
        renders += 1;
    });
    await clock.advance(1);
    engine.answer('engine.state', OPENED, {
        out: {
            event: 'snapshot',
            rev: 7,
            slices: { 'machines.list': { hash: 'a', state: 'fresh', value: FLEET } },
        },
    });
    return { engine, mirror, renders: () => renders };
}

describe('mirror', () => {
    test('a view is drawn once per revision that moves it, never for one that brings it back equal', async () => {
        // Given - a window holding the fleet, drawn from its first snapshot
        using _ = clock.run('2026-11-21T09:00:00Z');
        const { engine, mirror, renders } = await mirrored();
        const first = mirror.reading('machines.list');

        // When - the engine pushes the fleet equal, then grown, then only its state, then grown again equal
        engine.answer(
            'engine.state',
            changed(8, { hash: 'a', state: 'fresh', value: structuredClone(FLEET) }),
        );
        engine.answer('engine.state', changed(9, { hash: 'b', state: 'fresh', value: GROWN }));
        const grown = mirror.reading('machines.list');
        engine.answer('engine.state', changed(10, { hash: 'b', state: 'stale', why: 'late' }));
        engine.answer(
            'engine.state',
            changed(11, { hash: 'b', state: 'stale', value: structuredClone(GROWN), why: 'late' }),
        );

        // Then - three renders: the snapshot, the grown fleet, its stale mark; the grown parts kept their identity
        expect(first.kind === 'read' && first.out).toStrictEqual(FLEET);
        expect(renders()).toBe(3);
        const shown = mirror.reading('machines.list');
        expect(shown.kind === 'read' && shown.stale?.message).toBe('late');
        expect(shown.kind === 'read' && grown.kind === 'read' && shown.out === grown.out).toBe(
            true,
        );
    });

    test('a stream silent past three ticks is asked again from the revision held, its view kept', async () => {
        // Given - a window holding the fleet at revision 7
        using _ = clock.run('2026-11-21T09:00:00Z');
        const { engine, mirror, renders } = await mirrored();

        // When - no frame comes for 45 s
        await clock.advance(45_000);

        // Then - one new stream, asked since the revision held, and nothing drawn in between
        const asked = engine.requests('engine.state');
        expect(asked).toHaveLength(2);
        expect(asked[1]).toMatchObject({ in: { hold: true, since: { epoch: EPOCH, rev: 7 } } });
        expect(mirror.reading('machines.list').kind).toBe('read');
        expect(renders()).toBe(1);
    });

    test('a cut connection keeps the view and asks again a second later', async () => {
        // Given - a window holding the fleet
        using _ = clock.run('2026-11-21T09:00:00Z');
        const { engine, mirror } = await mirrored();

        // When - the connection ends
        engine.end('engine.state');
        await clock.advance(1000);

        // Then - the fleet is still drawn, and one stream more is open
        expect(mirror.reading('machines.list').kind).toBe('read');
        expect(engine.open('engine.state')).toBe(1);
        expect(engine.requests('engine.state')).toHaveLength(2);
    });

    test("a refused stream draws the engine's words and is asked again past three ticks", async () => {
        // Given - a window asking the fleet of an engine that refuses one of the views it names
        using _ = clock.run('2026-11-21T09:00:00Z');
        const engine = fakeEngine(HELLO);
        const mirror = new StateMirror(new Client(engine.transport), {
            grace: 500,
            lost: 45_000,
            retry: 1000,
        });
        mirror.subscriberOf('machines.list')(() => undefined);
        await clock.advance(1);
        const refusal = {
            code: 'engine.state-unknown-key',
            field: 'keys',
            message: 'no slice or view is named engine.guides',
            subject: 'engine.guides',
            type: 'refused',
        };

        // When - the engine refuses the stream, then answers it once asked again
        engine.answer('engine.state', { error: refusal });
        const refused = mirror.reading('machines.list');
        await clock.advance(44_999);
        const meanwhile = engine.requests('engine.state').length;
        await clock.advance(1);
        engine.answer('engine.state', OPENED, {
            out: {
                event: 'snapshot',
                rev: 7,
                slices: { 'machines.list': { hash: 'a', state: 'fresh', value: FLEET } },
            },
        });

        // Then - the refusal is the view's failure in its words, the stream asked once more after 45 s, and drawn
        expect(refused.kind === 'failed' && refused.error.message).toBe(refusal.message);
        expect(meanwhile).toBe(1);
        expect(engine.requests('engine.state')).toHaveLength(2);
        const shown = mirror.reading('machines.list');
        expect(shown.kind === 'read' && shown.out).toStrictEqual(FLEET);
    });

    test("the fleet and a machine's Overview are views; any other input, or a target on a Mac, is a call", () => {
        // Given - the reads the window makes of the two calls

        // When - each is named as a view
        const named = [
            viewOf({ input: {}, kind: 'watch', name: 'machines.list', target: undefined }),
            viewOf({ input: {}, kind: 'watch', name: 'overview.read', target: undefined }),
            viewOf({
                input: { Elsewhere: true },
                kind: 'watch',
                name: 'overview.read',
                target: undefined,
            }),
            viewOf({
                input: {},
                kind: 'watch',
                name: 'overview.read',
                target: { name: 'linux-1' },
            }),
            viewOf({
                input: {},
                kind: 'watch',
                name: 'overview.read',
                target: { name: 'linux-1', on: 'macbook-pro' },
            }),
            viewOf({
                input: { Elsewhere: true },
                kind: 'watch',
                name: 'overview.read',
                target: { name: 'mac-mini' },
            }),
            viewOf({
                input: { ThisMac: true },
                kind: 'watch',
                name: 'machines.list',
                target: undefined,
            }),
        ];

        // Then - the first four are views, another machine's Overview named by it, the last three their calls
        expect(named).toStrictEqual([
            'machines.list',
            'overview.read',
            'overview.read+elsewhere',
            'overview.read:linux-1',
            undefined,
            undefined,
            undefined,
        ]);
    });

    test("every Mac's ports, a machine's sign-ins and a Mac's Awake are views; a VM aimed at by target is a call", () => {
        // Given - the reads the window makes of ports, sign-ins and Awake

        // When - each is named as a view
        const named = [
            viewOf({
                input: { AllMachines: true },
                kind: 'read',
                name: 'ports.list',
                target: undefined,
            }),
            viewOf({
                input: { Of: 'linux-1' },
                kind: 'read',
                name: 'ports.list',
                target: undefined,
            }),
            viewOf({ input: {}, kind: 'watch', name: 'accounts.list', target: undefined }),
            viewOf({
                input: { Machine: 'linux-1' },
                kind: 'watch',
                name: 'accounts.list',
                target: undefined,
            }),
            viewOf({
                input: {},
                kind: 'watch',
                name: 'accounts.list',
                target: { name: 'mac-mini' },
            }),
            viewOf({ input: {}, kind: 'watch', name: 'awake.read', target: undefined }),
            viewOf({ input: {}, kind: 'watch', name: 'awake.read', target: { name: 'mac-mini' } }),
            viewOf({
                input: {},
                kind: 'watch',
                name: 'awake.read',
                target: { name: 'linux-1', on: 'macbook-pro' },
            }),
            viewOf({ input: {}, kind: 'read', name: 'ports.list', target: undefined }),
        ];

        // Then - all but the last two are views, named by machine when they are not This Mac's own
        expect(named).toStrictEqual([
            'ports.list+all-machines',
            'ports.list:linux-1',
            'accounts.list',
            'accounts.list:linux-1',
            'accounts.list:mac-mini',
            'awake.read',
            'awake.read:mac-mini',
            undefined,
            undefined,
        ]);
    });

    test('a still engine draws nothing new on the ports, sign-ins and Awake it pushes', async () => {
        // Given - a window holding every Mac's ports, a VM's sign-ins and a Mac's Awake from one snapshot
        using _ = clock.run('2026-11-21T09:00:00Z');
        const engine = fakeEngine(HELLO);
        const mirror = new StateMirror(new Client(engine.transport), {
            grace: 500,
            lost: 45_000,
            retry: 1000,
        });
        const views = [
            'ports.list+all-machines',
            'accounts.list:linux-1',
            'awake.read:mac-mini',
        ] as const;
        const drawn = { renders: 0 };
        for (const view of views) {
            mirror.subscriberOf(view)(() => {
                drawn.renders += 1;
            });
        }
        await clock.advance(1);
        const values = {
            'accounts.list:linux-1': { Programs: [] },
            'awake.read:mac-mini': { Mac: 'mac-mini' },
            'ports.list+all-machines': { all: { macs: [] } },
        };
        const slices = (value: boolean) =>
            Object.fromEntries(
                views.map((view) => [
                    view,
                    {
                        hash: view,
                        state: 'fresh',
                        ...(value ? { value: structuredClone(values[view]) } : {}),
                    },
                ]),
            );
        engine.answer('engine.state', OPENED, {
            out: { event: 'snapshot', rev: 7, slices: slices(true) },
        });
        const first = drawn.renders;

        // When - the engine pushes revisions that move none of them: sent without their values, then equal
        engine.answer('engine.state', { out: { event: 'changed', rev: 8, slices: slices(false) } });
        engine.answer('engine.state', { out: { event: 'changed', rev: 9, slices: slices(true) } });
        engine.answer('engine.state', { out: { event: 'tick', rev: 9 } });

        // Then - each view was drawn once, by the snapshot, and nothing after it
        expect(first).toBe(3);
        expect(drawn.renders).toBe(3);
    });

    test('a machine whose view the engine does not keep is refused to its call, until the engine keeps it', async () => {
        // Given - a window holding a server's Overview, which the engine's first snapshot does not hold
        using _ = clock.run('2026-11-21T09:00:00Z');
        const engine = fakeEngine(HELLO);
        const mirror = new StateMirror(new Client(engine.transport), {
            grace: 500,
            lost: 45_000,
            retry: 1000,
        });
        mirror.subscriberOf('overview.read:api')(() => undefined);
        await clock.advance(1);
        engine.answer('engine.state', OPENED, {
            out: {
                event: 'snapshot',
                rev: 7,
                slices: { 'machines.list': { hash: 'a', state: 'fresh', value: FLEET } },
            },
        });
        const absent = mirror.refusedOf('overview.read:api');
        mirror.subscriberOf('ports.list:api')(() => undefined);
        const later = mirror.refusedOf('ports.list:api');

        // When - the engine comes to keep the server's Overview
        engine.answer('engine.state', {
            out: {
                event: 'changed',
                rev: 8,
                slices: {
                    'overview.read:api': { hash: 'o', state: 'fresh', value: { Summary: {} } },
                },
            },
        });

        // Then - both were the calls' while the engine held neither; the Overview is now the view's
        expect(absent).toBe('not-kept');
        expect(later).toBe('not-kept');
        expect(mirror.refusedOf('overview.read:api')).toBeUndefined();
        expect(mirror.reading('overview.read:api').kind).toBe('read');
    });

    test("the engine's opening is its verdict: a misfit closes the stream, and another engine is judged again", async () => {
        // Given - a window holding only its link, over an engine whose stream opens older than the window
        using _ = clock.run('2026-11-21T09:00:00Z');
        const engine = fakeEngine(HELLO);
        const mirror = new StateMirror(new Client(engine.transport));
        let told = 0;
        mirror.subscribeFit(() => {
            told += 1;
        });
        await clock.advance(1);
        engine.answer('engine.state', {
            out: { ...OPENED.out, fit: { app: 1, engine: 0, verdict: 'engine-older' } },
        });
        const older = [mirror.fitOf(HELLO), engine.open('engine.state')];

        // When - the engine restarts as another, whose opening names no verdict
        engine.restart();
        const restarted = { ...HELLO, pid: HELLO.pid + 1 };
        const meanwhile = mirror.fitOf(restarted);
        engine.answer('engine.state', { out: { epoch: EPOCH, event: 'opened', rev: 1 } });

        // Then - older and closed, the new engine unjudged until it opens, then older too, each verdict told once
        expect(older).toStrictEqual(['engine-older', 0]);
        expect([meanwhile, mirror.fitOf(restarted), mirror.fitOf(HELLO)]).toStrictEqual([
            undefined,
            'engine-older',
            undefined,
        ]);
        expect([engine.asked('engine.state'), told]).toStrictEqual([2, 2]);
    });
});
