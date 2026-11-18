import type { MachinefileDrift, Target } from '@jterrazz/station-api';
import { useState } from 'react';

import { useWatch } from '../../../dependencies/dependencies';
import type { Reading as Read } from '../../../dependencies/dependencies';
import { PlanFileList } from '../../../features/machinefile/files';
import {
    countOn,
    differingLinesOf,
    kindsOf,
    onShelf,
    planFilesOf,
    SHELVES,
} from '../../../features/machinefile/model';
import type { Shelf } from '../../../features/machinefile/model';
import { Unanswered } from '../../../kit/readings';
import { Panel, Switch } from '../../../ui/index';

import styles from './packages.module.css';

const BY_FILE = 'by file, a later one wins';

function Lines({
    drift,
    shelf,
    target,
    setShelf,
}: {
    drift: MachinefileDrift;
    shelf: Shelf;
    target: Target | undefined;
    setShelf: (shelf: Shelf) => void;
}) {
    const apps = useWatch('apps.list', {}, target);
    const kinds = kindsOf(apps.kind === 'read' ? (apps.out ?? []) : []);
    const planFiles = planFilesOf(drift.Plan);
    return (
        <>
            <Switch
                label="Which packages"
                onChange={setShelf}
                options={SHELVES.map((id) => ({
                    count: countOn(planFiles, id, kinds),
                    id,
                    label: id,
                })).filter((option) => option.id !== 'Skills' || option.count > 0)}
                value={shelf}
            />
            <Panel note={BY_FILE} title={shelf}>
                <PlanFileList
                    look={{ differing: differingLinesOf(drift), kind: 'machine' }}
                    planFiles={onShelf(planFiles, shelf, kinds)}
                    shelf={shelf}
                />
            </Panel>
        </>
    );
}

export function Inventory({
    drift,
    target,
}: {
    drift: Read<MachinefileDrift>;
    target: Target | undefined;
}) {
    const [shelf, setShelf] = useState<Shelf>('Apps');
    if (drift.kind === 'read') {
        return <Lines drift={drift.out} setShelf={setShelf} shelf={shelf} target={target} />;
    }
    return (
        <Panel note={BY_FILE} title={shelf}>
            {drift.kind === 'failed' ? (
                <Unanswered error={drift.error} size="panel" />
            ) : (
                <p className={styles.text}>Reading its machine file…</p>
            )}
        </Panel>
    );
}
