import type { DiskReading, DiskRow } from '@jterrazz/station-api';

import type { Identity } from '../../design/index';
import { diskGestureOf, freesWhole } from '../../engine/provisional';
import { DISK_GROUP_COLOURS } from '../../engine/tones';
import { DISK_GROUP_WORDS, DISK_PENDING_WORD } from '../../engine/words';
import { bytes } from '../../model/format';

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

function identityOfGroup(group: DiskRow['Group']): Identity {
    return DISK_GROUP_COLOURS[group];
}

export type Part = { group: DiskRow['Group']; bytes: number; identity: Identity; label: string };

function readOf(rows: readonly DiskRow[]): number {
    let read = 0;
    for (const row of rows) {
        read += Math.max(0, row.Bytes);
    }
    return read;
}

export function rebuildableOf(rows: readonly DiskRow[]): number {
    return readOf(rows.filter((row) => freesWhole(row.Group)));
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

export function partsOf(reading: DiskReading): readonly Part[] {
    return groupsOf(reading.Rows ?? []).map(({ group, rows }) => ({
        bytes: readOf(rows),
        group,
        identity: identityOfGroup(group),
        label: DISK_GROUP_WORDS[group],
    }));
}
