import { describe, expect, test } from 'vitest';

import type { WindowProgress } from '../../../../domain/window-update';
import { scriptedWindowUpdate, windowScriptOf } from '../scripted-window-update';

function scripted(name: string) {
    const script = windowScriptOf(name);
    if (script === undefined) {
        throw new Error(`no window script ${name}`);
    }
    return scriptedWindowUpdate(script);
}

describe('scripted window update', () => {
    test('a failing install fails once with its progress, then installs on the next press', async () => {
        // Given - the script whose first install is refused
        const port = scripted('install-failing');
        const heard: WindowProgress[] = [];

        // When - it is checked, then installed twice
        const checked = await port.check();
        const first = await port.install((step) => {
            heard.push(step);
        });
        const second = await port.install(() => undefined);

        // Then - out, a refused install after its two chunks, then installed
        expect(checked).toMatchObject({ current: '0.21.0', kind: 'out', version: '0.22.0' });
        expect(heard.map((step) => step.kind)).toStrictEqual(['downloaded', 'downloaded']);
        expect([first.kind, second.kind]).toStrictEqual(['refused', 'installed']);
    });

    test('a holding install hears its first chunk and never ends, and an unknown name is no script', async () => {
        // Given - the holding script
        const port = scripted('install-holding');
        let installing: Promise<unknown> = Promise.resolve('never asked');

        // When - its install is asked, its chunk heard, and its end raced against a settled promise
        const heard = await new Promise<WindowProgress>((resolve) => {
            installing = port.install(resolve);
        });
        const ended = await Promise.race([installing, Promise.resolve('held')]);

        // Then - one chunk, no end, and no script by another name
        expect(heard).toStrictEqual({
            downloaded: 7_400_000,
            kind: 'downloaded',
            total: 18_000_000,
        });
        expect(ended).toBe('held');
        expect(windowScriptOf('later')).toBeUndefined();
    });
});
