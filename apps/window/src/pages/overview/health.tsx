import type { MachinesLatencyPoint, MachinesUptimeDay, MachinesVM } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { Chart, MissingMark, Panel, StatePill, Tile, Value } from '../../design/index';
import type { Measured } from '../../design/index';
import { vmCheck, vmSsh, vmState } from '../../engine/shown';
import { clock, duration, seconds, sentence } from '../../model/format';
import { SEPARATOR } from '../../separator';
import { between, shortDay } from '../activity/when';

import styles from './overview.module.css';

const NS_PER_MS = 1_000_000;
const NS_PER_SECOND = 1_000_000_000;

function millis(took: number): string {
    return String(Math.round(took / NS_PER_MS));
}

function Step({ label, children }: { label: string; children: ReactNode }) {
    return (
        <div className={styles.step}>
            <span className={styles.stepLabel}>{label}</span>
            {children}
        </div>
    );
}

const NS_PER_DAY = 86_400 * NS_PER_SECOND;

export function ageOf(at: string | undefined): string | undefined {
    const span = between(at, new Date().toISOString());
    if (span === undefined) {
        return undefined;
    }
    return `${span < NS_PER_DAY ? seconds(span) : duration(span)} ago`;
}

export function Health({ vm }: { vm: MachinesVM }) {
    const age = ageOf(vm.CheckStatus.at);
    return (
        <Panel note={age === undefined ? '' : `checked ${age}`} padded title="Health">
            <div className={styles.steps}>
                <Step label="Its state">
                    <StatePill {...vmState(vm)} />
                </Step>
                <Step label="Its address">
                    {vm.IP === '' ? (
                        <span className={styles.quiet}>—</span>
                    ) : (
                        <Value>{vm.IP}</Value>
                    )}
                </Step>
                <Step label="SSH">
                    <StatePill {...vmSsh(vm)} />
                </Step>
                <Step label="Its check">
                    <StatePill {...vmCheck(vm)} />
                </Step>
            </div>
        </Panel>
    );
}

export function CheckTile({ vm }: { vm: MachinesVM }) {
    const last = (vm.Latency ?? []).at(-1);
    return (
        <Tile
            foot={ageOf(vm.CheckStatus.at) ?? ''}
            footState={vmCheck(vm)}
            label="Check"
            unit={last ? 'ms' : ''}
            value={last ? millis(last.took) : <MissingMark />}
        />
    );
}

const NS_PER_HOUR = 3600 * NS_PER_SECOND;

function spanOf(points: readonly MachinesLatencyPoint[]): number | undefined {
    const [first, second] = points;
    const kept = between(first?.at, points.at(-1)?.at);
    const step = between(first?.at, second?.at);
    return kept === undefined || step === undefined ? undefined : kept + step;
}

function lastNote(span: number): string {
    return span >= NS_PER_HOUR && span < 1.5 * NS_PER_HOUR ? 'last hour' : `last ${seconds(span)}`;
}

export function CheckLatency({ vm }: { vm: MachinesVM }) {
    const points = vm.Latency ?? [];
    const last = points.at(-1);
    const tallest = Math.max(0, ...points.map((point) => point.took));
    const span = spanOf(points);
    return (
        <Panel
            action={
                last ? (
                    <span className={styles.meta}>{`${millis(last.took)} ms now`}</span>
                ) : undefined
            }
            note={span === undefined ? '' : lastNote(span)}
            padded
            title="Check latency"
        >
            {span !== undefined && points.length > 1 ? (
                <div>
                    <Chart
                        accent="last"
                        label="Latency of the check"
                        marks={points.map(
                            (point) =>
                                `${clock(point.at) ?? point.at}${SEPARATOR}${millis(point.took)} ms`,
                        )}
                        size="activity"
                        tone="new"
                        values={points.map((point) =>
                            tallest > 0 ? (point.took / tallest) * 100 : 0,
                        )}
                    />
                    <div className={styles.axis}>
                        <span>{`${seconds(span)} ago`}</span>
                        <span>Now</span>
                    </div>
                </div>
            ) : (
                <p className={styles.none}>
                    {`The engine kept ${points.length === 1 ? 'one check' : 'no check'} so far.`}
                </p>
            )}
        </Panel>
    );
}

function dayOf(day: MachinesUptimeDay): string {
    const restarts = day.restarts ?? 0;
    const down = day.down_s ?? 0;
    return sentence([
        shortDay(day.date) ?? day.date,
        restarts > 0 ? `${String(restarts)} ${restarts === 1 ? 'restart' : 'restarts'}` : undefined,
        down > 0 ? `down ${seconds(down * NS_PER_SECOND)}` : undefined,
    ]);
}

function daysOf(count: number): Measured {
    return { '--days': count };
}

// invariant: the record keeps a day only once the server went down or restarted on it, so a cell is such a day,
// invariant: never a day of a calendar: the days between two cells are not drawn.
export function LastDays({ vm }: { vm: MachinesVM }) {
    const days = vm.Uptime.days ?? [];
    const first = days[0];
    const last = days.at(-1);
    return (
        <Panel padded title="Days it went down">
            {first && last ? (
                <div>
                    <div
                        aria-label="Its days"
                        className={styles.strip}
                        role="img"
                        style={daysOf(days.length)}
                    >
                        {days.map((day) => (
                            <i key={day.date} title={dayOf(day)} />
                        ))}
                    </div>
                    <div className={styles.axis}>
                        <span>{shortDay(first.date)}</span>
                        <span>{shortDay(last.date)}</span>
                    </div>
                </div>
            ) : (
                <p className={styles.none}>The engine kept no such day.</p>
            )}
        </Panel>
    );
}
