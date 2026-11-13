import { describe, expect, test } from 'vitest';

import { documentPresence } from '../presence';

describe('presence', () => {
    test('a focus is the window back only after it lost the focus, and a listener left hears nothing', () => {
        // Given - the document's presence, listened to
        const seen: string[] = [];
        const stop = documentPresence().subscribe((one) => {
            seen.push(one);
        });

        // When - a focus it never lost, a blur then two focuses, then the listener leaves and the focus goes and comes
        globalThis.dispatchEvent(new Event('focus'));
        globalThis.dispatchEvent(new Event('blur'));
        globalThis.dispatchEvent(new Event('focus'));
        globalThis.dispatchEvent(new Event('focus'));
        stop();
        globalThis.dispatchEvent(new Event('blur'));
        globalThis.dispatchEvent(new Event('focus'));

        // Then - one back, for the focus regained, and the page counts as shown
        expect(seen).toStrictEqual(['back']);
        expect(documentPresence().visible()).toBe(true);
    });
});
