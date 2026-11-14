import type { DiskReading, DiskRow, Target } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { AppIcon, Button, Panel, ShareBar, TypeTag } from '../../design/index';
import type { SharePart } from '../../design/index';
import { SEPARATOR } from '../../design/separator';
import { diskGestureOf, freesWhole, holdsRebuildable } from '../../engine/provisional';
import { useReread } from '../../engine/react';
import { DISK_GROUP_WORDS, DISK_KIND_WORDS, DISK_PENDING_WORD } from '../../engine/words';
import { bytes, sentence } from '../../model/format';
import { byApp, groupsOf, partsOf, rebuildableOf, sizeOf } from './disk';
import type { AppRow, Group, Part } from './disk';
import { DiskLine, EraseLink, FreeAllButton } from './disk-gestures';
import type { DiskGesture } from './disk-gestures';

import pageStyles from '../page.module.css';
import styles from './disk.module.css';

function Used({ reading }: { reading: DiskReading }) {
    const { Free, Total } = reading;
    if (Total <= 0) {
        return <span />;
    }
    return (
        <p className={styles.used}>
            <span className={styles.figure}>{bytes(Total - Free)}</span>
            {`of ${bytes(Total)} used`}
        </p>
    );
}

function DiskBar({
    parts,
    reading,
    label,
}: {
    parts: readonly Part[];
    reading: DiskReading;
    label: string;
}) {
    const { Free, Total } = reading;
    if (Total <= 0) {
        return null;
    }
    const held: SharePart[] = parts
        .filter((part) => part.bytes > 0)
        .map((part) => ({
            identity: part.identity,
            key: part.group,
            label: `${part.label}${SEPARATOR}${bytes(part.bytes)}`,
            size: part.bytes,
        }));
    const unsorted = Total - Free - held.reduce((sum, part) => sum + part.size, 0);
    const used: SharePart[] =
        unsorted > 0
            ? [{ key: 'used', label: `Used${SEPARATOR}${bytes(unsorted)}`, size: unsorted }]
            : [];
    const free: SharePart[] =
        Free > 0
            ? [{ key: 'free', label: `Free${SEPARATOR}${bytes(Free)}`, rest: true, size: Free }]
            : [];
    return <ShareBar label={label} of={Total} parts={[...held, ...used, ...free]} />;
}

function Legend({ parts, free }: { parts: readonly Part[]; free: number }) {
    return (
        <ul aria-label="The groups" className={styles.legend}>
            {parts.map((part) => (
                <li data-identity={part.identity} key={part.group}>
                    <i className={styles.swatch} />
                    <span className={styles.word}>{part.label}</span>
                    {part.bytes > 0 ? <b className={styles.size}>{bytes(part.bytes)}</b> : null}
                </li>
            ))}
            {free > 0 ? (
                <li className={styles.free}>
                    <i className={styles.swatch} />
                    <span className={styles.word}>Free</span>
                    <b className={styles.size}>{bytes(free)}</b>
                </li>
            ) : null}
        </ul>
    );
}

type FootProps = { reading: DiskReading; freeAll: DiskGesture; name: string };

function Foot({ reading, freeAll, name }: FootProps) {
    if (!holdsRebuildable(reading)) {
        return null;
    }
    const held = bytes(rebuildableOf(reading.Rows ?? []));
    return (
        <div className={styles.foot}>
            <p>
                {`What can be rebuilt holds ${held}. `}
                <span className={styles.quiet}>Each app rebuilds its own when it needs it.</span>
            </p>
            <FreeAllButton gesture={freeAll} label={`Free ${held}`} name={name} variant="primary" />
        </div>
    );
}

type DiskPanelProps = {
    reading: DiskReading;
    name: string;
    title: string;
    freeAll: DiskGesture;
    sub?: ReactNode;
};

