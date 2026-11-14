import type { CoreVerdictVerdict } from '@jterrazz/station-api';

import type { MachinesCallsNewOn, NewMachineAnswer } from './machines-new';

// outside: the answers machines.new gives over the bench's worlds, written by hand until the call is on the wire and
// outside: recorded: the two-Mac one with the mini silent is the engine's story
// outside: specs/usecases/machines/new/the-sheet-of-two-macs (a71316035); each other is that answer over its world's
// outside: machines.list (its Rooms, its Roomiest), every verdict written as the engine's, none computed here.

const GB = 2 ** 30;

const CALM: CoreVerdictVerdict = { Sentence: '', Tone: 'plain' };

const STATION_LINUX: Omit<NewMachineAnswer['out'], 'Mac' | 'Name' | 'On'> = {
    From: 'station-linux',
    Lifetime: 'persistent',
    Lifetimes: ['ephemeral', 'persistent'],
    MinMemory: 2 * GB,
    PurposeMax: 120,
    Server: {
        Image: 'station-linux',
        Size: { CPUs: 4, DiskBytes: 50_000_000_000, MemoryBytes: 8 * GB },
    },
    Size: { CPUs: 4, DiskBytes: 20_000_000_000, MemoryBytes: 4 * GB },
};

const PRO: MachinesCallsNewOn = {
    Mac: 'jterrazz-macbook-pro',
    MaxMemory: 4 * GB,
    Ready: true,
    Room: {
        Capacity: {
            Free: 4 * GB,
            MacOS: 4 * GB,
            Total: 16 * GB,
            VMs: [{ Bytes: 8 * GB, Name: 'linux-1' }],
        },
        Mac: 'jterrazz-macbook-pro',
        Presets: [
            { Fit: 'fits', MemoryBytes: 4 * GB, Preset: 'small' },
            { Fit: 'no-room', MemoryBytes: 8 * GB, Preset: 'workspace' },
            { Fit: 'no-room', MemoryBytes: 12 * GB, Preset: 'gate' },
        ],
        RoomFor: 'small',
    },
    Server: { Fit: 'no-room', Verdict: CALM },
    ThisMac: true,
    Verdict: CALM,
};

const MINI: MachinesCallsNewOn = {
    Mac: 'jterrazz-mac-mini',
    MaxMemory: 36 * GB,
    Ready: true,
    Room: {
        Capacity: {
            Free: 36 * GB,
            MacOS: 4 * GB,
            Total: 64 * GB,
            VMs: [{ Bytes: 24 * GB, Name: 'infra' }],
        },
        Mac: 'jterrazz-mac-mini',
        Presets: [
            { Fit: 'fits', MemoryBytes: 4 * GB, Preset: 'small' },
            { Fit: 'fits', MemoryBytes: 8 * GB, Preset: 'workspace' },
            { Fit: 'fits', MemoryBytes: 12 * GB, Preset: 'gate' },
        ],
        RoomFor: 'gate',
    },
    Server: { Fit: 'fits', Verdict: CALM },
    ThisMac: false,
    Verdict: CALM,
};

const MINI_SILENT: MachinesCallsNewOn = {
    Mac: 'jterrazz-mac-mini',
    MaxMemory: 0,
    Ready: false,
    Room: null,
    Server: null,
    ThisMac: false,
    Verdict: { Sentence: 'mac-mini does not answer', Tone: 'bad' },
};

const PRO_ALONE: MachinesCallsNewOn = {
    Mac: 'jterrazz-macbook-pro',
    MaxMemory: 8 * GB,
    Ready: true,
    Room: null,
    Server: { Fit: 'fits', Verdict: CALM },
    ThisMac: true,
    Verdict: CALM,
};

const STORIES_PRO: MachinesCallsNewOn = {
    Mac: 'macbook-pro',
    MaxMemory: 28 * GB,
    Ready: true,
    Room: {
        Capacity: {
            Free: 28 * GB,
            MacOS: 4 * GB,
            Total: 48 * GB,
            VMs: [
                { Bytes: 8 * GB, Name: 'agent-7f' },
                { Bytes: 8 * GB, Name: 'linux-1' },
            ],
        },
        Mac: 'macbook-pro',
        Presets: [
            { Fit: 'fits', MemoryBytes: 4 * GB, Preset: 'small' },
            { Fit: 'fits', MemoryBytes: 8 * GB, Preset: 'workspace' },
            { Fit: 'fits', MemoryBytes: 12 * GB, Preset: 'gate' },
        ],
        RoomFor: 'gate',
    },
    Server: { Fit: 'fits', Verdict: CALM },
    ThisMac: true,
    Verdict: CALM,
};

const STORIES_MINI: MachinesCallsNewOn = {
    Mac: 'mac-mini',
    MaxMemory: 24 * GB,
    Ready: true,
    Room: {
        Capacity: {
            Free: 24 * GB,
            MacOS: 4 * GB,
            Total: 64 * GB,
            VMs: [
                { Bytes: 24 * GB, Name: 'infra' },
                { Bytes: 8 * GB, Name: 'linux-3' },
                { Bytes: 4 * GB, Name: 'review-bot' },
            ],
        },
        Mac: 'mac-mini',
        Presets: [
            { Fit: 'fits', MemoryBytes: 4 * GB, Preset: 'small' },
            { Fit: 'fits', MemoryBytes: 8 * GB, Preset: 'workspace' },
            { Fit: 'fits', MemoryBytes: 12 * GB, Preset: 'gate' },
        ],
        RoomFor: 'gate',
    },
    Server: { Fit: 'fits', Verdict: CALM },
    ThisMac: false,
    Verdict: CALM,
};

const STORIES_AIR: MachinesCallsNewOn = {
    Mac: 'macbook-air',
    MaxMemory: 0,
    Ready: false,
    Room: null,
    Server: null,
    ThisMac: false,
    Verdict: { Sentence: 'macbook-air is asleep', Tone: 'plain' },
};

export const ANSWERS: readonly NewMachineAnswer[] = [
    {
        fleet: 'jterrazz-macbook-pro jterrazz-mac-mini',
        out: { ...STATION_LINUX, Mac: 'jterrazz-mac-mini', Name: 'linux-2', On: [PRO, MINI] },
    },
    {
        fleet: 'jterrazz-macbook-pro jterrazz-mac-mini?',
        out: {
            ...STATION_LINUX,
            Mac: 'jterrazz-macbook-pro',
            Name: 'linux-2',
            On: [PRO, MINI_SILENT],
        },
    },
    {
        fleet: 'jterrazz-macbook-pro',
        out: { ...STATION_LINUX, Mac: 'jterrazz-macbook-pro', Name: 'linux-1', On: [PRO_ALONE] },
    },
    {
        fleet: 'macbook-pro mac-mini macbook-air?',
        out: {
            ...STATION_LINUX,
            Mac: 'macbook-pro',
            Name: 'linux-2',
            On: [STORIES_PRO, STORIES_MINI, STORIES_AIR],
        },
    },
];
