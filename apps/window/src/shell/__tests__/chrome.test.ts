import { describe, expect, test } from 'vitest';

import { placeOf, wearDress } from '../chrome';
import type { Place, Root, View } from '../chrome';

const MAC = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15';
const LINUX = 'Mozilla/5.0 (X11; Linux aarch64) AppleWebKit/605.1.15';

function rootOf(): Root & { held: Map<string, string> } {
    const held = new Map<string, string>();
    return {
        held,
        removeAttribute: (name) => {
            held.delete(name);
        },
        setAttribute: (name, value) => {
            held.set(name, value);
        },
    };
}

function viewOf(width: number, height: number): View & { resize: (w: number, h: number) => void } {
    const listeners: (() => void)[] = [];
    const view = {
        addEventListener: (_type: 'resize', listener: () => void) => {
            listeners.push(listener);
        },
        innerHeight: 900,
        innerWidth: 1400,
        resize: (w: number, h: number) => {
            view.innerWidth = w;
            view.innerHeight = h;
            for (const listener of listeners) {
                listener();
            }
        },
        screen: { height, width },
    };
    return view;
}

function worn(place: Place, view = viewOf(1728, 1117)): Map<string, string> {
    const root = rootOf();
    wearDress(root, place, view);
    return root.held;
}

describe('chrome', () => {
    test('the Tauri window on a Mac wears the overlay and the material, and nowhere else does', () => {
        // Given - the four places the page can open in
        const places = {
            linuxTauri: placeOf(true, LINUX, false),
            mac: placeOf(true, MAC, false),
            page: placeOf(false, MAC, false),
            proof: placeOf(false, LINUX, true),
        };

        // Then - only the Mac's window is translucent, and a proof page asks the overlay alone
        expect(
            Object.fromEntries(Object.entries(places).map(([k, p]) => [k, worn(p)])),
        ).toStrictEqual({
            linuxTauri: new Map(),
            mac: new Map([
                ['data-chrome', 'overlay'],
                ['data-material', 'vibrant'],
            ]),
            page: new Map(),
            proof: new Map([['data-chrome', 'overlay']]),
        });
    });

    test('a Mac window gone fullscreen drops the room it kept for the lights, and takes it back', () => {
        // Given - the Mac's window, then made to fill the screen
        const root = rootOf();
        const view = viewOf(1728, 1117);
        wearDress(root, 'mac', view);

        // When - it fills the screen
        view.resize(1728, 1117);

        // Then - the lights are hidden, so no row is kept for them; the material stays
        expect(root.held).toStrictEqual(new Map([['data-material', 'vibrant']]));

        // When - it leaves fullscreen
        view.resize(1400, 900);

        // Then - the row comes back
        expect(root.held.get('data-chrome')).toBe('overlay');
    });
});
