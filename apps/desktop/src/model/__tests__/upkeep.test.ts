import type { OverviewNeed } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { canWait, needsYou } from '../upkeep';

function need(Subject: string, Upkeep: OverviewNeed['Upkeep']): OverviewNeed {
    return {
        Args: null,
        Broken: false,
        Call: '',
        Command: '',
        Confirm: '',
        Machine: 'mbp',
        Noun: 'Apps',
        On: '',
        Path: '',
        Section: 'apps',
        Severity: 'needs-a-look',
        Subject,
        Text: Subject,
        Upkeep,
        Verb: 'fix',
    };
}

describe('upkeep', () => {
    test('the needs the engine says need you and those that can wait are told apart, in the engine order', () => {
        // Given - four needs, the engine's order mixing both upkeeps
        const needs = [
            need('one', 'can-wait'),
            need('two', 'needs-you'),
            need('three', 'can-wait'),
            need('four', 'needs-you'),
        ];

        // When - each upkeep is asked for
        const owed = needsYou(needs).map(({ Subject }) => Subject);
        const waiting = canWait(needs).map(({ Subject }) => Subject);

        // Then - each holds its own, in the order given
        expect(owed).toStrictEqual(['two', 'four']);
        expect(waiting).toStrictEqual(['one', 'three']);
    });
});
