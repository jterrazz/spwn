import type { ActivityReport } from '@jterrazz/station-api';
import { useState } from 'react';

import { Bar, identityOf, MachineIcon, Panel, Row, Switch } from '../../design/index';
import { machineKindOf } from '../../engine/provisional';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { busiestWithVms, vmOf } from './measures';
import type { Busy } from './measures';

import styles from './activity.module.css';

type By = 'cpu' | 'memory';

export function BusyShare({
    label,
    share,
    shown,
}: {
    label: string;
    share: number;
    shown: string;
}) {
    return (
        <span className={styles.busy}>
            <Bar label={label} percent={share} width="fixed" />
            <span className={styles.share}>{shown}</span>
        </span>
    );
}

function Process({
    row,
    by,
    leads,
    found,
    go,
}: {
    row: Busy;
    by: By;
    leads: boolean;
    found: Found;
    go: (to: Route) => void;
}) {
    const vm = row.vm === undefined ? undefined : vmOf(found.mac, row.vm);
    const name = row.vm ?? row.name;
    const measured = (
        <BusyShare
            label={`${by === 'cpu' ? 'CPU' : 'Memory'} of ${name}`}
            share={row.share}
            shown={row.shown}
        />
    );
    if (row.vm === undefined) {
        return (
            <div className={styles.process}>
                <Row
                    lead={leads ? null : undefined}
                    leadSize="machine"
                    size="slim"
                    title={name}
                    titleLook="mono"
                    value={measured}
                />
            </div>
        );
    }
    return (
        <Row
            lead={
                vm ? <MachineIcon identity={identityOf(vm.Name)} kind={machineKindOf(vm)} /> : null
            }
            leadSize="machine"
            onOpen={() => {
                go(machineRoute(name, 'Activity'));
            }}
            sentence="a machine of this Mac"
            size="slim"
            title={name}
            value={measured}
        />
    );
}

export function Busiest({
    report,
    found,
    go,
}: {
    report: ActivityReport;
    found: Found;
    go: (to: Route) => void;
}) {
    const [by, setBy] = useState<By>('cpu');
    const rows = busiestWithVms(report.last, by);
    const leads = rows.some((row) => row.vm !== undefined);
    return (
        <Panel
            action={
                <Switch
                    label="Busiest by"
                    onChange={setBy}
                    options={[
                        { id: 'cpu', label: 'By CPU' },
                        { id: 'memory', label: 'By memory' },
                    ]}
                    surface="panel"
                    value={by}
                />
            }
            title="Busiest"
        >
            {rows.length === 0 ? (
                <p className={styles.none}>The engine listed no process.</p>
            ) : null}
            {rows.map((row) => (
                <Process by={by} found={found} go={go} key={row.key} leads={leads} row={row} />
            ))}
        </Panel>
    );
}
