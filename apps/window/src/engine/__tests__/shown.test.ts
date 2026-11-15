import { describe, expect, test } from 'vitest';

import { followerState } from '../shown';

describe('shown', () => {
    test("a follower is shown by the engine's word and tone, a difference counted in its noun", () => {
        // Given - the three states the engine sends
        // When - each is shown
        const shown = [
            followerState({ Differences: 0, State: 'matches' }),
            followerState({ Differences: 1, State: 'differs' }),
            followerState({ Differences: 3, State: 'differs' }),
            followerState({ Differences: 0, State: 'unread' }),
        ];

        // Then - the words and tones are words.json's, "1 difference" and "3 differences" joined by a count
        expect(shown).toStrictEqual([
            { tone: 'ok', word: 'Matches' },
            { tone: 'warn', word: '1 difference' },
            { tone: 'warn', word: '3 differences' },
            { tone: 'plain', word: 'Not read' },
        ]);
    });
});
