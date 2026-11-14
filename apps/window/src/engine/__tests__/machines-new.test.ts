import { describe, expect, test } from 'vitest';

import { recordingsOf } from '../../fixtures/recordings';
import type { Out } from '../client';
import { keyOf } from '../frame';
import { answerFor } from '../machines-new';

const recordingFor = await recordingsOf('mini-offline', 'two-macs');

function listingOf(world: string): Out<'machines.list'> {
    const asked = keyOf({ in: {}, watch: 'machines.list' });
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<'machines.list'> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    const listing = outs.at(-1)?.out;
    if (!listing) {
        throw new Error(`${world} records no machines.list.`);
    }
    return listing;
}

describe('machines new', () => {
    test('the answer is the one written for the Macs listed, a silent Mac marked', () => {
        // Given - two-macs, and mini-offline where the mini does not answer

        // When - each is asked with no input
        const [both, silent] = [
            answerFor(listingOf('two-macs'), {}),
            answerFor(listingOf('mini-offline'), {}),
        ];

        // Then - the first is on the mini, which has the most room; the second on This Mac
        expect(both?.Mac).toBe('jterrazz-mac-mini');
        expect(silent?.Mac).toBe('jterrazz-macbook-pro');
        expect(silent?.On?.map((on) => on.Ready)).toStrictEqual([true, false]);
    });

    test('the input is applied as the engine applies it: a Mac offered is chosen, a file followed', () => {
        // Given - two-macs
        const listing = listingOf('two-macs');

        // When - it is asked on This Mac from linux-dev, then on a Mac it does not offer
        const asked = answerFor(listing, { From: 'linux-dev', On: 'jterrazz-macbook-pro' });
        const unknown = answerFor(listing, { On: 'nowhere' });

        // Then - the Mac and the file asked, and an unknown Mac left to the default
        expect([asked?.Mac, asked?.From]).toStrictEqual(['jterrazz-macbook-pro', 'linux-dev']);
        expect(unknown?.Mac).toBe('jterrazz-mac-mini');
    });

    test('a fleet no answer is written for is answered by nothing', () => {
        // Given - two-macs with its Macs renamed, so no answer names them
        const known = listingOf('two-macs');
        const macs = (known.All.Macs ?? []).map((mac) => ({
            ...mac,
            Name: `${mac.Name}-elsewhere`,
        }));
        const listing = { ...known, All: { ...known.All, Macs: macs } };

        // When - it is asked
        const answer = answerFor(listing, {});

        // Then - nothing, which the sheet draws as unanswered
        expect(answer).toBeUndefined();
    });
});
