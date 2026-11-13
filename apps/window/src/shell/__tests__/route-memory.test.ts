import { describe, expect, test } from 'vitest';

import { machineRoute } from '../route';
import { recalled, remember } from '../route-memory';
import type { Memory } from '../route-memory';

function memoryOf(text?: string): Memory & { held: Map<string, string> } {
    const held = new Map<string, string>();
    if (text !== undefined) {
        held.set('station.window.route', text);
    }
    return {
        getItem: (key) => held.get(key) ?? null,
        held,
        setItem: (key, value) => {
            held.set(key, value);
        },
    };
}

describe('route memory', () => {
    test('a route remembered is the route recalled', () => {
        // Given - the window left on linux-1's Logs
        const memory = memoryOf();
        remember(memory, machineRoute('linux-1', 'Logs'));

        // When - the next launch reads it back
        const route = recalled(memory, false);

        // Then - it opens where it was left
        expect(route).toStrictEqual(machineRoute('linux-1', 'Logs'));
    });

    test('a catalog item and a machine file are recalled with the page that holds them', () => {
        // Given - the window left on node's page, then on the file of api
        const item = memoryOf();
        remember(item, { item: 'node', page: 'catalog-apps' });
        const file = memoryOf();
        remember(file, { file: 'api', page: 'machine-files' });

        // Then - each reopens on what it held
        expect(recalled(item, false)).toStrictEqual({ item: 'node', page: 'catalog-apps' });
        expect(recalled(file, false)).toStrictEqual({ file: 'api', page: 'machine-files' });
    });

    test.each([
        { text: 'not json' },
        { text: '{"page":"nope"}' },
        { text: '{"page":"machine","machine":"","tab":"Logs"}' },
        { text: '{"page":"machine","machine":"linux-1","tab":"Nope"}' },
        { text: 'null' },
    ])('a stored value that names no route, $text, is no route', ({ text }) => {
        // Given - a value no window of this version wrote
        const memory = memoryOf(text);

        // Then - nothing is recalled, and the launch opens Home
        expect(recalled(memory, true)).toBeUndefined();
    });

    test('the gallery is recalled only by a window that holds it', () => {
        // Given - a proof window left on the gallery
        const memory = memoryOf('{"page":"gallery"}');

        // Then - a proof window reopens it, a production window does not
        expect(recalled(memory, true)).toStrictEqual({ page: 'gallery' });
        expect(recalled(memory, false)).toBeUndefined();
    });

    test('a webview without storage remembers nothing and recalls nothing', () => {
        // Given - no storage at all
        remember(undefined, { page: 'inbox' });

        // Then - the launch has no route to recall
        expect(recalled(undefined, false)).toBeUndefined();
    });
});
