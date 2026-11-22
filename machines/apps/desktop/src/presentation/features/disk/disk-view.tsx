import type { DiskReading, DiskRow, Target } from '@jterrazz/station-api';

import { bytes, sentence } from '../../../domain/format';
import { SEPARATOR } from '../../../domain/separator';
import { AppIcon, Panel, TypeTag } from '../../ui/index';
import { detailOf, wordOf } from '../../words/engine-words';
import { diskGestureOf, freesWhole } from '../../words/provisional';
import { DISK_PENDING_WORD } from '../../words/words';
import { byApp, groupsOf, sizeOf } from './disk';
import type { AppRow, Group } from './disk';
import { DiskLine, EraseLink } from './disk-gestures';

import pageStyles from '../../ui/page.module.css';
import styles from './disk.module.css';

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
                        row.Note,
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
                ...app.rows.map((one) => one.Note),
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
