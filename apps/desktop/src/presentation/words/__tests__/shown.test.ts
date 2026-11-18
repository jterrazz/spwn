import { describe, expect, test } from 'vitest';

import { followerState, holdAgainst } from '../shown';

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

    test('only a hold whose state the engine tones as a warning is said against a delete, as sent', () => {
        // Given - each hold state the engine sends, with a sentence the recordings carry
        const holds = [
            {
                By: 'studio-os-fe',
                Sentence: 'It is held by its lot, by studio-os-fe: work may be in progress.',
                State: 'held',
            },
            { Sentence: 'Stopped. Safe to delete.', State: 'free' },
            { Sentence: 'Yours. Stays until you delete it.', State: 'yours' },
            { Sentence: 'Always on. Stopping it needs its name typed.', State: 'always-on' },
            {
                Sentence: 'Not seen: its Mac does not answer, so nothing says what holds it.',
                State: 'not-seen',
            },
        ] as const;

        // When - each is read for a delete, and a machine with no hold
        const said = [...holds.map((hold) => holdAgainst(hold)), holdAgainst(undefined)];

        // Then - the held one alone is said, its sentence untouched
        expect(said).toStrictEqual([
            'It is held by its lot, by studio-os-fe: work may be in progress.',
            undefined,
            undefined,
            undefined,
            undefined,
            undefined,
        ]);
    });
});
