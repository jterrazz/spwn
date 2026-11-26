import type { MachinefileDiffering, MachinefileDrift } from '@jterrazz/station-api';
import { useState } from 'react';

import { differencesOf, planFilesOf } from '../../../features/machinefile/model';
import { AppIcon, Button, Panel, StateMark, Switch, Value } from '../../../ui/index';
import type { Identity } from '../../../ui/index';
import type { PlanLine } from '../../../words/provisional';
import { engineShown } from '../../../words/shown';

import styles from './packages.module.css';

type Direction = 'file' | 'machine';

const DIRECTIONS: readonly { id: Direction; label: string }[] = [
    { id: 'machine', label: 'Change the machine' },
    { id: 'file', label: 'Change the file' },
];

type PlanFileTag = { name: string; identity: Identity | undefined };

function DiffRow({
    line,
    planFile,
    toFile,
}: {
    line: PlanLine;
    planFile: PlanFileTag;
    toFile: boolean;
}) {
    const { Step, Seen } = line;
    return (
        <div className={styles.compare}>
            <div className={styles.item}>
                <AppIcon kind={Step.Kind} name={Step.Package} />
                <span className={styles.name}>{Step.Package}</span>
                {Step.Version === '' ? null : <Value look="raised">{Step.Version}</Value>}
                {planFile.name === '' ? null : (
                    <Value identity={planFile.identity} look="raised">
                        {planFile.name}
                    </Value>
                )}
            </div>
            <div className={styles.state}>
                <StateMark {...engineShown('MachinefileCondition', Seen.Condition)} />
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
                    <span className={styles.action}>
                        {engineShown('MachinefileAction', line.Action).word}
                    </span>
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
                <StateMark {...engineShown('MachinefileCondition', differing.Condition)} />
            </div>
            <div className={styles.will} />
        </div>
    );
}

function DiffFoot({
    toFile,
    onPlan,
    apply,
    left,
}: {
    toFile: boolean;
    onPlan: () => void;
    apply: string;
    left: string;
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
            <span className={styles.said}>
                <span>{apply}</span>
                {left === '' ? null : <span className={styles.muted}>{left}</span>}
            </span>
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
    const identities = new Map(
        planFilesOf(drift.Plan).map((planFile) => [planFile.name, planFile.identity]),
    );
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
                const name = line.Step.File.Name;
                return (
                    <DiffRow
                        key={`${line.Step.Kind}/${line.Step.Package}`}
                        line={line}
                        planFile={{ identity: identities.get(name), name }}
                        toFile={toFile}
                    />
                );
            })}
            <DiffFoot
                apply={drift.Apply.Sentence}
                left={drift.Apply.Left}
                onPlan={onPlan}
                toFile={toFile}
            />
        </Panel>
    );
}
