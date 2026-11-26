import type { ActivityReport, MachinesVM } from '@jterrazz/station-api';

import { busiestOf } from '../../../../domain/activity/measures';
import type { Found } from '../../../../domain/machines/machines';
import { shortDay } from '../../../../domain/when';
import { BusyShare } from '../../../features/activity/busiest';
import { useActivity } from '../../../features/activity/reading';
import { MissingMark, Panel, Row, Value } from '../../../ui/index';
import { NO_PURPOSE_WORD } from '../../../words/words';

import styles from './overview.module.css';

const BUSIEST_ROWS = 3;

function sentenceBy(created: string, creator: string): string {
    return creator === '' ? created : `${created}, by ${creator}`;
}

export function WhyItExists({ vm }: { vm: MachinesVM }) {
    const created = shortDay(vm.CreatedAt);
    return (
        <Panel padded title="Why it exists">
            <dl className={styles.kv}>
                <dt>Purpose</dt>
                <dd>{vm.Purpose === '' ? NO_PURPOSE_WORD : vm.Purpose}</dd>
                <dt>Created</dt>
                <dd>{created === undefined ? <MissingMark /> : sentenceBy(created, vm.Creator)}</dd>
                {vm.Hold === undefined ? null : (
                    <>
                        <dt>Hold</dt>
                        <dd>{vm.Hold.Sentence}</dd>
                    </>
                )}
                {vm.Check === '' ? null : (
                    <>
                        <dt>Check</dt>
                        <dd>
                            <Value look="raised">{vm.Check}</Value>
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
    const shown = useActivity(found, '1h');
    return shown?.kind === 'read' ? <Busiest report={shown.out} /> : null;
}
