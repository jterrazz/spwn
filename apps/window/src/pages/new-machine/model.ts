import type {
    MachinesCallsNewIn,
    MachinesCallsNewMachine,
    MachinesCallsNewOn,
    MachinesCreateServerInput,
    MachinesCreateWorkspaceInput,
    MachinesResourcesInput,
    Target,
} from '@jterrazz/station-api';

import type { Opening } from '../../shell/window';

export type Kind = 'server' | 'workspace';

export type Lifetime = NonNullable<MachinesCallsNewMachine['Lifetimes']>[number];

export type { Opening };

export type Draft = {
    kind: Kind;
    name: null | string;
    purpose: string;
    on: null | string;
    memory: null | number;
    lifetime: Lifetime | null;
};

export const EMPTY_DRAFT: Draft = {
    kind: 'workspace',
    lifetime: null,
    memory: null,
    name: null,
    on: null,
    purpose: '',
};

export const GB = 2 ** 30;

export function gbOf(bytes: number): number {
    return Math.round(bytes / GB);
}

export function askedOf(opening: Opening, draft: Draft): MachinesCallsNewIn {
    const on = draft.on ?? opening.on;
    return {
        ...(opening.from === undefined ? {} : { From: opening.from }),
        ...(on === undefined ? {} : { On: on }),
    };
}

export function chosenOnOf(answer: MachinesCallsNewMachine): MachinesCallsNewOn | undefined {
    return (answer.On ?? []).find((on) => on.Mac === answer.Mac);
}

function defaultMemoryOf(answer: MachinesCallsNewMachine, kind: Kind): number {
    return kind === 'server' && answer.Server !== null
        ? answer.Server.Size.MemoryBytes
        : answer.Size.MemoryBytes;
}

// invariant: the memory asked stays within the range the engine offers on the Mac chosen, MinMemory to its
// invariant: MaxMemory; an empty range leaves nothing to ask, and the sheet launches nothing.
export function memoryOf(
    answer: MachinesCallsNewMachine,
    on: MachinesCallsNewOn,
    draft: Draft,
): number | undefined {
    const wanted = draft.memory ?? defaultMemoryOf(answer, draft.kind);
    if (wanted <= on.MaxMemory) {
        return wanted;
    }
    return on.MaxMemory >= answer.MinMemory ? on.MaxMemory : undefined;
}

export function leavesFreeOf(on: MachinesCallsNewOn, memory: number | undefined): string {
    if (on.Room === null || memory === undefined) {
        return '';
    }
    const left = gbOf(on.Room.Capacity.Free) - gbOf(memory);
    return left < 0 ? '' : `leaves ${String(left)} GB free on ${on.ThisMac ? 'this Mac' : on.Mac}`;
}

export function nameOf(answer: MachinesCallsNewMachine, draft: Draft): string {
    return (draft.name ?? answer.Name).trim();
}

export function targetOfOn(on: MachinesCallsNewOn): Target | undefined {
    return on.ThisMac ? undefined : { name: on.Mac };
}

type Asked = { answer: MachinesCallsNewMachine; draft: Draft; memory: number | undefined };

function sizeOf({ answer, draft, memory }: Asked): MachinesResourcesInput | undefined {
    return memory === undefined || memory === defaultMemoryOf(answer, draft.kind)
        ? undefined
        : { MemoryBytes: memory };
}

function shared(asked: Asked): { Name?: string; Purpose?: string; Size?: MachinesResourcesInput } {
    const name = nameOf(asked.answer, asked.draft);
    const purpose = asked.draft.purpose.trim();
    const size = sizeOf(asked);
    return {
        ...(name === '' ? {} : { Name: name }),
        ...(purpose === '' ? {} : { Purpose: purpose }),
        ...(size === undefined ? {} : { Size: size }),
    };
}

export function workspaceInputOf(asked: Asked): MachinesCreateWorkspaceInput {
    return {
        Request: {
            ...shared(asked),
            From: asked.answer.From,
            Lifetime: asked.draft.lifetime ?? asked.answer.Lifetime,
        },
    };
}

export function serverInputOf(asked: Asked): MachinesCreateServerInput {
    const image = asked.answer.Server?.Image;
    return { ...shared(asked), ...(image === undefined ? {} : { Image: image }) };
}
