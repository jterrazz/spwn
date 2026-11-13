import type { CallName, MachinefilePlan, Target } from '@jterrazz/station-api';

import { AppIcon, Button, Panel, Row, Tile } from '../../design/index';
import { layerOf } from '../../engine/provisional';
import type { PlanLine } from '../../engine/provisional';
import { useRead } from '../../engine/react';
import type { Reading as Read } from '../../engine/react';
import { useGesture } from '../../engine/use-gesture';
import { CATALOG_SHELF_WORDS, PLAN_GROUP_WORDS, RESTART_WORDS } from '../../engine/words';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { Page, Tiles } from '../page';
import { byAction } from './model';

import styles from './packages.module.css';

type StepsProps = {
    lines: readonly PlanLine[];
    all: readonly PlanLine[];
    title: string;
    note?: string;
};

function Steps({ lines, all, title, note = String(lines.length) }: StepsProps) {
    return (
        <Panel note={note} title={title}>
            {lines.map((line) => {
                const layer = layerOf(line, all);
                return (
                    <Row
                        key={`${line.Step.Kind}/${line.Step.Package}`}
                        lead={
                            <span className={styles.step}>
                                <span aria-hidden="true" className={styles.waiting} />
                                <AppIcon kind={line.Step.Kind} name={line.Step.Package} />
                            </span>
                        }
                        leadSize="dot"
                        size="slim"
                        {...(line.Step.Version === '' ? {} : { tag: line.Step.Version })}
                        title={line.Step.Package}
                        trail={layer === '' ? null : `from ${layer}`}
                    />
                );
            })}
        </Panel>
    );
}

function Afterwards({ plan }: { plan: MachinefilePlan }) {
    const skipped = plan.Skipped ?? [];
    const restarts = plan.Restarts ?? [];
    return (
        <>
            {skipped.length === 0 ? null : (
                <Panel title="Skipped">
                    {skipped.map((skip) => (
                        <Row
                            key={skip.Package}
                            sentence={skip.Why}
                            size="slim"
                            title={skip.Package}
                        />
                    ))}
                </Panel>
            )}
            {restarts.length === 0 ? null : (
                <Panel title="Once applied, it needs">
                    {restarts.map((restart) => (
                        <Row key={restart} size="slim" title={RESTART_WORDS[restart]} />
                    ))}
                </Panel>
            )}
        </>
    );
}

const UNCHANGED: PlanLine['Action'] = 'keep';

export function PlanSteps({ plan }: { plan: MachinefilePlan }) {
    const all = [...(plan.Lines ?? []), ...(plan.Extra ?? [])];
    return (
        <>
            {byAction(plan.Lines ?? [])
                .filter(([action]) => action !== UNCHANGED)
                .map(([action, lines]) => (
                    <Steps all={all} key={action} lines={lines} title={PLAN_GROUP_WORDS[action]} />
                ))}
            {byAction(plan.Extra ?? [])
                .filter(([action]) => action !== UNCHANGED)
                .map(([action, lines]) => (
                    <Steps
                        all={all}
                        key={`extra/${action}`}
                        lines={lines}
                        note="not in the file"
                        title={PLAN_GROUP_WORDS[action]}
                    />
                ))}
            <Afterwards plan={plan} />
        </>
    );
}

const APPLY_REREADS: readonly CallName[] = ['machinefile.drift', 'machinefile.plan'];

type PlanProps = { target: Target | undefined; name: string; back: () => void };

function PlanBar({ target, name, back }: PlanProps) {
    const apply = useGesture('machinefile.apply', {
        input: {},
        rereads: APPLY_REREADS,
        subject: name,
        target,
    });
    return (
        <div className={styles.foot}>
            <span className={styles.item}>
                <GestureOutcome gesture={apply} />
            </span>
            <span className={styles.item}>
                <Button onClick={back}>Back</Button>
                <GestureButton
                    gesture={apply}
                    label="Apply"
                    title={`Apply the plan of ${name}?`}
                    variant="primary"
                />
            </span>
        </div>
    );
}

function PlanBody({ plan, name }: { plan: Read<MachinefilePlan>; name: string }) {
    if (plan.kind === 'failed') {
        return <Unanswered error={plan.error} size="tab" />;
    }
    if (plan.kind === 'reading') {
        return <Reading size="tab" what={`the plan of ${name}`} />;
    }
    return (
        <>
            <div className={styles.planHead}>
                <h1 className={styles.planTitle}>Plan for {name}</h1>
                <p className={styles.planLine}>
                    What applying {plan.out.Name} would change, in packages. Nothing has changed
                    yet.
                </p>
            </div>
            <PlanCounts plan={plan.out} />
            <PlanSteps plan={plan.out} />
        </>
    );
}

export function unitOf(lines: readonly PlanLine[]): string {
    const shelves = new Set(lines.map((line) => CATALOG_SHELF_WORDS[line.Step.Kind]));
    const [shelf] = shelves;
    const plural = shelves.size === 1 && shelf !== undefined ? shelf.toLowerCase() : 'packages';
    return lines.length === 1 ? plural.slice(0, -1) : plural;
}

const COUNTED: readonly { action: PlanLine['Action']; foot: string }[] = [
    { action: 'install', foot: 'from the file' },
    { action: 'adopt', foot: '' },
    { action: 'remove', foot: 'never the default' },
    { action: UNCHANGED, foot: 'as declared' },
];

function PlanCounts({ plan }: { plan: MachinefilePlan }) {
    const groups = new Map(byAction(plan.Lines ?? []));
    return (
        <Tiles>
            {COUNTED.map(({ action, foot }) => (
                <Tile
                    foot={
                        action === UNCHANGED ? `${unitOf(groups.get(action) ?? [])} ${foot}` : foot
                    }
                    key={action}
                    label={PLAN_GROUP_WORDS[action]}
                    value={groups.get(action)?.length ?? 0}
                />
            ))}
        </Tiles>
    );
}

export function PlanView(props: PlanProps) {
    const { target, name } = props;
    const plan = useRead('machinefile.plan', {}, target);
    return (
        <Page>
            <PlanBody name={name} plan={plan} />
            <PlanBar {...props} />
        </Page>
    );
}
