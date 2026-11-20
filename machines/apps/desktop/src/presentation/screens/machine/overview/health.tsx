import type { MachinesLatencyPoint, MachinesUptimeCell, MachinesVM } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { clock, duration, seconds } from '../../../../domain/format';
import { SEPARATOR } from '../../../../domain/separator';
import { between, shortDay } from '../../../../domain/when';
import type { Route } from '../../../navigation/route';
import { Chart, MissingMark, Panel, StatePill, Tile, Value } from '../../../ui/index';
import type { Measured } from '../../../ui/index';
import { dayOpensHistory, verdictShown, vmCheck, vmSsh, vmState } from '../../../words/shown';
import { UPTIME_MARK_TONES } from '../../../words/tones';

import styles from './overview.module.css';

type Go = (to: Route) => void;

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
            {vm.Health === undefined ? null : (
                <div className={styles.healthHead}>
                    <StatePill {...verdictShown(vm.Health)} />
                    {vm.Health.Line === undefined || vm.Health.Line === '' ? null : (
                        <span className={styles.healthLine}>{vm.Health.Line}</span>
                    )}
                </div>
            )}
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
            ) : vm.LatencyNote === undefined ? null : (
                <p className={styles.none}>{vm.LatencyNote}</p>
            )}
        </Panel>
    );
}

function daysOf(count: number): Measured {
    return { '--days': count };
}

function Day({ cell, go }: { cell: MachinesUptimeCell; go: Go }) {
    const tone = styles[UPTIME_MARK_TONES[cell.Mark]];
    const named =
        cell.Sentence === undefined ? {} : { 'aria-label': cell.Sentence, title: cell.Sentence };
    if (dayOpensHistory(cell)) {
        return (
            <button
                className={tone}
                onClick={() => {
                    go({ page: 'inbox' });
                }}
                type="button"
                {...named}
            />
        );
    }
    return <i className={tone} role={cell.Sentence === undefined ? undefined : 'img'} {...named} />;
}

export function UptimeStrip({ vm, go }: { vm: MachinesVM; go: Go }) {
    const days = vm.Strip?.Days ?? [];
    const first = days[0];
    const last = days.at(-1);
    return (
        <Panel note={vm.Strip?.Sentence ?? ''} padded title="Last 30 days">
            {first && last ? (
                <div>
                    <div
                        aria-label="Its days"
                        className={styles.strip}
                        role="group"
                        style={daysOf(days.length)}
                    >
                        {days.map((cell) => (
                            <Day cell={cell} go={go} key={cell.Date} />
                        ))}
                    </div>
                    <div className={styles.axis}>
                        <span>{shortDay(first.Date)}</span>
                        <span>{shortDay(last.Date)}</span>
                    </div>
                </div>
            ) : (
                <p className={styles.none}>The engine kept no day yet.</p>
            )}
        </Panel>
    );
}
