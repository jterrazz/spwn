import type { MachinefilePlan } from '@jterrazz/station-api';
import { useState } from 'react';

import { AppIcon, Button, Panel, StatePill, Switch, Value } from '../../design/index';
import { differingLines } from '../../engine/provisional';
import type { PlanLine } from '../../engine/provisional';
import { ACTION_TONES, CONDITION_TONES } from '../../engine/tones';
import { ACTION_WORDS, CONDITION_WORDS } from '../../engine/words';

import styles from './packages.module.css';

type Direction = 'file' | 'machine';

const DIRECTIONS: readonly { id: Direction; label: string }[] = [
    { id: 'machine', label: 'Change the machine' },
    { id: 'file', label: 'Change the file' },
];

function DiffRow({ line, toFile }: { line: PlanLine; toFile: boolean }) {
    const { Step, Seen } = line;
    return (
        <div className={styles.compare}>
            <div className={styles.item}>
                <AppIcon kind={Step.Kind} name={Step.Package} />
                <span className={styles.name}>{Step.Package}</span>
                {Step.Version === '' ? null : <Value>{Step.Version}</Value>}
                {Step.Origin.Name === '' ? null : <Value>{Step.Origin.Name}</Value>}
            </div>
            <div className={styles.state}>
                <StatePill
                    tone={CONDITION_TONES[Seen.Condition]}
                    word={CONDITION_WORDS[Seen.Condition]}
                />
                <span className={styles.muted}>
                    {Seen.Version === '' ? Seen.Detail : Seen.Version}
                </span>
            </div>
            <div className={styles.will}>
                {toFile ? (
                    <span className={styles.muted}>not given</span>
                ) : (
                    <StatePill tone={ACTION_TONES[line.Action]} word={ACTION_WORDS[line.Action]} />
                )}
            </div>
        </div>
    );
}

function DiffFoot({ toFile, onPlan }: { toFile: boolean; onPlan: () => void }) {
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
            <span>The plan says each step before anything is applied.</span>
            <Button onClick={onPlan} variant="primary">
                Plan
            </Button>
        </div>
    );
}

export function Differs({ plan, onPlan }: { plan: MachinefilePlan; onPlan: () => void }) {
    const [direction, setDirection] = useState<Direction>('machine');
    const lines = differingLines(plan);
    if (lines.length === 0) {
        return null;
    }
    const toFile = direction === 'file';
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
            title="Differs from its file"
        >
            <div className={`${styles.compare} ${styles.head}`}>
                <span>The file says</span>
                <span>This machine has</span>
                <span className={styles.will}>{toFile ? 'The file will' : 'The machine will'}</span>
            </div>
            {lines.map((line) => (
                <DiffRow
                    key={`${line.Step.Kind}/${line.Step.Package}`}
                    line={line}
                    toFile={toFile}
                />
            ))}
            <DiffFoot onPlan={onPlan} toFile={toFile} />
        </Panel>
    );
}
