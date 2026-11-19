import { describe, expect, test } from 'vitest';

import type { CoreLine } from '../../../../application/ports/core.port';
import { scriptedCore } from '../scripted-core';
import { scriptOf, SCRIPTS } from '../scripts';

describe('scripted core', () => {
    test('a scripted install plays its lines, then makes the core present', async () => {
        // Given - the missing script
        const core = scriptedCore(SCRIPTS.missing);
        const heard: CoreLine[] = [];
        const before = await core.present();

        // When - its install runs
        const end = await core.install((line) => {
            heard.push(line);
        }).ended;

        // Then - its lines as scripted, exit 0, and the core found afterwards
        expect(before).toBe(false);
        expect(heard).toStrictEqual(SCRIPTS.missing.install.lines);
        expect(end).toStrictEqual({ cancelled: false, code: 0, kind: 'ended' });
        await expect(core.present()).resolves.toBe(true);
    });

    test('a terminal install appears on its third look, and a start that succeeds is told', async () => {
        // Given - the terminal script, and a start watcher
        const started: string[] = [];
        const core = scriptedCore(SCRIPTS.terminal, {
            onStarted: () => {
                started.push('started');
            },
        });

        // When - the disk is looked at three times, then the engine started
        const looks = [await core.present(), await core.present(), await core.present()];
        await core.start(() => undefined).ended;

        // Then - absent twice, then present, and the start told once
        expect(looks).toStrictEqual([false, false, true]);
        expect(started).toStrictEqual(['started']);
    });

    test('a holding install ends only when cancelled, as a signal would end it', async () => {
        // Given - the holding script's install under way
        const run = scriptedCore(SCRIPTS.holding).install(() => undefined);

        // When - it is cancelled
        run.cancel();

        // Then - cancelled, with the code of a SIGTERM
        await expect(run.ended).resolves.toStrictEqual({
            cancelled: true,
            code: 143,
            kind: 'ended',
        });
    });

    test('a script is chosen by name, and an unknown name chooses none', () => {
        // Given - a known name, an unknown one, and no name
        const chosen = [scriptOf('failing'), scriptOf('nope'), scriptOf(null)];

        // Then - the known script, and nothing for the rest
        expect(chosen).toStrictEqual([SCRIPTS.failing, undefined, undefined]);
    });
});
