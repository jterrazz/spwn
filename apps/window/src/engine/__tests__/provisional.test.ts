import type { MachinesEngineView, OverviewNeed } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { recordingsOf } from '../../fixtures/recordings';
import type { Out } from '../client';
import { EngineError } from '../engine-error';
import { keyOf } from '../frame';
import {
    confirmFormOf,
    createRefusalPlaceOf,
    diskGestureOf,
    engineViewStateOf,
    freesWhole,
    holdsRebuildable,
    inboxIsUrgent,
    isPortShared,
    isSkill,
    machineKindOf,
    portsMacOf,
    shareEndOf,
} from '../provisional';
import { macState, vmState } from '../shown';

const recordingFor = await recordingsOf('bare-mac', 'disk-low', 'stories', 'two-macs');

function macsOf(world: string) {
    const asked = keyOf({ in: {}, watch: 'machines.list' });
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<'machines.list'> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    return outs.at(-1)?.out.All.Macs ?? [];
}

function listingOf(world: string) {
    const asked = keyOf({ call: 'ports.list', in: { AllMachines: true } });
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<'ports.list'> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    const listing = outs.at(-1)?.out;
    if (!listing) {
        throw new Error(`${world} records no ports.list.`);
    }
    return listing;
}

function portsOf(world: string) {
    return listingOf(world).all.macs ?? [];
}

function diskOf(world: string) {
    const asked = keyOf({ in: {}, watch: 'disk.read' });
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<'disk.read'> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    const reading = outs.at(-1)?.out;
    if (!reading) {
        throw new Error(`${world} records no disk.read of This Mac.`);
    }
    return reading;
}

