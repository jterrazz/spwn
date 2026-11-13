import type { ActivityReport, MachinesMacView, MachinesVM } from '@jterrazz/station-api';

import { Bar, Chart, identityOf, MachineIcon, Panel, Row, RowHead } from '../../design/index';
import type { Identity } from '../../design/index';
import { machineKindOf } from '../../engine/provisional';
import { memory, percent } from '../../model/format';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { chartedOf, cpuOf, marksOf } from '../activity/measures';
import { useActivity } from '../activity/reading';
import { split } from '../activity/tiles';

import styles from './home.module.css';

type Go = (to: Route) => void;

const SPARK_BARS = 8;

const COLUMNS = [
    { label: 'Machine' },
    { label: 'What it is for' },
    { label: 'CPU', numeric: true },
    { label: 'Memory, GB', numeric: true },
] as const;

function gb(value: number): string {
    return split(memory(value)).value;
}

function Cpu({
    value,
    spark,
    name,
}: {
    value: string;
    spark?: readonly (null | number)[] | undefined;
    name: string;
}) {
    return (
        <span className={styles.meter}>
            {spark === undefined ? null : (
                <Chart label={`CPU of ${name}, its last bars`} size="row" values={spark} />
            )}
            <span className={styles.figure}>{`${value} %`}</span>
        </span>
    );
}

function Held({
    used,
    total,
    name,
    identity,
}: {
    used: number;
    total: number;
    name: string;
    identity?: Identity | undefined;
}) {
    return (
        <span className={styles.meter}>
            <Bar
                identity={identity}
                label={`Memory of ${name}, used of its total`}
                percent={(used / total) * 100}
            />
            <span className={styles.figure}>{`${gb(used)} / ${gb(total)}`}</span>
        </span>
    );
}

function macCells(name: string, report: ActivityReport | undefined) {
    if (report === undefined) {
        return [null, null];
    }
    const cpu = cpuOf(report);
    const buckets = report.chart?.buckets ?? [];
    const spark = chartedOf(buckets) ? marksOf(buckets).values.slice(-SPARK_BARS) : undefined;
    const { mem_used: used, mem_total: total } = report.last.sample;
    return [
        cpu === undefined ? null : <Cpu key="cpu" name={name} spark={spark} value={cpu} />,
        total > 0 ? <Held key="memory" name={name} total={total} used={used} /> : null,
    ];
}

export function vmCells(
    vm: Pick<MachinesVM, 'Name' | 'Purpose' | 'Resources'>,
    report: ActivityReport | undefined,
) {
    const purpose = vm.Purpose === '' ? null : vm.Purpose;
    const usage = report?.last.vms?.find((one) => one.vm === vm.Name);
    if (usage === undefined) {
        return [purpose, null, null];
    }
    const total = vm.Resources.MemoryBytes;
    return [
        purpose,
        <Cpu key="cpu" name={vm.Name} value={percent(usage.cpu)} />,
        total > 0 ? (
            <Held
                identity={identityOf(vm.Name)}
                key="memory"
                name={vm.Name}
                total={total}
                used={usage.memory}
            />
        ) : null,
    ];
}

function MacRows({ mac, go }: { mac: MachinesMacView; go: Go }) {
    const { readers, shown } = useActivity({ mac }, '1h');
    const report = shown?.kind === 'read' && shown.out.silent !== true ? shown.out : undefined;
    return (
        <>
            {readers}
            <Row
                cells={[null, ...macCells(mac.Name, report)]}
                lead={<MachineIcon kind="mac" />}
                leadSize="machine"
                onOpen={() => {
                    go(machineRoute(mac.Name));
                }}
                template="fleet"
                title={mac.Name}
            />
            {(mac.VMs ?? []).map((vm) => (
                <Row
                    cells={vmCells(vm, report)}
                    key={vm.Name}
                    lead={<MachineIcon identity={identityOf(vm.Name)} kind={machineKindOf(vm)} />}
                    leadSize="machine"
                    nested
                    onOpen={() => {
                        go(machineRoute(vm.Name));
                    }}
                    template="fleet"
                    title={vm.Name}
                />
            ))}
        </>
    );
}

export function HomeMachines({ macs, go }: { macs: readonly MachinesMacView[]; go: Go }) {
    return (
        <Panel title="Machines">
            <RowHead columns={COLUMNS} lead template="fleet" />
            {macs.map((mac) => (
                <MacRows go={go} key={mac.Name} mac={mac} />
            ))}
        </Panel>
    );
}
