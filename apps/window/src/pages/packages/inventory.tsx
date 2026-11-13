import type { MachinefileDrift, MachinefilePlan, Target } from '@jterrazz/station-api';
import { useState } from 'react';

import { Panel, Switch } from '../../design/index';
import { differingLines } from '../../engine/provisional';
import { useWatch } from '../../engine/react';
import type { Reading as Read } from '../../engine/react';
import { Unanswered } from '../kit/readings';
import { LayerList } from './layers';
import { categoriesOf, countOn, layersOf, onShelf, SHELVES } from './model';
import type { Shelf } from './model';

import styles from './packages.module.css';

const BY_LAYER = 'by layer, a later one wins';

function Lines({
    plan,
    shelf,
    target,
    setShelf,
}: {
    plan: MachinefilePlan;
    shelf: Shelf;
    target: Target | undefined;
    setShelf: (shelf: Shelf) => void;
}) {
    const apps = useWatch('apps.list', {}, target);
    const categories = categoriesOf(apps.kind === 'read' ? (apps.out ?? []) : []);
    const layers = layersOf(plan);
    return (
        <>
            <Switch
                label="Which packages"
                onChange={setShelf}
                options={SHELVES.map((id) => ({
                    count: countOn(layers, id, categories),
                    id,
                    label: id,
                })).filter((option) => option.id !== 'Skills' || option.count > 0)}
                value={shelf}
            />
            <Panel note={BY_LAYER} title={shelf}>
                <LayerList
                    layers={onShelf(layers, shelf, categories)}
                    look={{ differing: new Set(differingLines(plan)), kind: 'machine' }}
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
        return <Lines plan={drift.out.Plan} setShelf={setShelf} shelf={shelf} target={target} />;
    }
    return (
        <Panel note={BY_LAYER} title={shelf}>
            {drift.kind === 'failed' ? (
                <Unanswered error={drift.error} size="panel" />
            ) : (
                <p className={styles.text}>Reading its machine file…</p>
            )}
        </Panel>
    );
}
