import type {
    CoreVerdictVerdict,
    MachinesCapacity,
    MachinesListOutput,
    MachinesResources,
} from '@jterrazz/station-api';

import { EngineError } from './engine-error';
import { READING } from './entry';
import type { Reading } from './entry';
import { ANSWERS } from './machines-new-answers';
import { useWatch } from './react';

// outside: `machines.new` is not on the wire of this tree's packages/api. These types copy, name for name, the ones
// outside: the engine generates for it (feat/window-fields-c6b, a71316035); they go, with this file and
// outside: machines-new-answers.ts, when packages/api carries them: the sheet then imports them from
// outside: @jterrazz/station-api, and useMachinesNew becomes the watch of machines.new with its input.

export type MachinesFit = 'fits' | 'no-room';

export type MachinesPreset = 'gate' | 'small' | 'workspace';

export type MachinesCallsNewIn = {
    From?: string;
    On?: string;
};

export type MachinesPresetOn = {
    Fit: 'fits' | 'no-room';
    MemoryBytes: number;
    Preset: 'gate' | 'small' | 'workspace';
};

export type MachinesRoom = {
    Capacity: MachinesCapacity;
    Mac: string;
    Presets: MachinesPresetOn[] | null;
    RoomFor: 'gate' | 'none' | 'small' | 'workspace';
};

export type MachinesCallsNewServerOn = {
    Fit: 'fits' | 'no-room';
    Verdict: CoreVerdictVerdict;
};

export type MachinesCallsNewOn = {
    Mac: string;
    MaxMemory: number;
    Ready: boolean;
    Room: MachinesRoom | null;
    Server: MachinesCallsNewServerOn | null;
    ThisMac: boolean;
    Verdict: CoreVerdictVerdict;
};

export type MachinesCallsNewServer = {
    Image: string;
    Size: MachinesResources;
};

export type MachinesCallsNewMachine = {
    From: string;
    Lifetime: 'always-on' | 'ephemeral' | 'persistent';
    Lifetimes: ('always-on' | 'ephemeral' | 'persistent')[] | null;
    Mac: string;
    MinMemory: number;
    Name: string;
    On: MachinesCallsNewOn[] | null;
    PurposeMax: number;
    Server: MachinesCallsNewServer | null;
    Size: MachinesResources;
};

export type NewMachineAnswer = { fleet: string; out: MachinesCallsNewMachine };

const NOT_ON_THE_WIRE = new EngineError(
    undefined,
    'machines.new is not on the wire yet, and no answer is written for these Macs',
);

// outside: an answer is chosen by the Macs machines.list lists, a silent or asleep one marked `?`, as a recording
// outside: is chosen by its request; the input is applied as the engine applies it, the Mac asked chosen and the
// outside: machine file asked followed.
function fleetOf(out: MachinesListOutput): string {
    return (out.All.Macs ?? [])
        .map((mac) => (mac.Reachable && !mac.Asleep ? mac.Name : `${mac.Name}?`))
        .join(' ');
}

export function answerFor(
    out: MachinesListOutput,
    input: MachinesCallsNewIn,
): MachinesCallsNewMachine | undefined {
    const answer = ANSWERS.find((one) => one.fleet === fleetOf(out))?.out;
    if (answer === undefined) {
        return undefined;
    }
    const asked = (answer.On ?? []).find((on) => on.Mac === input.On)?.Mac;
    return { ...answer, From: input.From ?? answer.From, Mac: asked ?? answer.Mac };
}

export function useMachinesNew(input: MachinesCallsNewIn): Reading<MachinesCallsNewMachine> {
    const listing = useWatch('machines.list', {});
    if (listing.kind !== 'read') {
        return listing;
    }
    const out = answerFor(listing.out, input);
    if (out === undefined) {
        return listing.done ? { error: NOT_ON_THE_WIRE, kind: 'failed' } : READING;
    }
    return { done: listing.done, kind: 'read', out };
}
