import { describe, expect, test } from 'vitest';

import type { Tone } from '../../../domain/tone';
import {
    APP_STATE_TONES,
    APP_WHY_TONES,
    CORE_TONES,
    DISK_RESULT_TONES,
    OWN_STATE_TONES,
    PORT_ANSWER_TONES,
    SHARE_STATE_TONES,
    UPDATE_TONES,
} from '../tones';

const TABLES: Readonly<Record<string, Readonly<Record<string, Tone>>>> = {
    APP_STATE_TONES,
    APP_WHY_TONES,
    CORE_TONES,
    DISK_RESULT_TONES,
    OWN_STATE_TONES,
    PORT_ANSWER_TONES,
    SHARE_STATE_TONES,
    UPDATE_TONES,
};

function keysOf(wanted: Tone): string[] {
    return Object.entries(TABLES)
        .flatMap(([name, table]) =>
            Object.entries(table)
                .filter(([, tone]) => tone === wanted)
                .map(([key]) => `${name}.${key}`),
        )
        .toSorted();
}

describe('tones', () => {
    test('orange is drawn only for a state that asks the owner to act or to look', () => {
        // Given - every table key that draws the orange tone
        const orange = keysOf('warn');

        // Then - the closed list of what waits on the owner: an unreachable port, a failing, late or missing app, a degraded part
        expect(orange).toStrictEqual([
            'APP_STATE_TONES.degraded',
            'APP_WHY_TONES.failing',
            'APP_WHY_TONES.interrupted',
            'APP_WHY_TONES.late',
            'APP_WHY_TONES.missing',
            'APP_WHY_TONES.outdated',
            'APP_WHY_TONES.partial',
            'APP_WHY_TONES.unmanaged',
            'DISK_RESULT_TONES.degraded',
            'PORT_ANSWER_TONES.unreachable',
        ]);
    });

    test('blue is drawn only for work in progress, a port that is starting included', () => {
        // Given - every table key that draws the blue tone
        const blue = keysOf('new');

        // Then - the closed list of what is running or being waited for
        expect(blue).toStrictEqual([
            'APP_STATE_TONES.pending',
            'CORE_TONES.installing',
            'CORE_TONES.started',
            'CORE_TONES.starting',
            'CORE_TONES.waiting',
            'DISK_RESULT_TONES.pending',
            'OWN_STATE_TONES.connecting',
            'OWN_STATE_TONES.reading',
            'OWN_STATE_TONES.window-out',
            'PORT_ANSWER_TONES.starting',
            'SHARE_STATE_TONES.waiting',
            'UPDATE_TONES.restarting',
            'UPDATE_TONES.updating',
        ]);
    });

    test('a state the window cannot know is drawn plain, never orange', () => {
        // Given - the window's own unknown state
        const { unknown } = OWN_STATE_TONES;

        // Then - it asks nothing of the owner
        expect(unknown).toBe('plain');
    });
});
