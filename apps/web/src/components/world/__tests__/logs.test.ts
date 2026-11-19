import { describe, expect, test } from 'vitest';

import { closedLog, logsPanelOf } from '../logs';
import type { SettledLog } from '../logs';

const BOOT = { message: 'world started', source: 'world', timestamp: '2026-11-19T10:00:00Z' };

function settledOf(worldId: string, thisOpening: boolean, lines = [BOOT]): SettledLog {
    return { lines, thisOpening, worldId };
}

describe('logs', () => {
    test('a panel just opened is loading, with no line, before any load has settled', () => {
        // Given - the logs panel opened on matrix, nothing read yet

        // Then - it loads, and states no emptiness
        expect(logsPanelOf(null, 'matrix', true)).toStrictEqual({ lines: [], loading: true });
    });

    test('a load settled during this opening is shown, loaded', () => {
        // Given - matrix read since the panel opened

        // Then - its lines, no longer loading
        expect(logsPanelOf(settledOf('matrix', true), 'matrix', true)).toStrictEqual({
            lines: [BOOT],
            loading: false,
        });
    });

    test('an empty log settled during this opening reads as settled, so the panel may say it is empty', () => {
        // Given - matrix read since the panel opened, with no event

        // Then - no line, not loading
        expect(logsPanelOf(settledOf('matrix', true, []), 'matrix', true)).toStrictEqual({
            lines: [],
            loading: false,
        });
    });

    test('a reopened panel keeps the lines it read before, and loads until this opening settles', () => {
        // Given - matrix read during an earlier opening, then the panel closed and opened again
        const reopened = closedLog(settledOf('matrix', true));

        // Then - the earlier lines stay on screen, and the panel loads
        expect(logsPanelOf(reopened, 'matrix', true)).toStrictEqual({
            lines: [BOOT],
            loading: true,
        });
    });

    test('a reopened panel over an empty log loads rather than say it is empty', () => {
        // Given - an empty matrix log read during an earlier opening, the panel reopened
        const reopened = closedLog(settledOf('matrix', true, []));

        // Then - loading, so no emptiness is claimed before this opening reads it
        expect(logsPanelOf(reopened, 'matrix', true)).toStrictEqual({ lines: [], loading: true });
    });

    test('a panel switched to another world shows none of the last world lines and loads', () => {
        // Given - matrix read in this opening, the panel now on zion

        // Then - no matrix line under zion, loading
        expect(logsPanelOf(settledOf('matrix', true), 'zion', true)).toStrictEqual({
            lines: [],
            loading: true,
        });
    });

    test('a closed panel is not loading', () => {
        // Given - the panel closed before anything was read

        // Then - nothing loads
        expect(logsPanelOf(null, 'matrix', false)).toStrictEqual({ lines: [], loading: false });
    });

    test('closing with nothing read leaves nothing read', () => {
        // Given - no load settled

        // Then - still none
        expect(closedLog(null)).toBeNull();
    });
});
