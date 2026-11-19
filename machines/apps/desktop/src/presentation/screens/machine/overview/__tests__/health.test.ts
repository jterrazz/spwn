import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { ageOf } from '../health';

describe('health', () => {
    test('a check is aged in seconds, minutes or hours within a day, in days past it', () => {
        // Given - the window read on 13 Nov at 10:00
        using _ = clock.at('2026-11-13T10:00:00Z');

        // Then - each check's age from then, and none for a check never run
        expect(ageOf('2026-11-13T09:59:57Z')).toBe('3 s ago');
        expect(ageOf('2026-11-13T07:00:00Z')).toBe('3 h ago');
        expect(ageOf('2026-11-07T02:00:00Z')).toBe('6 d 8 h ago');
        expect(ageOf(undefined)).toBeUndefined();
    });
});
