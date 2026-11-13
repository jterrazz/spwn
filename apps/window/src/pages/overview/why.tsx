import type { ActivityReport, MachinesVM } from '@jterrazz/station-api';

import { MissingMark, Panel, Row, Value } from '../../design/index';
import type { Found } from '../../model/machines';
import { BusyShare } from '../activity/busiest';
import { busiestOf } from '../activity/measures';
import { useActivity } from '../activity/reading';
import { shortDay } from '../activity/when';

import styles from './overview.module.css';

const BUSIEST_ROWS = 3;

export function WhyItExists({ vm }: { vm: MachinesVM }) {
    const created = shortDay(vm.CreatedAt);
    return (
        <Panel padded title="Why it exists">
            <dl className={styles.kv}>
                <dt>Purpose</dt>
                <dd>
                    <MissingMark />
                </dd>
                <dt>Created</dt>
                <dd>
                    {created ?? <MissingMark />}
                    {', by '}
                    <MissingMark />
                </dd>
                <dt>Hold</dt>
                <dd>
                    <MissingMark />
                </dd>
                {vm.Check === '' ? null : (
                    <>
                        <dt>Check</dt>
                        <dd>
                            <Value>{vm.Check}</Value>
                        </dd>
                    </>
                )}
            </dl>
        </Panel>
    );
}

function Busiest({ report }: { report: ActivityReport }) {
    const { top, sample } = report.last;
    const rows = busiestOf(top.cpu ?? [], 'cpu', sample.mem_total).slice(0, BUSIEST_ROWS);
    return (
        <Panel title="Busiest now">
            {rows.length === 0 ? (
                <p className={styles.none}>The engine listed no process.</p>
            ) : null}
            {rows.map((row) => (
                <div className={styles.process} key={row.key}>
                    <Row
                        size="slim"
                        title={row.name}
                        value={
                            <BusyShare
                                label={`CPU of ${row.name}`}
                                share={row.share}
                                shown={row.shown}
                            />
                        }
                    />
                </div>
            ))}
        </Panel>
    );
}

export function BusiestNow({ found }: { found: Found }) {
    const { readers, shown } = useActivity(found, '1h');
    return (
        <>
            {readers}
            {shown?.kind === 'read' ? <Busiest report={shown.out} /> : null}
        </>
    );
}
