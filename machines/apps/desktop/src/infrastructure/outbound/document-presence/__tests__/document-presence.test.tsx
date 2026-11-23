import { describe, expect, test } from 'vitest';

import { documentPresence } from '../document-presence';

describe('document presence', () => {
    test('a document shown counts as shown, whatever its focus', () => {
        // Given - the document's presence

        // When - the window loses the focus
        globalThis.dispatchEvent(new Event('blur'));

        // Then - the page still counts as shown
        expect(documentPresence().visible()).toBe(true);
    });
});
