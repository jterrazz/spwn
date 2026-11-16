import type { DiskReading, DiskRow, Target } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { bytes, sentence } from '../../../domain/format';
import { SEPARATOR } from '../../../domain/separator';
import { useReread } from '../../dependencies/dependencies';
import { AppIcon, Button, Panel, ShareBar, TypeTag } from '../../ui/index';
import type { SharePart } from '../../ui/index';
import { detailOf, wordOf } from '../../words/engine-words';
import { diskGestureOf, freesWhole } from '../../words/provisional';
import { DISK_PENDING_WORD } from '../../words/words';
import { byApp, groupsOf, holdPartsOf, kindPartsOf, sizeOf } from './disk';
import type { AppRow, Group, Part } from './disk';
import { DiskLine, EraseLink, FreeAllButton } from './disk-gestures';
import type { DiskGesture } from './disk-gestures';

import pageStyles from '../../ui/page.module.css';
import styles from './disk.module.css';

function Used({ reading }: { reading: DiskReading }) {
    const { Total, Used: used } = reading;
    if (Total <= 0) {
        return <span />;
    }
    return (
        <p className={styles.used}>
            <span className={styles.figure}>{bytes(used)}</span>
            {`of ${bytes(Total)} used`}
        </p>
    );
}

function DiskBar({
    parts,
    total,
    label,
}: {
    parts: readonly Part[];
    total: number;
    label: string;
}) {
    if (total <= 0 || parts.length === 0) {
        return null;
    }
    const held: SharePart[] = parts.map((part) => ({
        identity: part.rest ? undefined : part.identity,
        key: part.key,
        label: `${part.label}${SEPARATOR}${bytes(part.bytes)}`,
        rest: part.rest,
        size: part.bytes,
    }));
    return <ShareBar label={label} of={total} parts={held} />;
}

function Legend({ parts }: { parts: readonly Part[] }) {
    if (parts.length === 0) {
        return null;
    }
    return (
        <ul aria-label="The parts" className={styles.legend}>
            {parts.map((part) => (
                <li
                    className={part.rest ? styles.free : undefined}
                    data-identity={part.rest ? undefined : part.identity}
                    key={part.key}
                >
                    <i className={styles.swatch} />
                    <span className={styles.word}>{part.label}</span>
                    <b className={styles.size}>{bytes(part.bytes)}</b>
                </li>
            ))}
        </ul>
    );
}

type FootProps = { reading: DiskReading; freeAll: DiskGesture; name: string };

function Foot({ reading, freeAll, name }: FootProps) {
    const { CanFree } = reading;
    if (CanFree === undefined) {
        return null;
    }
    return (
        <div className={styles.foot}>
            <p>{CanFree.Sentence}</p>
            {CanFree.Call === undefined ? null : (
                <FreeAllButton
                    gesture={freeAll}
                    label={`Free ${bytes(CanFree.Bytes)}`}
                    name={name}
                    variant="primary"
                />
            )}
        </div>
    );
}

type DiskPanelProps = {
    reading: DiskReading;
    name: string;
    title: string;
    freeAll: DiskGesture;
    by: 'holds' | 'kinds';
    sub?: ReactNode;
    station?: ReactNode;
};

export function DiskPanel({ reading, name, title, freeAll, by, sub, station }: DiskPanelProps) {
    const parts = by === 'holds' ? holdPartsOf(reading.Station?.Holds) : kindPartsOf(reading.Kinds);
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
            <DiskBar label={`What holds the disk of ${name}`} parts={parts} total={reading.Total} />
            <Legend parts={parts} />
            {station}
            <Foot freeAll={freeAll} name={name} reading={reading} />
        </Panel>
    );
}

type GroupProps = { group: Group; target: Target | undefined };

export function GroupPanel({ group, target }: GroupProps) {
    return (
        <Panel note={String(group.rows.length)} title={wordOf('DiskGroup', group.group).Word}>
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
                    {`${detailOf('DiskGroup', group.group)} ${chipSize(group.rows)}`}
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
