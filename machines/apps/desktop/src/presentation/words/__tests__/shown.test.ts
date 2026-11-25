import type { OverviewNeed } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import type { Out } from '../../../application/ports/engine.port';
import { keyOf } from '../../../domain/request-key';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import {
    followerState,
    holdAgainst,
    macCardState,
    macState,
    needState,
    rollUpState,
    vmState,
} from '../shown';

const recordingFor = await recordingsOf('standings', 'two-macs');

function needMarked(Mark: OverviewNeed['Mark']): OverviewNeed {
    return {
        Args: null,
        Broken: true,
        Call: '',
        Command: '',
        Confirm: '',
        Detail: '',
        Machine: 'macbook-pro',
        Mark,
        Noun: 'Overview',
        On: '',
        Path: '',
        Section: 'overview',
        Severity: 'broken',
        Subject: '',
        Text: '',
        Upkeep: 'needs-you',
        Verb: 'fix',
    };
}

function macsOf(world: string) {
    const asked = keyOf({ in: {}, watch: 'machines.list' });
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<'machines.list'> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    return outs.at(-1)?.out.All.Macs ?? [];
}

describe('shown', () => {
    test("a Mac's and a VM's standing are shown by the engine's word and tone for it", () => {
        // Given - This Mac of two-macs, online, with api stopped and linux-1 running
        const [mac] = macsOf('two-macs');

        // When - their standings are read
        const shown = mac ? [macState(mac), ...(mac.VMs ?? []).map((vm) => vmState(vm))] : [];

        // Then - each is words.json's, lowercase, never the raw state
        expect(shown).toStrictEqual([
            { tone: 'ok', word: 'online' },
            { tone: 'plain', word: 'stopped' },
            { tone: 'ok', word: 'running' },
        ]);
    });

    test("a roll-up is the engine's word, after its count in the engine's noun when it sends one", () => {
        // Given - each roll-up state, down and not seen counted once and twice
        // When - each is shown
        const shown = [
            rollUpState({ N: 1, State: 'down' }),
            rollUpState({ N: 2, State: 'down' }),
            rollUpState({ N: 1, State: 'not-seen' }),
            rollUpState({ N: 2, State: 'not-seen' }),
            rollUpState({ State: 'empty' }),
            rollUpState({ State: 'fine' }),
        ];

        // Then - the noun is words.json's One or Other, and an uncounted state says its word alone
        expect(shown).toStrictEqual([
            { tone: 'bad', word: '1 machine down' },
            { tone: 'bad', word: '2 machines down' },
            { tone: 'warn', word: '1 Mac not seen' },
            { tone: 'warn', word: '2 Macs not seen' },
            { tone: 'plain', word: 'no machine yet' },
            { tone: 'ok', word: 'all running' },
        ]);
    });

    test("a need's pill is its Mark's word and tone, whatever its severity says", () => {
        // Given - one need per Mark, each with the severity the engine orders it by
        // When - each is shown
        const shown = (['exposed', 'down', 'needs-you', 'can-wait'] as const).map((mark) =>
            needState(needMarked(mark)),
        );

        // Then - words.json's OverviewMark, never the severity's
        expect(shown).toStrictEqual([
            { tone: 'bad', word: 'Exposed' },
            { tone: 'bad', word: 'Down' },
            { tone: 'warn', word: 'Needs you' },
            { tone: 'plain', word: 'Upkeep' },
        ]);
    });

    test("an online Mac's card says its machines' roll-up", () => {
        // Given - standings: macbook-pro online with loop-1 down, mac-mini online with no machine
        const macs = macsOf('standings');

        // When - each card's state is read
        const shown = macs.map((mac) => macCardState(mac));

        // Then - one machine down, and no machine yet
        expect(shown).toStrictEqual([
            { tone: 'bad', word: '1 machine down' },
            { tone: 'plain', word: 'no machine yet' },
        ]);
    });

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
            { Sentence: 'Always on. Stopping it asks yes or no.', State: 'always-on' },
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
