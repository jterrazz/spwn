import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { StateMirror, viewOf } from '../mirror';
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

    test("only This Mac's fleet and Overview are views; any other input or target is a call", () => {
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
                target: { name: 'linux-1', on: 'macbook-pro' },
            }),
            viewOf({
                input: { ThisMac: true },
                kind: 'watch',
                name: 'machines.list',
                target: undefined,
            }),
        ];

        // Then - the first three are views, the last two their calls
        expect(named).toStrictEqual([
            'machines.list',
            'overview.read',
            'overview.read+elsewhere',
            undefined,
            undefined,
        ]);
    });
});
