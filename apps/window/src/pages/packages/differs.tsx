import type { MachinefileDiffering, MachinefileDrift } from '@jterrazz/station-api';
import { useState } from 'react';

import { AppIcon, Button, Panel, StatePill, Switch, Value } from '../../design/index';
import type { Identity } from '../../design/index';
import { layerOf } from '../../engine/provisional';
import type { PlanLine } from '../../engine/provisional';
import { engineShown } from '../../engine/shown';
import { differencesOf, layersOf } from './model';

import styles from './packages.module.css';

type Direction = 'file' | 'machine';

const DIRECTIONS: readonly { id: Direction; label: string }[] = [
    { id: 'machine', label: 'Change the machine' },
    { id: 'file', label: 'Change the file' },
];

type Layered = { name: string; identity: Identity | undefined };

function DiffRow({ line, layer, toFile }: { line: PlanLine; layer: Layered; toFile: boolean }) {
    const { Step, Seen } = line;
    return (
        <div className={styles.compare}>
            <div className={styles.item}>
                <AppIcon kind={Step.Kind} name={Step.Package} />
                <span className={styles.name}>{Step.Package}</span>
                {Step.Version === '' ? null : <Value look="raised">{Step.Version}</Value>}
                {layer.name === '' ? null : (
                    <Value identity={layer.identity} look="raised">
                        {layer.name}
                    </Value>
                )}
            </div>
            <div className={styles.state}>
                <StatePill {...engineShown('MachinefileCondition', Seen.Condition)} />
                {Seen.Version === '' ? (
                    <span className={styles.muted}>{Seen.Detail}</span>
                ) : (
                    <Value look="raised">{Seen.Version}</Value>
                )}
            </div>
            <div className={styles.will}>
                {toFile ? (
                    <span className={styles.muted}>not given</span>
                ) : (
                    <StatePill dotted={false} {...engineShown('MachinefileAction', line.Action)} />
                )}
            </div>
        </div>
    );
}

function BlockRow({ differing }: { differing: MachinefileDiffering }) {
    return (
        <div className={styles.compare}>
            <div className={styles.item}>
                <AppIcon name={differing.Package} />
                <span className={styles.name}>{differing.Package}</span>
                <Value look="raised">{differing.Block}</Value>
            </div>
            <div className={styles.state}>
                <StatePill {...engineShown('MachinefileCondition', differing.Condition)} />
            </div>
            <div className={styles.will} />
        </div>
    );
}

function DiffFoot({
    toFile,
    onPlan,
    apply,
}: {
    toFile: boolean;
    onPlan: () => void;
    apply: string;
}) {
    if (toFile) {
        return (
            <div className={styles.foot}>
                <span>The engine has no capture yet: what the file would say is not given.</span>
                <Button disabled variant="primary">
                    Capture
                </Button>
            </div>
        );
    }
    return (
        <div className={styles.foot}>
            <span>{apply}</span>
            <Button onClick={onPlan} variant="primary">
                Plan
            </Button>
        </div>
    );
}

export function Differs({ drift, onPlan }: { drift: MachinefileDrift; onPlan: () => void }) {
    const [direction, setDirection] = useState<Direction>('machine');
    const differences = differencesOf(drift);
    if (differences.length === 0) {
        return null;
    }
    const toFile = direction === 'file';
    const plan = drift.Plan;
    const all = [...(plan.Lines ?? []), ...(plan.Extra ?? [])];
    const identities = new Map(layersOf(plan).map((layer) => [layer.name, layer.identity]));
    return (
        <Panel
            action={
                <Switch
                    label="Which side changes"
                    onChange={setDirection}
                    options={DIRECTIONS}
                    surface="panel"
                    value={direction}
                />
            }
            note={String(differences.length)}
            title="Differs from its file"
        >
            <div className={`${styles.compare} ${styles.head}`}>
                <span>The file says</span>
                <span>This machine has</span>
                <span className={styles.will}>{toFile ? 'The file will' : 'The machine will'}</span>
            </div>
            {differences.map(({ differing, line }) => {
                if (line === undefined) {
                    return (
                        <BlockRow
                            differing={differing}
                            key={`${differing.Package}/${differing.Block}`}
                        />
                    );
                }
                const name = layerOf(line, all);
                return (
                    <DiffRow
                        key={`${line.Step.Kind}/${line.Step.Package}`}
                        layer={{ identity: identities.get(name), name }}
                        line={line}
                        toFile={toFile}
                    />
                );
            })}
            <DiffFoot apply={drift.Apply.Sentence} onPlan={onPlan} toFile={toFile} />
        </Panel>
    );
}
