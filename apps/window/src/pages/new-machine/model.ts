import type {
    ImagesListOutput,
    MachinesCreateServerInput,
    MachinesCreateWorkspaceInput,
    MachinesResourcesInput,
} from '@jterrazz/station-api';

import type { Lifetime } from '../../engine/provisional';

export type Kind = 'server' | 'workspace';

export type Draft = {
    kind: Kind;
    name: string;
    image: null | string;
    lifetime: Lifetime | null;
    cpus: string;
    memoryGB: string;
    diskGB: string;
};

export const EMPTY_DRAFT: Draft = {
    cpus: '',
    diskGB: '',
    image: null,
    kind: 'workspace',
    lifetime: null,
    memoryGB: '',
    name: '',
};

const GB = 2 ** 30;

export function countOf(text: string): number | undefined {
    const trimmed = text.trim();
    if (!/^\d+$/u.test(trimmed)) {
        return undefined;
    }
    const count = Number(trimmed);
    return count > 0 ? count : undefined;
}

export function sizeIsWhole(draft: Draft): boolean {
    return [draft.cpus, draft.memoryGB, draft.diskGB].every(
        (text) => text.trim() === '' || countOf(text) !== undefined,
    );
}

function sizeOf(draft: Draft): MachinesResourcesInput | undefined {
    const cpus = countOf(draft.cpus);
    const memory = countOf(draft.memoryGB);
    const disk = countOf(draft.diskGB);
    const size: MachinesResourcesInput = {
        ...(cpus === undefined ? {} : { CPUs: cpus }),
        ...(memory === undefined ? {} : { MemoryBytes: memory * GB }),
        ...(disk === undefined ? {} : { DiskBytes: disk * GB }),
    };
    return Object.keys(size).length === 0 ? undefined : size;
}

function named(draft: Draft): { Name?: string; Image?: string; Size?: MachinesResourcesInput } {
    const name = draft.name.trim();
    const size = sizeOf(draft);
    return {
        ...(name === '' ? {} : { Name: name }),
        ...(draft.image === null ? {} : { Image: draft.image }),
        ...(size === undefined ? {} : { Size: size }),
    };
}

export function workspaceInputOf(draft: Draft): MachinesCreateWorkspaceInput {
    return {
        Request: {
            ...named(draft),
            ...(draft.lifetime === null ? {} : { Lifetime: draft.lifetime }),
        },
    };
}

export function serverInputOf(draft: Draft): MachinesCreateServerInput {
    return named(draft);
}

export function imagesOf(reports: ImagesListOutput): readonly string[] {
    return [...new Set((reports ?? []).map((report) => report.Record.name))];
}
