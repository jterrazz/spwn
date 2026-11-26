import { readFileSync } from 'node:fs';
import { join } from 'node:path';
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
    test('the Tauri window on a Mac wears the overlay, a proof page asks for it, and nothing else does', () => {
        // Given - the four places the page can open in
        const places = {
            linuxTauri: placeOf(true, LINUX, false),
            mac: placeOf(true, MAC, false),
            page: placeOf(false, MAC, false),
            proof: placeOf(false, LINUX, true),
        };

        // Then - the Mac's window and a proof page keep the lights' row, and no place sets anything more
        expect(
            Object.fromEntries(Object.entries(places).map(([k, p]) => [k, worn(p)])),
        ).toStrictEqual({
            linuxTauri: new Map(),
            mac: new Map([['data-chrome', 'overlay']]),
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

        // Then - the lights are hidden, so no row is kept for them
        expect(root.held).toStrictEqual(new Map());

        // When - it leaves fullscreen
        view.resize(1400, 900);

        // Then - the row comes back
        expect(root.held.get('data-chrome')).toBe('overlay');
    });

    test('the native window is opaque, and its ground is the s0 step of the page', () => {
        // Given - the main window as tauri.conf.json declares it, and the s0 step tokens.css declares
        const desktop = join(import.meta.dirname, '..', '..', '..', '..');
        const config: unknown = JSON.parse(
            readFileSync(join(desktop, 'src-tauri', 'tauri.conf.json'), 'utf8'),
        );
        const tokens = readFileSync(
            join(desktop, 'src', 'presentation', 'ui', 'tokens.css'),
            'utf8',
        );
        const s0 = /--s0:\s*(?<value>#[0-9a-f]{6});/u.exec(tokens)?.groups?.value;

        // Then - no transparency and no effect under it, and what shows behind the web view is s0
        expect(config).not.toHaveProperty(['app', 'windows', 0, 'transparent']);
        expect(config).not.toHaveProperty(['app', 'windows', 0, 'windowEffects']);
        expect(config).toHaveProperty(['app', 'windows', 0, 'backgroundColor'], s0);
    });
});
