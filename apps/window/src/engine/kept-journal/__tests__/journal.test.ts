import { describe, expect, test } from 'vitest';

import { recordingsOf } from '../../../fixtures/recordings';
import type { Out } from '../../client';
import { keyOf } from '../../frame';
import { answerFor } from '../journal';

const recordingFor = await recordingsOf('mini-offline', 'one-mac', 'stories', 'two-macs');

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

function titlesOf(world: string): readonly string[] | undefined {
    return answerFor(listingOf(world), {})?.Groups?.map((group) => group.Title);
}

describe('journal', () => {
    test('each world is answered by its own journal, chosen by the machines it lists', () => {
        // Given - four worlds

        // When - each is asked with no Mac
        const titles = ['stories', 'two-macs', 'mini-offline', 'one-mac'].map(titlesOf);

        // Then - the showcase's three days, two nights and their yesterday, and one-mac's empty journal
        expect(titles).toStrictEqual([
            ['Today', 'Last night', 'Yesterday'],
            ['Last night', 'Yesterday'],
            ['Last night', 'Yesterday'],
            [],
        ]);
    });

    test("a Mac asked keeps its own and its machines' lines, each narrowed to them, and no sum", () => {
        // Given - stories
        const listing = listingOf('stories');

        // When - it is asked for mac-mini, then for a Mac it does not list
        const mini = answerFor(listing, { Mac: 'mac-mini' });
        const nowhere = answerFor(listing, { Mac: 'nowhere' });

        // Then - the deletion on the mini, node and uv on its machines alone, and nothing for the unknown Mac
        const night = mini?.Groups?.find((group) => group.Night);
        expect(night?.Sum).toBe('');
        expect(night?.Lines?.map((line) => [line.Text, line.On])).toStrictEqual([
            ['node', ['mac-mini', 'infra']],
            ['uv', ['infra', 'review-bot']],
        ]);
        expect(mini?.Groups?.[0]?.Lines?.map((line) => line.Text)).toStrictEqual([
            'gate-2 deleted',
            'infra restarted',
        ]);
        expect(nowhere?.Groups).toStrictEqual([]);
    });

    test('a fleet no journal is written for is answered by nothing', () => {
        // Given - two-macs with its VMs renamed, so no answer is written for them
        const known = listingOf('two-macs');
        const macs = (known.All.Macs ?? []).map((mac) => ({
            ...mac,
            VMs: (mac.VMs ?? []).map((vm) => ({ ...vm, Name: `${vm.Name}-elsewhere` })),
        }));

        // When - it is asked
        const answer = answerFor({ ...known, All: { ...known.All, Macs: macs } }, {});

        // Then - nothing, which a page draws as not given
        expect(answer).toBeUndefined();
    });
});
