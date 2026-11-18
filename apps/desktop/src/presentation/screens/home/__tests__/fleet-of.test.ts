import type { ActivityReport, MachinesListOutput } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { keyOf } from '../../../../domain/request-key';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import type { Identity } from '../../../ui/index';
import { fleetOf } from '../fleet-of';

const recordingFor = await recordingsOf('stories', 'standings');

const HOUR_GRAPH = { graph: 3_600_000_000_000 };
const GB = 1024 ** 3;

function answersTo(request: Record<string, unknown>, world = 'stories'): readonly unknown[] {
    const asked = keyOf(request);
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    return exchange?.answers ?? [];
}

function answered(answer: unknown): boolean {
    return typeof answer === 'object' && answer !== null && 'out' in answer;
}

function listed(world = 'stories'): MachinesListOutput {
    const outs = answersTo({ in: {}, watch: 'machines.list' }, world).filter(
        (answer): answer is { out: MachinesListOutput } => answered(answer),
    );
    const last = outs.at(-1);
    if (last === undefined) {
        throw new Error(`${world} records machines.list.`);
    }
    return last.out;
}

function standingsFleet() {
    const listing = listed('standings');
    return fleetOf({
        identityOf: () => undefined,
        macs: listing.All.Macs ?? [],
        reports: new Map(),
        rooms: listing.Rooms ?? [],
        standing: listing.All.Standing,
    });
}

function reportOf(target: Record<string, unknown> | undefined): ActivityReport {
    const outs = answersTo({ in: HOUR_GRAPH, target, watch: 'activity.read' }).filter(
        (answer): answer is { out: ActivityReport } => answered(answer),
    );
    const last = outs.at(-1);
    if (last === undefined) {
        throw new Error("stories records each awake Mac's activity.read over an hour.");
    }
    return last.out;
}

const IDENTITIES: ReadonlyMap<string, Identity> = new Map<string, Identity>([
    ['agent-7f', 'blue'],
    ['infra', 'pink'],
]);

function storiesFleet() {
    const listing = listed();
    return fleetOf({
        identityOf: (name) => IDENTITIES.get(name),
        macs: listing.All.Macs ?? [],
        reports: new Map([
            ['macbook-pro', reportOf(undefined)],
            ['mac-mini', reportOf({ name: 'mac-mini' })],
        ]),
        rooms: listing.Rooms ?? [],
        standing: listing.All.Standing,
    });
}

