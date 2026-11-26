import { describe, expect, test } from 'vitest';

import { markVisibility } from '../visibility';
import type { Page, Root } from '../visibility';

function pageOf(
    state: DocumentVisibilityState,
): Page & { show: (next: DocumentVisibilityState) => void } {
    const listeners: (() => void)[] = [];
    const page = {
        addEventListener: (_type: 'visibilitychange', listener: () => void) => {
            listeners.push(listener);
        },
        show: (next: DocumentVisibilityState) => {
            page.visibilityState = next;
            for (const listener of listeners) {
                listener();
            }
        },
        visibilityState: state,
    };
    return page;
}

describe('visibility', () => {
    test('the root says hidden while the page is hidden and visible again once it shows', () => {
        // Given - a root and a page that opens hidden
        const root: Root = { dataset: {} };
        const page = pageOf('hidden');

        // When - the page is marked, then shown, then hidden again
        markVisibility(root, page);
        const opened = root.dataset.visibility;
        page.show('visible');
        const shown = root.dataset.visibility;
        page.show('hidden');

        // Then - the root followed it each time
        expect([opened, shown, root.dataset.visibility]).toStrictEqual([
            'hidden',
            'visible',
            'hidden',
        ]);
    });
});
