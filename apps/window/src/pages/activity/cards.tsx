import type { ActivityReport, ActivitySample } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { Chart, Missing, MissingMark, Panel } from '../../design/index';
import { PRESSURE_WORDS } from '../../engine/words';
import { memory, rate, sentence } from '../../model/format';
import { countedMarks, keptOver, memoryMarks, strided } from './history';
import { chartedOf, cpuOf, marksOf, memoryOf, spanOf } from './measures';
import type { Marks, Span } from './measures';
import { split } from './tiles';

import styles from './activity.module.css';

type CardProps = {
    report: ActivityReport;
    span: Span;
    name: string;
    history: readonly ActivitySample[] | undefined;
};

const MOST_BARS = 72;

function Figure({ value, unit }: { value: string | undefined; unit: string }) {
    return (
        <div className={styles.figure}>
            {value ?? <MissingMark />}
            <small className={styles.unit}>{unit}</small>
        </div>
    );
}

function ratedOf(report: ActivityReport): boolean {
    return report.window > 0;
}

function Card({ title, meta, children }: { title: string; meta: string; children: ReactNode }) {
    return (
        <Panel
            action={meta === '' ? undefined : <span className={styles.meta}>{meta}</span>}
            title={title}
        >
            <div className={styles.card}>{children}</div>
        </Panel>
    );
}

function Drawn({ marks, label, span }: { marks: Marks; label: string; span: Span }) {
    return (
        <>
            <Chart
                label={`${label} over ${spanOf(span).label}`}
                marks={marks.labels}
                size="activity"
                tone="new"
                values={marks.values}
            />
            <div className={styles.axis}>
                <span>{`${spanOf(span).label} ago`}</span>
                <span>now</span>
            </div>
        </>
    );
}

function CpuHistory({ report, span, name }: CardProps) {
    const buckets = report.chart?.buckets ?? [];
    if (buckets.length === 0) {
        return <Missing>No chart: the engine keeps no history of it over this window.</Missing>;
    }
    if (!chartedOf(buckets)) {
        return <Missing>No chart: no bar of this window holds a sample of the engine.</Missing>;
    }
    return <Drawn label={`CPU of ${name}`} marks={marksOf(buckets)} span={span} />;
}

function Kept({
    history,
    span,
    label,
    marksOf: drawn,
}: {
    history: readonly ActivitySample[] | undefined;
    span: Span;
    label: string;
    marksOf: (samples: readonly ActivitySample[]) => Marks;
}) {
    const kept = keptOver(history ?? [], spanOf(span).ns);
    const marks = drawn(kept);
    if (history === undefined || marks.values.length < 2) {
        return (
            <Missing>
                No chart: the engine kept fewer than two samples of it over this window.
            </Missing>
        );
    }
    return <Drawn label={label} marks={marks} span={span} />;
}

function Cpu(props: CardProps) {
    const { report } = props;
    const { sample } = report.last;
    const figure = cpuOf(report);
    return (
        <Card
            meta={sentence([
                sample.cpus > 0 ? `${String(sample.cpus)} CPUs` : undefined,
                sample.load.length > 0 ? `load ${sample.load.join(' ')}` : undefined,
            ])}
            title="CPU"
        >
            <Figure unit={figure === undefined ? '' : '%'} value={figure} />
            <CpuHistory {...props} />
        </Card>
    );
}

function MemoryFigure({ report }: { report: ActivityReport }) {
    const held = memoryOf(report);
    if (held === undefined) {
        return <Figure unit="" value={undefined} />;
    }
    const { value, unit } = split(memory(held.used));
    if (held.total === undefined) {
        return <Figure unit={`${unit} · its process`} value={value} />;
    }
    return <Figure unit={`${unit} of ${memory(held.total)}`} value={value} />;
}

function Memory({ report, name, span, history }: CardProps) {
    const { sample } = report.last;
    return (
        <Card
            meta={sentence([
                PRESSURE_WORDS[sample.pressure ?? ''],
                sample.mem_total > 0 ? `swap ${memory(sample.swap_used)}` : undefined,
            ])}
            title="Memory"
        >
            <MemoryFigure report={report} />
            <Kept
                history={history}
                label={`Memory of ${name}`}
                marksOf={(kept) => memoryMarks(strided(kept, MOST_BARS))}
                span={span}
            />
        </Card>
    );
}

type FlowProps = CardProps & {
    title: string;
    rates: readonly [number, number];
    words: readonly [string, string];
    counted: (sample: ActivitySample) => number;
};

function Flow({ report, name, span, history, title, rates, words, counted }: FlowProps) {
    const rated = ratedOf(report);
    const lead = rate(rates[0]);
    const other = rate(rates[1]);
    return (
        <Card meta={rated ? `${other.value} ${other.unit} ${words[1]}` : ''} title={title}>
            <Figure
                unit={rated ? `${lead.unit} ${words[0]}` : ''}
                value={rated ? lead.value : undefined}
            />
            <Kept
                history={history}
                label={`${title} ${words[0]} of ${name}, per sample`}
                marksOf={(kept) => countedMarks(strided(kept, MOST_BARS + 1), counted)}
                span={span}
            />
        </Card>
    );
}

export function Cards(props: CardProps) {
    const { report } = props;
    return (
        <div className={styles.cards}>
            <Cpu {...props} />
            <Memory {...props} />
            <Flow
                {...props}
                counted={(sample) => sample.disk_read}
                rates={[report.rates.DiskRead, report.rates.DiskWrite]}
                title="Disk"
                words={['read', 'written']}
            />
            <Flow
                {...props}
                counted={(sample) => sample.net_in}
                rates={[report.rates.NetIn, report.rates.NetOut]}
                title="Network"
                words={['in', 'out']}
            />
        </div>
    );
}