describe('fleet-of', () => {
    test("a Mac's machines come in machines.list's order, each placed on its Mac's track in its room's order after macOS", () => {
        // Given - stories: macbook-pro (48 GB, macOS 4) lists agent-7f, api, linux-1, linux-2, and its room shares agent-7f, linux-1, linux-2 (api is stopped)
        // When - the fleet is read
        const pro = storiesFleet().macs[0];

        // Then - macOS opens the track, each share sits after the one before it, and api has no share
        expect(pro?.system).toStrictEqual({ at: 0, used: 1, width: 4 / 48 });
        expect(pro?.machines.map((machine) => [machine.name, machine.share])).toStrictEqual([
            ['agent-7f', { at: 4 / 48, used: 5_726_623_061 / (8 * GB), width: 8 / 48 }],
            ['api', undefined],
            ['linux-1', { at: 12 / 48, used: 5_726_623_061 / (8 * GB), width: 8 / 48 }],
            ['linux-2', { at: 20 / 48, used: undefined, width: 8 / 48 }],
        ]);
    });

    test("each figure is the share's own used memory and the activity read's CPU, in GB and %", () => {
        // Given - stories: agent-7f uses 5.3 of its 8 GB, linux-2 is created with no use measured, the mini's infra uses 16 of 24; activity.read answers for macbook-pro and mac-mini, and refuses macbook-air, asleep and with no machine
        // When - the fleet is read
        const { macs } = storiesFleet();

        // Then - a Mac reads its CPU and its free, a machine its CPU and its used of its share, and what no read gives is empty; a Mac with no machine is counted by no line
        expect(
            macs.map((mac) => [mac.name, mac.kind, mac.cpu, mac.free, mac.thisMac]),
        ).toStrictEqual([
            ['macbook-pro', '4 machines', '22 %', '20 free', true],
            ['mac-mini', '3 machines', '36 %', '24 free', false],
            ['macbook-air', undefined, '', '20 free', false],
        ]);
        expect(
            macs.flatMap((mac) =>
                mac.machines.map((machine) => [machine.name, machine.cpu, machine.memory]),
            ),
        ).toStrictEqual([
            ['agent-7f', '1 %', '5.3 / 8'],
            ['api', '', ''],
            ['linux-1', '9 %', '5.3 / 8'],
            ['linux-2', '', 'of 8'],
            ['infra', '10 %', '16 / 24'],
            ['linux-3', '17 %', '5.3 / 8'],
            ['review-bot', '0.9 %', '2.7 / 4'],
        ]);
    });

    test("every word is the engine's: a machine's standing and head, a Mac's card, and the roll-up of all machines", () => {
        // Given - stories: agent-7f running with two differences, api stopped, linux-2 being built, infra down by its failing check; macbook-pro and mac-mini online, macbook-air asleep
        // When - the fleet is read
        const fleet = storiesFleet();
        const machines = fleet.macs.flatMap((mac) => mac.machines);

        // Then - each is copied as sent: an online Mac says its machines' roll-up, the asleep one its own standing, and the stage's line rolls every machine up
        expect(fleet.standing).toStrictEqual({ tone: 'bad', word: '1 machine down' });
        expect(fleet.thisMac).toBe('This Mac');
        expect(fleet.macs.map((mac) => mac.state)).toStrictEqual([
            { tone: 'ok', word: 'all running' },
            { tone: 'bad', word: '1 machine down' },
            { tone: 'plain', word: 'asleep' },
        ]);
        expect(machines[0]).toStrictEqual({
            cpu: '1 %',
            identity: 'blue',
            line: 'Write the changelog of v4. Up 2 h.',
            memory: '5.3 / 8',
            name: 'agent-7f',
            purpose: 'Write the changelog of v4',
            sentence: 'Running, with 2 differences from its file.',
            share: { at: 4 / 48, used: 5_726_623_061 / (8 * GB), width: 8 / 48 },
            state: { tone: 'ok', word: 'running' },
        });
        expect(
            machines.map((machine) => [
                machine.name,
                machine.state,
                machine.sentence,
                machine.line,
            ]),
        ).toStrictEqual([
            [
                'agent-7f',
                { tone: 'ok', word: 'running' },
                'Running, with 2 differences from its file.',
                'Write the changelog of v4. Up 2 h.',
            ],
            [
                'api',
                { tone: 'plain', word: 'stopped' },
                'api is stopped.',
                'It holds no memory. Start it to use it again.',
            ],
            [
                'linux-1',
                { tone: 'ok', word: 'running' },
                'Running, with 8 differences from its file.',
                'Lot L2.5, settings from a machine file. Up 2 h.',
            ],
            ['linux-2', { tone: 'new', word: 'creating' }, 'Creating.', undefined],
            [
                'infra',
                { tone: 'bad', word: 'down' },
                'infra is down.',
                'Its check fails since 10:18.',
            ],
            [
                'linux-3',
                { tone: 'ok', word: 'running' },
                'Running.',
                'Refresh the auth token. Up 2 h.',
            ],
            [
                'review-bot',
                { tone: 'ok', word: 'running' },
                'Running.',
                'Review every pull request of studio-station. Up 2 h.',
            ],
        ]);
    });

    test("a machine down, restarting or suspended is drawn in its own standing's word and tone", () => {
        // Given - standings: macbook-pro's loop-1 is down (always on, broken in a restart loop), restart-1 is restarting, suspended-1 is suspended
        // When - the fleet is read
        const [pro] = standingsFleet().macs;

        // Then - each line carries the engine's word and tone of its standing
        expect(pro?.machines.map((machine) => [machine.name, machine.state])).toStrictEqual([
            ['loop-1', { tone: 'bad', word: 'down' }],
            ['restart-1', { tone: 'new', word: 'restarting' }],
            ['suspended-1', { tone: 'plain', word: 'suspended' }],
        ]);
    });

    test("an online Mac's card says its machines' roll-up, counted in the engine's noun, and an empty one says so alone", () => {
        // Given - standings: macbook-pro online with one machine down among three, mac-mini online with no machine; all machines roll up to one down
        // When - the fleet is read
        const fleet = standingsFleet();

        // Then - the pro says one machine down over its count, the mini says no machine yet with no count, and the stage says one machine down
        expect(fleet.macs.map((mac) => [mac.name, mac.kind, mac.state])).toStrictEqual([
            ['macbook-pro', '3 machines', { tone: 'bad', word: '1 machine down' }],
            ['mac-mini', undefined, { tone: 'plain', word: 'no machine yet' }],
        ]);
        expect(fleet.standing).toStrictEqual({ tone: 'bad', word: '1 machine down' });
    });
});
