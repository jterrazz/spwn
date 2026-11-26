import type { DiskAmount, DiskRow } from '@jterrazz/station-api';

import { bytes } from '../../../domain/format';
import { wordOfSent } from '../../words/engine-words';
import { diskGestureOf } from '../../words/provisional';
import { DISK_KIND_STEPS, UNKNOWN_DISK_STEP } from '../../words/tones';
import type { DiskStep } from '../../words/tones';
import { DISK_PENDING_WORD } from '../../words/words';

// value: the engine's own line for a disk almost full (apps/engine/activity/findings.go): under a tenth free.
const LOW_FREE_SHARE = 0.1;

export function isLow({ Free, Total }: { Free: number; Total: number }): boolean {
    return Total > 0 && Free >= 0 && Free / Total < LOW_FREE_SHARE;
}

export type Group = { group: DiskRow['Group']; rows: readonly DiskRow[] };

export function groupsOf(rows: readonly DiskRow[]): readonly Group[] {
    const groups: { group: DiskRow['Group']; rows: DiskRow[] }[] = [];
    for (const row of rows) {
        const held = groups.find((candidate) => candidate.group === row.Group);
        if (held) {
            held.rows.push(row);
        } else {
            groups.push({ group: row.Group, rows: [row] });
        }
    }
    return groups;
}

export function sizeOf(row: DiskRow): string {
    if (row.Pending) {
        return DISK_PENDING_WORD;
    }
    return row.Bytes < 0 ? 'size not read' : bytes(row.Bytes);
}

export type Part = {
    key: string;
    bytes: number;
    step: DiskStep | undefined;
    label: string;
    rest: boolean;
};

function stepOf(code: string): DiskStep | undefined {
    if (code === 'free') {
        return undefined;
    }
    const steps: Readonly<Record<string, DiskStep>> = DISK_KIND_STEPS;
    return Object.hasOwn(steps, code) ? (steps[code] ?? UNKNOWN_DISK_STEP) : UNKNOWN_DISK_STEP;
}

type Amount = Pick<DiskAmount, 'Bytes'> & { Kind: string };

export function kindPartsOf(kinds: readonly Amount[] | undefined): readonly Part[] {
    return (kinds ?? []).map(({ Kind, Bytes }) => ({
        bytes: Bytes,
        key: Kind,
        label: wordOfSent('DiskKind', Kind),
        rest: Kind === 'free',
        step: stepOf(Kind),
    }));
}

export function listedOf(parts: readonly Part[]): readonly Part[] {
    return parts.filter((part) => part.rest || part.bytes > 0);
}

export type AppRow = { key: string; app: string; rows: readonly DiskRow[] };

function keyOf(row: DiskRow): string {
    return diskGestureOf(row.Group)?.call === 'disk.move'
        ? `${row.App}/${(row.Paths ?? []).join(':')}`
        : row.App;
}

export function byApp(rows: readonly DiskRow[]): readonly AppRow[] {
    const apps: { key: string; app: string; rows: DiskRow[] }[] = [];
    for (const row of rows) {
        const key = keyOf(row);
        const held = apps.find((candidate) => candidate.key === key);
        if (held) {
            held.rows.push(row);
        } else {
            apps.push({ app: row.App, key, rows: [row] });
        }
    }
    return apps;
}