describe('provisional', () => {
    test("a Mac's reach and a VM's state are shown by the one word their code has", () => {
        // Given - This Mac of two-macs, reachable, with a stopped and a running VM
        const [mac] = macsOf('two-macs');

        // When - their states are read
        const shown = mac ? [macState(mac), ...(mac.VMs ?? []).map((vm) => vmState(vm))] : [];

        // Then - each reads the words and tones tables, never its raw field
        expect(shown).toStrictEqual([
            { tone: 'ok', word: 'online' },
            { tone: 'plain', word: 'Stopped' },
            { tone: 'ok', word: 'Running' },
        ]);
    });

    test("a VM's port is shared when one of the VM's shares names its guest port", () => {
        // Given - the first Mac of stories, whose linux-1 shares 3000 and 5173 of the four ports it listens on
        const [mac] = portsOf('stories');
        const shares = (mac?.shared ?? []).filter((share) => share.vm === 'linux-1');

        // When - each of linux-1's ports is asked whether it is shared
        const shared = (mac?.ports ?? [])
            .filter((port) => port.Machine.Name === 'linux-1')
            .map((port) => [port.Port, isPortShared(port, shares)]);

        // Then - the two its shares name are, the two others are not
        expect(shared).toStrictEqual([
            [3000, true],
            [4321, false],
            [5173, true],
            [9229, false],
        ]);
    });

    test('a disk holds what can be rebuilt when one of its rows is in that group', () => {
        // Given - the disk of This Mac in disk-low, whose rows hold caches, and in bare-mac, which lists none
        const low = diskOf('disk-low');
        const bare = diskOf('bare-mac');

        // When - each is asked whether it holds what can be rebuilt
        const held = [holdsRebuildable(low), holdsRebuildable(bare)];

        // Then - the first does, the second does not
        expect(held).toStrictEqual([true, false]);
    });

    test('each Disk group names the call that would act on it, and never touched none', () => {
        // Given - the four groups

        // When - their gestures are read
        const gestures = (['misplaced', 'can-rebuild', 'your-data', 'never-touched'] as const).map(
            (group) => diskGestureOf(group),
        );

        // Then - move, free, erase that destroys, and nothing; only the rebuildable frees as a whole
        expect(gestures).toStrictEqual([
            { call: 'disk.move', danger: false },
            { call: 'disk.free', danger: false },
            { call: 'disk.erase', danger: true },
            undefined,
        ]);
        expect(
            (['misplaced', 'can-rebuild'] as const).map((group) => freesWhole(group)),
        ).toStrictEqual([false, true]);
    });

    test("a machine's delete, a server's restart and an erase are confirmed by their subject typed, a stop by a yes", () => {
        // Given - four calls calls.json confirms
        const calls = [
            'machines.delete',
            'machines.restart',
            'disk.erase',
            'machines.stop',
        ] as const;

        // Then - the name three times, then a yes
        expect(calls.map((call) => confirmFormOf(call))).toStrictEqual([
            'name',
            'name',
            'name',
            'yes',
        ]);
    });

    test('a server is the VM whose lifetime is always on, every other a workspace', () => {
        // Given - the VMs of This Mac in stories
        const [mac] = macsOf('stories');
        const vms = mac?.VMs ?? [];

        // When - each is asked its kind
        const kinds = vms.map((vm) => [vm.Lifetime, machineKindOf(vm)]);

        // Then - always on reads server, the others workspace
        expect(kinds.length).toBeGreaterThan(0);
        expect(
            kinds.every(
                ([lifetime, kind]) => kind === (lifetime === 'always-on' ? 'server' : 'workspace'),
            ),
        ).toBe(true);
    });

    test("a Mac's engine is read as engine.read's state from its booleans, and as none while not known", () => {
        // Given - a Mac's engine known running, known installed and stopped, known not installed, and not known
        const known: MachinesEngineView = {
            Installed: true,
            Known: true,
            Running: true,
            Version: '',
        };
        const engines = [
            known,
            { ...known, Running: false },
            { ...known, Installed: false, Running: false },
            { ...known, Known: false },
        ];

        // When - each is read as a state
        const states = engines.map((engine) => engineViewStateOf(engine));

        // Then - running, stopped, not installed, then no code
        expect(states).toStrictEqual(['running', 'stopped', 'not-installed', undefined]);
    });

    test('the Inbox is urgent when one of its needs is broken', () => {
        // Given - two needs, one broken
        const need: OverviewNeed = {
            Args: null,
            Broken: false,
            Call: '',
            Command: '',
            Machine: 'mbp',
            Path: '',
            Section: 'disk',
            Severity: 'needs-a-look',
            Subject: 'disk',
            Text: '',
        };

        // Then - a list holding the broken one is urgent, one without it is not
        expect(inboxIsUrgent([need, { ...need, Broken: true }])).toBe(true);
        expect(inboxIsUrgent([need])).toBe(false);
    });

    test("a share holds until its end, against the report's hour", () => {
        // Given - a report read at 10:00
        const now = '2026-11-12T10:00:00Z';

        // When - three ends are read: later, past, unreadable
        const ends = ['2026-11-12T11:00:00Z', '2026-11-12T09:00:00Z', 'soon'].map((until) =>
            shareEndOf(until, now),
        );

        // Then - an hour left, ended, and not read
        expect(ends).toStrictEqual([
            { kind: 'ends-in', nanoseconds: 3_600_000_000_000 },
            { kind: 'ended' },
            { kind: 'unread' },
        ]);
    });

    test('a catalog entry is a skill when the catalog files it under Agent skills, and no other is', () => {
        // Given - the category of a skill, of an app, and none
        const categories = ['Agent skills', 'Git', ''] as const;

        // When - each is asked whether it makes a skill
        const skills = categories.map((category) => isSkill(category));

        // Then - the skills' category alone
        expect(skills).toStrictEqual([true, false, false]);
    });

    test("this Mac's ports are found by its name, or by the flag when ports.list names it otherwise", () => {
        // Given - two-macs, whose ports.list names This Mac as machines.list does, and stories, which names it ''
        const [twoMacs, stories] = [listingOf('two-macs'), listingOf('stories')];

        // When - This Mac is looked for in each, by the name machines.list gives it
        const found = [
            portsMacOf(twoMacs, 'jterrazz-macbook-pro', true)?.name,
            portsMacOf(stories, 'macbook-pro', true)?.this_mac,
            portsMacOf(stories, 'macbook-pro', false),
        ];

        // Then - by name in the first; by the flag in the second, and only when it is This Mac
        expect(found).toStrictEqual(['jterrazz-macbook-pro', true, undefined]);
    });

    test("a create's refusal is placed by its code, then by whether its subject is a Mac the sheet offers", () => {
        // Given - a refusal of each placed code, one naming an offered Mac, one naming a machine, and no answer
        const macs = ['jterrazz-macbook-pro', 'jterrazz-mac-mini'];
        const refusals = [
            new EngineError({
                code: 'machines.bad-purpose',
                message: 'a purpose is one line',
                subject: 'x',
            }),
            new EngineError({
                code: 'machines.too-little-memory',
                message: 'too little',
                subject: 'jterrazz-mac-mini',
            }),
            new EngineError({
                code: 'machinefile.not-a-template',
                message: 'makes no image',
                subject: 'linux-dev',
            }),
            new EngineError({ code: 'machinefile.not-found', message: 'none', subject: 'x' }),
            new EngineError({
                code: 'machines.server-template',
                message: 'no server',
                subject: 'jterrazz-macbook-pro',
            }),
            new EngineError({
                code: 'machines.not-enough-memory',
                message: 'not enough',
                subject: 'jterrazz-macbook-pro',
            }),
            new EngineError({ code: 'machines.name-taken', message: 'taken', subject: 'linux-1' }),
            new EngineError(undefined),
        ];

        // When - each is placed
        const places = refusals.map((error) => createRefusalPlaceOf(error, macs));

        // Then - by code first, even when the subject is an offered Mac; then on the On row, then the foot twice
        expect(places).toStrictEqual([
            'purpose',
            'memory',
            'file',
            'file',
            'file',
            'on',
            'foot',
            'foot',
        ]);
    });
});