export function DiskPanel({ reading, name, title, freeAll, sub }: DiskPanelProps) {
    const parts = partsOf(reading);
    const reread = useReread();
    return (
        <Panel note={reading.Pending ? DISK_PENDING_WORD : ''} padded title={title}>
            <div className={styles.hero}>
                <Used reading={reading} />
                <Button
                    onClick={() => {
                        reread(['disk.read']);
                    }}
                >
                    Rescan
                </Button>
            </div>
            {sub === undefined ? null : <p className={styles.sub}>{sub}</p>}
            <DiskBar label={`What holds the disk of ${name}`} parts={parts} reading={reading} />
            <Legend free={reading.Free} parts={parts} />
            <Foot freeAll={freeAll} name={name} reading={reading} />
        </Panel>
    );
}

type GroupProps = { group: Group; target: Target | undefined };

export function GroupPanel({ group, target }: GroupProps) {
    return (
        <Panel note={String(group.rows.length)} title={DISK_GROUP_WORDS[group.group]}>
            {group.rows.map((row) => (
                <DiskLine
                    key={`${row.App}/${(row.Paths ?? []).join(':')}`}
                    lead={<AppIcon name={row.App} />}
                    row={row}
                    sentence={sentence([
                        (row.Paths ?? []).join(SEPARATOR),
                        row.MoveTo === '' ? undefined : `to ${row.MoveTo}`,
                    ])}
                    size={sizeOf(row)}
                    target={target}
                />
            ))}
        </Panel>
    );
}

export function groupOf(
    reading: DiskReading,
    wanted: (row: DiskRow) => boolean,
): Group | undefined {
    return groupsOf((reading.Rows ?? []).filter(wanted))[0];
}

function chipSize(rows: readonly DiskRow[]): string {
    const [only] = rows;
    if (rows.length === 1 && only !== undefined) {
        return sizeOf(only);
    }
    if (rows.some((row) => row.Pending)) {
        return DISK_PENDING_WORD;
    }
    return bytes(rows.reduce((sum, row) => sum + Math.max(0, row.Bytes), 0));
}

function Chips({ rows }: { rows: readonly DiskRow[] }) {
    return (
        <>
            {groupsOf(rows).map((group) => (
                <TypeTag key={group.group}>
                    {`${DISK_KIND_WORDS[group.group]} ${chipSize(group.rows)}`}
                </TypeTag>
            ))}
        </>
    );
}

function AppLine({ app, target }: { app: AppRow; target: Target | undefined }) {
    const [first] = app.rows;
    const row = app.rows.find((one) => freesWhole(one.Group)) ?? first;
    if (row === undefined) {
        return null;
    }
    const data = app.rows.find((one) => diskGestureOf(one.Group)?.call === 'disk.erase');
    const moves = app.rows.map((one) => one.MoveTo).filter((to) => to !== '');
    const erasedApart = data !== undefined && data !== row;
    return (
        <DiskLine
            chips={
                <>
                    <Chips rows={app.rows} />
                    {erasedApart ? <EraseLink row={data} target={target} /> : null}
                </>
            }
            lead={<AppIcon name={app.app} />}
            row={row}
            sentence={sentence([
                app.rows.flatMap((one) => one.Paths ?? []).join(SEPARATOR),
                ...moves.map((to) => `to ${to}`),
            ])}
            target={target}
        />
    );
}

type ByAppProps = { reading: DiskReading; name: string; target: Target | undefined; foot: string };

export function ByApp({ reading, name, target, foot }: ByAppProps) {
    const apps = byApp(reading.Rows ?? []);
    return (
        <Panel
            note="each app says in the catalog which of its folders are cache, data or settings"
            title="By app"
        >
            {apps.length === 0 ? (
                <p className={pageStyles.missing}>Nothing listed on {name}.</p>
            ) : (
                apps.map((app) => <AppLine app={app} key={app.key} target={target} />)
            )}
            <p className={styles.more}>{foot}</p>
        </Panel>
    );
}
