import type { DiskAmount, DiskRow } from '@jterrazz/station-api';

import { bytes } from '../../../domain/format';
import type { Identity } from '../../ui/index';
import { wordOfSent } from '../../words/engine-words';
import { diskGestureOf } from '../../words/provisional';
import { DISK_KIND_COLOURS, UNKNOWN_SHARE_COLOUR } from '../../words/tones';
import type { ShareColour } from '../../words/tones';
import { DISK_PENDING_WORD } from '../../words/words';

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
    identity: Identity;
    label: string;
    rest: boolean;
};

function colourOf(table: Readonly<Record<string, ShareColour>>, code: string): ShareColour {
    return Object.hasOwn(table, code)
        ? (table[code] ?? UNKNOWN_SHARE_COLOUR)
        : UNKNOWN_SHARE_COLOUR;
}

type Amount = Pick<DiskAmount, 'Bytes'> & { Kind: string };

export function kindPartsOf(kinds: readonly Amount[] | undefined): readonly Part[] {
    return (kinds ?? []).map(({ Kind, Bytes }) => ({
        bytes: Bytes,
        identity: colourOf(DISK_KIND_COLOURS, Kind),
        key: Kind,
        label: wordOfSent('DiskKind', Kind),
        rest: Kind === 'free',
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
