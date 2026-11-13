import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { EngineError } from '../engine-error';
import { GestureStore } from '../gesture-store';

const ERASE = 'call disk.erase {"App":"orbstack"} {"name":"jterrazz-macbook-pro"}';

const REFUSED = {
    error: new EngineError({ message: 'orbstack has no data station can erase', type: 'refused' }),
    kind: 'failed',
} as const;

describe('gesture store', () => {
    test("a row that leaves and comes back within the grace finds its gesture's refusal", async () => {
        // Given - an erase refused while its row was shown
        using _ = clock.run('2026-11-13T10:00:00Z');
        const store = new GestureStore({ grace: 500 });
        const leave = store.subscriberOf(ERASE)(() => {});
        store.set(ERASE, REFUSED);

        // When - the row leaves the list, then comes back 200 ms later
        leave();
        await clock.advance(200);
        store.subscriberOf(ERASE)(() => {});
        await clock.advance(1000);

        // Then - the refusal is still the row's
        expect(store.state(ERASE)).toBe(REFUSED);
    });

    test('a settled gesture nobody shows is forgotten after its grace, a running one is kept', async () => {
        // Given - a refused erase and a running one, each left by its row
        using _ = clock.run('2026-11-13T10:00:00Z');
        const store = new GestureStore({ grace: 500 });
        const running = 'call disk.erase {"App":"zed"} {"name":"jterrazz-macbook-pro"}';
        store.set(ERASE, REFUSED);
        store.set(running, { answer: 'zed', kind: 'running', progress: [] });
        store.subscriberOf(running)(() => {})();

        // When - the grace passes
        await clock.advance(600);

        // Then - the refusal is gone, the running gesture stays until it settles
        expect(store.state(ERASE).kind).toBe('idle');
        expect(store.state(running).kind).toBe('running');
    });

    test('a render that asks for the subscriber creates no entry; a subscription does, and keeps its function', () => {
        // Given - a store nobody has subscribed to
        const store = new GestureStore({ grace: 500 });

        // When - two renders ask for the subscriber, then one subscribes and two more renders ask
        const rendered = [store.subscriberOf(ERASE), store.subscriberOf(ERASE)];
        rendered[0]?.(() => {});
        const held = [store.subscriberOf(ERASE), store.subscriberOf(ERASE)];

        // Then - the renders before held nothing to share, those after share the entry's function
        expect(rendered[0]).not.toBe(rendered[1]);
        expect(held[0]).toBe(held[1]);
    });
});
