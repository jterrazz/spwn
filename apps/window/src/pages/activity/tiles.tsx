import type { ActivityReport } from '@jterrazz/station-api';
import type { ReactElement, ReactNode } from 'react';

import { Bar, Chart, identityOf, MissingMark, Tile } from '../../design/index';
import type { Identity } from '../../design/index';
import { ownState } from '../../engine/shown';
import type { OwnState } from '../../engine/words';
import { bytes, duration, memory, percent } from '../../model/format';
import type { Found } from '../../model/machines';
import type { Tone } from '../../tone';
import { Tiles } from '../page';
import { chartedOf, cpuOf, diskOf, marksOf, memoryOf } from './measures';
import type { Held } from './measures';
import { useActivity } from './reading';
import type { Heard } from './reading';
import { agoOf, firstUnitOf, shortDay } from './when';

const LAST_BARS = 12;

export function split(text: string): { value: string; unit: string } {
    const space = text.indexOf(' ');
    return space === -1
        ? { unit: '', value: text }
        : { unit: text.slice(space + 1), value: text.slice(0, space) };
}

export function Unknown({ label, state }: { label: string; state: OwnState }) {
    return <Tile foot="" label={label} value={<MissingMark>{ownState(state).word}</MissingMark>} />;
}

export function NotGivenTile({ label }: { label: string }) {
    return <Tile foot="" label={label} value={<MissingMark />} />;
}

// invariant: `chart.peak` is the index of the bucket holding the busiest point, -1 when none holds one
// (apps/engine/activity/chart.go); the figure drawn is that bucket's `max`.
export function cpuFootOf(report: ActivityReport): string {
    const { chart } = report;
    const { cpus, at } = report.last.sample;
    const peak = chart === undefined || chart.peak < 0 ? undefined : chart.buckets?.[chart.peak];
    if (chart !== undefined && peak !== undefined) {
        const ago = agoOf(chart.peak_at, at);
        return `Peak ${percent(peak.max)} %${ago === undefined ? '' : ` · ${ago}`}`;
    }
    return cpus > 0 ? `${String(cpus)} CPUs` : '';
}

export function CpuTile({
    report,
    name,
    label,
}: {
    report: ActivityReport;
    name: string;
    label: string;
}) {
    const cpu = cpuOf(report);
    if (cpu === undefined) {
        return <NotGivenTile label={label} />;
    }
    const buckets = report.chart?.buckets ?? [];
    return (
        <Tile
            foot={cpuFootOf(report)}
            label={label}
            unit="%"
            value={cpu}
            viz={
                chartedOf(buckets) ? (
                    <Chart
                        label={`CPU of ${name}, its last bars`}
                        values={marksOf(buckets).values.slice(-LAST_BARS)}
                    />
                ) : undefined
            }
        />
    );
}

function HeldTile({
    label,
    held,
    shown,
    identity,
    tone = 'plain',
}: {
    label: string;
    held: Held | undefined;
    shown: (value: number) => string;
    identity?: Identity | undefined;
    tone?: Tone;
}) {
    if (held === undefined) {
        return <NotGivenTile label={label} />;
    }
    const { value, unit } = split(shown(held.used));
    if (held.total === undefined) {
        return <Tile foot="its process" label={label} unit={unit} value={value} />;
    }
    return (
        <Tile
            foot={`of ${shown(held.total)}`}
            label={label}
            unit={unit}
            value={value}
            viz={
                <Bar
                    identity={identity}
                    label={`${label} used of its total`}
                    percent={(held.used / held.total) * 100}
                    tone={identity === undefined ? tone : 'plain'}
                    width="fixed"
                />
            }
        />
    );
}

function UptimeTile({
    up,
    report,
    label,
}: {
    up: number;
    report: ActivityReport | undefined;
    label: string;
}) {
    if (up <= 0) {
        return <NotGivenTile label={label} />;
    }
    const { value, unit } = firstUnitOf(duration(up));
    const since = shortDay(report?.last.sample.boot);
    return (
        <Tile
            foot={since === undefined ? '' : `since ${since}`}
            label={label}
            unit={unit}
            value={value}
        />
    );
}

export function unknownOf(shown: Heard | undefined): OwnState | undefined {
    if (shown === undefined) {
        return 'reading';
    }
    if (shown.kind === 'failed') {
        return 'no-answer';
    }
    return shown.out.silent === true ? 'not-measured' : undefined;
}

type Labels = { cpu: string; memory: string; disk: string; up: string };

function labelsOf(of: string | undefined): Labels {
    const named = (label: string) => (of === undefined ? label : `${label}, ${of}`);
    return { cpu: named('CPU'), disk: named('Disk'), memory: named('Memory'), up: named('Uptime') };
}

// value: a disk the meter does not measure is drawn by the tile the page hands, its own read of the disk.
function DiskHeld({
    report,
    label,
    disk,
}: {
    report: ActivityReport;
    label: string;
    disk: ReactElement | undefined;
}) {
    const held = diskOf(report.last.sample);
    if (held === undefined && disk !== undefined) {
        return disk;
    }
    return <HeldTile held={held} label={label} shown={bytes} />;
}

type TilesProps = {
    found: Found;
    up: number;
    of?: string;
    fourth?: ReactNode;
    disk?: ReactElement | undefined;
};

export function ActivityTiles({ found, up, of, fourth, disk }: TilesProps) {
    const name = found.vm?.Name ?? found.mac.Name;
    const labels = labelsOf(of);
    const { readers, shown } = useActivity(found, '1h');
    const unknown = unknownOf(shown);
    const report = shown?.kind === 'read' ? shown.out : undefined;
    return (
        <>
            {readers}
            <Tiles>
                {unknown !== undefined || report === undefined ? (
                    <>
                        <Unknown label={labels.cpu} state={unknown ?? 'reading'} />
                        <Unknown label={labels.memory} state={unknown ?? 'reading'} />
                        <Unknown label={labels.disk} state={unknown ?? 'reading'} />
                    </>
                ) : (
                    <>
                        <CpuTile label={labels.cpu} name={name} report={report} />
                        <HeldTile
                            held={memoryOf(report)}
                            identity={found.vm ? identityOf(found.vm.Name) : undefined}
                            label={labels.memory}
                            shown={memory}
                            tone="new"
                        />
                        <DiskHeld disk={disk} label={labels.disk} report={report} />
                    </>
                )}
                {fourth ?? <UptimeTile label={labels.up} report={report} up={up} />}
            </Tiles>
        </>
    );
}
